package remote

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
)

// Enroll sets up the PKI for a remote Smart Core node.
// This connects to the remote node specified by enrollment.TargetAddress,
// constructs a new client certificate signed using the certificate and key from authority,
// and invokes CreateEnrollment on the target with this information.
// The Certificate and RootCAs will be computed from the authority and will be ignored if provided in enrollment.
// If any remoteRoots are provided, the remote server will be checked using these as trust roots, otherwise any remote certificate will be allowed.
func Enroll(ctx context.Context, enrollment *enrollmentpb.Enrollment, authority pki.Source, remoteRoots ...string) (*enrollmentpb.Enrollment, error) {
	enrollment = proto.Clone(enrollment).(*enrollmentpb.Enrollment)

	// when in enrollment mode, the target node will be using a self-signed cert we won't be able to
	// automatically verify.
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if len(remoteRoots) > 0 {
		// when we have explicit remoteRoots we verify the remote certs!
		tlsConfig.InsecureSkipVerify = false
		tlsConfig.RootCAs = x509.NewCertPool()
		for _, root := range remoteRoots {
			tlsConfig.RootCAs.AppendCertsFromPEM([]byte(root))
		}

		// Make sure there's a timeout to avoid infinite (or 120s) connection loops in the case when tls fails.
		// There's no way to actually get the ClientConn to return on first tls error, it will continue to retry
		// until ctx expires.
		var cleanUp context.CancelFunc
		ctx, cleanUp = context.WithTimeout(ctx, 30*time.Second)
		defer cleanUp()
	}

	// the certInterceptor captures and saves the certificate presented by the server when the connection is opened
	creds := &certInterceptor{TransportCredentials: credentials.NewTLS(tlsConfig)}
	conn, err := grpc.NewClient(enrollment.TargetAddress,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	client := enrollmentpb.NewEnrollmentApiClient(conn)
	// any api call will do to force the connection to be established (or fail)
	_, err = client.GetEnrollment(ctx, &enrollmentpb.GetEnrollmentRequest{})
	if err != nil && status.Code(err) != codes.NotFound {
		// NotFound is expected if the remote node is not enrolled yet, ignore that error
		return nil, err
	}

	peerCerts, ok := creds.PeerCertificates()
	if !ok || len(peerCerts) == 0 {
		return nil, errors.New("peer did not present any certificates")
	}
	peerPublicKey := peerCerts[0].PublicKey

	authorityCert, roots, err := authority.Certs()
	if err != nil {
		return nil, err
	}

	enrollment.RootCas = pki.EncodeCertificates(roots)

	if enrollment.TargetName == "" {
		enrollment.TargetName = peerCerts[0].Subject.CommonName
	}
	certTemplate := newTargetCertificate(peerCerts, enrollment)
	enrollment.Certificate, err = pki.CreateCertificateChain(authorityCert, certTemplate, peerPublicKey,
		pki.WithAuthority(enrollment.TargetAddress),
	)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateEnrollment(ctx, &enrollmentpb.CreateEnrollmentRequest{
		Enrollment: enrollment,
	})
	if err != nil {
		return nil, err
	}

	return enrollment, nil
}

var (
	ErrNotEnrolled       = errors.New("not enrolled")
	ErrNotEnrolledWithUs = errors.New("not enrolled with us")
	ErrNotTrusted        = errors.New("not trusted")
)

// Forget asks a remote node to forget that they are enrolled using the given enrollment.
// Forget assumes that if the remote node trusts us then they also trust us to delete the enrollment.
// If certificate validation fails, we try again but this time check the remote enrollment against the passed one so
// we aren't deleting random enrollments.
func Forget(ctx context.Context, enrollment *enrollmentpb.Enrollment, tlsConfig *tls.Config) error {
	// We try a few things to get the remote node to forget about us.
	// 1. using tlsconfig, ask it to forget about us
	//    - if that fails with something like "not enrolled" (or succeeds) then we return
	// 2. if the request failed with something like "cert not valid"
	//    - connect without verifying the remote cert
	//    - get the current enrollment and compare it with the one we would have used
	//    - if it matches, ask the remote node to forget about us

	conn, err := grpc.NewClient(enrollment.TargetAddress,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		// An error here is because there's something wrong with our options or the target address.
		// In any case we can't recover from that.
		return err
	}
	defer conn.Close()

	client := enrollmentpb.NewEnrollmentApiClient(conn)
	_, err = client.DeleteEnrollment(ctx, &enrollmentpb.DeleteEnrollmentRequest{})
	switch {
	case err == nil: // success
		return nil
	case status.Code(err) == codes.NotFound: // not enrolled
		return ErrNotEnrolled
	case status.Code(err) == codes.Unavailable: // not trusted, continue to untrusted flow
		if st, ok := status.FromError(err); !ok || strings.Contains(st.Message(), "connection refused") {
			return err // early err because it's offline
		}
	default:
		return fmt.Errorf("trusted: %w", err)
	}

	conn, err = grpc.NewClient(enrollment.TargetAddress,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})),
	)
	if err != nil {
		// An error here is because there's something wrong with our options.
		// In any case we can't recover from that.
		return err
	}
	defer conn.Close()
	client = enrollmentpb.NewEnrollmentApiClient(conn)
	remoteEnrollment, err := client.GetEnrollment(ctx, &enrollmentpb.GetEnrollmentRequest{})
	switch {
	case err == nil: // success
	case status.Code(err) == codes.NotFound: // not enrolled
		return ErrNotEnrolled
	default:
		return fmt.Errorf("untrusted get: %w", err)
	}

	if remoteEnrollment.ManagerAddress != enrollment.ManagerAddress {
		return fmt.Errorf("%w: enrolled with %s", ErrNotEnrolledWithUs, remoteEnrollment.ManagerAddress)
	}

	_, err = client.DeleteEnrollment(ctx, &enrollmentpb.DeleteEnrollmentRequest{})
	if err != nil {
		switch {
		case err == nil: // success
		case status.Code(err) == codes.NotFound: // not enrolled
			return ErrNotEnrolled
		default:
			return fmt.Errorf("untrusted delete: %w", err)
		}
		return err
	}

	return nil
}

