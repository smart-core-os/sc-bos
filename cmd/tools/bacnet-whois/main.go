// Command bacnet-whois executes a BACnet WhoIs broadcast request and captures the replies in a CSV file.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"os"
	"time"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-bos/pkg/util/netutil"
)

var (
	LocalPort      = 47808
	LocalInterface = ""
	OutFile        = "bacnet-iam.csv"
	ScanSize       = bactypes.MaxInstance/4 + 1
)

func init() {
	flag.IntVar(&LocalPort, "port", LocalPort, "Local port to listen on")
	flag.StringVar(&LocalInterface, "iface", LocalInterface, "Local interface to listen on, defaults to the external ip interface")
	flag.StringVar(&OutFile, "out", OutFile, "Output file")
	flag.IntVar(&ScanSize, "scan-size", ScanSize, "Size of the block for each WhoIs request")
}

func main() {
	flag.Parse()
	if LocalInterface == "" {
		outAddr, err := netutil.OutboundAddr()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ifaceName, err := netutil.InterfaceNameForAddr(outAddr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		LocalInterface = ifaceName
	} else if addr, err := netip.ParseAddr(LocalInterface); err == nil {
		ifaceName, err := netutil.InterfaceNameForAddr(addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		LocalInterface = ifaceName
	}

	client, err := gobacnet.NewClient(LocalInterface, LocalPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()
	// client.Log.Level = logrus.FatalLevel

	outFile, err := os.Create(OutFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	fmt.Fprintf(outFile, "BACnet Device ID,IP:Port,Network,Address,Max APDU,Segmentation,Vendor\n")

	doWhoIs(outFile, client, -1, -1)
}

func doWhoIs(out io.Writer, client *gobacnet.Client, min, max int) {
	if min < 0 {
		fmt.Printf("Finding all devices\n")
	} else {
		fmt.Printf("Finding devices with IDs %d-%d\n", min, max)
	}
	wait := 10 * time.Second
	ctx, stop := context.WithTimeout(context.Background(), wait)
	defer stop()
	iAm, err := client.WhoIs(ctx, min, max)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, device := range iAm {
		fprintDeviceAddr(out, device)
	}
}

func fprintDeviceAddr(w io.Writer, device bactypes.Device) {
	var ipPort string
	if udpAddr, err := device.Addr.UDPAddr(); err != nil {
		ipPort = fmt.Sprintf("<<%s>>", err)
	} else {
		ipPort = udpAddr.String()
	}

	var net string
	if device.Addr.Net != 0 {
		net = fmt.Sprintf("%d", device.Addr.Net)
	}

	var adr string
	if octets := device.Addr.Adr; len(octets) > 0 {
		adr = fmt.Sprintf("%d", octets[0])
		for _, octet := range octets[1:] {
			adr += fmt.Sprintf(".%d", octet)
		}
	}

	var maxAPDU string
	if device.MaxApdu != 0 {
		maxAPDU = fmt.Sprintf("%d", device.MaxApdu)
	}
	fmt.Fprintf(w, "%d,%s,%s,%s,%s,%d,%d\n", device.ID.Instance, ipPort, net, adr, maxAPDU, device.Segmentation, device.Vendor)
}
