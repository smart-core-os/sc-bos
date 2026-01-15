// Package hub manages the enrollment process for a cohort of nodes.
// This package specifies a hub service, when active this exposes the NodeApi and integrates with the grpc certificate
// stack used for client and server gRPC negotiations.
package hub

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/timshannon/bolthold"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/internal/util/pki/expire"
	"github.com/smart-core-os/sc-bos/pkg/app/stores"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/bolthub"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/config"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/pgxhub"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/util/netutil"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func Factory() system.Factory {
	return &factory{}
}

type factory struct{}

func (f *factory) New(services system.Services) service.Lifecycle {
	s := &System{
		hubNode:         services.CohortManager,
		name:            services.Node.Name(),
		node:            services.Node,
		endpoint:        services.GRPCEndpoint,
		dataDir:         services.DataDir,
		sharedKey:       services.PrivateKey,
		clientTLSConfig: services.ClientTLSConfig,
		certs:           services.GRPCCerts,
		logger:          services.Logger.Named("hub"),
		boltDb:          services.Database,
		stores:          services.Stores,
	}
	s.Service = service.New(
		service.MonoApply(s.applyConfig),
		service.WithRetry[config.Root](service.RetryWithLogger(func(logContext service.RetryContext) {
			logContext.LogTo("applyConfig", s.logger)
		})),
	)
	return s
}

type System struct {
	*service.Service[config.Root]

	hubNode node.Remote

	name            string
	node            *node.Node
	endpoint        string
	dataDir         string
	sharedKey       pki.PrivateKey
	clientTLSConfig *tls.Config
	logger          *zap.Logger
	boltDb          *bolthold.Store
	stores          *stores.Stores

	certs   *pki.SourceSet
	sources []pki.Source
	undos   []node.Undo
}

func (s *System) applyConfig(ctx context.Context, cfg config.Root) error {
	s.undoAll()
	s.deleteSources()

	// todo: move this validation to the config parse function
	if cfg.Storage == nil {
		return errors.New("no storage")
	}
	var hubConn grpc.ClientConnInterface
	switch cfg.Storage.Type {
	case config.StorageTypeProxy:
		s.logger.Warn("proxy storage type is deprecated - use gateway to route requests to the hub instead")
		conn, err := s.hubNode.Connect(ctx)
		if err != nil {
			return err
		}
		hubConn = conn
	case config.StorageTypePostgres:
		var pool *pgxpool.Pool
		var err error
		if cfg.Storage.ConnectConfig.IsZero() {
			_, _, pool, err = s.stores.Postgres()
		} else {
			pool, err = pgxutil.Connect(ctx, cfg.Storage.ConnectConfig)
			if err == nil {
				go func() {
					<-ctx.Done()
					pool.Close()
				}()
			}
		}
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		server, err := pgxhub.NewServerFromPool(ctx, pool, pgxhub.WithLogger(s.logger))
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		// caSource sources the certs used to sign enrollment requests
		caSource := s.newCA(cfg)
		// grpcSource generates certs signed by caSource and is trusted by the cohort this controller manages
		grpcSource := s.newGRPC(cfg, caSource)

		server.Authority = caSource
		server.TestTLSConfig = s.clientTLSConfig
		server.ManagerName = cfg.Name
		if server.ManagerName == "" {
			server.ManagerName = s.name
		}
		server.ManagerAddr = cfg.Address
		if server.ManagerAddr == "" {
			server.ManagerAddr = s.endpoint
		}
		if server.ManagerAddr == "" {
			ipAddr, err := netutil.OutboundAddr()
			if err != nil {
				return err
			}
			server.ManagerAddr = ipAddr.String() + ":23557" // guess at the default port
		}

		s.sources = append(s.sources, grpcSource)
		s.certs.Append(grpcSource)
		hubConn = wrap.ServerToClient(hubpb.HubApi_ServiceDesc, server)
	case config.StorageTypeBolt:
		server := bolthub.NewServerFromBolthold(s.boltDb, s.logger)

		// caSource sources the certs used to sign enrollment requests
		caSource := s.newCA(cfg)
		// grpcSource generates certs signed by caSource and is trusted by the cohort this controller manages
		grpcSource := s.newGRPC(cfg, caSource)

		server.Authority = caSource
		server.TestTLSConfig = s.clientTLSConfig
		server.ManagerName = cfg.Name
		if server.ManagerName == "" {
			server.ManagerName = s.name
		}
		server.ManagerAddr = cfg.Address
		if server.ManagerAddr == "" {
			server.ManagerAddr = s.endpoint
		}
		if server.ManagerAddr == "" {
			ipAddr, err := netutil.OutboundAddr()
			if err != nil {
				return err
			}
			server.ManagerAddr = ipAddr.String() + ":23557" // guess at the default port
		}

		s.sources = append(s.sources, grpcSource)
		s.certs.Append(grpcSource)
		hubConn = wrap.ServerToClient(hubpb.HubApi_ServiceDesc, server)
	default:
		return fmt.Errorf("unsuported storage type %s", cfg.Storage.Type)
	}

	srv, err := node.RegistryConnService(hubpb.HubApi_ServiceDesc, hubConn)
	if err != nil {
		return err
	}
	undo, err := s.node.AnnounceService(srv)
	s.undos = append(s.undos, undo)

	return err
}

