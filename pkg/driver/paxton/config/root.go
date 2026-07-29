package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	// Auth holds the credentials used to get an access token from the Paxton API. Required.
	Auth Auth `json:"auth"`
	// BaseUrl is the base URL of the Paxton API, e.g. "https://paxton.example.com".
	BaseUrl string `json:"baseUrl,omitempty"`

	// DeviceNamePrefix is prepended to device names when announcing doors on the Smart Core node.
	DeviceNamePrefix string `json:"deviceNamePrefix,omitempty"`
	// CardHolderPrefix is prepended to cardholder names when announcing users on the Smart Core node.
	CardHolderPrefix string `json:"cardHolderPrefix,omitempty"`

	// DoorsInterval is how often the driver refreshes the list of doors from the API.
	// Defaults to 5 minutes.
	DoorsInterval *jsontypes.Duration `json:"doorsInterval,omitempty"`
	// EventsInterval is how often the driver polls the API for new access events when polling is enabled.
	// Defaults to 5 seconds.
	EventsInterval *jsontypes.Duration `json:"eventsInterval,omitempty"`
	// CardsInterval is how often the driver refreshes the list of cardholders from the API.
	// Defaults to 5 minutes.
	CardsInterval *jsontypes.Duration `json:"cardsInterval,omitempty"`

	// SeenEventsCleanupInterval controls how often the deduplication cache is swept for expired entries.
	// Defaults to 1 minute.
	SeenEventsCleanupInterval *jsontypes.Duration `json:"seenEventsCleanupInterval,omitempty"`
	// SeenEventsMaxAge controls how long event IDs are retained in the deduplication cache.
	// Defaults to 5 minutes.
	SeenEventsMaxAge *jsontypes.Duration `json:"seenEventsMaxAge,omitempty"`

	// EnableSecurityEvents enables the SecurityEvent trait, exposing access events via the Smart Core API.
	EnableSecurityEvents bool `json:"enableSecurityEvents,omitempty"`

	// DisablePolling disables REST API polling for events. Polling is enabled by default.
	DisablePolling bool `json:"disablePolling,omitempty"`

	// EnableSignalR enables SignalR live event streaming. Disabled by default.
	EnableSignalR bool `json:"enableSignalR,omitempty"`

	// InsecureSkipVerify skips TLS certificate verification for HTTPS requests. Use only in development.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// SecurityEventsName is the Smart Core node name under which security events are announced.
	// Required when EnableSecurityEvents is true.
	SecurityEventsName string `json:"securityEventsName,omitempty"`
}

type Auth struct {
	// Username is the Paxton operator username.
	Username string `json:"username"`
	// PasswordFile is the path to a file containing the operator password. Required.
	PasswordFile string `json:"passwordFile"`
	// Password is populated at runtime from PasswordFile and is never read from JSON.
	Password string `json:"-"`
	// GrantType is the OAuth2 grant type. Defaults to "password".
	GrantType string `json:"grantType"`
	// ClientId is the OAuth2 client ID registered with the Paxton server.
	ClientId string `json:"clientId"`
	// Scope is the OAuth2 scope requested. Defaults to "offline_access".
	Scope string `json:"scope"`
}

func ParseConfig(data []byte) (Root, error) {
	root := Root{}

	err := json.Unmarshal(data, &root)

	if err != nil {
		return Root{}, err
	}

	if root.BaseUrl == "" {
		return Root{}, fmt.Errorf("invalid config - baseUrl is required")
	}

	if root.Auth.PasswordFile == "" {
		return Root{}, fmt.Errorf("invalid config - auth.passwordFile is required")
	}

	pass, err := os.ReadFile(root.Auth.PasswordFile)
	if err != nil {
		return Root{}, fmt.Errorf("invalid config - reading auth.passwordFile: %w", err)
	}

	root.Auth.Password = string(bytes.TrimSpace(pass))

	if root.Auth.GrantType == "" {
		root.Auth.GrantType = "password"
	}

	if root.Auth.Scope == "" {
		root.Auth.Scope = "offline_access"
	}

	if root.EnableSecurityEvents && root.SecurityEventsName == "" {
		return Root{}, fmt.Errorf("invalid config - securityEventsName is required when enableSecurityEvents is true")
	}

	if root.EnableSecurityEvents && root.DisablePolling && !root.EnableSignalR {
		return Root{}, fmt.Errorf("invalid config - security events enabled but all event sources are disabled (disablePolling=true, enableSignalR=false)")
	}

	return root, nil
}
