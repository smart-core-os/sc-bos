// Command proxy-test is an experiment using grpc-proxy to see how applicable it would be for gateways.
package main

import (
	"context"
	"log"
	"net"

	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
)

func main() {
	realLis, err := net.Listen("tcp", ":")
	if err != nil {
		log.Fatalln(err)
	}
	realServer := grpc.NewServer()
	model := onoffpb.NewModel(onoffpb.WithInitialOnOff(&onoffpb.OnOff{State: onoffpb.OnOff_ON}))
	apiImpl := onoffpb.NewModelServer(model)
	onoffpb.RegisterOnOffApiServer(realServer, apiImpl)
	go realServer.Serve(realLis)
	defer realServer.Stop()

	log.Printf("Real server hosted on %v", realLis.Addr())
	realConn, err := grpc.NewClient(realLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln(err)
	}

	handler := proxy.TransparentHandler(func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		log.Printf("proxy(%v, %v)", ctx, fullMethodName)
		return ctx, realConn, nil
	})

	proxyLis, err := net.Listen("tcp", ":")
	if err != nil {
		log.Fatalln(err)
	}
	proxyServer := grpc.NewServer(grpc.UnknownServiceHandler(handler),
		grpc.ChainUnaryInterceptor(logUnaryServerCalls),
		grpc.ChainStreamInterceptor(logStreamServerCalls))
	go proxyServer.Serve(proxyLis)
	defer proxyServer.Stop()

	log.Printf("Proxy server hosted on %v", proxyLis.Addr())
	proxyConn, err := grpc.NewClient(proxyLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := onoffpb.NewOnOffApiClient(proxyConn)
	test, err := client.GetOnOff(context.Background(), &onoffpb.GetOnOffRequest{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Got response %v", test)
}

func logUnaryServerCalls(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	log.Printf("intercept(%v, %v, %v)", ctx, req, info)
	return handler(ctx, req)
}
func logStreamServerCalls(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Printf("intercept( %v, %v, %v)", srv, ss, info)
	return handler(srv, &loggingServerStream{ss})
}

type loggingServerStream struct {
	grpc.ServerStream
}

func (ss *loggingServerStream) SendMsg(m any) error {
	log.Printf("intercept.SendMsg((%T) {%v})", m, m)
	return ss.ServerStream.SendMsg(m)
}

func (ss *loggingServerStream) RecvMsg(m any) error {
	err := ss.ServerStream.RecvMsg(m)
	log.Printf("intercept.RecvMsg((%T) {%v}) %v", m, m, err)
	return err
}
