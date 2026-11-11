package config

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	auto.Config
	Source  *Source  `json:"source,omitempty"`
	Storage *Storage `json:"storage,omitempty"`
}

type Condition struct {
	pb *devicespb.Device_Query_Condition
}

func (c *Condition) UnmarshalJSON(bytes []byte) error {
	cond := &devicespb.Device_Query_Condition{}
	if err := protojson.Unmarshal(bytes, cond); err != nil {
		return fmt.Errorf("condition: %w", err)
	}
	*c = Condition{cond}
	return nil
}

func (c *Condition) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(c.pb)
}

type Source struct {
	Name  string     `json:"name,omitempty"`
	Trait trait.Name `json:"trait,omitempty"`
	// ReadMask instructs the history service to only read the specified fields
	ReadMask *FieldMask `json:"readMask,omitempty"`
	// PollingSchedule, when present, configures the auto to poll the source updates even if the source supports pull.
	// A new poll will be executed each time the given schedule triggers, but only changes will be recorded,
	// plus an unchanged reading every MaxInterval if that is set.
	// Polling is useful when it is not critical to collect every change to a device,
	// and excessive storage of historical records is a concern.
	PollingSchedule *jsontypes.Schedule `json:"pollingSchedule,omitempty"`
	// Devices is a list of conditions to filter which devices to record history for. Takes precedence over Name.
	// If empty, Name will be used as the history source.
	Devices []*Condition `json:"devices,omitempty"`
	// Timeout caps how long a single poll of the source may take before it is abandoned and retried.
	// Pull subscriptions are not affected.
	// Defaults to 30s.
	Timeout *jsontypes.Duration `json:"timeout,omitempty"`
	// MaxInterval, when set, records a reading at least this often even if the source's value hasn't changed.
	// Without it only changes are recorded, so a gap in history could mean the source was unreadable or just idle.
	// With it, a gap well beyond MaxInterval means the source could not be read.
	// Gaps between heartbeats run a little over MaxInterval, by up to Timeout, as each is timed from the last record.
	//
	// Once nothing has been recorded for MaxInterval the source is read on demand, with a Get, and the result recorded.
	// The auto never replays a value it already holds: if the Get fails nothing is recorded,
	// and it is retried up to three times, a minute apart, before waiting another MaxInterval.
	// The first is due MaxInterval after the auto starts, not at start.
	// This relies on the source's Get reading the device and failing when it can't.
	// Sources that answer Get from a cached value, as some model-backed drivers and zones do,
	// will have that value recorded instead, and leave no gap.
	// A source that doesn't implement Get is logged and not heartbeated.
	//
	// Each source then writes at least one record per MaxInterval, which adds up on a large estate,
	// and storage.ttl.maxCount covers a correspondingly shorter period.
	// It is intended for metering, where consumers difference readings across gaps; leave it unset elsewhere.
	// Absent or zero records changes only.
	// Either way, a reading that fails to be stored is followed by recording the next one, even if it is unchanged.
	MaxInterval *jsontypes.Duration `json:"maxInterval,omitempty"`
}

// DefaultTimeout is used for Source.Timeout and Storage.Timeout when they are absent.
const DefaultTimeout = 30 * time.Second

// TimeoutDuration returns Timeout or its default.
func (s Source) TimeoutDuration() time.Duration {
	return timeoutOrDefault(s.Timeout)
}

// MaxIntervalDuration returns MaxInterval, or 0 when it is absent.
func (s Source) MaxIntervalDuration() time.Duration {
	return s.MaxInterval.Or(0)
}

func (s *Source) DevicesPb() []*devicespb.Device_Query_Condition {
	if s == nil {
		return nil
	}
	conds := make([]*devicespb.Device_Query_Condition, len(s.Devices))
	for i, c := range s.Devices {
		conds[i] = c.pb
	}
	return conds
}

func (s Source) SourceName() string {
	return fmt.Sprintf("%s[%s]", s.Name, s.Trait)
}

type FieldMask fieldmaskpb.FieldMask

func (f *FieldMask) PB() *fieldmaskpb.FieldMask {
	if f == nil {
		return nil
	}
	return (*fieldmaskpb.FieldMask)(f)
}

func (f *FieldMask) UnmarshalJSON(bytes []byte) error {
	return protojson.Unmarshal(bytes, (*fieldmaskpb.FieldMask)(f))
}

func (f *FieldMask) MarshalJSON() ([]byte, error) {
	return protojson.Marshal((*fieldmaskpb.FieldMask)(f))
}

type Storage struct {
	Type string `json:"type,omitempty"`
	pgxutil.RoleConfig
	Name string `json:"name,omitempty"`
	// TTL is the time-to-live for records. Zero-value (not-specified) means "forever".
	TTL *TTL `json:"ttl,omitempty"`
	// Timeout caps how long writing a single record may take.
	// A record that can't be written in time is logged and dropped.
	// It relies on the store honouring ctx cancellation, which the hub and api stores do.
	// Defaults to 30s.
	Timeout *jsontypes.Duration `json:"timeout,omitempty"`
}

// TimeoutDuration returns Timeout or its default.
func (s *Storage) TimeoutDuration() time.Duration {
	if s == nil {
		return DefaultTimeout
	}
	return timeoutOrDefault(s.Timeout)
}

func timeoutOrDefault(d *jsontypes.Duration) time.Duration {
	if d == nil || d.Duration <= 0 {
		return DefaultTimeout
	}
	return d.Duration
}

type TTL struct {
	MaxAge   jsontypes.Duration `json:"maxAge"`
	MaxCount int64              `json:"maxCount,omitempty"`
}
