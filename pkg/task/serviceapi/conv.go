package serviceapi

import (
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func recordToProto(r *service.Record) *servicespb.Service {
	return stateToProto(r.Id, r.Kind, r.Service.State())
}

func stateToProto(id, kind string, state service.State) *servicespb.Service {
	return &servicespb.Service{
		Id:   id,
		Type: kind,

		Active:           state.Active,
		LastActiveTime:   timeToTimestamp(state.LastActiveTime),
		LastInactiveTime: timeToTimestamp(state.LastInactiveTime),

		Loading:              state.Loading,
		LastLoadingStartTime: timeToTimestamp(state.LastLoadingStartTime),
		LastLoadingEndTime:   timeToTimestamp(state.LastLoadingEndTime),

		ConfigRaw:      string(state.Config),
		LastConfigTime: timeToTimestamp(state.LastConfigTime),

		Error:         errorToString(state.Err),
		LastErrorTime: timeToTimestamp(state.LastErrTime),

		FailedAttempts:  int32(state.FailedAttempts),
		NextAttemptTime: timeToTimestamp(state.NextAttemptTime),
	}
}

func protoToState(s *servicespb.Service) (id, kind string, state service.State) {
	id = s.Id
	kind = s.Type
	state = service.State{
		Active:               s.Active,
		Config:               []byte(s.ConfigRaw),
		Loading:              s.Loading,
		Err:                  stringToError(s.Error),
		LastInactiveTime:     timestampToTime(s.LastInactiveTime),
		LastActiveTime:       timestampToTime(s.LastActiveTime),
		LastErrTime:          timestampToTime(s.LastErrorTime),
		LastConfigTime:       timestampToTime(s.LastConfigTime),
		LastLoadingStartTime: timestampToTime(s.LastLoadingStartTime),
		LastLoadingEndTime:   timestampToTime(s.LastLoadingEndTime),
	}
	return
}

func stateToPullServiceResponse(name, id, kind string, state service.State) *servicespb.PullServiceResponse {
	return &servicespb.PullServiceResponse{Changes: []*servicespb.PullServiceResponse_Change{
		{
			Name:       name,
			ChangeTime: timestamppb.Now(),
			Service:    stateToProto(id, kind, state),
		},
	}}
}

func timeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func timestampToTime(t *timestamppb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.AsTime()
}

func errorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func stringToError(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}
