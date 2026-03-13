package emergencylightpb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

type ModelServer struct {
	UnimplementedEmergencyLightApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	RegisterEmergencyLightApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetTestResultSet(context.Context, *GetTestResultSetRequest) (*TestResultSet, error) {
	return m.model.GetTestResultSet(), nil
}

func (m *ModelServer) StartFunctionTest(context.Context, *StartEmergencyTestRequest) (*StartEmergencyTestResponse, error) {
	m.model.RunFunctionTest()
	return &StartEmergencyTestResponse{}, nil
}

func (m *ModelServer) StartDurationTest(context.Context, *StartEmergencyTestRequest) (*StartEmergencyTestResponse, error) {
	m.model.RunDurationTest()
	return &StartEmergencyTestResponse{}, nil
}

func (m *ModelServer) StopEmergencyTest(context.Context, *StopEmergencyTestsRequest) (*StopEmergencyTestsResponse, error) {
	// No-op for this model, as tests are run immediately
	return &StopEmergencyTestsResponse{}, nil
}

func (m *ModelServer) PullTestResultSets(request *PullTestResultRequest, server grpc.ServerStreamingServer[PullTestResultsResponse]) error {
	for change := range m.model.PullTestResults(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &PullTestResultsResponse{Changes: []*PullTestResultsResponse_Change{{
			Name:       request.Name,
			ChangeTime: timestamppb.New(change.ChangeTime),
			TestResult: change.Value,
		}}}
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
