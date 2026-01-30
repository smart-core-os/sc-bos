package config

import (
	"encoding/json"

	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	Meters        []string            `json:"meters,omitempty"`
	MeterGroups   map[string][]string `json:"meterGroups,omitempty"`
	HistoryBackup *HistoryBackup      `json:"HistoryBackup,omitempty"`
}

type HistoryBackup struct {
	// Disabled whether to disable using the latest related row from history as a backup when the source read fails.
	Disabled bool `json:"disabled,omitempty"`
	// LookbackLimit is the maximum age of history records to consider as a backup source.
	// If not specified, there is no limit.
	LookbackLimit *jsontypes.Duration `json:"lookbackLimit,omitempty"`
	// PercentageOfAcceptableErrors is the percentage of read errors from meters in the zone
	// that are acceptable before disabling the use of history as a backup.
	// For example, if set to 5.0, then if more than 5% of the meters in the zone return read errors,
	// the latest related history record will not be used as a backup.
	// If zero, any read error will disable the use of history as a backup.
	// Acceptable values are between 0.0 and 100.0.
	// If value is not acceptable, it defaults to 0.0.
	PercentageOfAcceptableErrors float32 `json:"percentageOfAcceptableErrors,omitempty"`
}

func ParseConfig(b []byte) (Root, error) {
	var cfg Root
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}

	if cfg.HistoryBackup != nil && (cfg.HistoryBackup.PercentageOfAcceptableErrors < 0 || cfg.HistoryBackup.PercentageOfAcceptableErrors > 100) {
		cfg.HistoryBackup.PercentageOfAcceptableErrors = 0
	}
	return cfg, nil
}
