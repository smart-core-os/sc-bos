package devices

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func Test_deviceMatchesQuery(t *testing.T) {
	// simple validation test, exhaustive tests are at a lower level
	t.Run("matches", func(t *testing.T) {
		query := &devicespb.Device_Query{
			Conditions: []*devicespb.Device_Query_Condition{
				{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Lighting"}},
				{Field: "metadata.location.more.floor", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "4"}},
				{Field: "metadata.nics", Value: &devicespb.Device_Query_Condition_Matches{Matches: &devicespb.Device_Query{
					Conditions: []*devicespb.Device_Query_Condition{
						{Field: "gateway", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "1.2.3.4"}},
						{Field: "assignment", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "DHCP"}},
					},
				}}},
			},
		}
		device := &devicespb.Device{
			Name: "Light on floor 4",
			Metadata: &traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "Lighting",
				},
				Location: &traits.Metadata_Location{
					More: map[string]string{
						"floor": "4",
					},
				},
				Nics: []*traits.Metadata_NIC{
					{Gateway: "1.2.3.4", Assignment: traits.Metadata_NIC_STATIC},
					{Gateway: "not1.2.3.4", Assignment: traits.Metadata_NIC_DHCP},
					{Gateway: "1.2.3.4", Assignment: traits.Metadata_NIC_DHCP},
				},
			},
		}
		got := deviceMatchesQuery(query, device)
		if !got {
			t.Fatalf("deviceMatchesQuery want true, got false")
		}
	})

	t.Run("not matches", func(t *testing.T) {
		tests := []struct {
			name  string
			query *devicespb.Device_Query
		}{
			{
				"floor mismatch",
				&devicespb.Device_Query{
					Conditions: []*devicespb.Device_Query_Condition{
						{Field: "metadata.location.more.floor", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "4"}},
					},
				},
			},
			{
				"subsystem mismatch",
				&devicespb.Device_Query{
					Conditions: []*devicespb.Device_Query_Condition{
						{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "BMS"}},
					},
				},
			},
			{
				"nested mismatch",
				&devicespb.Device_Query{
					Conditions: []*devicespb.Device_Query_Condition{
						{Field: "metadata.nics", Value: &devicespb.Device_Query_Condition_Matches{Matches: &devicespb.Device_Query{
							Conditions: []*devicespb.Device_Query_Condition{
								{Field: "gateway", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "static.gw"}},
								{Field: "assignment", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "DHCP"}},
							},
						}}},
					},
				},
			},
		}
		device := &devicespb.Device{
			Name: "Light on floor 4",
			Metadata: &traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "Lighting",
				},
				Location: &traits.Metadata_Location{
					More: map[string]string{
						"floor": "5",
					},
				},
				Nics: []*traits.Metadata_NIC{
					{Gateway: "static.gw", Assignment: traits.Metadata_NIC_STATIC},
					{Gateway: "dhcp.gw", Assignment: traits.Metadata_NIC_DHCP},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := deviceMatchesQuery(tt.query, device)
				if got {
					t.Fatalf("deviceMatchesQuery want false, got true")
				}
			})
		}

	})
}

