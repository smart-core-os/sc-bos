package hpd

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type brightnessSensor struct {
	traits.UnimplementedBrightnessSensorApiServer

	logger *zap.Logger

	client *Client

	value *resource.Value // *traits.AmbientBrightness
}

func newBrightnessSensor(client *Client, logger *zap.Logger) *brightnessSensor {
	return &brightnessSensor{
		client: client,
		logger: logger,
		value:  resource.NewValue(resource.WithInitialValue(&traits.AmbientBrightness{}), resource.WithNoDuplicates()),
	}
}

func (b *brightnessSensor) GetAmbientBrightness(context.Context, *traits.GetAmbientBrightnessRequest) (*traits.AmbientBrightness, error) {
	response := SensorResponse{}
	if err := doGetRequest(b.client, &response, "sensor"); err != nil {
		return nil, err
	}
	if err := b.getUpdate(&response); err != nil {
		return nil, err
	}
	return b.value.Get().(*traits.AmbientBrightness), nil
}

func (b *brightnessSensor) getUpdate(response *SensorResponse) error {
	lev := &traits.AmbientBrightness{
		BrightnessLux: float32(response.Brightness1),
	}
	_, err := b.value.Set(lev)
	return err
}

func (b *brightnessSensor) PullAmbientBrightness(request *traits.PullAmbientBrightnessRequest, server grpc.ServerStreamingServer[traits.PullAmbientBrightnessResponse]) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	changes := b.value.Pull(ctx, resource.WithBackpressure(false))
	for change := range changes {
		v := change.Value.(*traits.AmbientBrightness)

		err := server.Send(&traits.PullAmbientBrightnessResponse{
			Changes: []*traits.PullAmbientBrightnessResponse_Change{
				{Name: request.GetName(), ChangeTime: timestamppb.New(change.ChangeTime), AmbientBrightness: v},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
