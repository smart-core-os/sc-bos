package paxton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesscredentialpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
)

const testUserID = 7

// fakeNet2 is an in-memory Net2 token store for a set of users.
type fakeNet2 struct {
	mu     sync.Mutex
	users  map[int][]UserToken
	lastID int
	// failNext, if non-zero, is the status returned by the next token request.
	failNext int
}

func newFakeNet2(tokens ...UserToken) *fakeNet2 {
	f := &fakeNet2{users: map[int][]UserToken{testUserID: nil}}
	for _, t := range tokens {
		f.users[testUserID] = append(f.users[testUserID], t)
		f.lastID = max(f.lastID, t.ID)
	}
	return f
}

var tokenPath = regexp.MustCompile(`^/api/v1/users/(\d+)/tokens(?:/(\d+))?$`)

func (f *fakeNet2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.URL.Path == "/api/v1/users" {
		var users []User
		for id := range f.users {
			users = append(users, User{ID: id, FirstName: "User", LastName: strconv.Itoa(id)})
		}
		writeJSON(w, http.StatusOK, users)
		return
	}

	if f.failNext != 0 {
		w.WriteHeader(f.failNext)
		f.failNext = 0
		return
	}
	m := tokenPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	userID, _ := strconv.Atoi(m[1])
	tokens, ok := f.users[userID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	idx := -1
	if m[2] != "" {
		tokenID, _ := strconv.Atoi(m[2])
		for i, t := range tokens {
			if t.ID == tokenID {
				idx = i
			}
		}
		if idx < 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	switch {
	case r.Method == http.MethodGet && idx < 0:
		writeJSON(w, http.StatusOK, tokens)
	case r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, tokens[idx])
	case r.Method == http.MethodPost:
		var in UserToken
		_ = json.NewDecoder(r.Body).Decode(&in)
		if f.issued(in.TokenValue, -1) {
			writeAlreadyIssued(w, in.TokenValue)
			return
		}
		f.lastID++
		in.ID = f.lastID
		f.users[userID] = append(tokens, in)
		writeJSON(w, http.StatusOK, in)
	case r.Method == http.MethodPut:
		var in UserToken
		_ = json.NewDecoder(r.Body).Decode(&in)
		if f.issued(in.TokenValue, tokens[idx].ID) {
			writeAlreadyIssued(w, in.TokenValue)
			return
		}
		tokens[idx] = in
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete:
		f.users[userID] = append(tokens[:idx], tokens[idx+1:]...)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// issued reports whether a token other than exceptID (-1 for none) holds value.
func (f *fakeNet2) issued(value string, exceptID int) bool {
	for _, tokens := range f.users {
		for _, t := range tokens {
			if t.TokenValue == value && t.ID != exceptID {
				return true
			}
		}
	}
	return false
}

// writeAlreadyIssued replies the way Net2 does to a duplicate value, naming it.
func writeAlreadyIssued(w http.ResponseWriter, value string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"message": fmt.Sprintf(": Card %s has already been issued.", value)})
}

func (f *fakeNet2) token(userID, tokenID int) (UserToken, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.users[userID] {
		if t.ID == tokenID {
			return t, true
		}
	}
	return UserToken{}, false
}

func (f *fakeNet2) fail(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext = code
}

func setupCredentialServer(t *testing.T, net2 *fakeNet2) (*credentialServer, *[]recordedRequest) {
	t.Helper()
	c, _, requests := setupTokenClient(t, net2.ServeHTTP)
	return newCredentialServer(c, testUserID, zap.NewNop()), requests
}

func TestCredentialServer_DescribeCredential(t *testing.T) {
	s, requests := setupCredentialServer(t, newFakeNet2())
	got, err := s.DescribeCredential(context.Background(), &accesscredentialpb.DescribeCredentialRequest{})
	require.NoError(t, err)
	assert.Empty(t, *requests, "Describe shouldn't call Net2")

	assert.Equal(t, []string{"type", "value", "state"}, got.GetResourceSupport().GetWritableFields().GetPaths())
	assert.False(t, got.GetResourceSupport().GetObservable())

	kinds := map[string]accesscredentialpb.Credential_Kind{}
	for _, ct := range got.GetTypes() {
		kinds[ct.GetId()] = ct.GetKind()
		assert.Equal(t, accesscredentialpb.CredentialType_CALLER_SUPPLIED, ct.GetValueSource(), ct.GetId())
		assert.Equal(t, writableStates, ct.GetWritableStates(), ct.GetId())
		assert.NotEmpty(t, ct.GetDisplayName(), ct.GetId())
	}
	assert.Equal(t, map[string]accesscredentialpb.Credential_Kind{
		"ProxCard":                    accesscredentialpb.Credential_CARD,
		"ProxIsoCard":                 accesscredentialpb.Credential_CARD,
		"ProxIsoCardWithoutMagstripe": accesscredentialpb.Credential_CARD,
		"HandsFreeKeyCard":            accesscredentialpb.Credential_CARD,
		"FingerprintVerificationCard": accesscredentialpb.Credential_CARD,
		"Keyfob":                      accesscredentialpb.Credential_FOB,
		"HandsFreeToken":              accesscredentialpb.Credential_FOB,
		"Watchprox":                   accesscredentialpb.Credential_FOB,
		"VehicleNumberPlate":          accesscredentialpb.Credential_VEHICLE_PLATE,
		"TelephoneCallerId":           accesscredentialpb.Credential_PHONE_NUMBER,
	}, kinds)

	// every Net2 type is offered apart from Unspecified
	for _, tt := range tokenTypes {
		_, offered := kinds[string(tt)]
		assert.Equal(t, tt != TokenTypeUnspecified, offered, tt)
	}
}

func TestTokenToCredential(t *testing.T) {
	tests := []struct {
		token UserToken
		want  *accesscredentialpb.Credential
	}{
		{
			UserToken{ID: 3, TokenType: TokenTypeProxCard, TokenValue: "111"},
			&accesscredentialpb.Credential{Id: "3", Type: "ProxCard", Kind: accesscredentialpb.Credential_CARD, Value: "111", State: accesscredentialpb.Credential_ACTIVE},
		},
		{
			UserToken{ID: 4, TokenType: TokenTypeKeyfob, TokenValue: "222", IsLost: true},
			&accesscredentialpb.Credential{Id: "4", Type: "Keyfob", Kind: accesscredentialpb.Credential_FOB, Value: "222", State: accesscredentialpb.Credential_LOST},
		},
		{
			UserToken{ID: 5, TokenType: TokenTypeUnspecified, TokenValue: "333"},
			&accesscredentialpb.Credential{Id: "5", Type: "Unspecified", Value: "333", State: accesscredentialpb.Credential_ACTIVE},
		},
	}
	for _, tt := range tests {
		assert.Empty(t, cmpDiff(tt.want, tokenToCredential(tt.token)))
	}
}

func TestCredentialServer_CreateCredential(t *testing.T) {
	net2 := newFakeNet2()
	s, requests := setupCredentialServer(t, net2)

	got, err := s.CreateCredential(context.Background(), &accesscredentialpb.CreateCredentialRequest{Credential: &accesscredentialpb.Credential{
		Id:          "99", // output only
		Type:        "Keyfob",
		Kind:        accesscredentialpb.Credential_CARD, // output only
		Value:       "12345678",
		State:       accesscredentialpb.Credential_LOST,
		NativeState: "ignored", // output only
	}})
	require.NoError(t, err)
	assert.Empty(t, cmpDiff(&accesscredentialpb.Credential{
		Id: "1", Type: "Keyfob", Kind: accesscredentialpb.Credential_FOB, Value: "12345678", State: accesscredentialpb.Credential_LOST,
	}, got))

	require.Len(t, *requests, 1)
	assert.Equal(t, http.MethodPost, (*requests)[0].method)
	assert.JSONEq(t, `{"TokenType":"Keyfob","TokenValue":"12345678","IsLost":true}`, string((*requests)[0].body))
}

func TestCredentialServer_CreateCredential_invalid(t *testing.T) {
	tests := []struct {
		name string
		c    *accesscredentialpb.Credential
	}{
		{"nil", nil},
		{"no type", &accesscredentialpb.Credential{Value: "1"}},
		{"unknown type", &accesscredentialpb.Credential{Type: "Magic", Value: "1"}},
		{"unspecified type", &accesscredentialpb.Credential{Type: "Unspecified", Value: "1"}},
		{"no value", &accesscredentialpb.Credential{Type: "ProxCard"}},
		{"disabled", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", State: accesscredentialpb.Credential_DISABLED}},
		{"stolen", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", State: accesscredentialpb.Credential_STOLEN}},
		{"expire_time", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", ExpireTime: timestamppb.Now()}},
		{"active_time", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", ActiveTime: timestamppb.Now()}},
		{"issue_level", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", IssueLevel: new(int32)}},
		{"invitation", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", Invitation: &accesscredentialpb.Credential_Invitation{}}},
		{"more", &accesscredentialpb.Credential{Type: "ProxCard", Value: "1", More: map[string]string{"a": "b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, requests := setupCredentialServer(t, newFakeNet2())
			_, err := s.CreateCredential(context.Background(), &accesscredentialpb.CreateCredentialRequest{Credential: tt.c})
			assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
			assert.Empty(t, *requests, "nothing should be sent to Net2")
		})
	}
}

func TestCredentialServer_CreateCredential_duplicate(t *testing.T) {
	const value = "87654321"

	t.Run("held by this user", func(t *testing.T) {
		// e.g. an earlier POST succeeded but the reply was lost and the request retried
		net2 := newFakeNet2(UserToken{ID: 19, TokenType: TokenTypeProxCard, TokenValue: value})
		s, requests := setupCredentialServer(t, net2)
		got, err := s.CreateCredential(context.Background(), &accesscredentialpb.CreateCredentialRequest{Credential: &accesscredentialpb.Credential{
			Type: "ProxCard", Value: value,
		}})
		require.NoError(t, err)
		assert.Equal(t, "19", got.GetId())
		require.Len(t, *requests, 2)
		assert.Equal(t, http.MethodPost, (*requests)[0].method)
		assert.Equal(t, http.MethodGet, (*requests)[1].method)
	})

	tests := []struct {
		name  string
		net2  *fakeNet2
		token UserToken
	}{
		{
			name:  "held by another user",
			net2:  &fakeNet2{users: map[int][]UserToken{testUserID: nil, 8: {{ID: 1, TokenType: TokenTypeProxCard, TokenValue: value}}}},
			token: UserToken{TokenType: TokenTypeProxCard, TokenValue: value},
		},
		{
			name:  "held by this user as another type",
			net2:  newFakeNet2(UserToken{ID: 19, TokenType: TokenTypeKeyfob, TokenValue: value}),
			token: UserToken{TokenType: TokenTypeProxCard, TokenValue: value},
		},
		{
			name:  "held by this user but lost",
			net2:  newFakeNet2(UserToken{ID: 19, TokenType: TokenTypeProxCard, TokenValue: value, IsLost: true}),
			token: UserToken{TokenType: TokenTypeProxCard, TokenValue: value},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := setupCredentialServer(t, tt.net2)
			_, err := s.CreateCredential(context.Background(), &accesscredentialpb.CreateCredentialRequest{Credential: &accesscredentialpb.Credential{
				Type: string(tt.token.TokenType), Value: tt.token.TokenValue,
			}})
			assert.Equal(t, codes.AlreadyExists, status.Code(err), err)
			assert.NotContains(t, err.Error(), value)
			assert.NotContains(t, err.Error(), "8", "shouldn't name the holder")
		})
	}
}

func TestCredentialServer_UpdateCredential(t *testing.T) {
	existing := UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"}

	tests := []struct {
		name     string
		req      *accesscredentialpb.UpdateCredentialRequest
		wantPut  string
		wantResp *accesscredentialpb.Credential
	}{
		{
			name: "mask state",
			req: &accesscredentialpb.UpdateCredentialRequest{
				Credential: &accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_LOST, Value: "not applied"},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
			},
			wantPut:  `{"Id":5,"TokenType":"ProxCard","TokenValue":"111","IsLost":true}`,
			wantResp: &accesscredentialpb.Credential{Id: "5", Type: "ProxCard", Kind: accesscredentialpb.Credential_CARD, Value: "111", State: accesscredentialpb.Credential_LOST},
		},
		{
			name: "mask value",
			req: &accesscredentialpb.UpdateCredentialRequest{
				Credential: &accesscredentialpb.Credential{Id: "5", Value: "222"},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
			},
			wantPut:  `{"Id":5,"TokenType":"ProxCard","TokenValue":"222","IsLost":false}`,
			wantResp: &accesscredentialpb.Credential{Id: "5", Type: "ProxCard", Kind: accesscredentialpb.Credential_CARD, Value: "222", State: accesscredentialpb.Credential_ACTIVE},
		},
		{
			name: "no mask replaces writable fields",
			req: &accesscredentialpb.UpdateCredentialRequest{
				Credential: &accesscredentialpb.Credential{Id: "5", Type: "Keyfob", Value: "333", Kind: accesscredentialpb.Credential_PIN},
			},
			wantPut:  `{"Id":5,"TokenType":"Keyfob","TokenValue":"333","IsLost":false}`,
			wantResp: &accesscredentialpb.Credential{Id: "5", Type: "Keyfob", Kind: accesscredentialpb.Credential_FOB, Value: "333", State: accesscredentialpb.Credential_ACTIVE},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, requests := setupCredentialServer(t, newFakeNet2(existing))
			got, err := s.UpdateCredential(context.Background(), tt.req)
			require.NoError(t, err)
			assert.Empty(t, cmpDiff(tt.wantResp, got))

			var methods []string
			for _, r := range *requests {
				methods = append(methods, r.method)
			}
			// read, full replace, read back what Net2 stored
			require.Equal(t, []string{http.MethodGet, http.MethodPut, http.MethodGet}, methods)
			assert.Equal(t, "/api/v1/users/7/tokens/5", (*requests)[1].path)
			assert.JSONEq(t, tt.wantPut, string((*requests)[1].body))
		})
	}
}

func TestCredentialServer_UpdateCredential_invalid(t *testing.T) {
	existing := UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"}
	tests := []struct {
		name string
		req  *accesscredentialpb.UpdateCredentialRequest
		code codes.Code
	}{
		{"nil", &accesscredentialpb.UpdateCredentialRequest{}, codes.InvalidArgument},
		{"read only field in mask", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", Kind: accesscredentialpb.Credential_FOB},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"kind"}},
		}, codes.InvalidArgument},
		{"unsupported field in mask", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", ExpireTime: timestamppb.Now()},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"expire_time"}},
		}, codes.InvalidArgument},
		{"unsupported field without mask", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", Type: "ProxCard", Value: "111", ExpireTime: timestamppb.Now()},
		}, codes.InvalidArgument},
		{"unsupported state", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_STOLEN},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}, codes.InvalidArgument},
		{"change to unknown type", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", Type: "Unspecified"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type"}},
		}, codes.InvalidArgument},
		{"clear value", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
		}, codes.InvalidArgument},
		{"no id", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{State: accesscredentialpb.Credential_LOST},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}, codes.InvalidArgument},
		{"malformed id", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "abc", State: accesscredentialpb.Credential_LOST},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}, codes.NotFound},
		{"missing", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "6", State: accesscredentialpb.Credential_LOST},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
		}, codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, requests := setupCredentialServer(t, newFakeNet2(existing))
			_, err := s.UpdateCredential(context.Background(), tt.req)
			assert.Equal(t, tt.code, status.Code(err), err)
			for _, r := range *requests {
				assert.NotEqual(t, http.MethodPut, r.method, "nothing should be written")
			}
		})
	}
}

