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
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb/standard"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

type Root struct {
	auto.Config
	Devices []*Condition `json:"devices"`
	Check   *HealthCheck `json:"check"`
	Source  Source       `json:"source"`
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

func (r *Root) CheckPb() *healthpb.HealthCheck {
	if r == nil || r.Check == nil {
		return nil
	}
	return r.Check.pb
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

// Source configures which property of a device is checked by a health check.
type Source struct {
	// Trait is the fully qualified name of a trait implemented by monitored devices.
	// TODO: Only some traits are supported, see internal/anytrait/registry.go for the list of supported traits.
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
				s2 := new(healthpb.HealthCheck_ComplianceImpact_Standard)
				proto.Merge(s2, s)                    // copy known standard
				proto.Merge(s2, impact.GetStandard()) // overwrite with any fields already set in config
				check.ComplianceImpacts[i].Standard = s2
			}
		}
	}
	return nil
}
