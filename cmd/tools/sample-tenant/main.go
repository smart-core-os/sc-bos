// Command sample-tenant provides an example tenant application, including auth.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	"github.com/smart-core-os/sc-bos/pkg/proto/lightpb"
)

var (
	flagInsecure     bool
	flagCACert       string
	flagTokenURL     string
	flagGRPCAddr     string
	flagClientId     string
	flagClientSecret string
)

func init() {
	flag.BoolVar(&flagInsecure, "insecure", false, "don't verify TLS certificates")
	flag.StringVar(&flagCACert, "ca", "", "path to Root CA certificate(s), PEM format X.509")
	flag.StringVar(&flagTokenURL, "token-url", "https://localhost:8443/oauth2/token", "URL of OAuth2 token endpoint")
	flag.StringVar(&flagGRPCAddr, "addr", "localhost:23557", "host:port of gRPC server to call")
	flag.StringVar(&flagClientId, "client-id", "", "OAuth2 client ID")
	flag.StringVar(&flagClientSecret, "client-secret", "", "OAuth2 client secret")
}

func main() {
	flag.Parse()
	tlsConfig := &tls.Config{}
	if flagInsecure {
		tlsConfig.InsecureSkipVerify = true
	} else if flagCACert != "" {
		pem, err := os.ReadFile(flagCACert)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "ERROR: read %q: %s", flagCACert, err.Error())
			os.Exit(1)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig.RootCAs = pool
	}

	ccConfig := &clientcredentials.Config{
		TokenURL:     flagTokenURL,
		ClientID:     flagClientId,
		ClientSecret: flagClientSecret,
	}
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	name := "light-1"
	conn, err := grpc.NewClient(flagGRPCAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ccConfig.TokenSource(ctx)}),
	)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: can't connect: %s\n", err.Error())
		os.Exit(1)
	}
	client := lightpb.NewLightApiClient(conn)

	res, err := client.GetBrightness(ctx, &lightpb.GetBrightnessRequest{Name: name})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: call GetBrightness error: %s\n", err.Error())
	} else {
		fmt.Printf("Get response: %v\n", res)
	}

	res, err = client.UpdateBrightness(ctx, &lightpb.UpdateBrightnessRequest{Name: name, Brightness: &lightpb.Brightness{LevelPercent: 75}})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: call UpdateBrightness error: %s\n", err.Error())
	} else {
		fmt.Printf("Update response: %v\n", res)
	}
}
