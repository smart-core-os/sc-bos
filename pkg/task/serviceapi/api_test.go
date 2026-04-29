package serviceapi

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
)

func TestApi_PullServices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		assertNoErr := func(tag string, err error) {
			t.Helper()
			if err != nil {
				t.Fatalf("%s: Unexpected error %v", tag, err)
			}
		}

		m := service.NewMap(createTestLifecycle, service.IdIsUUID)
		api := NewApi(m)
		now := time.Now()

		ctx := t.Context()
		responses := api.pullServices(ctx, &servicespb.PullServicesRequest{UpdatesOnly: true})

		_, state1, err := m.Create("id1", "k1", service.State{})
		assertNoErr("Create", err)

		synctest.Wait()
		got := <-responses

		want := &servicespb.PullServicesResponse_Change{
			Type:       typespb.ChangeType_ADD,
			ChangeTime: timeToTimestamp(now),
			NewValue:   stateToProto("id1", "k1", state1),
		}
		if diff := cmpProto(want, got); diff != "" {
			t.Fatalf("Change1: (-want,+got)\n%s", diff)
		}

		id1 := m.Get("id1")
		state2, err := id1.Service.Start()
		assertNoErr("id1.Start", err)

		synctest.Wait()
		got = <-responses

		want = &servicespb.PullServicesResponse_Change{
			Type: typespb.ChangeType_UPDATE, ChangeTime: timeToTimestamp(now),
			OldValue: stateToProto("id1", "k1", state1),
			NewValue: stateToProto("id1", "k1", state2),
		}
		if diff := cmpProto(want, got); diff != "" {
			t.Fatalf("Change2: (-want,+got)\n%s", diff)
		}

		state3, err := m.Delete("id1")
		assertNoErr("Delete", err)

		synctest.Wait()
		got = <-responses

		// there's a race here between the map removing the record and the service being stopped.
		// We don't know which will win, if the record is removed first then we'll get one event, the removal
		// If the stop wins then we'll get two events, an update to Stopped and then the removal.
		// It doesn't actually matter which one wins, but we do need to check in our test.
		if got.Type == typespb.ChangeType_UPDATE {
			want = &servicespb.PullServicesResponse_Change{
				Type:       typespb.ChangeType_UPDATE,
				ChangeTime: timeToTimestamp(now),
				OldValue:   stateToProto("id1", "k1", state2),
				NewValue:   stateToProto("id1", "k1", state3),
			}
			if diff := cmpProto(want, got); diff != "" {
				t.Fatalf("Change3 race: (-want,+got)\n%s", diff)
			}

			synctest.Wait()
			got = <-responses
		}

		want = &servicespb.PullServicesResponse_Change{
			Type: typespb.ChangeType_REMOVE, ChangeTime: timeToTimestamp(now),
			OldValue: stateToProto("id1", "k1", state3),
		}
		if diff := cmpProto(want, got); diff != "" {
			t.Fatalf("Change4: (-want,+got)\n%s", diff)
		}
	})
}

func cmpProto(want, got any) string {
	return cmp.Diff(want, got, protocmp.Transform())
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
