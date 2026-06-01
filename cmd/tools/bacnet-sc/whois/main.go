// Command bacnet-sc-whois broadcasts a Who-Is over a BACnet/SC hub connection
// and prints the I-Am responses. The hub routes the broadcast to all connected
// nodes; each replies with its device id and VMAC.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	bactypes "github.com/smart-core-os/gobacnet/types"

	"github.com/smart-core-os/sc-bos/cmd/tools/bacnet-sc/internal/scclient"
)

var (
	low  = flag.Int("low", -1, "Lowest device instance to include (-1 for all)")
	high = flag.Int("high", -1, "Highest device instance to include (-1 for all)")
)

func main() {
	flags := scclient.Register(flag.CommandLine)
	flag.Parse()

	client, err := flags.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), flags.Timeout)
	defer cancel()

	devices, err := client.WhoIs(ctx, *low, *high)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(1)
	}

	if flags.JSON {
		out := make([]deviceRow, len(devices))
		for i, d := range devices {
			out[i] = toRow(d)
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DeviceID\tVMAC\tNet\tAddr\tMaxAPDU\tVendor")
	for _, d := range devices {
		r := toRow(d)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\n", r.DeviceID, r.VMAC, r.Net, r.Addr, r.MaxAPDU, r.Vendor)
	}
	_ = w.Flush()
}

type deviceRow struct {
	DeviceID uint32 `json:"deviceID"`
	VMAC     string `json:"vmac"`
	Net      string `json:"net,omitempty"`
	Addr     string `json:"addr,omitempty"`
	MaxAPDU  uint32 `json:"maxAPDU"`
	Vendor   uint32 `json:"vendor"`
}

func toRow(d bactypes.Device) deviceRow {
	r := deviceRow{
		DeviceID: uint32(d.ID.Instance),
		VMAC:     vmacOf(d.Addr),
		MaxAPDU:  d.MaxApdu,
		Vendor:   d.Vendor,
	}
	if d.Addr.Net != 0 {
		r.Net = fmt.Sprintf("%d", d.Addr.Net)
	}
	if len(d.Addr.Adr) > 0 {
		r.Addr = fmt.Sprintf("%d", d.Addr.Adr[0])
		for _, b := range d.Addr.Adr[1:] {
			r.Addr += fmt.Sprintf(".%d", b)
		}
	}
	return r
}

// vmacOf returns the device's 6-octet VMAC as hex, or an empty string when the
// address does not carry one (e.g. when the device is unreachable).
func vmacOf(a bactypes.Address) string {
	if len(a.Mac) != 6 {
		return ""
	}
	return hex.EncodeToString(a.Mac)
}
