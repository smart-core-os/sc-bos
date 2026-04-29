package channelpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type ModelServer struct {
	UnimplementedChannelApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (s *ModelServer) Unwrap() any {
	return s.model
}

func (s *ModelServer) Register(server grpc.ServiceRegistrar) {
	RegisterChannelApiServer(server, s)
}

func (s *ModelServer) GetChosenChannel(_ context.Context, req *GetChosenChannelRequest) (*Channel, error) {
	return s.model.GetChosenChannel(resource.WithReadMask(req.ReadMask))
}

func (s *ModelServer) ChooseChannel(_ context.Context, req *ChooseChannelRequest) (*Channel, error) {
	return s.model.UpdateChosenChannel(req.Channel)
}

func (s *ModelServer) AdjustChannel(_ context.Context, _ *AdjustChannelRequest) (*Channel, error) {
	return nil, status.Errorf(codes.Unimplemented, "AdjustChannel not implemented")
}

func (s *ModelServer) ReturnChannel(_ context.Context, _ *ReturnChannelRequest) (*Channel, error) {
	return nil, status.Errorf(codes.Unimplemented, "ReturnChannel not implemented")
}

func (s *ModelServer) PullChosenChannel(request *PullChosenChannelRequest, server ChannelApi_PullChosenChannelServer) error {
	for update := range s.model.PullChosenChannel(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullChosenChannelResponse{Changes: []*PullChosenChannelResponse_Change{{
			Name:          request.Name,
			ChangeTime:    timestamppb.New(update.ChangeTime),
			ChosenChannel: update.Value,
		}}})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}
