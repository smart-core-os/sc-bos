package merge

import (
	"context"
	"encoding/json"
	"sync"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet/property"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/once"
)

type meterConfig struct {
	config.Trait
	Usage *config.ValueSource `json:"usage,omitempty"`
	// Unit is the unit of the usage reading, after any usage.scale is applied.
	// When empty the unit is read from the usage object's BACnet Units property, unless usage is scaled.
	Unit string `json:"unit,omitempty"`
}

func readMeterConfig(raw []byte) (cfg meterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type meterTrait struct {
	client *comm.Client
	known  known.Context
	fault  *traitFault
	logger *zap.Logger

	model *meterpb.Model
	*meterpb.ModelServer
	meterpb.UnimplementedMeterInfoServer
	config   meterConfig
	pollTask *task.Intermittent

	// readUnits reads the usage object's BACnet Units property, replaced in tests.
	readUnits  func(ctx context.Context) (any, error)
	unitOnce   once.RetryError
	unit       string // resolved from readUnits, valid once unitOnce succeeds
	warnScaled sync.Once
	warnNoUnit sync.Once
}

func newMeter(client *comm.Client, devices known.Context, fault *traitFault, config config.RawTrait, logger *zap.Logger) (*meterTrait, error) {
	cfg, err := readMeterConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := meterpb.NewModel(resource.WithMessageEquivalence(cmp.Equal(
		cmp.FloatValueApprox(0, 0.0001),
	)))
	t := &meterTrait{
		client:      client,
		known:       devices,
		fault:       fault,
		logger:      logger,
		model:       model,
		ModelServer: meterpb.NewModelServer(model),
		config:      cfg,
	}
	t.pollTask = task.NewIntermittent(t.startPoll)
	t.readUnits = t.readUsageUnits
	return t, nil
}

func (t *meterTrait) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(t.config.Name,
		node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(t)),
		node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(t)),
		node.HasTrait(meterpb.TraitName),
	)
}

func (t *meterTrait) DescribeMeterReading(ctx context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	unit, err := t.usageUnit(ctx)
	if err != nil {
		return nil, err
	}
	return &meterpb.MeterReadingSupport{
		ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
		UsageUnit:       unit,
	}, nil
}

// usageUnit returns the unit of the usage reading, or "" if the meter has none.
// Consumers that can't tell a meter's unit have to guess, so a meter with no unit is worth a warning.
//
// While the unit can't be read from the device usageUnit returns an Unavailable error rather than "".
// Consumers treat "" as the meter's answer: a zone group would assume the member matches it and stop asking.
func (t *meterTrait) usageUnit(ctx context.Context) (string, error) {
	if t.config.Unit != "" {
		return t.config.Unit, nil
	}
	if t.config.Usage == nil {
		return "", nil
	}
	if s := t.config.Usage.Scale; s != 0 && s != 1 {
		// The device's Units describe the value before scaling, so they'd be wrong for what we report.
		t.warnScaled.Do(func() {
			t.logger.Warn("meter usage is scaled but has no unit configured, set unit to the scaled unit",
				zap.Float64("scale", s))
		})
		return "", nil
	}
	err := t.unitOnce.Do(ctx, func() error {
		// not ctx: the read is shared by all waiting callers, so one caller giving up shouldn't fail it for the rest
		readCtx, cancel := context.WithTimeout(context.Background(), t.config.PollTimeoutDuration())
		defer cancel()
		raw, err := t.readUnits(readCtx)
		if err != nil {
			return err
		}
		n, err := comm.IntValue(raw)
		if err != nil {
			return err
		}
		t.unit = meterUnitSymbol(comm.EngineeringUnits(n))
		return nil
	})
	if err != nil {
		t.logger.Warn("reading meter units from device, will try again on next describe", zap.Error(err))
		return "", status.Errorf(codes.Unavailable, "reading meter units from device: %v", err)
	}
	if t.unit == "" {
		t.warnNoUnit.Do(func() {
			t.logger.Warn("meter has no usable unit on the device or in config, set unit")
		})
	}
	return t.unit, nil
}

func (t *meterTrait) readUsageUnits(ctx context.Context) (any, error) {
	src := *t.config.Usage
	units := config.PropertyID(property.Units)
	src.Property = &units
	src.Scale = 0
	return comm.ReadProperty(ctx, t.client, t.known, src)
}

// meterUnits are the spellings meter consumers recognise for BACnet engineering units a meter might report in.
// comm.EngineeringUnits.String uses typographic forms like "kW·h" that consumers (and dbo.UnitName) don't parse.
var meterUnits = map[comm.EngineeringUnits]string{
	16:  "J",
	17:  "kJ",
	18:  "Wh",
	19:  "kWh",
	79:  "ft³",
	80:  "m³",
	82:  "L",
	126: "MJ",
	146: "MWh",
}

// meterUnitSymbol returns the unit to report for u, or "" if u says nothing useful.
func meterUnitSymbol(u comm.EngineeringUnits) string {
	if s, ok := meterUnits[u]; ok {
		return s
	}
	const noUnits comm.EngineeringUnits = 95
	if u == noUnits {
		return ""
	}
	if s, ok := u.Symbol(); ok {
		return s
	}
	return ""
}

func (t *meterTrait) GetMeterReading(ctx context.Context, request *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	_, err := t.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return t.ModelServer.GetMeterReading(ctx, request)
}

func (t *meterTrait) PullMeterReadings(request *meterpb.PullMeterReadingsRequest, server meterpb.MeterApi_PullMeterReadingsServer) error {
	err := t.pollTask.Attach(server.Context())
	if err != nil {
		return err
	}

	// avoid returning the zero value if we are the first to attach since reboot
	timeoutCtx, cleanup := context.WithTimeout(server.Context(), t.config.PollTimeoutDuration())
	defer cleanup()
	for change := range t.model.PullMeterReadings(timeoutCtx) {
		if change.Value.Usage != 0 {
			break
		}
	}

	return t.ModelServer.PullMeterReadings(request, server)
}

func (t *meterTrait) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "meter", t.config.PollPeriodDuration(), t.config.PollTimeoutDuration(), t.logger, func(ctx context.Context) error {
		_, err := t.pollPeer(ctx)
		return err
	})
}

func (t *meterTrait) pollPeer(ctx context.Context) (*meterpb.MeterReading, error) {
	responses := comm.ReadProperties(ctx, t.client, t.known, *t.config.Usage)
	var errs []error
	usage, err := comm.Float32Value(responses[0])
	if err != nil {
		errs = append(errs, comm.ErrReadProperty{Prop: "usage", Cause: err})
	}
	t.fault.update(t.config.Name, trait.Meter, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	data := &meterpb.MeterReading{
		Usage: usage,
	}
	return t.model.UpdateMeterReading(data)
}

func (t *meterTrait) PollTask() *task.Intermittent { return t.pollTask }
