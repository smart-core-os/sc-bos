package sccexporter

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/sccexporter/config"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func TestNewPublishPacketSetsNodeID(t *testing.T) {
	withNode := &publisher{qos: 1, nodeID: "node-123"}
	pkt := withNode.newPublishPacket("tlm/devices/x/events/pointset", []byte(`{}`))
	require.NotNil(t, pkt.Properties)
	assert.Equal(t, "node-123", pkt.Properties.User.Get(nodeIDProperty))
	assert.Equal(t, byte(1), pkt.QoS)
	assert.Equal(t, "tlm/devices/x/events/pointset", pkt.Topic)

	withoutNode := &publisher{qos: 0}
	pkt2 := withoutNode.newPublishPacket("t", []byte(`{}`))
	assert.Nil(t, pkt2.Properties, "no node id ⇒ no user properties")
}

// TestPublisherPublishesOverTLSv5 drives the real transport: it stands up an
// embedded MQTT v5 broker with mutual TLS, connects the publisher with file-path
// certs, publishes UDMI telemetry + discovery, and asserts both land on the
// expected topics with payloads that round-trip through the UDMI types.
func TestPublisherPublishesOverTLSv5(t *testing.T) {
	dir, brokerTLS := writeTestCerts(t)
	addr, msgs := startTLSBroker(t, brokerTLS)

	qos := 1
	cfg := config.Mqtt{
		Host:           "tls://" + addr,
		TopicPrefix:    "tlm",
		ClientId:       "test-exporter",
		ClientCertPath: filepath.Join(dir, "client.crt"),
		ClientKeyPath:  filepath.Join(dir, "client.key"),
		CaCertPath:     filepath.Join(dir, "ca.crt"),
		Qos:            &qos,
		ConnectTimeout: &jsontypes.Duration{Duration: 5 * time.Second},
		PublishTimeout: &jsontypes.Duration{Duration: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pub, err := newPublisher(ctx, cfg, nil, zap.NewNop())
	require.NoError(t, err)
	defer pub.close(context.Background())

	const ref = "van/uk/brum/ugs/meters/elec-main"
	pointsetTop := pointsetTopic(cfg.TopicPrefix, ref)
	discoveryTop := discoveryTopic(cfg.TopicPrefix, ref)

	tele := udmi.PointsetEvent{
		Timestamp: time.Now().UTC(),
		Version:   udmi.PointsetVersion,
		Points:    udmi.PointsEvent{meterPointUsage: {PresentValue: 123.45}},
	}
	teleBytes, err := json.Marshal(tele)
	require.NoError(t, err)
	require.NoError(t, pub.publish(ctx, pointsetTop, teleBytes))

	disc := udmi.MetadataEvent{
		Timestamp: time.Now().UTC(),
		Version:   udmi.PointsetVersion,
		System:    udmi.MetadataSystem{Name: "Main Meter"},
		Pointset:  &udmi.MetadataPointset{Points: map[string]udmi.MetadataPoint{meterPointUsage: {Units: "kWh"}}},
	}
	discBytes, err := json.Marshal(disc)
	require.NoError(t, err)
	require.NoError(t, pub.publish(ctx, discoveryTop, discBytes))

	got := map[string]capturedMsg{}
	for len(got) < 2 {
		select {
		case m := <-msgs:
			got[m.topic] = m
		case <-time.After(10 * time.Second):
			t.Fatalf("timed out waiting for messages, received %d/2", len(got))
		}
	}

	teleMsg, ok := got[pointsetTop]
	require.True(t, ok, "telemetry not received on %s", pointsetTop)
	var teleRT udmi.PointsetEvent
	require.NoError(t, json.Unmarshal(teleMsg.payload, &teleRT))
	require.Contains(t, teleRT.Points, meterPointUsage)

	discMsg, ok := got[discoveryTop]
	require.True(t, ok, "discovery not received on %s", discoveryTop)
	var discRT udmi.MetadataEvent
	require.NoError(t, json.Unmarshal(discMsg.payload, &discRT))
	require.NotNil(t, discRT.Pointset)
	assert.Equal(t, "kWh", discRT.Pointset.Points[meterPointUsage].Units)
}

type capturedMsg struct {
	topic   string
	payload []byte
	user    map[string]string
}

// startTLSBroker starts an embedded MQTT v5 broker with the given (mutual) TLS
// config and an inline subscription that captures everything under tlm/#.
func startTLSBroker(t *testing.T, brokerTLS *tls.Config) (addr string, msgs <-chan capturedMsg) {
	t.Helper()
	server := mochi.New(&mochi.Options{InlineClient: true})
	require.NoError(t, server.AddHook(new(auth.AllowHook), nil))

	l := listeners.NewTCP(listeners.Config{ID: "t1", Address: "127.0.0.1:0", TLSConfig: brokerTLS})
	require.NoError(t, server.AddListener(l))
	require.NoError(t, server.Serve())
	t.Cleanup(func() { _ = server.Close() })

	ch := make(chan capturedMsg, 16)
	require.NoError(t, server.Subscribe("tlm/#", 1, func(_ *mochi.Client, _ packets.Subscription, pk packets.Packet) {
		user := make(map[string]string, len(pk.Properties.User))
		for _, u := range pk.Properties.User {
			user[u.Key] = u.Val
		}
		ch <- capturedMsg{topic: pk.TopicName, payload: append([]byte(nil), pk.Payload...), user: user}
	}))

	return l.Address(), ch
}

// writeTestCerts generates a CA plus a server and client leaf (both signed by the
// CA), writes the client cert/key and CA to a temp dir for the file-path config,
// and returns a broker TLS config that presents the server cert and requires a
// CA-signed client cert.
func writeTestCerts(t *testing.T) (dir string, brokerTLS *tls.Config) {
	t.Helper()
	dir = t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	serverPEM, serverKeyPEM := signLeaf(t, caCert, caKey, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "broker"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	})

	clientPEM, clientKeyPEM := signLeaf(t, caCert, caKey, &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "client-node"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})

	writeFile(t, dir, "ca.crt", caPEM)
	writeFile(t, dir, "client.crt", clientPEM)
	writeFile(t, dir, "client.key", clientKeyPEM)

	serverKeyPair, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))
	brokerTLS = &tls.Config{
		Certificates: []tls.Certificate{serverKeyPair},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	}
	return dir, brokerTLS
}

func signLeaf(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, tmpl *x509.Certificate) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
}
