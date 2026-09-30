package paxton

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// tokenExpiryMargin is how long before the reported expiry the access token is treated
// as expired, leaving headroom for a request to complete before the real expiry.
const tokenExpiryMargin = 30 * time.Second

// defaultAuthTimeout bounds a request for a new access token, including retries.
// Every other request waits for it, and the pollers' contexts have no deadline of their own.
const defaultAuthTimeout = 30 * time.Second

var errAuthTimeout = errors.New("auth: timed out waiting for Net2 to issue an access token")

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

	// authLock guards the token fields above. It's a channel rather than a sync.Mutex so
	// callers waiting on a slow token request can give up when their context ends.
	authLock    chan struct{}
	authTimeout time.Duration
}

// NewClientFromConfig builds a Client with the HTTP retry and TLS settings the driver uses.
// It lets code outside the driver, such as projects importing this package, talk to Net2
// directly. cfg.Auth.Password may be set directly; ParseConfig and a password file aren't
// needed. GrantType and Scope default the same way ParseConfig defaults them.
// systemCheck may be nil. Do and GetAccessToken update it, the token methods don't.
func NewClientFromConfig(cfg config.Root, logger *zap.Logger, systemCheck service.SystemCheck) *Client {
	cli := retryablehttp.NewClient()
	cli.RetryMax = 3
	cli.RetryWaitMax = 10 * time.Second
	// Return the last response once retries run out, so a 5xx reaches Do as a *StatusError.
	cli.ErrorHandler = retryablehttp.PassthroughErrorHandler

	// The dial and TLS handshake timeouts match the pooled transport retryablehttp would otherwise use.
	// Proxy is left unset: Net2 has always been reached directly, not via HTTP_PROXY.
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
	if cfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	cli.HTTPClient.Transport = transport

	if cfg.Auth.GrantType == "" {
		cfg.Auth.GrantType = "password"
	}
	if cfg.Auth.Scope == "" {
		cfg.Auth.Scope = "offline_access"
	}

	return NewClient(cli, logger, cfg, systemCheck)
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

		authLock:    make(chan struct{}, 1),
		authTimeout: defaultAuthTimeout,
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

// Do sends an authenticated request to Net2 and reports the outcome to the system check,
// so any failure, including a non-2xx response, marks it failed.
// The driver's polls use Do, which makes them the system check's source of truth.
func (c *Client) Do(ctx context.Context, req *retryablehttp.Request) (*http.Response, error) {
	resp, err := c.do(ctx, req)
	c.updateSystemCheck(err)
	return resp, err
}

// do is Do without the system check update. The token methods use it: they're driven by
// callers, whose mistakes and permissions say nothing about the health of Net2, and a
// success must not clear a fault a poll has just reported.
func (c *Client) do(ctx context.Context, req *retryablehttp.Request) (*http.Response, error) {
	token, err := c.auth(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	req.Header.Set("Accept", "application/json")

	resp, err := c.cli.Do(req)
	if err != nil {
		closeUnusedBody(resp)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := &StatusError{StatusCode: resp.StatusCode, Body: readErrBody(resp.Body)}
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close response body", zap.Error(closeErr))
		}
		return nil, err
	}
	return resp, nil
}

// closeUnusedBody closes the body of a response returned alongside an error.
// retryablehttp.PassthroughErrorHandler can return both, for example when ctx ends just as a response arrives.
func closeUnusedBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}

// StatusError is returned by Do and the token methods when Net2 responds with a non-2xx status.
// Use errors.As to inspect StatusCode, for example to tell a 404 from a 400.
// A Client built by NewClientFromConfig returns one for a 5xx too, once retries run out.
type StatusError struct {
	StatusCode int
	// Body holds the start of the response body, for diagnostics.
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("unexpected status %d: %s", e.StatusCode, e.Body)
}

// auth ensures the access token is valid and returns it.
//
// The lock is held for the whole token request so that concurrent callers don't all
// refresh at once; they wait here and reuse the token fetched by the first caller.
// A caller stops waiting when ctx ends, and the token request itself gives up after authTimeout.
func (c *Client) auth(ctx context.Context) (string, error) {
	select {
	case c.authLock <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-c.authLock }()

	// Refresh slightly before the real expiry so a request can't be sent with a token
	// that expires in flight.
	if time.Now().Before(c.expiry.Add(-tokenExpiryMargin)) {
		return c.accessToken, nil
	}

	token, err := c.requestToken(ctx)
	if err != nil && ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
		// Our timeout, not the caller's: report it as a failure rather than a cancellation,
		// which the system check would ignore.
		err = errAuthTimeout
	}
	return token, err
}

// requestToken fetches a new access token from Net2. The caller must hold authLock.
func (c *Client) requestToken(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.authTimeout)
	defer cancel()

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
		closeUnusedBody(resp)
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
