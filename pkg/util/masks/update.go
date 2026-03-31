package masks

import (
	"github.com/mennanov/fmutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type FieldUpdater struct {
	writableFields      *fieldmaskpb.FieldMask
	updateMask          *fieldmaskpb.FieldMask
	updateMaskFieldName string
	resetMask           *fieldmaskpb.FieldMask

	intersectionMask *fieldmaskpb.FieldMask
}

func NewFieldUpdater(opts ...FieldUpdaterOption) *FieldUpdater {
	f := &FieldUpdater{}
	for _, opt := range DefaultFieldUpdateOptions {
		opt(f)
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *FieldUpdater) Validate(m proto.Message) error {
	if f.updateMask != nil {
		// is the update mask valid?
		if !f.updateMask.IsValid(m) {
			return status.Errorf(codes.InvalidArgument, "%v mentions unknown fields", f.updateMaskFieldName)
		}

		// are fields mentioned in the update mask actually writable?
		if f.writableFields != nil {
			common := f.fullMask()
			if len(common.Paths) != len(f.updateMask.Paths) {
				return status.Errorf(codes.InvalidArgument, "%v mentions read-only fields", f.updateMaskFieldName)
			}
		}
	}
	if f.resetMask != nil {
		if !f.resetMask.IsValid(m) {
			return status.Errorf(codes.Internal, "resetMask mentions unknown fields %v", f.resetMask)
		}
	}

	return nil
}

// Merge copies the values in src into dst based on the configured field masks.
func (f *FieldUpdater) Merge(dst, src proto.Message) {
	if f.writableFields != nil && len(f.writableFields.Paths) == 0 {
		return // nothing is writable
	}

	var writableMask fmutils.NestedMask
	if f.writableFields != nil {
		writableMask = fmutils.NestedMaskFromPaths(f.writableFields.Paths)
	}

	// only allow writing writable fields by resetting non-writable fields in src
	writableMask.Filter(src)

	mask := f.updateMask
	if mask == nil {
		// no mask, make dst look like src
		if writableMask == nil {
			// if all fields are writable then reset all fields
			proto.Reset(dst)
		} else {
			// if only some fields are writable then reset only those
			writableMask.Prune(dst)
		}
	} else if len(mask.GetPaths()) == 0 {
		// non-nil mask with no paths => no changes
		return
	}

	nestedMask := fmutils.NestedMaskFromPaths(mask.GetPaths())
	nestedMask.Filter(src)
	proto.Merge(dst, src)

	// if a field mentioned by the mask is nil, we should clear it
	pruneEmpty(dst, src, nestedMask)

	if f.resetMask != nil {
		fmutils.Prune(dst, f.resetMask.Paths)
	}

	return
}

func pruneEmpty(dst, src proto.Message, mask fmutils.NestedMask) {
	dstPr := dst.ProtoReflect()
	srcPr := src.ProtoReflect()
	dstPr.Range(func(d protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldMask, ok := mask[string(d.Name())]
		if !ok {
			return true
		}
		if !srcPr.Has(d) {
			dstPr.Clear(d)
			return true
		}
		if d.Kind() == protoreflect.MessageKind && d.Cardinality() != protoreflect.Repeated {
			pruneEmpty(dstPr.Get(d).Message().Interface(), srcPr.Get(d).Message().Interface(), fieldMask)
		}
		return true
	})
}

func (f *FieldUpdater) fullMask() *fieldmaskpb.FieldMask {
	if f.intersectionMask == nil {
		var nonNilMasks []*fieldmaskpb.FieldMask
		if f.writableFields != nil {
			nonNilMasks = append(nonNilMasks, f.writableFields)
		}
		if f.updateMask != nil {
			nonNilMasks = append(nonNilMasks, f.updateMask)
		}

		switch len(nonNilMasks) {
		case 0:
			return nil
		case 1:
			f.intersectionMask = nonNilMasks[0]
		case 2:
			f.intersectionMask = fieldmaskpb.Intersect(nonNilMasks[0], nonNilMasks[1])
		default:
			f.intersectionMask = fieldmaskpb.Intersect(nonNilMasks[0], nonNilMasks[1], nonNilMasks[2:]...)
		}
	}
	return f.intersectionMask
}

type FieldUpdaterOption func(*FieldUpdater)

var DefaultFieldUpdateOptions = []FieldUpdaterOption{
	WithUpdateMaskFieldName("update_mask"),
}

// emptyFieldUpdaterOption is used when existing options decide they don't have anything to do
var emptyFieldUpdaterOption FieldUpdaterOption = func(_ *FieldUpdater) {
}

// WithWritableFields configures the FieldUpdater to validate that no non-writable fields are being written.
// A nil writableFields means all fields are writable.
func WithWritableFields(writableFields *fieldmaskpb.FieldMask) FieldUpdaterOption {
	if writableFields == nil {
		return emptyFieldUpdaterOption
	}
	return func(updater *FieldUpdater) {
		updater.writableFields = writableFields
	}
}

// WithUpdateMask configures the FieldUpdater to only apply values for the mentioned fields.
// A nil updateMask represents all writable fields.
func WithUpdateMask(updateMask *fieldmaskpb.FieldMask) FieldUpdaterOption {
	if updateMask == nil {
		return emptyFieldUpdaterOption
	}
	return func(updater *FieldUpdater) {
		updater.updateMask = updateMask
	}
}

// WithUpdateMaskFieldName configures the name of the field that holds the update mask.
// Will be used when constructing error messages.
// Defaults to "update_mask".
func WithUpdateMaskFieldName(name string) FieldUpdaterOption {
	return func(updater *FieldUpdater) {
		updater.updateMaskFieldName = name
	}
}

// WithResetMask configures the FieldUpdater to clear the values mentioned in the mask.
func WithResetMask(resetMask *fieldmaskpb.FieldMask) FieldUpdaterOption {
	return func(updater *FieldUpdater) {
		updater.resetMask = resetMask
	}
}

// WithResetPaths is like WithResetMask but accepts paths.
func WithResetPaths(paths ...string) FieldUpdaterOption {
	return WithResetMask(&fieldmaskpb.FieldMask{Paths: paths})
}
