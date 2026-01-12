package remote

import (
	"context"
	"crypto/tls"
	"encoding/pem"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
)

// Inspect connects to a remote node returning its public certs and metadata.
func Inspect(ctx context.Context, address string) (*hubpb.HubNodeInspection, error) {
	tlsConfig := &tls.Config{
		// We're actively trying to connect to the remote and fetch their cert
		InsecureSkipVerify: true,
	}

	// capture the server cert which we'll eventually return to the caller
	creds := &certInterceptor{TransportCredentials: credentials.NewTLS(tlsConfig)}
	conn, err := grpc.NewClient(address,
		// capture the tls cert
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	out := &hubpb.HubNodeInspection{}
	// do the API call first, to force the connection to be established (or fail)
	client := traits.NewMetadataApiClient(conn)
	md, err := client.GetMetadata(ctx, &traits.GetMetadataRequest{})
	if err != nil {
		return nil, err
	}
	out.Metadata = md

	// if the remote has certs then encode them as PEM in the response
	if certs, ok := creds.PeerCertificates(); ok && len(certs) > 0 {
		for _, cert := range certs {
			pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
			out.PublicCerts = append(out.PublicCerts, string(pemBytes))
		}
	}

	return out, nil
}
