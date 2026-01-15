package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/block"
	"github.com/smart-core-os/sc-bos/pkg/block/mdblock"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const (
	DefaultTokenURL = "https://accounts-api.airthings.com/v1/token"
	DefaultBaseURL  = "https://ext-api.airthings.com"
	DefaultPoll     = time.Minute
)

var (
	DefaultScopes = []string{"read:device"}
)

type Root struct {
	driver.BaseConfig
	Auth    Auth   `json:"auth,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"` // default: "https://ext-api.airthings.com"

	Locations []Location `json:"locations,omitempty"`
}

type Location struct {
	// One of ID or Name must be set.
	// If both are set ID takes precedence.
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`

	Poll *jsontypes.Duration `json:"poll,omitempty"` // default: 1m

	Devices []Device `json:"devices,omitempty"`
}

type Device struct {
	ID       string           `json:"id,omitempty"`       // AirThings segment ID
	Name     string           `json:"name,omitempty"`     // Smart Core name
	Metadata *traits.Metadata `json:"metadata,omitempty"` // Announced metadata for this SC device
	// Traits lists the trait names to announce for this device.
	// Status is always announced.
	// See traits.go for supported trait names.
	Traits []string `json:"traits,omitempty"`
}

type Auth struct {
	ClientID string `json:"clientID,omitempty"`
	ClientSecret
	TokenURL string   `json:"tokenURL,omitempty"` // default: "https://accounts-api.airthings.com/v1/token"
	Scopes   []string `json:"scopes,omitempty"`   // default: ["read:device"]
}

// ClientSecret allows specifying a client secret either directly or via a file.
type ClientSecret struct {
	ClientSecret     string `json:"clientSecret,omitempty"`
	ClientSecretFile string `json:"clientSecretFile,omitempty"`
}

// URL returns the full API URL for the given path.
// Path should start with a slash.
//
// Example:
//
//	c.baseURL = "https://ext-api.airthings.com"
//	c.URL("/v1/locations")
//	// "https://ext-api.airthings.com/v1/locations"
//	c.URL("/v1/locations/%v/latest-samples", 123)
//	// "https://ext-api.airthings.com/v1/locations/123/latest-samples"
func (c Root) URL(p string, args ...any) string {
	b := c.BaseURL
	if b == "" {
		b = DefaultBaseURL
	}
	return fmt.Sprintf(b+p, args...)
}

func (a Auth) ClientCredentialsConfig() (clientcredentials.Config, error) {
	cc := clientcredentials.Config{
		ClientID:     a.ClientID,
		ClientSecret: "",
		TokenURL:     a.TokenURL,
		Scopes:       a.Scopes,
	}
	if cc.TokenURL == "" {
		cc.TokenURL = DefaultTokenURL
	}
	if len(cc.Scopes) == 0 {
		cc.Scopes = DefaultScopes
	}
	var err error
	cc.ClientSecret, err = a.ClientSecret.Read()
	if err != nil {
		return cc, fmt.Errorf("read client secret %w", err)
	}

	// validate
	if cc.ClientID == "" {
		return cc, fmt.Errorf("clientID is required")
	}
	if cc.ClientSecret == "" {
		return cc, fmt.Errorf("clientSecret is required")
	}

	return cc, nil
}

// Read returns the password, either from ClientSecret or ClientSecretFile.
func (c ClientSecret) Read() (string, error) {
	if c.ClientSecret != "" {
		return c.ClientSecret, nil
	}
	bs, err := os.ReadFile(c.ClientSecretFile)
	if err != nil {
		return "", fmt.Errorf("%w: %q", err, c.ClientSecretFile)
	}
	return strings.TrimSpace(string(bs)), nil
}

var Blocks = []block.Block{
	{
		Path: []string{"locations"},
		Key:  "id",
		Blocks: []block.Block{
			{
				Path: []string{"devices"},
				Key:  "id",
				Blocks: []block.Block{
					{Path: []string{"metadata"}, Blocks: mdblock.Categories},
				},
			},
		},
	},
}
