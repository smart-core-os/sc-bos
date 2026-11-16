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

func newWaterTestJob(info meterpb.MeterInfoServer, nativeConsumption float32) *WaterJob {
	boundary := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	historyConn := wrap.ServerToClient(meterpb.MeterHistory_ServiceDesc, &testMeterHistoryServer{
		boundary:    boundary,
		latestUsage: nativeConsumption,
	})
	infoConn := wrap.ServerToClient(meterpb.MeterInfo_ServiceDesc, info)
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
		infoErr    error
		native     float32 // consumption in the meter's native unit
		wantLitres int32   // expected reported value in litres
		wantSkip   bool    // the meter can't be converted, so nothing is sent
	}{
		{name: "1 m3 -> 1000 litres", unit: "m3", native: 1, wantLitres: 1000},
		{name: "m³", unit: "m³", native: 2, wantLitres: 2000},
		{name: "cubic_meters", unit: "cubic_meters", native: 3, wantLitres: 3000},
		{name: "2000 cm3 -> 2 litres", unit: "cm3", native: 2000, wantLitres: 2},
		{name: "cm³", unit: "cm³", native: 4000, wantLitres: 4},
		{name: "5 litres -> 5 litres", unit: "litres", native: 5, wantLitres: 5},
		{name: "L", unit: "L", native: 6, wantLitres: 6},
		{name: "liters with spaces", unit: " Liters ", native: 7, wantLitres: 7},
		{name: "empty unit skipped", unit: "", native: 5, wantSkip: true},
		{name: "energy unit skipped", unit: "kWh", native: 5, wantSkip: true},
		{name: "unknown unit skipped", unit: "gal (US)", native: 5, wantSkip: true},
		{name: "describe error skipped", infoErr: status.Error(codes.Unavailable, "meter offline"), native: 5, wantSkip: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := newWaterTestJob(&testMeterInfoServer{usageUnit: tt.unit, err: tt.infoErr}, tt.native)

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

			var result types.WaterConsumption
			require.NoError(t, json.Unmarshal(captured, &result))
			assert.Equal(t, tt.wantLitres, result.TodaysWaterConsumption.Value)
			assert.Equal(t, "litres", result.TodaysWaterConsumption.Units)
		})
	}
}
