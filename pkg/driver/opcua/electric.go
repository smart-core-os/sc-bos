package opcua

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Electric implements the Smart Core Electric trait for OPC UA devices.
// It maps OPC UA variable nodes to Electric demand measurements (power, voltage, current, etc.).
//
// Each field is only served as current while the subscription vouches for its node (see
// pointLiveness). Get reads the rest live, leaves out any field it still cannot read, and
// reports Unavailable when that leaves nothing.
type Electric struct {
	electricpb.UnimplementedElectricApiServer

	cfg    config.ElectricConfig
	logger *zap.Logger
	scName string
	value  *resource.Value // *traits.ElectricDemand

	fields []*electricField
	reader pointReader
	live   *pointLiveness
}

// electricField is one ElectricDemand field and the node it is read from.
type electricField struct {
	name   string // the proto field name, for update masks
	source *config.ValueSource
	node   *ua.NodeID // for reading it live, nil when the node id does not parse
	get    func(*electricpb.ElectricDemand) *float32
	set    func(*electricpb.ElectricDemand, *float32)
}

func readElectricConfig(raw []byte) (cfg config.ElectricConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newElectric(n string, config config.RawTrait, reader pointReader, l *zap.Logger) (*Electric, error) {
	cfg, err := readElectricConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	fields := electricFields(cfg)
	nodeIds := make([]string, len(fields))
	for i, f := range fields {
		nodeIds[i] = f.source.NodeId
	}
	return &Electric{
		cfg:    cfg,
		logger: l,
		scName: n,
		value:  resource.NewValue(resource.WithInitialValue(&electricpb.ElectricDemand{}), resource.WithNoDuplicates()),
		fields: fields,
		reader: reader,
		live:   newPointLiveness(readerConnected(reader), nodeIds...),
	}, nil
}

// electricFields lists the demand fields cfg maps to a node.
func electricFields(cfg config.ElectricConfig) []*electricField {
	// the single-phase config is embedded by pointer, so a config with only phases leaves it nil
	if cfg.Demand == nil || cfg.Demand.ElectricPhaseConfig == nil {
		return nil
	}
	phase := cfg.Demand.ElectricPhaseConfig
	candidates := []*electricField{
		{name: "apparent_power", source: phase.ApparentPower,
			get: func(d *electricpb.ElectricDemand) *float32 { return d.ApparentPower },
			set: func(d *electricpb.ElectricDemand, v *float32) { d.ApparentPower = v }},
		{name: "reactive_power", source: phase.ReactivePower,
			get: func(d *electricpb.ElectricDemand) *float32 { return d.ReactivePower },
			set: func(d *electricpb.ElectricDemand, v *float32) { d.ReactivePower = v }},
		{name: "real_power", source: phase.RealPower,
			get: func(d *electricpb.ElectricDemand) *float32 { return d.RealPower },
			set: func(d *electricpb.ElectricDemand, v *float32) { d.RealPower = v }},
		{name: "power_factor", source: phase.PowerFactor,
			get: func(d *electricpb.ElectricDemand) *float32 { return d.PowerFactor },
			set: func(d *electricpb.ElectricDemand, v *float32) { d.PowerFactor = v }},
	}
	var fields []*electricField
	for _, f := range candidates {
		if f.source == nil {
			continue
		}
		f.node = parseLiveNode(f.source.NodeId)
		fields = append(fields, f)
	}
	return fields
}

func (e *Electric) GetDemand(ctx context.Context, _ *electricpb.GetDemandRequest) (*electricpb.ElectricDemand, error) {
	cached := e.value.Get().(*electricpb.ElectricDemand)
	out := &electricpb.ElectricDemand{}
	served := 0
	var unvouched []*electricField
	for _, f := range e.fields {
		if v := f.get(cached); v != nil && e.live.vouched(f.source.NodeId) {
			f.set(out, v)
			served++
			continue
		}
		unvouched = append(unvouched, f)
	}
	read, err := e.readLive(ctx, unvouched, out)
	if served+read == 0 {
		if err == nil {
			err = errors.New("no demand field can be read")
		}
		return nil, status.Errorf(codes.Unavailable, "electric demand: %v", err)
	}
	return out, nil
}

// readLive reads fields from the server into out, for the fields the subscription cannot vouch
// for, in one request. It reports how many it filled in, and why the others were left out.
func (e *Electric) readLive(ctx context.Context, fields []*electricField, out *electricpb.ElectricDemand) (int, error) {
	var nodes []*ua.NodeID
	var readable []*electricField
	var errs []error
	for _, f := range fields {
		if f.node == nil {
			errs = append(errs, fmt.Errorf("%s cannot be read", f.name))
			continue
		}
		nodes = append(nodes, f.node)
		readable = append(readable, f)
	}
	if len(readable) == 0 {
		return 0, errors.Join(errs...)
	}
	if e.reader == nil {
		return 0, errors.Join(append(errs, errors.New("no reader"))...)
	}

	gens := make([]uint64, len(readable))
	for i, f := range readable {
		gens[i] = e.live.beginRead(f.source.NodeId)
	}
	dvs, err := e.reader.ReadValues(ctx, nodes...)
	if err != nil {
		return 0, errors.Join(append(errs, err)...)
	}
	read := 0
	for i, f := range readable {
		raw, err := liveValue(dvs[i])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s %w", f.name, err))
			continue
		}
		v, err := e.scaled(f, raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.name, err))
			continue
		}
		f.set(out, &v)
		read++
		e.live.readGood(f.source.NodeId, gens[i], func() {
			if got := f.get(e.value.Get().(*electricpb.ElectricDemand)); got == nil || *got != v {
				e.setField(f, v)
			}
		})
	}
	return read, errors.Join(errs...)
}

