package paxton

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

// TokenType is the kind of credential a Net2 user token represents.
// The values match the Net2 API enum, which it sends and accepts as strings.
type TokenType string

const (
	TokenTypeUnspecified                 TokenType = "Unspecified"
	TokenTypeProxCard                    TokenType = "ProxCard"
	TokenTypeProxIsoCard                 TokenType = "ProxIsoCard"
	TokenTypeKeyfob                      TokenType = "Keyfob"
	TokenTypeHandsFreeToken              TokenType = "HandsFreeToken"
	TokenTypeWatchprox                   TokenType = "Watchprox"
	TokenTypeProxIsoCardWithoutMagstripe TokenType = "ProxIsoCardWithoutMagstripe"
	TokenTypeVehicleNumberPlate          TokenType = "VehicleNumberPlate"
	TokenTypeHandsFreeKeyCard            TokenType = "HandsFreeKeyCard"
	TokenTypeFingerprintVerificationCard TokenType = "FingerprintVerificationCard"
	TokenTypeTelephoneCallerId           TokenType = "TelephoneCallerId"
)

var tokenTypes = []TokenType{
	TokenTypeUnspecified,
	TokenTypeProxCard,
	TokenTypeProxIsoCard,
	TokenTypeKeyfob,
	TokenTypeHandsFreeToken,
	TokenTypeWatchprox,
	TokenTypeProxIsoCardWithoutMagstripe,
	TokenTypeVehicleNumberPlate,
	TokenTypeHandsFreeKeyCard,
	TokenTypeFingerprintVerificationCard,
	TokenTypeTelephoneCallerId,
}

// Valid reports whether t is one of the token types Net2 knows about.
func (t TokenType) Valid() bool {
	return slices.Contains(tokenTypes, t)
}

// UserToken is a credential (card, fob, number plate, ...) assigned to a Net2 user.
// TokenValue is a credential, so avoid logging it.
type UserToken struct {
	// ID is assigned by Net2. It is ignored when adding a token.
	ID         int       `json:"Id,omitempty"`
	TokenType  TokenType `json:"TokenType"`
	TokenValue string    `json:"TokenValue"`
	// IsLost disables the token while keeping its record, unlike deleting it.
	IsLost bool `json:"IsLost"`
}

// ErrInvalidToken is wrapped by the error AddUserToken or UpdateUserToken returns when the
// token is rejected before anything is sent to Net2.
var ErrInvalidToken = errors.New("invalid token")

// errEmptyBody is returned by doTokenRequest when a 2xx response that should have a body has none.
var errEmptyBody = errors.New("empty response body")

func (t UserToken) validate() error {
	if t.TokenValue == "" {
		return fmt.Errorf("%w: value is required", ErrInvalidToken)
	}
	if !t.TokenType.Valid() {
		return fmt.Errorf("%w: unknown type %q", ErrInvalidToken, t.TokenType)
	}
	return nil
}

// GetUserTokens returns all the tokens assigned to the Net2 user with the given ID.
func (c *Client) GetUserTokens(ctx context.Context, userID int) ([]UserToken, error) {
	var tokens []UserToken
	err := c.doTokenRequest(ctx, http.MethodGet, nil, &tokens, userID)
	if errors.Is(err, errEmptyBody) {
		// Net2 answers an empty events query with no body rather than [], so allow the same here.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetUserToken returns a single token assigned to the Net2 user with the given ID.
func (c *Client) GetUserToken(ctx context.Context, userID, tokenID int) (UserToken, error) {
	var token UserToken
	if err := c.doTokenRequest(ctx, http.MethodGet, nil, &token, userID, tokenID); err != nil {
		return UserToken{}, err
	}
	return token, nil
}

// AddUserToken assigns a new token to the Net2 user with the given ID and returns it as
// created by Net2, including its ID. Any ID set on t is ignored.
//
// Net2 rejects a value that has already been issued with a 400, whose body names the
// value. The same 400 can follow a retry after Net2 created the token but replied 5xx.
func (c *Client) AddUserToken(ctx context.Context, userID int, t UserToken) (UserToken, error) {
	if err := t.validate(); err != nil {
		return UserToken{}, err
	}
	t.ID = 0
	var created UserToken
	if err := c.doTokenRequest(ctx, http.MethodPost, t, &created, userID); err != nil {
		return UserToken{}, err
	}
	return created, nil
}

// UpdateUserToken replaces the token with the given ID on the Net2 user.
// Set IsLost to disable a token without deleting it.
func (c *Client) UpdateUserToken(ctx context.Context, userID, tokenID int, t UserToken) error {
	if err := t.validate(); err != nil {
		return err
	}
	t.ID = tokenID
	return c.doTokenRequest(ctx, http.MethodPut, t, nil, userID, tokenID)
}

// DeleteUserToken removes the token with the given ID from the Net2 user.
func (c *Client) DeleteUserToken(ctx context.Context, userID, tokenID int) error {
	return c.doTokenRequest(ctx, http.MethodDelete, nil, nil, userID, tokenID)
}

// doTokenRequest sends a request to api/v1/users/{userID}/tokens[/{tokenID}].
// A non-nil body is sent as JSON; a non-nil out is decoded from the JSON response, and a
// response with no body is errEmptyBody.
// Unlike the driver's polls, token requests don't update the system check, see Client.do.
func (c *Client) doTokenRequest(ctx context.Context, method string, body, out any, userID int, tokenID ...int) error {
	elems := []string{"api", "v1", "users", strconv.Itoa(userID), "tokens"}
	for _, id := range tokenID {
		elems = append(elems, strconv.Itoa(id))
	}
	reqUrl, err := url.JoinPath(c.baseUrl, elems...)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, reqUrl, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	if out == nil {
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(out)
	if errors.Is(err, io.EOF) {
		return errEmptyBody
	}
	return err
}
