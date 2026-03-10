package server

import "google.golang.org/grpc"

type GrpcApi interface {
	Register(server grpc.ServiceRegistrar)
}

type collection []GrpcApi

func (c collection) Register(server grpc.ServiceRegistrar) {
	for _, api := range c {
		api.Register(server)
	}
}

// Collection combines multiple GrpcApi instances into a single GrpcApi that all get registered at the same time.
func Collection(apis ...GrpcApi) GrpcApi {
	return collection(apis)
}
