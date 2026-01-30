package accesstoken

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/rpcutil"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
)

// Server implements an OAuth 2.0 token server that supports client credentials and password grant types.
//
// When WithClientCredentialFlow is set, the server will accept requests with grant_type=client_credentials.
// When WithPasswordFlow is set, the server will accept requests with grant_type=password. It is possible to use
// both on the same server.
//
// Error scenarios:
//   - Wrong credentials: error code "invalid_grant".
//   - Unsupported grant type: error code "unsupported_grant_type".
//   - Malformed request: error code "invalid_request".
//   - Authentication successful, but identity has no resource access: error code "unauthorized_client".
type Server struct {
	tokens *Source
	logger *zap.Logger

	clientCredentialVerifier Verifier
	clientCredentialValidity time.Duration

	passwordVerifier Verifier
	passwordValidity time.Duration
}

func NewServer(name string, opts ...ServerOption) (*Server, error) {
	key, err := generateKey()
	if err != nil {
		return nil, err
	}
	tokens := &Source{
		Key:    key,
		Issuer: name,
		Now:    time.Now,
	}

	s := &Server{tokens: tokens}
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

type ServerOption func(ts *Server)

func WithLogger(logger *zap.Logger) ServerOption {
	return func(ts *Server) {
		ts.logger = logger
	}
}

func WithClientCredentialFlow(v Verifier, validity time.Duration) ServerOption {
	return func(ts *Server) {
		ts.clientCredentialVerifier = v
		ts.clientCredentialValidity = validity
	}
}

func WithPasswordFlow(v Verifier, validity time.Duration) ServerOption {
	return func(ts *Server) {
		ts.passwordVerifier = v
		ts.passwordValidity = validity
	}
}

func WithPermittedSignatureAlgorithms(algs []string) ServerOption {
	return func(ts *Server) {
		ts.tokens.SignatureAlgorithms = algs
	}
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// time out operations after one minute
	ctx, cancel := context.WithTimeout(request.Context(), time.Minute)
	defer cancel()
	logger := rpcutil.HTTPLogger(request, s.logger)

	writer.Header().Set("Cache-Control", "no-store")

	// check and extract request parameters
	if request.Method != http.MethodPost {
		writeTokenError(writer, errInvalidRequest, logger)
		return
	}
	if err := parsePostForm(request); err != nil {
		writeTokenError(writer, err, logger)
		return
	}

	var err error
	switch request.PostForm.Get("grant_type") {
	case "client_credentials":
		err = s.clientCredentialsFlow(ctx, writer, request)
	case "password":
		err = s.passwordFlow(ctx, writer, request)
	default:
		err = errUnsupportedGrantType
	}

	if err != nil {
		writeTokenError(writer, err, logger)
	}
}

func parsePostForm(request *http.Request) error {
	ct := request.Header.Get("Content-Type")
	// RFC 7231, section 3.1.1.5 - empty type
	//   MAY be treated as application/octet-stream
	if ct == "" {
		ct = "application/octet-stream"
	}
	ct, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return err
	}
	switch ct {
	case "application/x-www-form-urlencoded":
		return request.ParseForm()
	case "multipart/form-data":
		return request.ParseMultipartForm(0)
	}
	return tokenError{
		Code:             400,
		ErrorName:        "incorrect_content_type",
		ErrorDescription: fmt.Sprintf("Content-Type is not application/x-www-form-urlencoded or multipart/form-data: %v", ct),
	}
}

func (s *Server) clientCredentialsFlow(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	if s.clientCredentialVerifier == nil {
		return errUnsupportedGrantType
	}
	clientId, clientSecret, err := s.clientCreds(request)
	if err != nil {
		return err
	}

	// lookup secret, and ensure it's for the matching client
	secretData, err := s.clientCredentialVerifier.Verify(ctx, clientId, clientSecret)
	if tokenErr := (tokenError{}); errors.As(err, &tokenErr) {
		return tokenErr
	} else if err != nil {
		return errInvalidClient
	}

	// generate an access token for the client
	token, err := s.tokens.GenerateAccessToken(secretData, s.clientCredentialValidity)
	if err != nil {
		return errors.New("failed to generate token")
	}

	// send response to the client
	response := tokenSuccessResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.clientCredentialValidity.Seconds()),
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return errors.New("failed to marshal response")
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(responseBytes)
	if err != nil {
		return errors.New("failed to write response body")
	}
	return nil
}

func (s *Server) passwordFlow(ctx context.Context, writer http.ResponseWriter, request *http.Request) error {
	if s.passwordVerifier == nil {
		return errUnsupportedGrantType
	}
	username, password, err := s.userCreds(request)
	if err != nil {
		return err
	}

	// lookup secret, and ensure it's for the matching client
	secretData, err := s.passwordVerifier.Verify(ctx, username, password)
	if tokenErr := (tokenError{}); errors.As(err, &tokenErr) {
		return tokenErr
	} else if err != nil {
		return errInvalidClient
	}

	// generate an access token for the client
	token, err := s.tokens.GenerateAccessToken(secretData, s.passwordValidity)
	if err != nil {
		return errors.New("failed to generate token")
	}

	// send response to the client
	response := tokenSuccessResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.passwordValidity.Seconds()),
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return errors.New("failed to marshal response")
	}

	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(responseBytes)
	if err != nil {
		return errors.New("failed to write response body")
	}
	return nil
}

func (s *Server) clientCreds(request *http.Request) (clientID string, clientSecret string, err error) {
	if !request.PostForm.Has("client_id") || !request.PostForm.Has("client_secret") {
		return "", "", errInvalidRequest
	}
	clientID = request.PostForm.Get("client_id")
	clientSecret = request.PostForm.Get("client_secret")
	return clientID, clientSecret, nil
}

func (s *Server) userCreds(request *http.Request) (username string, password string, err error) {
	if !request.PostForm.Has("username") || !request.PostForm.Has("password") {
		return "", "", errInvalidRequest
	}
	username = request.PostForm.Get("username")
	password = request.PostForm.Get("password")
	return username, password, nil
}

func (s *Server) TokenValidator() token.Validator {
	return s.tokens
}

func writeTokenError(writer http.ResponseWriter, err error, logger *zap.Logger) {
	tokErr, ok := err.(tokenError)
	if !ok {
		tokErr = tokenError{
			Code:             500,
			ErrorName:        "internal",
			ErrorDescription: err.Error(),
		}
	}

	body, marshalErr := json.Marshal(tokErr)
	if marshalErr != nil {
		logger.Error("failed to marshal error response", zap.Error(err))
		body = []byte(`{"error": "internal"}`)
	}

	writer.WriteHeader(tokErr.Code)
	_, writeErr := writer.Write(body)
	if writeErr != nil {
		logger.Error("failed to write error body", zap.Error(err))
	}
}

type tokenSuccessResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // in seconds
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type tokenError struct {
	Code             int    `json:"-"` // http status code to use
	ErrorName        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

func (te tokenError) Error() string {
	if te.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", te.ErrorName, te.ErrorDescription)
	} else {
		return te.ErrorName
	}
}

var (
	errInvalidRequest       = tokenError{Code: 400, ErrorName: "invalid_request"}
	errInvalidClient        = tokenError{Code: 401, ErrorName: "invalid_client"}
	errUnsupportedGrantType = tokenError{Code: 400, ErrorName: "unsupported_grant_type"}
)