func (s *System) deleteSources() {
	for _, source := range s.sources {
		s.certs.Delete(source)
	}
	s.sources = nil
}

func (s *System) undoAll() {
	for _, u := range s.undos {
		u()
	}
	s.undos = nil
}

func (s *System) newCA(cfg config.Root) pki.Source {
	// be flexible when loading cert and key from disk,
	// allow reusing the controllers private key as a pair with the dedicate CA cert.
	fileCA := pki.CacheSource(pki.FSSource(
		file(s.dataDir, cfg.Cert, "hub-ca.cert.pem"),
		file(s.dataDir, cfg.Cert, "hub-ca.key.pem"),
		file(s.dataDir, cfg.Cert, "hub.roots.pem"),
	), expire.After(15*time.Minute))

	// we only support self signed CA certs if the user didn't configure
	// paths to cert files
	if cfg.Cert != "" {
		return fileCA
	}

	selfSignedCA := pki.LazySource(func() (pki.Source, error) {
		name := s.name
		if name == "" {
			name = "hub-ca"
		}
		key, _, err := pki.LoadOrGeneratePrivateKey(filepath.Join(s.dataDir, "hub-self-signed-ca.key.pem"), s.logger)
		if err != nil {
			return nil, err
		}
		return pki.CacheSource(
			pki.SelfSignedSourceT(key,
				&x509.Certificate{
					Subject:               pkix.Name{CommonName: name},
					KeyUsage:              x509.KeyUsageCertSign,
					IsCA:                  true,
					BasicConstraintsValid: true,
				},
				pki.WithExpireAfter(10*365*24*time.Hour),
			),
			expire.AfterProgress(0.5),
			pki.WithFSCache(
				filepath.Join(s.dataDir, "hub-self-signed-ca.cert.pem"),
				filepath.Join(s.dataDir, "hub-self-signed-ca.roots.pem"),
				key),
		), nil
	})

	return &pki.SourceSet{
		fileCA,
		selfSignedCA,
	}
}

func (s *System) newGRPC(cfg config.Root, ca pki.Source) pki.Source {
	validity := cfg.Validity.Or(60 * 24 * time.Hour)
	name := cfg.Name
	if name == "" {
		name = s.name
	}
	if name == "" {
		name = "controller"
	}
	template := &x509.Certificate{
		Subject:     pkix.Name{CommonName: name},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	return pki.CacheSource(
		pki.AuthoritySource(ca, template, s.sharedKey, pki.WithIfaces(), pki.WithExpireAfter(validity)),
		expire.AfterProgress(0.5),
	)
}

func file(dir string, opts ...string) string {
	for _, opt := range opts {
		if opt != "" {
			return filepath.Join(dir, opt)
		}
	}
	return ""
}
