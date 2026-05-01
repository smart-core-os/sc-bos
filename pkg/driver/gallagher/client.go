package gallagher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/smart-core-os/sc-bos/pkg/task/service"
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
			},
		},
		ApiKey:      apiKey,
		systemCheck: systemCheck,
	}, nil
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
		c.updateSystemCheck(err)
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.updateSystemCheck(err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("response status: %d %s", resp.StatusCode, resp.Status)
		c.updateSystemCheck(err)
		return nil, err
	}
	c.updateSystemCheck(nil)
	return bytes, nil
}
