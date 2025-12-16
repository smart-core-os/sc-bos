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

func (e *Electric) handleElectricEvent(node *ua.NodeID, value any) {
	if e.cfg.Demand == nil {
		e.logger.Warn("electric trait configured without demand")
		return
	}
	switch {
	case e.cfg.Demand.ApparentPower != nil && nodeIdsAreEqual(e.cfg.Demand.ApparentPower.NodeId, node):
		ap, err := conv.Float32Value(value)
		if err != nil {
			e.logger.Warn("error reading float32 for apparent power", zap.String("error", err.Error()))
		}
		_, _ = e.value.Set(&traits.ElectricDemand{
			ApparentPower: &ap,
		}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
			Paths: []string{"apparent_power"},
		}))
	case e.cfg.Demand.ReactivePower != nil && nodeIdsAreEqual(e.cfg.Demand.ReactivePower.NodeId, node):
		rp, err := conv.Float32Value(value)
		if err != nil {
			e.logger.Warn("error reading float32 for reactive power", zap.String("error", err.Error()))
		}
		_, _ = e.value.Set(&traits.ElectricDemand{
			ReactivePower: &rp,
		}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
			Paths: []string{"reactive_power"},
		}))
	case e.cfg.Demand.RealPower != nil && nodeIdsAreEqual(e.cfg.Demand.RealPower.NodeId, node):
		rp, err := conv.Float32Value(value)
		if err != nil {
			e.logger.Warn("error reading float32 for real power", zap.String("error", err.Error()))
		}
		_, _ = e.value.Set(&traits.ElectricDemand{
			RealPower: &rp,
		}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
			Paths: []string{"real_power"},
		}))
	case e.cfg.Demand.PowerFactor != nil && nodeIdsAreEqual(e.cfg.Demand.PowerFactor.NodeId, node):
		pf, err := conv.Float32Value(value)
		if err != nil {
			e.logger.Warn("error reading float32 for power factor", zap.String("error", err.Error()))
		}
		_, _ = e.value.Set(&traits.ElectricDemand{
			PowerFactor: &pf,
		}, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
			Paths: []string{"power_factor"},
		}))
	}
}
