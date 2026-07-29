package paxton

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	signalrProtocol = "1.5"
	signalrHubName  = "eventhublocal"

	signalrMethodLiveEvents       = "liveEvents"
	signalrMethodDoorEvents       = "doorEvents"
	signalrMethodDoorStatusEvents = "doorStatusEvents"
	signalrMethodRollCallEvents   = "rollCallEvents"

	signalrSubscribeLiveEvents       = "subscribeToLiveEvents"
	signalrSubscribeDoorEvents       = "subscribeToDoorEvents"
	signalrSubscribeDoorStatusEvents = "subscribeToDoorStatusEvents"
	signalrSubscribeRollCallEvents   = "subscribeToRollCallEvents"
)

// signalrDispatch holds exactly one non-nil field indicating which subscription the message came from.
type signalrDispatch struct {
	liveEvent       *SignalRLiveEvent
	doorEvent       *SignalRDoorEvent
	doorStatusEvent *SignalRDoorStatusEvent
	rollCallEvent   *SignalRRollCallEvent
}

// TokenCategory is the type of credential presented in a liveEvent.
type TokenCategory int

const (
	TokenCategoryToken TokenCategory = iota
	TokenCategoryPIN
	TokenCategoryVehicleReg
	TokenCategoryCode
)

// SignalRLiveEvent is delivered on the liveEvents stream — all monitorable access-control events.
// The schema differs from the REST API Event type: field names follow camelCase JSON conventions
// and some fields (e.g. id) are absent.
type SignalRLiveEvent struct {
	EventTime        string        `json:"eventTime"`
	EventType        int           `json:"eventType"`
	EventSubType     int           `json:"eventSubType"`
	TokenType        TokenCategory `json:"tokenType"`
	TokenNumber      int           `json:"tokenNumber"`
	UserID           int           `json:"userId"`
	UserName         string        `json:"userName"`
	DeviceID         int           `json:"deviceId"`
	AreaName         string        `json:"areaName"`
	Details          string        `json:"details"`
	NotificationType string        `json:"notificationType"`
}

// toEvent converts a SignalRLiveEvent to the common Event type used by the processing pipeline.
// The SignalR liveEvent schema lacks an ID field, so ID is left as zero. A non-nil error
// indicates EventTime could not be parsed; the returned Event still carries the other fields
// but its EventTime (and therefore its dedup fingerprint) is the zero time.
func (e *SignalRLiveEvent) toEvent() (Event, error) {
	t, err := time.Parse(time.RFC3339, e.EventTime)
	firstName, surname := e.UserName, ""
	if i := strings.IndexByte(e.UserName, ' '); i >= 0 {
		firstName = e.UserName[:i]
		surname = e.UserName[i+1:]
	}
	return Event{
		EventTime:        t,
		EventType:        e.EventType,
		EventSubType:     e.EventSubType,
		CardNo:           e.TokenNumber,
		UserID:           e.UserID,
		FirstName:        firstName,
		Surname:          surname,
		Address:          e.DeviceID,
		DeviceName:       e.AreaName,
		EventDescription: e.Details,
	}, err
}

// SignalRDoorEvent is delivered on the doorEvents stream — fired when a door locks or unlocks.
type SignalRDoorEvent struct {
	DoorID int  `json:"doorId"`
	Locked bool `json:"locked"`
}

// SignalRDoorStatusEvent is delivered on the doorStatusEvents stream — door status updates.
type SignalRDoorStatusEvent struct {
	DoorID int                          `json:"doorId"`
	Status SignalRDoorStatusEventStatus `json:"status"`
}

// SignalRDoorStatusEventStatus holds the individual contact and alarm state flags for a door.
type SignalRDoorStatusEventStatus struct {
	IntruderAlarmArmed  bool `json:"intruderAlarmArmed"`
	PsuContactClosed    bool `json:"psuContactClosed"`
	TamperContactClosed bool `json:"tamperContactClosed"`
	DoorContactClosed   bool `json:"doorContactClosed"`
	AlarmTripped        bool `json:"alarmTripped"`
	DoorRelayOpen       bool `json:"doorRelayOpen"`
}

// SignalRRollCallEvent is delivered on the rollCallEvents stream.
// Model from the Paxton API documentation.
type SignalRRollCallEvent struct {
	RollCallReportID          int                               `json:"rollCallReportId"`
	RollCallEventType         string                            `json:"rollCallEventType"`
	MusterEventData           *SignalRMusterEventData           `json:"musterEventData,omitempty"`
	MarkedSafeRecordEventData *SignalRMarkedSafeRecordEventData `json:"markedSafeRecordEventData,omitempty"`
}

