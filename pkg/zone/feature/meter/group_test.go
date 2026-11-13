package meter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/historypb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-bos/pkg/zone/feature/meter/config"
)

func TestGroup_PullMeterReadings(t *testing.T) {
	type meter struct {
		events []event
		store  *memstore.Store
	}

	tests := []struct {
		name     string
		meters   map[string]*meter
		wantErrs []error // the first err terminates further processing in sequential PullMeterReadings calls
	}{
		{
			name: "meter with error event",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{err: errors.New("simulated error")},
						{usage: 30},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "multiple meters, one with error event",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{usage: 20},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{err: errors.New("simulated error")},
						{usage: 300},
					},
				},
			},
			wantErrs: nil,
		},
		{
			name: "multiple meters, both with error events",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{err: errors.New("simulated error")},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{err: errors.New("simulated error")},
						{usage: 300},
					},
				},
			},
			wantErrs: []error{errors.New("simulated error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			n := node.New(tt.name)
			timer := &timeInterceptor{
				base:    time.Date(2022, 1, 1, 12, 0, 13, 0, time.UTC), // random base time
				step:    10 * time.Second,
				current: 0,
			}

			m := &meterModelInterceptor{
				events:       make(map[string][]event),
				meterToIndex: make(map[string]int),
			}

			for mtr, mod := range tt.meters {
				m.events[mtr] = mod.events
				mod.store = memstore.New(memstore.WithNow(time.Now))
				m.meterToIndex[mtr] = 0
			}

			group := &Group{
				apiClient:        meterpb.NewMeterApiClient(n.ClientConn()),
				infoClient:       meterpb.NewMeterInfoClient(n.ClientConn()),
				historyApiClient: meterpb.NewMeterHistoryClient(n.ClientConn()),
				names:            maps.Keys(tt.meters),
				historyBackupConf: &config.HistoryBackup{
					Disabled:                     false,
					LookbackLimit:                &jsontypes.Duration{Duration: 3 * time.Second}, // terminate after 3s
					PercentageOfAcceptableErrors: float32(101.0),                                 // all errors acceptable for PullMeterReadings
				},
				now: timer.now,
			}

			for meter := range tt.meters {
				n.Announce(meter,
					node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(m)),
					node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(m)),
					node.HasServer(meterpb.RegisterMeterHistoryServer, meterpb.MeterHistoryServer(historypb.NewMeterServer(tt.meters[meter].store))),
					node.HasTrait(meterpb.TraitName),
				)
			}

			for idx := range tt.meters[maps.Keys(tt.meters)[0]].events {
				for meter := range tt.meters {
					if tt.meters[meter].events[idx].err == nil {
						rec, err := proto.Marshal(&meterpb.MeterReading{
							Usage: tt.meters[meter].events[idx].usage,
						})
						if err != nil {
							panic(err)
						}

						_, err = tt.meters[meter].store.Append(ctx, rec)
						if err != nil {
							panic(err)
						}
					}
				}
			}

			res := make(chan *meterpb.PullMeterReadingsResponse)

			go func() {
				err := group.PullMeterReadings(&meterpb.PullMeterReadingsRequest{Name: "group"}, &mockPullServer{
					ctx:     ctx,
					changes: res,
				})

				if err != nil && len(tt.wantErrs) > 0 && tt.wantErrs[0] == nil {
					t.Errorf("PullMeterReadings() unexpected error: %v", err)
					return
				}
				if err == nil && len(tt.wantErrs) > 0 && tt.wantErrs[0] != nil {
					t.Errorf("PullMeterReadings() expected error: %v", tt.wantErrs[0])
					return
				}
			}()

			for count := 0; count < len(tt.meters[maps.Keys(tt.meters)[0]].events)-1; count++ {
				_, more := <-res

				if !more {
					break
				}
			}
		})
	}
}

