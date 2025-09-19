package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/open-policy-agent/opa/rego"
	"github.com/rs/cors"
	"github.com/timshannon/bolthold"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"github.com/vanti-dev/sc-bos/internal/account"
	"github.com/vanti-dev/sc-bos/internal/manage/devices"
	"github.com/vanti-dev/sc-bos/internal/node/nodeopts"
	"github.com/vanti-dev/sc-bos/internal/util/grpc/interceptors"
	"github.com/vanti-dev/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/vanti-dev/sc-bos/internal/util/pki"
	"github.com/vanti-dev/sc-bos/internal/util/pki/expire"
	"github.com/vanti-dev/sc-bos/pkg/app/appconf"
	"github.com/vanti-dev/sc-bos/pkg/app/files"
	http2 "github.com/vanti-dev/sc-bos/pkg/app/http"
	"github.com/vanti-dev/sc-bos/pkg/app/stores"
	"github.com/vanti-dev/sc-bos/pkg/app/sysconf"
	"github.com/vanti-dev/sc-bos/pkg/auth/policy"
	"github.com/vanti-dev/sc-bos/pkg/auth/token"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/devicespb"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
	"github.com/vanti-dev/sc-bos/pkg/manage/enrollment"
	"github.com/vanti-dev/sc-bos/pkg/node"
	"github.com/vanti-dev/sc-bos/pkg/task"
	"github.com/vanti-dev/sc-bos/pkg/util/netutil"
)

