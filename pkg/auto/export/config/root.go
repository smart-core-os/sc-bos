package config

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-golang/pkg/cmp"
)

type Root struct {
	auto.Config

	// Broker configures an MQTT broker to export data to.
	Broker *MQTTBroker `json:"broker,omitempty"`

	Sources []RawSource `json:"sources,omitempty"`
}

type Source struct {
	Type        string      `json:"type,omitempty"`
	Name        string      `json:"name,omitempty"`
	TopicPrefix string      `json:"topicPrefix,omitempty"`
	Duplicates  *Duplicates `json:"duplicates,omitempty"`
}

type RawSource struct {
	Source
	Raw json.RawMessage `json:"-"`
}

func (r *RawSource) UnmarshalJSON(buf []byte) error {
	_ = json.Unmarshal(buf, &r.Raw)
	return json.Unmarshal(buf, &r.Source)
}

type Duplicates struct {
	Include bool `json:"include,omitempty"`

	// Consider floating point values of consecutive publications to be equal if they are within FloatMargin of each other.
	// 1.2 and 1.22 are within 0.1 FloatMargin of each other and would be classes as duplicates.
	FloatMargin *float64 `json:"floatMargin,omitempty"`

	// there's room here to configure what we class as duplicate: is 1.200000001 a duplicate of 1.2
	// Fields should map well to the sc-golang cmp package Value funcs.
}

func (d *Duplicates) TrackDuplicates() bool {
	return d == nil || !d.Include
}

func (d *Duplicates) Cmp() cmp.Message {
	if d == nil {
		return cmp.Equal(
			cmp.FloatValueApprox(0, 0.01),
			cmp.TimeValueWithin(10*time.Millisecond),
			cmp.DurationValueWithin(10*time.Millisecond),
		)
	}
	if d.Include {
		return func(x, y proto.Message) bool {
			return false
		}
	}

	var opts []cmp.Value
	if d.FloatMargin == nil {
		opts = append(opts, cmp.FloatValueApprox(0, 0.01))
	} else {
		opts = append(opts, cmp.FloatValueApprox(0, *d.FloatMargin))
	}

	// here we'd look at any other configuration options and apply them as cmp.Value to the response
	// We don't have any other options yet so we just return the default
	opts = append(opts,
		cmp.TimeValueWithin(10*time.Millisecond),
		cmp.DurationValueWithin(10*time.Millisecond),
	)
	return cmp.Equal(opts...)
}
