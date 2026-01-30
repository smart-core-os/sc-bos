package devices

import (
	"iter"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
)

// metadataCollector helps to combine multiple devicespb.Device into a devicespb.DevicesMetadata.
type metadataCollector struct {
	fields     []string
	md         *devicespb.DevicesMetadata
	seenFields map[string]*devicespb.DevicesMetadata_StringFieldCount
}

func newMetadataCollector(fields ...string) *metadataCollector {
	return &metadataCollector{
		fields:     fields,
		md:         &devicespb.DevicesMetadata{},
		seenFields: make(map[string]*devicespb.DevicesMetadata_StringFieldCount),
	}
}

func (m *metadataCollector) add(d *devicespb.Device) *devicespb.DevicesMetadata {
	m.md.TotalCount++
	for _, field := range m.fields {
		seen, ok := m.seenFields[field]
		if !ok {
			seen = &devicespb.DevicesMetadata_StringFieldCount{Field: field, Counts: make(map[string]uint32)}
			m.seenFields[field] = seen
			m.md.FieldCounts = append(m.md.FieldCounts, seen)
		}
		for val := range unique(getMessageString(field, d)) {
			seen.Counts[val]++
		}
	}
	return m.md
}

func (m *metadataCollector) remove(d *devicespb.Device) *devicespb.DevicesMetadata {
	m.md.TotalCount--
	for _, field := range m.fields {
		seen, ok := m.seenFields[field]
		if !ok {
			continue
		}
		for val := range unique(getMessageString(field, d)) {
			if seen.Counts[val] > 0 {
				seen.Counts[val]--
			}
		}
	}
	return m.md
}

func unique[T comparable](seq iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		// todo: benchmark this allocation
		seen := make(map[T]struct{})
		for v := range seq {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}
}
