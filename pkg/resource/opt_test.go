package resource

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/sc-api/go/types"
)

func TestWithUpdatesOnly(t *testing.T) {
	t.Parallel()

	t.Run("Value (default)", func(t *testing.T) {
		v := NewValue(WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_ON}))
		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx)
		var events []*ValueChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Set(&onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]proto.Message, len(events))
		for i, event := range events {
			got[i] = event.Value
		}
		want := []proto.Message{
			&onoffpb.OnOff{State: onoffpb.OnOff_ON},  // initial value
			&onoffpb.OnOff{State: onoffpb.OnOff_OFF}, // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Value (updates only)", func(t *testing.T) {
		v := NewValue(WithInitialValue(&onoffpb.OnOff{State: onoffpb.OnOff_ON}))
		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx, WithUpdatesOnly(true))
		var events []*ValueChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Set(&onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]proto.Message, len(events))
		for i, event := range events {
			got[i] = event.Value
		}
		want := []proto.Message{
			// no initial value
			&onoffpb.OnOff{State: onoffpb.OnOff_OFF}, // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})

	t.Run("Collection (default)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &onoffpb.OnOff{State: onoffpb.OnOff_ON})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx)
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "A", OldValue: nil, NewValue: &onoffpb.OnOff{State: onoffpb.OnOff_ON}, ChangeType: types.ChangeType_ADD},
			{Id: "A", OldValue: &onoffpb.OnOff{State: onoffpb.OnOff_ON}, NewValue: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}, ChangeType: types.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Collection (updates only)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &onoffpb.OnOff{State: onoffpb.OnOff_ON})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)
		c := v.Pull(ctx, WithUpdatesOnly(true))
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "A", OldValue: &onoffpb.OnOff{State: onoffpb.OnOff_ON}, NewValue: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}, ChangeType: types.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithInclude(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		c := NewCollection()
		add(t, c, "A", &onoffpb.OnOff{State: onoffpb.OnOff_ON})
		add(t, c, "B", &onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		add(t, c, "C", &onoffpb.OnOff{State: onoffpb.OnOff_STATE_UNSPECIFIED})

		t.Run("id filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				return id == "B" || id == "C"
			}))
			want := []proto.Message{
				&onoffpb.OnOff{State: onoffpb.OnOff_OFF},
				&onoffpb.OnOff{State: onoffpb.OnOff_STATE_UNSPECIFIED},
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})

		t.Run("body filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				itemVal := item.(*onoffpb.OnOff)
				return itemVal.State != onoffpb.OnOff_STATE_UNSPECIFIED
			}))
			want := []proto.Message{
				&onoffpb.OnOff{State: onoffpb.OnOff_ON},
				&onoffpb.OnOff{State: onoffpb.OnOff_OFF},
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})
	})

	t.Run("Pull", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", &onoffpb.OnOff{State: onoffpb.OnOff_ON})
		add(t, v, "B", &onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		add(t, v, "C", &onoffpb.OnOff{State: onoffpb.OnOff_STATE_UNSPECIFIED})

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)

		// pull only items that are off
		c := v.Pull(ctx, WithInclude(func(_ string, item proto.Message) bool {
			itemVal := item.(*onoffpb.OnOff)
			return itemVal.State == onoffpb.OnOff_OFF
		}))
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", &onoffpb.OnOff{State: onoffpb.OnOff_OFF})
		if err != nil {
			t.Fatal(err)
		}
		_, err = v.Update("B", &onoffpb.OnOff{State: onoffpb.OnOff_ON})
		if err != nil {
			t.Fatal(err)
		}

		time.AfterFunc(10*time.Millisecond, done)
		<-complete // wait for the inner go routine to complete

		got := make([]collectionChange, len(events))
		for i, event := range events {
			got[i] = collectionChange{Id: event.Id, OldValue: event.OldValue, NewValue: event.NewValue, ChangeType: event.ChangeType}
		}
		want := []collectionChange{
			{Id: "B", NewValue: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}, ChangeType: types.ChangeType_ADD},
			{Id: "A", NewValue: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}, ChangeType: types.ChangeType_ADD},
			{Id: "B", OldValue: &onoffpb.OnOff{State: onoffpb.OnOff_OFF}, ChangeType: types.ChangeType_REMOVE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithBackpressure_False(t *testing.T) {
	val := NewValue(WithInitialValue(&onoffpb.OnOff{}))

	t.Run("false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// with backpressure disabled, we can open a Pull, fail to receive, and it doesn't block
		_ = val.Pull(ctx, WithBackpressure(false))
		success := make(chan struct{})
		go func() {
			defer close(success)

			// do a set call, which shouldn't block or error
			_, err := val.Set(&onoffpb.OnOff{State: onoffpb.OnOff_OFF})
			if err != nil {
				t.Error(err)
			}
		}()

		select {
		case <-success:
		case <-time.After(100 * time.Millisecond):
			t.Error("calls blocked")
		}
	})

	t.Run("true", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// with backpressure enabled, we can open a Pull, fail to receive, and it will block calls to Set
		_ = val.Pull(ctx, WithBackpressure(true))
		completed := make(chan struct{})
		go func() {
			defer close(completed)

			// do a set call, which should block
			_, err := val.Set(&onoffpb.OnOff{State: onoffpb.OnOff_OFF})
			if err != nil {
				t.Error(err)
			}
		}()

		select {
		case <-completed:
			t.Error("expected call to Set to block")
		case <-time.After(100 * time.Millisecond):
		}
	})
}

