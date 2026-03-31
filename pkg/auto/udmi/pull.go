package udmi

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

type udmiExportMessagePuller struct {
	client udmipb.UdmiServiceClient
	name   string
}

func (m *udmiExportMessagePuller) Pull(ctx context.Context, changes chan<- *udmipb.PullExportMessagesResponse) error {
	stream, err := m.client.PullExportMessages(ctx, &udmipb.PullExportMessagesRequest{Name: m.name})
	if err != nil {
		return err
	}

	for {
		change, err := stream.Recv()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- change:
		}
	}
}

func (m *udmiExportMessagePuller) Poll(_ context.Context, _ chan<- *udmipb.PullExportMessagesResponse) error {
	return status.Error(codes.Unimplemented, "not supported")
}

type udmiControlTopicsPuller struct {
	client udmipb.UdmiServiceClient
	name   string
}

func (m *udmiControlTopicsPuller) Pull(ctx context.Context, changes chan<- *udmipb.PullControlTopicsResponse) error {
	stream, err := m.client.PullControlTopics(ctx, &udmipb.PullControlTopicsRequest{Name: m.name})
	if err != nil {
		return err
	}

	for {
		change, err := stream.Recv()
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- change:
		}
	}
}

func (m *udmiControlTopicsPuller) Poll(_ context.Context, _ chan<- *udmipb.PullControlTopicsResponse) error {
	return status.Error(codes.Unimplemented, "not supported")
}
