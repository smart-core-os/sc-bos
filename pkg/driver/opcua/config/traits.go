package config

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/conv"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

type Trait struct {
	Name     string           `json:"name,omitempty"`
	Kind     trait.Name       `json:"kind,omitempty"`
	Metadata *traits.Metadata `json:"metadata,omitempty"`
}

type RawTrait struct {
	Trait
	Raw json.RawMessage `json:"-"`
}

func (c *RawTrait) MarshalJSON() ([]byte, error) {
	return c.Raw, nil
}

func (c *RawTrait) UnmarshalJSON(buf []byte) error {
	if c == nil {
		*c = RawTrait{}
	}
	c.Raw = buf
	return json.Unmarshal(buf, &c.Trait)
}

// ValueSource configures a single Variable as the source of some trait value.
type ValueSource struct {
	NodeId string `json:"nodeId,omitempty"`
	Name   string `json:"name,omitempty"`
	// Description is a human-readable description of the source
	Description string `json:"description,omitempty"`
	// Optional. Used for converting simple units like kW -> W.
	// The value from the source will be multiplied by Scale when reading, and divided when writing.
	// For example if the trait is in watts and the device is in kW then Scale should be 1000 (aka kilo).
	Scale float64 `json:"scale,omitempty"`
	// Optional. Enum is a generic map to convert the OPC UA point value to something else.
	// For instance, converting the OCP UA value to an enum in a Smart Core trait which can be done by mapping the
	// OPC UA value as the key and the element from the generated <EnumName>_value field in the trait pb file.
	// The key needs to be an integer, it is defined as a string here for JSON marshaling.
	Enum map[string]string `json:"enum,omitempty"`
}

// Validate checks that the ValueSource has a valid NodeId.
func (v *ValueSource) Validate(fieldName string) error {
	if v == nil {
		return nil
	}
	if v.NodeId == "" {
		return fmt.Errorf("%s: nodeId is required", fieldName)
	}
	return nil
}

// GetValueFromIntKey get the value from the enum map given an integer OPC UA value
func (v *ValueSource) GetValueFromIntKey(val any) any {
	if v.Enum != nil {
		i, err := conv.IntValue(val)
		if err == nil {
			if s, ok := v.Enum[strconv.Itoa(i)]; ok {
				return s
			}
		}
	}
	return val
}

// Scaled returns val scaled by the Scale factor.
// If Scale is 0 or 1, or val is not a number, then val is returned unchanged.
// The value is multiplied by Scale when reading (e.g., kW * 1000 = W).
func (v *ValueSource) Scaled(val any) any {
	if val == nil {
		return val
	}
	if v.Scale == 0 || v.Scale == 1 {
		return val
	}
	switch val := val.(type) {
	case float32:
		return float32(float64(val) * v.Scale)
	case float64:
		return val * v.Scale
	case int:
		return int(float64(val) * v.Scale)
	case int32:
		return int32(float64(val) * v.Scale)
	case int64:
		return int64(float64(val) * v.Scale)
	case uint:
		return uint(float64(val) * v.Scale)
	case uint32:
		return uint32(float64(val) * v.Scale)
	case uint64:
		return uint64(float64(val) * v.Scale)
	}
	return val
}

// UdmiConfig is configured by a Device that wants to implement the UDMI trait.
type UdmiConfig struct {
	Trait
	// TopicPrefix is the prefix prepended to the topic in a mqttpb.MqttMessage
	TopicPrefix string `json:"topicPrefix,omitempty"`
	// Points the points we want to send to the UDMI bus. point name -> point config (nodeId and optional enum)
	Points map[string]*ValueSource `json:"points"`
}

// Validate checks that the UDMI config has at least one point configured.
func (c *UdmiConfig) Validate() error {
	if len(c.Points) == 0 {
		return fmt.Errorf("udmi trait: at least one point must be configured")
	}
	for name, point := range c.Points {
		if err := point.Validate(fmt.Sprintf("udmi point '%s'", name)); err != nil {
			return err
		}
	}
	return nil
}

// valueSources returns all ValueSource fields in the UdmiConfig for validation.
func (c *UdmiConfig) valueSources() []valueSourceField {
	fields := make([]valueSourceField, 0, len(c.Points))
	for name, point := range c.Points {
		fields = append(fields, valueSourceField{
			desc:  fmt.Sprintf("udmi trait point '%s'", name),
			value: point,
		})
	}
	return fields
}

// MeterConfig is configured by a Device that wants to implement the Meter trait.
type MeterConfig struct {
	Trait
	Unit  string       `json:"unit,omitempty"`
	Usage *ValueSource `json:"usage,omitempty"`
}

