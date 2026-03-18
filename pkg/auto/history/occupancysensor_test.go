package history

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	occupancysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

func Test_automation_collectOccupancyChanges(t *testing.T) {
	model := occupancysensorpb2.NewModel()
	// n is used as the clienter and announcer in the automation
	n := node.New("test")
	n.Announce("device",
		node.HasTrait(trait.OccupancySensor),
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer, occupancysensorpb.OccupancySensorApiServer(occupancysensorpb2.NewModelServer(model))),
	)

	collector := &automation{
		clients:   n,
		announcer: node.NewReplaceAnnouncer(n),
		logger:    zap.NewNop(),
	}

	payloads := make(chan []byte, 5)
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go func() {
		collector.collectOccupancyChanges(ctx, config.Source{Name: "device"}, payloads)
	}()

	if err := chans.IsEmptyWithin(payloads, time.Second); err != nil {
		t.Fatal(err)
	}

	if _, err := model.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED}); err != nil {
		t.Fatal(err)
	}
	payload, err := chans.RecvWithin(payloads, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	msg := &occupancysensorpb.Occupancy{}
	err = proto.Unmarshal(payload, msg)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED}, msg, protocmp.Transform()); diff != "" {
		t.Fatalf("payload (-want,+got)\n%s", diff)
	}
}
