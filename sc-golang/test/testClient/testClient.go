package main

import (
	"context"
	"fmt"
	"net/url"

	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-golang/pkg/client"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("Could not create logger: %v", err))
	}

	addr := "tcp://127.0.0.1:9443"

	u, err := url.Parse(addr)
	if err != nil {
		logger.Error("Could not parse address", zap.String("addr", addr), zap.Error(err))
	}

	// setup TLS creds
	c, err := credentials.NewClientTLSFromFile("test/certs/service.pem", "")
	// create a connection to the smart core server
	con := client.NewClient(u, client.AuthCredentials{Creds: c}, logger)

	ctx := context.Background()
	// get device list from server
	dev, err := con.GetDeviceList(ctx)
	if err != nil {
		logger.Error("Could not get device list", zap.Error(err))
	} else {

		logger.Debug("Got device list")

		for _, d := range dev {
			str := d.Name
			for _, t := range d.Traits {
				str += "\t" + t.GetName()
			}
			logger.Debug("  "+str, zap.String("device", d.String()))
		}
	}

}
