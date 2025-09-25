package health

import (
	"context"
	"fmt"
	"os"

	"github.com/smart-core-os/sc-golang/pkg/wrap"
	"github.com/vanti-dev/sc-bos/internal/health/healthdb"
	"github.com/vanti-dev/sc-bos/internal/health/healthhistory"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb"
)

// ExampleRegistry_history shows how to connect a [Registry] to a history recorder and server.
func ExampleRegistry_history() {
	ctx := context.Background()
	dbFile, err := os.CreateTemp(os.TempDir(), "example.db")
	if err != nil {
		panic(err)
	}
	dbFile.Close() // we don't need it, we just needed the filename
	defer func() { os.Remove(dbFile.Name()) }()

	// records is a database containing health check history
	records, err := healthdb.Open(ctx, dbFile.Name())
	if err != nil {
		panic(err)
	}
	defer records.Close()

	seeder := healthhistory.NewSeeder(records)     // seeders initialise checks from history
	recorder := healthhistory.NewRecorder(records) // recorders save check updates to history
	server := healthhistory.NewServer(records)     // servers expose history over gRPC

	// add an existing record to the db
	_ = recorder.Record(ctx, "device1", &gen.HealthCheck{
		Id:              healthpb.AbsID("example", "paper-level"),
		DisplayName:     "Paper level",
		Description:     "Check the level of the paper in the printer",
		EquipmentImpact: gen.HealthCheck_FUNCTION,
		Normality:       gen.HealthCheck_ABNORMAL,
	})

	registry := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *gen.HealthCheck) *gen.HealthCheck {
			return seeder.Seed(ctx, name, c)
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen.HealthCheck) {
			err := recorder.Record(ctx, name, c)
			if err != nil {
				panic(err)
			}
		}),
	)

	// create the check for device1 owned by "example"
	exampleChecks := registry.ForOwner("example")
	dev1Check, err := exampleChecks.NewFaultCheck("device1", &gen.HealthCheck{
		Id:          "paper-level",
		DisplayName: "Paper level",
	})
	if err != nil {
		panic(err)
	}
	defer dev1Check.Dispose()

	// perform a check
	dev1Check.ClearFaults() // all good now

	// use the history api to get the check results
	client := gen.NewHealthHistoryClient(wrap.ServerToClient(gen.HealthHistory_ServiceDesc, server))
	histResp, err := client.ListHealthCheckHistory(ctx, &gen.ListHealthCheckHistoryRequest{
		Name: "device1",
		Id:   healthpb.AbsID("example", "paper-level"),
	})
	if err != nil {
		panic(err)
	}
	for i, rec := range histResp.GetHealthCheckRecords() {
		fmt.Printf("Record %d: state=%v\n", i, rec.GetHealthCheck().GetNormality())
	}

	// Output:
	// Record 0: state=ABNORMAL
	// Record 1: state=NORMAL
}
