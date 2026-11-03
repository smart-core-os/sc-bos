package opcua

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

// fakeReader stands in for *Client's ReadValues, counting the requests and recording what each
// asked for, so a test can tell a value served from the subscription from one read live. Every
// node answers with value unless nodes has an answer of its own.
type fakeReader struct {
	mu    sync.Mutex
	value *ua.DataValue
	nodes map[string]*ua.DataValue
	err   error
	asked [][]string // the node ids of each request, in order
	down  bool       // what Connected reports, inverted so the zero value is connected
}

func (r *fakeReader) Connected() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.down
}

func (r *fakeReader) setDown(down bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.down = down
}

func (r *fakeReader) ReadValues(_ context.Context, nodeIds ...*ua.NodeID) ([]*ua.DataValue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var asked []string
	var out []*ua.DataValue
	for _, nodeId := range nodeIds {
		asked = append(asked, nodeId.String())
		dv, ok := r.nodes[nodeId.String()]
		if !ok {
			dv = r.value
		}
		out = append(out, dv)
	}
	r.asked = append(r.asked, asked)
	if r.err != nil {
		return nil, r.err
	}
	return out, nil
}

func (r *fakeReader) answer(value *ua.DataValue, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value, r.err = value, err
}

// answerNode sets the answer for one node, overriding value.
func (r *fakeReader) answerNode(nodeId string, value *ua.DataValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.nodes == nil {
		r.nodes = make(map[string]*ua.DataValue)
	}
	r.nodes[nodeId] = value
}

func (r *fakeReader) readCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.asked)
}

// lastRead is the node ids the most recent request asked for.
func (r *fakeReader) lastRead() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.asked) == 0 {
		return nil
	}
	return r.asked[len(r.asked)-1]
}

func goodValue(v any) *ua.DataValue {
	return &ua.DataValue{Value: ua.MustVariant(v), Status: ua.StatusOK}
}

const testMeterNode = "ns=2;s=Tag1"

