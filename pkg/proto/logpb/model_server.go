package logpb

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

const (
	defaultInitialCount = 200
	maxInitialCount     = 1000
)

// ModelServer is a LogApiServer backed by a Model.
type ModelServer struct {
	UnimplementedLogApiServer

	model *Model

	// GetDownloadLogUrl is called by the GetDownloadLogUrl RPC.
	// If nil the RPC returns UNIMPLEMENTED.
	GetDownloadLogUrlFunc func(ctx context.Context, req *GetDownloadLogUrlRequest) (*GetDownloadLogUrlResponse, error)
}

// NewModelServer returns a ModelServer backed by model.
func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

// PullLogMessages streams recent and then live log messages.
func (s *ModelServer) PullLogMessages(request *PullLogMessagesRequest, server LogApi_PullLogMessagesServer) error {
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
			if err := server.Send(&PullLogMessagesResponse{
				Changes: []*PullLogMessagesResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					Messages:   filtered,
				}},
			}); err != nil {
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
			if err := server.Send(&PullLogMessagesResponse{
				Changes: []*PullLogMessagesResponse_Change{{
					Name:       request.Name,
					ChangeTime: timestamppb.Now(),
					Messages:   filtered,
				}},
			}); err != nil {
				return err
			}
		}
	}
}

func filterMessages(msgs []*LogMessage, minLevel Level) []*LogMessage {
	if minLevel == Level_LEVEL_UNSPECIFIED {
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
func (s *ModelServer) GetLogLevel(_ context.Context, request *GetLogLevelRequest) (*LogLevel, error) {
	_ = request.Name
	return s.model.GetLogLevel()
}

// PullLogLevel streams changes to the log level.
func (s *ModelServer) PullLogLevel(request *PullLogLevelRequest, server LogApi_PullLogLevelServer) error {
	for change := range s.model.PullLogLevel(server.Context(),
		resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullLogLevelResponse{
			Changes: []*PullLogLevelResponse_Change{{
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
func (s *ModelServer) UpdateLogLevel(_ context.Context, request *UpdateLogLevelRequest) (*LogLevel, error) {
	return s.model.UpdateLogLevel(request.LogLevel)
}

// GetLogMetadata returns the current log file metadata.
func (s *ModelServer) GetLogMetadata(_ context.Context, request *GetLogMetadataRequest) (*LogMetadata, error) {
	_ = request.Name
	return s.model.GetLogMetadata()
}

// PullLogMetadata streams changes to the log file metadata.
func (s *ModelServer) PullLogMetadata(request *PullLogMetadataRequest, server LogApi_PullLogMetadataServer) error {
	for change := range s.model.PullLogMetadata(server.Context(),
		resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&PullLogMetadataResponse{
			Changes: []*PullLogMetadataResponse_Change{{
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
func (s *ModelServer) GetDownloadLogUrl(ctx context.Context, request *GetDownloadLogUrlRequest) (*GetDownloadLogUrlResponse, error) {
	if s.GetDownloadLogUrlFunc == nil {
		return nil, status.Error(codes.Unimplemented, "GetDownloadLogUrl is not configured")
	}
	return s.GetDownloadLogUrlFunc(ctx, request)
}