func TestServer_ListDevices(t *testing.T) {
	n := node.New("test")

	// create 40 devices, half are lights, half are hvac,
	// they're created interleaved to try and avoid page 1 being all lights and page 2 being all hvac
	for i := range 20 {
		n.Announce(fmt.Sprintf("device/%02d/light", i),
			node.HasMetadata(&traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "Lighting",
				},
			}),
			node.HasTrait(trait.Light))
		n.Announce(fmt.Sprintf("device/%02d/hvac", i),
			node.HasMetadata(&traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "HVAC",
				},
			}),
			node.HasTrait(trait.AirTemperature))
	}

	server := &Server{m: n}
	server.ChildPageSize = 5 // force multiple pages to be read from the parent

	mdNamed := func(name string) *traits.Metadata {
		return &traits.Metadata{
			Name:       name,
			Membership: &traits.Metadata_Membership{Subsystem: "Lighting"},
			Traits: []*traits.TraitMetadata{
				{Name: trait.Light.String()},
				{Name: trait.Metadata.String()},
			},
		}
	}

	wantPage1 := []*devicespb.Device{
		{Name: "device/00/light", Metadata: mdNamed("device/00/light")},
		{Name: "device/01/light", Metadata: mdNamed("device/01/light")},
		{Name: "device/02/light", Metadata: mdNamed("device/02/light")},
		{Name: "device/03/light", Metadata: mdNamed("device/03/light")},
		{Name: "device/04/light", Metadata: mdNamed("device/04/light")},
		{Name: "device/05/light", Metadata: mdNamed("device/05/light")},
		{Name: "device/06/light", Metadata: mdNamed("device/06/light")},
	}
	wantPage2 := []*devicespb.Device{
		{Name: "device/07/light", Metadata: mdNamed("device/07/light")},
		{Name: "device/08/light", Metadata: mdNamed("device/08/light")},
		{Name: "device/09/light", Metadata: mdNamed("device/09/light")},
		{Name: "device/10/light", Metadata: mdNamed("device/10/light")},
		{Name: "device/11/light", Metadata: mdNamed("device/11/light")},
		{Name: "device/12/light", Metadata: mdNamed("device/12/light")},
		{Name: "device/13/light", Metadata: mdNamed("device/13/light")},
	}
	wantPage3 := []*devicespb.Device{
		{Name: "device/14/light", Metadata: mdNamed("device/14/light")},
		{Name: "device/15/light", Metadata: mdNamed("device/15/light")},
		{Name: "device/16/light", Metadata: mdNamed("device/16/light")},
		{Name: "device/17/light", Metadata: mdNamed("device/17/light")},
		{Name: "device/18/light", Metadata: mdNamed("device/18/light")},
		{Name: "device/19/light", Metadata: mdNamed("device/19/light")},
	}

	// PAGE 1 - should return a full page
	devices, err := server.ListDevices(context.Background(), &devicespb.ListDevicesRequest{
		PageSize: 7,
		Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
			{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Lighting"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantPage1, devices.Devices, protocmp.Transform()); diff != "" {
		t.Fatalf("Page 1 (-want,+got)\n%s", diff)
	}
	if devices.NextPageToken == "" {
		t.Fatalf("Expecting a NextPageToken, but got none")
	}

	// PAGE 2 - should also return a full page
	devices, err = server.ListDevices(context.Background(), &devicespb.ListDevicesRequest{
		PageSize: 7,
		Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
			{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Lighting"}},
		}},
		PageToken: devices.NextPageToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantPage2, devices.Devices, protocmp.Transform()); diff != "" {
		t.Fatalf("Page 2 (-want,+got)\n%s", diff)
	}
	if devices.NextPageToken == "" {
		t.Fatalf("Expecting a NextPageToken, but got none")
	}

	// PAGE 3 - is not a full page, and should have no page token
	devices, err = server.ListDevices(context.Background(), &devicespb.ListDevicesRequest{
		PageSize: 7,
		Query: &devicespb.Device_Query{Conditions: []*devicespb.Device_Query_Condition{
			{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Lighting"}},
		}},
		PageToken: devices.NextPageToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantPage3, devices.Devices, protocmp.Transform()); diff != "" {
		t.Fatalf("Page 3 (-want,+got)\n%s", diff)
	}
	if devices.NextPageToken != "" {
		t.Fatalf("Expecting no NextPageToken, but got %s", devices.NextPageToken)
	}
}

func TestServer_GetDevicesMetadata(t *testing.T) {
	n := node.New("test")

	// create devices with different subsystems and locations
	for i := range 10 {
		n.Announce(fmt.Sprintf("devices/LTF-%03d", i+1),
			node.HasMetadata(&traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "Lighting",
				},
				Location: &traits.Metadata_Location{
					More: map[string]string{
						"floor": fmt.Sprintf("%d", i%3+1), // floors 1, 2, 3
					},
				},
			}),
			node.HasTrait(trait.Light))
	}
	for i := range 5 {
		n.Announce(fmt.Sprintf("devices/FCU-%03d", i+1),
			node.HasMetadata(&traits.Metadata{
				Membership: &traits.Metadata_Membership{
					Subsystem: "HVAC",
				},
				Location: &traits.Metadata_Location{
					More: map[string]string{
						"floor": fmt.Sprintf("%d", i%2+1), // floors 1, 2
					},
				},
			}),
			node.HasTrait(trait.AirTemperature))
	}

	server := &Server{m: n}

	t.Run("all devices", func(t *testing.T) {
		metadata, err := server.GetDevicesMetadata(context.Background(), &devicespb.GetDevicesMetadataRequest{
			Includes: &devicespb.DevicesMetadata_Include{
				Fields: []string{"metadata.membership.subsystem", "metadata.location.more.floor"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if metadata.TotalCount != 16 {
			t.Errorf("Expected total count 16, got %d", metadata.TotalCount)
		}

		// Check field counts
		subsystemCounts := findFieldCount(metadata.FieldCounts, "metadata.membership.subsystem")
		if subsystemCounts == nil {
			t.Fatal("Missing subsystem field counts")
		}
		if subsystemCounts.Counts["Lighting"] != 10 {
			t.Errorf("Expected 10 Lighting devices, got %d", subsystemCounts.Counts["Lighting"])
		}
		if subsystemCounts.Counts["HVAC"] != 5 {
			t.Errorf("Expected 5 HVAC devices, got %d", subsystemCounts.Counts["HVAC"])
		}

		floorCounts := findFieldCount(metadata.FieldCounts, "metadata.location.more.floor")
		if floorCounts == nil {
			t.Fatal("Missing floor field counts")
		}
		if floorCounts.Counts["1"] != 4+3 { // 4 lights + 3 hvac
			t.Errorf("Expected 6 devices on floor 1, got %d", floorCounts.Counts["1"])
		}
		if floorCounts.Counts["2"] != 3+2 { // 3 lights + 2 hvac
			t.Errorf("Expected 6 devices on floor 2, got %d", floorCounts.Counts["2"])
		}
		if floorCounts.Counts["3"] != 3 { // 3 lights
			t.Errorf("Expected 3 devices on floor 3, got %d", floorCounts.Counts["3"])
		}
	})

	t.Run("filtered by subsystem", func(t *testing.T) {
		metadata, err := server.GetDevicesMetadata(context.Background(), &devicespb.GetDevicesMetadataRequest{
			Query: &devicespb.Device_Query{
				Conditions: []*devicespb.Device_Query_Condition{
					{Field: "metadata.membership.subsystem", Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: "Lighting"}},
				},
			},
			Includes: &devicespb.DevicesMetadata_Include{
				Fields: []string{"metadata.location.more.floor"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		if metadata.TotalCount != 10 {
			t.Fatalf("Expected total count 10, got %d", metadata.TotalCount)
		}

		floorCounts := findFieldCount(metadata.FieldCounts, "metadata.location.more.floor")
		if floorCounts == nil {
			t.Fatal("Missing floor field counts")
		}
		if floorCounts.Counts["1"] != 4 {
			t.Fatalf("Expected 4 lighting devices on floor 1, got %d", floorCounts.Counts["1"])
		}
		if floorCounts.Counts["2"] != 3 {
			t.Fatalf("Expected 3 lighting devices on floor 2, got %d", floorCounts.Counts["2"])
		}
		if floorCounts.Counts["3"] != 3 {
			t.Fatalf("Expected 3 lighting devices on floor 3, got %d", floorCounts.Counts["3"])
		}
	})

	t.Run("no includes", func(t *testing.T) {
		metadata, err := server.GetDevicesMetadata(context.Background(), &devicespb.GetDevicesMetadataRequest{})
		if err != nil {
			t.Fatal(err)
		}

		if metadata.TotalCount != 16 {
			t.Fatalf("Expected total count 16, got %d", metadata.TotalCount)
		}

		if len(metadata.FieldCounts) != 0 {
			t.Fatalf("Expected no field counts when includes is empty, got %d", len(metadata.FieldCounts))
		}
	})
}

func findFieldCount(fieldCounts []*devicespb.DevicesMetadata_StringFieldCount, field string) *devicespb.DevicesMetadata_StringFieldCount {
	for _, fc := range fieldCounts {
		if fc.Field == field {
			return fc
		}
	}
	return nil
}