// Leaving state out of an update must not re-activate a lost card.
func TestCredentialServer_UpdateCredential_unsetStateKeepsLost(t *testing.T) {
	tests := []struct {
		name string
		req  *accesscredentialpb.UpdateCredentialRequest
	}{
		{"no mask", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", Type: "ProxIsoCard", Value: "111"},
		}},
		{"state in mask", &accesscredentialpb.UpdateCredentialRequest{
			Credential: &accesscredentialpb.Credential{Id: "5", Type: "ProxIsoCard"},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type", "state"}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			net2 := newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111", IsLost: true})
			s, _ := setupCredentialServer(t, net2)
			got, err := s.UpdateCredential(context.Background(), tt.req)
			require.NoError(t, err)
			assert.Equal(t, accesscredentialpb.Credential_LOST, got.GetState())
			stored, _ := net2.token(testUserID, 5)
			assert.Equal(t, UserToken{ID: 5, TokenType: TokenTypeProxIsoCard, TokenValue: "111", IsLost: true}, stored)
		})
	}
}

func TestCredentialServer_UpdateCredential_duplicate(t *testing.T) {
	const value = "222"
	net2 := newFakeNet2(
		UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"},
		UserToken{ID: 6, TokenType: TokenTypeProxCard, TokenValue: value},
	)
	s, _ := setupCredentialServer(t, net2)
	_, err := s.UpdateCredential(context.Background(), &accesscredentialpb.UpdateCredentialRequest{
		Credential: &accesscredentialpb.Credential{Id: "5", Value: value},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err), err)
	assert.NotContains(t, err.Error(), value)
}

