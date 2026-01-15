package devices

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
)

func TestMetadataCollection_repeated(t *testing.T) {
	c := newMetadataCollector("metadata.nics.assignment")
	c.add(&devicespb.Device{
		Metadata: &traits.Metadata{
			Nics: []*traits.Metadata_NIC{
				{Assignment: traits.Metadata_NIC_DHCP},
			},
		},
	})
	c.add(&devicespb.Device{
		Metadata: &traits.Metadata{
			Nics: []*traits.Metadata_NIC{
				{Assignment: traits.Metadata_NIC_STATIC},
			},
		},
	})

	d := &devicespb.Device{
		Metadata: &traits.Metadata{
			Nics: []*traits.Metadata_NIC{
				{Assignment: traits.Metadata_NIC_DHCP},
				{Assignment: traits.Metadata_NIC_DHCP},
				{Assignment: traits.Metadata_NIC_STATIC},
				{Assignment: traits.Metadata_NIC_STATIC},
			},
		},
	}

	got := c.add(d)
	want := &devicespb.DevicesMetadata{
		TotalCount: 3,
		FieldCounts: []*devicespb.DevicesMetadata_StringFieldCount{
			{
				Field: "metadata.nics.assignment",
				Counts: map[string]uint32{
					"DHCP":   2,
					"STATIC": 2,
				},
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatalf("GetDevicesMetadata.add (-want,+got):\n%s", diff)
	}

	got = c.remove(d)
	want = &devicespb.DevicesMetadata{
		TotalCount: 2,
		FieldCounts: []*devicespb.DevicesMetadata_StringFieldCount{
			{
				Field: "metadata.nics.assignment",
				Counts: map[string]uint32{
					"DHCP":   1,
					"STATIC": 1,
				},
			},
		},
	}
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Fatalf("GetDevicesMetadata.remove (-want,+got):\n%s", diff)
	}
}
