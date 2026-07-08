package sccexporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/sccexporter/config"
)

// nodeIDProperty is the MQTT v5 user property carrying the publisher node id.
// It is only meaningful on a plain/local MQTT v5 broker (dev), where Connect's
// adapter reads the originator identity from it. On the Event Grid path identity
// comes from the authenticated credential (enrichment), so it is ignored there.
const nodeIDProperty = "nodeId"

// publisher publishes UDMI messages to the Connect telemetry broker over MQTT v5.
type publisher struct {
	cm             *autopaho.ConnectionManager
	qos            byte
	nodeID         string // set as the v5 user property when non-empty
	publishTimeout time.Duration
}

// newPublisher builds an MQTT v5 connection manager for the Connect telemetry
// broker. The connection is established (and re-established) in the background for
// the lifetime of ctx; publishing gates on connectivity per call.
func newPublisher(ctx context.Context, cfg config.Mqtt, cred auto.CloudCredentialSource, logger *zap.Logger) (*publisher, error) {
	tlsCfg, err := buildTLSConfig(cfg, cred)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid mqtt host %q: %w", cfg.Host, err)
	}

	var nodeID string
	if cred != nil {
		nodeID = cred.NodeID()
	}

	clientID := cfg.ClientId
	if clientID == "" {
		clientID = nodeID
	}

	clientCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		TlsCfg:                        tlsCfg,
		KeepAlive:                     20,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ConnectTimeout:                cfg.ConnectTimeout.Duration,
		OnConnectionUp: func(*autopaho.ConnectionManager, *paho.Connack) {
			logger.Info("connected to Connect telemetry broker", zap.String("host", cfg.Host))
		},
		OnConnectError: func(err error) {
			logger.Warn("Connect telemetry broker connection error", zap.Error(err))
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
		},
	}

	cm, err := autopaho.NewConnection(ctx, clientCfg)
	if err != nil {
		return nil, err
	}

	return &publisher{
		cm:             cm,
		qos:            byte(*cfg.Qos),
		nodeID:         nodeID,
		publishTimeout: cfg.PublishTimeout.Duration,
	}, nil
}

// publish sends payload to topic at the configured QoS, waiting up to
// publishTimeout for the connection to be up. The nodeId user property is set
// when known (local MQTT path).
func (p *publisher) publish(ctx context.Context, topic string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	if err := p.cm.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("await connection: %w", err)
	}

	_, err := p.cm.Publish(ctx, p.newPublishPacket(topic, payload))
	return err
}

// newPublishPacket builds the MQTT v5 publish packet, stamping the nodeId user
// property when the node identity is known (local MQTT path).
func (p *publisher) newPublishPacket(topic string, payload []byte) *paho.Publish {
	pub := &paho.Publish{
		QoS:     p.qos,
		Topic:   topic,
		Payload: payload,
	}
	if p.nodeID != "" {
		pub.Properties = &paho.PublishProperties{}
		pub.Properties.User.Add(nodeIDProperty, p.nodeID)
	}
	return pub
}

// close disconnects the connection manager, waiting up to ctx for a clean
// DISCONNECT.
func (p *publisher) close(ctx context.Context) {
	_ = p.cm.Disconnect(ctx)
}

// buildTLSConfig assembles the client TLS config for the broker.
//
// In cloud-credential mode the client presents the Connect leaf certificate via
// GetClientCertificate (renewals picked up live) and verifies the broker against
// the system/public roots — Event Grid presents a normal public Azure TLS server
// cert, so the Connect root is not used for server verification. In the file-path
// dev/test mode the client cert is loaded from PEM and an optional CA verifies the
// broker (empty ⇒ system roots).
func buildTLSConfig(cfg config.Mqtt, cred auto.CloudCredentialSource) (*tls.Config, error) {
	if cfg.UseCloudCredential {
		if cred == nil {
			return nil, fmt.Errorf("mqtt.useCloudCredential is set but no cloud credential is available (pending PR #890)")
		}
		return &tls.Config{
			GetClientCertificate: cred.GetClientCertificate,
			MinVersion:           tls.VersionTLS12,
		}, nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if cfg.CaCertPath != "" {
		caCert, err := os.ReadFile(cfg.CaCertPath)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}
