package remote

import (
	"context"
	"crypto/tls"
	"encoding/pem"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

// Inspect connects to a remote node returning its public certs and metadata.
// A node with policy on refuses its metadata to an anonymous caller such as Inspect, so for one of those Inspect
// returns only the certs.
//
// Inspect verifies nothing about the node, which is why it sends no setup code: pass the certs it returns to Enroll
// along with the code, and the code only goes over a connection verified against them.
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
	defer conn.Close()

	out := &hubpb.HubNodeInspection{}
	// do the API call first, to force the connection to be established (or fail)
	client := metadatapb.NewMetadataApiClient(conn)
	md, err := client.GetMetadata(ctx, &metadatapb.GetMetadataRequest{})
	// A refusal still means the TLS handshake completed, so the certs are there to return.
	if err != nil && status.Code(err) != codes.Unauthenticated {
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
