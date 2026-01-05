package config

import (
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// Root describes the configuration available to the proxy driver.
type Root struct {
	driver.BaseConfig

	// Nodes represent Smart Core nodes that this controller is proxying.
	Nodes []Node `json:"nodes"`
}

// Node is a networked Smart Core node, identified by its host.
type Node struct {
	Host string `json:"host,omitempty"` // for accepted values see grpc.NewClient

	// TLS allows us to override the default enrollment managed TLS configuration.
	TLS TLS `json:"tls,omitempty"`

	// Name is the Smart Core name for the remote node.
	// Used when discovering traits.
	// If absent then the remote node should support empty/default names for Parent requests.
	Name string `json:"name,omitempty"`

	// Children is a list of devices that should be announced by the proxy server.
	// If empty then all devices on the proxy host will be available (provided the user has authorization)
	// Deprecated: use Devices instead.
	Children []Device `json:"children,omitempty"`

	// Devices is a list of devices that should be announced by the proxy server.
	// If empty then all devices on the proxy host will be available (provided the user has authorization)
	Devices []Device `json:"devices,omitempty"`

	// SkipChild skips associating the node and node children with this nodes parent trait.
	// When true this doesn't stop clients of this node from communicating with the child, but it does stop discovery of
	// this name from working.
	// Deprecated: use SkipDevice instead.
	SkipChild bool `json:"skipChild,omitempty"`

	// SkipDevice skips associating the node and node children with this nodes parent trait.
	// When true this doesn't stop clients of this node from communicating with the child, but it does stop discovery of
	// this name from working.
	SkipDevice bool `json:"skipDevice,omitempty"`

	// Traits defines the exact named traits we proxy for this remote.
	// If absent or empty the remote will be inspected using the Parent trait including all found traits.
	// Traits []Trait `json:"traits,omitempty"` // todo: support manual traits

	OAuth2 *OAuth2 `json:"oauth2,omitempty"`
}

// GetDevices returns the list of devices to proxy for this node.
// Use Devices if set, otherwise fall back to Children for backward compatibility.
func (n Node) GetDevices() []Device {
	if n.Devices != nil {
		return n.Devices
	}
	return n.Children
}

func (n Node) ShouldSkipDevice() bool {
	if n.SkipDevice {
		return true
	}
	return n.SkipChild
}

type TLS struct {
	// These override the default enrollment PKI and are useful for testing and if running without a management node.
	InsecureNoClientCert bool `json:"insecureNoClientCert,omitempty"` // don't present a client certificate when connecting to proxy servers
	InsecureSkipVerify   bool `json:"insecureSkipVerify,omitempty"`   // don't verify proxy server certificates
}

type Device struct {
	Name   string       `json:"name,omitempty"`
	Traits []trait.Name `json:"traits,omitempty"`
}

type OAuth2 struct {
	TokenEndpoint    string `json:"tokenEndpoint"`    // HTTP(S) URL of the OAuth 2 token endpoint
	ClientID         string `json:"clientId"`         // OAuth 2 client ID to authenticate with
	ClientSecretFile string `json:"clientSecretFile"` // Path to a file containing the OAuth 2 client secret
}
