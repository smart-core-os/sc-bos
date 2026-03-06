package onoffpb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-golang/internal/th"
)

// todo: test one, some, all failures for get, update, pull

func TestGroup_GetOnOff(t *testing.T) {
	tester := newOnOffTester(t, "A", "B")
	// initial state
	tester.assertGet(traits.OnOff_STATE_UNSPECIFIED)
	// one off
	tester.prepare(traits.OnOff_OFF, "B")
	tester.assertGet(traits.OnOff_OFF)
	// all off
	tester.prepare(traits.OnOff_OFF, "A", "B")
	tester.assertGet(traits.OnOff_OFF)
	// all on
	tester.prepare(traits.OnOff_ON, "A", "B")
	tester.assertGet(traits.OnOff_ON)
	// some on, prefer on
	tester.prepare(traits.OnOff_OFF, "A")
	tester.assertGet(traits.OnOff_ON)
}

func TestGroup_UpdateOnOff(t *testing.T) {
	tester := newOnOffTester(t, "A", "B")
	// check no writes happen without us knowing
	tester.confirm(traits.OnOff_STATE_UNSPECIFIED, "A", "B")
	// turn everything on
	tester.assertUpdate(traits.OnOff_ON)
	tester.confirm(traits.OnOff_ON, "A", "B")
	// turn everything off
	tester.assertUpdate(traits.OnOff_OFF)
	tester.confirm(traits.OnOff_OFF, "A", "B")
	// turn everything off again
	tester.prepare(traits.OnOff_ON, "A", "B")
	tester.assertUpdate(traits.OnOff_OFF)
	tester.confirm(traits.OnOff_OFF, "A", "B")
}

func TestGroup_PullOnOff(t *testing.T) {
	tester := newOnOffTester(t, "A", "B").pull()
	// current value sent immediately
	tester.assertPull(traits.OnOff_STATE_UNSPECIFIED)
	// message on first change
	tester.prepare(traits.OnOff_ON, "A")
	tester.assertPull(traits.OnOff_ON)
	// no message if no change
	tester.prepare(traits.OnOff_ON, "B")
	tester.assertNone()
	// message on subsequent change
	tester.prepare(traits.OnOff_OFF, "A", "B")
	tester.assertPull(traits.OnOff_OFF)
}

type onOffTester struct {
	t    *testing.T
	subj traits.OnOffApiClient
	impl traits.OnOffApiClient
}

func newOnOffTester(t *testing.T, members ...string) *onOffTester {
	devices := NewApiRouter(WithOnOffApiClientFactory(func(name string) (traits.OnOffApiClient, error) {
		return WrapApi(NewModelServer(NewModel())), nil
	}))
	impl := WrapApi(devices)
	group := NewGroup(impl, members...)

	// server and client setup
	lis := bufconn.Listen(1024 * 1024)
	// setup the server
	server := grpc.NewServer()
	traits.RegisterOnOffApiServer(server, group)
	t.Cleanup(func() {
		server.Stop()
	})
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("Server existed with error: %v", err)
		}
	}()

	// setup the client
	conn, err := th.Dial(lis)
	th.CheckErr(t, err, "dial")
	t.Cleanup(func() {
		conn.Close()
	})

	client := traits.NewOnOffApiClient(conn)

	return &onOffTester{
		t:    t,
		subj: client,
		impl: impl,
	}
}

func (t *onOffTester) prepare(state traits.OnOff_State, names ...string) {
	t.t.Helper()
	for _, name := range names {
		_, err := t.impl.UpdateOnOff(th.Ctx, &traits.UpdateOnOffRequest{Name: name, OnOff: &traits.OnOff{State: state}})
		th.CheckErr(t.t, err, fmt.Sprintf("%v.UpdateOnOff", name))
	}
}

func (t *onOffTester) confirm(state traits.OnOff_State, names ...string) {
	t.t.Helper()
	type badName struct {
		name  string
		state traits.OnOff_State
	}
	var badNames []badName
	for _, name := range names {
		got, err := t.impl.GetOnOff(th.Ctx, &traits.GetOnOffRequest{Name: name})
		th.CheckErr(t.t, err, fmt.Sprintf("%v.GetOnOff", name))
		if got.GetState() != state {
			badNames = append(badNames, badName{name: name, state: got.GetState()})
		}
	}

	l := len(badNames)
	switch {
	case l == 1:
		t.t.Fatalf("%v state is unexpected: want %v, got %v", badNames[0].name, state, badNames[0].state)
	case l > 1:
		lines := make([]string, len(badNames))
		for i, badName := range badNames {
			lines[i] = fmt.Sprintf("%v=%v", badName.name, badName.state)
		}
		t.t.Fatalf("%v members have unexpected state: want %v, got %v", len(badNames), state, strings.Join(lines, ", "))
	}
}

