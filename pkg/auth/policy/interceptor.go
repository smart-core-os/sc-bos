package policy

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
)

type Interceptor struct {
	logger      *zap.Logger
	auditLogger *zap.Logger
	auditModel  *logpb.Model
	policy      Policy
	verifier    token.Validator
}

func NewInterceptor(policy Policy, opts ...InterceptorOption) *Interceptor {
	interceptor := &Interceptor{
		logger: zap.NewNop(),
		policy: policy,
	}
	for _, o := range opts {
		o(interceptor)
	}
	return interceptor
}

func (i *Interceptor) HTTPInterceptor(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := i.checkPolicyHTTP(r)
		if err != nil {
			switch status.Code(err) {
			case codes.Unauthenticated:
				http.Error(w, err.Error(), http.StatusUnauthorized)
			case codes.PermissionDenied:
				http.Error(w, err.Error(), http.StatusForbidden)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func (i *Interceptor) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp any, err error) {
		_, err = i.checkPolicyGrpc(ctx, nil, req, StreamAttributes{
			IsServerStream: false,
			IsClientStream: false,
			Open:           false,
		})
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (i *Interceptor) GRPCStreamingInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// for client / bidirectional streams we don't have a request, so we'll evaluate the policy without one
		// to check if it's OK to open the stream.
		// This isn't necessary for server streams; the client will immediately send the request message, and the
		// generated code will call RecvMsg to get this *before* control is transferred to the service implementation,
		// so we can check then using the serverStreamInterceptor
		var cachedCreds *verifiedCreds
		if info.IsClientStream {
			var err error
			cachedCreds, err = i.checkPolicyGrpc(ss.Context(), nil, nil, StreamAttributes{
				IsServerStream: info.IsServerStream,
				IsClientStream: info.IsClientStream,
				Open:           false,
			})
			if err != nil {
				return err
			}
		}

		cb := func(msg any) error {
			streamAttrs := StreamAttributes{
				IsServerStream: info.IsServerStream,
				IsClientStream: info.IsClientStream,
				// Only client/bidirectional streams cause policies to be evaluated once the stream is already
				// open.
				Open: info.IsClientStream,
			}

			_, err := i.checkPolicyGrpc(ss.Context(), cachedCreds, msg, streamAttrs)
			return err
		}
		wrapped := &serverStreamInterceptor{
			ServerStream: ss,
			cb:           cb,
		}
		return handler(srv, wrapped)
	}
}

// Returns a set of verified credentials that can be used to speed up future calls to checkPolicyGrpc for the same
// call (useful for streams). Pass nil creds the first time, then cache the creds.
func (i *Interceptor) checkPolicyGrpc(ctx context.Context, creds *verifiedCreds, req any, stream StreamAttributes) (*verifiedCreds, error) {
	service, method, ok := rpcutil.ServiceMethod(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to resolve method")
	}

	if creds == nil {
		tkn, err := grpc_auth.AuthFromMD(ctx, "Bearer")
		var tokenClaims *token.Claims
		if err == nil && tkn != "" && i.verifier != nil {
			tokenClaims, err = i.verifier.ValidateAccessToken(ctx, tkn)
			if err != nil {
				tokenClaims = nil
				i.logger.Error("token failed verification", zap.Error(err))
			}
		}

		cert, valid := rpcutil.CertFromServerContext(ctx)

		creds = &verifiedCreds{
			cert:        cert,
			certValid:   valid,
			token:       tkn,
			tokenClaims: tokenClaims,
		}
	}

	input := Attributes{
		Protocol:           ProtocolGRPC,
		Service:            service,
		Method:             method,
		Stream:             stream,
		Request:            req,
		CertificatePresent: creds.cert != nil,
		CertificateValid:   creds.certValid,
		Certificate:        creds.cert,
		TokenPresent:       creds.token != "",
		TokenValid:         creds.tokenClaims != nil,
		TokenClaims:        creds.tokenClaims,
	}

	// rego.Eval (called by Validate) does a json.Marshal + json.Unmarshal on the input to convert the input to a map[string]any for use as data in policies.
	// We actually want our protos to use protojson instead which is what we'd expect when interacting with protos via json.
	// Using protojson is important, not only because it's a better defined proto-json mapping, but also because json.Marshal doesn't handle
	// messages created via the dynamicpb package.
	if m, ok := input.Request.(proto.Message); ok {
		jsonBytes, err := protojson.MarshalOptions{
			AllowPartial:      true, // avoid errors, this is not part of an RPC flow
			EmitDefaultValues: true, // make the policy files easier to write
		}.Marshal(m)
		if err != nil {
			// Keep the original message, but let people know that things aren't quite right.
			// We hope to never see this log message.
			i.logger.Warn("failed to marshal proto message to json during policy check", zap.Error(err))
		} else {
			// Avoid a json.Unmarshal(map[string]any), which would be followed by a json.Marshal in rego.Eval anyway
			input.Request = json.RawMessage(jsonBytes)
		}
	}

	queries, err := Validate(ctx, i.policy, input)
	addr := "unknown"
	if p, ok := peer.FromContext(ctx); ok {
		addr = p.Addr.String()
	}
	if err != nil {
		i.logger.Debug("request blocked by policy",
			zap.Any("attributes", input),
			zap.String("addr", addr),
			zap.Strings("queries", queries),
		)
	}
	// Only audit once per RPC, not for every message on an open client/bidirectional stream.
	if isWriteMethod(method) && !stream.Open {
		outcome := "allowed"
		if err != nil {
			outcome = "denied"
		}
		i.writeAuditEntry(outcome, creds, addr,
			auditField{"service", service},
			auditField{"method", method},
		)
	}
	return creds, err
}

func (i *Interceptor) checkPolicyHTTP(r *http.Request) (*verifiedCreds, error) {
	hdr := r.Header.Get("Authorization")
	creds := &verifiedCreds{}
	if hdr != "" {
		tkn, err := splitBearer(hdr)
		if err != nil {
			return nil, err
		}
		if i.verifier != nil {
			claims, err := i.verifier.ValidateAccessToken(r.Context(), tkn)
			if err != nil {
				i.logger.Error("token failed verification", zap.Error(err))
				return nil, err
			}
			creds.token = tkn
			creds.tokenClaims = claims
		}
	}
	if cert := httpPeerCert(r); cert != nil {
		creds.cert = cert
		creds.certValid = true
	}

	input := Attributes{
		Protocol:           ProtocolHTTP,
		Method:             r.Method,
		Path:               r.URL.Path,
		CertificatePresent: creds.cert != nil,
		CertificateValid:   creds.certValid,
		Certificate:        creds.cert,
		TokenPresent:       creds.token != "",
		TokenValid:         creds.tokenClaims != nil,
		TokenClaims:        creds.tokenClaims,
	}

	queries, err := Validate(r.Context(), i.policy, input)
	addr := r.RemoteAddr
	if err != nil {
		i.logger.Debug("request blocked by policy",
			zap.Any("attributes", input),
			zap.String("addr", addr),
			zap.Strings("queries", queries),
		)
	}
	if isHTTPWriteMethod(r.Method) {
		outcome := "allowed"
		if err != nil {
			outcome = "denied"
		}
		i.writeAuditEntry(outcome, creds, addr,
			auditField{"path", r.URL.Path},
			auditField{"httpMethod", r.Method},
		)
	}
	return creds, err
}

type InterceptorOption func(interceptor *Interceptor)

func WithLogger(logger *zap.Logger) InterceptorOption {
	return func(interceptor *Interceptor) {
		interceptor.logger = logger
	}
}

func WithTokenVerifier(tv token.Validator) InterceptorOption {
	return func(interceptor *Interceptor) {
		interceptor.verifier = tv
	}
}

func WithAuditLogger(logger *zap.Logger) InterceptorOption {
	return func(i *Interceptor) { i.auditLogger = logger }
}

func WithAuditModel(m *logpb.Model) InterceptorOption {
	return func(i *Interceptor) { i.auditModel = m }
}

type verifiedCreds struct {
	cert        *x509.Certificate
	certValid   bool
	token       string
	tokenClaims *token.Claims
}

// if we want to get the request of a server-to-client streaming call from within an interceptor, we need a way to
// intercept the RecvMsg call.
// serverStreamInterceptor will run cb on all messages received through the stream. In a server-streaming RPC,
// the first message will be the request message. If cb returns a non-nil error, then that call to RecvMsg
// will return the error from cb.
type serverStreamInterceptor struct {
	grpc.ServerStream
	cb func(m any) error
}

func (ss *serverStreamInterceptor) RecvMsg(m any) error {
	err := ss.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	return ss.cb(m)
}

func splitBearer(header string) (bearer string, err error) {
	tokenType, tokenValue, found := strings.Cut(header, " ")
	if !found {
		return "", status.Error(codes.Unauthenticated, "bad authorization header")
	}
	if !strings.EqualFold(tokenType, "bearer") {
		return "", status.Error(codes.Unauthenticated, "authorization header must be a bearer token")
	}

	return tokenValue, nil
}

func httpPeerCert(r *http.Request) *x509.Certificate {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		return nil
	}
	return r.TLS.VerifiedChains[0][0]
}

