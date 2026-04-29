package group

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestExecuteUpTo(t *testing.T) {
	t.Run("all ok", func(t *testing.T) {
		got, err := ExecuteUpTo(context.Background(), 0, []Member{
			ok("one"),
			ok("two"),
		})
		if err != nil {
			t.Fatalf("got err %v", err)
		}
		want := []proto.Message{
			msg("one"),
			msg("two"),
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})
	t.Run("enough ok", func(t *testing.T) {
		got, err := ExecuteUpTo(context.Background(), 1, []Member{
			fail("one"),
			ok("two"),
			ok("three"),
		})
		if err != nil {
			t.Fatalf("got err %v", err)
		}
		want := []proto.Message{
			nil,
			msg("two"),
			msg("three"),
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})
	t.Run("one fail", func(t *testing.T) {
		got, err := ExecuteUpTo(context.Background(), 0, []Member{
			fail("one"),
			ok("two"),
		})
		if err == nil {
			t.Fatalf("err is nil")
		}

		if err.Error() != "one" {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []proto.Message{
			nil,
			msg("two"),
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})
	t.Run("not enough ok", func(t *testing.T) {
		got, err := ExecuteUpTo(context.Background(), 1, []Member{
			fail("fail"),
			fail("fail"),
			ok("three"),
		})
		if err == nil {
			t.Fatalf("err is nil")
		}
		if err.Error() != "fail" {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []proto.Message{
			nil,
			nil,
			msg("three"),
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})
	t.Run("even fails", func(t *testing.T) {
		got, err := ExecuteUpTo(context.Background(), 2, []Member{
			fail("fail"),
			ok("two"),
			fail("fail"),
			ok("four"),
		})
		if err != nil {
			t.Fatalf("got err %v", err)
		}
		want := []proto.Message{
			nil,
			msg("two"),
			nil,
			msg("four"),
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})

	t.Run("eventually ok", func(t *testing.T) {
		ticks := make(chan struct{})
		result := make(chan msgsAndErr, 1) // expecting 1 result
		assertResultEmpty := func() {
			select {
			case r := <-result:
				t.Fatalf("expecting no result, got %v", r)
			default:
				// all ok
			}
		}

		go func() {
			msgs, err := ExecuteUpTo(context.Background(), 0, []Member{
				okLater("one", ticks),
				okLater("two", ticks),
			})
			result <- msgsAndErr{msgs, err}
		}()

		assertResultEmpty()
		ticks <- struct{}{}
		assertResultEmpty()
		ticks <- struct{}{}
		got := <-result
		want := []proto.Message{
			msg("one"),
			msg("two"),
		}
		if diff := cmp.Diff(want, got.msgs, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})

	t.Run("first fail", func(t *testing.T) {
		ticks := make(chan struct{})
		result := make(chan msgsAndErr, 1) // expecting 1 result

		go func() {
			msgs, err := ExecuteUpTo(context.Background(), 0, []Member{
				failLater("error", ticks),
				failLater("error", ticks),
			})
			result <- msgsAndErr{msgs, err}
		}()

		ticks <- struct{}{}

		r := <-result
		got := r.msgs
		err := r.err

		if err.Error() != "error" {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []proto.Message{
			nil,
			nil,
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.Fatalf("ExecuteUpTo (-want +got)\n%s", diff)
		}
	})

}

func TestExecuteOne(t *testing.T) {

}

func okLater(val string, ticks <-chan struct{}) Member {
	return func(ctx context.Context) (proto.Message, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticks:
			return msg(val), nil
		}
	}
}

func failLater(err string, ticks <-chan struct{}) Member {
	return func(ctx context.Context) (proto.Message, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticks:
			return nil, errors.New(err)
		}
	}
}

func ok(val string) Member {
	return func(_ context.Context) (proto.Message, error) {
		return msg(val), nil
	}
}

func fail(err string) Member {
	return func(_ context.Context) (proto.Message, error) {
		return nil, errors.New(err)
	}
}

func msg(val string) proto.Message {
	return &testMessage{val}
}

type testMessage struct {
	val string
}

func (t *testMessage) ProtoReflect() protoreflect.Message {
	return nil
}

type msgsAndErr struct {
	msgs []proto.Message
	err  error
}
