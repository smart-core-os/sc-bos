package sccexporter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/airqualitysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/trait/airtemperaturepb"
	"github.com/smart-core-os/sc-golang/pkg/trait/metadatapb"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func TestMetadata(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("metadata")

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(root.ClientConn()),
		},
	}

	metadata := &traits.Metadata{
		Name: "foo",
		Appearance: &traits.Metadata_Appearance{
			Title:       "Foo Device",
			Description: "A device for testing metadata",
		},
		Location: &traits.Metadata_Location{
			Floor: "1",
			Zone:  "bar",
		},
	}

	metaModel := metadatapb.NewModel(resource.WithInitialValue(metadata))
	modelServer := metadatapb.NewModelServer(metaModel)
	metaClient := node.WithClients(metadatapb.WrapApi(modelServer))
	root.Announce("foo", node.HasTrait(trait.Metadata, metaClient))

	sccexporter.initialiseClients(root)

	dev := newDevice("foo", logger, metadata)

	messagesCh := make(chan message, 1)

	agent := "test-agent"

	sccexporter.fetchAndPublishDeviceData(context.Background(), dev, agent, messagesCh, true, 30*time.Second)

	require.Len(t, messagesCh, 1)

	msg := <-messagesCh

	require.Equal(t, agent, msg.Agent)
	require.Equal(t, "foo", msg.Device.Name)
	require.NotEmpty(t, msg.Device.Data)
	require.Contains(t, msg.Device.Data, trait.Metadata)
	require.Contains(t, msg.Device.Data[trait.Metadata], "metadata")

	var receivedMetadata traits.Metadata
	err = protojson.Unmarshal(msg.Device.Data[trait.Metadata]["metadata"], &receivedMetadata)
	require.NoError(t, err)

	require.Equal(t, "foo", receivedMetadata.Name)
	require.Equal(t, "Foo Device", receivedMetadata.Appearance.Title)
	require.Equal(t, "A device for testing metadata", receivedMetadata.Appearance.Description)
	require.Equal(t, "1", receivedMetadata.Location.Floor)
	require.Equal(t, "bar", receivedMetadata.Location.Zone)
}