// auditField is a protocol-specific key/value pair appended to every audit entry.
type auditField struct {
	key   string
	value string
}

// writeAuditEntry records a single audit event to the configured audit sinks.
// outcome is "allowed" or "denied". extra carries protocol-specific fields
// (e.g. service+method for gRPC, path+httpMethod for HTTP).
func (i *Interceptor) writeAuditEntry(outcome string, creds *verifiedCreds, addr string, extra ...auditField) {
	if i.auditLogger == nil && i.auditModel == nil {
		return
	}
	subject, name := "", ""
	if creds.tokenClaims != nil {
		subject = creds.tokenClaims.Subject
		name = creds.tokenClaims.Name
	}
	certSubjectStr := certSubject(creds.cert)
	hasToken := creds.tokenClaims != nil

	if i.auditLogger != nil {
		fields := make([]zap.Field, 0, 7+len(extra))
		fields = append(fields, zap.String("outcome", outcome))
		for _, f := range extra {
			fields = append(fields, zap.String(f.key, f.value))
		}
		fields = append(fields,
			zap.String("peer", addr),
			zap.String("subject", subject),
			zap.String("name", name),
			zap.Bool("cert", creds.certValid),
			zap.String("certSubject", certSubjectStr),
			zap.Bool("token", hasToken),
		)
		i.auditLogger.Info("write", fields...)
	}

	if i.auditModel != nil {
		lvl := logpb.Level_LEVEL_INFO
		if outcome == "denied" {
			lvl = logpb.Level_LEVEL_WARN
		}
		allFields := make(map[string]string, 7+len(extra))
		allFields["outcome"] = outcome
		allFields["peer"] = addr
		allFields["subject"] = subject
		allFields["name"] = name
		allFields["cert"] = strconv.FormatBool(creds.certValid)
		allFields["certSubject"] = certSubjectStr
		allFields["token"] = strconv.FormatBool(hasToken)
		for _, f := range extra {
			allFields[f.key] = f.value
		}
		i.auditModel.AppendMessage(&logpb.LogMessage{
			Timestamp: timestamppb.Now(),
			Level:     lvl,
			Logger:    "audit",
			Message:   "write",
			Fields:    allFields,
		})
	}
}

// isWriteMethod reports whether the gRPC method name represents a mutating operation.
// It returns true for any method that does not begin with a known read-only prefix.
//
// The read-only prefixes cover all current SC-BOS proto read methods. Non-standard
// method names in the codebase (Identify, ConfigureService, RegenerateSecret,
// OnMessage) are all mutating and correctly classified as writes by this function.
func isWriteMethod(method string) bool {
	for _, prefix := range []string{"Get", "Pull", "Describe", "List"} {
		if strings.HasPrefix(method, prefix) {
			return false
		}
	}
	return true
}

func isHTTPWriteMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func certSubject(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	return cert.Subject.String()
}
