package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	auto.Config

	// The devices to monitor and reset
	Devices []string `json:"devices"`
	// Core configuration for the automation.
	// When the monitored devices leave the state described by Normal,
	// the automation will wait for ResetDelay before resetting the device to ResetState.
	Normal     StateRange          `json:"normal,omitempty,omitzero"`     // Defaults to {Max: ResetState}
	ResetDelay *jsontypes.Duration `json:"resetDelay,omitempty"`          // Defaults to 1 hour
	ResetState float32             `json:"resetState,omitempty,omitzero"` // Defaults to 0
	TimerStart TimerStart          `json:"timerStart,omitzero"`           // Defaults to "change"
}

type StateRange struct {
	Min *float32 `json:"min,omitempty"`
	Max *float32 `json:"max,omitempty"`
}

type TimerStart string

const (
	TimerStartAfterEnter  TimerStart = "enter"
	TimerStartAfterChange TimerStart = "change"
)

func ReadBytes(data []byte) (Root, error) {
	var cfg Root
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

const (
	DefaultResetDelay = time.Hour
	DefaultTimerStart = TimerStartAfterChange
)

func (cfg *Root) SetDefaults() {
	if cfg.ResetDelay == nil {
		cfg.ResetDelay = &jsontypes.Duration{Duration: DefaultResetDelay}
	}
	if cfg.Normal == (StateRange{}) {
		cfg.Normal = StateRange{Max: &cfg.ResetState}
	}
	if cfg.TimerStart == "" {
		cfg.TimerStart = DefaultTimerStart
	}
}

func (cfg *Root) Validate() error {
	checkPercent := func(n string, v float32) error {
		if v < 0 || v > 100 {
			return fmt.Errorf("%s must be between 0 and 100", n)
		}
		return nil
	}
	if err := checkPercent("resetState", cfg.ResetState); err != nil {
		return err
	}
	var minNorm, maxNorm float32
	if cfg.Normal.Min != nil {
		minNorm = *cfg.Normal.Min
	}
	if cfg.Normal.Max != nil {
		maxNorm = *cfg.Normal.Max
	}
	if err := checkPercent("normal.min", minNorm); err != nil {
		return err
	}
	if err := checkPercent("normal.max", maxNorm); err != nil {
		return err
	}
	if minNorm > maxNorm {
		return errors.New("normal.min must be less than or equal to normal.max")
	}
	if cfg.ResetState < minNorm || cfg.ResetState > maxNorm {
		return errors.New("resetState must be outside of normal range")
	}
	return nil
}
