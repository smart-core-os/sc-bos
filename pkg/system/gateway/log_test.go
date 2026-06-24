package gateway

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
)

// logTraitMd returns metadata advertising the Log trait, as pullDevices would
// record for a node's Log-trait device.
func logTraitMd() *metadatapb.Metadata {
	return &metadatapb.Metadata{Traits: []*metadatapb.TraitMetadata{{Name: string(logpb.TraitName)}}}
}

// TestLogAggregator_PullLogMessages verifies the gateway merges the LogApi
// streams of every cohort node that advertises the Log trait, tags each message
// with its source node, and picks up nodes added after the stream starts.
func TestLogAggregator_PullLogMessages(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, hub := newMockCohort(t)
		gw1 := env.newGatewayNode("gw1")
		ac1 := env.newNode("ac1")
		ac2 := env.newNode("ac2")

		// ac1 and ac2 expose the Log trait; the hub does not (so it contributes nothing).
		ac1Model := ac1.announceLog("ac1", logMsg("from ac1"))
		ac2.announceLog("ac2", logMsg("from ac2"))

		gw1Sys := &System{
			logger:     zaptest.NewLogger(t),
			self:       gw1.node,
			hub:        hub,
			reflection: gw1.reflect,
			newClient:  env.newClient,
		}
		gw1Cohort := newCohort()
		go gw1Sys.scanRemoteHub(t.Context(), gw1Cohort, hub.conn)
		synctest.Wait() // cohort populated: each node's devices + conns

		agg := newLogAggregator(gw1Cohort, nil, zaptest.NewLogger(t))
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		srv := &fakeLogServer{ctx: ctx}
		go func() {
			_ = agg.PullLogMessages(&logpb.PullLogMessagesRequest{Name: aggregateLogName, InitialCount: 100}, srv)
		}()
		synctest.Wait()

		// initial replay is merged from both nodes, each stamped with its source.
		got := srv.bySource()
		if !slices.Contains(got["ac1"], "from ac1") {
			t.Errorf("missing ac1 initial log; got %v", got)
		}
		if !slices.Contains(got["ac2"], "from ac2") {
			t.Errorf("missing ac2 initial log; got %v", got)
		}
		if _, ok := got[""]; ok {
			t.Errorf("found message with empty source: %v", got[""])
		}

		// a live message appended after the stream is established flows through too.
		ac1Model.AppendMessage(logMsg("ac1 live"))
		synctest.Wait()
		if got = srv.bySource(); !slices.Contains(got["ac1"], "ac1 live") {
			t.Errorf("missing ac1 live log; got %v", got["ac1"])
		}

		// a node enrolled after the stream started is picked up automatically.
		ac3 := env.newNode("ac3")
		ac3.announceLog("ac3", logMsg("from ac3"))
		synctest.Wait()
		if got = srv.bySource(); !slices.Contains(got["ac3"], "from ac3") {
			t.Errorf("missing log from node added mid-stream; got %v", got)
		}
	})
}

// TestLogAggregator_SkipsAggregatedDevices guards against double-counting: when a
// cohort node (e.g. a hub running a gateway) re-advertises another node's device,
// the aggregator must stream that device's logs only via the node that owns it,
// not via every node that merely aggregates it.
func TestLogAggregator_SkipsAggregatedDevices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// gwBox stands in for a hub/gateway that aggregates ac1's "ac1" device; acBox
		// is ac1 itself, reached directly. Both serve the Log trait at name "ac1" with
		// the same backlog, so a naive aggregator would emit it twice.
		gwBox := newMockRemoteNode(t, "gwbox")
		acBox := newMockRemoteNode(t, "acbox")
		gwBox.announceLog("ac1", logMsg("only once"))
		acBox.announceLog("ac1", logMsg("only once"))

		c := newCohort()
		gwRemote := newRemoteNode("gwbox", gwBox.conn)
		gwRemote.Self.Set(remoteDesc{name: "hub"}) // owns "hub"; "ac1" is merely aggregated
		gwRemote.Devices.Set(remoteDesc{name: "ac1", md: logTraitMd()})
		c.Nodes.Set(gwRemote)
		acRemote := newRemoteNode("acbox", acBox.conn)
		acRemote.Self.Set(remoteDesc{name: "ac1"}) // owns "ac1"
		acRemote.Devices.Set(remoteDesc{name: "ac1", md: logTraitMd()})
		c.Nodes.Set(acRemote)

		agg := newLogAggregator(c, nil, zaptest.NewLogger(t))
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		srv := &fakeLogServer{ctx: ctx}
		go func() {
			_ = agg.PullLogMessages(&logpb.PullLogMessagesRequest{Name: aggregateLogName, InitialCount: 100}, srv)
		}()
		synctest.Wait()

		if got := srv.bySource()["ac1"]; len(got) != 1 {
			t.Errorf("want ac1 log streamed once (via its owner), got %d copies: %v", len(got), got)
		}
	})
}

