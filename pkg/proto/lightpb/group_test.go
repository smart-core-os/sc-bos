package lightpb

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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/th"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
)

// todo: test one, some, all failures for get, update, pull

func TestGroup_GetBrightness(t *testing.T) {
	tester := newBrightnessTester(t, "A", "B")
	// initial state
	tester.assertGet(&Brightness{})
	// 0, 100
	tester.prepare(&Brightness{LevelPercent: 100}, "B")
	tester.assertGet(&Brightness{LevelPercent: 50})
	// 80, 80
	tester.prepare(&Brightness{LevelPercent: 80}, "A", "B")
	tester.assertGet(&Brightness{LevelPercent: 80})
	// 60, 80
	tester.prepare(&Brightness{LevelPercent: 60}, "A")
	tester.assertGet(&Brightness{LevelPercent: 70})
}

func TestGroup_UpdateBrightness(t *testing.T) {
	tester := newBrightnessTester(t, "A", "B")
	// check no writes happen without us knowing
	tester.confirm(&Brightness{}, "A", "B")
	// set a value
	tester.assertUpdate(&Brightness{LevelPercent: 40})
	tester.confirm(&Brightness{LevelPercent: 40}, "A", "B")
	// set another value
	tester.assertUpdate(&Brightness{LevelPercent: 90})
	tester.confirm(&Brightness{LevelPercent: 90}, "A", "B")
}

func TestGroup_PullBrightness(t *testing.T) {
	tester := newBrightnessTester(t, "A", "B").pull()
	// initial value sent immediately
	tester.assertPull(&Brightness{})
	// message on first change: 40, 0
	tester.prepare(&Brightness{LevelPercent: 40}, "A")
	tester.assertPull(&Brightness{LevelPercent: 20})
	// message on first change: 40, 40
	tester.prepare(&Brightness{LevelPercent: 40}, "B")
	tester.assertPull(&Brightness{LevelPercent: 40})
	// message on first change: 20, 40
	tester.prepare(&Brightness{LevelPercent: 20}, "A")
	tester.assertPull(&Brightness{LevelPercent: 30})
}

type brightnessTester struct {
	t    *testing.T
	subj LightApiClient
	impl LightApiClient
}

func newBrightnessTester(t *testing.T, members ...string) *brightnessTester {
	devices := NewApiRouter(WithLightApiClientFactory(func(name string) (LightApiClient, error) {
		return NewLightApiClient(wrap.ServerToClient(LightApi_ServiceDesc, NewModelServer(NewModel()))), nil
	}))
	impl := NewLightApiClient(wrap.ServerToClient(LightApi_ServiceDesc, devices))
	group := NewGroup(impl, members...)

	// server and client setup
	lis := bufconn.Listen(1024 * 1024)
	// setup the server
	server := grpc.NewServer()
	RegisterLightApiServer(server, group)
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

	client := NewLightApiClient(conn)

	return &brightnessTester{
		t:    t,
		subj: client,
		impl: impl,
	}
}

func (t *brightnessTester) prepare(state *Brightness, names ...string) {
	t.t.Helper()
	for _, name := range names {
		_, err := t.impl.UpdateBrightness(th.Ctx, &UpdateBrightnessRequest{Name: name, Brightness: state})
		th.CheckErr(t.t, err, fmt.Sprintf("%v.UpdateBrightness", name))
	}
}

func (t *brightnessTester) confirm(state *Brightness, names ...string) {
	t.t.Helper()
	type badName struct {
		name  string
		state *Brightness
	}
	var badNames []badName
	for _, name := range names {
		got, err := t.impl.GetBrightness(th.Ctx, &GetBrightnessRequest{Name: name})
		th.CheckErr(t.t, err, fmt.Sprintf("%v.GetBrightness", name))
		if !proto.Equal(got, state) {
			badNames = append(badNames, badName{name: name, state: got})
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

func (t *brightnessTester) assertGet(expected *Brightness) {
	t.t.Helper()
	res, err := t.subj.GetBrightness(th.Ctx, &GetBrightnessRequest{Name: "Parent"})
	th.CheckErr(t.t, err, "Parent.GetBrightness")
	if diff := cmp.Diff(expected, res, protocmp.Transform()); diff != "" {
		t.t.Fatalf("Parent.GetBrightness (-want,+got)\n%v", diff)
	}
}

func (t *brightnessTester) assertUpdate(state *Brightness, membersUpdated ...string) {
	t.t.Helper()
	updateState, err := t.subj.UpdateBrightness(th.Ctx, &UpdateBrightnessRequest{Name: "Parent", Brightness: state})
	th.CheckErr(t.t, err, "Parent.UpdateBrightness")
	// note: can't compare the update result with the given state as we might be updating just a few
	// It's more correct to compare with the GetBrightness state as that uses the same merge strategy
	getState, err := t.subj.GetBrightness(th.Ctx, &GetBrightnessRequest{Name: "Parent"})
	if diff := cmp.Diff(getState, updateState, protocmp.Transform()); diff != "" {
		t.t.Fatalf("Update state doesn't match read state (-want, +got)\n%v", diff)
	}
}

func (t *brightnessTester) pull() *brightnessStreamTester {
	t.t.Helper()
	s, err := t.subj.PullBrightness(th.Ctx, &PullBrightnessRequest{Name: "Parent"})
	th.CheckErr(t.t, err, "Parent.PullBrightness")
	return &brightnessStreamTester{brightnessTester: t, s: s, c: make(chan brightnessStreamMsg, 10)}
}

type brightnessStreamTester struct {
	*brightnessTester
	s LightApi_PullBrightnessClient
	c chan brightnessStreamMsg

	startOnce sync.Once
}

type brightnessStreamMsg struct {
	msg *PullBrightnessResponse
	err error
}

func (t *brightnessStreamTester) start() {
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
				case t.c <- brightnessStreamMsg{msg, err}:
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

func (t *brightnessStreamTester) assertNone() {
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

func (t *brightnessStreamTester) assertPull(want *Brightness) {
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
			t.t.Fatalf("Parent.PullBrightness.Recv want %v, got error %v", want, v.err)
		}
		if len(v.msg.Changes) == 0 {
			t.t.Fatalf("Parent.PullBrightness.Recv want %v, got no changes", want)
		}
		lastChange := v.msg.Changes[len(v.msg.Changes)-1]
		if lastChange.Name != "Parent" {
			t.t.Fatalf("Parent.PullBrightness.Recv Name want %v, got %v", "Parent", lastChange.Name)
		}
		got := lastChange.Brightness
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
			t.t.Fatalf("Parent.PullBrightness.Recv (-want,+got)\n%v", diff)
		}
	}

}
