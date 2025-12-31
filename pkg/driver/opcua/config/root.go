package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"
	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	transportpb "github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const (
	PointsEventTopicSuffix = "/event/pointset"
)

// valueSourceField represents a ValueSource field with its description for validation.
type valueSourceField struct {
	desc  string
	value *ValueSource
}

// OccupantImpact wraps gen.HealthCheck_OccupantImpact to support JSON unmarshaling from strings.
type OccupantImpact gen.HealthCheck_OccupantImpact

func (o *OccupantImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := gen.HealthCheck_OccupantImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid OccupantImpact value: %q (valid values: OCCUPANT_IMPACT_UNSPECIFIED, NO_OCCUPANT_IMPACT, COMFORT, HEALTH, LIFE, SECURITY)", s)
	}

	*o = OccupantImpact(val)
	return nil
}

func (o OccupantImpact) ToProto() gen.HealthCheck_OccupantImpact {
	return gen.HealthCheck_OccupantImpact(o)
}

// EquipmentImpact wraps gen.HealthCheck_EquipmentImpact to support JSON unmarshaling from strings.
type EquipmentImpact gen.HealthCheck_EquipmentImpact

func (e *EquipmentImpact) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.ToUpper(s)
	val, ok := gen.HealthCheck_EquipmentImpact_value[s]
	if !ok {
		return fmt.Errorf("invalid EquipmentImpact value: %q (valid values: EQUIPMENT_IMPACT_UNSPECIFIED, NO_EQUIPMENT_IMPACT, WARRANTY, LIFESPAN, FUNCTION)", s)
	}

	*e = EquipmentImpact(val)
	return nil
}

func (e EquipmentImpact) ToProto() gen.HealthCheck_EquipmentImpact {
	return gen.HealthCheck_EquipmentImpact(e)
}

// Conn config related to communicating with the OPC UA server.
type Conn struct {
	// Endpoint is the OPC UA server endpoint.
	Endpoint string `json:"endpoint,omitempty"`
	// SubscriptionInterval for OPC UA subscription, defaults to 5s if not set.
	SubscriptionInterval *jsontypes.Duration `json:"subscriptionInterval,omitempty,omitzero"`
	// ClientId is the ID of the client that will be used to connect to the OPC UA server.
	// Should be unique within the context of a server. If not set, a random ID will be generated.
	ClientId uint32 `json:"clientId,omitempty,omitzero"`
}

// Variable is an OPC UA VariableNode, which is essentially a data point which we can read/write to (with permission).
type Variable struct {
	// NodeId identifies the VariableNode in the OPC UA server.
	NodeId string `json:"nodeId,omitempty"`
	// ParsedNodeId is the parsed ua.NodeID.
	ParsedNodeId *ua.NodeID
}

// Device represents a smart core device.
type Device struct {
	// Name the Smart Core device name
	Name string `json:"name,omitempty"`
	// Meta the Smart Core device metadata
	Meta *traits.Metadata `json:"meta,omitempty"`
	// Variables a list of OPC variables the device has
	Variables []*Variable `json:"variables,omitempty"`
	// Traits a map Smart Core traits the device implements
	Traits []RawTrait `json:"traits,omitempty"`
	// Health contains settings for an opc ua device health check
	// If not configured, the occupant and equipment impact will default to UNSPECIFIED
	Health Health `json:"health"`
}

type Health struct {
	OccupantImpact  OccupantImpact  `json:"occupantImpact"`
	EquipmentImpact EquipmentImpact `json:"equipmentImpact"`
	SystemName      string          `json:"systemName"`
}

type Root struct {
	driver.BaseConfig

	Meta    *traits.Metadata `json:"meta,omitempty"`
	Conn    Conn             `json:"conn,omitempty"`
	Devices []Device         `json:"devices,omitempty"`

	// settings for the opc ua system health check
	SystemHealth Health `json:"systemHealth,omitempty"`
}

func ParseConfig(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}
	if cfg.SystemHealth.SystemName == "" {
		return cfg, fmt.Errorf("opcua driver config must specify systemHealth.systemName")
	}
	if cfg.Conn.SubscriptionInterval == nil {
		cfg.Conn.SubscriptionInterval = &jsontypes.Duration{Duration: 5 * time.Second}
	}
	if cfg.Conn.ClientId == 0 {
		cfg.Conn.ClientId = rand.Uint32()
	}

	for _, d := range cfg.Devices {
		for _, v := range d.Variables {
			nId, err := ua.ParseNodeID(v.NodeId)
			if err != nil {
				return cfg, err
			}
			v.ParsedNodeId = nId
		}

		if err := validateDeviceTraits(&d); err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}

// validateDeviceTraits validates trait configurations and checks that all nodeIds referenced in traits
// exist in the device's variable list.
func validateDeviceTraits(device *Device) error {
	validNodeIds := make(map[string]bool)
	for _, v := range device.Variables {
		validNodeIds[v.NodeId] = true
	}

	for _, t := range device.Traits {
		var valueSources []valueSourceField
		var err error

		switch t.Kind {
		case meterpb.TraitName:
			valueSources, err = getValueSourcesForTrait[*MeterConfig](device.Name, t.Raw)
		case trait.Electric:
			valueSources, err = getValueSourcesForTrait[*ElectricConfig](device.Name, t.Raw)
		case transportpb.TraitName:
			valueSources, err = getValueSourcesForTrait[*TransportConfig](device.Name, t.Raw)
		case udmipb.TraitName:
			valueSources, err = getValueSourcesForTrait[*UdmiConfig](device.Name, t.Raw)
		default:
			return fmt.Errorf("device '%s': unknown trait kind '%s'", device.Name, t.Kind)
		}
		if err != nil {
			return err
		}

		for _, field := range valueSources {
			if field.value != nil && field.value.NodeId != "" && !validNodeIds[field.value.NodeId] {
				return fmt.Errorf("device '%s': %s references nodeId '%s' which is not in device variables list",
					device.Name, field.desc, field.value.NodeId)
			}
		}
	}

	return nil
}

type traitWithValueSources interface {
	Validate() error
	valueSources() []valueSourceField
}

// getValueSourcesForTrait parses a trait config, validates it, and returns all its ValueSource fields.
func getValueSourcesForTrait[T traitWithValueSources](deviceName string, rawTrait json.RawMessage) ([]valueSourceField, error) {
	cfg := new(T)
	if err := json.Unmarshal(rawTrait, cfg); err != nil {
		return nil, fmt.Errorf("device '%s': failed to parse trait: %w", deviceName, err)
	}
	if err := (*cfg).Validate(); err != nil {
		return nil, fmt.Errorf("device '%s': %w", deviceName, err)
	}
	return (*cfg).valueSources(), nil
}