// Validate checks that the Meter config has usage configured.
func (c *MeterConfig) Validate() error {
	if c.Usage == nil {
		return fmt.Errorf("meter trait: usage is required")
	}
	return c.Usage.Validate("meter usage")
}

// valueSources returns all ValueSource fields in the MeterConfig for validation.
func (c *MeterConfig) valueSources() []valueSourceField {
	return []valueSourceField{
		{"meter trait usage", c.Usage},
	}
}

type Door struct {
	Title  string       `json:"title,omitempty"`
	Deck   int          `json:"deck,omitempty"`
	Status *ValueSource `json:"status,omitempty"`
}

type LocationType string

const (
	// SingleFloor tells us that the OPC UA node represents a single floor.
	SingleFloor LocationType = "SingleFloor"
)

type Location struct {
	// Type tells us how to interpret the value source. It must be one of the defined LocationType values.
	// For example, the node in the value source could describe a single location,
	// with other nodes telling us about the next location.
	// Or it could contain an array that lists all the destinations. It is unclear what this needs to handle,
	// so it needs to be flexible enough and extensible to handle future integrations.
	Type   LocationType `json:"type,omitempty"`
	Source ValueSource  `json:"source,omitempty"`
}

// TransportConfig is configured by a Device that wants to implement the Transport trait.
type TransportConfig struct {
	Trait
	ActualPosition  *ValueSource `json:"actualPosition,omitempty"`
	Doors           []*Door      `json:"doors,omitempty"`
	Load            *ValueSource `json:"load,omitempty"`
	LoadUnit        string       `json:"loadUnit,omitempty"`
	MaxLoad         int32        `json:"maxLoad,omitempty"`
	MovingDirection *ValueSource `json:"movingDirection,omitempty"`
	// The OPC UA node(s) which tells us the where the transport is going to stop at next.
	// If the OPC UA server has more than one point which tells us about the next destinations,
	// this array should be ordered so that it matches the order of the physical transport stops.
	// i.e [0] = first stop, [1] = second stop, etc.
	NextDestinations []*Location  `json:"nextDestinations,omitempty"`
	OperatingMode    *ValueSource `json:"operatingMode,omitempty"`
	Speed            *ValueSource `json:"speed,omitempty"`
	SpeedUnit        string       `json:"speedUnit,omitempty"`
}