// Bootstrap will obtain a Controller in a ready-to-run state.
func Bootstrap(ctx context.Context, config sysconf.Config) (*Controller, error) {
	logger, err := config.Logger.Build()
	if err != nil {
		return nil, err
	}

	// create data dir if it doesn't exist
	err = os.MkdirAll(config.DataDir, 0750)
	if err != nil {
		return nil, err
	}

	// load the external config file if possible
	// TODO: pull config from manager publication
	var externalConf appconf.Config
	filesLoaded, err := appconf.LoadIncludes("", &externalConf, config.AppConfig)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// warn that file(s) couldn't be found, but continue with default config
			logger.Warn("failed to load some config", zap.Strings("paths", config.AppConfig), zap.Error(err), zap.Strings("filesLoaded", filesLoaded))
		} else {
			return nil, err
		}
	} else {
		// successfully loaded the config
		logger.Debug("loaded external config", zap.Strings("paths", config.AppConfig), zap.Strings("includes", externalConf.Includes), zap.Strings("filesLoaded", filesLoaded))
	}
	confStore, err := appconf.LoadStore(externalConf, appconf.Schema{
		Drivers:     config.DriverConfigBlocks(),
		Automations: config.AutoConfigBlocks(),
		Zones:       config.ZoneConfigBlocks(),
	}, files.Path(config.DataDir, configDirName), logger)
	if err != nil {
		return nil, err
	}
	initialConfig := confStore.Active()

	// rootNode grants both local (in process) and networked (via grpc.Server) access to controller apis.
	// If you have a device you want expose via a Smart Core API, rootNode is where you'd do that via Announce.
	// If you need to know the brightness of another controller device, rootNode.Clients is how you'd do that.
	cName := initialConfig.Name
	if cName == "" {
		cName = config.Name
	}

	// external store for devices so we can attach multiple resources to it,
	// like metadata and health checks.
	deviceStore := devicespb.NewCollection(resource.WithIDInterceptor(func(oldID string) (newID string) {
		if oldID == "" {
			return cName
		}
		return oldID
	}))
	rootNode := node.New(cName, nodeopts.WithStore(deviceStore))
	rootNode.Logger = logger.Named("node")

	var accountStore *account.Store
	if config.Experimental != nil && config.Experimental.Accounts {
		accountLogger := logger.Named("account")
		accountStore, err = account.OpenStore(ctx, files.Path(config.DataDir, accountsFile), accountLogger)
		if err != nil {
			return nil, fmt.Errorf("load accounts: %w", err)
		}
		rootNode.Announce(rootNode.Name(),
			node.HasServer[gen.AccountApiServer](gen.RegisterAccountApiServer, account.NewServer(accountStore, accountLogger.Named("server"))),
		)
	}

	// Setup a local database for storing non-critical data.
	// This is made available to automations and systems as a local cache, for example for lighting reports.
	dbPath := files.Path(config.DataDir, "db.bolt")
	db, err := bolthold.Open(dbPath, 0750, nil)
	if err != nil {
		logger.Warn("failed to open local database file - some system components may fail", zap.Error(err),
			zap.String("path", dbPath))
	}

	// configurable shared storage for more permanent data
	storesConfig := config.Stores
	if storesConfig == nil {
		storesConfig = &stores.Config{}
	}
	storesConfig.DataDir = config.DataDir
	storesConfig.Logger = logger.Named("stores")
	store := stores.New(storesConfig)

	certConfig := config.CertConfig
	// Create a private key if it doesn't exist.
	// This key is used by the controller for incoming and outgoing connections, and as part of the enrolment process.
	key, keyPEM, err := pki.LoadOrGeneratePrivateKey(files.Path(config.DataDir, certConfig.KeyFile), logger)
	if err != nil {
		return nil, err
	}

	// enrollServer manages this controllers participation in a cohort.
	// When registered with a grpc.Server, the enrollServer will accept requests like CreateEnrollment which gives
	// this controller a new certificate for use during outgoing TLS connections to other cohort members.
	// In addition the enrollment will also include details of the trusted root certs so this controller can validate
	// incoming connections that contain a client certificate.
	//
	// enrollServer implements pki.Source providing these features without any extra work to setup.
	enrollServer, err := enrollment.LoadOrCreateServer(files.Path(config.DataDir, "enrollment"), keyPEM, logger.Named("enrollment"))
	if err != nil {
		return nil, err
	}

	// fileSource attempts to load a certificate and trust roots from disk.
	// The certificates public key must be paired with private key `key` loaded above.
	fileSource := pki.CacheSource(
		pki.FSKeySource(
			files.Path(config.DataDir, certConfig.CertFile), key,
			files.Path(config.DataDir, certConfig.RootsFile)),
		expire.BeforeInvalid(time.Hour),
	)

	// systemSource allows systems to contribute certificates to incoming and outgoing gRPC connections.
	systemSource := &pki.SourceSet{}

	// selfSignedSource creates a self signed certificate.
	// The certificates public key will be paired with and signed by `key`.
	ssCommonName := rootNode.Name()
	if ssCommonName == "" {
		ssCommonName = "localhost"
	}
	selfSignedOpts := []pki.CSROption{
		pki.WithExpireAfter(30 * 24 * time.Hour),
		pki.WithIfaces(),
	}
	if config.GRPCAddr != "" {
		selfSignedOpts = append(selfSignedOpts, pki.WithSAN(netutil.StripPort(config.GRPCAddr)))
	}
	for _, s := range config.SANs {
		selfSignedOpts = append(selfSignedOpts, pki.WithSAN(netutil.StripPort(s)))
	}
	selfSignedSource := pki.CacheSource(
		pki.SelfSignedSourceT(key, &x509.Certificate{
			Subject:               pkix.Name{CommonName: ssCommonName, Organization: []string{"Smart Core BOS"}},
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: true,
		}, selfSignedOpts...),
		expire.AfterProgress(0.5),
		pki.WithFSCache(files.Path(config.DataDir, "grpc-self-signed.cert.pem"), "", key),
	)

	// grpcSource is used by both incoming and outgoing grpc connections.
	// The server present the sources certificate and any intermediates between it and the roots to clients during TLS handshake.
	// If an incoming connection presents a client cert then it will be validated against the roots.
	// Outgoing connections will present the sources certificate as a client cert for validation by the remote party.
	// Config can indicate that different certs be used for grpc and https (inc grpc-web)
	grpcSource := &pki.SourceSet{
		enrollServer,
		fileSource,
		systemSource,
		selfSignedSource,
	}
	tlsGRPCServerConfig := pki.TLSServerConfig(grpcSource)
	tlsGRPCClientConfig := pki.TLSClientConfig(grpcSource)

	// Certs used for https (hosting and grpc-web) can be different from the Smart Core native grpc endpoint,
	// mostly to allow support for trusted certs on the https interface and cohort managed certs for grpc requests.
	httpCertSource := grpcSource
	if certConfig.HTTPCert {
		fileSource := pki.CacheSource(
			pki.FSSource(
				files.Path(config.DataDir, certConfig.HTTPCertFile),
				files.Path(config.DataDir, certConfig.HTTPKeyFile),
				""),
			expire.After(30*time.Minute),
		)
		httpCertSource = &pki.SourceSet{
			fileSource,
			selfSignedSource, // reuse the same self signed cert from grpc requests
		}
	}
	tlsHTTPServerConfig := pki.TLSServerConfig(httpCertSource)
	tlsHTTPServerConfig.ClientAuth = tls.NoClientCert

	// manager represents a delayed connection to the cohort manager.
	// Using the manager connection when we aren't enrolled will result in RPC calls returning 'not resolved' errors or similar.
	// As soon as we get enrolled those connections will be updated with the current manager address and will start to work.
	manager := node.DialChan(ctx, enrollServer.ManagerAddress(ctx),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsGRPCClientConfig)))

	var grpcOpts []grpc.ServerOption
	grpcOpts = append(grpcOpts,
		grpc.Creds(credentials.NewTLS(tlsGRPCServerConfig)),
		grpc.ChainStreamInterceptor(interceptors.CorrectStreamInfo(rootNode)),
	)

	// tokenValidator validates access tokens as part of the authorisation of requests to our APIs.
	// Claims associated with the token are presented along with other information when processing policy files.
	// Systems contribute validators to this set supporting different sources of token.
	tokenValidator := &token.ValidatorSet{}

	// configure request authorisation, here we setup grpc interceptors that decide if a request is denied or not.
	logPolicyMode(config.PolicyMode, logger)
	httpAuth := func(next http.Handler) http.Handler {
		return next
	}
	if pol := configPolicy(config); pol != nil {
		interceptor := policy.NewInterceptor(pol,
			policy.WithLogger(logger.Named("policy")),
			policy.WithTokenVerifier(tokenValidator),
		)
		grpcOpts = append(grpcOpts,
			grpc.ChainUnaryInterceptor(interceptor.GRPCUnaryInterceptor()),
			grpc.ChainStreamInterceptor(interceptor.GRPCStreamingInterceptor()),
		)
		httpAuth = interceptor.HTTPInterceptor
	}

	// here we set up our support for runtime added RPCs.
	grpcOpts = append(grpcOpts, grpc.UnknownServiceHandler(rootNode.ServerHandler()))

	grpcServer := grpc.NewServer(grpcOpts...)

	reflectionServer := reflectionapi.NewServer(grpcServer, rootNode)
	reflectionServer.Register(grpcServer)

	gen.RegisterEnrollmentApiServer(grpcServer, enrollServer)

	// DevicesApi
	var devicesApiOpts []devices.Option
	// work out the url download links should be using for targeting this controller
	if hostPort, err := config.ExternalHTTPEndpoint(); err == nil {
		devicesApiOpts = append(devicesApiOpts, devices.WithDownloadUrlBase(url.URL{
			Scheme: "https",
			Host:   hostPort,
			Path:   "/dl/devices",
		}))
	} else {
		logger.Error("failed to determine external http endpoint - download URLs unavailable", zap.Error(err))
	}
	devicesApi := devices.NewServer(rootNode, devicesApiOpts...)
	devicesApi.Register(grpcServer)

	// HealthApi, HealthHistoryApi, and adding health checks to the DevicesApi
	checkRegistry, closeHealthStore, err := setupHealthRegistry(ctx, config, deviceStore, rootNode, logger.Named("health"))
	if err != nil {
		return nil, err
	}

	grpcWebServer := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		// services are dynamic, the grpc.Server doesn't know about them all
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)

	// HTTP endpoint setup
	mux := http.NewServeMux()
	// Devices API aux routes
	devicesApi.RegisterHTTPMux(mux)

	// Static site hosting
	for _, site := range config.StaticHosting {
		handler := http2.NewStaticHandler(site.FilePath)
		mux.Handle(site.Path, http.StripPrefix(site.Path, handler))
		logger.Info("Serving static site", zap.String("path", site.Path), zap.String("filePath", site.FilePath))
	}

	// Well known APIs
	// Allow getting/updating the log level at run time
	mux.Handle("/__/log/level", httpAuth(config.Logger.Level))
	// Get version information about this binary
	mux.Handle("/__/version", httpAuth(Version))
	if !config.DisablePprof {
		// pprof endpoints, see net/http/pprof init() for more details
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("GET /debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
		mux.Handle("GET /__/debug/pprof/", httpAuth(http.StripPrefix("/__", pprofMux)))
	}

	// configure CORS setup
	co := cors.New(cors.Options{
		AllowedOrigins:   config.Cors.CorsOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{http2.HeaderAllowOrigin, http2.HeaderAuthorization, http2.HeaderContentType},
		AllowedMethods:   []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		Debug:            config.Cors.DebugMode,
	})
	corsWrap := co.Handler(mux)

	httpServer := &http.Server{
		Addr:      config.ListenHTTPS,
		TLSConfig: tlsHTTPServerConfig,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(request) || grpcWebServer.IsAcceptableGrpcCorsRequest(request) {
				grpcWebServer.ServeHTTP(writer, request)
			} else {
				corsWrap.ServeHTTP(writer, request)
			}
		}),
	}

	c := &Controller{
		SystemConfig:     config,
		ControllerConfig: confStore,
		Enrollment:       enrollServer,
		Logger:           logger,
		Node:             rootNode,
		Devices:          gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		CheckRegistry:    checkRegistry,
		DeviceStore:      deviceStore,
		Tasks:            &task.Group{},
		Database:         db,
		Stores:           store,
		Accounts:         accountStore,
		TokenValidators:  tokenValidator,
		GRPCCerts:        systemSource,
		ReflectionServer: reflectionServer,
		PrivateKey:       key,
		Mux:              mux,
		GRPC:             grpcServer,
		HTTP:             httpServer,
		ClientTLSConfig:  tlsGRPCClientConfig,
		ManagerConn:      manager,
	}
	c.Defer(manager.Close)
	c.Defer(store.Close)
	c.Defer(closeHealthStore)
	return c, nil
}

