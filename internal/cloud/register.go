package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RegisterOption configures Register.
type RegisterOption func(*registerConfig)

type registerConfig struct {
	httpClient *http.Client
}

// WithRegisterHTTPClient sets the HTTP client used for the registration request.
func WithRegisterHTTPClient(c *http.Client) RegisterOption {
	return func(rc *registerConfig) { rc.httpClient = c }
}

// Register calls the device registration endpoint at registerURL, exchanging
// the enrollment code (sent as a Bearer token) for client credentials.
// The returned Registration can be passed directly to NewHTTPClient.
func Register(ctx context.Context, enrollmentCode, registerURL, clientName string, opts ...RegisterOption) (Registration, error) {
	cfg := &registerConfig{httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(cfg)
	}

	body, err := json.Marshal(struct {
		ClientName string `json:"client_name"`
	}{ClientName: clientName})
	if err != nil {
		return Registration{}, fmt.Errorf("encode register request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL, bytes.NewReader(body))
	if err != nil {
		return Registration{}, fmt.Errorf("create register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+enrollmentCode)

	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		return Registration{}, fmt.Errorf("send register request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRegisterBodySize))
	if err != nil {
		return Registration{}, fmt.Errorf("read register response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		_ = json.Unmarshal(respBody, &apiErr)
		apiErr.StatusCode = resp.StatusCode
		return Registration{}, &apiErr
	}

	var reg Registration
	if err := json.Unmarshal(respBody, &reg); err != nil {
		return Registration{}, fmt.Errorf("decode register response: %w", err)
	}
	return reg, nil
}

const maxRegisterBodySize = 4 * 1024 // 4 KiB — registration response is tiny
