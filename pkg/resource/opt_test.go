package resource

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/testproto"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
)

// val returns a simple TestAllTypes with DefaultInt32 set to v, used as a distinguishable proto value.
func val(v int32) *testproto.TestAllTypes { return &testproto.TestAllTypes{DefaultInt32: v} }

func TestWithUpdatesOnly(t *testing.T) {
	t.Parallel()

	t.Run("Value (default)", func(t *testing.T) {
		v := NewValue(WithInitialValue(val(1)))
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

		_, err := v.Set(val(2))
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
			val(1), // initial value
			val(2), // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Value (updates only)", func(t *testing.T) {
		v := NewValue(WithInitialValue(val(1)))
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

		_, err := v.Set(val(2))
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
			val(2), // update value
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})

	t.Run("Collection (default)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", val(1))

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

		_, err := v.Update("A", val(2))
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
			{Id: "A", OldValue: nil, NewValue: val(1), ChangeType: typespb.ChangeType_ADD},
			{Id: "A", OldValue: val(1), NewValue: val(2), ChangeType: typespb.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
	t.Run("Collection (updates only)", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", val(1))

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

		_, err := v.Update("A", val(2))
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
			{Id: "A", OldValue: val(1), NewValue: val(2), ChangeType: typespb.ChangeType_UPDATE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithInclude(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		c := NewCollection()
		add(t, c, "A", val(1))
		add(t, c, "B", val(2))
		add(t, c, "C", val(0))

		t.Run("id filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				return id == "B" || id == "C"
			}))
			want := []proto.Message{
				val(2),
				val(0),
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})

		t.Run("body filter", func(t *testing.T) {
			got := c.List(WithInclude(func(id string, item proto.Message) bool {
				itemVal := item.(*testproto.TestAllTypes)
				return itemVal.DefaultInt32 != 0
			}))
			want := []proto.Message{
				val(1),
				val(2),
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Fatalf("(-want,+got)\n%v", diff)
			}
		})
	})

	t.Run("Pull", func(t *testing.T) {
		v := NewCollection()
		add(t, v, "A", val(1))
		add(t, v, "B", val(2))
		add(t, v, "C", val(0))

		ctx, done := context.WithCancel(context.Background())
		t.Cleanup(done)

		// pull only items that are off
		c := v.Pull(ctx, WithInclude(func(_ string, item proto.Message) bool {
			itemVal := item.(*testproto.TestAllTypes)
			return itemVal.DefaultInt32 == 2
		}))
		var events []*CollectionChange
		complete := make(chan struct{})
		go func() {
			defer close(complete)
			for change := range c {
				events = append(events, change)
			}
		}()

		_, err := v.Update("A", val(2))
		if err != nil {
			t.Fatal(err)
		}
		_, err = v.Update("B", val(1))
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
			{Id: "B", NewValue: val(2), ChangeType: typespb.ChangeType_ADD},
			{Id: "A", NewValue: val(2), ChangeType: typespb.ChangeType_ADD},
			{Id: "B", OldValue: val(2), ChangeType: typespb.ChangeType_REMOVE},
		}

		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Errorf("Incorrect events (-want, +got)\n%v", diff)
		}
	})
}

func TestWithBackpressure_False(t *testing.T) {
	v := NewValue(WithInitialValue(val(0)))

	t.Run("false", func(t *testing.T) {
		ctx := t.Context()

		// with backpressure disabled, we can open a Pull, fail to receive, and it doesn't block
		_ = v.Pull(ctx, WithBackpressure(false))
		success := make(chan struct{})
		go func() {
			defer close(success)

			// do a set call, which shouldn't block or error
			_, err := v.Set(val(2))
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
		ctx := t.Context()

		// with backpressure enabled, we can open a Pull, fail to receive, and it will block calls to Set
		_ = v.Pull(ctx, WithBackpressure(true))
		completed := make(chan struct{})
		go func() {
			defer close(completed)

			// do a set call, which should block
			_, err := v.Set(val(2))
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
	add(t, c, "A", val(1))
	expect := val(1)
	actual, ok := c.Get("a")
	if !ok {
		t.Error("expected to find item with id 'a'")
	}
	if diff := cmp.Diff(expect, actual, protocmp.Transform()); diff != "" {
		t.Errorf("Get('a') returned wrong value (-want,+got)\n%v", diff)
	}
}

func TestWithMerger(t *testing.T) {
	td := func(m proto.Message) *testproto.TestAllTypes {
		v, ok := m.(*testproto.TestAllTypes)
		if !ok {
			t.Fatalf("expected *testproto.TestAllTypes, got %T", m)
		}
		return v
	}

	v := NewValue(WithInitialValue(&testproto.TestAllTypes{DefaultString: "initial"}))
	ret, err := v.Set(&testproto.TestAllTypes{DefaultString: "write"},
		WithUpdatePaths("default_string"),
		InterceptBefore(func(old, new proto.Message) {
			if n := td(old).DefaultString; n != "initial" {
				t.Fatalf("expected old value to have DefaultString 'initial', got %q", n)
			}
			if n := td(new).DefaultString; n != "write" {
				t.Fatalf("expected new value to have DefaultString 'write', got %q", n)
			}
			td(new).DefaultString = "before"
		}),
		WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
			if n := td(dst).DefaultString; n != "initial" {
				t.Fatalf("expected dst value to have DefaultString 'initial', got %q", n)
			}
			if n := td(src).DefaultString; n != "before" {
				t.Fatalf("expected src value to have DefaultString 'before', got %q", n)
			}
			td(dst).DefaultString = "merge"

			// test that the mask updates what we expect
			m1 := &testproto.TestAllTypes{DefaultString: "name1", DefaultFloat: 1.0}
			m2 := &testproto.TestAllTypes{DefaultString: "name2", DefaultFloat: 2.0}
			want := &testproto.TestAllTypes{DefaultString: "name2", DefaultFloat: 1.0}
			mask.Merge(m1, m2) // should only update m1.DefaultString, not m1.DefaultFloat
			if diff := cmp.Diff(want, m1, protocmp.Transform()); diff != "" {
				t.Errorf("mask.Merge() mismatch (-want +got):\n%s", diff)
			}
		}),
		InterceptAfter(func(old, new proto.Message) {
			if n := td(old).DefaultString; n != "initial" {
				t.Fatalf("expected old value to have DefaultString 'initial', got %q", n)
			}
			if n := td(new).DefaultString; n != "merge" {
				t.Fatalf("expected new value to have DefaultString 'merge', got %q", n)
			}
			td(new).DefaultString = "after"
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if n := td(ret).DefaultString; n != "after" {
		t.Fatalf("expected returned value to have DefaultString 'after', got %q", n)
	}
}

// List CollectionChange but without the timestamp
type collectionChange struct {
	Id                 string
	OldValue, NewValue proto.Message
	ChangeType         typespb.ChangeType
}

func add(t *testing.T, c *Collection, id string, msg proto.Message) {
	t.Helper()
	_, err := c.Add(id, msg)
	if err != nil {
		t.Fatal(err)
	}
}