func (e *Electric) PullDemand(_ *electricpb.PullDemandRequest, server electricpb.ElectricApi_PullDemandServer) error {
	for value := range e.value.Pull(server.Context()) {
		err := server.Send(&electricpb.PullDemandResponse{Changes: []*electricpb.PullDemandResponse_Change{
			{
				Name:       e.scName,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Demand:     value.Value.(*electricpb.ElectricDemand),
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

// scaled converts and scales a raw value for f.
func (e *Electric) scaled(f *electricField, value any) (float32, error) {
	rawValue, err := conv.Float32Value(value)
	if err != nil {
		return 0, fmt.Errorf("error reading float32: %w", err)
	}
	scaled := f.source.Scaled(rawValue)
	v, ok := scaled.(float32)
	if !ok {
		return 0, fmt.Errorf("scaled value %v is not float32", scaled)
	}
	return v, nil
}

// setField updates the one field in the cached demand.
func (e *Electric) setField(f *electricField, v float32) {
	demand := &electricpb.ElectricDemand{}
	f.set(demand, &v)
	_, _ = e.value.Set(demand, resource.WithUpdateMask(&fieldmaskpb.FieldMask{
		Paths: []string{f.name},
	}))
}

// fieldsFor is every field read from node. More than one when the config maps two fields to
// the same node, and then a notification has to update them all: they share the node's
// liveness, so a field it skipped would be vouched for with whatever a live read last left.
func (e *Electric) fieldsFor(node *ua.NodeID) []*electricField {
	var fields []*electricField
	for _, f := range e.fields {
		if nodeIdsAreEqual(f.source.NodeId, node) {
			fields = append(fields, f)
		}
	}
	return fields
}

func (e *Electric) handleEvent(_ context.Context, node *ua.NodeID, value any) {
	if e.cfg.Demand == nil {
		e.logger.Warn("Electric trait configured without demand", zap.String("device", e.scName))
		return
	}
	fields := e.fieldsFor(node)
	if len(fields) == 0 {
		return
	}
	values := make([]float32, len(fields))
	for i, f := range fields {
		v, err := e.scaled(f, value)
		if err != nil {
			// the server sent a value we cannot use, so the cached one is no longer its value
			e.live.unusable(f.source.NodeId)
			e.logger.Warn("unusable value", zap.String("device", e.scName), zap.String("field", f.name), zap.Error(err))
			return
		}
		values[i] = v
	}
	e.live.good(fields[0].source.NodeId, func() {
		for i, f := range fields {
			e.setField(f, values[i])
		}
	})
}

func (e *Electric) pointBad(node *ua.NodeID) { e.live.unusable(node.String()) }
func (e *Electric) connectionSuspect()       { e.live.connectionSuspect() }
func (e *Electric) subscriptionLost()        { e.live.subscriptionLost() }
