package bacnet

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
)

func TestYabeRoom(t *testing.T) {
	if _, ok := os.LookupEnv("YABE"); !ok {
		t.Skip("Skipping test that relies on YABE room simulator")
	}
	client, err := gobacnet.NewClient("bridge100", 0)
	if err != nil {
		t.Fatalf("NewClient error %v", err)
	}
	t.Cleanup(client.Close)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	iAm, err := client.WhoIs(ctx, 800_000, 900_000)
	if err != nil {
		t.Fatalf("WhoIs error %v", err)
	}

	t.Logf("IAm %+v", iAm)

	netAddr, err := iAm[0].Addr.UDPAddr()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Connecting to %v", netAddr)
	bacAddr := bactypes.UDPToAddress(&netAddr)
	t.Logf("After conversion: iAm Addr:%v, udp addr:%v, bac addr:%v", iAm[0].Addr, netAddr, bacAddr)
	devices, err := client.RemoteDevices(ctx, bacAddr, iAm[0].ID.Instance)
	if err != nil {
		t.Fatalf("RemoteDevices error %v", err)
	}
	t.Logf("RemoteDevices %+v", devices)
}
