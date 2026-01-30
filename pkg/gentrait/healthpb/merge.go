package healthpb

import (
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// MergeCheck merges src into dst, which must share the same id.
// This is similar to proto.Merge, but some repeated fields are replaced instead of appended to.
func MergeCheck(merge func(dst, src proto.Message), dst, src *healthpb.HealthCheck) {
	if src == nil || dst == nil {
		return
	}
	if src.Id != dst.Id {
		return
	}
	var post []func()
	if v := dst.GetBounds().GetAbnormalValues(); v != nil {
		ov := v.Values
		v.Values = nil
		post = append(post, func() {
			v := dst.GetBounds().GetAbnormalValues()
			if v == nil || len(v.Values) > 0 {
				return // src set another of the one of fields, or updated this one
			}
			v.Values = ov
		})
	}
	if v := dst.GetBounds().GetNormalValues(); v != nil {
		ov := v.Values
		v.Values = nil
		post = append(post, func() {
			v := dst.GetBounds().GetNormalValues()
			if v == nil || len(v.Values) > 0 {
				return // src set another of the one of fields, or updated this one
			}
			v.Values = ov
		})
	}
	if v := dst.GetComplianceImpacts(); len(v) > 0 {
		ov := v
		dst.ComplianceImpacts = nil
		post = append(post, func() {
			if len(dst.GetComplianceImpacts()) > 0 {
				return // src updated the field
			}
			dst.ComplianceImpacts = ov
		})
	}

	// manual merging of timestamps
	dst.CreateTime, src.CreateTime = earliestTimestamp(dst.CreateTime, src.CreateTime), nil

	merge(dst, src)

	for _, f := range post {
		f()
	}
}

func earliestTimestamp(dst, src *timestamppb.Timestamp) *timestamppb.Timestamp {
	switch {
	case src == nil:
		return dst
	case dst == nil:
		return src
	case src.AsTime().Before(dst.AsTime()):
		return src
	default:
		return dst
	}
}

// MergeChecks adds src checks into dst, merging when ids match, returning the union.
// The dst checks must be sorted by ID in ascending order.
// The returned checks will be sorted by ID in ascending order as well.
func MergeChecks(merge func(dst, src proto.Message), dst []*healthpb.HealthCheck, src ...*healthpb.HealthCheck) []*healthpb.HealthCheck {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		sortChecks(src) // src checks can be in any order, the result should be sorted
		return src
	}

	for _, srcCheck := range src {
		dstIndex, found := findCheck(srcCheck.Id, dst)
		if found {
			// merge existing check
			MergeCheck(merge, dst[dstIndex], srcCheck)
			continue
		}
		// add new check, which we do later when we know how many we need to add
		dst = slices.Insert(dst, dstIndex, srcCheck)
	}

	return dst
}

// RemoveCheck removes the check with the given id from dst, returning the modified slice.
// If no such check exists, dst is returned unmodified.
func RemoveCheck(dst []*healthpb.HealthCheck, id string) []*healthpb.HealthCheck {
	index, found := findCheck(id, dst)
	if !found {
		return dst
	}
	return slices.Delete(dst, index, index+1)
}

// sortChecks sorts the checks by their ID in ascending order.
func sortChecks(checks []*healthpb.HealthCheck) {
	slices.SortFunc(checks, func(a, b *healthpb.HealthCheck) int {
		return strings.Compare(a.Id, b.Id)
	})
}

func findCheck(id string, checks []*healthpb.HealthCheck) (int, bool) {
	return slices.BinarySearchFunc(checks, id, func(check *healthpb.HealthCheck, id string) int {
		return strings.Compare(check.Id, id)
	})
}
