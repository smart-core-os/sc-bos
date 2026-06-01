// Command bacnet-sc-connect verifies a BACnet/SC hub connection: it dials the
// hub, completes the connect handshake, and prints what was negotiated. Use it
// when commissioning a node to check the URI, mutual-TLS certificates and any
// hub allow-listing of UUID/VMAC are all wired up correctly.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/smart-core-os/sc-bos/cmd/tools/bacnet-sc/internal/scclient"
)

var hold = flag.Duration("hold", 0, "Hold the connection open for this duration after handshake (sends/answers heartbeats)")

func main() {
	flags := scclient.Register(flag.CommandLine)
	flag.Parse()

	client, err := flags.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(1)
	}
	defer client.Close()

	info := connectInfo{
		Hub:        flags.Hub,
		HubVMAC:    client.HubVMAC().String(),
		OurVMAC:    client.OurVMAC().String(),
		DeviceUUID: uuid.UUID(client.DeviceUUID()).String(),
	}
	if flags.JSON {
		_ = json.NewEncoder(os.Stdout).Encode(info)
	} else {
		fmt.Printf("connected\n")
		fmt.Printf("  hub:         %s\n", info.Hub)
		fmt.Printf("  hub vmac:    %s\n", info.HubVMAC)
		fmt.Printf("  our vmac:    %s\n", info.OurVMAC)
		fmt.Printf("  device uuid: %s\n", info.DeviceUUID)
	}

	if *hold > 0 {
		if !flags.JSON {
			fmt.Printf("holding connection for %s...\n", hold.String())
		}
		time.Sleep(*hold)
	}
}

type connectInfo struct {
	Hub        string `json:"hub"`
	HubVMAC    string `json:"hubVMAC"`
	OurVMAC    string `json:"ourVMAC"`
	DeviceUUID string `json:"deviceUUID"`
}
