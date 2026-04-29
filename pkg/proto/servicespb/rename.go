package servicespb

import (
	"context"
)

// RenameApi returns a servicespb.ServicesApiServer that changes the names associated with requests before calling client.
func RenameApi(client ServicesApiClient, namer func(n string) string) ServicesApiServer {
	return &rename{client: client, namer: namer}
}

type rename struct {
	UnimplementedServicesApiServer
	client ServicesApiClient
	namer  func(n string) string
}

func (r *rename) GetService(ctx context.Context, request *GetServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.GetService(ctx, request)
}

func (r *rename) PullService(request *PullServiceRequest, server ServicesApi_PullServiceServer) error {
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

func (r *rename) CreateService(ctx context.Context, request *CreateServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.CreateService(ctx, request)
}

func (r *rename) DeleteService(ctx context.Context, request *DeleteServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.DeleteService(ctx, request)
}

func (r *rename) ListServices(ctx context.Context, request *ListServicesRequest) (*ListServicesResponse, error) {
	request.Name = r.namer(request.Name)
	return r.client.ListServices(ctx, request)
}

func (r *rename) PullServices(request *PullServicesRequest, server ServicesApi_PullServicesServer) error {
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

func (r *rename) StartService(ctx context.Context, request *StartServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.StartService(ctx, request)
}

func (r *rename) ConfigureService(ctx context.Context, request *ConfigureServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.ConfigureService(ctx, request)
}

func (r *rename) StopService(ctx context.Context, request *StopServiceRequest) (*Service, error) {
	request.Name = r.namer(request.Name)
	return r.client.StopService(ctx, request)
}

func (r *rename) GetServiceMetadata(ctx context.Context, request *GetServiceMetadataRequest) (*ServiceMetadata, error) {
	request.Name = r.namer(request.Name)
	return r.client.GetServiceMetadata(ctx, request)
}

func (r *rename) PullServiceMetadata(request *PullServiceMetadataRequest, server ServicesApi_PullServiceMetadataServer) error {
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