func newTestMeter(t *testing.T, reader pointReader) *Meter {
	t.Helper()
	meter, err := newMeter("test/device", config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"` + testMeterNode + `"}}`),
	}, reader, zaptest.NewLogger(t))
	require.NoError(t, err)
	return meter
}

func requireUnavailable(t *testing.T, err error) {
	t.Helper()
	require.Equal(t, codes.Unavailable, status.Code(err), "error = %v", err)
}

// TestMeter_GetMeterReading_neverRead is the meter that used to be exported as a live 0 every
// cycle: the subscription has never delivered, so there is no reading to serve.
func TestMeter_GetMeterReading_neverRead(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadTimeout}
	meter := newTestMeter(t, reader)

	reading, err := meter.GetMeterReading(t.Context(), &meterpb.GetMeterReadingRequest{})
	requireUnavailable(t, err)
	require.Nil(t, reading)
	require.Equal(t, 1, reader.readCount(), "a meter with nothing delivered should try a live read")

	// a live read that works serves its value, but vouches for nothing: the monitored item has
	// still never delivered, so the next Get reads again
	reader.answer(goodValue(float32(42)), nil)
	for want := 2; want <= 3; want++ {
		reading, err = meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(42), reading.Usage)
		require.NotNil(t, reading.EndTime)
		require.Equal(t, want, reader.readCount())
	}
}

func TestMeter_GetMeterReading_noReader(t *testing.T) {
	meter := newTestMeter(t, nil)
	_, err := meter.GetMeterReading(t.Context(), nil)
	requireUnavailable(t, err)
}

// TestMeter_GetMeterReading_delivered is the steady meter: one notification, then a fresh
// reading on every Get without asking the server, because the monitored item is the server
// sampling it.
func TestMeter_GetMeterReading_delivered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := &fakeReader{err: errors.New("should not be read")}
		meter := newTestMeter(t, reader)
		meter.handleEvent(t.Context(), mustParseNodeID(testMeterNode), float32(7))

		first, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(7), first.Usage)
		require.WithinDuration(t, time.Now(), first.EndTime.AsTime(), 0)

		time.Sleep(5 * time.Minute)
		second, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(7), second.Usage)
		require.WithinDuration(t, time.Now(), second.EndTime.AsTime(), 0, "each Get is stamped when it is served")
		require.Zero(t, reader.readCount())

		// the cached reading keeps the time the value changed, which is what Pull reports
		cached := meter.energyValue.Get().(*meterpb.MeterReading)
		require.Equal(t, first.EndTime.AsTime(), cached.EndTime.AsTime())
	})
}

// TestMeter_GetMeterReading_bad checks a Bad notification stops the subscription vouching for
// the value, and that even a good live read does not restore it: the monitored item itself is
// what reported Bad, so only it can say it has recovered.
func TestMeter_GetMeterReading_bad(t *testing.T) {
	reader := &fakeReader{value: &ua.DataValue{Status: ua.StatusBadCommunicationError}}
	meter := newTestMeter(t, reader)
	node := mustParseNodeID(testMeterNode)
	meter.handleEvent(t.Context(), node, float32(7))
	meter.pointBad(node)

	_, err := meter.GetMeterReading(t.Context(), nil)
	requireUnavailable(t, err)
	require.Equal(t, 1, reader.readCount())

	reader.answer(goodValue(float32(8)), nil)
	for want := 2; want <= 3; want++ {
		reading, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(8), reading.Usage)
		require.Equal(t, want, reader.readCount(), "a Bad point is read live on every Get")
	}

	// the next notification is the item recovering
	meter.handleEvent(t.Context(), node, float32(9))
	reading, err := meter.GetMeterReading(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, float32(9), reading.Usage)
	require.Equal(t, 3, reader.readCount())
}

// TestMeter_pointBad_otherNode checks a Bad status on some other point leaves the meter alone.
func TestMeter_pointBad_otherNode(t *testing.T) {
	meter := newTestMeter(t, &fakeReader{})
	meter.handleEvent(t.Context(), mustParseNodeID(testMeterNode), float32(7))
	meter.pointBad(mustParseNodeID("ns=2;s=Other"))
	require.True(t, meter.vouched())
}

// TestMeter_GetMeterReading_suspectThenGoodRead is the blip gopcua repairs underneath us. The
// transferred subscription does not resend an idle meter's value, so one live read has to
// show it survived, after which the subscription vouches for it again.
func TestMeter_GetMeterReading_suspectThenGoodRead(t *testing.T) {
	reader := &fakeReader{value: goodValue(float32(7))}
	meter := newTestMeter(t, reader)
	meter.handleEvent(t.Context(), mustParseNodeID(testMeterNode), float32(7))
	meter.connectionSuspect()

	for range 2 {
		reading, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(7), reading.Usage)
	}
	require.Equal(t, 1, reader.readCount(), "one good read after the blip is enough")
}

func TestMeter_GetMeterReading_suspectThenFailedRead(t *testing.T) {
	tests := []struct {
		name  string
		value *ua.DataValue
		err   error
	}{
		{name: "read error", err: ua.StatusBadConnectionClosed},
		{name: "bad status", value: &ua.DataValue{Status: ua.StatusBadNoCommunication}},
		{name: "no value", value: &ua.DataValue{Status: ua.StatusOK}},
		{name: "no data value"},
		{name: "not a number", value: goodValue("seven")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &fakeReader{value: tt.value, err: tt.err}
			meter := newTestMeter(t, reader)
			meter.handleEvent(t.Context(), mustParseNodeID(testMeterNode), float32(7))
			meter.connectionSuspect()

			for want := 1; want <= 2; want++ {
				reading, err := meter.GetMeterReading(t.Context(), nil)
				requireUnavailable(t, err)
				require.Nil(t, reading)
				require.Equal(t, want, reader.readCount(), "a failed read vouches for nothing")
			}
		})
	}
}

// TestMeter_GetMeterReading_subscriptionLost checks a meter reads live once its subscription
// has gone, until a new one delivers, and that a good live read does not stand in for that.
func TestMeter_GetMeterReading_subscriptionLost(t *testing.T) {
	reader := &fakeReader{value: goodValue(float32(7))}
	meter := newTestMeter(t, reader)
	node := mustParseNodeID(testMeterNode)
	meter.handleEvent(t.Context(), node, float32(7))
	meter.subscriptionLost()

	for want := 1; want <= 2; want++ {
		_, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, want, reader.readCount())
	}

	meter.handleEvent(t.Context(), node, float32(7))
	_, err := meter.GetMeterReading(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, 2, reader.readCount(), "the new subscription vouches once it delivers")
}

// TestMeter_readLive_cache checks a live read feeds the cached reading Pull serves, but only
// on a change, so Pull still emits once per value, stamped with when it changed.
func TestMeter_readLive_cache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := &fakeReader{value: goodValue(float32(2.5))}
		meter, err := newMeter("test/device", config.RawTrait{
			Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"Wh","usage":{"nodeId":"` + testMeterNode + `","scale":1000}}`),
		}, reader, zaptest.NewLogger(t))
		require.NoError(t, err)

		reading, err := meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, float32(2500), reading.Usage, "a live read is scaled like a notification")
		changed := time.Now()

		time.Sleep(time.Minute)
		_, err = meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		cached := meter.energyValue.Get().(*meterpb.MeterReading)
		require.Equal(t, float32(2500), cached.Usage)
		require.WithinDuration(t, changed, cached.EndTime.AsTime(), 0, "an unchanged read leaves the cache alone")

		reader.answer(goodValue(float32(3)), nil)
		_, err = meter.GetMeterReading(t.Context(), nil)
		require.NoError(t, err)
		cached = meter.energyValue.Get().(*meterpb.MeterReading)
		require.Equal(t, float32(3000), cached.Usage)
		require.WithinDuration(t, time.Now(), cached.EndTime.AsTime(), 0)
	})
}

