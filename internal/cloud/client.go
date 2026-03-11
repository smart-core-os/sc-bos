package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Client abstracts the check-in API.
type Client interface {
	CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error)
	DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error)
}

// HTTPClientOption configures an HTTPClient.
type HTTPClientOption func(*HTTPClient)

// WithHTTPClient sets the underlying http.Client (e.g. for tests).
func WithHTTPClient(c *http.Client) HTTPClientOption {
	return func(h *HTTPClient) {
		h.httpClient = c
	}
}

// HTTPClient implements Client using net/http.
type HTTPClient struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTPClient.
// The secret is used verbatim as the Bearer token value in authenticated requests to the server.
func NewHTTPClient(baseURL, secret string, opts ...HTTPClientOption) *HTTPClient {
	c := &HTTPClient{
		baseURL:    baseURL,
		secret:     secret,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/check-in", bytes.NewReader(body))
	if err != nil {
		return CheckInResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.secret)

	httpResp, err := c.httpClient.Do(httpReq)
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
func (c *HTTPClient) DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		_ = httpResp.Body.Close()
		return nil, fmt.Errorf("download config version: server returned status %d", httpResp.StatusCode)
	}

	return httpResp.Body, nil
}

const maxCheckInBodySize = 1024 * 1024 // 1 MiB
