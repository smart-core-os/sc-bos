package cloud

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// Client abstracts the device API surface used by the deployment loop and the
// renewal loop.
type Client interface {
	CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error)
	DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error)
	// Renew exchanges a fresh CSR (and key) for a new certificate over the
	// authenticated (mTLS) connection, returning the new Registration. It does not
	// persist or swap it — the caller persists first, then calls SetRegistration.
	Renew(ctx context.Context) (*Registration, error)
	// SetRegistration swaps the certificate presented on subsequent mTLS calls,
	// without reconnecting or restarting (hot reload).
	SetRegistration(cred *Registration)
}

// Registration is the controller's cloud identity under mTLS together with the
// SCC API endpoint it enrolled against: an EC P-256 private key, the
// Connect-CA-issued certificate chain (leaf first), and the API origin. The
// endpoint is held (and persisted) alongside the key material so check-in and
// renewal always reach the server that issued the certificate — including after
// a restart, and when a per-enrolment register URL override was used.
type Registration struct {
	// APIEndpoint is the SCC API origin (scheme://host) this credential was
	// issued by and authenticates against.
	APIEndpoint string
	Key         pki.PrivateKey
	Chain       []*x509.Certificate // leaf first, followed by any intermediates/CA
}

// Leaf returns the controller's own (leaf) certificate.
func (r *Registration) Leaf() *x509.Certificate {
	return r.Chain[0]
}

// NodeID returns the SCC node id, which the CA sets as the leaf certificate's
// subject common name. It is stable across renewals.
func (r *Registration) NodeID() string {
	return r.Chain[0].Subject.CommonName
}

// TLSCertificate adapts the registration into a *tls.Certificate suitable for
// presentation as a client certificate. It presents the full chain (leaf plus
// any intermediates) so a server that trusts only the root can build the path.
func (r *Registration) TLSCertificate() *tls.Certificate {
	cert := &tls.Certificate{
		PrivateKey:  r.Key,
		Leaf:        r.Chain[0],
		Certificate: make([][]byte, 0, len(r.Chain)),
	}
	for _, x := range r.Chain {
		cert.Certificate = append(cert.Certificate, x.Raw)
	}
	return cert
}

// newRegistration validates that chain is non-empty, leaf-first, and that its
// leaf public key matches key, returning a Registration for the given API
// endpoint ready for use.
func newRegistration(key pki.PrivateKey, chain []*x509.Certificate, apiEndpoint string) (*Registration, error) {
	if len(chain) == 0 {
		return nil, fmt.Errorf("certificate chain is empty")
	}
	if err := pki.ValidKeyPair(chain[0].PublicKey, key); err != nil {
		return nil, fmt.Errorf("leaf certificate does not match private key: %w", err)
	}
	return &Registration{APIEndpoint: apiEndpoint, Key: key, Chain: chain}, nil
}

type httpClientConfig struct {
	plainHTTP          *http.Client
	serverRoots        *x509.CertPool
	insecureSkipVerify bool
}

// HTTPClientOption configures an HTTPClient.
type HTTPClientOption func(*httpClientConfig)

// WithHTTPClient overrides the client used for payload downloads (URLs that
// carry their own auth, e.g. blob SAS links).
func WithHTTPClient(c *http.Client) HTTPClientOption {
	return func(cfg *httpClientConfig) { cfg.plainHTTP = c }
}

// WithServerRootCAs sets the roots used to verify the SCC server certificate on
// both the mTLS and download transports. Primarily for tests, where the server
// presents a dev-CA cert rather than a public one; in production the system
// roots verify the public server certificate.
func WithServerRootCAs(pool *x509.CertPool) HTTPClientOption {
	return func(cfg *httpClientConfig) { cfg.serverRoots = pool }
}

// WithInsecureSkipVerify disables verification of the SCC server certificate on
// both transports. DEV ONLY — for talking to a cloudsim with an ephemeral dev
// CA. Never enable against a real SCC.
func WithInsecureSkipVerify() HTTPClientOption {
	return func(cfg *httpClientConfig) { cfg.insecureSkipVerify = true }
}

// HTTPClient implements Client for talking to a Smart Core Connect cloud API
// under mutual TLS. It presents the controller's client certificate on the
// check-in and renew endpoints and verifies the server against the system roots
// (or injected roots in tests).
type HTTPClient struct {
	checkInEndpoint string
	renewEndpoint   string
	apiOrigin       string // scheme://host of the API, for same-origin download detection

	// reg is the current registration, swapped atomically on renewal so the cert
	// presented on the mTLS connection hot-reloads without reconnecting.
	reg       atomic.Pointer[Registration]
	mtlsHTTP  *http.Client // presents the client cert (check-in, renew)
	plainHTTP *http.Client // downloads where the URL carries its own auth
}