// Two updates to the same token, each changing a different field, must both be kept.
func TestCredentialServer_UpdateCredential_concurrent(t *testing.T) {
	net2 := newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"})
	// The first PUT waits until another GET arrives, which only happens if the updates overlap,
	// or for long enough that it would have.
	var gets atomic.Int32
	anotherGet := make(chan struct{})
	c, _, _ := setupTokenClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if gets.Add(1) == 2 {
				close(anotherGet)
			}
		case http.MethodPut:
			select {
			case <-anotherGet:
			case <-time.After(200 * time.Millisecond):
			}
		}
		net2.ServeHTTP(w, r)
	})
	s := newCredentialServer(c, testUserID, zap.NewNop())

	var wg sync.WaitGroup
	for _, req := range []*accesscredentialpb.UpdateCredentialRequest{
		{Credential: &accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_LOST}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}}},
		{Credential: &accesscredentialpb.Credential{Id: "5", Value: "222"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}}},
	} {
		wg.Go(func() {
			_, err := s.UpdateCredential(context.Background(), req)
			assert.NoError(t, err)
		})
	}
	wg.Wait()

	stored, _ := net2.token(testUserID, 5)
	assert.Equal(t, UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "222", IsLost: true}, stored)
}

// A token whose type this driver doesn't know can't be written back, which is the caller's problem, not Net2 being down.
func TestCredentialServer_UpdateCredential_unknownCurrentType(t *testing.T) {
	s, requests := setupCredentialServer(t, newFakeNet2(UserToken{ID: 5, TokenType: "AppleWallet", TokenValue: "111"}))
	_, err := s.UpdateCredential(context.Background(), &accesscredentialpb.UpdateCredentialRequest{
		Credential: &accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_LOST},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	for _, r := range *requests {
		assert.NotEqual(t, http.MethodPut, r.method, "nothing should be written")
	}
}

func TestCredentialServer_readMask(t *testing.T) {
	s, _ := setupCredentialServer(t, newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"}))
	ctx := context.Background()

	got, err := s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{Id: "5", ReadMask: &fieldmaskpb.FieldMask{Paths: []string{"id", "state"}}})
	require.NoError(t, err)
	assert.Empty(t, cmpDiff(&accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_ACTIVE}, got))

	for _, paths := range [][]string{{"more.x"}, {"no_such_field"}} {
		mask := &fieldmaskpb.FieldMask{Paths: paths}
		_, err := s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{Id: "5", ReadMask: mask})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "get %v: %v", paths, err)
		_, err = s.ListCredentials(ctx, &accesscredentialpb.ListCredentialsRequest{ReadMask: mask})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "list %v: %v", paths, err)
	}
}

