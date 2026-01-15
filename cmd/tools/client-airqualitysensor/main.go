// Command client-airqualitysensor provides a CLI tool for interacting with the [traits.NewAirQualitySensorApiClient].
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

var clientConfig client.Config
var (
	flagJson bool
)

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.BoolVar(&clientConfig.Get, "get", true, "perform a get request")
	flag.BoolVar(&clientConfig.Pull, "pull", false, "pull changes")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name for requests")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
	flag.BoolVar(&flagJson, "json", false, "output json")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
}

func run() error {
	log.Printf("dialling: %s", clientConfig.Endpoint)
	conn, err := client.NewConnection(clientConfig)
	if err != nil {
		return err
	}
	log.Printf("dialled")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := traits.NewAirQualitySensorApiClient(conn)

	get := func(c context.Context, name string) error {
		log.Printf("GetMeterReading %s", name)
		res, err := api.GetAirQuality(ctx, &traits.GetAirQualityRequest{Name: name})
		if err != nil {
			return err
		}
		log.Printf("%q got air quality %v", name, protoToString(res))
		return nil
	}

	pull := func(c context.Context, name string) error {
		log.Printf("PullMeterReadings %s", name)
		stream, err := api.PullAirQuality(ctx, &traits.PullAirQualityRequest{Name: name})
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		for {
			res, err := stream.Recv()
			if err != nil {
				return err
			}
			log.Printf("%q got change: %v", name, protoToString(res))
		}
	}

	grp, ctx := errgroup.WithContext(ctx)
	if clientConfig.Get {
		grp.Go(func() error {
			return get(ctx, clientConfig.Name)
		})
	}
	if clientConfig.Pull {
		grp.Go(func() error {
			return pull(ctx, clientConfig.Name)
		})
	}

	return grp.Wait()
}

func protoToString(m proto.Message) string {
	if flagJson {
		return protojson.Format(m)
	}
	return fmt.Sprint(m)
}
