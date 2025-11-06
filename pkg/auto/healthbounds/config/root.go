package config

import (
	"encoding/json"
	"fmt"

	"github.com/stoewer/go-strcase"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/vanti-dev/sc-bos/pkg/auto"
	"github.com/vanti-dev/sc-bos/pkg/gen"
	"github.com/vanti-dev/sc-bos/pkg/gentrait/healthpb/standard"
)

type Root struct {
	auto.Config
	Devices []*Condition `json:"devices"`
	Check   *HealthCheck `json:"check"`
	Source  Source       `json:"source"`
}

func (r *Root) DevicesPb() []*gen.Device_Query_Condition {
	if r == nil {
		return nil
	}
	conds := make([]*gen.Device_Query_Condition, len(r.Devices))
	for i, c := range r.Devices {
		conds[i] = c.pb
	}
	return conds
}

type Condition struct {
	pb *gen.Device_Query_Condition
}

func (c *Condition) UnmarshalJSON(bytes []byte) error {
	cond := &gen.Device_Query_Condition{}
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

func (r *Root) CheckPb() *gen.HealthCheck {
	if r == nil || r.Check == nil {
		return nil
	}
	return r.Check.pb
}

type HealthCheck struct {
	pb *gen.HealthCheck
}

func (h *HealthCheck) UnmarshalJSON(bytes []byte) error {
	hc := &gen.HealthCheck{}
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

// Source configures which property of a device is checked by a health check.
type Source struct {
	Trait    trait.Name `json:"trait"`
	Resource Resource   `json:"resource,omitempty"`
	Value    Value      `json:"value,omitempty"`
}

// Resource is the name of a trait resource.
// For example "OnOff" or "Brightness", for which there would be GetOnOff and/or PullOnOff rpc methods in the trait API.
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

func (v Value) ToProto() string {
	return strcase.SnakeCase(string(v))
}

func (v Value) ToFieldMask() *fieldmaskpb.FieldMask {
	if v == "" {
		return nil
	}
	// convert camelCase to snake_case for protobuf field names
	ps := strcase.SnakeCase(string(v)) // ascii only
	return &fieldmaskpb.FieldMask{Paths: []string{ps}}
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
	return cfg, err
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
			if s := standard.FindByDisplayName(impact.GetStandard().GetDisplayName()); s != nil {
				s2 := new(gen.HealthCheck_ComplianceImpact_Standard)
				proto.Merge(s2, s)                    // copy known standard
				proto.Merge(s2, impact.GetStandard()) // overwrite with any fields already set in config
				check.ComplianceImpacts[i].Standard = s2
			}
		}
	}
	return nil
}
