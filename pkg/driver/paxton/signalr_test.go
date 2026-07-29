package paxton

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/smart-core-os/sc-bos/pkg/driver/paxton/config"
)

// newSignalRTestClient builds a Client pointed at the given test server with retries disabled.
func newSignalRTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	cli := retryablehttp.NewClient()
	cli.RetryMax = 0
	cli.Logger = nil
	return NewClient(cli, zap.NewNop(), config.Root{
		BaseUrl: srv.URL,
		Auth:    config.Auth{},
	}, nil)
}

// authHandler handles the token endpoint used by all SignalR calls.
func authHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AuthResponse{
		AccessToken:    "test-token",
		ExpiryDatetime: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
	})
}

// ---- parseSignalRMessage ----

func TestParseSignalRMessage_LiveEvent(t *testing.T) {
	e := SignalRLiveEvent{EventType: 1, Details: "Door opened", UserID: 42}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: signalrMethodLiveEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	require.Len(t, dispatches, 1)
	assert.NotNil(t, dispatches[0].liveEvent)
	assert.Equal(t, 1, dispatches[0].liveEvent.EventType)
	assert.Equal(t, 42, dispatches[0].liveEvent.UserID)
}

func TestParseSignalRMessage_DoorEvent(t *testing.T) {
	e := SignalRDoorEvent{DoorID: 5, Locked: true}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: signalrMethodDoorEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	require.Len(t, dispatches, 1)
	assert.NotNil(t, dispatches[0].doorEvent)
	assert.Equal(t, 5, dispatches[0].doorEvent.DoorID)
	assert.True(t, dispatches[0].doorEvent.Locked)
}

func TestParseSignalRMessage_DoorStatusEvent(t *testing.T) {
	e := SignalRDoorStatusEvent{
		DoorID: 3,
		Status: SignalRDoorStatusEventStatus{DoorContactClosed: true, AlarmTripped: false},
	}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: signalrMethodDoorStatusEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	require.Len(t, dispatches, 1)
	assert.NotNil(t, dispatches[0].doorStatusEvent)
	assert.Equal(t, 3, dispatches[0].doorStatusEvent.DoorID)
	assert.True(t, dispatches[0].doorStatusEvent.Status.DoorContactClosed)
}

func TestParseSignalRMessage_RollCallEvent(t *testing.T) {
	e := SignalRRollCallEvent{
		RollCallReportID:  42,
		RollCallEventType: "muster",
		MusterEventData:   &SignalRMusterEventData{UserID: 7, MusteredAt: "2026-03-30T12:00:00Z"},
	}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: signalrMethodRollCallEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	require.Len(t, dispatches, 1)
	assert.NotNil(t, dispatches[0].rollCallEvent)
	assert.Equal(t, 42, dispatches[0].rollCallEvent.RollCallReportID)
}

func TestParseSignalRMessage_HubNameCaseInsensitive(t *testing.T) {
	e := SignalRDoorEvent{DoorID: 1}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: "EVENTHUBLOCAL", M: signalrMethodDoorEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	assert.Len(t, dispatches, 1)
}

func TestParseSignalRMessage_WrongHubSkipped(t *testing.T) {
	e := SignalRDoorEvent{DoorID: 1}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: "otherhub", M: signalrMethodDoorEvents, A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	assert.Empty(t, dispatches)
}

func TestParseSignalRMessage_UnknownMethodSkipped(t *testing.T) {
	arg, _ := json.Marshal(map[string]any{"foo": "bar"})
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: "unknownMethod", A: []json.RawMessage{arg}}},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	assert.Empty(t, dispatches)
}

func TestParseSignalRMessage_InvalidJSON(t *testing.T) {
	dispatches := new(Client).parseSignalRMessage([]byte("not json"))
	assert.Nil(t, dispatches)
}

func TestParseSignalRMessage_EmptyMArray(t *testing.T) {
	msg := signalrMessage{M: []signalrHubMessage{}}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	assert.Empty(t, dispatches)
}