type SignalRMusterEventData struct {
	UserID     int    `json:"userId"`
	MusteredAt string `json:"musteredAt"`
}

type SignalRMarkedSafeRecordEventData struct {
	UserID       int              `json:"userId"`
	MarkedSafeBy *SignalROperator `json:"markedSafeBy,omitempty"`
}

type SignalROperator struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

type signalrNegotiateResponse struct {
	ConnectionToken string `json:"ConnectionToken"`
	TryWebSockets   bool   `json:"TryWebSockets"`
}

// signalrMessage is the WebSocket frame envelope used by SignalR 2 for ASP.NET.
type signalrMessage struct {
	S int                 `json:"S"` // S=1 on the init/handshake frame
	M []signalrHubMessage `json:"M"` // hub method invocations
}

type signalrHubMessage struct {
	H string            `json:"H"` // hub name
	M string            `json:"M"` // method name
	A []json.RawMessage `json:"A"` // arguments
}

type signalrInvocation struct {
	H string   `json:"H"`
	M string   `json:"M"`
	A []string `json:"A"`
	I int      `json:"I"` // invocation sequence number
}

// streamSignalR opens a SignalR connection to eventHubLocal and subscribes to all four event streams:
//   - liveEvents: all monitorable access-control events
//   - doorEvents: per-door open/closed notifications (subscribed per door address)
//   - doorStatusEvents: per-door status updates (subscribed per door address)
//   - rollCallEvents: muster/safe updates (subscribed per roll call ID)
//
// The returned channel is closed when the context is cancelled or the connection drops.
// Callers should reconnect on channel close.
func (c *Client) streamSignalR(ctx context.Context, doorAddresses []int, rollCallIDs []int) (<-chan signalrDispatch, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	connToken, err := c.signalrNegotiate(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("negotiate: %w", err)
	}

	conn, err := c.signalrConnect(ctx, token, connToken)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := c.signalrStart(ctx, token, connToken); err != nil {
		if closeErr := conn.CloseNow(); closeErr != nil {
			c.logger.Error("failed to close signalr connection after start failure", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("start: %w", err)
	}

	seq := 1

	if err := c.signalrInvoke(ctx, conn, seq, signalrSubscribeLiveEvents); err != nil {
		if closeErr := conn.CloseNow(); closeErr != nil {
			c.logger.Error("failed to close signalr connection after subscribe failure", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("subscribe liveEvents: %w", err)
	}
	seq++

	for _, addr := range doorAddresses {
		addrStr := strconv.Itoa(addr)
		if err := c.signalrInvoke(ctx, conn, seq, signalrSubscribeDoorEvents, addrStr); err != nil {
			c.logger.Warn("failed to subscribe to doorEvents", zap.Int("address", addr), zap.Error(err))
		}
		seq++
		if err := c.signalrInvoke(ctx, conn, seq, signalrSubscribeDoorStatusEvents, addrStr); err != nil {
			c.logger.Warn("failed to subscribe to doorStatusEvents", zap.Int("address", addr), zap.Error(err))
		}
		seq++
	}

	for _, id := range rollCallIDs {
		if err := c.signalrInvoke(ctx, conn, seq, signalrSubscribeRollCallEvents, strconv.Itoa(id)); err != nil {
			c.logger.Warn("failed to subscribe to rollCallEvents", zap.Int("rollCallId", id), zap.Error(err))
		}
		seq++
	}

	ch := make(chan signalrDispatch, 64)
	go func() {
		defer close(ch)
		defer func() {
			if err := conn.CloseNow(); err != nil {
				c.logger.Error("failed to close signalr connection", zap.Error(err))
			}
		}()
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				if ctx.Err() == nil {
					c.logger.Error("signalr read error", zap.Error(err))
				}
				return
			}
			for _, d := range c.parseSignalRMessage(data) {
				select {
				case ch <- d:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// GetRollCallIDs fetches the current list of roll call report IDs from the API.
func (c *Client) GetRollCallIDs(ctx context.Context) ([]int, error) {
	reqURL, err := url.JoinPath(c.baseUrl, "api", "v1", "rollcallreports")
	if err != nil {
		return nil, err
	}
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	var reports []struct {
		ID int `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// signalrInvoke sends a hub method invocation over the WebSocket connection.
// Per the Paxton API guide, parameters are passed as strings.
// args is normalised to a non-nil slice so A always encodes as [] rather than null.
func (c *Client) signalrInvoke(ctx context.Context, conn *websocket.Conn, seq int, method string, args ...string) error {
	if args == nil {
		args = []string{}
	}
	payload, err := json.Marshal(signalrInvocation{
		H: signalrHubName,
		M: method,
		A: args,
		I: seq,
	})
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, payload)
}

// readErrBody reads up to 256 bytes from r to include in error messages.
func readErrBody(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 256))
	return strings.TrimSpace(string(b))
}

func (c *Client) signalrNegotiate(ctx context.Context, token string) (string, error) {
	reqURL, err := url.JoinPath(c.baseUrl, "signalr", "negotiate")
	if err != nil {
		return "", err
	}
	u, err := url.Parse(reqURL)
	if err != nil {
		return "", err
	}
	connData, _ := json.Marshal([]map[string]string{{"name": signalrHubName}})
	q := u.Query()
	q.Set("clientProtocol", signalrProtocol)
	q.Set("connectionData", string(connData))
	q.Set("token", token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.cli.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("negotiate: status %d: %s", resp.StatusCode, readErrBody(resp.Body))
	}

	var neg signalrNegotiateResponse
	if err := json.NewDecoder(resp.Body).Decode(&neg); err != nil {
		return "", err
	}
	if !neg.TryWebSockets {
		return "", fmt.Errorf("server does not support WebSockets")
	}
	return neg.ConnectionToken, nil
}

func (c *Client) signalrConnect(ctx context.Context, token, connToken string) (*websocket.Conn, error) {
	reqURL, err := url.JoinPath(c.baseUrl, "signalr", "connect")
	if err != nil {
		return nil, err
	}
	reqURL = strings.Replace(reqURL, "https://", "wss://", 1)
	reqURL = strings.Replace(reqURL, "http://", "ws://", 1)

	u, err := url.Parse(reqURL)
	if err != nil {
		return nil, err
	}
	connData, _ := json.Marshal([]map[string]string{{"name": signalrHubName}})
	q := u.Query()
	q.Set("transport", "webSockets")
	q.Set("clientProtocol", signalrProtocol)
	q.Set("connectionToken", connToken)
	q.Set("connectionData", string(connData))
	q.Set("token", token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPClient: c.cli.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(512 * 1024)
	return conn, nil
}

func (c *Client) signalrStart(ctx context.Context, token, connToken string) error {
	reqURL, err := url.JoinPath(c.baseUrl, "signalr", "start")
	if err != nil {
		return err
	}
	u, err := url.Parse(reqURL)
	if err != nil {
		return err
	}
	connData, _ := json.Marshal([]map[string]string{{"name": signalrHubName}})
	q := u.Query()
	q.Set("transport", "webSockets")
	q.Set("clientProtocol", signalrProtocol)
	q.Set("connectionToken", connToken)
	q.Set("connectionData", string(connData))
	q.Set("token", token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.cli.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close signalr start response body", zap.Error(err))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) parseSignalRMessage(data []byte) []signalrDispatch {
	var msg signalrMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}
	var dispatches []signalrDispatch
	for _, m := range msg.M {
		if !strings.EqualFold(m.H, signalrHubName) {
			continue
		}
		for _, arg := range m.A {
			switch m.M {
			case signalrMethodLiveEvents:
				var e SignalRLiveEvent
				if err := json.Unmarshal(arg, &e); err == nil {
					dispatches = append(dispatches, signalrDispatch{liveEvent: &e})
				}
			case signalrMethodDoorEvents:
				var e SignalRDoorEvent
				if err := json.Unmarshal(arg, &e); err == nil {
					dispatches = append(dispatches, signalrDispatch{doorEvent: &e})
				}
			case signalrMethodDoorStatusEvents:
				var e SignalRDoorStatusEvent
				if err := json.Unmarshal(arg, &e); err == nil {
					dispatches = append(dispatches, signalrDispatch{doorStatusEvent: &e})
				}
			case signalrMethodRollCallEvents:
				var e SignalRRollCallEvent
				if err := json.Unmarshal(arg, &e); err == nil {
					dispatches = append(dispatches, signalrDispatch{rollCallEvent: &e})
				}
			}
		}
	}
	return dispatches
}
