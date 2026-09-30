package paxton

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesscredentialpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// credentialType describes how a Net2 TokenType is presented as an accesscredentialpb.CredentialType.
type credentialType struct {
	tokenType   TokenType
	displayName string
	kind        accesscredentialpb.Credential_Kind
}

// credentialTypes lists every Net2 token type that can be created, in the order Describe returns them.
// TokenTypeUnspecified is deliberately absent: it can be read, but isn't offered for create.
var credentialTypes = []credentialType{
	{TokenTypeProxCard, "Proximity card", accesscredentialpb.Credential_CARD},
	{TokenTypeProxIsoCard, "Proximity ISO card", accesscredentialpb.Credential_CARD},
	{TokenTypeProxIsoCardWithoutMagstripe, "Proximity ISO card without magstripe", accesscredentialpb.Credential_CARD},
	{TokenTypeHandsFreeKeyCard, "Hands-free key card", accesscredentialpb.Credential_CARD},
	{TokenTypeFingerprintVerificationCard, "Fingerprint verification card", accesscredentialpb.Credential_CARD},
	{TokenTypeKeyfob, "Keyfob", accesscredentialpb.Credential_FOB},
	{TokenTypeHandsFreeToken, "Hands-free token", accesscredentialpb.Credential_FOB},
	{TokenTypeWatchprox, "Watchprox", accesscredentialpb.Credential_FOB},
	{TokenTypeVehicleNumberPlate, "Vehicle number plate", accesscredentialpb.Credential_VEHICLE_PLATE},
	{TokenTypeTelephoneCallerId, "Telephone caller ID", accesscredentialpb.Credential_PHONE_NUMBER},
}

// writableStates are the states a Net2 token can be set to, via UserToken.IsLost.
var writableStates = []accesscredentialpb.Credential_State{accesscredentialpb.Credential_ACTIVE, accesscredentialpb.Credential_LOST}

// credentialWritableFields are the Credential fields a Net2 token can represent.
var credentialWritableFields = &fieldmaskpb.FieldMask{Paths: []string{"type", "value", "state"}}

// credentialOutputOnlyFields are ignored on create and update, so a credential that was read can be sent back.
var credentialOutputOnlyFields = []string{"id", "kind", "native_state"}

func findCredentialType(t TokenType) (credentialType, bool) {
	i := slices.IndexFunc(credentialTypes, func(ct credentialType) bool { return ct.tokenType == t })
	if i < 0 {
		return credentialType{}, false
	}
	return credentialTypes[i], true
}

// credentialServer implements the AccessCredential trait for one Net2 user, backed by their user tokens.
//
// Token values are credentials: they are never logged or copied into error messages,
// and neither is StatusError.Body, which Net2 fills with the value when rejecting a duplicate.
type credentialServer struct {
	accesscredentialpb.UnimplementedAccessCredentialApiServer
	accesscredentialpb.UnimplementedAccessCredentialInfoServer

	client *Client
	userID int
	logger *zap.Logger

	// writes serialises writes to this user's tokens, see lockWrites.
	writes chan struct{}
}

func newCredentialServer(client *Client, userID int, logger *zap.Logger) *credentialServer {
	return &credentialServer{
		client: client,
		userID: userID,
		logger: logger.With(zap.Int("userId", userID)),
		writes: make(chan struct{}, 1),
	}
}

// lockWrites waits until no other write to this user's tokens is in progress, or until ctx ends.
// Net2 can only replace a whole token, so two concurrent updates could both read the same token
// and the second write would undo the first, for example re-activating a card that had just been
// marked lost. Writes made outside sc-bos, e.g. in the Net2 UI, aren't covered.
func (s *credentialServer) lockWrites(ctx context.Context) (unlock func(), err error) {
	select {
	case s.writes <- struct{}{}:
		return func() { <-s.writes }, nil
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}
}

func (s *credentialServer) GetCredential(ctx context.Context, req *accesscredentialpb.GetCredentialRequest) (*accesscredentialpb.Credential, error) {
	filter := masks.NewResponseFilter(masks.WithFieldMask(req.GetReadMask()))
	if err := filter.Validate(&accesscredentialpb.Credential{}); err != nil {
		return nil, err
	}
	tokenID, err := parseTokenID("id", req.GetId())
	if err != nil {
		return nil, err
	}
	token, err := s.client.GetUserToken(ctx, s.userID, tokenID)
	if err != nil {
		return nil, s.tokenError(ctx, "get", err)
	}
	c := tokenToCredential(token)
	filter.Filter(c)
	return c, nil
}

