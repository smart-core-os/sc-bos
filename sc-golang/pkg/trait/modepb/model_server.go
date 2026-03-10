package modepb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// ModelServer adapts a Model as a traits.ModeApiServer.
// Relative mode value updates will wrap around in both directions.
type ModelServer struct {
	traits.UnimplementedModeApiServer
	model *Model
}

func NewModelServer(model *Model) *ModelServer {
	return &ModelServer{model: model}
}

func (m *ModelServer) Register(server grpc.ServiceRegistrar) {
	traits.RegisterModeApiServer(server, m)
}

func (m *ModelServer) Unwrap() any {
	return m.model
}

func (m *ModelServer) GetModeValues(_ context.Context, request *traits.GetModeValuesRequest) (*traits.ModeValues, error) {
	return m.model.ModeValues(resource.WithReadMask(request.ReadMask)), nil
}

func (m *ModelServer) UpdateModeValues(_ context.Context, request *traits.UpdateModeValuesRequest) (*traits.ModeValues, error) {
	opts := []resource.WriteOption{
		resource.WithUpdateMask(request.UpdateMask),
	}
	if len(request.GetRelative().GetValues()) > 0 {
		opts = append(opts, resource.InterceptBefore(m.relativeAdjustment(request.Relative.Values)))
	}

	values := request.ModeValues
	if values == nil {
		// in case all updates are relative, we can't set nothing
		values = &traits.ModeValues{}
	}
	return m.model.UpdateModeValues(values, opts...)
}

// relativeAdjustment returns an interceptor function that applies the given relative updates to the current mode values.
// This is liberal and will choose the first available mode if, for example, we can't work out the current value index.
func (m *ModelServer) relativeAdjustment(relative map[string]int32) resource.UpdateInterceptor {
	return func(old, new proto.Message) {
		oldVal := old.(*traits.ModeValues)
		newVal := new.(*traits.ModeValues)
		if newVal.Values == nil {
			newVal.Values = make(map[string]string)
		}

	adjustments:
		for modeName, adjustment := range relative {
			oldValue, ok := oldVal.Values[modeName]
			values := m.model.AvailableValues(modeName)
			if len(values) == 0 {
				continue
			}

			// if there's no current value, just choose the first one
			if !ok {
				newVal.Values[modeName] = values[0].Name
				continue
			}

			// find the value index in our supported values, and adjust the value to the new index based on adjustment
			for i, value := range values {
				if value.Name == oldValue {
					newI := (int32(i) + adjustment) % int32(len(values))
					if newI < 0 {
						newI = int32(len(values)) + newI
					}
					newVal.Values[modeName] = values[newI].Name
					continue adjustments
				}
			}

			// apparently the current value isn't one of the supported values, let's correct that
			newVal.Values[modeName] = values[0].Name
		}
	}
}

func (m *ModelServer) PullModeValues(request *traits.PullModeValuesRequest, server traits.ModeApi_PullModeValuesServer) error {
	for change := range m.model.PullModeValues(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		err := server.Send(&traits.PullModeValuesResponse{Changes: []*traits.PullModeValuesResponse_Change{
			{
				Name:       request.Name,
				ChangeTime: timestamppb.New(change.ChangeTime),
				ModeValues: change.Value,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
