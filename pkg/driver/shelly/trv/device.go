package trv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/shelly/trv/config"
	"github.com/smart-core-os/sc-bos/pkg/util/concurrent"
)

const DefaultPollInterval = 30 * time.Second

type TRV struct {
	name   string
	logger *zap.Logger

	pollInterval time.Duration
	address      string
	username     string
	password     string

	airTemperatureServer *airTemperatureServer

	Data *concurrent.Value[ThermostatData]
}

func NewTRV(conf config.TRVConfig, logger *zap.Logger) (*TRV, error) {
	if conf.PollInterval == 0 {
		conf.PollInterval = DefaultPollInterval
	}

	trv := &TRV{
		name:   conf.Name,
		logger: logger,

		pollInterval: conf.PollInterval,
		address:      conf.Address,
		username:     conf.Username,
		password:     conf.Password,

		Data: concurrent.NewValue(ThermostatData{}),
	}

	trv.airTemperatureServer = &airTemperatureServer{trv: trv}

	go trv.Poll(context.Background())

	return trv, nil
}

func (t *TRV) Poll(ctx context.Context) error {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			_, err := t.Refresh(ctx)
			if err != nil {
				t.logger.Error("error refreshing thermostat data", zap.Error(err))
			}
		}
	}
}

// API calls
func (t *TRV) Refresh(ctx context.Context) (data ThermostatData, err error) {
	result, err := t.request(ctx, "thermostat/0", nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(result, &data)
	if err != nil {
		return
	}

	_, ok := t.Data.Set(ctx, data)
	if !ok {
		err = ctx.Err()
		return
	}
	return
}

func (t *TRV) SetTargetTemperature(ctx context.Context, temperature float64) error {
	if temperature < 4 || temperature > 31 {
		return errors.New("target temperature out of range")
	}

	_, err := t.request(ctx, "settings/thermostat/0", map[string]string{
		"target_t_enabled": "1",
		"target_t":         fmt.Sprint(temperature),
	})
	if err != nil {
		return err
	}

	// now update stuff with the new value
	_, err = t.Refresh(ctx)
	return err
}

func (t *TRV) ClearTargetTemperature(ctx context.Context) error {
	_, err := t.request(ctx, "settings/thermostat/0", map[string]string{
		"target_t_enabled": "0",
	})
	if err != nil {
		return err
	}

	// now update stuff with the new value
	_, err = t.Refresh(ctx)
	return err
}

func (t *TRV) request(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	reqUrl := fmt.Sprintf("http://%s/%s", t.address, endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	// setup parameters
	values := make(url.Values)
	for k, v := range params {
		values.Set(k, v)
	}
	req.URL.RawQuery = values.Encode()

	// setup http basic auth
	if t.username != "" || t.password != "" {
		req.SetBasicAuth(t.username, t.password)
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return body, fmt.Errorf("HTTP Error %d: %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	return body, nil
}

type ThermostatData struct {
	Position          float64 `json:"pos"`
	TargetTemperature struct {
		Enabled bool    `json:"enabled"`
		Value   float64 `json:"value"`
		Units   string  `json:"units"`
	} `json:"target_t"`
	Temperature struct {
		Value   float64 `json:"value"`
		Units   string  `json:"units"`
		IsValid bool    `json:"is_valid"`
	} `json:"tmp"`
	Schedule        bool `json:"schedule"`
	ScheduleProfile int  `json:"schedule_profile"`
}
