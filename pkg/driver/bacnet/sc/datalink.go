package sc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	bactypes "github.com/smart-core-os/gobacnet/types"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// subprotocolHub is the websocket subprotocol for a node-to-hub BACnet/SC
// connection (ASHRAE 135-2020 Clause AB.7.1).
const subprotocolHub = "hub.bsc.bacnet.org"

// readLimit bounds a single incoming websocket message. BACnet/SC max BVLC
// lengths are well under this; the generous ceiling tolerates large segmented
// reads without risking unbounded buffering.
const readLimit = 1 << 20

// ErrNotConnected is returned by Send when there is no live hub connection.
var ErrNotConnected = errors.New("bacnet/sc: not connected to a hub")

// datalink owns the websocket connection(s) to the BACnet/SC hub, the connect
// handshake, keepalive heartbeats and failover. It exposes Send for outgoing
// NPDUs and delivers incoming NPDUs via the onNPDU callback.
type datalink struct {
	uris           []string
	tlsConfig      *tls.Config
	our            VMAC
	uuid           [16]byte
	maxBVLC        uint16
	maxNPDU        uint16
	heartbeat      time.Duration
	connectTimeout time.Duration
	log            *zap.Logger

	// onNPDU is invoked for every received Encapsulated-NPDU with the
	// originating VMAC (zero if absent) and the raw NPDU payload.
	onNPDU func(orig VMAC, npdu []byte)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	conn    *websocket.Conn
	msgID   uint16
	hubVMAC VMAC
}

// start establishes the initial hub connection and launches the background
// manager that keeps it alive and fails over between hubs. It returns an error
// if no hub can be reached.
func (d *datalink) start() error {
	d.ctx, d.cancel = context.WithCancel(context.Background())
	conn, hub, uri, err := d.dialAny(d.ctx)
	if err != nil {
		d.cancel()
		return err
	}
	d.log.Info("bacnet/sc connected to hub", zap.String("uri", uri), zap.Stringer("hubVMAC", hub))
	d.setConn(conn, hub)
	d.wg.Add(1)
	go d.manage(conn)
	return nil
}

func (d *datalink) close() {
	if d.cancel == nil {
		return
	}
	d.cancel()
	d.mu.Lock()
	conn := d.conn
	d.conn = nil
	d.mu.Unlock()
	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "shutting down")
	}
	d.wg.Wait()
}

func (d *datalink) setConn(conn *websocket.Conn, hub VMAC) {
	d.mu.Lock()
	d.conn = conn
	d.hubVMAC = hub
	d.mu.Unlock()
}

func (d *datalink) nextMsgID() uint16 {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.msgID++
	return d.msgID
}

// Send wraps npdu in an Encapsulated-NPDU BVLC message and writes it to the hub.
func (d *datalink) Send(dest bactypes.Address, npdu []byte) error {
	d.mu.Lock()
	conn := d.conn
	d.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	msg := bvlcMessage{
		Function:  bvlcEncapsulatedNPDU,
		MessageID: d.nextMsgID(),
		HasOrig:   true,
		Orig:      d.our,
		HasDest:   true,
		Dest:      addressToVMAC(dest),
		Payload:   npdu,
	}
	ctx, cancel := context.WithTimeout(d.ctx, d.connectTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageBinary, msg.encode())
}

// manage serves the current connection, then reconnects (with failover and
// backoff) until the datalink is closed.
func (d *datalink) manage(conn *websocket.Conn) {
	defer d.wg.Done()
	for {
		d.serve(conn)
		if d.ctx.Err() != nil {
			return
		}
		d.log.Warn("bacnet/sc hub connection lost, reconnecting")
		d.setConn(nil, VMAC{})

		next, hub, uri := d.reconnect()
		if next == nil {
			return // ctx cancelled
		}
		d.log.Info("bacnet/sc reconnected to hub", zap.String("uri", uri), zap.Stringer("hubVMAC", hub))
		d.setConn(next, hub)
		conn = next
	}
}

// serve runs the read loop and heartbeat for conn, returning when the
// connection fails or the datalink is closed.
func (d *datalink) serve(conn *websocket.Conn) {
	connCtx, connCancel := context.WithCancel(d.ctx)
	defer connCancel()

	go func() {
		t := time.NewTicker(d.heartbeat)
		defer t.Stop()
		for {
			select {
			case <-connCtx.Done():
				return
			case <-t.C:
				if err := d.sendHeartbeat(connCtx, conn); err != nil {
					connCancel()
					return
				}
			}
		}
	}()

	for {
		typ, data, err := conn.Read(connCtx)
		if err != nil {
			return
		}
		if typ != websocket.MessageBinary {
			continue
		}
		d.handleFrame(connCtx, conn, data)
	}
}

