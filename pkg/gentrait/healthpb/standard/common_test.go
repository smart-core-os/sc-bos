package standard

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFindByDisplayName(t *testing.T) {
	if diff := cmp.Diff(BS5266_1_2016, FindByDisplayName(BS5266_1_2016.GetDisplayName()), protocmp.Transform()); diff != "" {
		t.Errorf("FindByDisplayName() mismatch (-want +got):\n%s", diff)
	}
	if got := FindByDisplayName("non-existent"); got != nil {
		t.Errorf("FindByDisplayName() = %v, want nil", got)
	}
}
