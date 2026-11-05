package occupancy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/enterleavesensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

func Test_mergeOccupancy(t *testing.T) {
	tests := []struct {
		name    string
		args    []*occupancysensorpb.Occupancy
		want    *occupancysensorpb.Occupancy
		wantErr bool
	}{
		{"empty", nil, nil, true},
		{"one", []*occupancysensorpb.Occupancy{{State: occupancysensorpb.Occupancy_OCCUPIED}}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED}, false},
		{"earliestOccupancy", []*occupancysensorpb.Occupancy{
			{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))}, false},
		{"latestUnoccupied", []*occupancysensorpb.Occupancy{
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(100, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(50, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(80, 0))},
			{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))},
		}, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, StateChangeTime: timestamppb.New(time.Unix(120, 0))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeOccupancy(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("mergeOccupancy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mergeOccupancy() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroup_GetOccupancy_enterLeaveOnly(t *testing.T) {
	tests := []struct {
		name         string
		enter, leave int
		want         *occupancysensorpb.Occupancy
	}{
		{"occupied", 5, 2, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 3}},
		{"empty", 4, 4, &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_UNOCCUPIED, PeopleCount: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			model := enterleavesensorpb.NewModel()
			createEvents(t, model, tt.enter, tt.leave)
			group := &Group{clients: []occupancysensorpb.OccupancySensorApiClient{enterLeaveClient(model, "counter")}}

			got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
			if err != nil {
				t.Fatalf("GetOccupancy: %v", err)
			}
			if got == nil {
				t.Fatal("GetOccupancy returned nil occupancy")
			}
			if diff := cmp.Diff(tt.want, got, protocmp.Transform(), protocmp.IgnoreFields(&occupancysensorpb.Occupancy{}, "state_change_time")); diff != "" {
				t.Errorf("GetOccupancy (-want +got):\n%s", diff)
			}

			// Get and Pull share the enter/leave model, so the first Pull value is what Get returned.
			stream, err := occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, group)).
				PullOccupancy(ctx, &occupancysensorpb.PullOccupancyRequest{Name: "zone"})
			if err != nil {
				t.Fatalf("PullOccupancy: %v", err)
			}
			res, err := stream.Recv()
			if err != nil {
				t.Fatalf("PullOccupancy Recv: %v", err)
			}
			if diff := cmp.Diff(got, res.Changes[0].Occupancy, protocmp.Transform()); diff != "" {
				t.Errorf("first PullOccupancy differs from GetOccupancy (-get +pull):\n%s", diff)
			}
		})
	}
}

