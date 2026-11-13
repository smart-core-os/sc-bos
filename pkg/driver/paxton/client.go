package paxton

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
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

	// checkMu guards failing and is held across the systemCheck call, so reports reach the check in order.
	checkMu sync.Mutex
	// failing holds the latest error from each source (an endpoint path or authSource) that is currently failing.
	// The doors, users and events loops share one check, so one loop's success must not clear another's failure.
	failing map[string]error
}

// authSource is the source reported for access token failures.
const authSource = "auth"

// outcome is the result of one call to a source, for report.
type outcome struct {
	source string
	err    error
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

// report records the outcome of calls to each source and updates the system check from every source's latest
// outcome: running only when no source is failing, otherwise failed with every failure listed.
func (c *Client) report(outcomes ...outcome) {
	if c.systemCheck == nil {
		return
	}
	c.checkMu.Lock()
	defer c.checkMu.Unlock()
	var recorded bool
	for _, o := range outcomes {
		switch {
		case errors.Is(o.err, context.Canceled) || errors.Is(o.err, context.DeadlineExceeded):
			continue // says nothing about the source, the caller gave up
		case o.err != nil:
			if c.failing == nil {
				c.failing = make(map[string]error)
			}
			c.failing[o.source] = o.err
		default:
			delete(c.failing, o.source)
		}
		recorded = true
	}
	if !recorded {
		return
	}
	if len(c.failing) == 0 {
		c.systemCheck.MarkRunning()
		return
	}
	// sorted so the fault text, and so the check, only changes when a failure does
	sources := make([]string, 0, len(c.failing))
	for source := range c.failing {
		sources = append(sources, source)
	}
	slices.Sort(sources)
	msgs := make([]string, len(sources))
	for i, source := range sources {
		msgs[i] = fmt.Sprintf("%s: %v", source, c.failing[source])
	}
	c.systemCheck.MarkFailed(errors.New(strings.Join(msgs, "; ")))
}

// Do sends req with a valid access token, reporting the outcome against req's path on the system check.
func (c *Client) Do(ctx context.Context, req *retryablehttp.Request) (*http.Response, error) {
	return c.do(ctx, req, req.URL.Path)
}

// doBestEffort is Do for requests the driver can work without. Their failures are left to the caller to log:
// they are only retried when the caller next asks, so holding one on the check could keep it failed for good.
func (c *Client) doBestEffort(ctx context.Context, req *retryablehttp.Request) (*http.Response, error) {
	return c.do(ctx, req, "")
}

// do sends req, reporting the token outcome and, when source is not empty, the request outcome against source.
func (c *Client) do(ctx context.Context, req *retryablehttp.Request, source string) (*http.Response, error) {
	token, err := c.auth(ctx)
	if err != nil {
		c.report(outcome{authSource, err})
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", token))
	req.Header.Set("Accept", "application/json")

	reportReq := func(err error) {
		if source == "" {
			c.report(outcome{source: authSource})
		} else {
			c.report(outcome{source: authSource}, outcome{source, err})
		}
	}

	resp, err := c.cli.Do(req)
	if err != nil {
		reportReq(err)
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("unexpected status %d: %s", resp.StatusCode, readErrBody(resp.Body))
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Error("failed to close response body", zap.Error(closeErr))
		}
		reportReq(err)
		return nil, err
	}

	reportReq(nil)
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
	c.report(outcome{authSource, err})
	return token, err
}

type AuthResponse struct {
	AccessToken    string `json:"access_token"`
	TokenType      string `json:"token_type"`
	ExpiresIn      int    `json:"expires_in"` // seconds
	RefreshToken   string `json:"refresh_token"`
	ExpiryDatetime string `json:"expiry_datetime"`
}
