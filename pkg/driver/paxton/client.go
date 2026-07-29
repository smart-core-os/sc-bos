package paxton

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// tokenExpiryMargin is how long before the reported expiry the access token is treated
// as expired, leaving headroom for a request to complete before the real expiry.
const tokenExpiryMargin = 30 * time.Second

type Client struct {
	cli         *retryablehttp.Client
	logger      *zap.Logger
	systemCheck service.SystemCheck

	baseUrl   string
	username  string
	password  string
	grantType string
	clientId  string
	scope     string

	accessToken  string
	refreshToken string
	expiry       time.Time

	mtx sync.Mutex
}

func NewClient(cli *retryablehttp.Client, logger *zap.Logger, cfg config.Root, systemCheck service.SystemCheck) *Client {
	return &Client{
		cli:         cli,
		logger:      logger,
		systemCheck: systemCheck,

		baseUrl:   cfg.BaseUrl,
		username:  cfg.Auth.Username,
		password:  cfg.Auth.Password,
		grantType: cfg.Auth.GrantType,
		clientId:  cfg.Auth.ClientId,
		scope:     cfg.Auth.Scope,
	}
}

func (c *Client) updateSystemCheck(err error) {
	if c.systemCheck == nil {
		return
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}
	if err != nil {
		c.systemCheck.MarkFailed(err)
	} else {
		c.systemCheck.MarkRunning()
	}
}

func (c *Client) Do(ctx context.Context, req *retryablehttp.Request) (*http.Response, error) {
	token, err := c.auth(ctx)
	if err != nil {
		c.updateSystemCheck(err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	req.Header.Set("Accept", "application/json")

	resp, err := c.cli.Do(req)
	if err != nil {
		c.updateSystemCheck(err)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("unexpected status %d: %s", resp.StatusCode, readErrBody(resp.Body))
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close response body", zap.Error(closeErr))
		}
		c.updateSystemCheck(err)
		return nil, err
	}

	c.updateSystemCheck(nil)
	return resp, nil
}

// auth ensures the access token is valid and returns it.
//
// The mutex is held for the whole token request so that concurrent callers don't all
// refresh at once; they block here and reuse the token fetched by the first caller.
func (c *Client) auth(ctx context.Context) (string, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// Refresh slightly before the real expiry so a request can't be sent with a token
	// that expires in flight.
	if time.Now().Before(c.expiry.Add(-tokenExpiryMargin)) {
		return c.accessToken, nil
	}

	reqUrl, err := url.JoinPath(c.baseUrl, "api", "v1", "authorization", "tokens")
	if err != nil {
		return "", err
	}

	formData := url.Values{}
	formData.Set("grant_type", c.grantType)
	formData.Set("client_id", c.clientId)
	formData.Set("scope", c.scope)
	formData.Set("username", c.username)
	formData.Set("password", c.password)

	body := bytes.NewBufferString(formData.Encode())

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, reqUrl, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.cli.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth: unexpected status %d: %s", resp.StatusCode, readErrBody(resp.Body))
	}

	response := &AuthResponse{}
	if err = json.NewDecoder(resp.Body).Decode(response); err != nil {
		return "", err
	}

	c.accessToken = response.AccessToken
	c.refreshToken = response.RefreshToken

	c.expiry, err = time.Parse(time.RFC3339Nano, response.ExpiryDatetime)
	if err != nil {
		return "", err
	}

	return c.accessToken, nil
}

// GetAccessToken ensures the token is valid and returns the current access token.
// Used by the SignalR client which must pass the token as a query parameter.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	token, err := c.auth(ctx)
	c.updateSystemCheck(err)
	return token, err
}

type AuthResponse struct {
	AccessToken    string `json:"access_token"`
	TokenType      string `json:"token_type"`
	ExpiresIn      int    `json:"expires_in"` // seconds
	RefreshToken   string `json:"refresh_token"`
	ExpiryDatetime string `json:"expiry_datetime"`
}
