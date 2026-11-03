package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
)

// TestElectric_GetDemand_neverRead is the empty demand the trait used to serve without error
// for a device it had never read.
func TestElectric_GetDemand_neverRead(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadTimeout}
	electric := newTestElectric(t, reader, `{"realPower":{"nodeId":"ns=2;s=Power"}}`)

	demand, err := electric.GetDemand(t.Context(), &electricpb.GetDemandRequest{})
	requireUnavailable(t, err)
	require.Nil(t, demand)
	require.Equal(t, 1, reader.readCount())
}

const (
	testPowerNode  = "ns=2;s=Power"
	testFactorNode = "ns=2;s=Factor"
	// testElectricDemand maps two fields, so a test can have one vouched and one not
	testElectricDemand = `{"realPower":{"nodeId":"` + testPowerNode + `"},"powerFactor":{"nodeId":"` + testFactorNode + `"}}`
)

func newTestElectric(t *testing.T, reader pointReader, demand string) *Electric {
	t.Helper()
	electric, err := newElectric("test/device", config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":` + demand + `}`),
	}, reader, zaptest.NewLogger(t))
	require.NoError(t, err)
	return electric
}

func ptr(v float32) *float32 { return &v }

// TestElectric_GetDemand_vouched checks fields the subscription vouches for are served without
// asking the server.
func TestElectric_GetDemand_vouched(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadTimeout}
	electric := newTestElectric(t, reader, testElectricDemand)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))
	electric.handleEvent(t.Context(), mustParseNodeID(testFactorNode), float32(0.9))

	demand, err := electric.GetDemand(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, ptr(1500), demand.RealPower)
	require.Equal(t, ptr(0.9), demand.PowerFactor)
	require.Zero(t, reader.readCount())
}

// TestElectric_GetDemand_partial checks a field the subscription cannot vouch for is read live
// alongside the vouched ones, and only that field is asked for.
func TestElectric_GetDemand_partial(t *testing.T) {
	reader := &fakeReader{value: goodValue(float32(0.8))}
	electric := newTestElectric(t, reader, testElectricDemand)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))

	demand, err := electric.GetDemand(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, ptr(1500), demand.RealPower)
	require.Equal(t, ptr(0.8), demand.PowerFactor)
	require.Equal(t, []string{testFactorNode}, reader.lastRead())

	// and a field that still cannot be read is left out rather than served stale or as zero
	reader.answer(&ua.DataValue{Status: ua.StatusBadNoCommunication}, nil)
	demand, err = electric.GetDemand(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, ptr(1500), demand.RealPower)
	require.Nil(t, demand.PowerFactor)
}

// TestElectric_GetDemand_bad checks a Bad point's field is left out of Get, even though the
// cache still holds the value it had.
func TestElectric_GetDemand_bad(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadTimeout}
	electric := newTestElectric(t, reader, testElectricDemand)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))
	electric.handleEvent(t.Context(), mustParseNodeID(testFactorNode), float32(0.9))
	electric.pointBad(mustParseNodeID(testFactorNode))

	demand, err := electric.GetDemand(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, ptr(1500), demand.RealPower)
	require.Nil(t, demand.PowerFactor)

	// with every field gone there is nothing to serve
	electric.pointBad(mustParseNodeID(testPowerNode))
	_, err = electric.GetDemand(t.Context(), nil)
	requireUnavailable(t, err)
}

// TestElectric_GetDemand_connectionSuspect is the blip gopcua repairs underneath us: every
// field is read live once, in one request, and a good read puts them back on the
// subscription's word.
func TestElectric_GetDemand_connectionSuspect(t *testing.T) {
	reader := &fakeReader{}
	reader.answerNode(testPowerNode, goodValue(float32(1500)))
	reader.answerNode(testFactorNode, goodValue(float32(0.9)))
	electric := newTestElectric(t, reader, testElectricDemand)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))
	electric.handleEvent(t.Context(), mustParseNodeID(testFactorNode), float32(0.9))
	electric.connectionSuspect()

	for range 2 {
		demand, err := electric.GetDemand(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, ptr(1500), demand.RealPower)
		require.Equal(t, ptr(0.9), demand.PowerFactor)
	}
	require.Equal(t, 1, reader.readCount(), "one good read after the blip is enough")
	require.ElementsMatch(t, []string{testPowerNode, testFactorNode}, reader.lastRead())
}

func TestElectric_GetDemand_subscriptionLost(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadConnectionClosed}
	electric := newTestElectric(t, reader, testElectricDemand)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))
	electric.subscriptionLost()

	_, err := electric.GetDemand(t.Context(), nil)
	requireUnavailable(t, err)
}

// TestElectric_readLive_updatesCache checks a live read feeds the demand Pull serves, but only
// on a change.
func TestElectric_readLive_updatesCache(t *testing.T) {
	reader := &fakeReader{value: goodValue(float32(2.5))}
	electric, err := newElectric("test/device", config.RawTrait{
		Raw: []byte(`{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"` + testPowerNode + `","scale":1000}}}`),
	}, reader, zaptest.NewLogger(t))
	require.NoError(t, err)

	demand, err := electric.GetDemand(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, ptr(2500), demand.RealPower, "a live read is scaled like a notification")
	cached := electric.value.Get().(*electricpb.ElectricDemand)
	require.Equal(t, ptr(2500), cached.RealPower)
}

