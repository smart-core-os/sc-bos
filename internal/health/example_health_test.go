package health

import (
	"context"
	"fmt"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

func ExampleRegistry_healthApi() {
	// The registry callbacks can be called from multiple goroutines.
	// While the node and model types are concurrency-safe, the announced map is not.
	var mu sync.Mutex
	n := node.New("example-node")
	type device struct {
		undo node.Undo
		m    *healthpb.Model
	}
	announced := make(map[string]device)

	registry := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *gen_healthpb.HealthCheck) *gen_healthpb.HealthCheck {
			mu.Lock()
			defer mu.Unlock()
			m := healthpb.NewModel()
			undo := n.Announce(name, node.HasTrait(healthpb.TraitName, node.WithClients(gen_healthpb.WrapApi(healthpb.NewModelServer(m)))))
			announced[name] = device{undo: undo, m: m}
			return nil
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen_healthpb.HealthCheck) {
			mu.Lock()
			defer mu.Unlock()
			a := announced[name]
			_, err := a.m.CreateHealthCheck(c)
			if err != nil {
				panic(fmt.Errorf("failed to create health check: %w", err))
			}
		}),
		healthpb.WithOnCheckDelete(func(name, id string) {
			mu.Lock()
			defer mu.Unlock()
			a, ok := announced[name]
			if !ok {
				panic(fmt.Errorf("can't delete unknown name: %q", name)) // shouldn't happen
			}
			err := a.m.DeleteHealthCheck(id)
			if err != nil {
				panic(fmt.Errorf("can't delete health check %q.%q: %w", name, id, err))
			}
		}),
		healthpb.WithOnNameDelete(func(name string) {
			mu.Lock()
			defer mu.Unlock()
			a, ok := announced[name]
			if !ok {
				return
			}
			a.undo()
			delete(announced, name)
		}),
	)

	// set up checks for the example
	exampleChecks := registry.ForOwner("example")
	// create a health check for device1
	errCheck, err := exampleChecks.NewFaultCheck("device1", &gen_healthpb.HealthCheck{})
	if err != nil {
		panic(err)
	}
	defer errCheck.Dispose()
	// report on the health of the device
	errCheck.SetFault(&gen_healthpb.HealthCheck_Error{SummaryText: "needs filter change"})

	// later, use the HealthApi to query the health
	client := gen_healthpb.NewHealthApiClient(n.ClientConn())
	resp, err := client.ListHealthChecks(context.TODO(), &gen_healthpb.ListHealthChecksRequest{Name: "device1"})
	if err != nil {
		panic(err)
	}
	for _, c := range resp.HealthChecks {
		fmt.Printf("Check %q is %v: %v\n", c.GetId(), c.GetNormality(), c.GetFaults().GetCurrentFaults()[0].GetSummaryText())
	}
	// Output: Check "example" is ABNORMAL: needs filter change
}