// TestLogAggregator_NoBacklogReplayOnReAdd checks that a device which is removed
// and re-announced does not have its backlog replayed (which would duplicate
// lines); only new live messages should flow after the re-add.
func TestLogAggregator_NoBacklogReplayOnReAdd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		acBox := newMockRemoteNode(t, "acbox")
		model := acBox.announceLog("ac1", logMsg("backlog"))

		c := newCohort()
		acRemote := newRemoteNode("acbox", acBox.conn)
		acRemote.Self.Set(remoteDesc{name: "ac1"})
		dev := remoteDesc{name: "ac1", md: logTraitMd()}
		acRemote.Devices.Set(dev)
		c.Nodes.Set(acRemote)

		agg := newLogAggregator(c, nil, zaptest.NewLogger(t))
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		srv := &fakeLogServer{ctx: ctx}
		go func() {
			_ = agg.PullLogMessages(&logpb.PullLogMessagesRequest{Name: aggregateLogName, InitialCount: 100}, srv)
		}()
		synctest.Wait()
		if got := srv.bySource()["ac1"]; !slices.Equal(got, []string{"backlog"}) {
			t.Fatalf("initial replay: want [backlog], got %v", got)
		}

		// Remove then re-add the device. The backlog must not be replayed.
		acRemote.Devices.Remove(dev)
		synctest.Wait()
		acRemote.Devices.Set(dev)
		synctest.Wait()

		// A new live message after the re-add should still flow.
		model.AppendMessage(logMsg("after readd"))
		synctest.Wait()

		got := srv.bySource()["ac1"]
		if want := []string{"backlog", "after readd"}; !slices.Equal(got, want) {
			t.Errorf("after re-add: want %v (backlog not replayed), got %v", want, got)
		}
	})
}

// TestLogAggregator_IncludesOwnNode verifies the aggregate also covers the
// gateway node itself (which is not a cohort member), and that the native-name
// filter still keeps it to the node's own devices rather than the remote devices
// it proxies into its own DevicesApi.
func TestLogAggregator_IncludesOwnNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// selfBox is the gateway node. It serves the Log trait for its own device
		// ("gw") and, as a gateway does, also re-advertises a proxied remote device
		// ("ac1"). Only the former should be aggregated as self's contribution.
		selfBox := newMockRemoteNode(t, "gw")
		selfBox.announceLog("gw", logMsg("from gw itself"))
		selfBox.announceLog("ac1", logMsg("proxied via gw"))

		self := newRemoteNode("gw", selfBox.conn)
		self.Self.Set(remoteDesc{name: "gw"})
		self.Devices.Set(remoteDesc{name: "gw", md: logTraitMd()})  // native to this node
		self.Devices.Set(remoteDesc{name: "ac1", md: logTraitMd()}) // proxied; not native

		agg := newLogAggregator(newCohort(), self, zaptest.NewLogger(t)) // empty cohort: only self contributes
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		srv := &fakeLogServer{ctx: ctx}
		go func() {
			_ = agg.PullLogMessages(&logpb.PullLogMessagesRequest{Name: aggregateLogName, InitialCount: 100}, srv)
		}()
		synctest.Wait()

		got := srv.bySource()
		if !slices.Contains(got["gw"], "from gw itself") {
			t.Errorf("aggregate should include the gateway's own logs; got %v", got)
		}
		if _, ok := got["ac1"]; ok {
			t.Errorf("self should not stream proxied (non-native) devices; got %v", got["ac1"])
		}
	})
}

// announceLog makes the mock node serve the Log trait under name, seeded with
// msgs. The returned model can be used to append further messages live.
func (n *mockRemoteNode) announceLog(name string, msgs ...*logpb.LogMessage) *logpb.Model {
	n.t.Helper()
	model := logpb.NewModel(1000)
	for _, m := range msgs {
		model.AppendMessage(m)
	}
	srv := logpb.NewModelServer(model)
	n.node.Announce(name,
		node.HasServer(logpb.RegisterLogApiServer, logpb.LogApiServer(srv)),
		node.HasTrait(logpb.TraitName),
	)
	return model
}

func logMsg(text string) *logpb.LogMessage {
	return &logpb.LogMessage{
		Timestamp: timestamppb.Now(),
		Level:     logpb.Level_LEVEL_INFO,
		Message:   text,
	}
}

// fakeLogServer is a logpb.LogApi_PullLogMessagesServer that records everything sent to it.
type fakeLogServer struct {
	ctx  context.Context
	mu   sync.Mutex
	sent []*logpb.PullLogMessagesResponse
}

func (f *fakeLogServer) Send(resp *logpb.PullLogMessagesResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, resp)
	return nil
}

// bySource collapses everything received into source-node -> message texts.
func (f *fakeLogServer) bySource() map[string][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]string{}
	for _, resp := range f.sent {
		for _, ch := range resp.Changes {
			for _, m := range ch.Messages {
				out[m.Source] = append(out[m.Source], m.Message)
			}
		}
	}
	return out
}

func (f *fakeLogServer) Context() context.Context     { return f.ctx }
func (f *fakeLogServer) SetHeader(metadata.MD) error  { return nil }
func (f *fakeLogServer) SendHeader(metadata.MD) error { return nil }
func (f *fakeLogServer) SetTrailer(metadata.MD)       {}
func (f *fakeLogServer) SendMsg(any) error            { return nil }
func (f *fakeLogServer) RecvMsg(any) error            { return nil }