// logPolicyMode lets the user know if they are using an insecure policy mode.
func logPolicyMode(mode sysconf.PolicyMode, logger *zap.Logger) {
	switch mode {
	case sysconf.PolicyOn: // don't log the default mode
	case sysconf.PolicyOff:
		logger.Warn("no request authorization will be performed (--policy-mode=off)")
	case sysconf.PolicyCheck:
		logger.Warn("unauthenticated requests will be permitted (--policy-mode=check)")
	default:
		// this shouldn't happen as unknown modes are caught in the config parsing
		logger.Warn("unknown policy mode", zap.String("mode", string(mode)))
	}
}

// configPolicy converts the given config into a policy.Policy.
// Returns nil if no policy should be applied.
func configPolicy(config sysconf.Config) policy.Policy {
	if config.PolicyMode == sysconf.PolicyOff {
		return nil
	}

	pol := config.Policy
	if pol == nil {
		pol = policy.Default(false)
	}

	// only invoke the policy if we have a token or certificate
	if config.PolicyMode == sysconf.PolicyCheck {
		oldPol := pol
		pol = policy.Func(func(ctx context.Context, query string, input policy.Attributes) (rego.ResultSet, error) {
			if !input.TokenPresent && !input.CertificatePresent {
				// No token or cert, so we don't need to check the policy.
				// Return that the policy is satisfied.
				// See [rego.ResultSet.Allowed] for what conditions we are satisfying.
				return rego.ResultSet{{
					Expressions: []*rego.ExpressionValue{{
						Value: true,
					}},
				}}, nil
			}
			return oldPol.EvalPolicy(ctx, query, input)
		})
	}
	return pol
}

