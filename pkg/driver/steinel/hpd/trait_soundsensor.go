package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type soundSensor struct {
	soundsensorpb.UnimplementedSoundSensorApiServer

	logger *zap.Logger

	client *Client

	value *resource.Value // *soundsensorpb.SoundLevel
}

func newSoundSensor(client *Client, logger *zap.Logger) *soundSensor {
	return &soundSensor{
		value:  resource.NewValue(resource.WithInitialValue(&soundsensorpb.SoundLevel{}), resource.WithNoDuplicates()),
		client: client,
		logger: logger,
	}
}

func (s *soundSensor) GetSoundLevel(context.Context, *soundsensorpb.GetSoundLevelRequest) (*soundsensorpb.SoundLevel, error) {
	response := SensorResponse{}
	if err := doGetRequest(s.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := s.getUpdate(&response); err != nil {
		return nil, err
	}
	return s.value.Get().(*soundsensorpb.SoundLevel), nil
}

func (s *soundSensor) getUpdate(response *SensorResponse) error {
	lev := &soundsensorpb.SoundLevel{
		SoundPressureLevel: ptr(float32(response.Noise)),
	}
	_, err := s.value.Set(lev)
	return err
}

func (s *soundSensor) PullSoundLevel(request *soundsensorpb.PullSoundLevelRequest, server soundsensorpb.SoundSensorApi_PullSoundLevelServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := s.value.Pull(ctx, resource.WithBackpressure(false))
	for change := range changes {
		v := change.Value.(*soundsensorpb.SoundLevel)

		err := server.Send(&soundsensorpb.PullSoundLevelResponse{
			Changes: []*soundsensorpb.PullSoundLevelResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), SoundLevel: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