// Every Credential field is either writable, ignored as output only, or rejected,
// so a field added to the proto can't be silently dropped.
func TestCheckUnsupportedFields(t *testing.T) {
	fields := (&accesscredentialpb.Credential{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		fd := fields.Get(i)
		name := string(fd.Name())
		t.Run(name, func(t *testing.T) {
			c := &accesscredentialpb.Credential{}
			m := c.ProtoReflect()
			switch {
			case fd.IsMap():
				m.Mutable(fd).Map().Set(protoreflect.ValueOfString("k").MapKey(), protoreflect.ValueOfString("v"))
			case fd.Message() != nil:
				m.Mutable(fd)
			case fd.Kind() == protoreflect.StringKind:
				m.Set(fd, protoreflect.ValueOfString("x"))
			case fd.Kind() == protoreflect.EnumKind:
				m.Set(fd, protoreflect.ValueOfEnum(1))
			case fd.Kind() == protoreflect.Int32Kind:
				m.Set(fd, protoreflect.ValueOfInt32(1))
			default:
				t.Fatalf("test doesn't know how to set a %v field", fd.Kind())
			}
			require.True(t, m.Has(fd))

			err := checkUnsupportedFields(c)
			supported := slices.Contains(credentialWritableFields.GetPaths(), name) || slices.Contains(credentialOutputOnlyFields, name)
			if supported {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
			}
		})
	}
}

func TestCredentialServer_ListCredentials_paging(t *testing.T) {
	// Net2 returns tokens in its own order; ids 2 and 10 check the order is numeric, not lexical.
	var tokens []UserToken
	for _, id := range []int{10, 2, 19, 1, 7, 3, 25} {
		tokens = append(tokens, UserToken{ID: id, TokenType: TokenTypeProxCard, TokenValue: strconv.Itoa(1000 + id)})
	}
	s, _ := setupCredentialServer(t, newFakeNet2(tokens...))

	var pages [][]string
	req := &accesscredentialpb.ListCredentialsRequest{PageSize: 3}
	for range 5 {
		res, err := s.ListCredentials(context.Background(), req)
		require.NoError(t, err)
		assert.EqualValues(t, 7, res.GetTotalSize())
		var ids []string
		for _, c := range res.GetCredentials() {
			ids = append(ids, c.GetId())
		}
		pages = append(pages, ids)
		if res.GetNextPageToken() == "" {
			break
		}
		req.PageToken = res.GetNextPageToken()
	}
	assert.Equal(t, [][]string{{"1", "2", "3"}, {"7", "10", "19"}, {"25"}}, pages)
}

// Net2 issues token ID 0 (e.g. to the first sample user), so it must be addressable.
func TestCredentialServer_tokenIDZero(t *testing.T) {
	net2 := newFakeNet2(UserToken{ID: 0, TokenType: TokenTypeUnspecified, TokenValue: "12340001"})
	s, _ := setupCredentialServer(t, net2)
	ctx := context.Background()

	got, err := s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{Id: "0"})
	require.NoError(t, err)
	assert.Equal(t, "0", got.GetId())

	got, err = s.UpdateCredential(ctx, &accesscredentialpb.UpdateCredentialRequest{
		Credential: &accesscredentialpb.Credential{Id: "0", State: accesscredentialpb.Credential_LOST},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	})
	require.NoError(t, err)
	assert.Equal(t, accesscredentialpb.Credential_LOST, got.GetState())

	_, err = s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "0"})
	require.NoError(t, err)
	assert.Empty(t, net2.users[testUserID])
}

