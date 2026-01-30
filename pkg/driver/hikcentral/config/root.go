package config

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	driver.BaseConfig

	API      *API      `json:"api,omitempty"`
	Settings *Settings `json:"settings,omitempty"`

	// Metadata applied to all cameras
	Metadata *traits.Metadata `json:"metadata,omitempty"`
	Cameras  []*Camera        `json:"cameras,omitempty"`
}

type API struct {
	Address    string              `json:"address,omitempty"`
	AppKey     string              `json:"appKey,omitempty"`
	Secret     string              `json:"secret,omitempty"`
	SecretFile string              `json:"secretFile,omitempty"`
	Timeout    *jsontypes.Duration `json:"timeout,omitempty"`
}

type Settings struct {
	InfoPoll      *jsontypes.Duration `json:"infoPoll,omitempty"`      // How often to poll for camera info updates. Defaults to 5 minutes
	OccupancyPoll *jsontypes.Duration `json:"occupancyPoll,omitempty"` // How often to poll for occupancy updates. Defaults to 1 minute
	EventsPoll    *jsontypes.Duration `json:"eventsPoll,omitempty"`    // How often to poll for events updates. Defaults to 30 seconds
	StreamPoll    *jsontypes.Duration `json:"streamPoll,omitempty"`    // How often to poll for stream updates. Defaults to 1 minute
}

type Camera struct {
	Name      string `json:"name,omitempty"`
	Topic     string `json:"topic,omitempty"`
	IndexCode string `json:"indexCode,omitempty"`
	// Metadata applied to this camera
	Metadata  *traits.Metadata `json:"metadata,omitempty"`
	IpAddress string           `json:"ipAddress,omitempty"`
}

func ReadBytes(raw []byte) (dst Root, err error) {
	dst = Root{}
	err = json.Unmarshal(raw, &dst)
	if err != nil {
		return dst, err
	}
	if dst.API.SecretFile != "" {
		dst.API.Secret, err = readSecret(dst.API.SecretFile)
		if err != nil {
			return dst, err
		}
	}
	if dst.API.Timeout == nil {
		dst.API.Timeout = &jsontypes.Duration{Duration: 5 * time.Second}
	}
	if dst.Settings == nil {
		dst.Settings = &Settings{
			InfoPoll:      &jsontypes.Duration{Duration: 5 * time.Minute},
			OccupancyPoll: &jsontypes.Duration{Duration: 1 * time.Minute},
			EventsPoll:    &jsontypes.Duration{Duration: 30 * time.Second},
			StreamPoll:    &jsontypes.Duration{Duration: 1 * time.Minute},
		}
	}

	return
}

func readSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	str := string(raw)
	str = strings.TrimSpace(str)
	return str, nil
}
