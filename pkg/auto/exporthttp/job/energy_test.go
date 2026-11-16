package job

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// testMeterInfoServer returns a fixed usage unit, or err, for all DescribeMeterReading calls.
type testMeterInfoServer struct {
	meterpb.UnimplementedMeterInfoServer
	usageUnit string
	err       error
}

func (s *testMeterInfoServer) DescribeMeterReading(_ context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &meterpb.MeterReadingSupport{UsageUnit: s.usageUnit}, nil
}

// perMeterInfoServer returns a usage unit per meter name.
type perMeterInfoServer struct {
	meterpb.UnimplementedMeterInfoServer
	units map[string]string
}

func (s *perMeterInfoServer) DescribeMeterReading(_ context.Context, req *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	return &meterpb.MeterReadingSupport{UsageUnit: s.units[req.Name]}, nil
}

// testMeterHistoryServer returns a zero-usage record for "old" EndTime queries
// (before boundary) and a latestUsage record for "recent" queries (after boundary).
// This maps to how getRecordsByTime calls ListMeterReadingHistory twice:
// once with EndTime approx. = PreviousExecution and once with EndTime approx. = now.
type testMeterHistoryServer struct {
	meterpb.UnimplementedMeterHistoryServer
	boundary    time.Time
	latestUsage float32
}

func (s *testMeterHistoryServer) ListMeterReadingHistory(_ context.Context, req *meterpb.ListMeterReadingHistoryRequest) (*meterpb.ListMeterReadingHistoryResponse, error) {
	var usage float32
	if req.GetPeriod().GetEndTime().AsTime().After(s.boundary) {
		usage = s.latestUsage
	}
	return &meterpb.ListMeterReadingHistoryResponse{
		MeterReadingRecords: []*meterpb.MeterReadingRecord{
			{MeterReading: &meterpb.MeterReading{Usage: usage}},
		},
	}, nil
}

// newEnergyTestJob builds an EnergyJob wired to in-process test servers.
// nativeConsumption is the meter reading delta in the meter's native unit.
// PreviousExecution defaults to time.Time{} (year 0001) so the "earliest" query's
// EndTime is ancient, well before the year-2000 boundary used by the history server.
func newEnergyTestJob(info meterpb.MeterInfoServer, nativeConsumption float32, meters ...string) *EnergyJob {
	boundary := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	historyConn := wrap.ServerToClient(meterpb.MeterHistory_ServiceDesc, &testMeterHistoryServer{
		boundary:    boundary,
		latestUsage: nativeConsumption,
	})
	infoConn := wrap.ServerToClient(meterpb.MeterInfo_ServiceDesc, info)
	if len(meters) == 0 {
		meters = []string{"test-meter"}
	}
	return &EnergyJob{
		BaseJob:    BaseJob{Logger: zap.NewNop()},
		client:     meterpb.NewMeterHistoryClient(historyConn),
		infoClient: meterpb.NewMeterInfoClient(infoConn),
		Meters:     meters,
	}
}

func Test_EnergyJob_Do(t *testing.T) {
	tests := []struct {
		name     string
		unit     string
		infoErr  error
		native   float32 // consumption in the meter's native unit
		wantKWh  float32 // expected reported value in kWh
		wantSkip bool    // the meter can't be converted, so nothing is sent
	}{
		{name: "1000 Wh -> 1 kWh", unit: "wh", native: 1000, wantKWh: 1},
		{name: "Wh mixed case", unit: "Wh", native: 1000, wantKWh: 1},
		{name: "watt_hours", unit: "watt_hours", native: 2000, wantKWh: 2},
		{name: "1 kWh -> 1 kWh", unit: "kwh", native: 1, wantKWh: 1},
		{name: "kWh mixed case", unit: "kWh", native: 5, wantKWh: 5},
		{name: "1 MWh -> 1000 kWh", unit: "mwh", native: 1, wantKWh: 1000},
		{name: "empty unit skipped", unit: "", native: 1000, wantSkip: true},
		{name: "power unit skipped", unit: "W", native: 1000, wantSkip: true},
		{name: "volume unit skipped", unit: "m³", native: 1000, wantSkip: true},
		{name: "describe error skipped", infoErr: status.Error(codes.Unavailable, "meter offline"), native: 1000, wantSkip: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := newEnergyTestJob(&testMeterInfoServer{usageUnit: tt.unit, err: tt.infoErr}, tt.native)

			var captured []byte
			err := job.Do(context.Background(), func(_ context.Context, _ string, body []byte) error {
				captured = body
				return nil
			})
			require.NoError(t, err)
			if tt.wantSkip {
				require.Nil(t, captured, "sendFn should not have been called, got %s", captured)
				return
			}
			require.NotNil(t, captured, "sendFn should have been called")

			var result types.EnergyConsumption
			require.NoError(t, json.Unmarshal(captured, &result))
			assert.Equal(t, tt.wantKWh, result.TodaysEnergyConsumption.Value)
			assert.Equal(t, "kWh", result.TodaysEnergyConsumption.Units)
		})
	}
}

// A meter in an unknown unit is left out of the total rather than counted as kWh.
func Test_EnergyJob_Do_skipsOnlyUnknownMeters(t *testing.T) {
	info := &perMeterInfoServer{units: map[string]string{
		"wh-meter":      "Wh",
		"unknown-meter": "",
	}}
	job := newEnergyTestJob(info, 3000, "wh-meter", "unknown-meter")

	var captured []byte
	err := job.Do(context.Background(), func(_ context.Context, _ string, body []byte) error {
		captured = body
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, captured, "sendFn should have been called")
	var result types.EnergyConsumption
	require.NoError(t, json.Unmarshal(captured, &result))
	// 3000 Wh is 3 kWh; before SCB-1444 the unknown meter's 3000 was added as kWh too
	assert.Equal(t, float32(3), result.TodaysEnergyConsumption.Value)
}