// TestElectric_sharedNode checks two fields mapped to the same node both follow its
// notifications. They share the node's liveness, so a field a notification skipped would be
// vouched for with whatever a live read last cached, however stale.
func TestElectric_sharedNode(t *testing.T) {
	reader := &fakeReader{err: ua.StatusBadTimeout}
	electric := newTestElectric(t, reader,
		`{"realPower":{"nodeId":"`+testPowerNode+`"},"apparentPower":{"nodeId":"`+testPowerNode+`","scale":2}}`)
	node := mustParseNodeID(testPowerNode)

	for _, v := range []float32{1500, 1600} {
		electric.handleEvent(t.Context(), node, v)
		demand, err := electric.GetDemand(t.Context(), nil)
		require.NoError(t, err)
		require.Equal(t, ptr(v), demand.RealPower)
		require.Equal(t, ptr(2*v), demand.ApparentPower, "each field is scaled on its own")
	}
	require.Zero(t, reader.readCount())
}

// TestElectric_phasesOnly checks a config that only sets phases, which leaves the embedded
// single-phase config nil, neither panics on a notification nor claims a reading.
func TestElectric_phasesOnly(t *testing.T) {
	electric := newTestElectric(t, &fakeReader{}, `{"phases":[{"realPower":{"nodeId":"`+testPowerNode+`"}}]}`)
	electric.handleEvent(t.Context(), mustParseNodeID(testPowerNode), float32(1500))
	_, err := electric.GetDemand(t.Context(), nil)
	require.Equal(t, codes.Unavailable, status.Code(err))
}

func TestElectric_handleElectricEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      string
		nodeId      string
		value       any
		checkField  string
		wantValue   float32
		wantHandled bool
	}{
		{
			name:        "real power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       float32(1500.5),
			checkField:  "realPower",
			wantValue:   1500.5,
			wantHandled: true,
		},
		{
			name:        "apparent power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"apparentPower":{"nodeId":"ns=2;s=ApparentPower"}}}`,
			nodeId:      "ns=2;s=ApparentPower",
			value:       float32(2000.0),
			checkField:  "apparentPower",
			wantValue:   2000.0,
			wantHandled: true,
		},
		{
			name:        "reactive power",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"reactivePower":{"nodeId":"ns=2;s=ReactivePower"}}}`,
			nodeId:      "ns=2;s=ReactivePower",
			value:       float32(500.0),
			checkField:  "reactivePower",
			wantValue:   500.0,
			wantHandled: true,
		},
		{
			name:        "power factor",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"powerFactor":{"nodeId":"ns=2;s=PF"}}}`,
			nodeId:      "ns=2;s=PF",
			value:       float32(0.95),
			checkField:  "powerFactor",
			wantValue:   0.95,
			wantHandled: true,
		},
		{
			name:        "real power with scaling (kW to W)",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power","scale":1000}}}`,
			nodeId:      "ns=2;s=Power",
			value:       float32(2.5),
			checkField:  "realPower",
			wantValue:   2500.0,
			wantHandled: true,
		},
		{
			name:        "int32 value converted to float32",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       int32(1000),
			checkField:  "realPower",
			wantValue:   1000.0,
			wantHandled: true,
		},
		{
			name:        "wrong node id - not handled",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=WrongNode",
			value:       float32(1500.0),
			checkField:  "realPower",
			wantValue:   0,
			wantHandled: false,
		},
		{
			name:        "invalid value type",
			config:      `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			nodeId:      "ns=2;s=Power",
			value:       "not a number",
			checkField:  "realPower",
			wantValue:   0,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			// a reader that always fails, so a value the trait did not take is Unavailable
			// rather than served from a live read
			electric, err := newElectric("test/device", cfg, &fakeReader{err: ua.StatusBadTimeout}, logger)
			if err != nil {
				t.Fatalf("newElectric() error = %v", err)
			}

			nodeId, _ := ua.ParseNodeID(tt.nodeId)
			electric.handleEvent(t.Context(), nodeId, tt.value)

			demand, err := electric.GetDemand(t.Context(), &electricpb.GetDemandRequest{})
			if !tt.wantHandled {
				if status.Code(err) != codes.Unavailable {
					t.Errorf("After handleElectricEvent, GetDemand() error = %v, want Unavailable", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetDemand() error = %v", err)
			}

			var actualValue float32
			switch tt.checkField {
			case "realPower":
				if demand.RealPower != nil {
					actualValue = *demand.RealPower
				}
			case "apparentPower":
				if demand.ApparentPower != nil {
					actualValue = *demand.ApparentPower
				}
			case "reactivePower":
				if demand.ReactivePower != nil {
					actualValue = *demand.ReactivePower
				}
			case "powerFactor":
				if demand.PowerFactor != nil {
					actualValue = *demand.PowerFactor
				}
			}

			if actualValue != tt.wantValue {
				t.Errorf("After handleElectricEvent, %s = %v, want %v", tt.checkField, actualValue, tt.wantValue)
			}
		})
	}
}

func TestElectric_handleElectricEvent_NoDemand(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create Electric with nil demand (should be caught by validation, but test defensive code)
	electric := &Electric{
		cfg:    config.ElectricConfig{},
		logger: logger,
		scName: "test/device",
	}

	nodeId, _ := ua.ParseNodeID("ns=2;s=Power")
	// Should not panic, just log warning and return
	electric.handleEvent(t.Context(), nodeId, float32(100.0))
}

func TestElectric_newElectric_InvalidConfig(t *testing.T) {
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
			config:    `{"kind":"smartcore.traits.Electric","demand":{"realPower":{"nodeId":"ns=2;s=Power"}}}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.RawTrait{Raw: []byte(tt.config)}
			_, err := newElectric("test/device", cfg, nil, logger)
			if (err != nil) != tt.wantError {
				t.Errorf("newElectric() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
