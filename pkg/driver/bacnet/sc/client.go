// Package sc implements a BACnet/SC (secure connect) client for the bacnet
// driver. It speaks the BACnet/SC data link - a websocket-over-TLS connection to
// a hub - while reusing gobacnet's NPDU/APDU encoding and transaction managers
// (the tsm and utsm packages) for everything above the data link.
//
// Only the data link differs from BACnet/IP, so the request orchestration in
// this package (see request.go) mirrors gobacnet's own Client methods, swapping
// gobacnet's UDP send for the websocket datalink. gobacnet is licensed GPLv2
// with a linking exception; the small amount of orchestration ported here is
// derivative of that library.
package sc

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smart-core-os/gobacnet/encoding"
	"github.com/smart-core-os/gobacnet/enum/pdutype"
	"github.com/smart-core-os/gobacnet/tsm"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/utsm"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/bclient"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

const maxInt = int(^uint(0) >> 1)

// defaults applied when the corresponding config field is zero.
const (
	defaultMaxBVLC        = 1497
	defaultMaxNPDU        = 1497
	defaultHeartbeat      = 30 * time.Second
	defaultConnectTimeout = 10 * time.Second
	defaultMaxConcurrent  = 20
)

// Client is a BACnet/SC client. It satisfies bclient.Client.
type Client struct {
	link *datalink
	tsm  *tsm.TSM
	utsm *utsm.Manager
	log  *zap.Logger
}

var _ bclient.Client = (*Client)(nil)

// NewClient builds a BACnet/SC client from config and connects to the hub. It
// returns an error if the configuration is invalid or no hub can be reached.
func NewClient(cfg config.SecureConnect, maxConcurrent uint8, logger *zap.Logger) (*Client, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	tlsConfig, err := cfg.TLS.Read("", nil)
	if err != nil {
		return nil, fmt.Errorf("read tls config: %w", err)
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}
	// BACnet/SC mandates TLS 1.2 or later (Clause AB.7.2).
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		tlsConfig = tlsConfig.Clone()
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	vmac, err := resolveVMAC(cfg.VMAC)
	if err != nil {
		return nil, err
	}
	devUUID, err := resolveUUID(cfg.DeviceUUID)
	if err != nil {
		return nil, err
	}

	uris := []string{cfg.PrimaryHubURI}
	if cfg.FailoverHubURI != "" {
		uris = append(uris, cfg.FailoverHubURI)
	}

	if maxConcurrent == 0 {
		maxConcurrent = defaultMaxConcurrent
	}

	c := &Client{
		tsm: tsm.New(maxConcurrent),
		utsm: utsm.NewManager(
			utsm.DefaultSubscriberTimeout(10*time.Second),
			utsm.DefaultSubscriberLastReceivedTimeout(2*time.Second),
		),
		log: logger,
	}
	c.link = &datalink{
		uris:           uris,
		tlsConfig:      tlsConfig,
		our:            vmac,
		uuid:           devUUID,
		maxBVLC:        orUint16(cfg.MaxBVLCLength, defaultMaxBVLC),
		maxNPDU:        orUint16(cfg.MaxNPDULength, defaultMaxNPDU),
		heartbeat:      orDuration(cfg.HeartbeatInterval.Duration, defaultHeartbeat),
		connectTimeout: orDuration(cfg.ConnectTimeout.Duration, defaultConnectTimeout),
		log:            logger,
		onNPDU:         c.handleNPDU,
	}
	logger.Info("bacnet/sc client starting",
		zap.Strings("hubs", uris),
		zap.Stringer("vmac", vmac),
		zap.String("deviceUUID", uuid.UUID(devUUID).String()))
	if err := c.link.start(); err != nil {
		return nil, fmt.Errorf("connect to bacnet/sc hub: %w", err)
	}
	return c, nil
}

// Close disconnects from the hub and releases resources.
func (c *Client) Close() {
	c.link.close()
}

// handleNPDU routes a received NPDU to the waiting transaction (confirmed
// responses) or the unconfirmed manager (I-Am). It mirrors gobacnet's
// Client.handleMsg, minus the BVLC/UDP handling which the datalink performs.
func (c *Client) handleNPDU(orig VMAC, b []byte) {
	dec := encoding.NewDecoder(b)
	var npdu bactypes.NPDU
	if err := dec.NPDU(&npdu); err != nil {
		c.log.Debug("bacnet/sc bad npdu", zap.Error(err))
		return
	}
	if npdu.IsNetworkLayerMessage {
		return
	}
	// Snapshot the APDU bytes before decoding; confirmed responses are matched by
	// invoke id and re-decoded by the waiting request.
	apduBytes := dec.Bytes()
	var apdu bactypes.APDU
	if err := dec.APDU(&apdu); err != nil {
		c.log.Debug("bacnet/sc bad apdu", zap.Error(err))
		return
	}

	switch apdu.DataType {
	case pdutype.UnconfirmedServiceRequest:
		if apdu.UnconfirmedService != bactypes.ServiceUnconfirmedIAm {
			return
		}
		idec := encoding.NewDecoder(apdu.RawData)
		var iam bactypes.IAm
		if err := idec.IAm(&iam); err != nil {
			c.log.Debug("bacnet/sc bad iam", zap.Error(err))
			return
		}
		// Address the device by its originating VMAC; carry any NPDU source so
		// devices reached via a BACnet router remain addressable.
		var net uint16
		var adr []byte
		if npdu.Source != nil {
			net = npdu.Source.Net
			adr = npdu.Source.Adr
		}
		iam.Addr = vmacToAddress(orig, net, adr)
		c.utsm.Publish(int(iam.ID.Instance), iam)
	case pdutype.SimpleAck, pdutype.ComplexAck, pdutype.ConfirmedServiceRequest:
		_ = c.tsm.Send(apdu.InvokeId, apduBytes)
	case pdutype.Error:
		_ = c.tsm.Send(apdu.InvokeId, apdu.Error)
	}
}

// requestNPDU builds the common NPDU header for a confirmed request to dest.
func requestNPDU(dest bactypes.Address) bactypes.NPDU {
	return bactypes.NPDU{
		Version:               bactypes.ProtocolVersion,
		Destination:           &dest,
		Source:                nil, // local; source VMAC is carried in the BVLC header
		IsNetworkLayerMessage: false,
		ExpectingReply:        true,
		Priority:              bactypes.Normal,
		HopCount:              bactypes.DefaultHopCount,
	}
}

func resolveVMAC(s string) (VMAC, error) {
	if s == "" {
		return RandomVMAC()
	}
	return ParseVMAC(s)
}

func resolveUUID(s string) ([16]byte, error) {
	if s == "" {
		return uuid.New(), nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return [16]byte{}, fmt.Errorf("deviceUUID %q: %w", s, err)
	}
	return u, nil
}

func orUint16(v, def uint16) uint16 {
	if v == 0 {
		return def
	}
	return v
}

func orDuration(v, def time.Duration) time.Duration {
	if v == 0 {
		return def
	}
	return v
}
