package devices

import (
	"testing"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// TestQuery_healthCheckScopedToId pins the query shape a dashboard uses to show only the
// checks that say whether a device is doing its job, rather than every check it carries.
//
// Two things are being pinned, and both are easy to get subtly wrong:
//
//  1. The id condition has to sit in the same any_of query as the abnormality condition, so
//     that the id and the abnormality hold of the *same* check. As a sibling condition on
//     health_checks it would match a device whose scoped check merely exists while some other
//     check is the abnormal one - which is the whole failure being designed out.
//  2. The id is matched by string_contains rather than string_equal, because what a client
//     sees is healthpb.AbsID(owner, id): the driver's own "kind:id" prefixed onto the check id
//     the driver declared. Nothing in a dashboard config knows the driver instance name.
func TestQuery_healthCheckScopedToId(t *testing.T) {
	// as registered by the opcua driver running as service "metering-01" of kind "opcua"
	const (
		statusId        = "opcua:metering-01:deviceStatusCheck"
		informationalId = "opcua:metering-01:informationalPointCheck"
		boardId         = "opcua:metering-01:boardHealth"
	)

	meter := func(name string, checks ...*healthpb.HealthCheck) *devicespb.Device {
		return &devicespb.Device{Name: name, HealthChecks: checks}
	}
	normal := func(id string) *healthpb.HealthCheck {
		return &healthpb.HealthCheck{Id: id, Normality: healthpb.HealthCheck_NORMAL}
	}
	abnormal := func(id string) *healthpb.HealthCheck {
		return &healthpb.HealthCheck{Id: id, Normality: healthpb.HealthCheck_ABNORMAL}
	}
	unreliable := func(id string) *healthpb.HealthCheck {
		return &healthpb.HealthCheck{Id: id, Normality: healthpb.HealthCheck_NORMAL,
			Reliability: &healthpb.HealthCheck_Reliability{State: healthpb.HealthCheck_Reliability_BAD_RESPONSE}}
	}

	tests := []struct {
		name   string
		device *devicespb.Device
		want   bool
	}{
		{
			name:   "healthy device",
			device: meter("Meter01", normal(statusId), normal(informationalId), normal(boardId)),
			want:   false,
		},
		{
			// the case the whole change exists for: a spare per-phase register nothing reads
			// has gone away, and the meter's usage reading is fine
			name:   "only the informational check is abnormal",
			device: meter("Meter02", normal(statusId), abnormal(informationalId), normal(boardId)),
			want:   false,
		},
		{
			name:   "the device status check is abnormal",
			device: meter("Meter03", abnormal(statusId), normal(informationalId), normal(boardId)),
			want:   true,
		},
		{
			name:   "the device status check is unreadable",
			device: meter("Meter04", unreliable(statusId), normal(informationalId), normal(boardId)),
			want:   true,
		},
		{
			// a board reporting its own IEC 61850 fault is worth an engineer's attention
			name:   "the board health check is abnormal",
			device: meter("Meter05", normal(statusId), normal(informationalId), abnormal(boardId)),
			want:   true,
		},
		{
			name:   "an unscoped check is abnormal",
			device: meter("Meter06", normal(statusId), &healthpb.HealthCheck{Id: "opcua:metering-01:somethingElse", Normality: healthpb.HealthCheck_ABNORMAL}),
			want:   false,
		},
	}

	query := scopedHealthQuery("deviceStatusCheck", "boardHealth")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceMatchesQuery(query, tt.device); got != tt.want {
				t.Errorf("deviceMatchesQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

// scopedHealthQuery builds the query unhealthyDeviceConditions(checkId) produces in
// ui/ops/src/composables/devices.js. Kept in the same shape so a change to one is a visible
// change to the other.
func scopedHealthQuery(checkIds ...string) *devicespb.Device_Query {
	abnormal := &devicespb.Device_Query_Condition{
		Field: "normality",
		Value: &devicespb.Device_Query_Condition_StringIn{StringIn: &devicespb.Device_Query_StringList{
			Strings: []string{"ABNORMAL", "HIGH", "LOW"},
		}},
	}
	unreliable := &devicespb.Device_Query_Condition{
		Field: "reliability.state",
		Value: &devicespb.Device_Query_Condition_StringIn{StringIn: &devicespb.Device_Query_StringList{
			Strings: []string{"UNRELIABLE", "CONN_TRANSIENT_FAILURE", "SEND_FAILURE", "NO_RESPONSE",
				"BAD_RESPONSE", "NOT_FOUND", "PERMISSION_DENIED"},
		}},
	}

	// one query per (check id, dimension) pair, ORed by any_of. Conditions within a query are
	// conjunctive, which is what ties the id to the abnormality on a single check.
	var queries []*devicespb.Device_Query
	for _, id := range checkIds {
		idCondition := &devicespb.Device_Query_Condition{
			Field: "id",
			Value: &devicespb.Device_Query_Condition_StringContains{StringContains: ":" + id},
		}
		for _, dimension := range []*devicespb.Device_Query_Condition{abnormal, unreliable} {
			queries = append(queries, &devicespb.Device_Query{
				Conditions: []*devicespb.Device_Query_Condition{idCondition, dimension},
			})
		}
	}

	return &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{{
		Field: "health_checks",
		Value: &devicespb.Device_Query_Condition_AnyOf{AnyOf: &devicespb.Device_Query_QueryList{Queries: queries}},
	}}}
}
