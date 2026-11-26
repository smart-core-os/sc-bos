package merge

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

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
	// Production is read into MeterReading.Produced, for meters that export as well as import.
	Production *config.ValueSource `json:"production,omitempty"`
	// ProducedUnit is the unit of the production reading, after any production.scale is applied.
	// When empty the unit is read from the production object's BACnet Units property, unless production is scaled.
	// It is not taken from Unit: import and export are the same commodity but needn't be in the same unit.
	ProducedUnit string `json:"producedUnit,omitempty"`
}

func readMeterConfig(raw []byte) (cfg meterConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		return
	}
	if cfg.Usage == nil && cfg.Production == nil {
		err = errors.New("meter needs at least one of usage or production")
	}
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

	usageUnit    *readingUnit
	producedUnit *readingUnit // nil unless production is configured
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
	t.usageUnit = t.newReadingUnit("usage", "unit", cfg.Unit, cfg.Usage)
	if cfg.Production != nil {
		t.producedUnit = t.newReadingUnit("production", "producedUnit", cfg.ProducedUnit, cfg.Production)
	}
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
	usageUnit, err := t.usageUnit.get(ctx)
	if err != nil {
		return nil, err
	}
	support := &meterpb.MeterReadingSupport{
		ResourceSupport: &typespb.ResourceSupport{Readable: true, Observable: true},
		UsageUnit:       usageUnit,
	}
	// The produced unit stays empty unless production is read: consumers, dbo.MeterFields among them, use its
	// presence to decide whether the meter reports export at all, and a spurious unit yields a constant-zero series.
	if t.producedUnit != nil {
		support.ProducedUnit, err = t.producedUnit.get(ctx)
		if err != nil {
			return nil, err
		}
	}
	return support, nil
}

// readingUnit resolves the unit of one of the meter's readings: the configured unit, else the BACnet Units
// property of the object the reading comes from.
type readingUnit struct {
	reading    string // "usage" or "production", for messages
	configKey  string // the config field that sets the unit, for messages
	configured string
	src        *config.ValueSource // nil if the reading isn't configured
	timeout    time.Duration
	logger     *zap.Logger

	// read reads src's BACnet Units property, replaced in tests.
	read       func(ctx context.Context) (any, error)
	once       once.RetryError
	unit       string // resolved from read, valid once once succeeds
	warnScaled sync.Once
	warnNoUnit sync.Once
}

func (t *meterTrait) newReadingUnit(reading, configKey, configured string, src *config.ValueSource) *readingUnit {
	u := &readingUnit{
		reading:    reading,
		configKey:  configKey,
		configured: configured,
		src:        src,
		timeout:    t.config.PollTimeoutDuration(),
		logger:     t.logger.With(zap.String("reading", reading)),
	}
	u.read = func(ctx context.Context) (any, error) {
		src := *u.src
		units := config.PropertyID(property.Units)
		src.Property = &units
		src.Scale = 0
		return comm.ReadProperty(ctx, t.client, t.known, src)
	}
	return u
}

// get returns the unit of the reading, or "" if it has none.
// Consumers that can't tell a meter's unit have to guess, so a meter with no unit is worth a warning.
//
// While the unit can't be read from the device get returns an Unavailable error rather than "".
// Consumers treat "" as the meter's answer: a zone group would assume the member matches it and stop asking.
func (u *readingUnit) get(ctx context.Context) (string, error) {
	if u.configured != "" {
		return u.configured, nil
	}
	if u.src == nil {
		return "", nil
	}
	if s := u.src.Scale; s != 0 && s != 1 {
		// The device's Units describe the value before scaling, so they'd be wrong for what we report.
		u.warnScaled.Do(func() {
			u.logger.Warn("meter reading is scaled but has no unit configured, set "+u.configKey+" to the scaled unit",
				zap.Float64("scale", s))
		})
		return "", nil
	}
	err := u.once.Do(ctx, func() error {
		// not ctx: the read is shared by all waiting callers, so one caller giving up shouldn't fail it for the rest
		readCtx, cancel := context.WithTimeout(context.Background(), u.timeout)
		defer cancel()
		raw, err := u.read(readCtx)
		if err != nil {
			return err
		}
		n, err := comm.IntValue(raw)
		if err != nil {
			return err
		}
		u.unit = meterUnitSymbol(comm.EngineeringUnits(n))
		return nil
	})
	if err != nil {
		u.logger.Warn("reading meter units from device, will try again on next describe", zap.Error(err))
		return "", status.Errorf(codes.Unavailable, "reading meter %s units from device: %v", u.reading, err)
	}
	if u.unit == "" {
		u.warnNoUnit.Do(func() {
			u.logger.Warn("meter reading has no usable unit on the device or in config, set " + u.configKey)
		})
	}
	return u.unit, nil
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
		if change.Value.Usage != 0 || change.Value.Produced != 0 {
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
	reads := t.meterReads()
	srcs := make([]config.ValueSource, len(reads))
	for i, r := range reads {
		srcs[i] = r.src
	}
	data, errs := applyMeterReads(reads, comm.ReadProperties(ctx, t.client, t.known, srcs...))
	t.fault.update(t.config.Name, trait.Meter, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}
	return t.model.UpdateMeterReading(data)
}

// meterRead is one configured value source and where its value goes in a MeterReading.
type meterRead struct {
	prop string // names the reading in errors
	src  config.ValueSource
	set  func(r *meterpb.MeterReading, v float32)
}

// meterReads returns the value sources to read on each poll, in the order their responses come back.
func (t *meterTrait) meterReads() []meterRead {
	var reads []meterRead
	if t.config.Usage != nil {
		reads = append(reads, meterRead{prop: "usage", src: *t.config.Usage,
			set: func(r *meterpb.MeterReading, v float32) { r.Usage = v }})
	}
	if t.config.Production != nil {
		reads = append(reads, meterRead{prop: "production", src: *t.config.Production,
			set: func(r *meterpb.MeterReading, v float32) { r.Produced = v }})
	}
	return reads
}

// applyMeterReads builds a reading from responses, which pair with reads by index.
func applyMeterReads(reads []meterRead, responses []any) (*meterpb.MeterReading, []error) {
	data := &meterpb.MeterReading{}
	var errs []error
	for i, r := range reads {
		v, err := comm.Float32Value(responses[i])
		if err != nil {
			errs = append(errs, comm.ErrReadProperty{Prop: r.prop, Cause: err})
			continue
		}
		r.set(data, v)
	}
	return data, errs
}

func (t *meterTrait) PollTask() *task.Intermittent { return t.pollTask }
