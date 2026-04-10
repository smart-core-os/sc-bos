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

func newWaterTestJob(usageUnit string, nativeConsumption float32) *WaterJob {
	boundary := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	historyConn := wrap.ServerToClient(meterpb.MeterHistory_ServiceDesc, &testMeterHistoryServer{
		boundary:    boundary,
		latestUsage: nativeConsumption,
	})
	infoConn := wrap.ServerToClient(meterpb.MeterInfo_ServiceDesc, &testMeterInfoServer{usageUnit: usageUnit})
	return &WaterJob{
		BaseJob:    BaseJob{Logger: zap.NewNop()},
		client:     meterpb.NewMeterHistoryClient(historyConn),
		infoClient: meterpb.NewMeterInfoClient(infoConn),
		Meters:     []string{"test-meter"},
	}
}

func Test_WaterJob_Do(t *testing.T) {
	tests := []struct {
		name       string
		unit       string
		native     float32 // consumption in the meter's native unit
		wantLitres int32   // expected reported value in litres
	}{
		{"1 m3 -> 1000 litres", "m3", 1, 1000},
		{"2000 cm3 -> 2 litres", "cm3", 2000, 2},
		{"5 litres -> 5 litres", "litres", 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := newWaterTestJob(tt.unit, tt.native)

			var captured []byte
			err := job.Do(context.Background(), func(_ context.Context, _ string, body []byte) error {
				captured = body
				return nil
			})
			require.NoError(t, err)
			require.NotNil(t, captured, "sendFn should have been called")

			var result types.WaterConsumption
			require.NoError(t, json.Unmarshal(captured, &result))
			assert.Equal(t, tt.wantLitres, result.TodaysWaterConsumption.Value)
			assert.Equal(t, "litres", result.TodaysWaterConsumption.Units)
		})
	}
}
