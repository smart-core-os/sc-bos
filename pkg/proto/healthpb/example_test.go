package healthpb

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
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
	tempCheck, _ := checks.NewBoundsCheck(deviceName, &HealthCheck{
		// Id can be absent if this owner only ever has one check per device
		DisplayName:     "Ambient Temperature",
		Description:     "Checks the ambient air temperature is within a comfortable range",
		OccupantImpact:  HealthCheck_COMFORT,
		EquipmentImpact: HealthCheck_NO_EQUIPMENT_IMPACT,
		Check: &HealthCheck_Bounds_{Bounds: &HealthCheck_Bounds{
			Expected: &HealthCheck_Bounds_NormalRange{NormalRange: &HealthCheck_ValueRange{
				Low:      FloatValue(15),
				High:     FloatValue(25),
				Deadband: FloatValue(2),
			}},
			DisplayUnit: "°C",
		}},
	})
	defer tempCheck.Dispose()

	client := airtemperaturepb.NewAirTemperatureApiClient(clientConn)
	stream, err := client.PullAirTemperature(ctx, &airtemperaturepb.PullAirTemperatureRequest{Name: deviceName})
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
	funcTest, _ := checks.NewFaultCheck(deviceName, &HealthCheck{
		Id:              "el_function_test",
		DisplayName:     "Emergency Light Function Test",
		Description:     "Checks the emergency light function test status",
		OccupantImpact:  HealthCheck_LIFE,
		EquipmentImpact: HealthCheck_NO_EQUIPMENT_IMPACT,
		ComplianceImpacts: []*HealthCheck_ComplianceImpact{
			{Standard: BS5266_1_2016, Contribution: HealthCheck_ComplianceImpact_FAIL},
		},
	})
	defer funcTest.Dispose()
	durTest, _ := checks.NewFaultCheck(deviceName, &HealthCheck{
		Id:              "el_duration_test",
		DisplayName:     "Emergency Light Duration Test",
		Description:     "Checks the emergency light duration test status",
		OccupantImpact:  HealthCheck_LIFE,
		EquipmentImpact: HealthCheck_NO_EQUIPMENT_IMPACT,
		ComplianceImpacts: []*HealthCheck_ComplianceImpact{
			{Standard: BS5266_1_2016, Contribution: HealthCheck_ComplianceImpact_FAIL},
		},
	})
	defer durTest.Dispose()

	// A utility for updating test results
	updateTestResults := func(c *FaultCheck, r *emergencylightpb.EmergencyTestResult) {
		switch code := r.GetResult(); code {
		case emergencylightpb.EmergencyTestResult_TEST_PASSED:
			c.ClearFaults()
		default:
			c.SetFault(&HealthCheck_Error{
				SummaryText: code.String(),
				DetailsText: fmt.Sprintf("device reported test failure: %s", code),
				Code:        &HealthCheck_Error_Code{Code: code.String(), System: "Smart Core"},
			})
		}
	}

	client := emergencylightpb.NewEmergencyLightApiClient(clientConn)
	stream, err := client.PullTestResultSets(ctx, &emergencylightpb.PullTestResultRequest{Name: deviceName})
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