// TestMeter_GetMeterReading_connectionDown checks a delivered value is not served as current
// while the connection is down, even when nothing cleared it: a notification in flight when the
// connection drops can land after the drop, and gopcua reports a long outage only once.
func TestMeter_GetMeterReading_connectionDown(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadServerNotConnected}
	meter := newTestMeter(t, reader)
	node := mustParseNodeID(testMeterNode)
	meter.connectionSuspect()
	meter.handleEvent(t.Context(), node, float32(7)) // the value that was in flight
	reader.setDown(true)

	_, err := meter.GetMeterReading(t.Context(), nil)
	requireUnavailable(t, err)

	reader.setDown(false)
	reading, err := meter.GetMeterReading(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, float32(7), reading.Usage)
}

func TestMeter_DescribeMeterReading(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`),
	}

	meter, err := newMeter("test/device", cfg, nil, logger)
	if err != nil {
		t.Fatalf("newMeter() error = %v", err)
	}

	support, err := meter.DescribeMeterReading(t.Context(), &meterpb.DescribeMeterReadingRequest{})
	if err != nil {
		t.Errorf("DescribeMeterReading() error = %v", err)
	}
	if support.UsageUnit != "kWh" {
		t.Errorf("UsageUnit = %v, want kWh", support.UsageUnit)
	}
}

func TestMeter_handleMeterEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      string
		nodeId      string
		value       any
		wantUsage   float32
		wantHandled bool
	}{
		{
			name:        "valid float32 value",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       float32(123.45),
			wantUsage:   123.45,
			wantHandled: true,
		},
		{
			name:        "valid int32 value",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       int32(100),
			wantUsage:   100.0,
			wantHandled: true,
		},
		{
			name:        "value with scaling",
			config:      `{"kind":"smartcore.bos.Meter","unit":"Wh","usage":{"nodeId":"ns=2;s=Tag1","scale":1000}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       float32(2.5),
			wantUsage:   2500.0,
			wantHandled: true,
		},
		{
			name:        "wrong node id - not handled",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag2",
			value:       float32(123.45),
			wantHandled: false,
		},
		{
			name:        "invalid value type - not handled",
			config:      `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			nodeId:      "ns=2;s=Tag1",
			value:       "not a number",
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			// a reader that always fails, so a value the meter did not take is Unavailable
			// rather than served from a live read
			meter, err := newMeter("test/device", cfg, &fakeReader{err: ua.StatusBadTimeout}, logger)
			if err != nil {
				t.Fatalf("newMeter() error = %v", err)
			}

			nodeId, _ := ua.ParseNodeID(tt.nodeId)
			meter.handleEvent(t.Context(), nodeId, tt.value)

			reading, err := meter.GetMeterReading(t.Context(), &meterpb.GetMeterReadingRequest{})
			if !tt.wantHandled {
				if status.Code(err) != codes.Unavailable {
					t.Errorf("After handleMeterEvent, GetMeterReading() error = %v, want Unavailable", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetMeterReading() error = %v", err)
			}
			if reading.Usage != tt.wantUsage {
				t.Errorf("After handleMeterEvent, usage = %v, want %v", reading.Usage, tt.wantUsage)
			}
		})
	}
}

// TestMeter_handleEvent_unusableValue checks a value the meter cannot convert stops it serving
// the one before, which is no longer what the server holds.
func TestMeter_handleEvent_unusableValue(t *testing.T) {
	meter := newTestMeter(t, &fakeReader{err: ua.StatusBadTimeout})
	node := mustParseNodeID(testMeterNode)
	meter.handleEvent(t.Context(), node, float32(7))
	meter.handleEvent(t.Context(), node, "not a number")

	_, err := meter.GetMeterReading(t.Context(), nil)
	requireUnavailable(t, err)
}

func TestMeter_newMeter_InvalidConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name:      "invalid json",
			config:    `{invalid json}`,
			wantError: true,
		},
		{
			name:      "valid config",
			config:    `{"kind":"smartcore.bos.Meter","unit":"kWh","usage":{"nodeId":"ns=2;s=Tag1"}}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newMeter("test/device", cfg, nil, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newMeter() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
