package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestSecureConnect_JSON(t *testing.T) {
	const in = `{
		"name": "bacnet-sc",
		"type": "bacnet",
		"secureConnect": {
			"primaryHubURI": "wss://hub.example.com:47808",
			"failoverHubURI": "wss://hub2.example.com:47808",
			"deviceUUID": "1b671a64-40d5-491e-99b0-da01ff1f3341",
			"vmac": "010203040506",
			"heartbeatInterval": "15s",
			"connectTimeout": "5s",
			"maxBVLCLength": 1497,
			"tls": {
				"certificates": [{"certificate": "/etc/bsc/cert.pem", "privateKey": "/etc/bsc/key.pem"}],
				"rootCAs": "/etc/bsc/ca.pem"
			}
		}
	}`

	root := Defaults()
	if err := json.Unmarshal([]byte(in), &root); err != nil {
		t.Fatal(err)
	}
	sc := root.SecureConnect
	if sc == nil {
		t.Fatal("SecureConnect is nil")
	}
	if sc.PrimaryHubURI != "wss://hub.example.com:47808" {
		t.Errorf("PrimaryHubURI = %q", sc.PrimaryHubURI)
	}
	if sc.HeartbeatInterval.Duration != 15*time.Second {
		t.Errorf("HeartbeatInterval = %v", sc.HeartbeatInterval.Duration)
	}
	if sc.MaxBVLCLength != 1497 {
		t.Errorf("MaxBVLCLength = %d", sc.MaxBVLCLength)
	}
	if len(sc.TLS.Certificates) != 1 {
		t.Fatalf("expected 1 TLS certificate, got %d", len(sc.TLS.Certificates))
	}

	// round-trips without error
	if _, err := json.Marshal(root); err != nil {
		t.Fatal(err)
	}

	if err := sc.Validate(); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}
}

func TestSecureConnect_Validate(t *testing.T) {
	clientCert := []jsontypes.TLSCertificate{{Certificate: "cert-pem", PrivateKey: "key-pem"}}
	fullTLS := jsontypes.TLSConfig{Certificates: clientCert, RootCAs: "ca-pem"}
	certNoCA := jsontypes.TLSConfig{Certificates: clientCert}

	cases := []struct {
		name    string
		in      SecureConnect
		wantErr bool
	}{
		{name: "missing primary", in: SecureConnect{}, wantErr: true},
		{name: "bad scheme", in: SecureConnect{PrimaryHubURI: "http://hub.example.com"}, wantErr: true},
		{name: "wss without client cert", in: SecureConnect{PrimaryHubURI: "wss://hub.example.com:47808"}, wantErr: true},
		{name: "wss cert but no CA and no skip", in: SecureConnect{PrimaryHubURI: "wss://hub.example.com:47808", TLS: certNoCA}, wantErr: true},
		{name: "wss cert no CA but skip verify", in: SecureConnect{PrimaryHubURI: "wss://hub.example.com:47808", TLS: jsontypes.TLSConfig{Certificates: clientCert, InsecureSkipVerify: true}}, wantErr: false},
		{name: "wss complete", in: SecureConnect{PrimaryHubURI: "wss://hub.example.com:47808", TLS: fullTLS}, wantErr: false},
		{name: "ws no tls allowed", in: SecureConnect{PrimaryHubURI: "ws://127.0.0.1:8080"}, wantErr: false},
		{name: "bad failover", in: SecureConnect{PrimaryHubURI: "ws://127.0.0.1:8080", FailoverHubURI: "nonsense"}, wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
