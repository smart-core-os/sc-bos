package merge

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

// ResourceSupport merges all ResourceSupport values into a single ResourceSupport.
// If any item is readable, the result will be readable, similarly for writable and observable.
// If any items disagree on PullSupport, the result will be PULL_SUPPORT_UNSPECIFIED.
// PullPoll is the maximum of all PullPoll values.
func ResourceSupport[E any](all []E, fn func(E) *types.ResourceSupport) *types.ResourceSupport {
	var dst *types.ResourceSupport
	only := true
	for _, item := range all {
		src := fn(item)
		switch {
		case src == nil:
			continue
		case dst == nil:
			dst = src
			continue
		case only:
			only = false
			dst = proto.Clone(dst).(*types.ResourceSupport)
		}

		resourceSupport(dst, src)
	}
	return dst
}

func resourceSupport(dst, src *types.ResourceSupport) {
	dst.Readable = dst.Readable || src.Readable
	dst.Writable = dst.Writable || src.Writable
	dst.Observable = dst.Observable || src.Observable
	dst.PullSupport = protoEnum(dst.PullSupport, src.PullSupport)
	dst.PullPoll = MaxDuration([]*types.ResourceSupport{dst, src}, func(s *types.ResourceSupport) *durationpb.Duration {
		return s.GetPullPoll()
	})
}

func Int32Attributes[E any](all []E, fn func(E) *types.Int32Attributes) *types.Int32Attributes {
	var dst *types.Int32Attributes
	only := true
	for _, item := range all {
		src := fn(item)
		switch {
		case src == nil:
			continue
		case dst == nil:
			dst = src
			continue
		case only:
			only = false
			dst = proto.Clone(dst).(*types.Int32Attributes)
		}

		int32Attributes(dst, src)
	}
	return dst
}

func int32Attributes(dst, src *types.Int32Attributes) {
	dst.Bounds = Int32Bounds([]*types.Int32Attributes{dst, src}, func(s *types.Int32Attributes) *types.Int32Bounds {
		return s.GetBounds()
	})
	if dst.Step == 0 || (src.Step != 0 && src.Step < dst.Step) {
		dst.Step = src.Step
	}
	dst.SupportsDelta = dst.SupportsDelta || src.SupportsDelta
	dst.RampSupport = protoEnum(dst.RampSupport, src.RampSupport)
	if dst.DefaultCapping == nil {
		dst.DefaultCapping = src.DefaultCapping
	} else if src.DefaultCapping != nil {
		dst.DefaultCapping.Min = protoEnum(dst.DefaultCapping.Min, src.DefaultCapping.Min)
		dst.DefaultCapping.Max = protoEnum(dst.DefaultCapping.Max, src.DefaultCapping.Max)
		dst.DefaultCapping.Step = protoEnum(dst.DefaultCapping.Step, src.DefaultCapping.Step)
	}
}

func protoEnum[T ~int32](dst, src T) T {
	if src == 0 {
		return dst
	}
	if src != dst {
		return 0 // UNSUPPORTED
	}
	return dst
}
