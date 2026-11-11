package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

type Health struct {
	OccupantImpact  OccupantImpact  `json:"occupantImpact"`
	EquipmentImpact EquipmentImpact `json:"equipmentImpact"`
}

// DefaultTraitFaultAfter is how long a trait's reads must keep failing before its health check goes abnormal,
// when TraitHealth.FaultAfter is not set.
const DefaultTraitFaultAfter = 2 * time.Minute

// TraitHealth configures the health check each trait publishes for its own BACnet reads.
type TraitHealth struct {
	Health
	// FaultAfter is how long a trait's reads must keep failing before its health check goes abnormal.
	// A single failed poll that recovers within this time leaves the check normal.
	// Defaults to DefaultTraitFaultAfter. Zero raises a fault on the first failed read.
	FaultAfter *Duration `json:"faultAfter,omitempty"`
}

// FaultAfterDuration returns FaultAfter, or DefaultTraitFaultAfter if it is not set.
func (h TraitHealth) FaultAfterDuration() time.Duration {
	if h.FaultAfter == nil {
		return DefaultTraitFaultAfter
	}
	return max(h.FaultAfter.Duration, 0)
}

type HealthCheck struct {
	Health
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	ErrorCode   string `json:"errorCode"`
	Summary     string `json:"summary"`
}

// OccupantImpact wraps healthpb.HealthCheck_OccupantImpact to support JSON unmarshaling from strings.
type OccupantImpact healthpb.HealthCheck_OccupantImpact

func (o *OccupantImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := healthpb.HealthCheck_OccupantImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid OccupantImpact value: %q (valid values: OCCUPANT_IMPACT_UNSPECIFIED, NO_OCCUPANT_IMPACT, COMFORT, HEALTH, LIFE, SECURITY)", s)
	}

	*o = OccupantImpact(val)
	return nil
}

func (o OccupantImpact) MarshalJSON() ([]byte, error) {
	name := healthpb.HealthCheck_OccupantImpact_name[int32(o)]
	if name == "" {
		return nil, fmt.Errorf("invalid OccupantImpact value: %d", o)
	}
	return json.Marshal(name)
}

func (o OccupantImpact) ToProto() healthpb.HealthCheck_OccupantImpact {
	return healthpb.HealthCheck_OccupantImpact(o)
}

// EquipmentImpact wraps healthpb.HealthCheck_EquipmentImpact to support JSON unmarshaling from strings.
type EquipmentImpact healthpb.HealthCheck_EquipmentImpact

func (e *EquipmentImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := healthpb.HealthCheck_EquipmentImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid EquipmentImpact value: %q (valid values: EQUIPMENT_IMPACT_UNSPECIFIED, NO_EQUIPMENT_IMPACT, WARRANTY, LIFESPAN, FUNCTION)", s)
	}

	*e = EquipmentImpact(val)
	return nil
}

func (e EquipmentImpact) MarshalJSON() ([]byte, error) {
	name := healthpb.HealthCheck_EquipmentImpact_name[int32(e)]
	if name == "" {
		return nil, fmt.Errorf("invalid EquipmentImpact value: %d", e)
	}
	return json.Marshal(name)
}

func (e EquipmentImpact) ToProto() healthpb.HealthCheck_EquipmentImpact {
	return healthpb.HealthCheck_EquipmentImpact(e)
}
