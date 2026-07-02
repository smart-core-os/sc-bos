package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/internal/protobuf/protopath2"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	auto.Config
	Devices []*Condition `json:"devices"`
	Check   *HealthCheck `json:"check"`
	Source  Source       `json:"source"`
	// Tolerance is the maximum allowed absolute difference between the measured value and the set
	// point. The check goes abnormal when abs(measured - setPoint) exceeds this for longer than
	// Duration. Expressed in the value's native unit (e.g. °C). Must be greater than zero.
	Tolerance float64 `json:"tolerance"`
	// Duration is how long the deviation must continuously exceed Tolerance before the check goes
	// abnormal. The countdown restarts when the set point changes (the equipment gets a fresh window
	// to reach the new target) and is cleared when the deviation returns within Tolerance.
	// Must be greater than zero.
	Duration jsontypes.Duration `json:"duration"`
	// MaxDuration, when set, is an absolute backstop: the check goes abnormal once the deviation has
	// exceeded Tolerance continuously for this long, regardless of how often the set point changes.
	// It catches equipment that never tracks while its set point is repeatedly adjusted, which would
	// otherwise keep restarting the Duration countdown and never trip. Unset (zero) disables the
	// backstop. Must be greater than Duration when set.
	//
	// MaxDuration supplements Duration rather than replacing it: Duration is always required, and
	// MaxDuration is optional. Setting MaxDuration on its own is not valid.
	MaxDuration jsontypes.Duration `json:"maxDuration,omitempty"`
}

func (r *Root) DevicesPb() []*devicespb.Device_Query_Condition {
	if r == nil {
		return nil
	}
	conds := make([]*devicespb.Device_Query_Condition, len(r.Devices))
	for i, c := range r.Devices {
		conds[i] = c.pb
	}
	return conds
}

func (r *Root) CheckPb() *healthpb.HealthCheck {
	if r == nil || r.Check == nil {
		return nil
	}
	return r.Check.pb
}

type Condition struct {
	pb *devicespb.Device_Query_Condition
}

func (c *Condition) UnmarshalJSON(bytes []byte) error {
	cond := &devicespb.Device_Query_Condition{}
	err := protojson.Unmarshal(bytes, cond)
	if err != nil {
		return fmt.Errorf("condition: %w", err)
	}
	*c = Condition{cond}
	return nil
}

func (c *Condition) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(c.pb)
}

type HealthCheck struct {
	pb *healthpb.HealthCheck
}

func (h *HealthCheck) UnmarshalJSON(bytes []byte) error {
	hc := &healthpb.HealthCheck{}
	err := protojson.Unmarshal(bytes, hc)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	*h = HealthCheck{hc}
	return nil
}

func (h *HealthCheck) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(h.pb)
}

// Source configures which properties of a device are compared by the health check.
// The measured value and the set point are read from the same trait resource.
type Source struct {
	// Trait is the fully qualified name of a trait implemented by monitored devices.
	// The trait must be registered in the shared trait registry (pkg/auto/internal/anytrait).
	Trait    trait.Name `json:"trait"`
	Resource Resource   `json:"resource,omitempty"`
	// Measured is the dot-separated path to the measured value, e.g. "ambientTemperature.valueCelsius".
	Measured Value `json:"measured"`
	// SetPoint is the dot-separated path to the target value, e.g. "temperatureSetPoint.valueCelsius".
	SetPoint Value `json:"setPoint"`
}

// Resource is the name of a trait resource.
// For example "AirTemperature", for which there would be GetAirTemperature and/or PullAirTemperature
// rpc methods in the trait API.
// When empty, the first declared resource in the trait is used.
type Resource string

func (r Resource) String() string {
	return string(r)
}

// Value is a dot-separated path to a field in a trait resource specified in Source.
type Value string

func (v Value) String() string {
	return string(v)
}

func (v Value) Parse(md protoreflect.MessageDescriptor) (protopath.Path, *fieldmaskpb.FieldMask, error) {
	p, err := protopath2.ParsePath(md, string(v))
	if err != nil {
		return nil, nil, err
	}
	if len(v) == 0 || len(p) == 1 {
		return p, nil, nil
	}

	// the field mask is like p.String without the root step
	fmPath := p[1:].String()
	fmPath = strings.TrimPrefix(fmPath, ".")
	// validation will have been done by ParsePath
	fm := &fieldmaskpb.FieldMask{Paths: []string{fmPath}}
	return p, fm, nil
}

func Read(data []byte) (Root, error) {
	var cfg Root
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return Root{}, err
	}
	err = Hydrate(&cfg)
	if err != nil {
		return Root{}, err
	}
	if err := Validate(&cfg); err != nil {
		return Root{}, err
	}
	return cfg, err
}

// Validate checks that the config is internally consistent.
// Path validity against the trait resource is checked later, once the trait is resolved.
func Validate(cfg *Root) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Tolerance <= 0 {
		return fmt.Errorf("tolerance must be greater than zero, got %v", cfg.Tolerance)
	}
	if cfg.Duration.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero, got %v", cfg.Duration.Duration)
	}
	if cfg.MaxDuration.Duration != 0 && cfg.MaxDuration.Duration <= cfg.Duration.Duration {
		return fmt.Errorf("maxDuration must be greater than duration (%v), got %v", cfg.Duration.Duration, cfg.MaxDuration.Duration)
	}
	if cfg.Source.Measured == "" {
		return fmt.Errorf("source.measured is required")
	}
	if cfg.Source.SetPoint == "" {
		return fmt.Errorf("source.setPoint is required")
	}
	if cfg.Source.Trait == "" {
		return fmt.Errorf("source.trait is required")
	}
	return nil
}

// Hydrate fills in additional details in the config that are not specified directly in JSON.
// For example, it fills in known details about standards referenced in compliance impacts.
func Hydrate(cfg *Root) error {
	if cfg == nil {
		return nil
	}
	if check := cfg.CheckPb(); check != nil {
		for i, impact := range check.GetComplianceImpacts() {
			// fill in more details for standards that we know about
			if s := healthpb.FindStandardByDisplayName(impact.GetStandard().GetDisplayName()); s != nil {
				s2 := new(healthpb.HealthCheck_ComplianceImpact_Standard)
				proto.Merge(s2, s)                    // copy known standard
				proto.Merge(s2, impact.GetStandard()) // overwrite with any fields already set in config
				check.ComplianceImpacts[i].Standard = s2
			}
		}
	}
	return nil
}
