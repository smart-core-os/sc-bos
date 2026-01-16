package gallagher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

func newHttpClient(baseURL string, apiKey string, caPath string, certPath string, keyPath string) (*client, error) {

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{clientCert},
				},
			},
		},
		apiKey: apiKey,
	}, nil
}

func (c *client) getUrl(p string) string {
	return fmt.Sprintf("%s/%s", c.baseURL, p)
}

func (c *client) doRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "GGL-API-KEY "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status: %d %s", resp.StatusCode, resp.Status)
	}
	return bytes, nil
}
