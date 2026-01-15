package alert

import (
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type Model struct {
	alerts []*resource.Value // of *alertpb.Alert
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) GetAllAlerts() []*alertpb.Alert {
	var results []*alertpb.Alert

	for _, v := range m.alerts {
		results = append(results, v.Get().(*alertpb.Alert))
	}
	return results
}

func (m *Model) AddAlert(a *alertpb.Alert, opts ...resource.WriteOption) {
	value := resource.NewValue(resource.WithInitialValue(&alertpb.Alert{}))
	value.Set(a, opts...)
	m.alerts = append(m.alerts, value)
}
