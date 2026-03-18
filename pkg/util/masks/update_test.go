package masks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-bos/internal/testproto"
)

// Tests merging of messages using a field mask.
// Scalar fields should be updated only if they appear in the mask
// Repeated fields are appended if they appear in the mask.
//
// Regression Test: FieldUpdater.Merge used to panic with repeated fields
func TestFieldUpdater_Merge(t *testing.T) {
	dst := &testproto.TestAllTypes{
		DefaultInt32:   1,
		DefaultString:  "old title",
		DefaultFloat:   1.0,
		RepeatedString: []string{"a"},
	}

	src := &testproto.TestAllTypes{
		DefaultInt32:   1,
		DefaultString:  "new title",
		DefaultFloat:   2.0,
		RepeatedString: []string{"b", "c"},
	}

	mask, err := fieldmaskpb.New(dst, "default_string", "repeated_string")
	if err != nil {
		t.Fatal(err)
	}

	expect := &testproto.TestAllTypes{
		DefaultInt32:   1,
		DefaultString:  "new title",
		DefaultFloat:   1.0,
		RepeatedString: []string{"a", "b", "c"},
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
