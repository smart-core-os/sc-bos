package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gopcua/opcua/ua"
	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	transportpb "github.com/smart-core-os/sc-bos/pkg/gentrait/transport"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

const (
	PointsEventTopicSuffix = "/event/pointset"
)

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
}

// Timing configuration for OPC UA driver operations and retry behavior.
// Note: Timing values set the defaults used by the service framework's retry policy.
// Changes to these values require a driver restart to take effect.
type Timing struct {
	// Timeout for individual OPC UA operations (currently unused, reserved for future use)
	Timeout jsontypes.Duration `json:"timeout,omitempty,omitzero"`
}

type Root struct {
	driver.BaseConfig

	Meta    *traits.Metadata `json:"meta,omitempty"`
	Conn    Conn             `json:"conn,omitempty"`
	Devices []Device         `json:"devices,omitempty"`
	Timing  Timing           `json:"Timing,omitempty"`
}

func ParseConfig(data []byte) (cfg Root, err error) {
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, err
	}
	if cfg.Conn.SubscriptionInterval == nil {
		cfg.Conn.SubscriptionInterval = &jsontypes.Duration{Duration: 5 * time.Second}
	}
	if cfg.Timing.Timeout.Duration == 0 {
		cfg.Timing.Timeout = jsontypes.Duration{Duration: 10 * time.Second}
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

	checkNodeId := func(nodeId, context string) error {
		if nodeId != "" && !validNodeIds[nodeId] {
			return fmt.Errorf("device '%s': %s references nodeId '%s' which is not in device variables list",
				device.Name, context, nodeId)
		}
		return nil
	}

	validateValueSource := func(vs *ValueSource, context string) error {
		if vs != nil {
			return checkNodeId(vs.NodeId, context)
		}
		return nil
	}

	for _, t := range device.Traits {
		var err error
		switch t.Kind {
		case meterpb.TraitName:
			err = validateMeterTrait(device.Name, t.Raw, validateValueSource)
		case trait.Electric:
			err = validateElectricTrait(device.Name, t.Raw, validateValueSource)
		case transportpb.TraitName:
			err = validateTransportTrait(device.Name, t.Raw, validateValueSource)
		case udmipb.TraitName:
			err = validateUdmiTrait(device.Name, t.Raw, validateValueSource)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMeterTrait(deviceName string, rawTrait json.RawMessage, validateValueSource func(*ValueSource, string) error) error {
	var cfg MeterConfig
	if err := json.Unmarshal(rawTrait, &cfg); err != nil {
		return fmt.Errorf("device '%s': failed to parse meter trait: %w", deviceName, err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("device '%s': %w", deviceName, err)
	}
	if err := validateValueSource(cfg.Usage, "meter trait usage"); err != nil {
		return err
	}
	return nil
}

func validateElectricTrait(deviceName string, rawTrait json.RawMessage, validateValueSource func(*ValueSource, string) error) error {
	var cfg ElectricConfig
	if err := json.Unmarshal(rawTrait, &cfg); err != nil {
		return fmt.Errorf("device '%s': failed to parse electric trait: %w", deviceName, err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("device '%s': %w", deviceName, err)
	}

	// Validate single phase nodeIds
	if cfg.Demand.ElectricPhaseConfig != nil {
		if err := validateValueSource(cfg.Demand.Current, "electric trait current"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.Voltage, "electric trait voltage"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.Rating, "electric trait rating"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.PowerFactor, "electric trait powerFactor"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.RealPower, "electric trait realPower"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.ApparentPower, "electric trait apparentPower"); err != nil {
			return err
		}
		if err := validateValueSource(cfg.Demand.ReactivePower, "electric trait reactivePower"); err != nil {
			return err
		}
	}

	// Validate multi-phase nodeIds
	for i, phase := range cfg.Demand.Phases {
		if err := validateValueSource(phase.Current, fmt.Sprintf("electric trait phase[%d] current", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.Voltage, fmt.Sprintf("electric trait phase[%d] voltage", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.Rating, fmt.Sprintf("electric trait phase[%d] rating", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.PowerFactor, fmt.Sprintf("electric trait phase[%d] powerFactor", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.RealPower, fmt.Sprintf("electric trait phase[%d] realPower", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.ApparentPower, fmt.Sprintf("electric trait phase[%d] apparentPower", i)); err != nil {
			return err
		}
		if err := validateValueSource(phase.ReactivePower, fmt.Sprintf("electric trait phase[%d] reactivePower", i)); err != nil {
			return err
		}
	}

	return nil
}

func validateTransportTrait(deviceName string, rawTrait json.RawMessage, validateValueSource func(*ValueSource, string) error) error {
	var cfg TransportConfig
	if err := json.Unmarshal(rawTrait, &cfg); err != nil {
		return fmt.Errorf("device '%s': failed to parse transport trait: %w", deviceName, err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("device '%s': %w", deviceName, err)
	}

	if err := validateValueSource(cfg.ActualPosition, "transport trait actualPosition"); err != nil {
		return err
	}
	if err := validateValueSource(cfg.Load, "transport trait load"); err != nil {
		return err
	}
	if err := validateValueSource(cfg.MovingDirection, "transport trait movingDirection"); err != nil {
		return err
	}
	if err := validateValueSource(cfg.OperatingMode, "transport trait operatingMode"); err != nil {
		return err
	}
	if err := validateValueSource(cfg.Speed, "transport trait speed"); err != nil {
		return err
	}

	for i, door := range cfg.Doors {
		if err := validateValueSource(door.Status, fmt.Sprintf("transport trait door[%d] status", i)); err != nil {
			return err
		}
	}

	for i, dest := range cfg.NextDestinations {
		if err := validateValueSource(&dest.Source, fmt.Sprintf("transport trait nextDestinations[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func validateUdmiTrait(deviceName string, rawTrait json.RawMessage, validateValueSource func(*ValueSource, string) error) error {
	var cfg UdmiConfig
	if err := json.Unmarshal(rawTrait, &cfg); err != nil {
		return fmt.Errorf("device '%s': failed to parse udmi trait: %w", deviceName, err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("device '%s': %w", deviceName, err)
	}

	for name, point := range cfg.Points {
		if err := validateValueSource(point, fmt.Sprintf("udmi trait point '%s'", name)); err != nil {
			return err
		}
	}

	return nil
}
