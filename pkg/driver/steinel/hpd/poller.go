package hpd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"go.uber.org/zap"
)

var errSensorMissing = errors.New("sensor returned no data")

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

	// processing is a lock, held by sending to it, that serialises process, which
	// runs from the poll loop and from UdmiServiceServer.GetExportMessage.
	// Concurrent writes to a sensor's value can fail with Aborted, which process would
	// report as a driver fault. It's a channel so a caller can give up waiting when its ctx is
	// done, rather than queue behind a stalled read.
	processing chan struct{}
}

func newPoller(client *Client, pollInterval time.Duration, logger *zap.Logger, fc *healthpb.FaultCheck, sensors ...sensor) *poller {
	return &poller{
		client:       client,
		pollInterval: pollInterval,
		logger:       logger,
		sensors:      sensors,
		faultCheck:   fc,
		processing:   make(chan struct{}, 1),
	}
}

func (p *poller) startPoll(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	_ = p.process(ctx) // failures are logged and reported to the fault check

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = p.process(ctx)
		}
	}
}

// process reads the sensor once, updating each sensor's value and the fault check.
// It returns an error if the read didn't refresh every sensor, so callers that need
// the values to be current, like UdmiServiceServer.GetExportMessage, can tell.
func (p *poller) process(ctx context.Context) error {
	select {
	case p.processing <- struct{}{}:
		defer func() { <-p.processing }()
	case <-ctx.Done():
		return ctx.Err()
	}

	response := SensorResponse{}
	if err := doGetRequest(ctx, p.client, &response, "sensor"); err != nil {
		if ctx.Err() != nil {
			return ctx.Err() // we're stopping, not a device fault
		}
		h := noResponse
		var (
			typeErr   *json.UnmarshalTypeError
			syntaxErr *json.SyntaxError
		)
		if errors.As(err, &typeErr) || errors.As(err, &syntaxErr) {
			// the device responded, we just couldn't make sense of the body
			h = badResponse
		}
		p.faultCheck.UpdateReliability(ctx, h)
		p.logger.Error("failed to GET sensor", zap.Error(err))
		return fmt.Errorf("get sensor: %w", err)
	}

	if response.SensorName == "" {
		// The base can answer 200 with an empty body ({}) when it is connected but the
		// sensor module itself is missing. Comms are fine, but the response carries no
		// sensor data, so treat it as a bad response rather than healthy.
		p.faultCheck.UpdateReliability(ctx, sensorMissing)
		p.logger.Error("sensor returned no data, the sensor module may be missing or disconnected")
		return errSensorMissing
	}

	var errs []error
	for _, sensor := range p.sensors {
		if err := sensor.GetUpdate(&response); err != nil {
			// we should not get here, this is a driver issue if it happens
			p.faultCheck.SetFault(driverError)
			errs = append(errs, fmt.Errorf("%s: %w", sensor.GetName(), err))
			p.logger.Error("sensor failed refreshing data", zap.String("sensor", sensor.GetName()), zap.Error(err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	p.faultCheck.ClearFaults()
	return nil
}
