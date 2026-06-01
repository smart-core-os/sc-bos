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
	if l := len(args); l < 3 || l > 7 {
		log.Fatalf("Usage: <cmd> nic[:port] server[:port] [device] [-write Type:instance=value] [-priority 1-16]")
	}
	nicPort, serverPort := args[1], args[2]
	deviceStr := "4194303"
	writeSpec := ""
	writePriority := uint(8)
	for i := 3; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-write=") {
			writeSpec = strings.TrimPrefix(args[i], "-write=")
		} else if args[i] == "-write" && i+1 < len(args) {
			i++
			writeSpec = args[i]
		} else if strings.HasPrefix(args[i], "-priority=") {
			p, err := strconv.ParseUint(strings.TrimPrefix(args[i], "-priority="), 10, 8)
			if err != nil || p < 1 || p > 16 {
				log.Fatal("bad priority: must be 1-16")
			}
			writePriority = uint(p)
		} else if args[i] == "-priority" && i+1 < len(args) {
			i++
			p, err := strconv.ParseUint(args[i], 10, 8)
			if err != nil || p < 1 || p > 16 {
				log.Fatal("bad priority: must be 1-16")
			}
			writePriority = uint(p)
		} else if deviceStr == "4194303" {
			deviceStr = args[i]
		}
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
		entries[i].value = readPresentValue(ctx, client, dev, entries[i].objType, entries[i].instance)
	}

	fmt.Printf("\n%-30s %-10s %-40s %s\n", "Type", "Instance", "Name", "Present Value")
	fmt.Println(strings.Repeat("-", 100))
	for _, e := range entries {
		fmt.Printf("%-30s %-10d %-40s %s\n", e.objType, e.instance, e.name, e.value)
	}
	fmt.Printf("\nTotal: %d objects\n", len(entries))

	if writeSpec == "" {
		return
	}

	// Parse -write Type:instance=value
	keyPart, rawValue, ok := strings.Cut(writeSpec, "=")
	if !ok {
		log.Fatalf("-write: expected Type:instance=value, got %q", writeSpec)
	}
	typePart, instancePart, ok := strings.Cut(keyPart, ":")
	if !ok {
		log.Fatalf("-write: expected Type:instance, got %q", keyPart)
	}
	writeType, ok := objecttype.FromString(typePart)
	if !ok {
		log.Fatalf("-write: unknown object type %q", typePart)
	}
	writeInstance, err := strconv.ParseUint(instancePart, 10, 32)
	if err != nil {
		log.Fatalf("-write: bad instance %q: %v", instancePart, err)
	}
	writeValue, err := parseWriteValue(rawValue)
	if err != nil {
		log.Fatalf("-write: bad value %q: %v", rawValue, err)
	}

	writeObjID := bactypes.ObjectID{Type: writeType, Instance: bactypes.ObjectInstance(writeInstance)}
	wp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: writeObjID,
			Properties: []bactypes.Property{{
				ID:         property.PresentValue,
				ArrayIndex: bactypes.ArrayAll,
				Data:       writeValue,
			}},
		},
	}

	fmt.Printf("\nWriting %v to %s:%d (priority %d)...\n", rawValue, writeType, writeInstance, writePriority)
	if err := client.WriteProperty(ctx, dev, wp, writePriority); err != nil {
		fmt.Printf("Write failed: %v\n", err)
		fmt.Println("Hint: analog objects need a decimal value (e.g. 18.0); binary/multi-state use integers")
	} else {
		fmt.Println("Write succeeded")
	}

	readBack := readPresentValue(ctx, client, dev, writeType, bactypes.ObjectInstance(writeInstance))
	fmt.Printf("Read back %s:%d present-value = %s\n", writeType, writeInstance, readBack)
}

func readPresentValue(ctx context.Context, client *gobacnet.Client, dev bactypes.Device, objType objecttype.ObjectType, instance bactypes.ObjectInstance) string {
	rp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: bactypes.ObjectID{Type: objType, Instance: instance},
			Properties: []bactypes.Property{{
				ID:         property.PresentValue,
				ArrayIndex: bactypes.ArrayAll,
			}},
		},
	}
	resp, err := client.ReadProperty(ctx, dev, rp)
	if err != nil || len(resp.Object.Properties) == 0 {
		return "-"
	}
	return fmt.Sprintf("%v", resp.Object.Properties[0].Data)
}

// parseWriteValue converts a string to the appropriate BACnet Go type.
// "true"/"false" → bool, integer strings → uint32, decimal strings → float32.
func parseWriteValue(s string) (interface{}, error) {
	switch strings.ToLower(s) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	if i, err := strconv.ParseUint(s, 10, 32); err == nil {
		return uint32(i), nil
	}
	if f, err := strconv.ParseFloat(s, 32); err == nil {
		return float32(f), nil
	}
	return nil, fmt.Errorf("cannot parse %q as bool, uint32, or float32", s)
}
