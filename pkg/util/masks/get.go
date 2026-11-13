package masks

import (
	"github.com/mennanov/fmutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
//
// A path may continue past a map field to name map keys: "labels.a" keeps only the "a" entry of the labels map.
// Map keys are compared to the path segment as strings, and a path may continue past a key into a message value.
// A path that continues past a scalar, a repeated scalar or a map's scalar value has nothing more to select,
// so the extra segments are ignored and the field (or map entry) is kept whole.
// Paths through a repeated message field apply to every element.
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
	filterMsg(msg.ProtoReflect(), fmutils.NestedMaskFromPaths(r.fields.GetPaths()))
}

// FilterClone is like Filter but clones and returns a new msg instead of modifying the original.
func (r *ResponseFilter) FilterClone(msg proto.Message) proto.Message {
	if r.fields == nil {
		return msg
	}
	if msg == nil {
		return msg
	}
	clone := proto.Clone(msg)
	r.Filter(clone)
	return clone
}

// filterMsg keeps the fields of msg named in mask and clears the rest, as described on ResponseFilter.Filter.
// It replaces fmutils.NestedMask.Filter, which panics when a path continues into a map or past a repeated scalar.
// Read masks come straight from callers and aren't validated before filtering, so this must accept any mask.
func filterMsg(msg protoreflect.Message, mask fmutils.NestedMask) {
	if len(mask) == 0 {
		return
	}
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		sub, ok := mask[string(fd.Name())]
		switch {
		case !ok:
			msg.Clear(fd)
		case len(sub) == 0:
			// the whole field was asked for
		case fd.IsMap():
			filterMap(v.Map(), fd.MapValue(), sub)
		case fd.IsList():
			if fd.Message() != nil {
				list := v.List()
				for i := range list.Len() {
					filterMsg(list.Get(i).Message(), sub)
				}
			}
		case fd.Message() != nil:
			filterMsg(v.Message(), sub)
		}
		return true
	})
}

// filterMap keeps the entries of m whose keys are named in mask, filtering message values by any deeper mask.
func filterMap(m protoreflect.Map, valueFd protoreflect.FieldDescriptor, mask fmutils.NestedMask) {
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		sub, ok := mask[k.String()]
		switch {
		case !ok:
			m.Clear(k)
		case len(sub) > 0 && valueFd.Message() != nil:
			filterMsg(v.Message(), sub)
		}
		return true
	})
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
