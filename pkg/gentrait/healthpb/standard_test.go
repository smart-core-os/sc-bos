package healthpb

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFindByDisplayName(t *testing.T) {
	if diff := cmp.Diff(BS5266_1_2016, FindStandardByDisplayName(BS5266_1_2016.GetDisplayName()), protocmp.Transform()); diff != "" {
		t.Errorf("FindStandardByDisplayName() mismatch (-want +got):\n%s", diff)
	}
	if got := FindStandardByDisplayName("non-existent"); got != nil {
		t.Errorf("FindStandardByDisplayName() = %v, want nil", got)
	}
}