func (t *onOffTester) assertGet(expected traits.OnOff_State) {
	t.t.Helper()
	res, err := t.subj.GetOnOff(th.Ctx, &traits.GetOnOffRequest{Name: "Parent"})
	th.CheckErr(t.t, err, "Parent.GetOnOff")
	if res.GetState() != expected {
		t.t.Fatalf("Parent.GetOnOff want %v, got %v", expected, res.GetState())
	}
}

func (t *onOffTester) assertUpdate(state traits.OnOff_State, membersUpdated ...string) {
	t.t.Helper()
	updateState, err := t.subj.UpdateOnOff(th.Ctx, &traits.UpdateOnOffRequest{Name: "Parent", OnOff: &traits.OnOff{State: state}})
	th.CheckErr(t.t, err, "Parent.UpdateOnOff")
	// note: can't compare the update result with the given state as we might be updating just a few
	// It's more correct to compare with the GetOnOff state as that uses the same merge strategy
	getState, err := t.subj.GetOnOff(th.Ctx, &traits.GetOnOffRequest{Name: "Parent"})
	if diff := cmp.Diff(getState, updateState, protocmp.Transform()); diff != "" {
		t.t.Fatalf("Update state doesn't match read state (-want, +got)\n%v", diff)
	}
}

func (t *onOffTester) pull() *onOffStreamTester {
	t.t.Helper()
	s, err := t.subj.PullOnOff(th.Ctx, &traits.PullOnOffRequest{Name: "Parent"})
	th.CheckErr(t.t, err, "Parent.PullOnOff")
	return &onOffStreamTester{onOffTester: t, s: s, c: make(chan onOffStreamMsg, 10)}
}

type onOffStreamTester struct {
	*onOffTester
	s traits.OnOffApi_PullOnOffClient
	c chan onOffStreamMsg

	startOnce sync.Once
}

type onOffStreamMsg struct {
	msg *traits.PullOnOffResponse
	err error
}

func (t *onOffStreamTester) start() {
	t.t.Helper()
	t.startOnce.Do(func() {
		t.t.Helper()
		ctx, done := context.WithCancel(th.Ctx)
		t.t.Cleanup(done)

		started := make(chan struct{}, 1)

		go func() {
			t.t.Helper()
			// haven't technically started yet, but this is closer than without the goroutine
			go func() {
				started <- struct{}{}
			}()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				msg, err := t.s.Recv()
				sendTimeout := time.NewTimer(th.StreamTimout)
				select {
				case <-sendTimeout.C:
					t.t.Errorf("Message received when none were expected: %v %v", msg, err)
					return
				case t.c <- onOffStreamMsg{msg, err}:
					sendTimeout.Stop()
				}
				if err != nil {
					return
				}
			}
		}()

		<-started
	})
}

func (t *onOffStreamTester) assertNone() {
	t.start()
	t.t.Helper()
	timer := time.NewTimer(th.StreamTimout)
	defer timer.Stop()
	select {
	case v := <-t.c:
		t.t.Fatalf("No messages expected on stream, got %v", v)
	case <-timer.C:
		// good case
	}
}

func (t *onOffStreamTester) assertPull(want traits.OnOff_State) {
	t.start()
	t.t.Helper()
	now := time.Now()
	timer := time.NewTimer(th.StreamTimout)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.t.Fatalf("want %v, got timeout after %v", want, time.Now().Sub(now))
	case v := <-t.c:
		if v.err != nil {
			t.t.Fatalf("Parent.PullOnOff.Recv want %v, got error %v", want, v.err)
		}
		if len(v.msg.Changes) == 0 {
			t.t.Fatalf("Parent.PullOnOff.Recv want %v, got no changes", want)
		}
		lastChange := v.msg.Changes[len(v.msg.Changes)-1]
		if lastChange.Name != "Parent" {
			t.t.Fatalf("Parent.PullOnOff.Recv Name want %v, got %v", "Parent", lastChange.Name)
		}
		got := lastChange.OnOff.State
		if want != got {
			t.t.Fatalf("Parent.PullOnOff.Recv want %v, got %v", want, got)
		}
	}

}
