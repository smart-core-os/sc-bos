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
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/rs/cors"
	"github.com/timshannon/bolthold"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-bos/internal/account"
	"github.com/smart-core-os/sc-bos/internal/audit"
	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/internal/download"
	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	opscloud "github.com/smart-core-os/sc-bos/internal/opsapi"
	"github.com/smart-core-os/sc-bos/internal/router"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/interceptors"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/interceptors/protopkg"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/internal/util/pki/expire"
	"github.com/smart-core-os/sc-bos/pkg/app/files"
	http2 "github.com/smart-core-os/sc-bos/pkg/app/http"
	"github.com/smart-core-os/sc-bos/pkg/app/logcapture"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/app/sysconf"
	"github.com/smart-core-os/sc-bos/pkg/auth/policy"
	"github.com/smart-core-os/sc-bos/pkg/auth/token"
	"github.com/smart-core-os/sc-bos/pkg/history/dataretention"
	"github.com/smart-core-os/sc-bos/pkg/manage/enrollment"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accountpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/ops/cloudpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/system/boot"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/netutil"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// Bootstrap will obtain a Controller in a ready-to-run state.
func Bootstrap(ctx context.Context, config sysconf.Config) (*Controller, error) {
	// Wrap the root logger with a dynamic capture core so the Log trait system plugin
	// can register itself to receive all log entries after the fact.
	capture := &logcapture.Core{}
	logger, err := config.Logger.Build(zap.WrapCore(capture.WrapCoreFunc()))
	if err != nil {
		return nil, err
	}

	if err = os.MkdirAll(config.DataDir, 0750); err != nil {
		return nil, err
	}

	ci, err := initCloud(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	confStore, err := loadAppConfig(ctx, config, ci, logger)
	if err != nil {
		return nil, err
	}
	initialConfig := confStore.Active()

	cName := initialConfig.Name
	if cName == "" {
		cName = config.Name
	}
	idOrNodeName := func(oldID string) string {
		if oldID == "" {
			return cName
		}
		return oldID
	}
	// deviceStore is an external store for devices, allowing multiple resources (metadata, health checks)
	// to be attached to each device entry independently.
	deviceStore := devicespb.NewCollection(
		resource.WithIDInterceptor(idOrNodeName),
		resource.WithNoDuplicates(),
	)
	// nodeRouter allows non-Announced services to also be served via rootNode.ClientConn().
	nodeRouter := router.New(router.WithKeyInterceptor(func(key string) (string, error) {
		return idOrNodeName(key), nil
	}))
	// rootNode grants both local (in-process) and networked (via grpc.Server) access to controller APIs.
	// Announce devices on rootNode to expose them via Smart Core APIs; use rootNode.Clients to call them.
	rootNode := node.New(cName, nodeopts.WithStore(deviceStore), nodeopts.WithRouter(nodeRouter))
	rootNode.Logger = logger.Named("node")

	var accountStore *account.Store
	if config.Experimental != nil && config.Experimental.Accounts {
		accountLogger := logger.Named("account")
		accountStore, err = account.OpenStore(ctx, files.Path(config.DataDir, accountsFile), accountLogger)
		if err != nil {
			return nil, fmt.Errorf("load accounts: %w", err)
		}
		rootNode.Announce(rootNode.Name(),
			node.HasServer[accountpb.AccountApiServer](accountpb.RegisterAccountApiServer, account.NewServer(accountStore, accountLogger.Named("server"))),
		)
	}

	// CloudConnectionApi is announced directly (not via a system plugin) so the Ops UI can always
	// read cloud status and initiate enrollment, even before any systems have started.
	rootNode.Announce(cName,
		node.HasServer[cloudpb.CloudConnectionApiServer](cloudpb.RegisterCloudConnectionApiServer, opscloud.NewCloudConnectionServer(ci.Conn, cName, ci.RegisterURL)),
	)

	dbPath := files.Path(config.DataDir, "db.bolt")
	db, err := bolthold.Open(dbPath, 0750, nil)
	if err != nil {
		logger.Warn("failed to open local database file - some system components may fail", zap.Error(err),
			zap.String("path", dbPath))
	}
	storesConfig := config.Stores
	if storesConfig == nil {
		storesConfig = &stores.Config{}
	}
	storesConfig.DataDir = config.DataDir
	storesConfig.Logger = logger.Named("stores")
	store := stores.New(storesConfig)

	pi, err := initPKI(config, cName, logger)
	if err != nil {
		return nil, err
	}

	// manager is a delayed connection to the cohort manager.
	// Using it before enrollment results in 'not resolved' RPC errors; it becomes functional
	// once enrolled as the manager address is updated automatically.
	manager := node.DialChan(ctx, pi.EnrollServer.ManagerAddress(ctx),
		grpc.WithTransportCredentials(credentials.NewTLS(pi.GRPCClient)))

	ai := initAuth(config, logger)

	grpcServer, reflectionServer := buildGRPCServer(rootNode, nodeRouter, pi, ai)

	downloadRouter, err := buildDownloadRouter(config, logger)
	if err != nil {
		return nil, err
	}
	devicesApi := buildDevicesAPI(rootNode, nodeRouter, downloadRouter)

	checkRegistry, closeHealthStore, err := setupHealthRegistry(ctx, config, deviceStore, rootNode, logger.Named("health"))
	if err != nil {
		return nil, err
	}

	mux := buildHTTPMux(config, downloadRouter, ai.HTTPAuth, logger)
	setupAuditLog(ctx, downloadRouter, rootNode, ai.AuditSetup, logger)
	httpServer := buildHTTPServer(config, pi, grpcServer, mux)

	logLevel := config.Logger.Level
	c := &Controller{
		SystemConfig:     config,
		ControllerConfig: confStore,
		Enrollment:       pi.EnrollServer,
		Cloud:            ci.Conn,
		Logger:           logger,
		LogCapture:       capture,
		LogLevel:         &logLevel,
		Node:             rootNode,
		Devices:          devicespb.NewDevicesApiClient(wrap.ServerToClient(devicespb.DevicesApi_ServiceDesc, devicesApi)),
		CheckRegistry:    checkRegistry,
		DeviceStore:      deviceStore,
		Tasks:            &task.Group{},
		Database:         db,
		Stores:           store,
		Accounts:         accountStore,
		TokenValidators:  ai.TokenValidator,
		GRPCCerts:        pi.SystemSource,
		ReflectionServer: reflectionServer,
		PrivateKey:       pi.Key,
		Mux:              mux,
		GRPC:             grpcServer,
		HTTP:             httpServer,
		DownloadRouter:   downloadRouter,
		ClientTLSConfig:  pi.GRPCClient,
		ManagerConn:      manager,
	}
	c.Defer(manager.Close)
	c.Defer(store.Close)
	c.Defer(closeHealthStore)
	c.Defer(ci.DataRoot.Close)
	if ai.Interceptor != nil {
		c.Defer(ai.Interceptor.Close)
	}
	if ai.AuditSetup != nil {
		c.Defer(ai.AuditSetup.Close)
	}
	c.Defer(ci.Store.Close)
	return c, nil
}

// cloudInfo holds results of cloud connection setup.
type cloudInfo struct {
	Conn        *cloud.Conn
	Store       *cloud.DeploymentStore
	DataRoot    *os.Root
	RegisterURL string
}

// pkiInfo holds results of PKI and TLS configuration.
type pkiInfo struct {
	Key          pki.PrivateKey
	EnrollServer *enrollment.Server
	SystemSource *pki.SourceSet
	GRPCServer   *tls.Config
	GRPCClient   *tls.Config
	HTTPServer   *tls.Config
}

// authInfo holds results of auth/policy/audit setup.
type authInfo struct {
	TokenValidator *token.ValidatorSet
	Interceptor    *policy.Interceptor
	AuditSetup     *audit.Setup
	HTTPAuth       func(http.Handler) http.Handler
}

// initCloud sets up the cloud connection. The Conn is always created; it is a no-op when unconfigured.
func initCloud(ctx context.Context, config sysconf.Config, logger *zap.Logger) (cloudInfo, error) {
	cloudDataDir := filepath.Join(config.DataDir, cloudDirName)
	if err := os.MkdirAll(cloudDataDir, 0750); err != nil {
		return cloudInfo{}, fmt.Errorf("failed to create cloud data directory: %w", err)
	}
	cloudDataRoot, err := os.OpenRoot(cloudDataDir)
	if err != nil {
		return cloudInfo{}, fmt.Errorf("failed to open cloud data directory: %w", err)
	}
	defer func() {
		if err != nil {
			_ = cloudDataRoot.Close()
		}
	}()

	var storeOptions []cloud.StoreOption
	var connOptions []cloud.ConnOption
	var registerURL string
	if config.Cloud != nil {
		storeOptions = append(storeOptions, cloud.WithPreserveDownloads(config.Cloud.PreserveDownloads))
		if config.Cloud.MaxDeploymentSizeMiB != 0 {
			storeOptions = append(storeOptions, cloud.WithMaxDeploymentSize(int64(config.Cloud.MaxDeploymentSizeMiB)*1024*1024))
		}
		registerURL = config.Cloud.RegisterURL
	}
	storeOptions = append(storeOptions, cloud.WithStoreLogger(logger.Named("cloud.store")))
	cloudStore := cloud.NewDeploymentStore(cloudDataRoot, storeOptions...)
	connOptions = append(connOptions, cloud.WithUpdaterOptions(cloud.WithLogger(logger.Named("cloud"))))

	regStorePath := filepath.Join(cloudDataDir, "registration.json")
	regStore := cloud.NewFileRegistrationStore(regStorePath)

	cloudConn, err := cloud.OpenConn(ctx, regStore, cloudStore, connOptions...)
	if err != nil {
		return cloudInfo{}, fmt.Errorf("failed to start cloud connection: %w", err)
	}

	return cloudInfo{
		Conn:        cloudConn,
		Store:       cloudStore,
		DataRoot:    cloudDataRoot,
		RegisterURL: registerURL,
	}, nil
}

func loadAppConfig(ctx context.Context, config sysconf.Config, ci cloudInfo, logger *zap.Logger) (ConfigStore, error) {
	if ci.Conn.State().Connectivity != cloud.Unconfigured {
		return loadCloudAppConfig(ctx, config, ci.Store, ci.Conn, logger)
	}
	return loadLocalAppConfig(config, logger)
}

func initPKI(config sysconf.Config, nodeName string, logger *zap.Logger) (pkiInfo, error) {
	certConfig := config.CertConfig

	// key is created on first run and reused across restarts; it is shared by all TLS sources
	// (enrollment, file-based, and self-signed) and is also used during the enrollment process.
	key, keyPEM, err := pki.LoadOrGeneratePrivateKey(files.Path(config.DataDir, certConfig.KeyFile), logger)
	if err != nil {
		return pkiInfo{}, err
	}

	// enrollServer manages this controller's participation in a cohort.
	// It accepts CreateEnrollment requests, issuing certificates for outgoing TLS connections to other cohort
	// members and providing trusted root certs so this controller can validate incoming client certificates.
	// enrollServer also implements pki.Source, contributing these certs to the TLS config with no extra wiring.
	enrollServer, err := enrollment.LoadOrCreateServer(files.Path(config.DataDir, "enrollment"), keyPEM, logger.Named("enrollment"))
	if err != nil {
		return pkiInfo{}, err
	}

	// fileSource loads a certificate and trust roots from disk; its public key must be paired with key.
	fileSource := pki.CacheSource(
		pki.FSKeySource(
			files.Path(config.DataDir, certConfig.CertFile), key,
			files.Path(config.DataDir, certConfig.RootsFile)),
		expire.BeforeInvalid(time.Hour),
	)

	// systemSource is intentionally empty here; system plugins contribute their own certificates at runtime
	// via Controller.GRPCCerts, enabling dynamic cert rotation without restarting the server.
	systemSource := &pki.SourceSet{}

	ssCommonName := nodeName
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
	// selfSignedSource is a fallback when no enrolled or file-based cert is available.
	// It is shared by both the gRPC and HTTP TLS stacks (see httpCertSource below).
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

	// grpcSource drives both incoming and outgoing gRPC TLS.
	// The server presents its certificate chain to clients; if a client presents a cert it is validated against the roots.
	// Outgoing connections present the same certificate as a client cert for mutual TLS with cohort members.
	// HTTP/gRPC-web can use a separate cert chain (httpCertSource below) to support externally-trusted HTTPS
	// while keeping cohort-managed certs for native gRPC.
	grpcSource := &pki.SourceSet{
		enrollServer,
		fileSource,
		systemSource,
		selfSignedSource,
	}
	tlsMinVersion, err := certConfig.ParseTLSMinVersion()
	if err != nil {
		return pkiInfo{}, fmt.Errorf("certs.tlsMinVersion: %w", err)
	}
	tlsVersionOpt := pki.WithMinVersion(tlsMinVersion)
	tlsGRPCServerConfig := pki.TLSServerConfig(grpcSource, tlsVersionOpt)
	tlsGRPCClientConfig := pki.TLSClientConfig(grpcSource, tlsVersionOpt)

	httpCertSource := pki.Source(grpcSource)
	if certConfig.HTTPCert {
		httpFileSource := pki.CacheSource(
			pki.FSSource(
				files.Path(config.DataDir, certConfig.HTTPCertFile),
				files.Path(config.DataDir, certConfig.HTTPKeyFile),
				""),
			expire.After(30*time.Minute),
		)
		httpCertSource = &pki.SourceSet{
			httpFileSource,
			selfSignedSource,
		}
	}
	tlsHTTPServerConfig := pki.TLSServerConfig(httpCertSource, tlsVersionOpt)
	tlsHTTPServerConfig.ClientAuth = tls.NoClientCert

	return pkiInfo{
		Key:          key,
		EnrollServer: enrollServer,
		SystemSource: systemSource,
		GRPCServer:   tlsGRPCServerConfig,
		GRPCClient:   tlsGRPCClientConfig,
		HTTPServer:   tlsHTTPServerConfig,
	}, nil
}

func initAuth(config sysconf.Config, logger *zap.Logger) authInfo {
	// tokenValidator is populated at runtime by system plugins, each registering support for a different
	// token issuer (e.g. Keycloak, local accounts). Claims from validated tokens are forwarded to the
	// policy engine alongside each request.
	tokenValidator := &token.ValidatorSet{}
	httpAuth := func(next http.Handler) http.Handler { return next }

	logPolicyMode(config.PolicyMode, logger)
	pol := configPolicy(config)

	var interceptor *policy.Interceptor
	var auditSetup *audit.Setup
	if al := config.AuditLog; al != nil {
		var err error
		auditSetup, err = audit.NewSetup(al.Filename, al.MaxSizeMB, al.MaxAgeDays, al.MaxBackups, al.Compress)
		if err != nil {
			// Non-fatal: the controller remains fully functional without a persistent audit log file.
			logger.Error("failed to open audit log file, continuing with in-memory audit only", zap.String("file", al.Filename), zap.Error(err))
		}
	}
	if pol != nil || auditSetup != nil {
		if pol == nil {
			logger.Warn("no access policy configured; running interceptor for audit logging only (all requests allowed)")
			pol = policy.AllowAll
		}
		opts := []policy.InterceptorOption{
			policy.WithLogger(logger.Named("policy")),
			policy.WithTokenVerifier(tokenValidator),
		}
		if auditSetup != nil {
			opts = append(opts, policy.WithAuditSink(auditSetup))
		}
		interceptor = policy.NewInterceptor(pol, opts...)
		httpAuth = interceptor.HTTPInterceptor
	}

	return authInfo{
		TokenValidator: tokenValidator,
		Interceptor:    interceptor,
		AuditSetup:     auditSetup,
		HTTPAuth:       httpAuth,
	}
}

func buildGRPCServer(rootNode *node.Node, nodeRouter *router.Router, pi pkiInfo, ai authInfo) (*grpc.Server, *reflectionapi.Server) {
	var grpcOpts []grpc.ServerOption

	// migrationInterceptor rewrites old unversioned service paths to their new versioned equivalents,
	// allowing old clients to communicate with new servers during a rolling upgrade.
	// It must be registered before CorrectStreamInfo so that paths are already rewritten
	// by the time CorrectStreamInfo records the canonical /service/method.
	migrationInterceptor := protopkg.NewOldToNewInterceptor()
	grpcOpts = append(grpcOpts,
		grpc.ChainUnaryInterceptor(migrationInterceptor.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(migrationInterceptor.StreamInterceptor()),
	)
	grpcOpts = append(grpcOpts,
		grpc.Creds(credentials.NewTLS(pi.GRPCServer)),
		grpc.ChainStreamInterceptor(interceptors.CorrectStreamInfo(rootNode)),
	)
	if ai.Interceptor != nil {
		grpcOpts = append(grpcOpts,
			grpc.ChainUnaryInterceptor(ai.Interceptor.GRPCUnaryInterceptor()),
			grpc.ChainStreamInterceptor(ai.Interceptor.GRPCStreamingInterceptor()),
		)
	}
	grpcOpts = append(grpcOpts, grpc.UnknownServiceHandler(rootNode.ServerHandler()))

	grpcServer := grpc.NewServer(grpcOpts...)
	reflectionServer := reflectionapi.NewServer(grpcServer, rootNode)
	reflectionServer.Register(nodeRouter)
	enrollmentpb.RegisterEnrollmentApiServer(nodeRouter, pi.EnrollServer)

	return grpcServer, reflectionServer
}

// where download router is mounted
const downloadPathPrefix = "/download"

// buildDownloadRouter constructs the shared signed-URL router used by every
// subsystem that issues capability-style download URLs (devices CSV exports,
// audit log exports, system log exports). All such subsystems register handlers
// on this single router and share its HMAC key and URL prefix.
func buildDownloadRouter(config sysconf.Config, logger *zap.Logger) (*download.Router, error) {
	downloadBaseURL := downloadPathPrefix
	if hostPort, err := config.ExternalHTTPEndpoint(); err == nil {
		downloadBaseURL = "https://" + hostPort + downloadPathPrefix
	} else {
		logger.Error("failed to determine external http endpoint - download URLs unavailable", zap.Error(err))
	}
	downloadKey, err := config.ResolveDownloadHMACKey(logger)
	if err != nil {
		return nil, err
	}
	return download.NewRouter(
		download.NewHMACSigner(downloadKey),
		download.WithBaseURL(downloadBaseURL),
		download.WithTTL(time.Hour),
	), nil
}

func buildDevicesAPI(rootNode *node.Node, nodeRouter *router.Router, downloadRouter *download.Router) *devices.Server {
	devicesApi := devices.NewServer(rootNode, devices.WithURLGenerator(downloadRouter))
	devicesApi.Register(nodeRouter)
	downloadRouter.Handle(devices.DownloadType, devicesApi)
	return devicesApi
}

func buildHTTPMux(config sysconf.Config, downloadRouter *download.Router, httpAuth func(http.Handler) http.Handler, logger *zap.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	// Shared download endpoint. Handlers compress their own responses when appropriate.
	mux.Handle(downloadPathPrefix+"/", downloadRouter)

	for _, site := range config.StaticHosting {
		handler := http2.NewStaticHandler(site.FilePath)
		mux.Handle(site.Path, http.StripPrefix(site.Path, handler))
		logger.Info("Serving static site", zap.String("path", site.Path), zap.String("filePath", site.FilePath))
	}

	mux.Handle("/__/log/level", httpAuth(config.Logger.Level))
	mux.Handle("/__/version", httpAuth(Version))
	if !config.DisablePprof {
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("GET /debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
		mux.Handle("GET /__/debug/pprof/", httpAuth(http.StripPrefix("/__", pprofMux)))
	}

	return mux
}

// setupAuditLog wires the audit log subsystem against the shared download router,
// starts the background metadata refresh goroutine, and announces the audit-log
// device on the node so it appears as a Log trait.
func setupAuditLog(ctx context.Context, downloadRouter *download.Router, rootNode *node.Node, auditSetup *audit.Setup, logger *zap.Logger) {
	if auditSetup == nil {
		return
	}
	auditSrv := auditSetup.NewModelServer(downloadRouter)
	auditSetup.StartMetadataRefresh(ctx, logger.Named("audit"))

	auditLogName := "audit-log"
	if rootNode.Name() != "" {
		auditLogName = path.Join(rootNode.Name(), "audit-log")
	}
	rootNode.Announce(auditLogName,
		node.HasTrait(logpb.TraitName, node.WithClients(logpb.WrapApi(auditSrv))),
	)
}

func buildHTTPServer(config sysconf.Config, pi pkiInfo, grpcServer *grpc.Server, mux *http.ServeMux) *http.Server {
	// grpcWebServer wraps the gRPC server so browser clients can call it over HTTP/1.1.
	// CorsForRegisteredEndpointsOnly is false because services are registered dynamically at runtime.
	grpcWebServer := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
	)
	co := cors.New(cors.Options{
		AllowedOrigins:   config.Cors.CorsOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{http2.HeaderAllowOrigin, http2.HeaderAuthorization, http2.HeaderContentType},
		AllowedMethods:   []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		Debug:            config.Cors.DebugMode,
	})
	corsWrap := co.Handler(mux)
	return &http.Server{
		Addr:      config.ListenHTTPS,
		TLSConfig: pi.HTTPServer,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(request) || grpcWebServer.IsAcceptableGrpcCorsRequest(request) {
				grpcWebServer.ServeHTTP(writer, request)
			} else {
				corsWrap.ServeHTTP(writer, request)
			}
		}),
	}
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
	ControllerConfig ConfigStore
	Enrollment       *enrollment.Server
	Cloud            *cloud.Conn

	// services for drivers/automations
	Logger          *zap.Logger
	LogCapture      *logcapture.Core // dynamic tee for log capture (nil if not built via Bootstrap)
	LogLevel        *zap.AtomicLevel // controls the root logger's minimum level
	Node            *node.Node
	Devices         devicespb.DevicesApiClient
	DeviceStore     *devicespb.Collection // for low level control of devices
	Tasks           *task.Group
	Database        *bolthold.Store
	TokenValidators *token.ValidatorSet
	GRPCCerts       *pki.SourceSet
	Stores          *stores.Stores
	Accounts        *account.Store
	CheckRegistry   *healthpb.Registry

	ReflectionServer *reflectionapi.Server

	Mux            *http.ServeMux
	GRPC           *grpc.Server
	HTTP           *http.Server
	DownloadRouter *download.Router

	PrivateKey      pki.PrivateKey
	ClientTLSConfig *tls.Config
	ManagerConn     node.Remote

	deferred []Deferred

	rebootCh chan string // signals a requested clean reboot; set in Run; value is the reason (empty if boot system handled the state file)
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
	defer func() {
		writeControllerRebootState(c.SystemConfig.DataDir, err)
	}()

	// metadata associated with the node itself
	// we don't support changing metadata while running
	c.Node.Announce(c.Node.Name(), node.HasMetadata(initialConfig.Metadata))

	storeUndo := dataretention.Start(ctx, c.Node, c.SystemConfig.Name, c.Stores, c.SystemConfig.Stores, c.CheckRegistry.ForOwner("stores"), c.Logger)
	defer storeUndo()

	group, ctx := errgroup.WithContext(ctx)
	c.rebootCh = make(chan string, 1)
	group.Go(func() error {
		select {
		case reason := <-c.rebootCh:
			return restartNowError{reason: reason}
		case <-ctx.Done():
			return nil
		}
	})
	if c.Enrollment != nil {
		group.Go(func() error {
			return c.Enrollment.AutoRenew(ctx)
		})
	}
	if c.Cloud != nil {
		group.Go(func() error {
			pollInterval := 5 * time.Minute
			if c.SystemConfig.Cloud != nil {
				pollInterval = c.SystemConfig.Cloud.PollInterval.Duration
			}
			needRestart := cloud.AutoPoll(ctx, c.Cloud, pollInterval, c.Logger.Named("cloud.auto-poll"))
			if needRestart {
				c.Logger.Info("triggering automatic restart to apply new deployment")
				return restartNowError{reason: "automatic restart to install deployment"}
			}
			return nil
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
	cloudDirName  = "cloud"
	accountsFile  = "accounts.sqlite3"
)

// a sentinel error type - does not indicate an actual error, but a request to restart the controller immediately
type restartNowError struct {
	reason string
}

func (e restartNowError) Error() string {
	return "automatic restart triggered"
}

func (e restartNowError) ExitCode() int {
	return 0
}

// writeControllerRebootState writes the reboot state file based on how Controller.Run exited.
// Only writes for clean exits (graceful shutdown or deployment restart with a known reason).
// When the exit was triggered via the Boot RPC, the boot system has already written the state
// file (with actor info included), so we skip to avoid overwriting it.
func writeControllerRebootState(dataDir string, err error) {
	// Use errors.As so wrapping (e.g. multierr) doesn't silently fall through to the
	// default case and skip writing the state file.
	var st boot.RebootState
	var rne restartNowError
	switch {
	case err == nil:
		st = boot.RebootState{CleanExit: true}
	case errors.As(err, &rne):
		if rne.reason == "" {
			return // boot system wrote the state file (with actor); don't overwrite
		}
		st = boot.RebootState{Reason: rne.reason, CleanExit: true}
	default:
		return // unexpected error; leave in-progress marker as crash indicator
	}
	_ = boot.WriteStateFile(dataDir, st)
}
