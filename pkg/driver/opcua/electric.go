package opcua

import (
	"context"
	"encoding/json"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Electric implements the Smart Core Electric trait for OPC UA devices.
// It maps OPC UA variable nodes to Electric demand measurements (power, voltage, current, etc.).
type Electric struct {
	traits.UnimplementedElectricApiServer

	cfg    config.ElectricConfig
	logger *zap.Logger
	scName string
	value  *resource.Value // *traits.ElectricDemand
}

func readElectricConfig(raw []byte) (cfg config.ElectricConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newElectric(n string, config config.RawTrait, l *zap.Logger) (*Electric, error) {
	cfg, err := readElectricConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	return &Electric{
		cfg:    cfg,
		logger: l,
		scName: n,
		value:  resource.NewValue(resource.WithInitialValue(&traits.ElectricDemand{}), resource.WithNoDuplicates()),
	}, nil
}

func (e *Electric) GetDemand(context.Context, *traits.GetDemandRequest) (*traits.ElectricDemand, error) {
	return e.value.Get().(*traits.ElectricDemand), nil
}

func (e *Electric) PullDemand(_ *traits.PullDemandRequest, server traits.ElectricApi_PullDemandServer) error {
	for value := range e.value.Pull(server.Context()) {
		err := server.Send(&traits.PullDemandResponse{Changes: []*traits.PullDemandResponse_Change{
			{
				Name:       e.scName,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Demand:     value.Value.(*traits.ElectricDemand),
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

// updatePowerField is a helper that converts, scales, and updates a single power measurement field.
// It handles conversion to float32, scaling via the ValueSource, type assertion, and updating the demand state.
func (e *Electric) updatePowerField(value any, source *config.ValueSource, fieldName string, setter func(*traits.ElectricDemand, float32)) {

	rawValue, err := conv.Float32Value(value)
	if err != nil {
		e.logger.Warn("error reading float32", zap.String("device", e.scName), zap.String("field", fieldName), zap.Error(err))
		return
	}

	scaled := source.Scaled(rawValue)
	powerValue, ok := scaled.(float32)
	if !ok {
		e.logger.Warn("scaled value is not float32", zap.String("device", e.scName), zap.String("field", fieldName), zap.Any("value", scaled))
		return
	}

	demand := &traits.ElectricDemand{}
	setter(demand, powerValue)
	_, _ = e.value.Set(demand, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
		Paths: []string{fieldName},
	}))
}

func (e *Electric) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	if e.cfg.Demand == nil {
		e.logger.Warn("Electric trait configured without demand", zap.String("device", e.scName))
		return
	}

	switch {
	case e.cfg.Demand.ApparentPower != nil && nodeIdsAreEqual(e.cfg.Demand.ApparentPower.NodeId, node):
		e.updatePowerField(value, e.cfg.Demand.ApparentPower, "apparent_power", func(d *traits.ElectricDemand, v float32) {
			d.ApparentPower = &v
		})
	case e.cfg.Demand.ReactivePower != nil && nodeIdsAreEqual(e.cfg.Demand.ReactivePower.NodeId, node):
		e.updatePowerField(value, e.cfg.Demand.ReactivePower, "reactive_power", func(d *traits.ElectricDemand, v float32) {
			d.ReactivePower = &v
		})
	case e.cfg.Demand.RealPower != nil && nodeIdsAreEqual(e.cfg.Demand.RealPower.NodeId, node):
		e.updatePowerField(value, e.cfg.Demand.RealPower, "real_power", func(d *traits.ElectricDemand, v float32) {
			d.RealPower = &v
		})
	case e.cfg.Demand.PowerFactor != nil && nodeIdsAreEqual(e.cfg.Demand.PowerFactor.NodeId, node):
		e.updatePowerField(value, e.cfg.Demand.PowerFactor, "power_factor", func(d *traits.ElectricDemand, v float32) {
			d.PowerFactor = &v
		})
	}
}