func TestWithIDInterceptor(t *testing.T) {
	c := NewCollection(WithIDInterceptor(strings.ToLower))
	add(t, c, "A", &onoffpb.OnOff{State: onoffpb.OnOff_ON})
	expect := &onoffpb.OnOff{State: onoffpb.OnOff_ON}
	actual, ok := c.Get("a")
	if !ok {
		t.Error("expected to find item with id 'a'")
	}
	if diff := cmp.Diff(expect, actual, protocmp.Transform()); diff != "" {
		t.Errorf("Get('a') returned wrong value (-want,+got)\n%v", diff)
	}
}

func TestWithMerger(t *testing.T) {
	md := func(m proto.Message) *metadatapb.Metadata {
		md, ok := m.(*metadatapb.Metadata)
		if !ok {
			t.Fatalf("expected *traits.Metadata, got %T", m)
		}
		return md
	}

	v := NewValue(WithInitialValue(&metadatapb.Metadata{Name: "initial"}))
	ret, err := v.Set(&metadatapb.Metadata{Name: "write"},
		WithUpdatePaths("name"),
		InterceptBefore(func(old, new proto.Message) {
			if n := md(old).Name; n != "initial" {
				t.Fatalf("expected old value to have Name 'initial', got %q", n)
			}
			if n := md(new).Name; n != "write" {
				t.Fatalf("expected new value to have Name 'write', got %q", n)
			}
			md(new).Name = "before"
		}),
		WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
			if n := md(dst).Name; n != "initial" {
				t.Fatalf("expected dst value to have Name 'initial', got %q", n)
			}
			if n := md(src).Name; n != "before" {
				t.Fatalf("expected src value to have Name 'before', got %q", n)
			}
			md(dst).Name = "merge"

			// test that the mask updates what we expect
			m1 := &metadatapb.Metadata{Name: "name1", Appearance: &metadatapb.Metadata_Appearance{Title: "title1"}}
			m2 := &metadatapb.Metadata{Name: "name2", Appearance: &metadatapb.Metadata_Appearance{Title: "title2"}}
			want := &metadatapb.Metadata{Name: "name2", Appearance: &metadatapb.Metadata_Appearance{Title: "title1"}}
			mask.Merge(m1, m2) // should only update m1.Name, not m1.Appearance.Title
			if diff := cmp.Diff(want, m1, protocmp.Transform()); diff != "" {
				t.Errorf("mask.Merge() mismatch (-want +got):\n%s", diff)
			}
		}),
		InterceptAfter(func(old, new proto.Message) {
			if n := md(old).Name; n != "initial" {
				t.Fatalf("expected old value to have Name 'initial', got %q", n)
			}
			if n := md(new).Name; n != "merge" {
				t.Fatalf("expected new value to have Name 'merge', got %q", n)
			}
			md(new).Name = "after"
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if n := md(ret).Name; n != "after" {
		t.Fatalf("expected returned value to have Name 'after', got %q", n)
	}
}

// List CollectionChange but without the timestamp
type collectionChange struct {
	Id                 string
	OldValue, NewValue proto.Message
	ChangeType         types.ChangeType
}

func add(t *testing.T, c *Collection, id string, msg proto.Message) {
	t.Helper()
	_, err := c.Add(id, msg)
	if err != nil {
		t.Fatal(err)
	}
}