func TestGroup_GetMeterReading(t *testing.T) {
	type meter struct {
		events []event
		store  *memstore.Store
	}

	tests := []struct {
		name             string
		meters           map[string]*meter
		acceptableErrors float32
		want             []float32
		wantErrs         []error // the first err terminates further processing in sequential GetMeterReading calls
	}{
		{
			name: "single meter, single event",
			meters: map[string]*meter{
				"meter1": {
					events: []event{{usage: 10}},
				},
			},
			want:     []float32{10},
			wantErrs: nil,
		},
		{
			name: "single meter, multiple events",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{usage: 20},
						{usage: 30},
					},
				},
			},
			want:     []float32{10, 20, 30},
			wantErrs: nil,
		},
		{
			name: "multiple meters, single event each",
			meters: map[string]*meter{
				"meter1": {
					events: []event{{usage: 10}},
				},
				"meter2": {
					events: []event{{usage: 20}},
				},
			},
			want:     []float32{30},
			wantErrs: nil,
		},
		{
			name: "multiple meters, multiple events each",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{usage: 20},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{usage: 200},
						{usage: 300},
					},
				},
			},
			want:     []float32{110, 220, 330},
			wantErrs: nil,
		},
		{
			name: "meter with error event",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{err: errors.New("simulated error")},
						{usage: 30},
					},
				},
			},
			want:             []float32{10, 10, 30}, // index 0 gets replayed on error at index 1
			wantErrs:         nil,
			acceptableErrors: float32(110.0),
		},
		{
			name: "meter with all error events",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{err: errors.New("simulated error")},
						{err: errors.New("simulated error")},
						{err: errors.New("simulated error")},
					},
				},
			},
			wantErrs:         []error{errors.New("simulated error")},
			acceptableErrors: float32(50.0),
		},
		{
			name: "multiple meters, one with error event",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{usage: 20},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{err: errors.New("simulated error")},
						{usage: 300},
					},
				},
			},
			want:             []float32{110, 120, 330}, // index 0 gets replayed on error at index 1 for meter2
			wantErrs:         nil,
			acceptableErrors: float32(50.0),
		},
		{
			name: "multiple meters, one with all error events",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{usage: 20},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{err: errors.New("simulated error")},
						{err: errors.New("simulated error")},
						{err: errors.New("simulated error")},
					},
				},
			},
			wantErrs:         []error{errors.New("simulated error")},
			acceptableErrors: float32(99.0),
		},
		{
			name: "multiple meters, both with error events",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{err: errors.New("simulated error")},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{err: errors.New("simulated error")},
						{usage: 300},
					},
				},
			},
			want:             []float32{110},
			wantErrs:         []error{nil, errors.New("simulated error")},
			acceptableErrors: float32(10.0),
		},
		{
			name: "low acceptable error percentage",
			meters: map[string]*meter{
				"meter1": {
					events: []event{
						{usage: 10},
						{err: errors.New("simulated error")},
						{usage: 30},
					},
				},
				"meter2": {
					events: []event{
						{usage: 100},
						{usage: 200},
						{usage: 300},
					},
				},
			},
			wantErrs:         []error{errors.New("simulated error")},
			acceptableErrors: float32(33.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			n := node.New(tt.name)
			timer := &timeInterceptor{
				base:    time.Date(2022, 1, 1, 12, 0, 13, 0, time.UTC), // random base time
				step:    10 * time.Second,
				current: 0,
			}

			m := &meterModelInterceptor{
				events:       make(map[string][]event),
				meterToIndex: make(map[string]int),
			}

			for mtr, mod := range tt.meters {
				m.events[mtr] = mod.events
				mod.store = memstore.New(memstore.WithNow(timer.now))
			}

			group := &Group{
				apiClient:        meterpb.NewMeterApiClient(n.ClientConn()),
				infoClient:       meterpb.NewMeterInfoClient(n.ClientConn()),
				historyApiClient: meterpb.NewMeterHistoryClient(n.ClientConn()),
				names:            maps.Keys(tt.meters),
				historyBackupConf: &config.HistoryBackup{
					Disabled:                     false,
					LookbackLimit:                &jsontypes.Duration{Duration: 3 * time.Minute},
					PercentageOfAcceptableErrors: tt.acceptableErrors,
				},
				now: timer.now,
			}

			for meter := range tt.meters {
				m.meterToIndex[meter] = 0
				n.Announce(meter,
					node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(m)),
					node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(m)),
					node.HasServer(meterpb.RegisterMeterHistoryServer, meterpb.MeterHistoryServer(historypb.NewMeterServer(tt.meters[meter].store))),
					node.HasTrait(meterpb.TraitName),
				)
			}

			for idx, want := range tt.want {
				for meter := range tt.meters {
					if tt.meters[meter].events[idx].err == nil {
						rec, err := proto.Marshal(&meterpb.MeterReading{
							Usage: tt.meters[meter].events[idx].usage,
						})
						if err != nil {
							panic(err)
						}

						_, err = tt.meters[meter].store.Append(ctx, rec)
						if err != nil {
							panic(err)
						}
					}
				}
				got, err := group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
				if len(tt.wantErrs) <= idx && err != nil {
					t.Errorf("GetMeterReading() unexpected error: %v", err)
					return
				}
				if len(tt.wantErrs) > idx && err == nil {
					assert.Equal(t, tt.wantErrs[idx], err, tt.name)
				}
				if err != nil && tt.wantErrs[idx] != nil {
					return
				}

				if diff := cmp.Diff(want, got.GetUsage(), protocmp.Transform()); diff != "" {
					t.Errorf("GetMeterReading() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func Test_mergeMeterReading(t *testing.T) {
	err := errors.New("expected error")
	reading := func(val float32) *meterpb.MeterReading {
		return &meterpb.MeterReading{
			Usage: val,
		}
	}
	both := func(usage, produced float32) *meterpb.MeterReading {
		return &meterpb.MeterReading{Usage: usage, Produced: produced}
	}

	tests := []struct {
		name    string // defaults to a summary of in
		in      []value
		scales  []memberScale
		want    *meterpb.MeterReading
		wantErr bool
	}{
		// simple cases
		{in: nil, wantErr: true},
		{in: []value{{}}, wantErr: true},
		{in: []value{{err: err}}, wantErr: true},
		{in: []value{{err: err, val: reading(10)}}, wantErr: true},
		// all present
		{in: []value{{val: reading(10)}}, want: reading(10)},
		{in: []value{{val: reading(10)}, {val: reading(20)}}, want: reading(30)},
		// some missing
		{in: []value{{val: reading(10)}, {}}, wantErr: true},
		{in: []value{{}, {val: reading(10)}}, wantErr: true},
		// some errors
		{in: []value{{err: err}, {}}, wantErr: true},
		{in: []value{{}, {err: err}}, wantErr: true},
		// mixed missing and error
		{in: []value{{}, {err: err}, {val: reading(10)}}, wantErr: true},
		// produced (SCB-1428)
		{name: "produced summed", in: []value{{val: both(10, 1)}, {val: both(20, 2)}}, want: both(30, 3)},
		{name: "produced only member", in: []value{{val: both(10, 0)}, {val: both(0, 5)}}, want: both(10, 5)},
		// scaled (SCB-1445)
		{
			name:   "Wh member scaled to kWh",
			in:     []value{{val: reading(10)}, {val: reading(2000)}},
			scales: []memberScale{identityScale, {usage: 1e-3, produced: 1e-3}},
			want:   reading(12),
		},
		{
			name:   "usage and produced scaled independently",
			in:     []value{{val: both(1, 1)}, {val: both(1, 1)}},
			scales: []memberScale{identityScale, {usage: 1000, produced: 1e-3}},
			want:   both(1001, 1.001),
		},
	}
	for _, tt := range tests {
		name := tt.name
		if name == "" && len(tt.in) == 0 {
			name = "empty"
		} else if name == "" {
			var names []string
			for _, v := range tt.in {
				switch {
				case v.err != nil && v.val != nil:
					names = append(names, "err+val")
				case v.err != nil:
					names = append(names, "err")
				case v.val == nil:
					names = append(names, "nil")
				default:
					names = append(names, fmt.Sprintf("%v", v.val.Usage))
				}
			}
			name = strings.Join(names, ",")
		}
		t.Run(name, func(t *testing.T) {
			got, err := mergeMeterReading(tt.in, tt.scales)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeMeterReading() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform(), cmpopts.EquateApprox(0, 1e-4)); diff != "" {
				t.Errorf("mergeMeterReading() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_newUnitPlan(t *testing.T) {
	support := func(usage, produced string) *meterpb.MeterReadingSupport {
		return &meterpb.MeterReadingSupport{UsageUnit: usage, ProducedUnit: produced}
	}
	offline := status.Error(codes.Unavailable, "offline")

	type member struct {
		name    string
		support *meterpb.MeterReadingSupport
		err     error
	}
	tests := []struct {
		name             string
		members          []member
		prevUsage        string // the usage unit an earlier, partial plan reported
		wantUsage        string
		wantProduced     string
		wantScales       []memberScale
		wantComplete     bool
		wantErr          codes.Code // OK for no clash
		wantDescribeErr  bool
		wantNoUnits      []string
		wantErrMentioned []string // members the clash error names
	}{
		{
			name:         "all kWh",
			members:      []member{{"a", support("kWh", ""), nil}, {"b", support("kWh", ""), nil}},
			wantUsage:    "kWh",
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
		},
		{
			name:         "kWh and Wh",
			members:      []member{{"a", support("kWh", ""), nil}, {"b", support("Wh", ""), nil}},
			wantUsage:    "kWh",
			wantScales:   []memberScale{identityScale, {usage: 1e-3, produced: 1}},
			wantComplete: true,
		},
		{
			name:         "group unit spelled as first member",
			members:      []member{{"a", support("watt_hours", ""), nil}, {"b", support("MWh", ""), nil}},
			wantUsage:    "watt_hours",
			wantScales:   []memberScale{identityScale, {usage: 1e6, produced: 1}},
			wantComplete: true,
		},
		{
			name:         "same unit different spelling",
			members:      []member{{"a", support("kWh", ""), nil}, {"b", support("kilowatt_hours", ""), nil}},
			wantUsage:    "kWh",
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
		},
		{
			name:         "unknown units that match",
			members:      []member{{"a", support("therms", ""), nil}, {"b", support("Therms", ""), nil}},
			wantUsage:    "therms",
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
		},
		{
			name:         "kWh and empty",
			members:      []member{{"a", support("kWh", ""), nil}, {"b", support("", ""), nil}},
			wantUsage:    "kWh",
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
			wantNoUnits:  []string{"b"},
		},
		{
			name:         "empty first takes unit from later member",
			members:      []member{{"a", support("", ""), nil}, {"b", support("Wh", ""), nil}},
			wantUsage:    "Wh",
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
			wantNoUnits:  []string{"a"},
		},
		{
			name:         "all empty",
			members:      []member{{"a", support("", ""), nil}, {"b", support("", ""), nil}},
			wantScales:   []memberScale{identityScale, identityScale},
			wantComplete: true,
		},
		{
			name:             "kWh and m³ clash",
			members:          []member{{"a", support("kWh", ""), nil}, {"b", support("m³", ""), nil}},
			wantUsage:        "kWh",
			wantComplete:     true,
			wantErr:          codes.FailedPrecondition,
			wantErrMentioned: []string{"b", "m³"},
		},
		{
			name:             "energy and power clash",
			members:          []member{{"a", support("kWh", ""), nil}, {"b", support("kW", ""), nil}},
			wantUsage:        "kWh",
			wantComplete:     true,
			wantErr:          codes.FailedPrecondition,
			wantErrMentioned: []string{"b", "kW"},
		},
		{
			name: "produced only member",
			members: []member{
				{"consumer", support("kWh", ""), nil},
				{"pv", support("", "Wh"), nil},
				{"both", support("kWh", "kWh"), nil},
			},
			wantUsage:    "kWh",
			wantProduced: "Wh",
			wantScales:   []memberScale{identityScale, identityScale, {usage: 1, produced: 1000}},
			wantComplete: true,
		},
		{
			name:             "produced clash",
			members:          []member{{"a", support("kWh", "kWh"), nil}, {"b", support("kWh", "L"), nil}},
			wantUsage:        "kWh",
			wantProduced:     "kWh",
			wantComplete:     true,
			wantErr:          codes.FailedPrecondition,
			wantErrMentioned: []string{"b produced", "L"},
		},
		{
			name:         "member offline",
			members:      []member{{"a", nil, offline}, {"b", support("Wh", ""), nil}, {"c", support("kWh", ""), nil}},
			wantUsage:    "Wh",
			wantScales:   []memberScale{identityScale, identityScale, {usage: 1000, produced: 1}},
			wantComplete: false,
		},
		{
			name:         "earlier unit kept when the first member answers late",
			members:      []member{{"a", support("Wh", ""), nil}, {"b", support("kWh", ""), nil}},
			prevUsage:    "kWh",
			wantUsage:    "kWh",
			wantScales:   []memberScale{{usage: 1e-3, produced: 1}, identityScale},
			wantComplete: true,
		},
		{
			name:            "all offline",
			members:         []member{{"a", nil, offline}, {"b", nil, offline}},
			wantScales:      []memberScale{identityScale, identityScale},
			wantComplete:    false,
			wantDescribeErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var names []string
			var supports []*meterpb.MeterReadingSupport
			var errs []error
			for _, m := range tt.members {
				names = append(names, m.name)
				supports = append(supports, m.support)
				errs = append(errs, m.err)
			}
			var prev *unitPlan
			if tt.prevUsage != "" {
				prev = &unitPlan{usageUnit: tt.prevUsage}
			}
			got := newUnitPlan(names, supports, errs, prev)
			assert.Equal(t, tt.wantUsage, got.usageUnit, "usageUnit")
			assert.Equal(t, tt.wantProduced, got.producedUnit, "producedUnit")
			assert.Equal(t, tt.wantComplete, got.complete, "complete")
			assert.Equal(t, tt.wantNoUnits, got.noUnits, "noUnits")
			assert.Equal(t, tt.wantDescribeErr, got.describeErr != nil, "describeErr = %v", got.describeErr)
			assert.Equal(t, tt.wantErr, status.Code(got.err), "err = %v", got.err)
			for _, s := range tt.wantErrMentioned {
				assert.ErrorContains(t, got.err, s)
			}
			if tt.wantErr != codes.OK {
				return
			}
			if diff := cmp.Diff(tt.wantScales, got.scales, cmp.AllowUnexported(memberScale{}), cmpopts.EquateApprox(0, 1e-9)); diff != "" {
				t.Errorf("scales mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// newUnitsTestGroup announces each meter on a node backed by m and returns a Group over them, in the given order.
func newUnitsTestGroup(t *testing.T, m *meterModelInterceptor, names ...string) *Group {
	t.Helper()
	n := node.New(t.Name())
	m.meterToIndex = make(map[string]int)
	for _, name := range names {
		m.meterToIndex[name] = 0
		n.Announce(name,
			node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(m)),
			node.HasServer(meterpb.RegisterMeterInfoServer, meterpb.MeterInfoServer(m)),
			node.HasTrait(meterpb.TraitName),
		)
	}
	return &Group{
		apiClient:         meterpb.NewMeterApiClient(n.ClientConn()),
		infoClient:        meterpb.NewMeterInfoClient(n.ClientConn()),
		historyApiClient:  meterpb.NewMeterHistoryClient(n.ClientConn()),
		names:             names,
		historyBackupConf: &config.HistoryBackup{Disabled: true},
		now:               time.Now,
	}
}

func TestGroup_mixedUnits(t *testing.T) {
	t.Run("Wh and kWh sum in kWh", func(t *testing.T) {
		m := &meterModelInterceptor{
			events: map[string][]event{
				"kwh": {{usage: 10, produced: 1}},
				"wh":  {{usage: 2500}},
				"pv":  {{produced: 3000}},
			},
			support: map[string]*meterpb.MeterReadingSupport{
				"kwh": {UsageUnit: "kWh", ProducedUnit: "kWh"},
				"wh":  {UsageUnit: "Wh"},
				"pv":  {ProducedUnit: "Wh"},
			},
		}
		group := newUnitsTestGroup(t, m, "kwh", "wh", "pv")
		ctx := t.Context()

		desc, err := group.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: "group"})
		if err != nil {
			t.Fatalf("DescribeMeterReading: %v", err)
		}
		assert.Equal(t, "kWh", desc.UsageUnit)
		assert.Equal(t, "kWh", desc.ProducedUnit)

		got, err := group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
		if err != nil {
			t.Fatalf("GetMeterReading: %v", err)
		}
		assert.InDelta(t, 12.5, got.Usage, 1e-4) // 10 kWh + 2500 Wh
		assert.InDelta(t, 4, got.Produced, 1e-4) // 1 kWh + 3000 Wh
	})

	t.Run("m³ member fails reads", func(t *testing.T) {
		m := &meterModelInterceptor{
			events: map[string][]event{
				"kwh":   {{usage: 10}},
				"water": {{usage: 3}},
			},
			support: map[string]*meterpb.MeterReadingSupport{
				"kwh":   {UsageUnit: "kWh"},
				"water": {UsageUnit: "m³"},
			},
		}
		group := newUnitsTestGroup(t, m, "kwh", "water")
		ctx := t.Context()

		_, err := group.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: "group"})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err), "Describe err = %v", err)
		_, err = group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err), "Get err = %v", err)
		err = group.PullMeterReadings(&meterpb.PullMeterReadingsRequest{Name: "group"}, &mockPullServer{ctx: ctx})
		assert.Equal(t, codes.FailedPrecondition, status.Code(err), "Pull err = %v", err)
	})

	t.Run("offline member doesn't block reads", func(t *testing.T) {
		m := &meterModelInterceptor{
			events: map[string][]event{
				"kwh":     {{usage: 10}},
				"offline": {{usage: 5}},
			},
			describeErr: map[string]error{"offline": status.Error(codes.Unavailable, "offline")},
		}
		group := newUnitsTestGroup(t, m, "kwh", "offline")
		ctx := t.Context()

		got, err := group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
		if err != nil {
			t.Fatalf("GetMeterReading: %v", err)
		}
		assert.InDelta(t, 15, got.Usage, 1e-4)

		// the member comes back in Wh, the plan only notices after planRetryInterval
		m.mtx.Lock()
		delete(m.describeErr, "offline")
		m.support = map[string]*meterpb.MeterReadingSupport{"offline": {UsageUnit: "Wh"}}
		m.mtx.Unlock()
		got, err = group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
		if err != nil {
			t.Fatalf("GetMeterReading: %v", err)
		}
		assert.InDelta(t, 15, got.Usage, 1e-4, "plan re-resolved before planRetryInterval")

		now := time.Now().Add(planRetryInterval)
		group.now = func() time.Time { return now }
		// re-resolving happens in the background, so the read that starts it still uses the old plan
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			got, err := group.GetMeterReading(ctx, &meterpb.GetMeterReadingRequest{Name: "group"})
			if assert.NoError(c, err) {
				assert.InDelta(c, 10.005, got.Usage, 1e-4)
			}
		}, 5*time.Second, 10*time.Millisecond)
	})
}

