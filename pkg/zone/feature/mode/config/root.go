package config

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"

	"github.com/smart-core-os/sc-bos/pkg/zone"
)

type Root struct {
	zone.Config

	Modes map[string][]Option `json:"modes,omitempty"`
}

func ReadConfig(in io.Reader) (Root, error) {
	var root Root
	if err := json.NewDecoder(in).Decode(&root); err != nil {
		return Root{}, err
	}
	root.Hydrate()
	return root, nil
}

func ReadConfigBytes(in []byte) (Root, error) {
	return ReadConfig(bytes.NewReader(in))
}

func (r *Root) WriteConfig(out io.Writer) error {
	r.Unhydrate()
	return json.NewEncoder(out).Encode(r)
}

// Hydrate fills defaults with their actual values.
// Called as part of ReadConfig.
// r is modified in place, each OptionSource will have its Mode and Value set based on the Modes key or Option.Name.
func (r *Root) Hydrate() {
	for mode, options := range r.Modes {
		for oi, option := range options {
			if option.Key == "" {
				option.Key = mode
			}
			if option.Value == "" {
				option.Value = option.Name
			}
			for si, source := range option.Sources {
				if source.Mode == "" {
					source.Mode = option.Key
				}
				if source.Value == "" {
					source.Value = option.Value
				}
				option.Sources[si] = source
			}
			options[oi] = option
		}
		r.Modes[mode] = options
	}
}

// Unhydrate removes defaults and replaces them with empty strings.
// Called as part of WriteConfig.
// r is modified in place, each OptionSource will have its Mode and Value set to "" if it is equal to Modes key or Option.Name.
func (r *Root) Unhydrate() {
	for mode, options := range r.Modes {
		for oi, option := range options {
			for si, source := range option.Sources {
				if source.Mode == option.Key {
					source.Mode = ""
				}
				if source.Value == option.Value {
					source.Value = ""
				}
				option.Sources[si] = source
			}
			if option.Key == mode {
				option.Key = ""
			}
			if option.Value == option.Name {
				option.Value = ""
			}
			options[oi] = option
		}
		r.Modes[mode] = options
	}
}

func (r Root) AllDeviceNames() []string {
	var dst []string
	for _, options := range r.Modes {
		for _, option := range options {
			for _, source := range option.Sources {
				for _, device := range source.Devices {
					i := sort.SearchStrings(dst, device)
					switch {
					case i == len(dst):
						dst = append(dst, device)
					case dst[i] != device:
						dst = append(dst, "")
						copy(dst[i+1:], dst[i:])
						dst[i] = device
						// else device is already in dst
					}
				}
			}
		}
	}
	return dst
}

type Option struct {
	Name    string           `json:"name,omitempty"`  // The name of the option. Used as default for OptionSource.Value
	Key     string           `json:"key,omitempty"`   // The mode name, defaults to Root.Modes key. e.g. "occupancy". Used as default for OptionSource.Mode
	Value   string           `json:"value,omitempty"` // The value used by this option, defaults to Option.Name. e.g. "occupied
	Sources []SourceOrString `json:"sources,omitempty"`
}

type OptionSource struct {
	Devices []string `json:"devices,omitempty"`
	Mode    string   `json:"mode,omitempty"`  // The mode name, defaults to Option.Key. e.g. "occupancy"
	Value   string   `json:"value,omitempty"` // The value used by this option, defaults to Option.Value. e.g. "occupied"
}

// SourceOrString is like OptionSource but for simple cases like `{"devices": ["foo"]}` un/marshals from/to `"foo"`.
type SourceOrString struct {
	OptionSource
}

func (m SourceOrString) MarshalJSON() ([]byte, error) {
	if m.OptionSource.Mode == "" && m.OptionSource.Value == "" && len(m.OptionSource.Devices) == 1 {
		return json.Marshal(m.OptionSource.Devices[0])
	}
	return json.Marshal(m.OptionSource)
}

func (m *SourceOrString) UnmarshalJSON(bytes []byte) error {
	if bytes[0] == '"' {
		var v string
		if err := json.Unmarshal(bytes, &v); err != nil {
			return err
		}
		*m = SourceOrString{OptionSource: OptionSource{Devices: []string{v}}}
		return nil
	}

	var v OptionSource
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err
	}
	*m = SourceOrString{OptionSource: v}

	return nil
}