func (d *datalink) handleFrame(ctx context.Context, conn *websocket.Conn, data []byte) {
	msg, err := decodeBVLC(data)
	if err != nil {
		d.log.Debug("bacnet/sc dropping malformed bvlc", zap.Error(err))
		return
	}
	switch msg.Function {
	case bvlcEncapsulatedNPDU:
		if d.onNPDU != nil {
			var orig VMAC
			if msg.HasOrig {
				orig = msg.Orig
			}
			d.onNPDU(orig, msg.Payload)
		}
	case bvlcHeartbeatRequest:
		ack := bvlcMessage{Function: bvlcHeartbeatACK, MessageID: msg.MessageID}
		_ = conn.Write(ctx, websocket.MessageBinary, ack.encode())
	case bvlcHeartbeatACK:
		// expected response to our own heartbeat; nothing to do.
	case bvlcDisconnectRequest:
		ack := bvlcMessage{Function: bvlcDisconnectACK, MessageID: msg.MessageID}
		_ = conn.Write(ctx, websocket.MessageBinary, ack.encode())
		_ = conn.Close(websocket.StatusNormalClosure, "hub requested disconnect")
	case bvlcResult:
		if r, err := decodeResult(msg.Payload); err == nil && r.Result != 0 {
			d.log.Warn("bacnet/sc received NAK",
				zap.Stringer("for", r.Function),
				zap.Uint16("errorClass", r.ErrorClass),
				zap.Uint16("errorCode", r.ErrorCode))
		}
	default:
		d.log.Debug("bacnet/sc ignoring bvlc", zap.Stringer("function", msg.Function))
	}
}

func (d *datalink) sendHeartbeat(ctx context.Context, conn *websocket.Conn) error {
	msg := bvlcMessage{Function: bvlcHeartbeatRequest, MessageID: d.nextMsgID()}
	ctx, cancel := context.WithTimeout(ctx, d.connectTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageBinary, msg.encode())
}

// reconnect loops over the configured hubs with capped exponential backoff until
// one accepts the connection or the datalink is closed (returns nil conn).
func (d *datalink) reconnect() (*websocket.Conn, VMAC, string) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second
	for {
		conn, hub, uri, err := d.dialAny(d.ctx)
		if err == nil {
			return conn, hub, uri
		}
		if d.ctx.Err() != nil {
			return nil, VMAC{}, ""
		}
		d.log.Debug("bacnet/sc reconnect attempt failed", zap.Error(err), zap.Duration("retryIn", backoff))
		select {
		case <-d.ctx.Done():
			return nil, VMAC{}, ""
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// dialAny tries each configured hub URI in order, returning the first that
// completes the connect handshake.
func (d *datalink) dialAny(ctx context.Context) (*websocket.Conn, VMAC, string, error) {
	var errs error
	for _, uri := range d.uris {
		conn, hub, err := d.connectOnce(ctx, uri)
		if err == nil {
			return conn, hub, uri, nil
		}
		errs = errors.Join(errs, fmt.Errorf("%s: %w", uri, err))
	}
	if errs == nil {
		errs = errors.New("no hub URIs configured")
	}
	return nil, VMAC{}, "", errs
}

// connectOnce dials uri and performs the BACnet/SC connect handshake
// (Connect-Request / Connect-Accept), returning the hub's VMAC on success.
func (d *datalink) connectOnce(ctx context.Context, uri string) (*websocket.Conn, VMAC, error) {
	dialCtx, cancel := context.WithTimeout(ctx, d.connectTimeout)
	defer cancel()

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: d.tlsConfig}}
	conn, _, err := websocket.Dial(dialCtx, uri, &websocket.DialOptions{
		HTTPClient:   httpClient,
		Subprotocols: []string{subprotocolHub},
	})
	if err != nil {
		return nil, VMAC{}, fmt.Errorf("websocket dial: %w", err)
	}
	conn.SetReadLimit(readLimit)

	req := bvlcMessage{
		Function:  bvlcConnectRequest,
		MessageID: d.nextMsgID(),
		Payload: connectInfo{
			VMAC:          d.our,
			DeviceUUID:    d.uuid,
			MaxBVLCLength: d.maxBVLC,
			MaxNPDULength: d.maxNPDU,
		}.encode(),
	}
	if err := conn.Write(dialCtx, websocket.MessageBinary, req.encode()); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "connect-request write failed")
		return nil, VMAC{}, fmt.Errorf("connect-request: %w", err)
	}

	for {
		typ, data, err := conn.Read(dialCtx)
		if err != nil {
			_ = conn.Close(websocket.StatusInternalError, "awaiting connect-accept")
			return nil, VMAC{}, fmt.Errorf("awaiting connect-accept: %w", err)
		}
		if typ != websocket.MessageBinary {
			continue
		}
		msg, err := decodeBVLC(data)
		if err != nil {
			continue
		}
		switch msg.Function {
		case bvlcConnectAccept:
			info, err := decodeConnectInfo(msg.Payload)
			if err != nil {
				_ = conn.Close(websocket.StatusProtocolError, "bad connect-accept")
				return nil, VMAC{}, fmt.Errorf("connect-accept: %w", err)
			}
			return conn, info.VMAC, nil
		case bvlcResult:
			_ = conn.Close(websocket.StatusProtocolError, "connect rejected")
			if r, derr := decodeResult(msg.Payload); derr == nil {
				return nil, VMAC{}, fmt.Errorf("connect rejected (class %d code %d)", r.ErrorClass, r.ErrorCode)
			}
			return nil, VMAC{}, errors.New("connect rejected by hub")
		default:
			// ignore advertisements etc. until we see the accept
		}
	}
}
