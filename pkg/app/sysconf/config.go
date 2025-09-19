// Package sysconf provides system level configuration.
package sysconf

import (
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/vanti-dev/sc-bos/pkg/app/http"
	"github.com/vanti-dev/sc-bos/pkg/app/stores"
	"github.com/vanti-dev/sc-bos/pkg/auth/policy"
	"github.com/vanti-dev/sc-bos/pkg/auto"
	"github.com/vanti-dev/sc-bos/pkg/block"
	"github.com/vanti-dev/sc-bos/pkg/driver"
	"github.com/vanti-dev/sc-bos/pkg/system"
	"github.com/vanti-dev/sc-bos/pkg/util/jsontypes"
	"github.com/vanti-dev/sc-bos/pkg/util/netutil"
	"github.com/vanti-dev/sc-bos/pkg/zone"
)

// Load loads into dst any user supplied config from json files and CLI arguments. CLI arguments take precedence.
func Load(dst *Config) error {
	// We call LoadFromArgs twice because args can be used to specify config file paths,
	// but also args should override config values specified in json files.

	if _, err := LoadFromArgs(dst, os.Args[1:]...); err != nil {
		return err
	}
	if err := LoadAllFromJSON(dst); err != nil {
		return err
	}
	if err := LoadFromConfigFilesJSON(dst); err != nil {
		return err
	}
	if _, err := LoadFromArgs(dst, os.Args[1:]...); err != nil {
		return err
	}

	// do any post processing
	dst.Normalize()

	return nil
}

// Config configures how the controller should run.
type Config struct {
	ConfigDirs  []string `json:"-"` // Dirs to look in for system config files. Defaults to [".conf"]
	ConfigFiles []string `json:"-"` // Filenames we load in ConfigDirs for system config. Defaults to ["system.conf.json", "system.json"]

	// The smart core name of the controller.
	// Can be overridden by app config.
	Name string `json:"name,omitempty"`

	Logger      *zap.Config `json:"logger,omitempty"`
	ListenGRPC  string      `json:"listenGrpc,omitempty"`
	ListenHTTPS string      `json:"listenHttps,omitempty"`

	// Preferred host:port others use to connect to us.
	// Typically used when the controller constructs and shares its own address with others,
	// for example during enrollment or when producing download links.
	// Can optionally contain a port, otherwise the port from ListenGRPC/ListenHTTPS is used.
	GRPCAddr string `json:"grpcAddr,omitempty"` // Defaults to netutil.OutboundAddr
	HTTPAddr string `json:"httpAddr,omitempty"` // Defaults to GRPCAddr

	SANs []string `json:"sans,omitempty"` // Subject Alternative Names for the self-signed cert

	AppConfig []string `json:"appConfig,omitempty"` // defaults to [".conf/app.conf.json"]
	DataDir   string   `json:"dataDir,omitempty"`   // defaults to .data/

	Stores *stores.Config `json:"stores,omitempty"`

	StaticHosting []http.StaticHostingConfig `json:"staticHosting"`
	CertConfig    *Certs                     `json:"certs,omitempty"`
	Cors          http.CorsConfig            `json:"cors,omitempty"`

	DisablePprof bool `json:"disablePprof"` // don't register net/http/pprof handlers

	Health *Health `json:"health,omitempty"`

	Systems map[string]system.RawConfig `json:"systems,omitempty"`

	Policy     policy.Policy `json:"-"` // Override the policy used for RPC calls. Defaults to policy.Default
	PolicyMode PolicyMode    `json:"-"` // How to apply the policy. Unsafe and can disable security checks. Defaults to PolicyOn.

	Experimental *Experimental `json:"experimental,omitempty"`

	DriverFactories map[string]driver.Factory `json:"-"` // keyed by driver name
	AutoFactories   map[string]auto.Factory   `json:"-"` // keyed by automation type
	SystemFactories map[string]system.Factory `json:"-"` // keyed by system type
	ZoneFactories   map[string]zone.Factory   `json:"-"` // keyed by zone type
}

// DriverConfigBlocks returns a map of driver type to a block list that describes the config for that driver.
// It scans DriverFactories for factories that implement the BlockSource interface.
// Drivers that do not implement BlockSource are not included in the output.
func (c *Config) DriverConfigBlocks() map[string][]block.Block {
	return extractConfigBlocks(c, c.DriverFactories)
}

// AutoConfigBlocks returns a map of automation type to a block list that describes the config for that automation.
// It scans AutoFactories for factories that implement the BlockSource interface.
// Automations that do not implement BlockSource are not included in the output.
func (c *Config) AutoConfigBlocks() map[string][]block.Block {
	return extractConfigBlocks(c, c.AutoFactories)
}

// ZoneConfigBlocks returns a map of zone type to a block list that describes the config for that zone.
// It scans ZoneFactories for factories that implement the BlockSource interface.
// Zone types that do not implement BlockSource are not included in the output.
func (c *Config) ZoneConfigBlocks() map[string][]block.Block {
	return extractConfigBlocks(c, c.ZoneFactories)
}

// ExternalHTTPEndpoint returns the way that the server should be reached over HTTPS from the outside world.
// This can be overridden by setting HTTPAddr.
func (c *Config) ExternalHTTPEndpoint() (string, error) {
	return netutil.MergeHostPort(c.ListenHTTPS, c.HTTPAddr)
}

// ExternalGRPCEndpoint returns the way that the server should be reached over gRPC from the outside world.
// This can be overridden by setting GRPCAddr.
func (c *Config) ExternalGRPCEndpoint() (string, error) {
	return netutil.MergeHostPort(c.ListenGRPC, c.GRPCAddr)
}

