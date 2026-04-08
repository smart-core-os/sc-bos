package job

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/exporthttp/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// testMeterInfoServer returns a fixed usage unit for all DescribeMeterReading calls.
type testMeterInfoServer struct {
	meterpb.UnimplementedMeterInfoServer
	usageUnit string
}

func (s *testMeterInfoServer) DescribeMeterReading(_ context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	return &meterpb.MeterReadingSupport{UsageUnit: s.usageUnit}, nil
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
func newEnergyTestJob(usageUnit string, nativeConsumption float32) *EnergyJob {
	boundary := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	historyConn := wrap.ServerToClient(meterpb.MeterHistory_ServiceDesc, &testMeterHistoryServer{
		boundary:    boundary,
		latestUsage: nativeConsumption,
	})
	infoConn := wrap.ServerToClient(meterpb.MeterInfo_ServiceDesc, &testMeterInfoServer{usageUnit: usageUnit})
	return &EnergyJob{
		BaseJob:    BaseJob{Logger: zap.NewNop()},
		client:     meterpb.NewMeterHistoryClient(historyConn),
		infoClient: meterpb.NewMeterInfoClient(infoConn),
		Meters:     []string{"test-meter"},
	}
}

func Test_EnergyJob_Do(t *testing.T) {
	tests := []struct {
		name    string
		unit    string
		native  float32 // consumption in the meter's native unit
		wantKWh float32 // expected reported value in kWh
	}{
		{"1000 Wh -> 1 kWh", "wh", 1000, 1},
		{"1 kWh -> 1 kWh", "kwh", 1, 1},
		{"1 MWh -> 1000 kWh", "mwh", 1, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := newEnergyTestJob(tt.unit, tt.native)

			var captured []byte
			err := job.Do(context.Background(), func(_ context.Context, _ string, body []byte) error {
				captured = body
				return nil
			})
			require.NoError(t, err)
			require.NotNil(t, captured, "sendFn should have been called")

			var result types.EnergyConsumption
			require.NoError(t, json.Unmarshal(captured, &result))
			assert.Equal(t, tt.wantKWh, result.TodaysEnergyConsumption.Value)
			assert.Equal(t, "kWh", result.TodaysEnergyConsumption.Units)
		})
	}
}
