package servicepb

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
)

// RenameApi returns a servicespb.ServicesApiServer that changes the names associated with requests before calling client.
func RenameApi(client servicespb.ServicesApiClient, namer func(n string) string) servicespb.ServicesApiServer {
	return &rename{client: client, namer: namer}
}

type rename struct {
	servicespb.UnimplementedServicesApiServer
	client servicespb.ServicesApiClient
	namer  func(n string) string
}

func (r *rename) GetService(ctx context.Context, request *servicespb.GetServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.GetService(ctx, request)
}

func (r *rename) PullService(request *servicespb.PullServiceRequest, server servicespb.ServicesApi_PullServiceServer) error {
	request.Name = r.namer(request.Name)
	stream, err := r.client.PullService(server.Context(), request)
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			change.Name = request.Name
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
}

func (r *rename) CreateService(ctx context.Context, request *servicespb.CreateServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.CreateService(ctx, request)
}

func (r *rename) DeleteService(ctx context.Context, request *servicespb.DeleteServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.DeleteService(ctx, request)
}

func (r *rename) ListServices(ctx context.Context, request *servicespb.ListServicesRequest) (*servicespb.ListServicesResponse, error) {
	request.Name = r.namer(request.Name)
	return r.client.ListServices(ctx, request)
}

func (r *rename) PullServices(request *servicespb.PullServicesRequest, server servicespb.ServicesApi_PullServicesServer) error {
	request.Name = r.namer(request.Name)
	stream, err := r.client.PullServices(server.Context(), request)
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			change.Name = request.Name
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
}

func (r *rename) StartService(ctx context.Context, request *servicespb.StartServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.StartService(ctx, request)
}

func (r *rename) ConfigureService(ctx context.Context, request *servicespb.ConfigureServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.ConfigureService(ctx, request)
}

func (r *rename) StopService(ctx context.Context, request *servicespb.StopServiceRequest) (*servicespb.Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.StopService(ctx, request)
}

func (r *rename) GetServiceMetadata(ctx context.Context, request *servicespb.GetServiceMetadataRequest) (*servicespb.ServiceMetadata, error) {
	request.Name = r.namer(request.Name)
	return r.client.GetServiceMetadata(ctx, request)
}

func (r *rename) PullServiceMetadata(request *servicespb.PullServiceMetadataRequest, server servicespb.ServicesApi_PullServiceMetadataServer) error {
	request.Name = r.namer(request.Name)
	stream, err := r.client.PullServiceMetadata(server.Context(), request)
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			change.Name = request.Name
		}
		err = server.Send(msg)
		if err != nil {
			return err
		}
	}
}
