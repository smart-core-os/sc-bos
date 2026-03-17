package trv

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

type airTemperatureServer struct {
	traits.UnimplementedAirTemperatureApiServer

	trv *TRV
}

func (a *airTemperatureServer) GetAirTemperature(_ context.Context, _ *traits.GetAirTemperatureRequest) (*traits.AirTemperature, error) {
	data, _ := a.trv.Data.Get()
	airTemperature := DataToAirTemperature(data)
	return airTemperature, nil
}

func (a *airTemperatureServer) UpdateAirTemperature(ctx context.Context, request *traits.UpdateAirTemperatureRequest) (*traits.AirTemperature, error) {
	airTemperature := request.GetState()
	if setPoint := airTemperature.GetTemperatureSetPoint(); setPoint != nil {
		err := a.trv.SetTargetTemperature(ctx, setPoint.ValueCelsius)
		if err != nil {
			return nil, err
		}
	}

	data, err := a.trv.Refresh(ctx)
	if err != nil {
		return nil, err
	}
	return DataToAirTemperature(data), nil
}

func (a *airTemperatureServer) PullAirTemperature(request *traits.PullAirTemperatureRequest, server traits.AirTemperatureApi_PullAirTemperatureServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	send := func(data ThermostatData, t time.Time) error {
		airTemperature := DataToAirTemperature(data)
		change := &traits.PullAirTemperatureResponse_Change{
			Name:           request.GetName(),
			ChangeTime:     timestamppb.New(t),
			AirTemperature: airTemperature,
		}
		res := &traits.PullAirTemperatureResponse{
			Changes: []*traits.PullAirTemperatureResponse_Change{change},
		}
		return server.Send(res)
	}

	initial, changes := a.trv.Data.Changes(ctx, false, 1)
	err := send(initial, time.Now())
	if err != nil {
		return err
	}

	for change := range changes {
		err = send(change.New, change.Timestamp)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ traits.AirTemperatureApiServer = (*airTemperatureServer)(nil)

func DataToAirTemperature(data ThermostatData) *traits.AirTemperature {
	airTemperature := &traits.AirTemperature{}

	if data.Temperature.IsValid {
		airTemperature.AmbientTemperature = &types.Temperature{
			ValueCelsius: data.Temperature.Value,
		}
	}

	if data.TargetTemperature.Enabled {
		airTemperature.TemperatureGoal = &traits.AirTemperature_TemperatureSetPoint{
			TemperatureSetPoint: &types.Temperature{ValueCelsius: data.TargetTemperature.Value},
		}
	}

	if data.Position > 0 {
		airTemperature.Mode = traits.AirTemperature_HEAT
	} else {
		airTemperature.Mode = traits.AirTemperature_OFF
	}

	return airTemperature
}