func TestFetchAndPublishDeviceData(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	ctx := context.Background()
	agent := "test-agent"

	t.Run("single trait with data", func(t *testing.T) {
		dev := newDevice("test-device", logger, nil)

		testDataJSON := `{
			"value1": 42.5,
			"value2": "test-value"
		}`

		dev.traits[trait.Name("test-trait")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return map[string]json.RawMessage{
				"testResource": json.RawMessage(testDataJSON),
			}, nil
		}

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 30*time.Second)

		require.Len(t, messagesCh, 1)
		msg := <-messagesCh

		require.Equal(t, agent, msg.Agent)
		require.Equal(t, "test-device", msg.Device.Name)
		require.NotEmpty(t, msg.Device.Data)
		require.Contains(t, msg.Device.Data, trait.Name("test-trait"))
		require.Contains(t, msg.Device.Data[trait.Name("test-trait")], "testResource")

		var data map[string]any
		err = json.Unmarshal(msg.Device.Data[trait.Name("test-trait")]["testResource"], &data)
		require.NoError(t, err)
		require.Equal(t, 42.5, data["value1"])
		require.Equal(t, "test-value", data["value2"])
	})

	t.Run("multiple traits with data", func(t *testing.T) {
		dev := newDevice("multi-trait-device", logger, nil)

		trait1JSON := `{
			"measurement": 100.0,
			"status": "active"
		}`
		dev.traits[trait.Name("trait1")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return map[string]json.RawMessage{
				"resource1": json.RawMessage(trait1JSON),
			}, nil
		}

		trait2JSON := `{
			"temperature": 22.5,
			"humidity": 45.0
		}`
		dev.traits[trait.Name("trait2")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return map[string]json.RawMessage{
				"resource2": json.RawMessage(trait2JSON),
			}, nil
		}

		trait3JSON := `{
			"value": 42,
			"unit": "kWh"
		}`
		dev.traits[trait.Name("trait3")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return map[string]json.RawMessage{
				"resource3": json.RawMessage(trait3JSON),
			}, nil
		}

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 30*time.Second)

		require.Len(t, messagesCh, 1)
		msg := <-messagesCh

		require.Len(t, msg.Device.Data, 3)
		require.Contains(t, msg.Device.Data, trait.Name("trait1"))
		require.Contains(t, msg.Device.Data, trait.Name("trait2"))
		require.Contains(t, msg.Device.Data, trait.Name("trait3"))

		var data1 map[string]any
		err = json.Unmarshal(msg.Device.Data[trait.Name("trait1")]["resource1"], &data1)
		require.NoError(t, err)
		require.Equal(t, float64(100.0), data1["measurement"])
		require.Equal(t, "active", data1["status"])

		var data2 map[string]any
		err = json.Unmarshal(msg.Device.Data[trait.Name("trait2")]["resource2"], &data2)
		require.NoError(t, err)
		require.Equal(t, float64(22.5), data2["temperature"])
		require.Equal(t, float64(45.0), data2["humidity"])

		var data3 map[string]any
		err = json.Unmarshal(msg.Device.Data[trait.Name("trait3")]["resource3"], &data3)
		require.NoError(t, err)
		require.Equal(t, float64(42), data3["value"])
		require.Equal(t, "kWh", data3["unit"])
	})

	t.Run("trait fetcher returns error", func(t *testing.T) {
		dev := newDevice("error-device", logger, nil)
		dev.traits[trait.Name("failing-trait")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return nil, context.DeadlineExceeded
		}

		workingDataJSON := `{
			"value": 123
		}`

		dev.traits[trait.Name("working-trait")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return map[string]json.RawMessage{
				"workingResource": json.RawMessage(workingDataJSON),
			}, nil
		}

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 30*time.Second)

		require.Len(t, messagesCh, 1)
		msg := <-messagesCh

		require.NotEmpty(t, msg.Device.Data)
		require.Contains(t, msg.Device.Data, trait.Name("working-trait"))

		var data map[string]any
		err = json.Unmarshal(msg.Device.Data[trait.Name("working-trait")]["workingResource"], &data)
		require.NoError(t, err)
		require.Equal(t, float64(123), data["value"])
	})

	t.Run("all traits fail - no message sent", func(t *testing.T) {
		dev := newDevice("all-fail-device", logger, nil)
		dev.traits[trait.Name("trait1")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return nil, context.DeadlineExceeded
		}
		dev.traits[trait.Name("trait2")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			return nil, context.Canceled
		}

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 30*time.Second)

		require.Len(t, messagesCh, 0)
	})

	t.Run("device with no traits", func(t *testing.T) {
		dev := newDevice("empty-device", logger, nil)

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 30*time.Second)

		require.Len(t, messagesCh, 0)
	})

	t.Run("timeout on slow device", func(t *testing.T) {
		dev := newDevice("slow-device", logger, nil)
		dev.traits[trait.Name("slow-trait")] = func(ctx context.Context) (map[string]json.RawMessage, error) {
			select {
			case <-time.After(2 * time.Second):
				return map[string]json.RawMessage{
					"slowResource": json.RawMessage(`{"value": "should-not-see-this"}`),
				}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		messagesCh := make(chan message, 1)

		a := &AutoImpl{
			Services: auto.Services{
				Logger: logger,
			},
		}

		a.fetchAndPublishDeviceData(ctx, dev, agent, messagesCh, false, 100*time.Millisecond)

		require.Len(t, messagesCh, 0)
	})
}

func TestGetMeterDeviceAndData(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("metadata")

	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	meterReading := &gen.MeterReading{
		Usage:     123.45,
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		Produced:  67.89,
	}

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel(resource.WithInitialValue(meterReading))
	modelServer := meterpb.NewModelServer(meterModel)
	meterClient := node.WithClients(gen.WrapMeterApi(modelServer))
	root.Announce("foo",
		node.HasTrait(meterpb.TraitName, meterClient),
		node.HasServices(root.ClientConn(), gen.DevicesApi_ServiceDesc),
	)

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		},
	}
	sccexporter.initialiseClients(root)

	allDevices := make(map[string]*device)
	err = sccexporter.getAllTraitImplementors(context.Background(), meterpb.TraitName, allDevices)
	require.NoError(t, err)

	require.Len(t, allDevices, 1)
	dev, exists := allDevices["foo"]
	require.True(t, exists)
	require.Equal(t, "foo", dev.name)

	res := allDevices["foo"].traits
	require.Len(t, res, 1)
	traitData, err := res[meterpb.TraitName](context.Background())
	require.NoError(t, err)

	// Verify the structure contains meterReading resource
	require.Contains(t, traitData, "meterReading")

	reading := gen.MeterReading{}
	err = protojson.Unmarshal(traitData["meterReading"], &reading)
	require.NoError(t, err)

	require.Equal(t, meterReading.Usage, reading.Usage)
	require.Equal(t, meterReading.Produced, reading.Produced)
	require.Equal(t, meterReading.StartTime.AsTime(), reading.StartTime.AsTime())
	require.Equal(t, meterReading.EndTime.AsTime(), reading.EndTime.AsTime())
}

