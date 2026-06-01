// Package scclient holds the CLI flag plumbing shared by the bacnet-sc-* tools.
// Each tool registers Flags onto its own flag.FlagSet (or the default one) then
// calls NewClient after flag.Parse to dial the configured BACnet/SC hub.
package scclient

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/sc"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

// Flags holds the BACnet/SC connection flags shared by the bacnet-sc-* tools.
// Register binds them onto a flag.FlagSet.
type Flags struct {
	Hub      string
	Failover string
	Cert     string
	Key      string
	CA       string
	UUID     string
	VMAC     string
	Insecure bool
	JSON     bool
	Timeout  time.Duration
	Verbose  bool
}

// Register binds the flags onto fs. Pass flag.CommandLine to use the default set.
func Register(fs *flag.FlagSet) *Flags {
	f := &Flags{}
	fs.StringVar(&f.Hub, "hub", "", "Primary BACnet/SC hub URI (wss:// in production, ws:// for testing) - required")
	fs.StringVar(&f.Failover, "failover", "", "Failover hub URI (optional)")
	fs.StringVar(&f.Cert, "cert", "", "Path to this node's PEM client certificate (mutual TLS)")
	fs.StringVar(&f.Key, "key", "", "Path to this node's PEM private key (mutual TLS)")
	fs.StringVar(&f.CA, "ca", "", "Path to the PEM CA bundle that signed the hub's server certificate")
	fs.StringVar(&f.UUID, "uuid", "", "This node's BACnet/SC device UUID (RFC 4122); random if empty")
	fs.StringVar(&f.VMAC, "vmac", "", "This node's 6-octet VMAC as hex (e.g. 010203040506); random if empty")
	fs.BoolVar(&f.Insecure, "insecure", false, "Skip TLS verification of the hub (testing only)")
	fs.BoolVar(&f.JSON, "json", false, "Emit results as JSON instead of plain text")
	fs.DurationVar(&f.Timeout, "timeout", 30*time.Second, "Overall timeout for the operation")
	fs.BoolVar(&f.Verbose, "verbose", false, "Verbose logging from the SC client")
	return f
}

// Config builds the driver-level SecureConnect config from the parsed flags.
func (f *Flags) Config() (config.SecureConnect, error) {
	if f.Hub == "" {
		return config.SecureConnect{}, fmt.Errorf("-hub is required")
	}
	cfg := config.SecureConnect{
		PrimaryHubURI:  f.Hub,
		FailoverHubURI: f.Failover,
		DeviceUUID:     f.UUID,
		VMAC:           f.VMAC,
		TLS:            jsontypes.TLSConfig{InsecureSkipVerify: f.Insecure},
	}
	if f.Cert != "" || f.Key != "" {
		if f.Cert == "" || f.Key == "" {
			return cfg, fmt.Errorf("-cert and -key must be provided together")
		}
		certPath, err := absPath(f.Cert)
		if err != nil {
			return cfg, fmt.Errorf("-cert: %w", err)
		}
		keyPath, err := absPath(f.Key)
		if err != nil {
			return cfg, fmt.Errorf("-key: %w", err)
		}
		cfg.TLS.Certificates = []jsontypes.TLSCertificate{{
			Certificate: jsontypes.PEM(certPath),
			PrivateKey:  jsontypes.PEM(keyPath),
		}}
	}
	if f.CA != "" {
		caPath, err := absPath(f.CA)
		if err != nil {
			return cfg, fmt.Errorf("-ca: %w", err)
		}
		cfg.TLS.RootCAs = jsontypes.PEM(caPath)
	}
	return cfg, nil
}

// Logger returns the zap logger for the tool, controlled by -verbose.
func (f *Flags) Logger() *zap.Logger {
	if f.Verbose {
		l, _ := zap.NewDevelopment()
		return l
	}
	// Quiet by default - tools print their own output. Errors still surface via
	// the returned errors from sc.NewClient and request methods.
	return zap.NewNop()
}

// NewClient dials the configured hub and returns a connected client.
func (f *Flags) NewClient() (*sc.Client, error) {
	cfg, err := f.Config()
	if err != nil {
		return nil, err
	}
	return sc.NewClient(cfg, 0, f.Logger())
}

// absPath turns p into an absolute path so jsontypes.PEM treats it as a file
// reference (PEM.IsPath requires "./" or absolute paths).
func absPath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	return filepath.Abs(p)
}
