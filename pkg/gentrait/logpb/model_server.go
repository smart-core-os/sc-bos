package logpb

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

const (
	defaultInitialCount = 200
	maxInitialCount     = 1000
)

// ModelServer is a LogApiServer backed by a Model.
type ModelServer struct {
	logpb.UnimplementedLogApiServer

	model *Model

	// GetDownloadLogUrl is called by the GetDownloadLogUrl RPC.
	// If nil the RPC returns UNIMPLEMENTED.
	GetDownloadLogUrlFunc func(ctx context.Context, req *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error)
}

// NewModelServer returns a ModelServer backed by model.
func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

// PullLogMessages streams recent and then live log messages.
func (s *ModelServer) PullLogMessages(request *logpb.PullLogMessagesRequest, server logpb.LogApi_PullLogMessagesServer) error {
	if request.LoggerFilter != "" {
		return status.Error(codes.Unimplemented, "logger_filter is not yet supported")
	}

	n := int(request.InitialCount)
	switch {
	case request.UpdatesOnly:
		n = 0
	case n <= 0:
		n = defaultInitialCount
	case n > maxInitialCount:
		n = maxInitialCount
	}

	// Replay ring buffer tail before subscribing so we don't miss entries
	// that arrive between the tail snapshot and the Subscribe call.
	ch, cancel := s.model.Subscribe()
	defer cancel()

	initial := s.model.TailMessages(n)
	if len(initial) > 0 {
		filtered := filterMessages(initial, request.MinLevel)
		if len(filtered) > 0 {
			if err := server.Send(&logpb.PullLogMessagesResponse{Messages: filtered}); err != nil {
				return err
			}
		}
	}

	ctx := server.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch, ok := <-ch:
			if !ok {
				return nil
			}
			filtered := filterMessages(batch, request.MinLevel)
			if len(filtered) == 0 {
				continue
			}
			if err := server.Send(&logpb.PullLogMessagesResponse{Messages: filtered}); err != nil {
				return err
			}
		}
	}
}

func filterMessages(msgs []*logpb.LogMessage, minLevel logpb.LogLevel_Level) []*logpb.LogMessage {
	if minLevel == logpb.LogLevel_LEVEL_UNSPECIFIED {
		return msgs
	}
	out := msgs[:0:0] // reuse underlying array header but don't share
	for _, m := range msgs {
		if m.Level >= minLevel {
			out = append(out, m)
		}
	}
	return out
}

// GetLogLevel returns the current log level.
func (s *ModelServer) GetLogLevel(_ context.Context, request *logpb.GetLogLevelRequest) (*logpb.LogLevel, error) {
	_ = request.Name
	return s.model.GetLogLevel()
}

// PullLogLevel streams changes to the log level.
func (s *ModelServer) PullLogLevel(request *logpb.PullLogLevelRequest, server logpb.LogApi_PullLogLevelServer) error {
	for change := range s.model.PullLogLevel(server.Context(),
		resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&logpb.PullLogLevelResponse{
			Changes: []*logpb.PullLogLevelResponse_Change{{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				LogLevel:   change.Value,
			}},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

// UpdateLogLevel sets the log level.
func (s *ModelServer) UpdateLogLevel(_ context.Context, request *logpb.UpdateLogLevelRequest) (*logpb.LogLevel, error) {
	return s.model.UpdateLogLevel(request.LogLevel)
}

// GetLogMetadata returns the current log file metadata.
func (s *ModelServer) GetLogMetadata(_ context.Context, request *logpb.GetLogMetadataRequest) (*logpb.LogMetadata, error) {
	_ = request.Name
	return s.model.GetLogMetadata()
}

// PullLogMetadata streams changes to the log file metadata.
func (s *ModelServer) PullLogMetadata(request *logpb.PullLogMetadataRequest, server logpb.LogApi_PullLogMetadataServer) error {
	for change := range s.model.PullLogMetadata(server.Context(),
		resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&logpb.PullLogMetadataResponse{
			Changes: []*logpb.PullLogMetadataResponse_Change{{
				Name:        request.Name,
				ChangeTime:  timestamppb.New(change.ChangeTime),
				LogMetadata: change.Value,
			}},
		})
		if err != nil {
			return err
		}
	}
	return server.Context().Err()
}

// GetDownloadLogUrl returns HTTP URLs for direct log file download.
func (s *ModelServer) GetDownloadLogUrl(ctx context.Context, request *logpb.GetDownloadLogUrlRequest) (*logpb.GetDownloadLogUrlResponse, error) {
	if s.GetDownloadLogUrlFunc == nil {
		return nil, status.Error(codes.Unimplemented, "GetDownloadLogUrl is not configured")
	}
	return s.GetDownloadLogUrlFunc(ctx, request)
}
