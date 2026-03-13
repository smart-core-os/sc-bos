package masks

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/smart-core-os/sc-api/go/traits"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"testing"
	"time"
)

// Tests merging of messages using a field mask.
// Scalar fields should be updated only if they appear in the mask
// Repeated fields are appended if they appear in the mask.
//
// Regression Test: FieldUpdater.Merge used to panic with repeated fields
func TestFieldUpdater_Merge(t *testing.T) {
	dst := &traits.ElectricMode{
		Id:          "foo",
		Title:       "old title",
		Description: "old description",
		Segments: []*traits.ElectricMode_Segment{
			{Length: durationpb.New(30 * time.Second), Magnitude: 0.5},
		},
	}

	src := &traits.ElectricMode{
		Id:          "foo",
		Title:       "new title",
		Description: "new description",
		Segments: []*traits.ElectricMode_Segment{
			{Length: durationpb.New(time.Minute), Magnitude: 1},
			{Length: durationpb.New(2 * time.Minute), Magnitude: 2},
		},
	}

	mask, err := fieldmaskpb.New(dst, "title", "segments")
	if err != nil {
		t.Fatal(err)
	}

	expect := &traits.ElectricMode{
		Id:          "foo",
		Title:       "new title",
		Description: "old description",
		Segments: []*traits.ElectricMode_Segment{
			{Length: durationpb.New(30 * time.Second), Magnitude: 0.5},
			{Length: durationpb.New(time.Minute), Magnitude: 1},
			{Length: durationpb.New(2 * time.Minute), Magnitude: 2},
		},
	}

	updater := NewFieldUpdater(WithUpdateMask(mask))
	updater.Merge(dst, src)

	diff := cmp.Diff(dst, expect, cmpopts.EquateEmpty(), protobufEquality)
	if diff != "" {
		t.Error(diff)
	}
}

var protobufEquality = cmp.Comparer(func(x, y proto.Message) bool {
	return proto.Equal(x, y)
})
