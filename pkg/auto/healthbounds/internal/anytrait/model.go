package anytrait

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-golang/pkg/trait"
)

type Trait struct {
	name      trait.Name
	resources []Resource
}

func (t Trait) Name() trait.Name {
	return t.name
}

func (t Trait) Resources() []Resource {
	return t.resources
}

type getFunc func(ctx context.Context, conn grpc.ClientConnInterface, req GetRequest) (Value, error)
type pullFunc func(ctx context.Context, conn grpc.ClientConnInterface, req PullRequest) (Stream, error)

type Resource struct {
	name string
	desc protoreflect.MessageDescriptor // a description of the resource message type
	get  getFunc
	pull pullFunc
}

func (r Resource) Name() string {
	return r.name
}

// Message returns a new empty instance of the resource message type.
func (r Resource) Message() protoreflect.MessageDescriptor {
	return r.desc
}

func (r Resource) Get(ctx context.Context, conn grpc.ClientConnInterface, req GetRequest) (Value, error) {
	if r.get != nil {
		return r.get(ctx, conn, req)
	}
	return Value{}, status.Errorf(codes.Unimplemented, "get not implemented")
}

func (r Resource) Pull(ctx context.Context, conn grpc.ClientConnInterface, req PullRequest) (Stream, error) {
	if r.pull != nil {
		return r.pull(ctx, conn, req)
	}
	return Stream{}, status.Errorf(codes.Unimplemented, "pull not implemented")
}

type Value struct {
	pb proto.Message
}

func (v Value) Proto() proto.Message {
	return v.pb
}

type ValueChange struct {
	Value      Value
	ChangeTime *timestamppb.Timestamp
}

type (
	ReadRequest struct {
		Name     string
		ReadMask *fieldmaskpb.FieldMask
	}
	GetRequest struct {
		ReadRequest
	}
	PullRequest struct {
		ReadRequest
		UpdatesOnly bool
	}
	PullResponse struct {
		Changes []ValueChange
	}
)

type Stream struct {
	recv func() (PullResponse, error)
}

func (s Stream) Recv() (PullResponse, error) {
	return s.recv()
}