func TestGroup_GetOccupancy_errors(t *testing.T) {
	ctx := t.Context()
	wantErr := status.Error(codes.Unavailable, "counter offline")

	t.Run("only source fails", func(t *testing.T) {
		group := &Group{clients: []occupancysensorpb.OccupancySensorApiClient{failingClient(wantErr)}}
		got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
		if !errors.Is(err, wantErr) {
			t.Errorf("GetOccupancy error = %v, want %v", err, wantErr)
		}
		if got != nil {
			t.Errorf("GetOccupancy = %v, want nil", got)
		}
	})

	t.Run("one of several fails", func(t *testing.T) {
		n := node.New("test")
		// "pir-missing" is never announced, so getting it fails.
		pir := occupancysensorpb.NewModel()
		if _, err := pir.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 1}); err != nil {
			t.Fatal(err)
		}
		n.Announce("pir",
			node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(pir))),
		)
		counter := enterleavesensorpb.NewModel()
		createEvents(t, counter, 3, 1)
		group := &Group{
			client:  occupancysensorpb.NewOccupancySensorApiClient(n.ClientConn()),
			names:   []string{"pir", "pir-missing"},
			clients: []occupancysensorpb.OccupancySensorApiClient{enterLeaveClient(counter, "counter")},
			logger:  zap.NewNop(),
		}

		got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
		if err != nil {
			t.Fatalf("GetOccupancy: %v", err)
		}
		want := &occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 3}
		if diff := cmp.Diff(want, got, protocmp.Transform(), protocmp.IgnoreFields(&occupancysensorpb.Occupancy{}, "state_change_time")); diff != "" {
			t.Errorf("GetOccupancy (-want +got):\n%s", diff)
		}
	})

	t.Run("every source fails", func(t *testing.T) {
		n := node.New("test")
		group := &Group{
			client:  occupancysensorpb.NewOccupancySensorApiClient(n.ClientConn()),
			names:   []string{"pir-missing"},
			clients: []occupancysensorpb.OccupancySensorApiClient{failingClient(wantErr)},
		}
		got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
		if err == nil {
			t.Fatalf("GetOccupancy = %v, want an error", got)
		}
	})

	// An unreachable counter must not read as an empty zone, whatever the SLA accepts.
	for name, slaCfg := range map[string]*sla{
		"no SLA":          nil,
		"SLA accepts all": {percentageOfAcceptableFailures: 100, errs: &sync.Map{}},
	} {
		t.Run("only enter leave sensor unreachable, "+name, func(t *testing.T) {
			n := node.New("test")
			server := &enterLeave{
				model:  occupancysensorpb.NewModel(),
				client: enterleavesensorpb.NewEnterLeaveSensorApiClient(n.ClientConn()),
				names:  []string{"counter-missing"},
				sla:    slaCfg,
			}
			group := &Group{clients: []occupancysensorpb.OccupancySensorApiClient{
				occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, server)),
			}}
			got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
			if err == nil {
				t.Fatalf("GetOccupancy = %v, want an error", got)
			}
		})
	}

	t.Run("one of two enter leave sensors unreachable", func(t *testing.T) {
		n := node.New("test")
		counter := enterleavesensorpb.NewModel()
		createEvents(t, counter, 2, 0)
		n.Announce("counter",
			node.HasServer(enterleavesensorpb.RegisterEnterLeaveSensorApiServer, enterleavesensorpb.EnterLeaveSensorApiServer(enterleavesensorpb.NewModelServer(counter))),
		)
		server := &enterLeave{
			model:  occupancysensorpb.NewModel(),
			client: enterleavesensorpb.NewEnterLeaveSensorApiClient(n.ClientConn()),
			names:  []string{"counter", "counter-missing"},
			logger: zap.NewNop(),
		}
		group := &Group{clients: []occupancysensorpb.OccupancySensorApiClient{
			occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, server)),
		}}
		got, err := group.GetOccupancy(ctx, &occupancysensorpb.GetOccupancyRequest{Name: "zone"})
		if err != nil {
			t.Fatalf("GetOccupancy: %v", err)
		}
		if got.GetPeopleCount() != 2 {
			t.Errorf("people count = %d, want 2 from the reachable sensor", got.GetPeopleCount())
		}
	})
}

// enterLeaveClient returns an occupancy client fed by an enter/leave sensor model, as the zone builds it.
func enterLeaveClient(model *enterleavesensorpb.Model, name string) occupancysensorpb.OccupancySensorApiClient {
	elClient := enterleavesensorpb.NewEnterLeaveSensorApiClient(
		wrap.ServerToClient(enterleavesensorpb.EnterLeaveSensorApi_ServiceDesc, enterleavesensorpb.NewModelServer(model)))
	server := &enterLeave{
		model:  occupancysensorpb.NewModel(),
		client: elClient,
		names:  []string{name},
	}
	return occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, server))
}

func createEvents(t *testing.T, model *enterleavesensorpb.Model, enter, leave int) {
	t.Helper()
	for range enter {
		if err := model.CreateEnterLeaveEvent(&enterleavesensorpb.EnterLeaveEvent{Direction: enterleavesensorpb.EnterLeaveEvent_ENTER}); err != nil {
			t.Fatal(err)
		}
	}
	for range leave {
		if err := model.CreateEnterLeaveEvent(&enterleavesensorpb.EnterLeaveEvent{Direction: enterleavesensorpb.EnterLeaveEvent_LEAVE}); err != nil {
			t.Fatal(err)
		}
	}
}

type failingServer struct {
	occupancysensorpb.UnimplementedOccupancySensorApiServer
	err error
}

func (f failingServer) GetOccupancy(context.Context, *occupancysensorpb.GetOccupancyRequest) (*occupancysensorpb.Occupancy, error) {
	return nil, f.err
}

func failingClient(err error) occupancysensorpb.OccupancySensorApiClient {
	return occupancysensorpb.NewOccupancySensorApiClient(wrap.ServerToClient(occupancysensorpb.OccupancySensorApi_ServiceDesc, failingServer{err: err}))
}
