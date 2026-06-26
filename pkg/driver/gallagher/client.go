package gallagher

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

// All controllers share a single Client (and therefore a single Transport)
// and poll the same Gallagher host, often bursting on the same schedule
// boundary. The stdlib default of MaxIdleConnsPerHost == 2 closes the excess
// idle connections after each burst, so subsequent requests open a fresh
// TCP+TLS session every time. These limits keep connections pooled and reused.
const (
	maxIdleConns        = 100
	maxIdleConnsPerHost = 100
	idleConnTimeout     = 90 * time.Second
)

type Client struct {
	BaseURL     string
	HTTPClient  *http.Client
	ApiKey      string
	systemCheck service.SystemCheck
}

func newHttpClient(baseURL string, apiKey string, caPath string, certPath string, keyPath string, systemCheck service.SystemCheck) (*Client, error) {

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, _ := tls.LoadX509KeyPair(certPath, keyPath)

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{clientCert},
				},
				MaxIdleConns:        maxIdleConns,
				MaxIdleConnsPerHost: maxIdleConnsPerHost,
				IdleConnTimeout:     idleConnTimeout,
			},
		},
		ApiKey:      apiKey,
		systemCheck: systemCheck,
	}, nil
}

func (c *Client) getUrl(p string) string {
	return fmt.Sprintf("%s/%s", c.BaseURL, p)
}

func (c *Client) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "GGL-API-KEY "+c.ApiKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		service.UpdateSystemCheck(c.systemCheck, err)
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		service.UpdateSystemCheck(c.systemCheck, err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("response status: %d %s", resp.StatusCode, resp.Status)
		service.UpdateSystemCheck(c.systemCheck, err)
		return nil, err
	}
	service.UpdateSystemCheck(c.systemCheck, nil)
	return bytes, nil
}
