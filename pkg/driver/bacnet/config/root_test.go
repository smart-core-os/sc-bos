package config

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

//go:embed testdata
var testdata embed.FS

func TestJSON(t *testing.T) {
	// We're mostly testing that json marshal/unmarshal don't error here

	fileBytes, err := fs.ReadFile(testdata, "testdata/sample.json")
	if err != nil {
		t.Fatal(err)
	}

	var root Root
	err = json.Unmarshal(fileBytes, &root)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Got root %+v", root)
	t.Logf("root.cov %+v", root.COV)
	t.Logf("root.discovert %+v", root.Discovery)

	outBytes, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Written: %s", outBytes)
}

func TestDestinationAddress_Bytes(t *testing.T) {
	tests := []struct {
		name string
		in   DestinationAddress
		out  []byte
	}{
		{
			name: "empty",
			in:   "",
			out:  nil,
		},
		{
			name: "simple",
			in:   "15",
			out:  []byte{15},
		},
		{
			name: "longer",
			in:   "4.15",
			out:  []byte{4, 15},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("wrong bytes (-want +got):\n%s", diff)
			}
		})
	}
}

func TestComm_ToAddress(t *testing.T) {
	ip := netip.AddrPortFrom(netip.AddrFrom4([4]byte{10, 11, 12, 13}), 0xBAC0)
	tests := []struct {
		name string
		in   *Comm
		out  bactypes.Address
	}{
		{
			name: "empty",
			in:   &Comm{},
			out:  bactypes.Address{},
		},
		{
			name: "w/IP",
			in: &Comm{
				IP: &ip,
			},
			out: bactypes.Address{
				MacLen: 6,
				Mac:    []uint8{10, 11, 12, 13, 0xBA, 0xC0},
			},
		},
		{
			name: "w/dest",
			in: &Comm{
				IP: &ip,
				Destination: &Destination{
					Network: 50011,
					Address: "15",
				},
			},
			out: bactypes.Address{
				MacLen: 6,
				Mac:    []uint8{10, 11, 12, 13, 0xBA, 0xC0},
				Adr:    []uint8{15},
				Len:    1,
				Net:    50011,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.ToAddress()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("wrong bytes (-want +got):\n%s", diff)
			}
		})
	}
}
