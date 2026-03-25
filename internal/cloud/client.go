package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Client abstracts the check-in API.
type Client interface {
	CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error)
	DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error)
}

// Credentials holds the information needed to authenticate with the SCC BOS-facing API.
// These fields match the registration endpoint response, so the struct can be deserialized directly
// from a persisted credentials file.
type Credentials struct {
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	TokenEndpoint   string `json:"token_endpoint"`
	CheckInEndpoint string `json:"check_in_endpoint"`
}

// HTTPClientOption configures an HTTPClient.
type HTTPClientOption func(*HTTPClient)

// WithHTTPClient sets the underlying http.Client (e.g. for tests).
func WithHTTPClient(c *http.Client) HTTPClientOption {
	return func(h *HTTPClient) {
		h.plainHTTP = c
	}
}

// HTTPClient implements Client for talking to a Smart Core Connect cloud API.
type HTTPClient struct {
	checkInEndpoint   string
	authenticatedHTTP *http.Client // attaches OAuth2 access tokens to requests
	plainHTTP         *http.Client // for downloads where the URL carries its own auth
}

// NewHTTPClient creates a new HTTPClient for talking to a Smart Core Connect cloud API.
func NewHTTPClient(creds Credentials, opts ...HTTPClientOption) *HTTPClient {
	c := &HTTPClient{
		checkInEndpoint: creds.CheckInEndpoint,
		plainHTTP:       http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}

	oauthConfig := clientcredentials.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		TokenURL:     creds.TokenEndpoint,
	}
	oauthCtx := context.WithValue(context.Background(), oauth2.HTTPClient, c.plainHTTP)
	c.authenticatedHTTP = oauthConfig.Client(oauthCtx)
	return c
}

// CheckIn sends a POST request to the check-in endpoint and returns the server response.
// A zero-valued req is valid; the server accepts an empty body.
//
// The error return may be an *APIError which contains additional details about the error response from the server.
func (c *HTTPClient) CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.checkInEndpoint, bytes.NewReader(body))
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.authenticatedHTTP.Do(httpReq)
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	// cap at 1 MiB — the check-in response is a small JSON payload
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, maxCheckInBodySize))
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var apiErr APIError
		decodeErr := json.Unmarshal(respBody, &apiErr)
		apiErr.StatusCode = httpResp.StatusCode
		return CheckInResponse{}, errors.Join(&apiErr, decodeErr)
	}

	var resp CheckInResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return CheckInResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return resp, nil
}

// DownloadPayload fetches the payload at the given URL.
// Caller must close the returned ReadCloser.
//
// If the client's endpoint is using HTTPS, then the provided URL must also use HTTPS.
func (c *HTTPClient) DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error) {
	if strings.HasPrefix(c.checkInEndpoint, "https:") {
		if !strings.HasPrefix(url, "https:") {
			return nil, errInsecureDownloadURL
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	// only include the Authorization header if the download URL is on the same domain as the API server, as the
	// credentials are only intended for that server
	httpClient := c.plainHTTP
	if c.isOnAPIDomain(url) {
		httpClient = c.authenticatedHTTP
	}
	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		_ = httpResp.Body.Close()
		return nil, fmt.Errorf("download config version: server returned status %d", httpResp.StatusCode)
	}

	return httpResp.Body, nil
}

func (c *HTTPClient) isOnAPIDomain(urlStr string) bool {
	// compare scheme + host (ignore path) of the client's checkInEndpoint and the provided URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	baseURL, err := url.Parse(c.checkInEndpoint)
	if err != nil {
		return false
	}
	return u.Scheme == baseURL.Scheme && u.Host == baseURL.Host
}

const maxCheckInBodySize = 1024 * 1024 // 1 MiB

var errInsecureDownloadURL = errors.New("insecure payload URL - must use https for downloads when configured with secure API server")
