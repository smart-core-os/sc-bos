package sc

import (
	"bytes"
	"testing"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

func TestBVLCRoundTrip(t *testing.T) {
	cases := []bvlcMessage{
		{Function: bvlcHeartbeatRequest, MessageID: 7},
		{Function: bvlcConnectRequest, MessageID: 1, Payload: []byte{1, 2, 3}},
		{
			Function:  bvlcEncapsulatedNPDU,
			MessageID: 42,
			HasOrig:   true,
			Orig:      VMAC{1, 2, 3, 4, 5, 6},
			HasDest:   true,
			Dest:      BroadcastVMAC,
			Payload:   []byte{0xDE, 0xAD, 0xBE, 0xEF},
		},
	}
	for _, want := range cases {
		got, err := decodeBVLC(want.encode())
		if err != nil {
			t.Fatalf("decode %s: %v", want.Function, err)
		}
		if got.Function != want.Function || got.MessageID != want.MessageID {
			t.Errorf("%s: header mismatch got %+v want %+v", want.Function, got, want)
		}
		if got.HasOrig != want.HasOrig || got.Orig != want.Orig {
			t.Errorf("%s: orig mismatch got %v/%v", want.Function, got.HasOrig, got.Orig)
		}
		if got.HasDest != want.HasDest || got.Dest != want.Dest {
			t.Errorf("%s: dest mismatch got %v/%v", want.Function, got.HasDest, got.Dest)
		}
		if !bytes.Equal(got.Payload, want.Payload) {
			t.Errorf("%s: payload mismatch got %v want %v", want.Function, got.Payload, want.Payload)
		}
	}
}

func TestDecodeBVLCRejectsOptions(t *testing.T) {
	// function, control=ctrlDataOptions, message id
	b := []byte{byte(bvlcEncapsulatedNPDU), ctrlDataOptions, 0x00, 0x01}
	if _, err := decodeBVLC(b); err == nil {
		t.Fatal("expected error for header options, got nil")
	}
}

func TestConnectInfoRoundTrip(t *testing.T) {
	want := connectInfo{
		VMAC:          VMAC{6, 5, 4, 3, 2, 1},
		DeviceUUID:    [16]byte{0xAA, 0xBB},
		MaxBVLCLength: 1497,
		MaxNPDULength: 1024,
	}
	got, err := decodeConnectInfo(want.encode())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseVMAC(t *testing.T) {
	cases := map[string]VMAC{
		"010203040506":      {1, 2, 3, 4, 5, 6},
		"01:02:03:04:05:06": {1, 2, 3, 4, 5, 6},
		"AA-BB-CC-DD-EE-FF": {0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}
	for in, want := range cases {
		got, err := ParseVMAC(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != want {
			t.Errorf("%s: got %v want %v", in, got, want)
		}
	}
	if _, err := ParseVMAC("0102"); err == nil {
		t.Error("expected error for short vmac")
	}
}

func TestAddressVMACMapping(t *testing.T) {
	bcast := bactypes.Address{Net: 0xFFFF}
	bcast.SetBroadcast(true)
	if got := addressToVMAC(bcast); got != BroadcastVMAC {
		t.Errorf("broadcast address -> %v, want broadcast vmac", got)
	}

	v := VMAC{9, 8, 7, 6, 5, 4}
	addr := vmacToAddress(v, 0, nil)
	if addressToVMAC(addr) != v {
		t.Errorf("round trip vmac mismatch: %v", addressToVMAC(addr))
	}
}
