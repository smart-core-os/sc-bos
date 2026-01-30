package securityevent

import (
	"context"
	"strconv"

	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
)

type ModelServer struct {
	securityeventpb.UnimplementedSecurityEventApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server *grpc.Server) {
	securityeventpb.RegisterSecurityEventApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) ListSecurityEvents(ctx context.Context, req *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {
	// page token is just the index of where we left off (if any)
	// this works with the current basic implementation because we only support a list of all events without filtering/sorting
	// and the events are stored in ascending chronological order. If this either of these things change, this will need to be rethought
	pageToken := req.GetPageToken()
	startIndex := m.model.GetSecurityEventCount()
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

	resp := &securityeventpb.ListSecurityEventsResponse{}
	resp.SecurityEvents = m.model.ListSecurityEvents(startIndex, int(count))

	if int(count) == len(resp.SecurityEvents) {
		npt := startIndex - int(count)
		if npt > 0 {
			resp.NextPageToken = strconv.Itoa(startIndex - int(count))
		}
	}
	resp.TotalSize = int32(m.model.GetSecurityEventCount())
	return resp, nil
}

// PullSecurityEvents returns a channel of security events
// If updatesOnly is false, only the previous 50 events will be sent before any new events
// For historical events use ListSecurityEvents
func (m *ModelServer) PullSecurityEvents(request *securityeventpb.PullSecurityEventsRequest, server securityeventpb.SecurityEventApi_PullSecurityEventsServer) error {
	return m.model.pullSecurityEventsWrapper(request, server)
}