func extractConfigBlocks[Factory any](c *Config, factories map[string]Factory) map[string][]block.Block {
	blocks := make(map[string][]block.Block)
	for name, factory := range factories {
		switch source := any(factory).(type) {
		case BlockSource:
			blocks[name] = source.ConfigBlocks()
		case BlockSource2:
			blocks[name] = source.ConfigBlocks(c)
		}
	}
	return blocks
}

// Certs encapsulates different settings used for loading and present certificates to clients and servers.
type Certs struct {
	KeyFile   string `json:"keyFile,omitempty"`
	CertFile  string `json:"certFile,omitempty"`
	RootsFile string `json:"rootsFile,omitempty"`

	HTTPCert     bool   `json:"httpCert,omitempty"` // have the https stack (grpc-web and hosting) use different pki.Source from the grpc stack
	HTTPKeyFile  string `json:"httpKeyFile,omitempty"`
	HTTPCertFile string `json:"httpCertFile,omitempty"`
}

type PolicyMode string

const (
	PolicyOn    PolicyMode = "on"    // Always check requests against the policy.
	PolicyOff   PolicyMode = "off"   // Never check requests against the policy, allow all requests.
	PolicyCheck PolicyMode = "check" // Check requests against the policy if the request has a token or client cert.
)

// Health configures the health check system.
type Health struct {
	// DBPath is the location of the SQLite database file used to store health check results.
	// Relative paths are relative to DataDir.
	// Defaults to "health/checks.sqlite3".
	DBPath string    `json:"dbPath,omitempty"`
	TTL    HealthTTL `json:"ttl,omitempty"` // how long to keep health check results
}

type HealthTTL struct {
	MinCount *int                `json:"minCount,omitempty"` // defaults to 1, setting to 0 can disable seeding
	MaxCount *int                `json:"maxCount,omitempty"` // defaults to no max count
	MaxAge   *jsontypes.Duration `json:"maxAge,omitempty"`   // defaults to 1 week
}

func Default() Config {
	logConf := zap.NewDevelopmentConfig()
	one := 1
	config := Config{
		ConfigDirs:  []string{".conf"},
		ConfigFiles: []string{"system.conf.json", "system.json"},

		Logger:      &logConf,
		ListenGRPC:  ":23557",
		ListenHTTPS: ":443",

		AppConfig: []string{".conf/app.conf.json"},
		DataDir:   ".data",

		Cors: http.CorsConfig{
			DebugMode: false,
			// todo: this should really default to the default host
			CorsOrigins: []string{"*"},
		},
		StaticHosting: []http.StaticHostingConfig{},

		Health: &Health{
			DBPath: "health/checks.sqlite3",
			TTL: HealthTTL{
				MinCount: &one,
				MaxAge:   &jsontypes.Duration{Duration: 7 * 24 * time.Hour},
			},
		},

		CertConfig: &Certs{
			KeyFile:      "grpc.key.pem",
			CertFile:     "grpc.cert.pem",
			RootsFile:    "grpc.roots.pem",
			HTTPCert:     false,
			HTTPKeyFile:  "", // while these have defaults, we can't specify them and still have the "turn on if specified" feature
			HTTPCertFile: "",
		},
		Policy:     nil,
		PolicyMode: PolicyOn,
	}
	config.Logger.DisableStacktrace = true // because it's annoying

	return config
}

// Experimental configures feature flags for experimental features.
// These features are not considered stable and may be changed. They are not recommended for production use, so
// are disabled by default.
type Experimental struct {
	Accounts bool `json:"accounts,omitempty"` // enable account management features
}

// Normalize adjusts c to apply defaults that are based on the values of other fields.
// Normalize should be called explicitly if not using Load.
func (c *Config) Normalize() {
	// set defaults for the external addresses by autodiscovery
	// gRPC address will default to the outbound address
	if addr, err := netutil.OutboundAddr(); err == nil {
		if grpcAddr, err := netutil.MergeHostPort(addr.String(), c.GRPCAddr); err == nil {
			c.GRPCAddr = grpcAddr
		}
	}
	// HTTP address will default to the gRPC address (but not the gRPC port)
	bareGrpcHost := netutil.StripPort(c.GRPCAddr)
	if httpAddr, err := netutil.MergeHostPort(bareGrpcHost, c.HTTPAddr); err == nil {
		c.HTTPAddr = httpAddr
	}

	if c.GRPCAddr == "" {
		if addr, err := netutil.OutboundAddr(); err == nil {
			c.GRPCAddr = addr.String()
		}
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = c.GRPCAddr
	}

	c.CertConfig = c.CertConfig.FillDefaults()
}

func (c *Certs) FillDefaults() *Certs {
	or := func(a *string, b string) {
		if *a == "" {
			*a = b
		}
	}

	// if the config specifies http key or cert file paths, assume they want to use it
	if c.HTTPKeyFile != "" || c.HTTPCertFile != "" {
		c.HTTPCert = true
	}
	or(&c.HTTPKeyFile, "https.key.pem")
	or(&c.HTTPCertFile, "https.cert.pem")

	return c
}

// BlockSource is an interface that can be implemented by a factory to provide a list of blocks that describe the config
// for that driver/automation/zone type.
// This is used to produce granular diffs for config changes.
type BlockSource interface {
	ConfigBlocks() []block.Block
}

// BlockSource2 is like BlockSource but allows the factory to receive the config as an argument.
type BlockSource2 interface {
	ConfigBlocks(cfg *Config) []block.Block
}
