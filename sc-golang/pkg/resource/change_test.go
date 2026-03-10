package resource

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func TestCollectionChange_include(t *testing.T) {
	var (
		now    = time.Now()
		on     = &traits.OnOff{State: traits.OnOff_ON}
		off    = &traits.OnOff{State: traits.OnOff_OFF}
		change = func(ov, nv proto.Message, kind types.ChangeType) *CollectionChange {
			return &CollectionChange{
				Id:         "id",
				ChangeTime: now,
				ChangeType: kind,
				OldValue:   ov,
				NewValue:   nv,
			}
		}
		only = func(wantItem proto.Message) FilterFunc {
			return func(_ string, item proto.Message) bool {
				if item == nil {
					return false
				}
				return proto.Equal(item, wantItem)
			}
		}
	)

	tests := []struct {
		name     string
		in, want *CollectionChange
		wantOk   bool
		filter   FilterFunc
	}{
		{
			name:   "no filter, ADD",
			in:     change(nil, on, types.ChangeType_ADD),
			want:   change(nil, on, types.ChangeType_ADD),
			wantOk: true,
		},
		{
			name:   "no filter, UPDATE",
			in:     change(on, off, types.ChangeType_UPDATE),
			want:   change(on, off, types.ChangeType_UPDATE),
			wantOk: true,
		},
		{
			name:   "no filter, REMOVE",
			in:     change(on, nil, types.ChangeType_REMOVE),
			want:   change(on, nil, types.ChangeType_REMOVE),
			wantOk: true,
		},
		{
			name:   "ADD on",
			in:     change(nil, on, types.ChangeType_ADD),
			want:   change(nil, on, types.ChangeType_ADD),
			wantOk: true,
			filter: only(on),
		},
		{
			name:   "ADD off",
			in:     change(nil, off, types.ChangeType_ADD),
			filter: only(on),
		},
		{
			name:   "UPDATE on->on",
			in:     change(on, on, types.ChangeType_UPDATE),
			want:   change(on, on, types.ChangeType_UPDATE),
			wantOk: true,
			filter: only(on),
		},
		{
			name:   "UPDATE on->off",
			in:     change(on, off, types.ChangeType_UPDATE),
			want:   change(on, nil, types.ChangeType_REMOVE),
			wantOk: true,
			filter: only(on),
		},
		{
			name:   "UPDATE off->on",
			in:     change(off, on, types.ChangeType_UPDATE),
			want:   change(nil, on, types.ChangeType_ADD),
			wantOk: true,
			filter: only(on),
		},
		{
			name:   "UPDATE off->off",
			in:     change(off, off, types.ChangeType_UPDATE),
			filter: only(on),
		},
		{
			name:   "REMOVE on",
			in:     change(on, nil, types.ChangeType_REMOVE),
			want:   change(on, nil, types.ChangeType_REMOVE),
			wantOk: true,
			filter: only(on),
		},
		{
			name:   "REMOVE off",
			in:     change(off, nil, types.ChangeType_REMOVE),
			filter: only(on),
		},
		{
			name:   "empty change",
			in:     change(nil, nil, types.ChangeType_UPDATE),
			filter: only(on),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.in.include(tt.filter)
			if ok != tt.wantOk {
				t.Errorf("CollectionChange.include() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			assertCollectionChangeEqual(t, got, tt.want)
		})
	}
}

func assertCollectionChangeEqual(t *testing.T, got, want *CollectionChange) {
	t.Helper()
	if got == want {
		return
	}
	if (got == nil) != (want == nil) {
		t.Errorf("CollectionChange: got %v, want %v", got, want)
		return
	}
	if got.Id != want.Id {
		t.Errorf("CollectionChange.Id: got %v, want %v", got.Id, want.Id)
	}
	if !got.ChangeTime.Equal(want.ChangeTime) {
		t.Errorf("CollectionChange.ChangeTime: got %v, want %v", got.ChangeTime, want.ChangeTime)
	}
	if got.ChangeType != want.ChangeType {
		t.Errorf("CollectionChange.ChangeType: got %v, want %v", got.ChangeType, want.ChangeType)
	}
	if diff := cmp.Diff(want.OldValue, got.OldValue, protocmp.Transform()); diff != "" {
		t.Errorf("CollectionChange.OldValue (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want.NewValue, got.NewValue, protocmp.Transform()); diff != "" {
		t.Errorf("CollectionChange.NewValue (-want +got):\n%s", diff)
	}
	//goland:noinspection GoDeprecation
	if got.SeedValue != want.SeedValue {
		t.Errorf("CollectionChange.SeedValue: got %v, want %v", got.SeedValue, want.SeedValue)
	}
	if got.LastSeedValue != want.LastSeedValue {
		t.Errorf("CollectionChange.LastSeedValue: got %v, want %v", got.LastSeedValue, want.LastSeedValue)
	}
}
