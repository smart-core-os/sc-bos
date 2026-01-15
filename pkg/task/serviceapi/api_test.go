package serviceapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func TestApi_PullServices(t *testing.T) {
	assertNoErr := func(tag string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: Unexpected error %v", tag, err)
		}
	}

	m := service.NewMap(createTestLifecycle, service.IdIsUUID)
	api := NewApi(m)
	now := time.UnixMilli(0)
	api.now = func() time.Time {
		return now
	}
	t.Cleanup(service.MapSetNow(m, api.now))

	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	responses := api.pullServices(ctx, &servicespb.PullServicesRequest{UpdatesOnly: true})

	_, state1, err := m.Create("id1", "k1", service.State{})
	assertNoErr("Create", err)

	got, err := receiveWithin(responses, time.Second)
	assertNoErr("Receive", err)

	want := &servicespb.PullServicesResponse_Change{
		Type:       types.ChangeType_ADD,
		ChangeTime: timeToTimestamp(now),
		NewValue:   stateToProto("id1", "k1", state1),
	}
	if diff := cmpProto(want, got); diff != "" {
		t.Fatalf("Change1: (-want,+got)\n%s", diff)
	}

	id1 := m.Get("id1")
	state2, err := id1.Service.Start()
	assertNoErr("id1.Start", err)
	got, err = receiveWithin(responses, time.Millisecond)
	assertNoErr("Receive", err)

	want = &servicespb.PullServicesResponse_Change{
		Type: types.ChangeType_UPDATE, ChangeTime: timeToTimestamp(now),
		OldValue: stateToProto("id1", "k1", state1),
		NewValue: stateToProto("id1", "k1", state2),
	}
	if diff := cmpProto(want, got); diff != "" {
		t.Fatalf("Change2: (-want,+got)\n%s", diff)
	}

	state3, err := m.Delete("id1")
	assertNoErr("Delete", err)
	got, err = receiveWithin(responses, time.Millisecond)
	assertNoErr("Receive", err)

	// there's a race here between the map removing the record and the service being stopped.
	// We don't know which will win, if the record is removed first then we'll get one event, the removal
	// If the stop wins then we'll get two events, an update to Stopped and then the removal.
	// It doesn't actually matter which one wins, but we do need to check in our test.
	if got.Type == types.ChangeType_UPDATE {
		want = &servicespb.PullServicesResponse_Change{
			Type:       types.ChangeType_UPDATE,
			ChangeTime: timeToTimestamp(now),
			OldValue:   stateToProto("id1", "k1", state2),
			NewValue:   stateToProto("id1", "k1", state3),
		}
		if diff := cmpProto(want, got); diff != "" {
			t.Fatalf("Change3 race: (-want,+got)\n%s", diff)
		}

		got, err = receiveWithin(responses, time.Millisecond)
		assertNoErr("Receive race", err)
	}

	want = &servicespb.PullServicesResponse_Change{
		Type: types.ChangeType_REMOVE, ChangeTime: timeToTimestamp(now),
		OldValue: stateToProto("id1", "k1", state3),
	}
	if diff := cmpProto(want, got); diff != "" {
		t.Fatalf("Change2: (-want,+got)\n%s", diff)
	}
}

func cmpProto(want, got any) string {
	return cmp.Diff(want, got, protocmp.Transform())
}

func receiveWithin[T any](c <-chan T, wait time.Duration) (T, error) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case v := <-c:
		return v, nil
	case <-timer.C:
		var zero T
		return zero, context.DeadlineExceeded
	}
}

var createTestLifecycle = func(id, kind string) (service.Lifecycle, error) {
	return newTestLifecycle(), nil
}

func newTestLifecycle() service.Lifecycle {
	return service.New(func(ctx context.Context, config string) error {
		return nil
	}, service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
}
