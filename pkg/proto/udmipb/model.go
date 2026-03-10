package udmipb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	exportMessage *resource.Value // of *udmipb.MqttMessage
}

func NewModel(opts ...resource.Option) *Model {
	defaultOptions := []resource.Option{resource.WithInitialValue(&MqttMessage{})}
	value := resource.NewValue(append(defaultOptions, opts...)...)
	return &Model{
		exportMessage: value,
	}
}

func (m *Model) GetExportMessage(opts ...resource.ReadOption) (*MqttMessage, error) {
	return m.exportMessage.Get(opts...).(*MqttMessage), nil
}

func (m *Model) UpdateExportMessage(message *MqttMessage, opts ...resource.WriteOption) (*MqttMessage, error) {
	res, err := m.exportMessage.Set(message, opts...)
	if err != nil {
		return nil, err
	}
	return res.(*MqttMessage), nil
}

func (m *Model) PullExportMessages(ctx context.Context, opts ...resource.ReadOption) <-chan PullExportMessageChange {
	return resources.PullValue[*MqttMessage](ctx, m.exportMessage.Pull(ctx, opts...))
}

type PullExportMessageChange = resources.ValueChange[*MqttMessage]
