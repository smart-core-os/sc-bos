package healthpb

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb/standard"
)

var (
	clientConn grpc.ClientConnInterface
	checks     *Checks
	ctx        = context.TODO()
	deviceName = "MyDevice"
)

// ExampleBoundsCheck shows how to create and use a BoundsCheck health check.
// This example creates a health check that monitors the ambient temperature of a device
// to track how comfortable the environment is for its occupants.
func ExampleBoundsCheck() {
	// This bounds check monitors when a value exceeds a normal range.
	tempCheck, _ := checks.NewBoundsCheck(deviceName, &gen.HealthCheck{
		// Id can be absent if this owner only ever has one check per device
		DisplayName:     "Ambient Temperature",
		Description:     "Checks the ambient air temperature is within a comfortable range",
		OccupantImpact:  gen.HealthCheck_COMFORT,
		EquipmentImpact: gen.HealthCheck_NO_EQUIPMENT_IMPACT,
		Check: &gen.HealthCheck_Bounds_{Bounds: &gen.HealthCheck_Bounds{
			Expected: &gen.HealthCheck_Bounds_NormalRange{NormalRange: &gen.HealthCheck_ValueRange{
				Low:      FloatValue(15),
				High:     FloatValue(25),
				Deadband: FloatValue(2),
			}},
			DisplayUnit: "°C",
		}},
	})
	defer tempCheck.Dispose()

	client := traits.NewAirTemperatureApiClient(clientConn)
	stream, err := client.PullAirTemperature(ctx, &traits.PullAirTemperatureRequest{Name: deviceName})
	tempCheck.UpdateReliability(ctx, ReliabilityFromErr(err))
	if err != nil {
		return
	}
	for {
		changes, err := stream.Recv()
		tempCheck.UpdateReliability(ctx, ReliabilityFromErr(err))
		if err != nil {
			return
		}
		lastChange := changes.GetChanges()[len(changes.Changes)-1]
		val := lastChange.GetAirTemperature()
		tempCheck.UpdateValue(ctx, FloatValue(val.GetAmbientTemperature().GetValueCelsius()))
	}
}

// ExampleFaultCheck shows how to create and use an FaultCheck health check.
// This example demonstrates how emergency lighting test results can be monitored
// using two FaultCheck health checks, one for function tests and one for duration tests.
func ExampleFaultCheck() {
	funcTest, _ := checks.NewFaultCheck(deviceName, &gen.HealthCheck{
		Id:              "el_function_test",
		DisplayName:     "Emergency Light Function Test",
		Description:     "Checks the emergency light function test status",
		OccupantImpact:  gen.HealthCheck_LIFE,
		EquipmentImpact: gen.HealthCheck_NO_EQUIPMENT_IMPACT,
		ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
			{Standard: standard.BS5266_1_2016, Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
		},
	})
	defer funcTest.Dispose()
	durTest, _ := checks.NewFaultCheck(deviceName, &gen.HealthCheck{
		Id:              "el_duration_test",
		DisplayName:     "Emergency Light Duration Test",
		Description:     "Checks the emergency light duration test status",
		OccupantImpact:  gen.HealthCheck_LIFE,
		EquipmentImpact: gen.HealthCheck_NO_EQUIPMENT_IMPACT,
		ComplianceImpacts: []*gen.HealthCheck_ComplianceImpact{
			{Standard: standard.BS5266_1_2016, Contribution: gen.HealthCheck_ComplianceImpact_FAIL},
		},
	})
	defer durTest.Dispose()

	// A utility for updating test results
	updateTestResults := func(c *FaultCheck, r *gen.EmergencyTestResult) {
		switch code := r.GetResult(); code {
		case gen.EmergencyTestResult_TEST_PASSED:
			c.ClearFaults()
		default:
			c.SetFault(&gen.HealthCheck_Error{
				SummaryText: code.String(),
				DetailsText: fmt.Sprintf("device reported test failure: %s", code),
				Code:        &gen.HealthCheck_Error_Code{Code: code.String(), System: "Smart Core"},
			})
		}
	}

	client := gen.NewEmergencyLightApiClient(clientConn)
	stream, err := client.PullTestResultSets(ctx, &gen.PullTestResultRequest{Name: deviceName})
	funcTest.UpdateReliability(ctx, ReliabilityFromErr(err))
	if err != nil {
		return
	}
	for {
		changes, err := stream.Recv()
		funcTest.UpdateReliability(ctx, ReliabilityFromErr(err))
		if err != nil {
			return
		}
		if len(changes.GetChanges()) == 0 {
			continue
		}
		lastChange := changes.GetChanges()[len(changes.Changes)-1]
		updateTestResults(funcTest, lastChange.GetTestResult().GetFunctionTest())
		updateTestResults(durTest, lastChange.GetTestResult().GetDurationTest())
	}
}
