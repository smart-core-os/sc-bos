// Command bacnet-comm-test performs a simple comm check against a BACnet device.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
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
	nic, localPortStr, _ := strings.Cut(nicPort, ":")
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("Connecting to %v", bacAddr)
	devices, err := client.RemoteDevices(ctx, bacAddr, bactypes.ObjectInstance(deviceNum))
	if err != nil {
		log.Fatalf("Error reading device info! %v", err)
	}
	if len(devices) == 0 {
		log.Fatal("no devices returned")
	}
	dev := devices[0]
	log.Printf("Device %d  vendor=%d  maxApdu=%d", dev.ID.Instance, dev.Vendor, dev.MaxApdu)

	log.Printf("Fetching object list...")
	dev, err = client.Objects(ctx, dev)
	if err != nil {
		log.Fatalf("Error reading objects! %v", err)
	}

	type entry struct {
		objType  objecttype.ObjectType
		instance bactypes.ObjectInstance
		name     string
		value    string
	}
	var entries []entry
	for objType, instances := range dev.Objects {
		for instance, obj := range instances {
			entries = append(entries, entry{objType, instance, obj.Name, ""})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].objType != entries[j].objType {
			return entries[i].objType < entries[j].objType
		}
		return entries[i].instance < entries[j].instance
	})

	log.Printf("Fetching present values for %d objects...", len(entries))
	for i := range entries {
		e := &entries[i]
		rp := bactypes.ReadPropertyData{
			Object: bactypes.Object{
				ID: bactypes.ObjectID{Type: e.objType, Instance: e.instance},
				Properties: []bactypes.Property{{
					ID:         property.PresentValue,
					ArrayIndex: bactypes.ArrayAll,
				}},
			},
		}
		resp, err := client.ReadProperty(ctx, dev, rp)
		if err != nil {
			e.value = "-"
		} else if len(resp.Object.Properties) > 0 {
			e.value = fmt.Sprintf("%v", resp.Object.Properties[0].Data)
		} else {
			e.value = "-"
		}
	}

	fmt.Printf("\n%-30s %-10s %-40s %s\n", "Type", "Instance", "Name", "Present Value")
	fmt.Println(strings.Repeat("-", 100))
	for _, e := range entries {
		fmt.Printf("%-30s %-10d %-40s %s\n", e.objType, e.instance, e.name, e.value)
	}
	fmt.Printf("\nTotal: %d objects\n", len(entries))
}
