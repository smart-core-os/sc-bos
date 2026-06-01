package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// SecureConnect configures the driver to talk BACnet/SC (secure connect) instead
// of BACnet/IP. When Root.SecureConnect is non-nil the driver connects to a
// BACnet/SC hub over a secure websocket (wss) using mutual TLS, rather than
// opening a UDP socket.
//
// Devices are still configured under Root.Devices; on BACnet/SC they are normally
// located by Who-Is on their device instance id. Omit the device's `comm` block
// entirely (an empty `"comm": {}` is treated the same), as the hub routes messages
// by virtual MAC (VMAC) rather than IP address.
type SecureConnect struct {
	// PrimaryHubURI is the websocket URI of the primary BACnet/SC hub, e.g.
	// "wss://hub.example.com:47808". Required.
	PrimaryHubURI string `json:"primaryHubURI,omitempty"`
	// FailoverHubURI is an optional secondary hub the client fails over to when
	// the primary connection is lost.
	FailoverHubURI string `json:"failoverHubURI,omitempty"`

	// TLS configures the mutual-TLS connection to the hub. BACnet/SC mandates TLS
	// 1.2 or later; the client enforces a 1.2 minimum. Typically holds the node's
	// operational certificate + private key (Certificates) and the CA that signed
	// the hub's certificate (RootCAs).
	TLS jsontypes.TLSConfig `json:"tls,omitempty"`

	// DeviceUUID is this node's BACnet/SC device UUID (RFC 4122). If empty a random
	// UUID is generated on startup and logged.
	DeviceUUID string `json:"deviceUUID,omitempty"`
	// VMAC is this node's 6-octet virtual MAC address as 12 hex characters
	// (optionally colon separated), e.g. "010203040506". If empty a random VMAC is
	// generated on startup.
	VMAC string `json:"vmac,omitempty"`

	// MaxBVLCLength is the largest BVLC message this node will accept, advertised in
	// the connect handshake. Defaults to 1497 (the BACnet/SC minimum) if zero.
	MaxBVLCLength uint16 `json:"maxBVLCLength,omitempty"`
	// MaxNPDULength is the largest NPDU this node will accept, advertised in the
	// connect handshake. Defaults to 1497 if zero.
	MaxNPDULength uint16 `json:"maxNPDULength,omitempty"`

	// HeartbeatInterval is how often a heartbeat is sent on an idle connection to
	// keep it alive. Defaults to 30s if zero.
	HeartbeatInterval Duration `json:"heartbeatInterval"`
	// ConnectTimeout bounds the websocket dial plus connect handshake. Defaults to
	// 10s if zero.
	ConnectTimeout Duration `json:"connectTimeout"`
}

// Validate reports configuration errors that should fail driver startup with an
// actionable message, rather than surfacing later as obscure connection errors.
func (s *SecureConnect) Validate() error {
	if s.PrimaryHubURI == "" {
		return errors.New("secureConnect.primaryHubURI is required (the wss:// URI of the BACnet/SC hub)")
	}
	if err := validateHubURI("primaryHubURI", s.PrimaryHubURI); err != nil {
		return err
	}
	if s.FailoverHubURI != "" {
		if err := validateHubURI("failoverHubURI", s.FailoverHubURI); err != nil {
			return err
		}
	}

	// wss:// is mutual TLS: this node needs a client certificate, and a way to
	// verify the hub. ws:// (no TLS) is permitted for local testing only.
	if hubScheme(s.PrimaryHubURI) == "wss" {
		if len(s.TLS.Certificates) == 0 {
			return errors.New("secureConnect.tls.certificates is required for wss:// (BACnet/SC uses mutual TLS): provide this node's client certificate and private key")
		}
		for i, c := range s.TLS.Certificates {
			if c.Certificate == "" || c.PrivateKey == "" {
				return fmt.Errorf("secureConnect.tls.certificates[%d] needs both certificate and privateKey", i)
			}
		}
		if len(s.TLS.RootCAs) == 0 && !s.TLS.InsecureSkipVerify {
			return errors.New("secureConnect.tls.rootCAs is required to verify the hub (or set tls.insecureSkipVerify for testing only)")
		}
	}
	return nil
}

func validateHubURI(field, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("secureConnect.%s %q: %w", field, raw, err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("secureConnect.%s %q must use a ws:// or wss:// scheme", field, raw)
	}
	if u.Host == "" {
		return fmt.Errorf("secureConnect.%s %q is missing a host", field, raw)
	}
	return nil
}

func hubScheme(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		return u.Scheme
	}
	return ""
}
