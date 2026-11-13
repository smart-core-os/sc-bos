package meter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/dbo"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/cmp"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/merge"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/meter/config"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/run"
)

type Group struct {
	meterpb.UnimplementedMeterApiServer
	meterpb.UnimplementedMeterInfoServer
	apiClient        meterpb.MeterApiClient
	infoClient       meterpb.MeterInfoClient
	historyApiClient meterpb.MeterHistoryClient
	names            []string
	readOnly         bool

	planMu      sync.Mutex
	plan        *unitPlan // nil until first resolved, replaced until plan.complete
	planAt      time.Time // when plan was resolved
	resolving   bool      // a background re-resolve of plan is running
	warnedEmpty sync.Map  // member name -> struct{}, members we've warned declare no units

	historyBackupConf *config.HistoryBackup
	now               func() time.Time

	logger *zap.Logger
}

const (
	describeMemberTimeout = 5 * time.Second
	// planRetryInterval limits how often members are described again while some haven't answered.
	planRetryInterval = time.Minute
)

func (g *Group) DescribeMeterReading(ctx context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	plan := g.unitPlan(ctx)
	if plan.err != nil {
		return nil, plan.err
	}
	if plan.describeErr != nil {
		return nil, plan.describeErr
	}

	return &meterpb.MeterReadingSupport{
		ResourceSupport: &typespb.ResourceSupport{
			Readable:   true,
			Writable:   !g.readOnly,
			Observable: true,
		},
		UsageUnit:    plan.usageUnit,
		ProducedUnit: plan.producedUnit,
	}, nil
}

// unitPlan returns how to convert and combine member readings.
// Once every member has described itself the plan is fixed for the life of the group.
// Until then a partial plan is used, so a member that's down doesn't stop reads,
// and members are described again in the background at most once every planRetryInterval.
func (g *Group) unitPlan(ctx context.Context) *unitPlan {
	g.planMu.Lock()
	defer g.planMu.Unlock()
	if g.plan == nil {
		// there's nothing to answer with yet, so callers wait for the first resolve
		g.setPlanLocked(g.resolveUnitPlan(ctx, nil))
		return g.plan
	}
	if !g.plan.complete && !g.resolving && g.timeNow().Sub(g.planAt) >= planRetryInterval {
		// A member that's still down takes describeMemberTimeout to fail, reads shouldn't wait for it.
		g.resolving = true
		prev := g.plan
		go func() {
			p := g.resolveUnitPlan(ctx, prev)
			g.planMu.Lock()
			defer g.planMu.Unlock()
			g.resolving = false
			g.setPlanLocked(p)
		}()
	}
	return g.plan
}

func (g *Group) setPlanLocked(p *unitPlan) {
	g.plan = p
	g.planAt = g.timeNow()
	for _, name := range p.noUnits {
		if _, warned := g.warnedEmpty.LoadOrStore(name, struct{}{}); !warned {
			g.log().Warn("meter group member declares no units, assuming it matches the group",
				zap.String("member", name), zap.String("usageUnit", p.usageUnit), zap.String("producedUnit", p.producedUnit))
		}
	}
	if p.err != nil {
		g.log().Warn("meter group can't combine its members", zap.Error(p.err))
	}
}

// resolveUnitPlan describes every member and works out a plan from the answers, keeping prev's units if it has any.
func (g *Group) resolveUnitPlan(ctx context.Context, prev *unitPlan) *unitPlan {
	// The plan is shared by every caller, so one caller giving up shouldn't cut the describes short.
	ctx = context.WithoutCancel(ctx)
	supports := make([]*meterpb.MeterReadingSupport, len(g.names))
	errs := make([]error, len(g.names))
	var wg sync.WaitGroup
	for i, name := range g.names {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(ctx, describeMemberTimeout)
			defer cancel()
			supports[i], errs[i] = g.infoClient.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: name})
		})
	}
	wg.Wait()
	return newUnitPlan(g.names, supports, errs, prev)
}

func (g *Group) timeNow() time.Time {
	if g.now == nil {
		return time.Now()
	}
	return g.now()
}

func (g *Group) log() *zap.Logger {
	if g.logger == nil {
		return zap.NewNop()
	}
	return g.logger
}

