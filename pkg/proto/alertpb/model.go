package alertpb

import (
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type Model struct {
	alerts []*resource.Value // of *alertpb.Alert
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) GetAllAlerts() []*Alert {
	var results []*Alert

	for _, v := range m.alerts {
		results = append(results, v.Get().(*Alert))
	}
	return results
}

func (m *Model) AddAlert(a *Alert, opts ...resource.WriteOption) {
	value := resource.NewValue(resource.WithInitialValue(&Alert{}))
	value.Set(a, opts...)
	m.alerts = append(m.alerts, value)
}
