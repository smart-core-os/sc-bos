package main

import (
	"context"
	"flag"
	"log"

	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/proto/transportpb"
	"github.com/smart-core-os/sc-bos/pkg/util/client"
)

var clientConfig client.Config

func init() {
	flag.StringVar(&clientConfig.Endpoint, "endpoint", "localhost:23557", "smart core endpoint")
	flag.BoolVar(&clientConfig.Get, "get", true, "perform a get request")
	flag.BoolVar(&clientConfig.Pull, "pull", true, "pull changes")
	flag.StringVar(&clientConfig.Name, "name", "", "smart core name for requests")
	flag.BoolVar(&clientConfig.TLS.InsecureNoClientCert, "insecure-no-client-cert", false, "")
	flag.BoolVar(&clientConfig.TLS.InsecureSkipVerify, "insecure-skip-verify", false, "")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
}

func printTransport(res *transportpb.Transport) {
	log.Printf("------------------------------------")
	log.Printf("Actual Position: %s", res.ActualPosition.Floor)
	log.Printf("Moving Direction: %s", res.MovingDirection)
	if res.Load != nil {
		log.Printf("Load: %f", *res.Load)
	}
	for _, dest := range res.NextDestinations {
		log.Printf("Next Destination: %s", dest.Floor)
	}
	for _, door := range res.Doors {
		log.Printf("Door Status: %s", door.Status)
	}
	log.Printf("------------------------------------")
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
	tc := transportpb.NewTransportApiClient(conn)

	get := func(c context.Context, name string) error {
		req := transportpb.GetTransportRequest{Name: name}
		log.Printf("GetTransportRequest %s", name)
		res, err := tc.GetTransport(ctx, &req)
		if err != nil {
			return err
		}
		printTransport(res)
		return nil
	}

	pull := func(c context.Context, name string) error {
		log.Printf("PullTransportRequest %s", name)
		stream, err := tc.PullTransport(ctx, &transportpb.PullTransportRequest{Name: name})
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
			for _, change := range res.Changes {
				printTransport(change.Transport)
			}
		}
	}

	grp, ctx := errgroup.WithContext(ctx)
	if clientConfig.Get {
		err := get(ctx, clientConfig.Name)
		if err != nil {
			return err
		}
	}
	if clientConfig.Pull {
		grp.Go(func() error {
			return pull(ctx, clientConfig.Name)
		})
	}

	return grp.Wait()
}
