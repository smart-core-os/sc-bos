// Command bacnet-sc-write writes a single property on a BACnet/SC device. The
// device is located by Who-Is on its instance id (the hub routes by VMAC).
//
//	bacnet-sc-write -hub wss://hub -cert ... -key ... -ca ... \
//	    -device 10000 -object AnalogValue:1 -property present-value \
//	    -type real -value 72.5 -priority 8
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
	deviceID   = flag.Int("device", 0, "Device instance id to write to (required)")
	objectStr  = flag.String("object", "", "Object id, e.g. AnalogValue:1 (required)")
	propertyID = flag.String("property", "present-value", "Property name or numeric id (default present-value)")
	valueStr   = flag.String("value", "", "Value to write, interpreted per -type (required, except for -type null)")
	valueType  = flag.String("type", "real", "Value type: real|unsigned|signed|boolean|string|null")
	priority   = flag.Uint("priority", 16, "Write priority (1=highest, 16=lowest)")
	confirm    = flag.Bool("confirm", false, "Required acknowledgement that you intend to write to a live device")
)

func main() {
	flags := scclient.Register(flag.CommandLine)
	flag.Parse()
	if *deviceID == 0 || *objectStr == "" {
		fmt.Fprintln(os.Stderr, "ERR: -device and -object are required")
		flag.Usage()
		os.Exit(2)
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "ERR: refusing to write without -confirm; this affects a live device")
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
	value, err := parseValue(*valueType, *valueStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR: -value/-type:", err)
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

	wp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: bactypes.ObjectID(objID),
			Properties: []bactypes.Property{
				{ID: propID, ArrayIndex: bactypes.ArrayAll, Data: value},
			},
		},
	}
	if err := client.WriteProperty(ctx, dev, wp, *priority); err != nil {
		fmt.Fprintln(os.Stderr, "ERR: WriteProperty:", err)
		os.Exit(1)
	}

	if flags.JSON {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
			"device":   *deviceID,
			"object":   objID.String(),
			"property": propID.String(),
			"value":    *valueStr,
			"type":     *valueType,
			"priority": *priority,
			"ok":       true,
		})
		return
	}
	fmt.Printf("ok: wrote %s.%s = %s (type=%s, priority=%d)\n", objID, propID, *valueStr, *valueType, *priority)
}

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

// parseValue converts the CLI -value string into the concrete Go type the
// gobacnet encoder maps to the BACnet primitive of the selected -type.
func parseValue(typ, s string) (any, error) {
	switch typ {
	case "real":
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return nil, err
		}
		return float32(f), nil
	case "unsigned":
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(n), nil
	case "signed":
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(n), nil
	case "boolean":
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, err
		}
		return b, nil
	case "string":
		return s, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported -type %q (use real|unsigned|signed|boolean|string|null)", typ)
	}
}
