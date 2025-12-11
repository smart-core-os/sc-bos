package hpd

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
)

type sensor interface {
	GetUpdate(response *SensorResponse) error
	GetName() string
}

type poller struct {
	client       *Client
	pollInterval time.Duration
	logger       *zap.Logger
	sensors      []sensor
	faultCheck   *healthpb.FaultCheck
}

func newPoller(client *Client, pollInterval time.Duration, logger *zap.Logger, fc *healthpb.FaultCheck, sensors ...sensor) *poller {
	return &poller{
		client:       client,
		pollInterval: pollInterval,
		logger:       logger,
		sensors:      sensors,
		faultCheck:   fc,
	}
}

func (p *poller) startPoll(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	p.process(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.process(ctx)
		}
	}
}

func (p *poller) process(ctx context.Context) {
	response := SensorResponse{}
	if err := doGetRequest(p.client, &response, "sensor"); err != nil {
		h := &gen.HealthCheck_Reliability{}
		var unsupportedTypeErr *json.UnmarshalTypeError
		if errors.Is(err, unsupportedTypeErr) {
			h = noResponse
		} else {
			h = badResponse
		}
		p.faultCheck.UpdateReliability(ctx, h)
		p.logger.Error("failed to GET sensor", zap.Error(err))
		return
	}

	fault := false
	for _, sensor := range p.sensors {
		if err := sensor.GetUpdate(&response); err != nil {
			// we should not get here, this is a driver issue if it happens
			p.faultCheck.SetFault(driverError)
			fault = true
			p.logger.Error("sensor failed refreshing data", zap.String("sensor", sensor.GetName()), zap.Error(err))
		}
	}

	if !fault {
		p.faultCheck.ClearFaults()
	}
}
