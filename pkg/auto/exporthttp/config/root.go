package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

const AuthenticationBearer = "Bearer"

type Root struct {
	auto.Config

	BaseUrl string         `json:"baseUrl"`
	Site    string         `json:"site"`
	Auth    Authentication `json:"authentication"`
	Logs    bool           `json:"logs"`
	Sources struct {
		Occupancy   *Occupancy   `json:"occupancy,omitempty"`
		Temperature *Temperature `json:"temperature,omitempty"`
		Energy      *Energy      `json:"energy,omitempty"`
		AirQuality  *AirQuality  `json:"airQuality,omitempty"`
		Water       *Water       `json:"water,omitempty"`
	} `json:"sources"`
}

type Authentication struct {
	Type       string `json:"type"`
	SecretFile string `json:"secretFile"`
	Token      string `json:"-"`
}

type Source struct {
	Path     string              `json:"path"`
	Schedule *jsontypes.Schedule `json:"schedule"`
	Timeout  *jsontypes.Duration `json:"timeout,omitempty"`
}

type Occupancy struct {
	Source
	Sensors []string `json:"sensors"`
}

type Temperature struct {
	Source
	Sensors []string `json:"sensors"`
}

type Energy struct {
	Source
	Meters []string `json:"meters"`
}

type AirQuality struct {
	Source
	Sensors []string `json:"sensors"`
}

type Water struct {
	Source
	Meters []string `json:"meters"`
}

func ReadBytes(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return
	}

	if cfg.BaseUrl == "" {
		err = fmt.Errorf("baseUrl not specified")
		return
	}

	switch cfg.Auth.Type {
	case AuthenticationBearer:
		var tok []byte
		tok, err = os.ReadFile(cfg.Auth.SecretFile)
		if err != nil {
			return
		}
		cfg.Auth.Token = string(tok)
	default:
		err = fmt.Errorf("authentication type %s not yet supported", cfg.Auth.Type)
		return
	}

	return
}
