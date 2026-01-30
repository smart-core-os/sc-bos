// Command enrollment-tool provides a CLI tool for enrolling a node with a hub.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
	"github.com/smart-core-os/sc-bos/pkg/proto/enrollmentpb"
	"github.com/smart-core-os/sc-bos/pkg/system/hub/remote"
)

var (
	flagAddr        string
	flagCert        string
	flagKey         string
	flagCA          string
	flagName        string
	flagManagerName string
	flagManagerAddr string
)

func init() {
	flag.StringVar(&flagAddr, "addr", "localhost:23557", "address (host:port) of Area Controller gRPC server")
	flag.StringVar(&flagCert, "cert", "", "path to App Server intermediate CA certificate")
	flag.StringVar(&flagKey, "key", "", "path to App Server intermediate CA private key")
	flag.StringVar(&flagCA, "ca", "", "path to root CA")
	flag.StringVar(&flagName, "name", "", "name to assign to the enrolled node")
	flag.StringVar(&flagManagerName, "manager-name", "", "Smart Core name of the node's new manager")
	flag.StringVar(&flagManagerAddr, "manager-addr", "", "Address (host:port) of the node's new manager")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
}

func run() error {
	if flagCert == "" || flagKey == "" {
		return errors.New("cert and key options must be specified")
	}
	authority := pki.FSSource(flagCert, flagKey, flagCA)

	en := &enrollmentpb.Enrollment{
		TargetName:     flagName,
		TargetAddress:  flagAddr,
		ManagerName:    flagManagerName,
		ManagerAddress: flagManagerAddr,
	}
	en, err := remote.Enroll(context.Background(), en, authority)
	fmt.Println(protojson.Format(en))
	return err
}