// NewHTTPClient creates an HTTPClient for the SCC API rooted at apiOrigin, which
// must be an origin (scheme://host); the device endpoints are formed by
// appending their paths to it. reg is the current registration (may be nil until
// enrolled).
func NewHTTPClient(reg *Registration, apiOrigin string, opts ...HTTPClientOption) *HTTPClient {
	cfg := &httpClientConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	origin := strings.TrimRight(apiOrigin, "/")
	c := &HTTPClient{
		checkInEndpoint: origin + "/v1/device/check-in",
		renewEndpoint:   origin + "/v1/device/certificate/renew",
		apiOrigin:       origin,
	}
	c.reg.Store(reg)

	c.mtlsHTTP = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:           tls.VersionTLS12,
			RootCAs:              cfg.serverRoots, // nil => system roots
			InsecureSkipVerify:   cfg.insecureSkipVerify,
			GetClientCertificate: c.clientCertificate,
		},
	}}

	switch {
	case cfg.plainHTTP != nil:
		c.plainHTTP = cfg.plainHTTP
	case cfg.serverRoots != nil || cfg.insecureSkipVerify:
		c.plainHTTP = &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				RootCAs:            cfg.serverRoots,
				InsecureSkipVerify: cfg.insecureSkipVerify,
			},
		}}
	default:
		c.plainHTTP = http.DefaultClient
	}
	return c
}

// SetRegistration swaps the certificate presented on subsequent mTLS calls.
func (c *HTTPClient) SetRegistration(reg *Registration) { c.reg.Store(reg) }

// clientCertificate is the tls.Config GetClientCertificate callback; it presents
// the current registration's certificate chain, read afresh each handshake.
func (c *HTTPClient) clientCertificate(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	reg := c.reg.Load()
	if reg == nil {
		return &tls.Certificate{}, nil // present no certificate
	}
	return reg.TLSCertificate(), nil
}

// CloseIdleConnections closes any pooled idle keep-alive connections on both
// transports. Called when this client is being discarded (re-enrollment, unlink).
func (c *HTTPClient) CloseIdleConnections() {
	c.mtlsHTTP.CloseIdleConnections()
	c.plainHTTP.CloseIdleConnections()
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

	httpResp, err := c.mtlsHTTP.Do(httpReq)
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

// Renew generates a fresh key + CSR and exchanges it for a new certificate over
// the authenticated (mTLS) connection, returning the new Registration. The CSR
// subject is ignored by the CA on renewal (it keeps the same CN and credentialId).
func (c *HTTPClient) Renew(ctx context.Context) (*Registration, error) {
	cur := c.reg.Load()
	if cur == nil {
		return nil, ErrNotRegistered
	}
	// Renewal authenticates via the current client certificate (no bearer); the CA
	// ignores the CSR subject and keeps the same CN and credentialId.
	return enroll(ctx, c.mtlsHTTP, c.renewEndpoint, cur.NodeID(), "")
}

// DownloadPayload fetches the payload at the given URL.
// Caller must close the returned ReadCloser.
//
// If the client's endpoint is using HTTPS, then the provided URL must also use HTTPS.
func (c *HTTPClient) DownloadPayload(ctx context.Context, downloadURL string) (io.ReadCloser, error) {
	if strings.HasPrefix(c.apiOrigin, "https:") {
		if !strings.HasPrefix(downloadURL, "https:") {
			return nil, errInsecureDownloadURL
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	// Present the client certificate only when downloading from the API origin;
	// external (blob SAS) URLs carry their own auth and get the plain client.
	httpClient := c.plainHTTP
	if c.isOnAPIDomain(downloadURL) {
		httpClient = c.mtlsHTTP
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
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme+"://"+u.Host == c.apiOrigin
}

// OriginOf returns the scheme://host of rawURL, trimming any path. If rawURL
// cannot be parsed it is returned trimmed of a trailing slash as a best effort.
func OriginOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimRight(rawURL, "/")
	}
	return u.Scheme + "://" + u.Host
}

const maxCheckInBodySize = 1024 * 1024 // 1 MiB

var errInsecureDownloadURL = errors.New("insecure payload URL - must use https for downloads when configured with secure API server")
