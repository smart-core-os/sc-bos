package udmipb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	exportMessage *resource.Value // of *udmipb.MqttMessage
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&udmipb.MqttMessage{})}
	value := resource.NewValue(append(defaultOptions, opts...)...)
	return &Model{
		exportMessage: value,
	}
}

func (m *Model) GetExportMessage(opts ...resource.ReadOption) (*udmipb.MqttMessage, error) {
	return m.exportMessage.Get(opts...).(*udmipb.MqttMessage), nil
}

func (m *Model) UpdateExportMessage(message *udmipb.MqttMessage, opts ...resource.WriteOption) (*udmipb.MqttMessage, error) {
	res, err := m.exportMessage.Set(message, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*udmipb.MqttMessage), nil
}

func (m *Model) PullExportMessages(ctx context.Context, opts ...resource.ReadOption) <-chan PullExportMessageChange {
	return resources.PullValue[*udmipb.MqttMessage](ctx, m.exportMessage.Pull(ctx, opts...))
}

type PullExportMessageChange = resources.ValueChange[*udmipb.MqttMessage]
