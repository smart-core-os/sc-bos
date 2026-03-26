package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTBroker struct {
	Host         string `json:"host,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	PasswordFile string `json:"passwordFile,omitempty"`
	// CACertFile is the path to a PEM-encoded CA certificate file used to verify the broker's certificate.
	// If empty, the system root CAs are used.
	CACertFile string `json:"caCertFile,omitempty"`
	// ClientCertFile and ClientKeyFile are paths to PEM-encoded client certificate and private key files
	// for mutual TLS authentication. Both must be set together.
	ClientCertFile string `json:"clientCertFile,omitempty"`
	ClientKeyFile  string `json:"clientKeyFile,omitempty"`
}

func (b MQTTBroker) ClientOptions() (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.Host)
	password := b.Password
	if password == "" {
		if b.PasswordFile != "" {
			var passFileBody []byte
			passFileBody, err := os.ReadFile(b.PasswordFile)
			if err != nil {
				return nil, err
			}

			password = strings.TrimSpace(string(passFileBody))
		}
	}

	if password != "" { // allow connection without password if no password provided
		opts.SetPassword(password)
	}

	opts.SetUsername(b.Username)
	opts.SetOrderMatters(false)

	if tlsCfg, err := b.tlsConfig(); err != nil {
		return nil, err
	} else if tlsCfg != nil {
		opts.SetTLSConfig(tlsCfg)
	}

	return opts, nil
}

// tlsConfig returns a *tls.Config if any TLS fields are set, or nil if none are.
func (b MQTTBroker) tlsConfig() (*tls.Config, error) {
	if b.CACertFile == "" && b.ClientCertFile == "" && b.ClientKeyFile == "" {
		return nil, nil
	}

	cfg := &tls.Config{}

	if b.CACertFile != "" {
		caCert, err := os.ReadFile(b.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert %q: %w", b.CACertFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert %q", b.CACertFile)
		}
		cfg.RootCAs = pool
	}

	// If either client cert or key is set, both must be set. Load the client cert/key pair if provided.
	if b.ClientCertFile != "" || b.ClientKeyFile != "" {
		if b.ClientCertFile == "" || b.ClientKeyFile == "" {
			return nil, fmt.Errorf("clientCertFile and clientKeyFile must both be set")
		}
		cert, err := tls.LoadX509KeyPair(b.ClientCertFile, b.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading client cert/key: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}
