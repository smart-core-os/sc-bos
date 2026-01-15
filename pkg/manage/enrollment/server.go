package enrollment

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
)

type Server struct {
	enrollmentpb.UnimplementedEnrollmentApiServer
	logger *zap.Logger
	dir    string
	keyPEM []byte

	m          sync.Mutex
	enrollment Enrollment
	done       chan struct{}

	enrollmentChanged minibus.Bus[Enrollment]
}

func (es *Server) GetEnrollment(_ context.Context, _ *enrollmentpb.GetEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	es.m.Lock()
	defer es.m.Unlock()

	select {
	case <-es.done:
		e := es.enrollment
		eProto := &enrollmentpb.Enrollment{
			TargetName:     e.RootDeviceName,
			TargetAddress:  e.LocalAddress,
			ManagerName:    e.ManagerName,
			ManagerAddress: e.ManagerAddress,
		}
		return eProto, nil
	default:
		return nil, status.Error(codes.NotFound, "not enrolled")
	}
}

// NewServer creates an enrollment server, without attempting to load an existing enrollment.
// The new server will be in an un-enrolled state.
// New enrollments will be saved in the provided directory.
func NewServer(dir string, keyPEM []byte, logger *zap.Logger) *Server {
	es := &Server{
		logger: logger,
		dir:    dir,
		keyPEM: keyPEM,
		done:   make(chan struct{}),
	}
	return es
}

// LoadOrCreateServer will try to load an enrollment from disk. If successful, a server in the enrolled state is
// returned. Otherwise, a server in the unenrolled state is returned and new enrollments will be saved in the
// provided directory.
func LoadOrCreateServer(dir string, keyPEM []byte, logger *zap.Logger) (*Server, error) {
	es := NewServer(dir, keyPEM, logger)
	enrollment, err := LoadEnrollment(dir, keyPEM)
	if err == nil {
		es.enrollment = enrollment
		close(es.done)
		logger.Info("The controller is enrolled with a hub", zap.String("hubAddress", enrollment.ManagerAddress))
	} else if !errors.Is(err, ErrNotEnrolled) {
		return nil, err
	}

	return es, nil
}

func (es *Server) CreateEnrollment(ctx context.Context, request *enrollmentpb.CreateEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	logger := rpcutil.ServerLogger(ctx, es.logger)

	// only allow one enrollment at a time
	es.m.Lock()
	defer es.m.Unlock()

	select {
	case <-es.done:
		return nil, status.Error(codes.AlreadyExists, "already enrolled")
	default:
	}

	cert, err := tls.X509KeyPair(request.GetEnrollment().GetCertificate(), es.keyPEM)
	if err != nil {
		logger.Error("invalid enrollment certificate", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "invalid certificate")
	}

	roots, err := pki.ParseCertificatesPEM(request.GetEnrollment().GetRootCas())
	if err != nil {
		logger.Error("invalid enrollment root", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "invalid root certificate(s)")
	}
	if len(roots) != 1 {
		return nil, status.Error(codes.InvalidArgument, "only 1 root CA is supported")
	}

	enrollment := Enrollment{
		RootDeviceName: request.GetEnrollment().GetTargetName(),
		ManagerName:    request.GetEnrollment().GetManagerName(),
		ManagerAddress: request.GetEnrollment().GetManagerAddress(),
		LocalAddress:   request.GetEnrollment().GetTargetAddress(),
		RootCA:         roots[0],
		Cert:           cert,
	}
	err = SaveEnrollment(es.dir, enrollment)
	if err != nil {
		logger.Error("failed to save enrollment", zap.Error(err), zap.String("dir", es.dir))
		return nil, status.Error(codes.Internal, "failed to save enrollment")
	}

	es.enrollment = enrollment
	go es.enrollmentChanged.Send(context.Background(), enrollment)
	close(es.done)

	logger.Info("The controller is newly enrolled with a hub", zap.String("hubAddress", enrollment.ManagerAddress))
	return request.GetEnrollment(), nil
}