func TestGetMeterDeviceAndDataWithInfo(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("meter")

	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	meterReading := &gen.MeterReading{
		Usage:     123.45,
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		Produced:  67.89,
	}

	meterInfo := &gen.MeterReadingSupport{
		UsageUnit:    "kWh",
		ProducedUnit: "kWh",
	}

	devicesApi := devices.NewServer(root)
	meterModel := meterpb.NewModel(resource.WithInitialValue(meterReading))
	modelServer := meterpb.NewModelServer(meterModel)
	infoServer := &meterpb.InfoServer{MeterReading: meterInfo}
	meterClient := node.WithClients(
		gen.WrapMeterApi(modelServer),
		gen.WrapMeterInfo(infoServer),
	)
	root.Announce("foo",
		node.HasTrait(meterpb.TraitName, meterClient),
		node.HasServices(root.ClientConn(), gen.DevicesApi_ServiceDesc),
	)

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		},
	}
	sccexporter.initialiseClients(root)

	allDevices := make(map[string]*device)
	err = sccexporter.getAllTraitImplementors(context.Background(), meterpb.TraitName, allDevices)
	require.NoError(t, err)

	require.Len(t, allDevices, 1)
	dev, exists := allDevices["foo"]
	require.True(t, exists)
	require.Equal(t, "foo", dev.name)

	sccexporter.getMeterInfo(context.Background(), meterpb.TraitName, allDevices)

	require.NotNil(t, dev.info[meterpb.TraitName])
	support, ok := dev.info[meterpb.TraitName].(*gen.MeterReadingSupport)
	require.True(t, ok)
	require.Equal(t, "kWh", support.UsageUnit)
	require.Equal(t, "kWh", support.ProducedUnit)

	res := allDevices["foo"].traits
	require.Len(t, res, 1)
	traitData, err := res[meterpb.TraitName](context.Background())
	require.NoError(t, err)

	// Verify the structure contains both meterReading and meterReadingInfo
	require.Contains(t, traitData, "meterReading")
	require.Contains(t, traitData, "meterReadingInfo")

	// Verify meter reading data
	var reading gen.MeterReading
	err = protojson.Unmarshal(traitData["meterReading"], &reading)
	require.NoError(t, err)
	require.Equal(t, meterReading.Usage, reading.Usage)
	require.Equal(t, meterReading.Produced, reading.Produced)

	// Verify meter reading info
	var info gen.MeterReadingSupport
	err = protojson.Unmarshal(traitData["meterReadingInfo"], &info)
	require.NoError(t, err)
	require.Equal(t, "kWh", info.UsageUnit)
	require.Equal(t, "kWh", info.ProducedUnit)
}

func TestGetAirQualityDeviceAndData(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("airquality")

	co2Level := float32(450.5)
	score := float32(75.5)

	airQuality := &traits.AirQuality{
		CarbonDioxideLevel: &co2Level,
		Score:              &score,
	}

	devicesApi := devices.NewServer(root)
	airQualityModel := airqualitysensorpb.NewModel()
	_, err = airQualityModel.UpdateAirQuality(airQuality)
	require.NoError(t, err)
	modelServer := airqualitysensorpb.NewModelServer(airQualityModel)
	airQualityClient := node.WithClients(airqualitysensorpb.WrapApi(modelServer))
	root.Announce("foo",
		node.HasTrait(trait.AirQualitySensor, airQualityClient),
		node.HasServices(root.ClientConn(), gen.DevicesApi_ServiceDesc),
	)

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		},
	}
	sccexporter.initialiseClients(root)

	allDevices := make(map[string]*device)
	err = sccexporter.getAllTraitImplementors(context.Background(), trait.AirQualitySensor, allDevices)
	require.NoError(t, err)

	require.Len(t, allDevices, 1)
	dev, exists := allDevices["foo"]
	require.True(t, exists)
	require.Equal(t, "foo", dev.name)

	res := allDevices["foo"].traits
	require.Len(t, res, 1)
	traitData, err := res[trait.AirQualitySensor](context.Background())
	require.NoError(t, err)

	// Verify the structure contains airQuality resource
	require.Contains(t, traitData, "airQuality")

	receivedAirQuality := traits.AirQuality{}
	err = protojson.Unmarshal(traitData["airQuality"], &receivedAirQuality)
	require.NoError(t, err)

	require.Equal(t, *airQuality.CarbonDioxideLevel, *receivedAirQuality.CarbonDioxideLevel)
	require.Equal(t, *airQuality.Score, *receivedAirQuality.Score)
}