// A Pull that starts before every member has described itself picks up their units as they answer.
func TestGroup_PullMeterReadings_unitsResolveMidStream(t *testing.T) {
	tests := []struct {
		name      string
		lateUnit  string
		wantUsage float32 // once the late member's unit is known
		wantCode  codes.Code
	}{
		{name: "late member scaled", lateUnit: "Wh", wantUsage: 12},
		{name: "late member clashes", lateUnit: "m³", wantCode: codes.FailedPrecondition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &meterModelInterceptor{
				events: map[string][]event{
					"kwh":  {{usage: 10}},
					"late": {{usage: 2000}},
				},
				describeErr: map[string]error{"late": status.Error(codes.Unavailable, "offline")},
			}
			group := newUnitsTestGroup(t, m, "kwh", "late")
			var offset atomic.Int64 // Pull reads now on its own goroutines
			group.now = func() time.Time { return time.Now().Add(time.Duration(offset.Load())) }

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			changes := make(chan *meterpb.PullMeterReadingsResponse)
			pullErr := make(chan error, 1)
			go func() {
				pullErr <- group.PullMeterReadings(&meterpb.PullMeterReadingsRequest{Name: "group"}, &mockPullServer{ctx: ctx, changes: changes})
			}()

			first := <-changes
			assert.InDelta(t, 2010, first.Changes[0].MeterReading.Usage, 1e-4, "late member assumed to match before it answers")

			m.mtx.Lock()
			delete(m.describeErr, "late")
			m.support = map[string]*meterpb.MeterReadingSupport{"late": {UsageUnit: tt.lateUnit}}
			m.mtx.Unlock()
			offset.Store(int64(planRetryInterval))

			timeout := time.After(5 * time.Second)
			for {
				select {
				case res := <-changes:
					if tt.wantCode == codes.OK && math.Abs(float64(res.Changes[0].MeterReading.Usage-tt.wantUsage)) < 1e-4 {
						return
					}
				case err := <-pullErr:
					if tt.wantCode == codes.OK {
						t.Fatalf("Pull ended: %v", err)
					}
					assert.Equal(t, tt.wantCode, status.Code(err), "Pull err = %v", err)
					return
				case <-timeout:
					t.Fatal("timed out waiting for the late member's unit to apply")
				}
			}
		})
	}
}