func (es *Server) UpdateEnrollment(ctx context.Context, request *enrollmentpb.UpdateEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	logger := rpcutil.ServerLogger(ctx, es.logger)

	es.m.Lock()
	defer es.m.Unlock()

	select {
	case <-es.done:
	default:
		return nil, status.Error(codes.NotFound, "not enrolled")
	}

	en := es.enrollment
	oldLeaf, err := x509.ParseCertificate(en.Cert.Certificate[0])
	if err != nil {
		logger.Error("Failed to parse old enrollment certificate", zap.Error(err))
		return nil, status.Error(codes.Unknown, "failed to parse old certificate")
	}

	cert, err := tls.X509KeyPair(request.GetEnrollment().GetCertificate(), es.keyPEM)
	if err != nil {
		logger.Debug("invalid enrollment certificate", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "invalid certificate")
	}

	newLeaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		// shouldn't happen because tls.X509KeyPair parses the cert too
		logger.Error("Failed to parse new enrollment certificate", zap.Error(err))
		return nil, status.Error(codes.Unknown, "failed to parse new certificate")
	}

	if !haveSameIssuer(oldLeaf, newLeaf) {
		logger.Debug("Updated enrollment cert not from same issuer", zap.Any("old", oldLeaf.Issuer), zap.Any("new", newLeaf.Issuer))
		return nil, status.Error(codes.InvalidArgument, "issuers don't match")
	}

	roots, err := pki.ParseCertificatesPEM(request.GetEnrollment().GetRootCas())
	if err != nil {
		logger.Error("invalid enrollment root", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "invalid root certificate(s)")
	}
	if len(roots) != 1 {
		return nil, status.Error(codes.InvalidArgument, "only 1 root CA is supported")
	}

	enrollment := Enrollment{
		RootDeviceName: request.GetEnrollment().GetTargetName(),
		ManagerName:    request.GetEnrollment().GetManagerName(),
		ManagerAddress: request.GetEnrollment().GetManagerAddress(),
		LocalAddress:   request.GetEnrollment().GetTargetAddress(),
		RootCA:         roots[0],
		Cert:           cert,
	}
	err = SaveEnrollment(es.dir, enrollment)
	if err != nil {
		logger.Debug("failed to save enrollment", zap.Error(err), zap.String("dir", es.dir))
		return nil, status.Error(codes.Aborted, "failed to store enrollment")
	}

	es.enrollment = enrollment
	go es.enrollmentChanged.Send(context.Background(), enrollment)
	logger.Info("The controllers enrollment has been renewed", zap.String("hubAddress", enrollment.ManagerAddress))
	return request.GetEnrollment(), nil
}

func (es *Server) DeleteEnrollment(ctx context.Context, request *enrollmentpb.DeleteEnrollmentRequest) (*enrollmentpb.Enrollment, error) {
	// delete only if we are enrolled already
	es.m.Lock()
	defer es.m.Unlock()

	select {
	case <-es.done:
	default:
		return nil, status.Error(codes.NotFound, "not enrolled")
	}

	// 0. remember state needed to undo this delete if needed
	// 1. clear the enrollment pki.Source
	// 2. delete the enrollment files
	// 3. reset tracking state

	en := es.enrollment
	es.enrollment = Enrollment{}
	if err := DeleteEnrollment(es.dir); err != nil {
		es.enrollment = en
		// try our best to save the enrollment again
		var saveErr error
		for range 5 {
			saveErr = SaveEnrollment(es.dir, en)
			if saveErr == nil {
				break
			}
		}
		if saveErr != nil {
			es.logger.Error("delete failed, rollback failed - manual intervention required", zap.NamedError("delErr", err), zap.NamedError("rollbackErr", saveErr))
			return nil, status.Errorf(codes.DataLoss, "failed to delete, failed to rollback")
		}
		return nil, status.Errorf(codes.Aborted, "failed to delete, aborted")
	}
	es.done = make(chan struct{})
	go es.enrollmentChanged.Send(context.Background(), Enrollment{})

	es.logger.Info("The controller is no longer enrolled with a hub", zap.String("hubAddress", en.ManagerAddress))

	return &enrollmentpb.Enrollment{
		TargetName:     en.RootDeviceName,
		ManagerName:    en.ManagerName,
		ManagerAddress: en.ManagerAddress,
	}, nil
}