func TestCredentialServer_DeleteCredential(t *testing.T) {
	net2 := newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"})
	s, _ := setupCredentialServer(t, net2)
	ctx := context.Background()

	_, err := s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "5"})
	require.NoError(t, err)
	assert.Empty(t, net2.users[testUserID])

	_, err = s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "5"})
	assert.Equal(t, codes.NotFound, status.Code(err), err)
	_, err = s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "5", AllowMissing: true})
	assert.NoError(t, err)
	_, err = s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "not-a-number", AllowMissing: true})
	assert.NoError(t, err)
}

// If Net2 deletes the token but replies with a 5xx, the retried DELETE gets a 404 for a delete that worked.
func TestCredentialServer_DeleteCredential_retried(t *testing.T) {
	net2 := newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"})
	var deletes atomic.Int32
	c, _, _ := setupTokenClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && deletes.Add(1) == 1 {
			net2.ServeHTTP(httptest.NewRecorder(), r) // delete it, then lose the reply
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		net2.ServeHTTP(w, r)
	})
	c.cli.RetryMax = 1
	c.cli.RetryWaitMin = time.Millisecond
	c.cli.RetryWaitMax = time.Millisecond
	s := newCredentialServer(c, testUserID, zap.NewNop())

	_, err := s.DeleteCredential(context.Background(), &accesscredentialpb.DeleteCredentialRequest{Id: "5"})
	require.NoError(t, err)
	assert.EqualValues(t, 2, deletes.Load(), "the DELETE should have been retried")
	_, held := net2.token(testUserID, 5)
	assert.False(t, held)
}