type Controller struct {
	SystemConfig     sysconf.Config
	ControllerConfig *appconf.Store
	Enrollment       *enrollment.Server

	// services for drivers/automations
	Logger          *zap.Logger
	Node            *node.Node
	Devices         gen.DevicesApiClient
	DeviceStore     *devicespb.Collection // for low level control of devices
	Tasks           *task.Group
	Database        *bolthold.Store
	TokenValidators *token.ValidatorSet
	GRPCCerts       *pki.SourceSet
	Stores          *stores.Stores
	Accounts        *account.Store
	CheckRegistry   *healthpb.Registry

	ReflectionServer *reflectionapi.Server

	Mux  *http.ServeMux
	GRPC *grpc.Server
	HTTP *http.Server

	PrivateKey      pki.PrivateKey
	ClientTLSConfig *tls.Config
	ManagerConn     node.Remote

	deferred []Deferred
}

type Deferred func() error

// Defer indicates that the given Deferred should be executed when the Controllers Run method returns.
func (c *Controller) Defer(d Deferred) {
	c.deferred = append(c.deferred, d)
}

func (c *Controller) Run(ctx context.Context) (err error) {
	initialConfig := c.ControllerConfig.Active()
	defer func() {
		for _, d := range c.deferred {
			err = multierr.Append(err, d())
		}
	}()

	// metadata associated with the node itself
	// we don't support changing metadata while running
	c.Node.Announce(c.Node.Name(), node.HasMetadata(initialConfig.Metadata))

	group, ctx := errgroup.WithContext(ctx)
	if c.Enrollment != nil {
		group.Go(func() error {
			return c.Enrollment.AutoRenew(ctx)
		})
	}
	if c.SystemConfig.ListenGRPC != "" {
		group.Go(func() error {
			return ServeGRPC(ctx, c.GRPC, c.SystemConfig.ListenGRPC, 15*time.Second, c.Logger.Named("server.grpc"))
		})
	}
	if c.SystemConfig.ListenHTTPS != "" {
		group.Go(func() error {
			return ServeHTTPS(ctx, c.HTTP, 15*time.Second, c.Logger.Named("server.https"))
		})
	}

	// load and start the systems
	systemServices, err := c.startSystems()
	if err != nil {
		return err
	}
	announceSystemServices(c, systemServices, c.SystemConfig.SystemFactories)
	go logServiceMapChanges(ctx, c.Logger.Named("system"), systemServices)
	// load and start the drivers
	driverServices, err := c.startDrivers(initialConfig.Drivers)
	if err != nil {
		return err
	}
	announceServices(c, "drivers", driverServices, c.SystemConfig.DriverFactories, c.ControllerConfig.Drivers())
	go logServiceMapChanges(ctx, c.Logger.Named("driver"), driverServices)
	// load and start the automations
	autoServices, err := c.startAutomations(initialConfig.Automation)
	if err != nil {
		return err
	}
	announceAutoServices(c, autoServices, c.SystemConfig.AutoFactories)
	go logServiceMapChanges(ctx, c.Logger.Named("auto"), autoServices)
	// load and start the zones
	zoneServices, err := c.startZones(initialConfig.Zones)
	if err != nil {
		return err
	}
	announceServices(c, "zones", zoneServices, c.SystemConfig.ZoneFactories, c.ControllerConfig.Zones())
	go logServiceMapChanges(ctx, c.Logger.Named("zone"), zoneServices)

	err = multierr.Append(err, group.Wait())
	return
}

const (
	configDirName = "config"
	accountsFile  = "accounts.sqlite3"
)
