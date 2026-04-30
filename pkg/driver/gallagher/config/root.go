package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const PointsEventTopicSuffix = "/event/pointset"

type ScDevice struct {
	Meta   *metadatapb.Metadata `json:"meta,omitempty"`
	ScName string               `json:"scName,omitempty"`
}

type Root struct {
	driver.BaseConfig
	HTTP           *HTTP  `json:"http,omitempty"`
	ScNamePrefix   string `json:"scNamePrefix,omitempty"`
	CaPath         string `json:"caPath,omitempty"`
	ClientCertPath string `json:"clientCertPath,omitempty"`
	ClientKeyPath  string `json:"clientKeyPath,omitempty"`
	// poll the cardholders api for updates on this schedule, defaults to once per minute
	RefreshCardholders *jsontypes.Schedule `json:"refreshCardholders,omitempty"`
	// poll the alerts API for updates on this schedule, defaults to once per minute
	RefreshAlarms *jsontypes.Schedule `json:"refreshAlerts,omitempty"`
	// poll the doors on this schedule, defaults to once per day
	RefreshDoors *jsontypes.Schedule `json:"refreshDoors,omitempty"`
	// poll the access zones API for updates on this schedule, defaults to once per minute
	RefreshAccessZones *jsontypes.Schedule `json:"refreshAccessZones,omitempty"`
	UdmiExportInterval jsontypes.Duration  `json:"udmiExportInterval"`
	TopicPrefix        string              `json:"topicPrefix,omitempty"`

	RefreshOccupancyInterval *jsontypes.Duration `json:"refreshOccupancyInterval,omitempty"`

	// number of security events to store, defaults to 200 if not set
	NumSecurityEvents     int  `json:"numSecurityEvents,omitempty"`
	OccupancyCountEnabled bool `json:"occupancyCountEnabled,omitempty"`
}

type HTTP struct {
	// BaseURL is the base URL of the Gallagher API, e.g. https://gallagher.example.com/api
	// must include the /api suffix
	BaseURL    string `json:"baseUrl,omitempty"`
	ApiKeyFile string `json:"apiKeyFile,omitempty"`
	ApiKey     string `json:"-"`
}

func ReadBytes(data []byte) (Root, error) {
	var cfg Root
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.HTTP == nil {
		return cfg, fmt.Errorf("HTTP config is not set")
	}

	if cfg.HTTP.BaseURL == "" {
		return cfg, fmt.Errorf("BaseURL is not set")
	}

	bytes, err := os.ReadFile(cfg.HTTP.ApiKeyFile)
	if err != nil {
		return cfg, fmt.Errorf("error reading api key file: %w", err)
	}
	cfg.HTTP.ApiKey = string(bytes)

	cfg.ApplyDefaults()
	return cfg, nil
}

func (cfg *Root) ApplyDefaults() {

	if cfg.RefreshCardholders == nil {
		cfg.RefreshCardholders = jsontypes.MustParseSchedule("* * * * *")
	}

	if cfg.UdmiExportInterval.Duration == 0 {
		cfg.UdmiExportInterval.Duration = 5 * time.Second
	}

	if cfg.RefreshDoors == nil {
		cfg.RefreshDoors = jsontypes.MustParseSchedule("0 0 * * *")
	}

	if cfg.RefreshAccessZones == nil {
		cfg.RefreshAccessZones = jsontypes.MustParseSchedule("* * * * *")
	}

	if cfg.RefreshAlarms == nil {
		cfg.RefreshAlarms = jsontypes.MustParseSchedule("* * * * *")
	}

	if cfg.NumSecurityEvents == 0 {
		cfg.NumSecurityEvents = 200
	}
}