// memberScale converts one member's readings into the group's units.
type memberScale struct {
	usage, produced float64
}

var identityScale = memberScale{usage: 1, produced: 1}

// unitPlan is the group's units and how to convert each member's readings into them.
type unitPlan struct {
	usageUnit, producedUnit string
	scales                  []memberScale // indexed like Group.names
	complete                bool          // every member described itself
	err                     error         // members' units can't be reconciled, reads fail with this
	describeErr             error         // set only if no member described itself, the last error seen
	noUnits                 []string      // members that answered but declare no units at all
}

// newUnitPlan works out the group's units from its members' descriptions.
// supports and errs are indexed like names, a member with a non-nil err hasn't answered.
//
// Each quantity's unit is the first unit a member declares for it, spelled as that member spells it.
// Once a unit has been reported, prev carries it forward: a member answering later is converted to it rather than
// changing the group's unit, which would make readings jump by its scale with nothing telling clients.
// Members in another unit of the same family (Wh, kWh, MWh or W, kW) are scaled to match.
// A member declaring no unit is assumed to already match, which is how every member was treated before units were
// checked at all.
// Any other difference can't be combined, and the plan's err says which members differ.
func newUnitPlan(names []string, supports []*meterpb.MeterReadingSupport, errs []error, prev *unitPlan) *unitPlan {
	p := &unitPlan{scales: make([]memberScale, len(names)), complete: true}
	if prev != nil {
		p.usageUnit, p.producedUnit = prev.usageUnit, prev.producedUnit
	}
	answered := 0
	for i := range names {
		p.scales[i] = identityScale
		if errs[i] != nil {
			p.complete = false
			p.describeErr = errs[i]
			continue
		}
		answered++
		if p.usageUnit == "" {
			p.usageUnit = supports[i].GetUsageUnit()
		}
		if p.producedUnit == "" {
			p.producedUnit = supports[i].GetProducedUnit()
		}
	}
	if answered > 0 {
		p.describeErr = nil
	}

	var clashes []string
	for i, name := range names {
		if errs[i] != nil {
			continue // assumed to match until it answers
		}
		usageUnit, producedUnit := supports[i].GetUsageUnit(), supports[i].GetProducedUnit()
		usage, ok := unitFactor(usageUnit, p.usageUnit)
		if !ok {
			clashes = append(clashes, fmt.Sprintf("%s usage is in %q", name, usageUnit))
		}
		produced, ok := unitFactor(producedUnit, p.producedUnit)
		if !ok {
			clashes = append(clashes, fmt.Sprintf("%s produced is in %q", name, producedUnit))
		}
		p.scales[i] = memberScale{usage: usage, produced: produced}
		if usageUnit == "" && producedUnit == "" && (p.usageUnit != "" || p.producedUnit != "") {
			p.noUnits = append(p.noUnits, name)
		}
	}
	if len(clashes) > 0 {
		p.err = status.Errorf(codes.FailedPrecondition, "meter group members can't be combined into usage in %q and produced in %q: %s",
			p.usageUnit, p.producedUnit, strings.Join(clashes, "; "))
	}
	return p
}

// unitFamilies are units that convert to each other, keyed by dbo.UnitName, valued by their size in the family's base
// unit.
var unitFamilies = []map[string]float64{
	{dbo.UnitWattHours: 1, dbo.UnitKilowattHours: 1e3, dbo.UnitMegawattHours: 1e6},
	{dbo.UnitWatts: 1, dbo.UnitKilowatts: 1e3},
}

// unitFactor returns what to multiply a value in unit from by to express it in unit to,
// and false if the two can't be converted.
// An empty from is assumed to already be in to.
func unitFactor(from, to string) (float64, bool) {
	if from == "" || to == "" {
		return 1, true
	}
	cf, ct := dbo.UnitName(from), dbo.UnitName(to)
	if cf != "" && cf == ct {
		return 1, true
	}
	for _, family := range unitFamilies {
		f, okF := family[cf]
		t, okT := family[ct]
		if okF && okT {
			return f / t, true
		}
	}
	// units dbo doesn't know can still match each other
	if strings.EqualFold(strings.TrimSpace(from), strings.TrimSpace(to)) {
		return 1, true
	}
	return 0, false
}

