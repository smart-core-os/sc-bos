package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/smart-core-os/sc-api/go/info"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-golang/pkg/server"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func main() {

	z, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("Could not create logger: %v", err))
	}

	ctx := context.Background()

	// setup TLS creds
	c, err := credentials.NewServerTLSFromFile("test/certs/service.pem", "test/certs/service.key")
	if err != nil {
		z.Fatal("Could not create credentials", zap.Error(err))
	}
	s := server.NewServer(ctx, server.NewAuthProvider(c, z), z)

	addr, err := url.Parse("tcp://127.0.0.1:9443")
	done := s.Startup(addr)

	t1 := &info.Trait{
		Name: string(trait.Light),
	}

	s.RegisterDevice(&info.Device{
		Name:   "/test/device",
		Traits: []*info.Trait{t1},
		Title:  "Test Device",
	})

	// wait for termination
	select {
	case err := <-done:
		// something caused an error
		if err != nil {
			log.Printf("Shutting down: %v", err)
		} else {
			log.Println("Shutting down")
		}
	case <-ctx.Done():
		// interrupt
		log.Println("Shutting down")
	}

}