func TestCredentialServer_emptyID(t *testing.T) {
	s, requests := setupCredentialServer(t, newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"}))
	ctx := context.Background()
	_, err := s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	for _, allowMissing := range []bool{false, true} {
		_, err = s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{AllowMissing: allowMissing})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "allow_missing=%v: %v", allowMissing, err)
	}
	assert.Empty(t, *requests, "nothing should be sent to Net2")
}

func TestCredentialServer_errorCodes(t *testing.T) {
	tests := []struct {
		status int
		code   codes.Code
	}{
		{http.StatusNotFound, codes.NotFound},
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusInternalServerError, codes.Unavailable},
		{http.StatusServiceUnavailable, codes.Unavailable},
		{http.StatusForbidden, codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			net2 := newFakeNet2(UserToken{ID: 5, TokenType: TokenTypeProxCard, TokenValue: "111"})
			s, _ := setupCredentialServer(t, net2)
			ctx := context.Background()

			calls := map[string]func() error{
				"get": func() error {
					_, err := s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{Id: "5"})
					return err
				},
				"list": func() error {
					_, err := s.ListCredentials(ctx, &accesscredentialpb.ListCredentialsRequest{})
					return err
				},
				"create": func() error {
					_, err := s.CreateCredential(ctx, &accesscredentialpb.CreateCredentialRequest{Credential: &accesscredentialpb.Credential{Type: "ProxCard", Value: "222"}})
					return err
				},
				"update": func() error {
					_, err := s.UpdateCredential(ctx, &accesscredentialpb.UpdateCredentialRequest{
						Credential: &accesscredentialpb.Credential{Id: "5", State: accesscredentialpb.Credential_LOST},
						UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
					})
					return err
				},
				"delete": func() error {
					_, err := s.DeleteCredential(ctx, &accesscredentialpb.DeleteCredentialRequest{Id: "5"})
					return err
				},
			}
			for name, call := range calls {
				net2.fail(tt.status)
				err := call()
				assert.Equal(t, tt.code, status.Code(err), "%s: %v", name, err)
			}
		})
	}
}

