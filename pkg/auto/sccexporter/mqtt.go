package sccexporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/sccexporter/config"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

type TraitData map[string]json.RawMessage

type Device struct {
	Name string `json:"name"`
	// Data is a map where the key is the trait name and the value is a nested object
	// containing resource names mapped to their data/info
	Data map[trait.Name]TraitData `json:"data,omitempty"`
}

type message struct {
	Agent     string    `json:"agent"`
	Device    Device    `json:"device"`
	Timestamp time.Time `json:"timestamp"`
}

type sccConnector struct {
	logger         *zap.Logger
	messagesCh     chan message
	mqttCfg        config.Mqtt
	mqttClient     mqtt.Client
	reconnecting   bool
	lastReconnect  time.Time
	reconnectDelay time.Duration
}

func newSccConnector(logger *zap.Logger, mqttCfg config.Mqtt, client mqtt.Client) *sccConnector {
	s := &sccConnector{
		logger:         logger,
		messagesCh:     make(chan message, 100),
		mqttCfg:        mqttCfg,
		mqttClient:     client,
		reconnecting:   false,
		reconnectDelay: time.Second,
	}
	return s
}

// reconnect attempts to reconnect to the MQTT broker with exponential backoff.
// Returns true if reconnection succeeded, false otherwise.
// do not call this more than once per sccConnector instance
func (s *sccConnector) reconnect(ctx context.Context) bool {
	if s.reconnecting {
		return false
	}

	now := time.Now()
	if now.Sub(s.lastReconnect) < s.reconnectDelay {
		s.logger.Debug("skipping reconnect attempt, too soon since last attempt",
			zap.Duration("delay", s.reconnectDelay),
			zap.Duration("elapsed", now.Sub(s.lastReconnect)))
		return false
	}

	s.reconnecting = true
	s.lastReconnect = now
	defer func() {
		s.reconnecting = false
	}()

	s.logger.Info("attempting to reconnect to MQTT broker",
		zap.Duration("backoff", s.reconnectDelay))

	s.mqttClient.Disconnect(500)

	select {
	case <-ctx.Done():
		return false
	case <-time.After(100 * time.Millisecond):
	}

	token := s.mqttClient.Connect()
	if !token.WaitTimeout(s.mqttCfg.ConnectTimeout.Duration) {
		s.logger.Error("timeout reconnecting to mqtt broker",
			zap.Duration("backoff", s.reconnectDelay))
		s.reconnectDelay = min(s.reconnectDelay*2, 60*time.Second)
		return false
	}

	if err := token.Error(); err != nil {
		s.logger.Error("failed to reconnect to mqtt broker",
			zap.Error(err),
			zap.Duration("backoff", s.reconnectDelay))
		s.reconnectDelay = min(s.reconnectDelay*2, 60*time.Second)
		return false
	}

	s.logger.Info("successfully reconnected to MQTT broker")
	s.reconnectDelay = time.Second
	return true
}

// publishToScc listens on the messages channel and publishes messages to the SCC MQTT broker
// do not call this more than once per sccConnector instance
func (s *sccConnector) publishToScc(ctx context.Context) error {

	token := s.mqttClient.Connect()
	if !token.WaitTimeout(s.mqttCfg.ConnectTimeout.Duration) {
		s.logger.Warn("failed to connect to MQTT client on startup, will retry on next message")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case m, ok := <-s.messagesCh:
			if !ok {
				return nil
			}

			if !s.mqttClient.IsConnected() {
				if !s.reconnect(ctx) {
					s.logger.Warn("skipping message, mqtt client not connected",
						zap.String("device", m.Device.Name))
					continue
				}
			}

			bytes, err := json.Marshal(m)
			if err != nil {
				s.logger.Error("failed to marshal scc payload", zap.Error(err))
				continue
			}

			t := s.mqttClient.Publish(s.mqttCfg.Topic, byte(*s.mqttCfg.Qos), false, bytes)
			if !t.WaitTimeout(s.mqttCfg.PublishTimeout.Duration) {
				s.logger.Warn("timeout publishing message for device",
					zap.String("device", m.Device.Name),
					zap.Duration("timeout", s.mqttCfg.PublishTimeout.Duration))
				continue
			}
			s.reconnectDelay = time.Second
		}
	}
}

// newMqttClient creates a new MQTT client with TLS configuration
// assumes the config contains valid paths with valid certs.
func newMqttClient(cfg config.Mqtt) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Host)
	opts.SetClientID(cfg.ClientId)
	opts.SetOrderMatters(false)

	cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(cfg.CaCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	opts.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	return mqtt.NewClient(opts), nil
}