func TestParseSignalRMessage_MultipleMessagesInOneFrame(t *testing.T) {
	door1, _ := json.Marshal(SignalRDoorEvent{DoorID: 1})
	door2, _ := json.Marshal(SignalRDoorEvent{DoorID: 2})
	msg := signalrMessage{
		M: []signalrHubMessage{
			{H: signalrHubName, M: signalrMethodDoorEvents, A: []json.RawMessage{door1, door2}},
		},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	assert.Len(t, dispatches, 2)
}

func TestParseSignalRMessage_MixedMethods(t *testing.T) {
	liveArg, _ := json.Marshal(SignalRLiveEvent{EventType: 10})
	doorArg, _ := json.Marshal(SignalRDoorEvent{DoorID: 3})
	msg := signalrMessage{
		M: []signalrHubMessage{
			{H: signalrHubName, M: signalrMethodLiveEvents, A: []json.RawMessage{liveArg}},
			{H: signalrHubName, M: signalrMethodDoorEvents, A: []json.RawMessage{doorArg}},
		},
	}
	data, _ := json.Marshal(msg)

	dispatches := new(Client).parseSignalRMessage(data)
	require.Len(t, dispatches, 2)
	assert.NotNil(t, dispatches[0].liveEvent)
	assert.NotNil(t, dispatches[1].doorEvent)
}

// ---- signalrNegotiate ----

func TestSignalRNegotiate_ReturnsConnectionToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(signalrNegotiateResponse{
			ConnectionToken: "tok-abc",
			TryWebSockets:   true,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	token, err := c.GetAccessToken(context.Background())
	require.NoError(t, err)

	connToken, err := c.signalrNegotiate(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, "tok-abc", connToken)
}

func TestSignalRNegotiate_ErrorWhenWebSocketsNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(signalrNegotiateResponse{
			ConnectionToken: "tok-abc",
			TryWebSockets:   false,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	token, err := c.GetAccessToken(context.Background())
	require.NoError(t, err)

	_, err = c.signalrNegotiate(context.Background(), token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WebSockets")
}

func TestSignalRNegotiate_ErrorOnBadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	token, err := c.GetAccessToken(context.Background())
	require.NoError(t, err)

	_, err = c.signalrNegotiate(context.Background(), token)
	require.Error(t, err)
}

// ---- signalrStart ----

func TestSignalRStart_SuccessOn200(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/signalr/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	token, err := c.GetAccessToken(context.Background())
	require.NoError(t, err)

	err = c.signalrStart(context.Background(), token, "conn-token")
	require.NoError(t, err)
}

func TestSignalRStart_ErrorOnNon200(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/signalr/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	token, err := c.GetAccessToken(context.Background())
	require.NoError(t, err)

	err = c.signalrStart(context.Background(), token, "conn-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

// ---- GetRollCallIDs ----

func TestGetRollCallIDs_ReturnsParsedIDs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/api/v1/rollcallreports", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]int{{"Id": 1}, {"Id": 2}, {"Id": 3}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ids, err := c.GetRollCallIDs(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, ids)
}

func TestGetRollCallIDs_EmptyList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)
	mux.HandleFunc("/api/v1/rollcallreports", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]int{})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ids, err := c.GetRollCallIDs(context.Background())
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// ---- streamSignalR ----

// setupSignalRServer starts a fake Paxton server that handles negotiate, start, and
// the WebSocket connect endpoint. wsHandler is called with the accepted server-side
// connection so tests can send frames and close the connection.
func setupSignalRServer(t *testing.T, wsHandler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/authorization/tokens", authHandler)

	mux.HandleFunc("/signalr/negotiate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(signalrNegotiateResponse{
			ConnectionToken: "test-conn-token",
			TryWebSockets:   true,
		})
	})

	mux.HandleFunc("/signalr/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/signalr/connect", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		wsHandler(conn)
	})

	return httptest.NewServer(mux)
}

func TestStreamSignalR_ReceivesLiveEvent(t *testing.T) {
	e := SignalRLiveEvent{UserID: 99, Details: "Granted"}
	arg, _ := json.Marshal(e)
	msg := signalrMessage{
		M: []signalrHubMessage{{H: signalrHubName, M: signalrMethodLiveEvents, A: []json.RawMessage{arg}}},
	}

	srv := setupSignalRServer(t, func(conn *websocket.Conn) {
		defer conn.CloseNow()
		ctx := context.Background()
		_ = wsjson.Write(ctx, conn, msg)
		// hold open briefly so the client goroutine can read the message
		time.Sleep(50 * time.Millisecond)
	})
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.streamSignalR(ctx, nil, nil)
	require.NoError(t, err)

	d, ok := <-ch
	require.True(t, ok, "expected a dispatch from channel")
	require.NotNil(t, d.liveEvent)
	assert.Equal(t, 99, d.liveEvent.UserID)
}

func TestStreamSignalR_ChannelClosedOnContextCancel(t *testing.T) {
	srv := setupSignalRServer(t, func(conn *websocket.Conn) {
		defer conn.CloseNow()
		// block until the client disconnects
		_, _, _ = conn.Read(context.Background())
	})
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.streamSignalR(ctx, nil, nil)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("channel was not closed after context cancellation")
	}
}

func TestStreamSignalR_ChannelClosedWhenServerDrops(t *testing.T) {
	srv := setupSignalRServer(t, func(conn *websocket.Conn) {
		// immediately close without sending anything
		conn.CloseNow()
	})
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.streamSignalR(ctx, nil, nil)
	require.NoError(t, err)

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed when server drops the connection")
	case <-ctx.Done():
		t.Fatal("channel was not closed after server dropped the connection")
	}
}

func TestStreamSignalR_SubscribesDoorAndRollCallIDs(t *testing.T) {
	received := make(chan signalrInvocation, 16)

	srv := setupSignalRServer(t, func(conn *websocket.Conn) {
		defer conn.CloseNow()
		ctx := context.Background()
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var inv signalrInvocation
			if err := json.Unmarshal(data, &inv); err == nil {
				received <- inv
			}
		}
	})
	defer srv.Close()

	c := newSignalRTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.streamSignalR(ctx, []int{10, 20}, []int{5})
	require.NoError(t, err)
	defer func() { cancel(); <-ch }()

	// Collect all invocations with a short deadline.
	deadline := time.After(500 * time.Millisecond)
	var invocations []signalrInvocation
loop:
	for {
		select {
		case inv := <-received:
			invocations = append(invocations, inv)
		case <-deadline:
			break loop
		}
	}

	methods := make([]string, 0, len(invocations))
	for _, inv := range invocations {
		methods = append(methods, inv.M)
	}

	assert.Contains(t, methods, signalrSubscribeLiveEvents)
	assert.Contains(t, methods, signalrSubscribeDoorEvents)
	assert.Contains(t, methods, signalrSubscribeDoorStatusEvents)
	assert.Contains(t, methods, signalrSubscribeRollCallEvents)
}
