// Command client-securityevent provides a CLI tool for interacting with the [securityeventpb.SecurityEventApiClient].
package main

import (
	"context"
	"flag"
	"log"

	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
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

func run() error {
	log.Printf("dialling: %s", clientConfig.Endpoint)
	conn, err := client.NewConnection(clientConfig)
	if err != nil {
		return err
	}
	log.Printf("dialled")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	em := securityeventpb.NewSecurityEventApiClient(conn)

	get := func(c context.Context, name string) error {
		req := securityeventpb.ListSecurityEventsRequest{Name: name}
		for {
			log.Printf("ListSecurityEvents %s", name)
			res, err := em.ListSecurityEvents(ctx, &req)
			if err != nil {
				return err
			}
			log.Printf("got %d events for %s", len(res.SecurityEvents), name)
			for _, event := range res.SecurityEvents {
				log.Printf("event: %v", event)
			}
			if res.NextPageToken == "" {
				break
			}
			req.PageToken = res.NextPageToken
		}
		return nil
	}

	pull := func(c context.Context, name string) error {
		log.Printf("PullSecurityEvents %s", name)
		stream, err := em.PullSecurityEvents(ctx, &securityeventpb.PullSecurityEventsRequest{Name: name})
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
				log.Printf("got change for %s: %v", name, change.NewValue)
			}
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
