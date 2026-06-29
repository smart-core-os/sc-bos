package cloud

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
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

// WithRegisterInsecureSkipVerify makes the registration request skip verification
// of the SCC server certificate. DEV ONLY — for enrolling against a cloudsim with
// an ephemeral dev CA. Never enable against a real SCC.
func WithRegisterInsecureSkipVerify() RegisterOption {
	return func(rc *registerConfig) {
		rc.httpClient = &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true},
		}}
	}
}

// Register enrolls this controller at the device registration endpoint
// (registerURL), authorised by the one-time enrollment code (sent as a Bearer
// token). It generates an EC P-256 key locally, submits a CSR carrying nodeName
// in the subject, and returns the issued Registration (key + Connect-CA chain).
// The key never leaves the controller.
func Register(ctx context.Context, enrollmentCode, registerURL, nodeName string, opts ...RegisterOption) (*Registration, error) {
	cfg := &registerConfig{httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(cfg)
	}

	return enroll(ctx, cfg.httpClient, registerURL, nodeName, enrollmentCode)
}

// enroll generates a fresh EC P-256 key, builds a CSR carrying commonName, submits
// it to the enrollment endpoint at url, and returns the issued Registration (key +
// Connect-CA chain). bearer, when non-empty, authorises the request (the
// enrollment code at registration); renewal passes "" and is authorised by the
// client certificate the mTLS httpClient already presents. The key never leaves
// the controller. Shared by Register and HTTPClient.Renew.
func enroll(ctx context.Context, httpClient *http.Client, url, commonName, bearer string) (*Registration, error) {
	key, err := pki.GenerateECP256Key()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	csrDER, err := pki.CreateCSRDER(key, commonName)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}
	chain, err := postCSR(ctx, httpClient, url, csrDER, bearer)
	if err != nil {
		return nil, err
	}
	// The endpoint travels with the registration: OriginOf(url) is the API origin
	// that issued this certificate, so it is persisted and reused for check-in and
	// renewal — even across restarts and when a register-URL override was used.
	return newRegistration(key, chain, OriginOf(url))
}

// postCSR submits a DER PKCS#10 CSR to an enrollment endpoint and returns the
// issued certificate chain (leaf first). The protocol is inspired by RFC 7030
// (EST) but is not an EST implementation: the CSR body is base64-encoded DER
// with content type application/pkcs10, and the response is a PEM chain
// (application/pem-certificate-chain). bearer, when non-empty, is sent as the
// Authorization (used by registration with the enrollment code; renewal
// authenticates via mTLS and passes "").
func postCSR(ctx context.Context, httpClient *http.Client, url string, csrDER []byte, bearer string) ([]*x509.Certificate, error) {
	body := base64.StdEncoding.EncodeToString(csrDER)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/pkcs10")
	req.Header.Set("Accept", "application/pem-certificate-chain")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxEnrollBodySize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		_ = json.Unmarshal(respBody, &apiErr)
		apiErr.StatusCode = resp.StatusCode
		return nil, &apiErr
	}

	chain, err := pki.ParseCertificatesPEM(respBody)
	if err != nil {
		return nil, fmt.Errorf("parse certificate chain: %w", err)
	}
	if len(chain) == 0 {
		return nil, fmt.Errorf("certificate chain response contained no certificates")
	}
	return chain, nil
}

const maxEnrollBodySize = 64 * 1024 // a PEM EC cert chain is small
