// Command bacnet-sc-read reads a single property from a BACnet/SC device. The
// device is located by Who-Is on its instance id (the hub routes by VMAC).
//
//	bacnet-sc-read -hub wss://hub -cert ... -key ... -ca ... \
//	    -device 10000 -object AnalogInput:0 -property present-value
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"

	"github.com/smart-core-os/sc-bos/cmd/tools/bacnet-sc/internal/scclient"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

var (
	deviceID   = flag.Int("device", 0, "Device instance id to read from (required)")
	objectStr  = flag.String("object", "", "Object id, e.g. AnalogInput:0 (required)")
	propertyID = flag.String("property", "present-value", "Property name or numeric id (default present-value)")
)

func main() {
	flags := scclient.Register(flag.CommandLine)
	flag.Parse()
	if *deviceID == 0 || *objectStr == "" {
		fmt.Fprintln(os.Stderr, "ERR: -device and -object are required")
		flag.Usage()
		os.Exit(2)
	}

	objID, err := config.ObjectIDFromString(*objectStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR: -object:", err)
		os.Exit(2)
	}
	propID, err := parseProperty(*propertyID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR: -property:", err)
		os.Exit(2)
	}

	client, err := flags.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(1)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), flags.Timeout)
	defer cancel()

	devices, err := client.WhoIs(ctx, *deviceID, *deviceID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR: WhoIs:", err)
		os.Exit(1)
	}
	if len(devices) == 0 {
		fmt.Fprintf(os.Stderr, "ERR: no device with id %d responded\n", *deviceID)
		os.Exit(1)
	}
	dev := devices[0]

	rp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: bactypes.ObjectID(objID),
			Properties: []bactypes.Property{
				{ID: propID, ArrayIndex: bactypes.ArrayAll},
			},
		},
	}
	res, err := client.ReadProperty(ctx, dev, rp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR: ReadProperty:", err)
		os.Exit(1)
	}
	if len(res.Object.Properties) == 0 {
		fmt.Fprintln(os.Stderr, "ERR: no property in response")
		os.Exit(1)
	}
	value := res.Object.Properties[0].Data

	if flags.JSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"device":   *deviceID,
			"object":   objID.String(),
			"property": propID.String(),
			"value":    value,
		})
		return
	}
	fmt.Printf("%v\n", value)
}

// parseProperty accepts either a property name (e.g. "present-value") or a
// numeric id, mirroring config.PropertyID.UnmarshalJSON.
func parseProperty(s string) (property.ID, error) {
	if id, ok := property.FromString(s); ok {
		return id, nil
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("unknown property name or id %q", s)
	}
	return property.ID(n), nil
}
