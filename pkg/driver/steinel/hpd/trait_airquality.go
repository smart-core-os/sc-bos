package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/mqttpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type AirQualitySensor struct {
	traits.UnimplementedAirQualitySensorApiServer
	mqttpb.UnimplementedMqttServiceServer
	udmipb.UnimplementedUdmiServiceServer

	logger *zap.Logger

	client *Client

	AirQualityValue *resource.Value
}

var _ sensor = (*AirQualitySensor)(nil)

func newAirQualitySensor(client *Client, logger *zap.Logger) *AirQualitySensor {
	return &AirQualitySensor{
		client:          client,
		logger:          logger,
		AirQualityValue: resource.NewValue(resource.WithInitialValue(&traits.AirQuality{}), resource.WithNoDuplicates()),
	}
}

func (a *AirQualitySensor) GetAirQuality(_ context.Context, _ *traits.GetAirQualityRequest) (*traits.AirQuality, error) {
	response := SensorResponse{}
	if err := doGetRequest(a.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := a.GetUpdate(&response); err != nil {
		return nil, err
	}
	return a.AirQualityValue.Get().(*traits.AirQuality), nil
}

func (a *AirQualitySensor) PullAirQuality(request *traits.PullAirQualityRequest, server traits.AirQualitySensorApi_PullAirQualityServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := a.AirQualityValue.Pull(ctx)

	for change := range changes {
		v := change.Value.(*traits.AirQuality)

		err := server.Send(&traits.PullAirQualityResponse{
			Changes: []*traits.PullAirQualityResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), AirQuality: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *AirQualitySensor) GetUpdate(response *SensorResponse) error {
	co2 := float32(response.CO2)
	// voc is exposed as ppb, we need to convert to ppm
	voc := float32(response.VOC) / 1000
	airPressure := float32(response.AirPressure)
	infectionRisk := float32(response.AerosolRiskOfInfection)

	q := &traits.AirQuality{
		CarbonDioxideLevel:       &co2,
		VolatileOrganicCompounds: &voc,
	}

	if airPressure > 0 {
		q.AirPressure = &airPressure
	}
	if infectionRisk > 0 {
		q.InfectionRisk = &infectionRisk
	}

	if response.IAQ > 0 {
		// the HPD3 (and possibly other Steinels that have this prop) use a range of 0-2000, with lower numbers being
		// better. Over 500 is considered unacceptable, so we're mapping 500-2000 onto 0-10%
		score := 2000 - response.IAQ
		if score < 1500 {
			score = score / 150
		} else {
			score = 10 + (score-1500)/5
		}
		fscore := float32(score)
		q.Score = &fscore
	}

	_, err := a.AirQualityValue.Set(q)
	if err != nil {
		return err
	}

	return nil
}

func (a *AirQualitySensor) GetName() string {
	return "Air Quality"
}
