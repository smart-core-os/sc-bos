package merge

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

// test the emergencyImpl alarmConfig for various bounds & error input
func Test_emergency(t *testing.T) {
	impl := &emergencyImpl{}
	okLowerBound := 233.9999
	okUpperBound := 234.0001

	tests := []struct {
		name       string
		alarmConf  AlarmConfig
		checkFunc  func(any) (*traits.Emergency, error)
		pointValue any
		wantLevel  traits.Emergency_Level
		wantReason string
		wantErr    bool
	}{
		{
			name: "value below lower bound",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
				AlarmReason:  "Sensor is not reading the expected value",
			},
			pointValue: 233.9998,
			wantLevel:  traits.Emergency_EMERGENCY,
			wantReason: "Sensor is not reading the expected value",
			wantErr:    false,
		},
		{
			name: "value on lower bound",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
			},
			pointValue: 233.9999,
			wantLevel:  traits.Emergency_OK,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "value ~equal",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
			},
			pointValue: 234.0,
			wantLevel:  traits.Emergency_OK,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "value on upper bound",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
			},
			pointValue: 234.0001,
			wantLevel:  traits.Emergency_OK,
			wantReason: "",
			wantErr:    false,
		},
		{
			name: "value above upper bound",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
				AlarmReason:  "Sensor is not reading the expected value",
			},
			pointValue: 234.0002,
			wantLevel:  traits.Emergency_EMERGENCY,
			wantReason: "Sensor is not reading the expected value",
			wantErr:    false,
		},
		{
			name: "value is error",
			alarmConf: AlarmConfig{
				OkLowerBound: &okLowerBound,
				OkUpperBound: &okUpperBound,
				AlarmReason:  "",
			},
			pointValue: errors.New("error"),
			wantLevel:  traits.Emergency_EMERGENCY,
			wantReason: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impl.config.AlarmConfig = &tt.alarmConf
			emergency, err := impl.checkValueForEmergency(tt.pointValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if emergency == nil && err == nil {
				t.Errorf("expect emergency when no error")
				return
			}
			if err != nil {
				// we have checked the error conditions are met
				// don't bother continuing and checking the emergency as it will be nil
				return
			}
			if emergency.Level != tt.wantLevel {
				t.Errorf("%s: got = %v, want %v", tt.name, emergency.Level, tt.wantLevel)
			}
			if emergency.Reason != tt.wantReason {
				t.Errorf("%s: got = %v, want %v", tt.name, emergency.Reason, tt.wantReason)
			}
		})
	}
}

func Test_emergency_default_inf(t *testing.T) {

	emergencyCfg := emergencyConfig{
		AlarmConfig: &AlarmConfig{
			OkLowerBound: nil,
			OkUpperBound: nil,
		},
	}

	bytes, err := json.Marshal(emergencyCfg)
	if err != nil {
		t.Errorf("error marshalling emergencyCfg: %v", err)
	}

	cfg, err := readEmergencyConfig(bytes)
	if err != nil {
		t.Errorf("error reading config: %v", err)
	}

	if *cfg.AlarmConfig.OkLowerBound != math.Inf(-1) ||
		*cfg.AlarmConfig.OkUpperBound != math.Inf(1) {
		t.Errorf("Ok Lower or Upper Bound not set")
	}
}
