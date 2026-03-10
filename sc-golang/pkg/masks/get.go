package masks

import (
	"github.com/mennanov/fmutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ResponseFilter provides utilities for applying FieldMasks to responses.
type ResponseFilter struct {
	fields *fieldmaskpb.FieldMask
}

// NewResponseFilter creates a new ResponseFilter with the given options applied.
func NewResponseFilter(opts ...ResponseFilterOption) *ResponseFilter {
	res := &ResponseFilter{}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func (r *ResponseFilter) Validate(msg proto.Message) error {
	if r.fields != nil {
		if !r.fields.IsValid(msg) {
			return status.Errorf(codes.InvalidArgument, "%v mentions unknown fields", r.fields)
		}
	}
	return nil
}

// Filter resets all fields in msg that are not requested via the WithFieldMask option.
// If no field mask is configured, will not modify the msg.
// This changes the original message.
func (r *ResponseFilter) Filter(msg proto.Message) {
	if r.fields == nil {
		return
	}
	if msg == nil {
		return
	}
	if len(r.fields.GetPaths()) == 0 {
		proto.Reset(msg)
		return
	}
	fmutils.Filter(msg, r.fields.GetPaths())
}

// FilterClone is like Filter but clones and returns a new msg instead of modifying the original.
func (r *ResponseFilter) FilterClone(msg proto.Message) proto.Message {
	if r.fields == nil {
		return msg
	}
	if msg == nil {
		return msg
	}
	if len(r.fields.GetPaths()) == 0 {
		clone := proto.Clone(msg)
		proto.Reset(clone)
		return clone
	}
	clone := proto.Clone(msg)
	fmutils.Filter(clone, r.fields.GetPaths())
	return clone
}

type ResponseFilterOption func(*ResponseFilter)

var emptyResponseFilterOption ResponseFilterOption = func(_ *ResponseFilter) {
}

func WithFieldMask(fm *fieldmaskpb.FieldMask) ResponseFilterOption {
	if fm == nil {
		return emptyResponseFilterOption
	}
	return func(filter *ResponseFilter) {
		filter.fields = fm
	}
}

func WithFieldMaskPaths(paths ...string) ResponseFilterOption {
	return WithFieldMask(&fieldmaskpb.FieldMask{Paths: paths})
}
