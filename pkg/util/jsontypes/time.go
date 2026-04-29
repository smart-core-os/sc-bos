package jsontypes

import (
	"encoding/json"
	"strings"
	"time"
)

// Duration wraps a time.Duration.
// It marshals / unmarshals to a string with the time.ParseDuration format.
type Duration struct {
	time.Duration
}

// Or returns d.Duration if d is not nil, or ifAbsent if d is nil.
// Or only really works if Duration is used as a pointer
//
//	type Config struct {
//		TTL *Duration // pointer type
//	}
//
//	ttl := config.TTL.Or(15*time.Minute)
func (d *Duration) Or(ifAbsent time.Duration) time.Duration {
	if d == nil {
		return ifAbsent
	}
	return d.Duration
}

//goland:noinspection GoMixedReceiverTypes
func (d *Duration) UnmarshalJSON(raw []byte) error {
	var str string
	err := json.Unmarshal(raw, &str)
	if err != nil {
		return err
	}
	parsed, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (d Duration) MarshalJSON() ([]byte, error) {
	str := d.String()
	if strings.HasSuffix(str, "m0s") {
		str = str[:len(str)-2]
	}
	str = strings.Replace(str, "h0m", "h", 1)
	return json.Marshal(str)
}
