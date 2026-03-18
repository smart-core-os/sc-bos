package wastepb

import (
	"context"
	"strconv"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/wastepb"
)

type ModelServer struct {
	wastepb.UnimplementedWasteApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	wastepb.RegisterWasteApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) ListWasteRecords(ctx context.Context, req *wastepb.ListWasteRecordsRequest) (*wastepb.ListWasteRecordsResponse, error) {
	// page token is just the index of where we left off (if any)
	// this works with the current basic implementation because we only support a list of all events without filtering/sorting
	// and the events are stored in ascending chronological order. If this either of these things change, this will need to be rethought
	pageToken := req.GetPageToken()
	startIndex := m.model.GetWasteRecordCount()
	if pageToken != "" {
		_, err := strconv.Atoi(req.GetPageToken())
		if err != nil {
			return nil, err
		}
		startIndex, _ = strconv.Atoi(pageToken)
	}

	count := req.PageSize
	if count == 0 {
		count = 50
	} else if count > 1000 {
		count = 1000
	}

	resp := &wastepb.ListWasteRecordsResponse{}
	resp.WasteRecords = m.model.ListWasteRecords(startIndex, int(count))

	if int(count) == len(resp.WasteRecords) {
		npt := startIndex - int(count)
		if npt > 0 {
			resp.NextPageToken = strconv.Itoa(startIndex - int(count))
		}
	}
	resp.TotalSize = int32(m.model.GetWasteRecordCount())
	return resp, nil
}

// PullWasteRecords returns a channel of WasteRecords
// If updatesOnly is false, only the previous 50 events will be sent before any new events
// For historical events use ListWasteRecords
func (m *ModelServer) PullWasteRecords(request *wastepb.PullWasteRecordsRequest, server wastepb.WasteApi_PullWasteRecordsServer) error {
	return m.model.pullWasteRecordsWrapper(request, server)
}
