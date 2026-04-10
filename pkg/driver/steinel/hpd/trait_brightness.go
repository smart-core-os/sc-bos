package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/brightnesssensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type brightnessSensor struct {
	brightnesssensorpb.UnimplementedBrightnessSensorApiServer

	logger *zap.Logger

	client *Client

	value *resource.Value // *traits.AmbientBrightness
}

func newBrightnessSensor(client *Client, logger *zap.Logger) *brightnessSensor {
	return &brightnessSensor{
		client: client,
		logger: logger,
		value:  resource.NewValue(resource.WithInitialValue(&brightnesssensorpb.AmbientBrightness{}), resource.WithNoDuplicates()),
	}
}

func (b *brightnessSensor) GetAmbientBrightness(context.Context, *brightnesssensorpb.GetAmbientBrightnessRequest) (*brightnesssensorpb.AmbientBrightness, error) {
	response := SensorResponse{}
	if err := doGetRequest(b.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := b.getUpdate(&response); err != nil {
		return nil, err
	}
	return b.value.Get().(*brightnesssensorpb.AmbientBrightness), nil
}

func (b *brightnessSensor) getUpdate(response *SensorResponse) error {
	lev := &brightnesssensorpb.AmbientBrightness{
		BrightnessLux: float32(response.Brightness1),
	}
	_, err := b.value.Set(lev)
	return err
}

func (b *brightnessSensor) PullAmbientBrightness(request *brightnesssensorpb.PullAmbientBrightnessRequest, server grpc.ServerStreamingServer[brightnesssensorpb.PullAmbientBrightnessResponse]) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := b.value.Pull(ctx, resource.WithBackpressure(false))
	for change := range changes {
		v := change.Value.(*brightnesssensorpb.AmbientBrightness)

		err := server.Send(&brightnesssensorpb.PullAmbientBrightnessResponse{
			Changes: []*brightnesssensorpb.PullAmbientBrightnessResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), AmbientBrightness: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