func (s *credentialServer) ListCredentials(ctx context.Context, req *accesscredentialpb.ListCredentialsRequest) (*accesscredentialpb.ListCredentialsResponse, error) {
	tokens, err := s.client.GetUserTokens(ctx, s.userID)
	if err != nil {
		return nil, s.tokenError(ctx, "list", err)
	}
	all := make([]*accesscredentialpb.Credential, len(tokens))
	for i, token := range tokens {
		all[i] = tokenToCredential(token)
	}
	return accesscredentialpb.ListPage(all, req)
}

func (s *credentialServer) CreateCredential(ctx context.Context, req *accesscredentialpb.CreateCredentialRequest) (*accesscredentialpb.Credential, error) {
	c := req.GetCredential()
	if c == nil {
		return nil, status.Error(codes.InvalidArgument, "credential is required")
	}
	if err := checkUnsupportedFields(c); err != nil {
		return nil, err
	}
	ct, ok := findCredentialType(TokenType(c.GetType()))
	if !ok {
		return nil, unknownTypeError(c.GetType())
	}
	token, err := credentialToToken(c)
	if err != nil {
		return nil, err
	}

	unlock, err := s.lockWrites(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	created, err := s.client.AddUserToken(ctx, s.userID, token)
	if err != nil {
		if isDuplicateTokenError(err) {
			return s.findRetriedCreate(ctx, token)
		}
		return nil, s.tokenError(ctx, "create", err)
	}
	s.logger.Info("created credential", zap.Int("tokenId", created.ID), zap.String("tokenType", string(ct.tokenType)))
	return tokenToCredential(created), nil
}

// findRetriedCreate handles Net2 rejecting a new token because its value is already issued.
// That happens when the value belongs to another token, but also when an earlier attempt of the same
// POST succeeded and was retried after a 5xx. In the second case this user already holds an identical token,
// which is returned as though it had just been created.
// A token that differs in anything other than ID, e.g. one already marked lost, was not created by this request.
func (s *credentialServer) findRetriedCreate(ctx context.Context, want UserToken) (*accesscredentialpb.Credential, error) {
	tokens, err := s.client.GetUserTokens(ctx, s.userID)
	if err != nil {
		return nil, s.tokenError(ctx, "create", err)
	}
	for _, token := range tokens {
		if token.TokenType == want.TokenType && token.TokenValue == want.TokenValue && token.IsLost == want.IsLost {
			s.logger.Info("credential already held by user, treating create as successful",
				zap.Int("tokenId", token.ID), zap.String("tokenType", string(token.TokenType)))
			return tokenToCredential(token), nil
		}
	}
	return nil, errValueAlreadyIssued
}

func (s *credentialServer) UpdateCredential(ctx context.Context, req *accesscredentialpb.UpdateCredentialRequest) (*accesscredentialpb.Credential, error) {
	c := req.GetCredential()
	if c == nil {
		return nil, status.Error(codes.InvalidArgument, "credential is required")
	}
	updater := masks.NewFieldUpdater(
		masks.WithWritableFields(credentialWritableFields),
		masks.WithUpdateMask(req.GetUpdateMask()),
	)
	if err := updater.Validate(c); err != nil {
		return nil, err
	}
	if req.GetUpdateMask() == nil {
		// Validate has already rejected unsupported fields named in a mask, without one they'd be silently dropped.
		if err := checkUnsupportedFields(c); err != nil {
			return nil, err
		}
	}
	tokenID, err := parseTokenID("credential.id", c.GetId())
	if err != nil {
		return nil, err
	}

	unlock, err := s.lockWrites(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	current, err := s.client.GetUserToken(ctx, s.userID, tokenID)
	if err != nil {
		return nil, s.tokenError(ctx, "update", err)
	}
	merged := tokenToCredential(current)
	currentState := merged.GetState()
	updater.Merge(merged, proto.Clone(c))
	if merged.GetState() == accesscredentialpb.Credential_STATE_UNSPECIFIED {
		// An unset state leaves the state as it was. Reading it as ACTIVE, as create does, would
		// re-activate a lost card whenever a client left state out.
		merged.State = currentState
	}
	if merged.GetType() != string(current.TokenType) {
		if _, ok := findCredentialType(TokenType(merged.GetType())); !ok {
			return nil, unknownTypeError(merged.GetType())
		}
	}
	token, err := credentialToToken(merged)
	if err != nil {
		return nil, err
	}

	// Net2 only supports replacing the whole token, hence the read-merge-write.
	if err := s.client.UpdateUserToken(ctx, s.userID, tokenID, token); err != nil {
		return nil, s.tokenError(ctx, "update", err)
	}
	s.logger.Info("updated credential", zap.Int("tokenId", tokenID), zap.String("tokenType", string(token.TokenType)))

	stored, err := s.client.GetUserToken(ctx, s.userID, tokenID)
	if err != nil {
		// The update succeeded, so report what was written rather than failing.
		s.logger.Warn("failed to read back updated credential", zap.Int("tokenId", tokenID), safeErrorField(err))
		token.ID = tokenID
		return tokenToCredential(token), nil
	}
	return tokenToCredential(stored), nil
}

func (s *credentialServer) DeleteCredential(ctx context.Context, req *accesscredentialpb.DeleteCredentialRequest) (*accesscredentialpb.DeleteCredentialResponse, error) {
	tokenID, err := parseTokenID("id", req.GetId())
	if err != nil {
		if req.GetAllowMissing() && status.Code(err) == codes.NotFound {
			return &accesscredentialpb.DeleteCredentialResponse{}, nil
		}
		return nil, err
	}

	unlock, err := s.lockWrites(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if !req.GetAllowMissing() {
		// Check the token exists up front rather than trusting a 404 from the DELETE:
		// if Net2 deletes the token but replies with a 5xx, the retried DELETE gets a 404.
		if _, err := s.client.GetUserToken(ctx, s.userID, tokenID); err != nil {
			return nil, s.tokenError(ctx, "delete", err)
		}
	}
	if err := s.client.DeleteUserToken(ctx, s.userID, tokenID); err != nil && !isStatus(err, http.StatusNotFound) {
		return nil, s.tokenError(ctx, "delete", err)
	}
	s.logger.Info("deleted credential", zap.Int("tokenId", tokenID))
	return &accesscredentialpb.DeleteCredentialResponse{}, nil
}

func (s *credentialServer) DescribeCredential(_ context.Context, _ *accesscredentialpb.DescribeCredentialRequest) (*accesscredentialpb.CredentialSupport, error) {
	res := &accesscredentialpb.CredentialSupport{
		ResourceSupport: &typespb.ResourceSupport{
			Readable:       true,
			Writable:       true,
			WritableFields: proto.Clone(credentialWritableFields).(*fieldmaskpb.FieldMask),
		},
	}
	for _, ct := range credentialTypes {
		res.Types = append(res.Types, &accesscredentialpb.CredentialType{
			Id:             string(ct.tokenType),
			DisplayName:    ct.displayName,
			Kind:           ct.kind,
			ValueSource:    accesscredentialpb.CredentialType_CALLER_SUPPLIED,
			WritableStates: slices.Clone(writableStates),
		})
	}
	return res, nil
}

var errCredentialNotFound = status.Error(codes.NotFound, "credential not found")

// errValueAlreadyIssued doesn't say who holds the value: that would let a caller probe other users' credentials.
var errValueAlreadyIssued = status.Error(codes.AlreadyExists, "a credential with this value has already been issued")

func unknownTypeError(t string) error {
	if t == "" {
		return status.Error(codes.InvalidArgument, "credential.type is required")
	}
	return status.Errorf(codes.InvalidArgument, "unsupported credential type %q", t)
}

// tokenError converts an error from the Net2 token API to a gRPC status.
// The status message never includes a StatusError, whose body can contain a credential value.
func (s *credentialServer) tokenError(ctx context.Context, op string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return status.FromContextError(ctxErr).Err()
	}
	if errors.Is(err, ErrInvalidToken) {
		// Rejected before anything was sent. The message says why, without the value.
		return status.Errorf(codes.InvalidArgument, "unsupported credential: %v", err)
	}
	var se *StatusError
	if errors.As(err, &se) {
		switch se.StatusCode {
		case http.StatusNotFound:
			if op == "list" {
				return status.Error(codes.NotFound, "cardholder not found")
			}
			return errCredentialNotFound
		case http.StatusBadRequest:
			if isDuplicateTokenError(err) {
				return errValueAlreadyIssued
			}
			return status.Errorf(codes.InvalidArgument, "Net2 rejected the %s request", op)
		}
	}
	s.logger.Warn("Net2 token request failed", zap.String("op", op), safeErrorField(err))
	return status.Error(codes.Unavailable, "Net2 is unavailable")
}

// safeErrorField returns a zap field describing err without any response body, which may contain a credential value.
func safeErrorField(err error) zap.Field {
	var se *StatusError
	if errors.As(err, &se) {
		return zap.Int("status", se.StatusCode)
	}
	return zap.Error(err)
}

func isStatus(err error, code int) bool {
	var se *StatusError
	return errors.As(err, &se) && se.StatusCode == code
}

// isDuplicateTokenError reports whether err is Net2 rejecting a token value that has already been issued.
// Net2 responds 400 with a body like {"message": ": Card 1234 has already been issued."}.
func isDuplicateTokenError(err error) bool {
	var se *StatusError
	return errors.As(err, &se) && se.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(se.Body), "already been issued")
}

// parseTokenID parses a credential id, which is a Net2 token ID. field names the id in errors.
// An id that isn't a token ID can't match a credential, so it's NotFound rather than InvalidArgument.
func parseTokenID(field, id string) (int, error) {
	if id == "" {
		return 0, status.Errorf(codes.InvalidArgument, "%s is required", field)
	}
	n, err := strconv.Atoi(id)
	if err != nil || n < 0 { // Net2 does issue token ID 0
		return 0, errCredentialNotFound
	}
	return n, nil
}

func tokenToCredential(t UserToken) *accesscredentialpb.Credential {
	c := &accesscredentialpb.Credential{
		Id:    strconv.Itoa(t.ID),
		Type:  string(t.TokenType),
		Value: t.TokenValue,
		State: accesscredentialpb.Credential_ACTIVE,
	}
	if ct, ok := findCredentialType(t.TokenType); ok {
		c.Kind = ct.kind
	}
	if t.IsLost {
		c.State = accesscredentialpb.Credential_LOST
	}
	return c
}

// credentialToToken converts c to a Net2 token, ignoring output only fields.
// c.type must already have been checked. An unset state is ACTIVE, as it is on create,
// so UpdateCredential fills in the current state first.
func credentialToToken(c *accesscredentialpb.Credential) (UserToken, error) {
	t := UserToken{
		TokenType:  TokenType(c.GetType()),
		TokenValue: c.GetValue(),
	}
	if t.TokenValue == "" {
		return UserToken{}, status.Error(codes.InvalidArgument, "credential.value is required")
	}
	switch c.GetState() {
	case accesscredentialpb.Credential_STATE_UNSPECIFIED, accesscredentialpb.Credential_ACTIVE:
	case accesscredentialpb.Credential_LOST:
		t.IsLost = true
	default:
		return UserToken{}, status.Errorf(codes.InvalidArgument, "credential.state %v is not supported, use ACTIVE or LOST", c.GetState())
	}
	return t, nil
}

// checkUnsupportedFields rejects any field set on c that a Net2 token has nowhere to store.
// It checks every field rather than a list of unsupported ones, so a field added to Credential
// later is rejected, not silently dropped, until this driver supports it.
func checkUnsupportedFields(c *accesscredentialpb.Credential) error {
	var unsupported []string
	c.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		name := string(fd.Name())
		if !slices.Contains(credentialWritableFields.GetPaths(), name) && !slices.Contains(credentialOutputOnlyFields, name) {
			unsupported = append(unsupported, name)
		}
		return true
	})
	if len(unsupported) > 0 {
		slices.Sort(unsupported) // Range order is undefined
		return status.Errorf(codes.InvalidArgument, "Net2 does not support credential fields: %s", strings.Join(unsupported, ", "))
	}
	return nil
}
