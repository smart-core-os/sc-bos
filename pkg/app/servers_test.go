package app

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

func TestServeHTTPS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certSource := pki.SelfSignedSource(privateKey)

	gotRequest := make(chan struct{})
	finishHandler := make(chan struct{})
	defer close(finishHandler) // so the handler doesn't run forever
	server := &http.Server{
		Addr:      "localhost:20427",
		TLSConfig: pki.TLSServerConfig(certSource),
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			close(gotRequest)
			<-finishHandler
			writer.WriteHeader(200)
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() {
		err := ServeHTTPS(ctx, server, time.Second, zap.NewNop())
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		serveErr <- err
	}()

	// N.B.: in some cases, the client issues a GET request before the server is listening on the port.
	// without explicit TLS timeout configurations, the TLS handshake hangs forever
	time.Sleep(10 * time.Millisecond)

	go func() {
		client := &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}},
		}
		_, err := client.Get(fmt.Sprintf("https://%s/", server.Addr))
		// the request will never complete as we kill the server
		if err == nil {
			t.Error("expected GET to fail!")
		}
	}()

	// once the handler is blocked, kill the server
	<-gotRequest
	cancel()
	// after half a second, it should still be waiting
	time.Sleep(500 * time.Millisecond)
	select {
	case err := <-serveErr:
		log.Fatalf("server finished early: %s", err.Error())
	default:
	}
	// wait further, and it should have been force-stopped
	time.Sleep(time.Second)
	select {
	case err := <-serveErr:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected stop error: %v", err)
		}
	default:
		t.Error("server didn't stop")
	}
}

func TestServeGRPC(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certSource := pki.SelfSignedSource(privateKey)

	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(pki.TLSServerConfig(certSource))))
	reflection.Register(server) // we don't need reflection for the test, it's just a convenient service to use

	serveCtx, cancelServe := context.WithCancel(context.Background())
	clientCtx, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()

	serveErr := make(chan error, 1)
	addr := "localhost:20428"
	go func() {
		err := ServeGRPC(serveCtx, server, addr, time.Second, zap.NewNop())
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		serveErr <- err
	}()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})),
	)
	if err != nil {
		t.Fatal(err)
	}
	// open a stream connection to the server so it can't stop gracefully
	client := grpc_reflection_v1alpha.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(clientCtx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()

	// tell the server to stop
	cancelServe()
	time.Sleep(500 * time.Millisecond)
	// the connection should still be open at this point, as the server can't stop gracefully and less than 1s has elapsed
	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{})
	if err != nil {
		t.Errorf("unexpected client stream error (1): %v", err)
	}
	time.Sleep(time.Second)
	// connection should have been forced closed now
	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{})
	if err == nil {
		t.Errorf("unexpected client stream error (2): %v", err)
	}
	select {
	case err := <-serveErr:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("unexpected serve error: %v", err)
		}
	default:
		t.Error("Server should have stopped by now")
	}

}