// Renew updates the PKI for a remote Smart Core node.
// This connects to the remote node specified by enrollment.TargetAddress using tlsConfig,
// signs the servers public certificate using authority,
// and calls EnrollmentApi.UpdateEnrollment on the remote node.
func Renew(ctx context.Context, enrollment *enrollmentpb.Enrollment, authority pki.Source, tlsConfig *tls.Config) (*enrollmentpb.Enrollment, error) {
	enrollment = proto.Clone(enrollment).(*enrollmentpb.Enrollment)

	// the certInterceptor captures and saves the certificate presented by the server when the connection is opened
	creds := &certInterceptor{TransportCredentials: credentials.NewTLS(tlsConfig)}
	conn, err := grpc.NewClient(enrollment.TargetAddress,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	client := enrollmentpb.NewEnrollmentApiClient(conn)
	// any api call will do to force the connection to be established (or fail)
	_, err = client.GetEnrollment(ctx, &enrollmentpb.GetEnrollmentRequest{})
	if err != nil {
		return nil, err
	}

	peerCerts, ok := creds.PeerCertificates()
	if !ok || len(peerCerts) == 0 {
		return nil, errors.New("peer did not present any certificates")
	}
	peerPublicKey := peerCerts[0].PublicKey

	authorityCert, roots, err := authority.Certs()
	if err != nil {
		return nil, err
	}

	enrollment.RootCas = pki.EncodeCertificates(roots)

	certTemplate := newTargetCertificate(peerCerts, enrollment)
	enrollment.Certificate, err = pki.CreateCertificateChain(authorityCert, certTemplate, peerPublicKey,
		pki.WithAuthority(enrollment.TargetAddress),
	)
	if err != nil {
		return nil, err
	}

	_, err = client.UpdateEnrollment(ctx, &enrollmentpb.UpdateEnrollmentRequest{
		Enrollment: enrollment,
	})
	if err != nil {
		return nil, err
	}

	return enrollment, nil
}

func newTargetCertificate(certs []*x509.Certificate, enrollment *enrollmentpb.Enrollment) *x509.Certificate {
	cert := certs[0]
	cn := enrollment.TargetName
	if cn == "" {
		cn = cert.Subject.CommonName
	}
	subject := cert.Subject
	subject.CommonName = cn
	return &x509.Certificate{
		Subject:     subject,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		// Warning: Adding this URI SAN stops the cert from working in curl (at least)
		// URIs: []*url.URL{{
		// 	Scheme: "smart-core",
		// 	Opaque: enrollment.TargetName,
		// }},
	}
}