func (es *Server) TestEnrollment(ctx context.Context, _ *enrollmentpb.TestEnrollmentRequest) (*enrollmentpb.TestEnrollmentResponse, error) {
	e, ok := es.Enrollment()
	if !ok {
		return nil, status.Error(codes.NotFound, "not enrolled")
	}

	tlsConfig := pki.TLSClientConfig(pki.FuncSource(func() (*tls.Certificate, []*x509.Certificate, error) {
		return &e.Cert, []*x509.Certificate{e.RootCA}, nil
	}))
	conn, err := grpc.NewClient(e.ManagerAddress, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, err
	}
	client := hubpb.NewHubApiClient(conn)
	_, err = client.TestHubNode(ctx, &hubpb.TestHubNodeRequest{Address: e.LocalAddress})
	res := &enrollmentpb.TestEnrollmentResponse{}
	if err != nil {
		if s, ok := status.FromError(err); ok {
			res.Code = int32(s.Code())
			res.Error = s.Message()
		} else {
			res.Error = err.Error()
		}
	}
	return res, nil
}

func (es *Server) Wait(ctx context.Context) (enrollment Enrollment, done bool) {
	select {
	case <-es.done:
		es.m.Lock()
		defer es.m.Unlock()
		return es.enrollment, true
	case <-ctx.Done():
		return es.enrollment, false
	}
}

func (es *Server) Enrollment() (enrollment Enrollment, ok bool) {
	select {
	case <-es.done:
		es.m.Lock()
		defer es.m.Unlock()
		return es.enrollment, true
	default:
		return Enrollment{}, false
	}
}

// Certs implements pki.Source and provides a certificate source that provides the latest known enrollment certificate.
// If the certificate source is used while this Server has no enrollment, an error will be returned.
// This is therefore not suitable for use in enrollment mode - use a self-signed certificate source (with the enrollment
// private key) instead.
func (es *Server) Certs() (*tls.Certificate, []*x509.Certificate, error) {
	es.m.Lock()
	defer es.m.Unlock()
	return es.certsLocked()
}

func (es *Server) certsLocked() (*tls.Certificate, []*x509.Certificate, error) {
	// check that we are enrolled
	select {
	case <-es.done:
	default:
		return nil, nil, ErrNotEnrolled
	}

	cert := es.enrollment.Cert
	roots := []*x509.Certificate{es.enrollment.RootCA}
	return &cert, roots, nil
}