// Net2 names the card number in its error body; none of it should reach a caller.
func TestCredentialServer_valueNotInErrors(t *testing.T) {
	const value = "55554444"
	errBody := func(w http.ResponseWriter, code int) {
		writeJSON(w, code, map[string]string{"message": "Card " + value + " is not acceptable"})
	}
	for _, code := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			c, _, _ := setupTokenClient(t, func(w http.ResponseWriter, _ *http.Request) { errBody(w, code) })
			s := newCredentialServer(c, testUserID, zap.NewNop())
			ctx := context.Background()

			_, err := s.CreateCredential(ctx, &accesscredentialpb.CreateCredentialRequest{Credential: &accesscredentialpb.Credential{Type: "ProxCard", Value: value}})
			require.Error(t, err)
			assert.NotContains(t, err.Error(), value)

			_, err = s.UpdateCredential(ctx, &accesscredentialpb.UpdateCredentialRequest{Credential: &accesscredentialpb.Credential{Id: "5", Type: "ProxCard", Value: value}})
			require.Error(t, err)
			assert.NotContains(t, err.Error(), value)

			_, err = s.GetCredential(ctx, &accesscredentialpb.GetCredentialRequest{Id: "5"})
			require.Error(t, err)
			assert.NotContains(t, err.Error(), value)
		})
	}
}