// Validate checks that the Transport config has at least one field configured.
func (c *TransportConfig) Validate() error {
	hasAtLeastOne := c.ActualPosition != nil ||
		len(c.Doors) > 0 ||
		c.Load != nil ||
		c.MovingDirection != nil ||
		len(c.NextDestinations) > 0 ||
		c.OperatingMode != nil ||
		c.Speed != nil

	if !hasAtLeastOne {
		return fmt.Errorf("transport trait: at least one field must be configured")
	}

	// Validate individual value sources
	if err := c.ActualPosition.Validate("transport actualPosition"); err != nil {
		return err
	}
	if err := c.Load.Validate("transport load"); err != nil {
		return err
	}
	if err := c.MovingDirection.Validate("transport movingDirection"); err != nil {
		return err
	}
	if err := c.OperatingMode.Validate("transport operatingMode"); err != nil {
		return err
	}
	if err := c.Speed.Validate("transport speed"); err != nil {
		return err
	}

	// Validate doors
	for i, door := range c.Doors {
		if door.Status != nil {
			if err := door.Status.Validate(fmt.Sprintf("transport door[%d] status", i)); err != nil {
				return err
			}
		}
	}

	// Validate next destinations
	for i, dest := range c.NextDestinations {
		if err := dest.Source.Validate(fmt.Sprintf("transport nextDestinations[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// valueSources returns all ValueSource fields in the TransportConfig for validation.
func (c *TransportConfig) valueSources() []valueSourceField {
	fields := []valueSourceField{
		{"transport trait actualPosition", c.ActualPosition},
		{"transport trait load", c.Load},
		{"transport trait movingDirection", c.MovingDirection},
		{"transport trait operatingMode", c.OperatingMode},
		{"transport trait speed", c.Speed},
	}

	for i, door := range c.Doors {
		fields = append(fields, valueSourceField{
			desc:  fmt.Sprintf("transport trait door[%d] status", i),
			value: door.Status,
		})
	}

	for i, dest := range c.NextDestinations {
		fields = append(fields, valueSourceField{
			desc:  fmt.Sprintf("transport trait nextDestinations[%d]", i),
			value: &dest.Source,
		})
	}

	return fields
}

type ElectricConfig struct {
	Trait
	Demand *ElectricDemandConfig `json:"demand,omitempty"`
}

// Validate checks that the Electric config has demand configured.
func (c *ElectricConfig) Validate() error {
	if c.Demand == nil {
		return fmt.Errorf("electric trait: demand is required")
	}
	return c.Demand.Validate()
}

// valueSources returns all ValueSource fields in the ElectricConfig for validation.
func (c *ElectricConfig) valueSources() []valueSourceField {
	var fields []valueSourceField
	if c.Demand != nil {
		// Single phase
		if c.Demand.ElectricPhaseConfig != nil {
			fields = append(fields, c.Demand.ElectricPhaseConfig.valueSources("electric trait")...)
		}
		// Multi-phase
		for i, phase := range c.Demand.Phases {
			if phase.hasAnyField() {
				fields = append(fields, phase.valueSources(fmt.Sprintf("electric trait phase[%d]", i))...)
			}
		}
	}
	return fields
}

type ElectricDemandConfig struct {
	*ElectricPhaseConfig                        // single phase
	Phases               [3]ElectricPhaseConfig `json:"phases,omitempty"`
}

// Validate checks that the ElectricDemand config has at least one field configured.
func (c *ElectricDemandConfig) Validate() error {
	// Check if single phase or multi-phase has at least one field configured
	hasSinglePhase := c.ElectricPhaseConfig != nil && c.ElectricPhaseConfig.hasAnyField()
	hasMultiPhase := c.Phases[0].hasAnyField() || c.Phases[1].hasAnyField() || c.Phases[2].hasAnyField()

	if !hasSinglePhase && !hasMultiPhase {
		return fmt.Errorf("electric demand: at least one power measurement field must be configured")
	}

	// Validate single phase if configured
	if c.ElectricPhaseConfig != nil {
		if err := c.ElectricPhaseConfig.Validate("electric demand"); err != nil {
			return err
		}
	}

	// Validate multi-phase if configured
	for i := range c.Phases {
		if c.Phases[i].hasAnyField() {
			if err := c.Phases[i].Validate(fmt.Sprintf("electric demand phase[%d]", i)); err != nil {
				return err
			}
		}
	}

	return nil
}

type ElectricPhaseConfig struct {
	Current *ValueSource `json:"current,omitempty"`
	Voltage *ValueSource `json:"voltage,omitempty"`
	Rating  *ValueSource `json:"rating,omitempty"`

	PowerFactor   *ValueSource `json:"powerFactor,omitempty"`
	RealPower     *ValueSource `json:"realPower,omitempty"`
	ApparentPower *ValueSource `json:"apparentPower,omitempty"`
	ReactivePower *ValueSource `json:"reactivePower,omitempty"`
}

// hasAnyField returns true if any field in the phase config is configured.
func (c *ElectricPhaseConfig) hasAnyField() bool {
	return c.Current != nil ||
		c.Voltage != nil ||
		c.Rating != nil ||
		c.PowerFactor != nil ||
		c.RealPower != nil ||
		c.ApparentPower != nil ||
		c.ReactivePower != nil
}

// Validate validates all configured value sources in the phase config.
func (c *ElectricPhaseConfig) Validate(prefix string) error {
	if err := c.Current.Validate(prefix + " current"); err != nil {
		return err
	}
	if err := c.Voltage.Validate(prefix + " voltage"); err != nil {
		return err
	}
	if err := c.Rating.Validate(prefix + " rating"); err != nil {
		return err
	}
	if err := c.PowerFactor.Validate(prefix + " powerFactor"); err != nil {
		return err
	}
	if err := c.RealPower.Validate(prefix + " realPower"); err != nil {
		return err
	}
	if err := c.ApparentPower.Validate(prefix + " apparentPower"); err != nil {
		return err
	}
	if err := c.ReactivePower.Validate(prefix + " reactivePower"); err != nil {
		return err
	}
	return nil
}

// valueSources returns all ValueSource fields in the ElectricPhaseConfig for validation.
func (c *ElectricPhaseConfig) valueSources(prefix string) []valueSourceField {
	return []valueSourceField{
		{prefix + " current", c.Current},
		{prefix + " voltage", c.Voltage},
		{prefix + " rating", c.Rating},
		{prefix + " powerFactor", c.PowerFactor},
		{prefix + " realPower", c.RealPower},
		{prefix + " apparentPower", c.ApparentPower},
		{prefix + " reactivePower", c.ReactivePower},
	}
}
