package statuspb

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

type ModelServer struct {
	statuspb.UnimplementedStatusApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{
		model: model,
	}
}

func (s *ModelServer) GetCurrentStatus(_ context.Context, req *statuspb.GetCurrentStatusRequest) (*statuspb.StatusLog, error) {
	return s.model.GetCurrentStatus(req.ReadMask)
}

func (s *ModelServer) PullCurrentStatus(request *statuspb.PullCurrentStatusRequest, server statuspb.StatusApi_PullCurrentStatusServer) error {
	ctx := server.Context()
	changes := s.model.PullCurrentStatus(ctx, request.ReadMask, request.UpdatesOnly)
	for change := range changes {
		err := server.Send(&statuspb.PullCurrentStatusResponse{Changes: []*statuspb.PullCurrentStatusResponse_Change{{
			Name:          request.Name,
			CurrentStatus: change.StatusLog,
			ChangeTime:    timestamppb.New(change.ChangeTime),
		}}})
		if err != nil {
			return err
		}
	}
	return nil
}
