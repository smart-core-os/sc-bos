// Command bacnet-comm-test performs a simple comm check against a BACnet device.
package main

import (
	"context"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
)

func main() {
	args := os.Args
	if l := len(args); l < 3 || l > 4 {
		log.Fatalf("Usage: <cmd> nic[:port] server[:port] [device]")
	}
	nicPort, serverPort := args[1], args[2]
	deviceStr := "4194303"
	if len(args) == 4 {
		deviceStr = args[3]
	}

	localPort := 0 // defaults to 47808
	nic, localPortStr, _ := net.SplitHostPort(nicPort)
	if localPortStr != "" {
		var err error
		localPort, err = strconv.Atoi(localPortStr)
		if err != nil {
			log.Fatal("bad local port", localPortStr, err)
		}
	}

	deviceNum, err := strconv.ParseInt(deviceStr, 10, 32)
	if err != nil {
		log.Fatal("bad device", deviceStr, err)
	}

	client, err := gobacnet.NewClient(nic, localPort)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	uri, err := url.ParseRequestURI("bacnet://" + serverPort)
	if err != nil {
		log.Fatal("server", err)
	}
	portStr := uri.Port()
	if portStr == "" {
		portStr = "47808"
	}
	portNum, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		log.Fatal("server port", portStr, err)
	}
	ip := net.ParseIP(uri.Hostname())
	if ip == nil {
		log.Fatal("bad server ip", uri.Hostname())
	}
	bacAddr := bactypes.UDPToAddress(&net.UDPAddr{IP: ip, Port: int(portNum)})
	log.Printf("Connecting to %v", bacAddr)
	devices, err := client.RemoteDevices(ctx, bacAddr, bactypes.ObjectInstance(deviceNum))
	if err != nil {
		log.Fatalf("Error reading device info! %v", err)
	}
	log.Printf("Success! {devices=%+v}", devices)
}
