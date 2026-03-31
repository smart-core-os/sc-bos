package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver/proxy/config"
)

func newOAuth2Credentials(cfg config.OAuth2, client *http.Client) (*oauth2Credentials, error) {
	secret, err := os.ReadFile(cfg.ClientSecretFile)
	if err != nil {
		return nil, err
	}

	return &oauth2Credentials{
		client:       client,
		url:          cfg.TokenEndpoint,
		clientId:     cfg.ClientID,
		clientSecret: strings.TrimSpace(string(secret)),
		reqLimit:     make(chan struct{}, 1),
	}, nil
}

// oauth2Credentials is a credentials.PerRPCCredentials which fetches OAuth 2 access tokens from an HTTP endpoint
// using the client credentials flow.
//
// Only one request to the endpoint may be in progress at a time - this is enforced using the reqLimit channel.
// The access token will be automatically refreshed once expired.
type oauth2Credentials struct {
	client       *http.Client
	url          string
	clientId     string
	clientSecret string

	reqLimit chan struct{} // used like a select-able mutex
	tokenM   sync.RWMutex  // protects below mutable variables
	token    string
	expires  time.Time
}

func (o *oauth2Credentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	token, valid := o.getCurrentToken()
	if valid {
		return requestMetadata(token), nil
	}

	// acquire request lock
	select {
	case o.reqLimit <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() {
		// this will never block
		<-o.reqLimit
	}()

	// it's possible another goroutine has updated the token in between the last check and acquiring the request lock
	token, valid = o.getCurrentToken()
	if valid {
		return requestMetadata(token), nil
	}

	token, expires, err := fetchNewToken(ctx, o.client, o.url, o.clientId, o.clientSecret)
	if err != nil {
		return nil, err
	}

	o.tokenM.Lock()
	o.token = token
	o.expires = expires
	o.tokenM.Unlock()

	return requestMetadata(token), nil
}

func (o *oauth2Credentials) RequireTransportSecurity() bool {
	return true
}

func (o *oauth2Credentials) getCurrentToken() (token string, valid bool) {
	o.tokenM.RLock()
	defer o.tokenM.RUnlock()
	return o.token, o.expires.After(time.Now())
}

func requestMetadata(token string) map[string]string {
	return map[string]string{"authorization": "Bearer " + token}
}

func fetchNewToken(ctx context.Context, client *http.Client, endpoint, clientId, secret string) (token string, expires time.Time, err error) {
	reqBody := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientId},
		"client_secret": {secret},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(reqBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("read response body: %w", err)
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("status %s: %s", resp.Status, respBody)
		return
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	err = json.Unmarshal(respBody, &parsed)
	if err != nil {
		err = fmt.Errorf("parse oauth2 response: %w", err)
		return
	}

	expires = time.Now().Add(time.Second * time.Duration(parsed.ExpiresIn))
	return parsed.AccessToken, expires, nil
}
