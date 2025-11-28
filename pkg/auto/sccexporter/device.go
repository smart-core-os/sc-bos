package sccexporter

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// DataFetcher is a function that fetches device data for a specific trait
// Returns a map of resource names to their JSON data (e.g., "meterReading" -> data, "meterReadingInfo" -> info)
type DataFetcher func(ctx context.Context) (map[string]json.RawMessage, error)

type device struct {
	name     string
	logger   *zap.Logger
	traits   map[trait.Name]DataFetcher
	info     map[trait.Name]proto.Message
	metaData *traits.Metadata
}

func newDevice(name string, logger *zap.Logger, metaData *traits.Metadata) *device {
	d := &device{
		name:     name,
		logger:   logger,
		traits:   make(map[trait.Name]DataFetcher),
		info:     make(map[trait.Name]proto.Message),
		metaData: metaData,
	}
	return d
}

func (d *device) getMeterData(ctx context.Context, meterClient gen.MeterApiClient) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)

	reading, err := meterClient.GetMeterReading(ctx, &gen.GetMeterReadingRequest{
		Name: d.name,
	})
	if err != nil {
		d.logger.Error("failed to get meter reading", zap.String("meter", d.name), zap.Error(err))
		return nil, err
	}

	// Marshal the meter reading to JSON using protojson
	readingBytes, err := protojson.Marshal(reading)
	if err != nil {
		d.logger.Error("failed to marshal meter reading", zap.String("meter", d.name), zap.Error(err))
		return nil, err
	}

	result["meterReading"] = readingBytes

	// Add meter reading info if available
	info, ok := d.info[meterpb.TraitName].(*gen.MeterReadingSupport)
	if ok && info != nil {
		infoBytes, err := protojson.Marshal(info)
		if err != nil {
			d.logger.Error("failed to marshal meter reading info", zap.String("meter", d.name), zap.Error(err))
		} else {
			result["meterReadingInfo"] = infoBytes
		}
	}

	return result, nil
}

func (d *device) getAirQualitySensorData(ctx context.Context, airQualityClient traits.AirQualitySensorApiClient) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)

	airQuality, err := airQualityClient.GetAirQuality(ctx, &traits.GetAirQualityRequest{
		Name: d.name,
	})
	if err != nil {
		d.logger.Error("failed to get air quality", zap.String("airQualitySensor", d.name), zap.Error(err))
		return nil, err
	}

	aq, err := protojson.Marshal(airQuality)
	if err != nil {
		d.logger.Error("failed to marshal air quality", zap.String("airQualitySensor", d.name), zap.Error(err))
		return nil, err
	}

	result["airQuality"] = aq
	return result, nil
}

func (d *device) getOccupancySensorData(ctx context.Context, occupancyClient traits.OccupancySensorApiClient) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)

	occupancy, err := occupancyClient.GetOccupancy(ctx, &traits.GetOccupancyRequest{
		Name: d.name,
	})
	if err != nil {
		d.logger.Error("failed to get occupancy", zap.String("occupancySensor", d.name), zap.Error(err))
		return nil, err
	}

	o, err := protojson.Marshal(occupancy)
	if err != nil {
		d.logger.Error("failed to marshal occupancy", zap.String("occupancySensor", d.name), zap.Error(err))
		return nil, err
	}

	result["occupancy"] = o
	return result, nil
}

func (d *device) getAirTemperatureData(ctx context.Context, client traits.AirTemperatureApiClient) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)

	airTemperature, err := client.GetAirTemperature(ctx, &traits.GetAirTemperatureRequest{
		Name: d.name,
	})
	if err != nil {
		d.logger.Error("failed to get air temperature", zap.String("airTemperatureSensor", d.name), zap.Error(err))
		return nil, err
	}

	at, err := protojson.Marshal(airTemperature)
	if err != nil {
		d.logger.Error("failed to marshal air temperature", zap.String("airTemperatureSensor", d.name), zap.Error(err))
		return nil, err
	}

	result["airTemperature"] = at
	return result, nil
}
