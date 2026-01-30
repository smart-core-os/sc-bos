package config

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

// Trait is the common configuration for bacnet device traits.
// Specific implementations that pull objects together into a trait should embed this type into their specific config
// types.
//
//	type OnOffConfig struct {
//	  Trait
//	  Value *ValueSource `json:"value,omitempty"`
//	}
type Trait struct {
	Name     string           `json:"name,omitempty"`
	Kind     trait.Name       `json:"kind,omitempty"`
	Metadata *traits.Metadata `json:"metadata,omitempty"`
	// poll period used to poll the objects for pull updates
	// defaults to 10s
	PollPeriod *Duration `json:"pollPeriod,omitempty"`
	// how long to wait when running a poll for it to respond. Defaults to PollPeriod.
	PollTimeout *Duration `json:"pollTimeout,omitempty"`
	// after a write update, how long to wait by default before giving up on reads becoming consistent with the written value.
	// defaults to 1m
	DefaultRWConsistencyTimeout *Duration `json:"defaultRWConsistencyTimeout,omitempty"`
	// When reading multiple properties, split the properties into chunks of this size and execute in parallel.
	// 0 means do not chunk.
	ChunkSize int `json:"chunkSize,omitempty"`
	// Each trait can have its own health check configuration.
	// If not configured, occupant and equipment impact will default to UNSPECIFIED.
	Health Health `json:"health"`
}

func (t *Trait) PollPeriodDuration() time.Duration {
	if t.PollPeriod != nil && t.PollPeriod.Duration != 0 {
		return t.PollPeriod.Duration
	}
	return time.Second * 10
}

func (t *Trait) PollTimeoutDuration() time.Duration {
	if t.PollTimeout != nil && t.PollTimeout.Duration != 0 {
		return t.PollTimeout.Duration
	}
	return t.PollPeriodDuration()
}

func (t *Trait) DefaultRWConsistencyTimeoutDuration() time.Duration {
	if t.DefaultRWConsistencyTimeout != nil && t.DefaultRWConsistencyTimeout.Duration != 0 {
		return t.DefaultRWConsistencyTimeout.Duration
	}
	return time.Minute
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

// ValueSource configures a single object property as the source of some trait value.
type ValueSource struct {
	Device   *DeviceRef  `json:"device,omitempty"`
	Object   *ObjectRef  `json:"object,omitempty"`
	Property *PropertyID `json:"property,omitempty"`
	// used for converting simple units like kW -> W.
	// The value from the source will be multiplied by Scale when reading, and divided when writing.
	// For example if the trait is in watts and the device is in kW then Scale should be 1000 (aka kilo).
	Scale float64 `json:"scale,omitempty"`
}

// Lookup finds the gobacnet device, object, and property this ValueSource refers to.
func (vs ValueSource) Lookup(ctx known.Context) (bactypes.Device, bactypes.Object, property.ID, error) {
	p := property.PresentValue
	if vs.Property != nil {
		p = property.ID(*vs.Property)
	}

	if vs.Device == nil || vs.Object == nil {
		return bactypes.Device{}, bactypes.Object{}, p, errors.New("missing device or object")
	}

	device, err := vs.Device.Lookup(ctx)
	if err != nil {
		return device, bactypes.Object{}, p, err
	}
	object, err := vs.Object.Lookup(device, ctx)
	return device, object, p, err
}

// Scaled returns v scaled by the Scale factor.
// If vs.Scale is 0 or v is not a number then v is returned unchanged.
func (vs ValueSource) Scaled(v any) any {
	if v == nil {
		return v
	}
	if vs.Scale == 0 {
		return v
	}
	switch v := v.(type) {
	case float32:
		return float32(float64(v) * vs.Scale)
	case float64:
		return v * vs.Scale
	case int:
		return int(float64(v) * vs.Scale)
	case int32:
		return int32(float64(v) * vs.Scale)
	case int64:
		return int64(float64(v) * vs.Scale)
	case uint:
		return uint(float64(v) * vs.Scale)
	case uint32:
		return uint32(float64(v) * vs.Scale)
	case uint64:
		return uint64(float64(v) * vs.Scale)
	}
	return v
}

func (vs ValueSource) String() string {
	res := ""
	if vs.Device != nil {
		res += strconv.Itoa(int(vs.Device.id))
	}
	if vs.Object != nil {
		res += ":" + vs.Object.id.String()
	}
	return res
}

type DeviceRef struct {
	id   bactypes.ObjectInstance
	name string
}

func NewDeviceRef(name string) *DeviceRef {
	return &DeviceRef{name: name}
}

func NewDeviceRefID(id bactypes.ObjectInstance) *DeviceRef {
	return &DeviceRef{id: id}
}

func (d DeviceRef) Lookup(ctx known.Context) (bactypes.Device, error) {
	if d.name != "" {
		return ctx.LookupDeviceByName(d.name)
	}
	return ctx.LookupDeviceByID(d.id)
}

func (d DeviceRef) MarshalJSON() ([]byte, error) {
	if d.name != "" {
		return json.Marshal(d.name)
	}
	return json.Marshal(d.id)
}

func (d *DeviceRef) UnmarshalJSON(bytes []byte) error {
	var val any
	if err := json.Unmarshal(bytes, &val); err != nil {
		return err
	}

	switch v := val.(type) {
	case float64:
		*d = DeviceRef{id: bactypes.ObjectInstance(v)}
		return nil
	case string:
		*d = DeviceRef{name: v}
		return nil
	default:
		return errors.New("invalid device ref")
	}
}

type ObjectRef struct {
	id   ObjectID
	name string
}

func NewObjectRef(name string) *ObjectRef {
	return &ObjectRef{name: name}
}

func NewObjectRefID(id ObjectID) *ObjectRef {
	return &ObjectRef{id: id}
}

func (o ObjectRef) Lookup(device bactypes.Device, ctx known.Context) (bactypes.Object, error) {
	if o.name != "" {
		return ctx.LookupObjectByName(device, o.name)
	}
	return ctx.LookupObjectByID(device, bactypes.ObjectID(o.id))
}

func (o ObjectRef) MarshalJSON() ([]byte, error) {
	if o.name != "" {
		return json.Marshal(o.name)
	}
	return json.Marshal(o.id)
}

func (o *ObjectRef) UnmarshalJSON(bytes []byte) error {
	var s string
	if err := json.Unmarshal(bytes, &s); err != nil {
		return err
	}

	oid, err := ObjectIDFromString(s)
	if err == nil {
		*o = ObjectRef{id: oid}
		return nil
	}
	*o = ObjectRef{name: s}
	return nil
}
