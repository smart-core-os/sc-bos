package history

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/occupancysensorpb"
)

func Test_automation_collectOccupancyChanges(t *testing.T) {
	model := occupancysensorpb.NewModel()
	// n is used as the clienter and announcer in the automation
	n := node.New("test")
	n.Announce("device",
		node.HasTrait(trait.OccupancySensor),
		node.HasServer(traits.RegisterOccupancySensorApiServer, traits.OccupancySensorApiServer(occupancysensorpb.NewModelServer(model))),
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

	if _, err := model.SetOccupancy(&traits.Occupancy{State: traits.Occupancy_OCCUPIED}); err != nil {
		t.Fatal(err)
	}
	payload, err := chans.RecvWithin(payloads, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	msg := &traits.Occupancy{}
	err = proto.Unmarshal(payload, msg)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(&traits.Occupancy{State: traits.Occupancy_OCCUPIED}, msg, protocmp.Transform()); diff != "" {
		t.Fatalf("payload (-want,+got)\n%s", diff)
	}
}
