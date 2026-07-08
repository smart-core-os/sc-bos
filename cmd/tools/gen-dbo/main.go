// Command gen-dbo generates a Google Digital Buildings building-configuration file
// from a Smart Core node's meter devices. It dials a node, lists meter devices,
// reads their reading support (for units/capability), and emits one DBO entity per
// meter with an identity translation over the DBO standard field names the
// sccexporter publishes. See .claude/plans/dbo-conformance-plan.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

var clientConfig client.Config
var outFile string

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name to interrogate for the device list")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
	flag.StringVar(&outFile, "out", "", "output file (default stdout); use a path to write a file")
}

func main() {
	flag.Parse()

	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cleanup()

	conn, err := client.NewConnection(clientConfig)
	if err != nil {
		panic(err)
	}

	devicesClient := devicespb.NewDevicesApiClient(conn)
	meterInfo := meterpb.NewMeterInfoClient(conn)

	meters, err := listMeters(ctx, devicesClient)
	if err != nil {
		panic(err)
	}

	bc := BuildingConfig{}
	for _, device := range meters {
		var support *meterpb.MeterReadingSupport
		if s, err := meterInfo.DescribeMeterReading(ctx, &meterpb.DescribeMeterReadingRequest{Name: device.Name}); err != nil {
			log.Printf("no meter info for %s (units will be omitted): %v", device.Name, err)
		} else {
			support = s
		}
		guid, entity := meterEntity(device.Name, support)
		bc[guid] = entity
	}

	data, err := bc.Marshal()
	if err != nil {
		panic(err)
	}

	out := os.Stdout
	if outFile != "" {
		log.Println("writing building config to", outFile)
		f, err := os.Create(outFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		out = f
	}
	if _, err := out.Write(data); err != nil {
		panic(err)
	}
}

// listMeters returns all devices implementing the Meter trait, paging through the
// device list. Each returned Device carries its metadata inline.
func listMeters(ctx context.Context, devicesClient devicespb.DevicesApiClient) ([]*devicespb.Device, error) {
	req := &devicespb.ListDevicesRequest{
		PageSize: 1000,
		Query: &devicespb.Device_Query{
			Conditions: []*devicespb.Device_Query_Condition{{
				Field: "metadata.traits.name",
				Value: &devicespb.Device_Query_Condition_StringEqual{StringEqual: string(meterpb.TraitName)},
			}},
		},
	}

	var all []*devicespb.Device
	for {
		res, err := devicesClient.ListDevices(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("list devices: %w", err)
		}
		all = append(all, res.Devices...)
		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}
	return all, nil
}