// RequestRenew asks the hub to renew our certificate.
// Errors if this node is not enrolled with a hub.
func (es *Server) RequestRenew(ctx context.Context) error {
	es.m.Lock()
	clientCert, roots, err := es.certsLocked()
	hubAddress := es.enrollment.ManagerAddress
	localAddress := es.enrollment.LocalAddress
	es.m.Unlock()

	if err != nil {
		return err
	}
	if hubAddress == "" {
		return errors.New("hub address not known")
	}
	if localAddress == "" {
		return errors.New("local address not known")
	}

	source := pki.FuncSource(func() (*tls.Certificate, []*x509.Certificate, error) {
		return clientCert, roots, err
	})
	tlsConfig := pki.TLSClientConfig(source)
	conn, err := grpc.NewClient(hubAddress, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := hubpb.NewHubApiClient(conn)
	// warning, do not hold es.m lock when invoking this method or we'll get a deadlock that includes a network hop
	// which would be really hard to debug!
	_, err = client.RenewHubNode(ctx, &hubpb.RenewHubNodeRequest{Address: localAddress})
	return err
}

func (es *Server) AutoRenew(ctx context.Context) error {
	const afterProgress = 0.75

	enrollments := es.Enrollments(ctx)
	var renewAfter *time.Timer
	var timerC <-chan time.Time
	var renewAttempt int // for tracking failures and for computing delays
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case enrollment, ok := <-enrollments:
			// if the enrollment changes we:
			// 1. if we aren't enrolled: stop any renewal timers and clean up resources
			// 2. if we are already enrolled: stop any existing timers and continue with 3
			// 3. if we are newly enrolled: work out when to auto-renew and setup timers to wake us up then.

			if !ok {
				return ctx.Err()
			}

			renewAttempt = 0 // reset any backoff settings

			if enrollment.IsZero() {
				// stop auto-renewing for now
				if renewAfter == nil {
					// we already weren't auto-renewing
					continue
				}
				renewAfter.Stop()
				renewAfter = nil
				timerC = nil
				continue
			}

			// start auto-renewing
			if renewAfter != nil {
				// Start with a new timer.
				// this is easier than tracking resets and drains
				renewAfter.Stop()
				renewAfter = nil
				timerC = nil
			}
			leaf, err := pki.TLSLeaf(&enrollment.Cert)
			if err != nil {
				// this shouldn't happen because the cert will have already been validated during enrollment
				es.logger.Error("Unexpected cert parsing error during auto-renewal check", zap.Error(err))
				continue
			}
			maxAge := leaf.NotAfter.Sub(leaf.NotBefore)
			renewAge := time.Duration(float64(maxAge) * afterProgress)
			renewTime := leaf.NotBefore.Add(renewAge)
			now := time.Now()
			renewDelay := renewTime.Sub(now)
			es.logger.Debug("Auto-renewal of enrolled certificate scheduled", zap.Time("at", renewTime))
			// wait until the cert is reaching expiry before we renew it,
			// note negative durations cause the timer to trigger immediately
			renewAfter = time.NewTimer(renewDelay)
			timerC = renewAfter.C
		case <-timerC:
			err := es.RequestRenew(ctx)
			if err != nil {
				renewAttempt++
				if renewAttempt == 1 {
					es.logger.Warn("Auto-renewal of enrolled certificate failed, will retry", zap.Error(err))
				} else if renewAttempt%20 == 0 {
					es.logger.Warn("Auto-renewal of enrolled certificate is still failing", zap.Error(err), zap.Int("attempts", renewAttempt))
				}
				newDelay := time.Duration(float64(100*time.Millisecond) * math.Pow(1.1, float64(renewAttempt)))
				if newDelay > 5*time.Minute {
					newDelay = 5 * time.Minute
				}
				renewAfter.Reset(newDelay)
			}
		}
	}
}

// Enrollments returns a chan that emits whenever the enrollment status or properties for this server change.
func (es *Server) Enrollments(ctx context.Context) <-chan Enrollment {
	changes := es.enrollmentChanged.Listen(ctx)
	// send the initial data right away
	out := make(chan Enrollment, 1)
	if en, ok := es.Enrollment(); ok {
		out <- en
	}

	go func() {
		defer close(out)
		for en := range changes {
			out <- en
		}
	}()
	return out
}

// ManagerAddress returns a chan that emits the manager address whenever it changes.
// Cancel the given context to stop listening for changes.
func (es *Server) ManagerAddress(ctx context.Context) <-chan string {
	changes := es.Enrollments(ctx)
	// send the initial data right away
	out := make(chan string)
	go func() {
		defer close(out)
		for addr := range changes {
			out <- addr.ManagerAddress
		}
	}()
	return out
}

func haveSameIssuer(a, b *x509.Certificate) bool {
	if a == nil || b == nil {
		return a == b
	}
	return bytes.Equal(a.RawIssuer, b.RawIssuer)
}
