package health

import (
	"context"
	"fmt"
	"os"

	"github.com/smart-core-os/sc-bos/internal/health/healthdb"
	"github.com/smart-core-os/sc-bos/internal/health/healthhistory"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
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
	_ = recorder.Record(ctx, "device1", &gen_healthpb.HealthCheck{
		Id:              healthpb.AbsID("example", "paper-level"),
		DisplayName:     "Paper level",
		Description:     "Check the level of the paper in the printer",
		EquipmentImpact: gen_healthpb.HealthCheck_FUNCTION,
		Normality:       gen_healthpb.HealthCheck_ABNORMAL,
	})

	registry := healthpb.NewRegistry(
		healthpb.WithOnCheckCreate(func(name string, c *gen_healthpb.HealthCheck) *gen_healthpb.HealthCheck {
			return seeder.Seed(ctx, name, c)
		}),
		healthpb.WithOnCheckUpdate(func(name string, c *gen_healthpb.HealthCheck) {
			err := recorder.Record(ctx, name, c)
			if err != nil {
				panic(err)
			}
		}),
	)

	// create the check for device1 owned by "example"
	exampleChecks := registry.ForOwner("example")
	dev1Check, err := exampleChecks.NewFaultCheck("device1", &gen_healthpb.HealthCheck{
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
	client := gen_healthpb.NewHealthHistoryClient(wrap.ServerToClient(gen_healthpb.HealthHistory_ServiceDesc, server))
	histResp, err := client.ListHealthCheckHistory(ctx, &gen_healthpb.ListHealthCheckHistoryRequest{
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