func TestGetAirTemperatureDeviceAndData(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("airtemperature")

	celsius := 22.5

	airTemperature := &traits.AirTemperature{
		AmbientTemperature: &types.Temperature{ValueCelsius: celsius},
	}

	devicesApi := devices.NewServer(root)
	airTemperatureModel := airtemperaturepb.NewModel()
	_, err = airTemperatureModel.UpdateAirTemperature(airTemperature)
	require.NoError(t, err)
	modelServer := airtemperaturepb.NewModelServer(airTemperatureModel)
	airTemperatureClient := node.WithClients(airtemperaturepb.WrapApi(modelServer))
	root.Announce("foo",
		node.HasTrait(trait.AirTemperature, airTemperatureClient),
		node.HasServices(root.ClientConn(), gen.DevicesApi_ServiceDesc),
	)

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		},
	}
	sccexporter.initialiseClients(root)

	allDevices := make(map[string]*device)
	err = sccexporter.getAllTraitImplementors(context.Background(), trait.AirTemperature, allDevices)
	require.NoError(t, err)

	require.Len(t, allDevices, 1)
	dev, exists := allDevices["foo"]
	require.True(t, exists)
	require.Equal(t, "foo", dev.name)

	res := allDevices["foo"].traits
	require.Len(t, res, 1)
	traitData, err := res[trait.AirTemperature](context.Background())
	require.NoError(t, err)

	// Verify the structure contains airTemperature resource
	require.Contains(t, traitData, "airTemperature")

	receivedAirTemperature := traits.AirTemperature{}
	err = protojson.Unmarshal(traitData["airTemperature"], &receivedAirTemperature)
	require.NoError(t, err)

	require.Equal(t, airTemperature.AmbientTemperature.ValueCelsius, receivedAirTemperature.AmbientTemperature.ValueCelsius)
}

func TestGetOccupancyDeviceAndData(t *testing.T) {

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	root := node.New("occupancy")

	stateChangeTime := time.Now().Add(-5 * time.Minute)

	occupancy := &traits.Occupancy{
		State:           traits.Occupancy_OCCUPIED,
		PeopleCount:     5,
		StateChangeTime: timestamppb.New(stateChangeTime),
	}

	devicesApi := devices.NewServer(root)
	occupancyModel := occupancysensorpb.NewModel()
	_, err = occupancyModel.SetOccupancy(occupancy)
	require.NoError(t, err)
	modelServer := occupancysensorpb.NewModelServer(occupancyModel)
	occupancyClient := node.WithClients(occupancysensorpb.WrapApi(modelServer))
	root.Announce("foo",
		node.HasTrait(trait.OccupancySensor, occupancyClient),
		node.HasServices(root.ClientConn(), gen.DevicesApi_ServiceDesc),
	)

	sccexporter := &AutoImpl{
		Services: auto.Services{
			Logger:  logger,
			Node:    root,
			Devices: gen.NewDevicesApiClient(wrap.ServerToClient(gen.DevicesApi_ServiceDesc, devicesApi)),
		},
	}
	sccexporter.initialiseClients(root)

	allDevices := make(map[string]*device)
	err = sccexporter.getAllTraitImplementors(context.Background(), trait.OccupancySensor, allDevices)
	require.NoError(t, err)

	require.Len(t, allDevices, 1)
	dev, exists := allDevices["foo"]
	require.True(t, exists)
	require.Equal(t, "foo", dev.name)

	res := allDevices["foo"].traits
	require.Len(t, res, 1)
	traitData, err := res[trait.OccupancySensor](context.Background())
	require.NoError(t, err)

	// Verify the structure contains occupancy resource
	require.Contains(t, traitData, "occupancy")

	receivedOccupancy := traits.Occupancy{}
	err = protojson.Unmarshal(traitData["occupancy"], &receivedOccupancy)
	require.NoError(t, err)

	require.Equal(t, occupancy.State, receivedOccupancy.State)
	require.Equal(t, occupancy.PeopleCount, receivedOccupancy.PeopleCount)
	require.Equal(t, occupancy.StateChangeTime.AsTime(), receivedOccupancy.StateChangeTime.AsTime())
}