func (g *Group) GetMeterReading(ctx context.Context, request *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	plan := g.unitPlan(ctx)
	if plan.err != nil {
		return nil, plan.err
	}

	allRes := make([]value, len(g.names))
	fns := make([]func(), len(g.names))

	countedErrs := atomic.Int32{}

	for i, name := range g.names {
		request := proto.Clone(request).(*meterpb.GetMeterReadingRequest)
		request.Name = name
		i := i
		fns[i] = func() {
			res, err := g.apiClient.GetMeterReading(ctx, request)

			if err != nil {
				countedErrs.Add(1)

				historyRes, historyErr := g.attemptHistoricalReading(ctx, name, err)

				if historyErr == nil {
					if float32(100*(countedErrs.Load()/int32(len(g.names)))) < g.historyBackupConf.PercentageOfAcceptableErrors {
						// use historical reading if available, within lookback limit, and we haven't exceeded the acceptable error percentage
						res = historyRes
						err = nil
					}
				}
			}

			allRes[i] = value{name: name, val: res, err: err}
		}
	}
	if err := run.InParallel(ctx, run.DefaultConcurrency, fns...); err != nil {
		return nil, err
	}
	return mergeMeterReading(allRes, plan.scales)
}

type nameToError struct {
	sync.Map // meter name key -> error value
}

func newNameToError() *nameToError {
	return &nameToError{}
}

func (n *nameToError) store(name string, v error) {
	n.Store(name, v)
}

func (n *nameToError) countErrs() int {
	count := 0
	n.Range(func(key, value any) bool {
		if value != nil && value.(error) != nil {
			count++
		}
		return true
	})

	return count
}

