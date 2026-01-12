package meter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/historypb"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/history/memstore"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_meterpb "github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
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
				apiClient:        gen_meterpb.NewMeterApiClient(n.ClientConn()),
				historyApiClient: gen_meterpb.NewMeterHistoryClient(n.ClientConn()),
				names:            maps.Keys(tt.meters),
				historyBackupConf: &config.HistoryBackup{
					Disabled:                     false,
					LookbackLimit:                &jsontypes.Duration{Duration: 3 * time.Second}, // terminate after 3s
					PercentageOfAcceptableErrors: float32(101.0),                                 // all errors acceptable for PullMeterReadings
				},
				now: timer.now,
			}

			for meter := range tt.meters {
				n.Announce(meter, node.HasTrait(
					meterpb.TraitName,
					node.WithClients(
						gen_meterpb.WrapApi(m),
						gen_meterpb.WrapInfo(m),
						gen_meterpb.WrapHistory(historypb.NewMeterServer(tt.meters[meter].store)),
					),
				))
			}

			for idx := range tt.meters[maps.Keys(tt.meters)[0]].events {
				for meter := range tt.meters {
					if tt.meters[meter].events[idx].err == nil {
						rec, err := proto.Marshal(&gen_meterpb.MeterReading{
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

			res := make(chan *gen_meterpb.PullMeterReadingsResponse)

			go func() {
				err := group.PullMeterReadings(&gen_meterpb.PullMeterReadingsRequest{Name: "group"}, &mockPullServer{
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
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
				apiClient:        gen_meterpb.NewMeterApiClient(n.ClientConn()),
				historyApiClient: gen_meterpb.NewMeterHistoryClient(n.ClientConn()),
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
				n.Announce(meter, node.HasTrait(
					meterpb.TraitName,
					node.WithClients(
						gen_meterpb.WrapApi(m),
						gen_meterpb.WrapInfo(m),
						gen_meterpb.WrapHistory(historypb.NewMeterServer(tt.meters[meter].store)),
					),
				))
			}

			for idx, want := range tt.want {
				for meter := range tt.meters {
					if tt.meters[meter].events[idx].err == nil {
						rec, err := proto.Marshal(&gen_meterpb.MeterReading{
							Usage: tt.meters[meter].events[idx].usage,
						})
						if err != nil {
							panic(err)
						}

						_, err = tt.meters[meter].store.Append(nil, rec)
						if err != nil {
							panic(err)
						}
					}
				}
				got, err := group.GetMeterReading(ctx, &gen_meterpb.GetMeterReadingRequest{Name: "group"})
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
	reading := func(val float32) *gen_meterpb.MeterReading {
		return &gen_meterpb.MeterReading{
			Usage: val,
		}
	}

	tests := []struct {
		in      []value
		want    *gen_meterpb.MeterReading
		wantErr bool
	}{
		// simple cases
		{nil, nil, true},
		{[]value{{}}, nil, true},
		{[]value{{err: err}}, nil, true},
		{[]value{{err: err, val: reading(10)}}, nil, true},
		// all present
		{[]value{{val: reading(10)}}, reading(10), false},
		{[]value{{val: reading(10)}, {val: reading(20)}}, reading(30), false},
		// some missing
		{[]value{{val: reading(10)}, {}}, nil, true},
		{[]value{{}, {val: reading(10)}}, nil, true},
		// some errors
		{[]value{{err: err}, {}}, nil, true},
		{[]value{{}, {err: err}}, nil, true},
		// mixed missing and error
		{[]value{{}, {err: err}, {val: reading(10)}}, nil, true},
	}
	for _, tt := range tests {
		name := ""
		if len(tt.in) == 0 {
			name = "empty"
		} else {
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
			got, err := mergeMeterReading(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeMeterReading() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeMeterReading() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