// Checks the trait is only announced when enabled, and that requests are routed to the right Net2 user by name.
func TestDriver_refreshCardholders_credentials(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			net2 := &fakeNet2{users: map[int][]UserToken{
				7: {{ID: 1, TokenType: TokenTypeProxCard, TokenValue: "7001"}},
				8: {{ID: 2, TokenType: TokenTypeKeyfob, TokenValue: "8002"}},
			}}
			client, _, _ := setupTokenClient(t, net2.ServeHTTP)
			n := node.New("test")
			d := &Driver{client: client, logger: zap.NewNop()}
			cfg := config.Root{CardHolderPrefix: "paxton", EnableCredentialManagement: enabled}
			ctx := context.Background()
			require.NoError(t, d.refreshCardholders(ctx, n, cfg))

			dev, err := n.GetDevice("paxton/cardholder/8")
			require.NoError(t, err)
			var traits []string
			for _, tm := range dev.GetMetadata().GetTraits() {
				traits = append(traits, tm.GetName())
			}
			assert.Contains(t, traits, string(accesspb.TraitName))
			if !enabled {
				assert.NotContains(t, traits, string(accesscredentialpb.TraitName))
				return
			}
			assert.Contains(t, traits, string(accesscredentialpb.TraitName))

			api := accesscredentialpb.NewAccessCredentialApiClient(n.ClientConn())
			res, err := api.ListCredentials(ctx, &accesscredentialpb.ListCredentialsRequest{Name: "paxton/cardholder/8"})
			require.NoError(t, err)
			require.Len(t, res.GetCredentials(), 1)
			assert.Equal(t, "8002", res.GetCredentials()[0].GetValue())

			info := accesscredentialpb.NewAccessCredentialInfoClient(n.ClientConn())
			support, err := info.DescribeCredential(ctx, &accesscredentialpb.DescribeCredentialRequest{Name: "paxton/cardholder/7"})
			require.NoError(t, err)
			assert.NotEmpty(t, support.GetTypes())
		})
	}
}

func cmpDiff(want, got proto.Message) string {
	return cmp.Diff(want, got, protocmp.Transform())
}