func (g *Group) PullMeterReadings(request *meterpb.PullMeterReadingsRequest, server meterpb.MeterApi_PullMeterReadingsServer) error {
	if len(g.names) == 0 {
		return status.Error(codes.FailedPrecondition, "zone has no meter names")
	}
	plan := g.unitPlan(server.Context())
	if plan.err != nil {
		return plan.err
	}

	changes := make(chan value)
	defer close(changes)

	// we use a map to track errors per name
	// so that we can calculate the percentage of errors across all names
	// and decide whether to use historical readings as a backup
	// based on the configured acceptable error percentage
	errs := newNameToError()

	group, ctx := errgroup.WithContext(server.Context())
	for _, name := range g.names {
		request := proto.Clone(request).(*meterpb.PullMeterReadingsRequest)
		request.Name = name
		handleResp := func(err error) error {
			errs.store(request.Name, err)
			historyRes, historyErr := g.attemptHistoricalReading(ctx, request.Name, err)

			if historyErr != nil {
				changes <- value{name: request.Name, err: err}
				return err
			}
			if float32(100*(errs.countErrs())/len(g.names)) < g.historyBackupConf.PercentageOfAcceptableErrors {
				// use historical reading if available, within lookback limit, and we haven't exceeded the acceptable error percentage
				changes <- value{name: request.Name, val: historyRes}
				return nil
			}
			return err
		}
		group.Go(func() error {
			return pull.Changes(ctx, pull.NewFetcher(
				func(ctx context.Context, changes chan<- value) error {
					stream, err := g.apiClient.PullMeterReadings(ctx, request)
					if err != nil {
						errs.Store(request.Name, err)
						// if the stream fails pull.Changes should fall back to Get/Polling
						changes <- value{name: request.Name, err: err}
						return err
					}
					errs.Delete(request.Name)
					for {
						res, err := stream.Recv()
						if err != nil {
							if err2 := handleResp(err); err2 != nil {
								return err
							}
							// since the stream has errored and a history reading (if any) has been used
							// we don't delete the original error from the errs map
							continue
						}
						errs.Delete(request.Name)
						for _, change := range res.Changes {
							changes <- value{name: request.Name, val: change.MeterReading}
						}
					}
				},
				func(ctx context.Context, changes chan<- value) error {
					res, err := g.apiClient.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: name, ReadMask: request.ReadMask})
					if err != nil {
						if err2 := handleResp(err); err2 != nil {
							return err
						}
						return nil
					}
					changes <- value{name: request.Name, val: res}
					return nil
				}),
				changes,
			)
		})
	}

	group.Go(func() error {
		// The fetchers send on changes without watching ctx, so once we stop reading they'd block forever and
		// group.Wait would never return. Keep draining until changes is closed after group.Wait.
		defer func() {
			go func() {
				for range changes {
				}
			}()
		}()
		// indexes reports which index in values each name name has
		indexes := make(map[string]int, len(g.names))
		for i, name := range g.names {
			indexes[name] = i
		}
		values := make([]value, len(g.names))

		var last *meterpb.MeterReading
		eq := cmp.Equal(cmp.FloatValueApprox(0, 0.001))
		filter := masks.NewResponseFilter(masks.WithFieldMask(request.ReadMask))

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case change := <-changes:
				values[indexes[change.name]] = change
				if !plan.complete {
					// unitPlan is rate limited, so this is cheap to ask on every change
					plan = g.unitPlan(ctx)
					if plan.err != nil {
						return plan.err
					}
				}
				r, err := mergeMeterReading(values, plan.scales)
				if err != nil {
					continue
				}
				filter.Filter(r)

				// don't send duplicates
				if eq(last, r) {
					continue
				}
				last = r

				err = server.Send(&meterpb.PullMeterReadingsResponse{
					Changes: []*meterpb.PullMeterReadingsResponse_Change{
						{
							Name:         request.Name,
							ChangeTime:   timestamppb.Now(),
							MeterReading: r,
						},
					},
				})
				if err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func (g *Group) attemptHistoricalReading(ctx context.Context, name string, originalErr error) (*meterpb.MeterReading, error) {
	if g.historyBackupConf == nil || g.historyBackupConf.Disabled {
		return nil, originalErr
	}

	// try to get the latest backup reading from the history table if the main read fails
	latest, historyErr := g.historyApiClient.ListMeterReadingHistory(ctx, &meterpb.ListMeterReadingHistoryRequest{
		Name:     name,
		PageSize: 1,
		OrderBy:  "record_time DESC",
	})

	if historyErr != nil || len(latest.GetMeterReadingRecords()) == 0 {
		return nil, originalErr // we forward the original error
	}

	if g.historyBackupConf.LookbackLimit != nil {
		if g.now().After(latest.GetMeterReadingRecords()[0].GetRecordTime().AsTime().Add(g.historyBackupConf.LookbackLimit.Duration)) {
			// latest history record is too old
			return nil, originalErr // we forward the original error
		}
	}

	return latest.GetMeterReadingRecords()[0].MeterReading, nil
}

type value struct {
	name string
	val  *meterpb.MeterReading
	err  error
}

// mergeMeterReading sums all into one reading, scaling each by the entry of scales at the same index.
// History backup readings are per member too, so they scale the same way.
// A nil scales leaves the readings as they are.
func mergeMeterReading(all []value, scales []memberScale) (*meterpb.MeterReading, error) {
	switch len(all) {
	case 0:
		return nil, status.Error(codes.FailedPrecondition, "zone has no meter names")
	default:
		// we need all configured meters to be present and have values for this meter to make any sense.
		valCount := 0
		for _, v := range all {
			if v.err != nil {
				return nil, v.err
			}
			if v.val != nil {
				valCount++
			}
		}
		if valCount < len(all) {
			return nil, status.Errorf(codes.Unavailable, "collecting initial data, please try again soon (%d/%d)", valCount, len(all))
		}

		var usage, produced float64
		for i, v := range all {
			scale := identityScale
			if scales != nil {
				scale = scales[i]
			}
			usage += float64(v.val.Usage) * scale.usage
			produced += float64(v.val.Produced) * scale.produced
		}
		out := &meterpb.MeterReading{
			Usage:    float32(usage),
			Produced: float32(produced),
		}
		out.StartTime = merge.EarliestTimestamp(all, func(v value) *timestamppb.Timestamp {
			if v.err != nil || v.val == nil {
				return nil
			}
			return v.val.StartTime
		})
		out.EndTime = merge.LatestTimestamp(all, func(v value) *timestamppb.Timestamp {
			if v.err != nil || v.val == nil {
				return nil
			}
			return v.val.EndTime
		})
		return out, nil
	}
}
