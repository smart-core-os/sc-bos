package gateway

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/logpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/rx"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

// aggregateLogName is the device name under which the gateway exposes the
// cohort-wide aggregated Log trait. Querying PullLogMessages with this name
// merges the logs of every cohort node into a single stream.
const aggregateLogName = "logs"

// logAggregator implements logpb.LogApiServer by merging the PullLogMessages
// streams of every cohort node into one, stamping each message with the name of
// the node it came from.
//
// Only PullLogMessages is aggregated; per-node log level and metadata management
// remain reachable via the gateway's per-node proxy (e.g. UpdateLogLevel with
// name="ac1"). All other LogApi methods return codes.Unimplemented via the
// embedded UnimplementedLogApiServer.
type logAggregator struct {
	logpb.UnimplementedLogApiServer
	cohort *cohort
	// self represents this gateway node itself, streamed alongside the cohort so
	// the aggregate includes the node an operator is connected to (which is not a
	// cohort member). nil if this node has no name to attribute its logs to.
	self   *remoteNode
	logger *zap.Logger
}

func newLogAggregator(c *cohort, self *remoteNode, logger *zap.Logger) *logAggregator {
	return &logAggregator{cohort: c, self: self, logger: logger}
}

// PullLogMessages fans the request out to every cohort node that advertises the
// Log trait, merging their responses into server. Each message is tagged with
// its source node name. The merge runs until the caller's context is done.
func (a *logAggregator) PullLogMessages(req *logpb.PullLogMessagesRequest, server logpb.LogApi_PullLogMessagesServer) error {
	ctx := server.Context()
	merged := make(chan *logpb.PullLogMessagesResponse)

	// Cohort membership is managed in its own goroutine, deliberately decoupled
	// from the server.Send loop below. server.Send blocks under gRPC flow control
	// whenever the client reads slowly; if we serviced cohort.Nodes changes in the
	// same loop, that stall would back up into cohort.Nodes' minibus fan-out, which
	// holds the rx.Set mutex until every listener accepts. A single slow log client
	// would then freeze cohort membership updates for the whole gateway.
	go a.streamNodes(ctx, req, merged)

	// The gateway node itself isn't in the cohort; stream its own logs too so the
	// aggregate covers every node, including the one the caller is connected to.
	if a.self != nil {
		go a.streamNode(ctx, a.self, req, merged)
	}

	// Single sender: only this goroutine calls server.Send, so the stream stays
	// safe for concurrent use while many nodes feed merged.
	for {
		select {
		case <-ctx.Done():
			return nil
		case resp := <-merged:
			if err := server.Send(resp); err != nil {
				return err
			}
		}
	}
}

// streamNodes keeps one streamNode worker running per cohort node, forwarding
// their merged output to out. It runs until ctx is done. Keeping this loop
// separate from PullLogMessages' server.Send loop ensures a slow gRPC client can
// never stall the cohort.Nodes subscription (see PullLogMessages).
func (a *logAggregator) streamNodes(ctx context.Context, req *logpb.PullLogMessagesRequest, out chan<- *logpb.PullLogMessagesResponse) {
	// A streamer goroutine per node, keyed by addr. Cancelling its context stops
	// the streamer. All bookkeeping happens in the single loop below so the map
	// is never touched concurrently.
	streamers := tasks{}
	defer streamers.callAll()

	start := func(n *remoteNode) {
		nodeCtx, stop := context.WithCancel(ctx)
		streamers[n.addr] = stop
		go a.streamNode(nodeCtx, n, req, out)
	}

	nodes, nodeChanges := a.cohort.Nodes.Sub(ctx)
	for _, n := range nodes.All {
		start(n)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case change, ok := <-nodeChanges:
			if !ok {
				return
			}
			if change.Old != nil {
				streamers.remove(change.Old.addr)
			}
			if change.New != nil {
				start(change.New)
			}
		}
	}
}

// activeStream tracks one in-flight per-device log stream. Its pointer identity
// lets streamNode tell, when a stream reports it has stopped, whether the map
// entry still refers to that same stream rather than a newer one started after a
// quick remove/re-add.
type activeStream struct {
	stop context.CancelFunc
}

// streamNode forwards the logs of every Log-trait device owned by one cohort
// node to out for as long as ctx is live. Each such device is identified by name
// (the routing key for its LogApi, and the source label applied to its
// messages), discovered and tracked via the node's device list.
//
// Only devices native to this node are streamed. A gateway — including a hub
// running a gateway system — re-advertises the devices of the nodes it
// aggregates; those same nodes are also in our cohort and streamed via their own
// direct connections, so streaming the aggregated copies too would duplicate
// every line. We therefore skip any device whose name this node does not own,
// mirroring shouldProxyDevice's gateway-mode filtering in announce.go.
func (a *logAggregator) streamNode(ctx context.Context, n *remoteNode, req *logpb.PullLogMessagesRequest, out chan<- *logpb.PullLogMessagesResponse) {
	logger := a.logger.With(zap.String("remoteAddr", n.addr))

	// Track the node's own name so we can tell its native devices from ones it
	// merely aggregates. It's read live (rather than waited on) so we never park
	// on a timer — which would misbehave under synctest — and so we re-filter when
	// the name first arrives or changes.
	self, selfChanges := n.Self.Sub(ctx)
	selfName := self.name
	isNative := func(name string) bool {
		// Until we learn the node's name, stream everything rather than go silent;
		// once the name arrives the loop re-filters and drops any non-native device.
		// Duplicates only arise for gateway nodes, whose name we reliably learn.
		if selfName == "" {
			return true
		}
		return name == selfName || strings.HasPrefix(name, selfName+"/")
	}
	eligible := func(d remoteDesc) bool {
		return hasLogTrait(d.md) &&
			d.name != aggregateLogName && // another gateway's aggregate endpoint; its nodes are streamed directly
			isNative(d.name) // a device aggregated from another node is streamed via its owner instead
	}

	devices, deviceChanges := n.Devices.Sub(ctx)
	// devs is the current device set; active maps a device name to its running
	// stream; seen records every device name we've ever streamed. All are touched
	// only by the loop below, so they stay single-owner. Stream goroutines can't
	// mutate them directly; they post closures to cleanups instead, to be run on
	// the loop goroutine.
	devs := map[string]remoteDesc{}
	active := map[string]*activeStream{}
	seen := map[string]bool{}
	cleanups := make(chan func())
	defer func() {
		for _, as := range active {
			as.stop()
		}
	}()

	post := func(f func()) {
		select {
		case cleanups <- f:
		case <-ctx.Done():
		}
	}

	start := func(d remoteDesc) {
		if _, ok := active[d.name]; ok {
			return // already streaming this device
		}
		name := d.name
		// Replay the backlog only the first time we stream a device. A re-announce
		// (or trait flap) restarts the stream; replaying would duplicate lines, so
		// subsequent runs ask for live updates only, matching runNodeStream's own
		// across-reconnect behaviour.
		updatesOnly := seen[name]
		seen[name] = true
		streamCtx, cancel := context.WithCancel(ctx)
		as := &activeStream{stop: cancel}
		active[name] = as
		go func() {
			a.runNodeStream(streamCtx, n.conn, name, req, updatesOnly, out, logger.With(zap.String("source", name)))
			// The stream stopped for good (e.g. the device doesn't implement the
			// trait). Drop our entry so a later re-announce can start it again,
			// unless a newer stream has already replaced it.
			post(func() {
				if active[name] == as {
					delete(active, name)
				}
			})
		}()
	}

	stop := func(name string) {
		if as, ok := active[name]; ok {
			delete(active, name)
			as.stop()
		}
	}

	// apply reconciles a single device against the current filter: stream it if
	// eligible, otherwise make sure it isn't.
	apply := func(d remoteDesc) {
		devs[d.name] = d
		if eligible(d) {
			start(d)
		} else {
			stop(d.name)
		}
	}

	for _, d := range devices.All {
		apply(d)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case f := <-cleanups:
			f()
		case newSelf := <-selfChanges:
			if newSelf.name == selfName {
				continue
			}
			selfName = newSelf.name
			// Ownership changed: re-filter every known device against the new name.
			for _, d := range devs {
				apply(d)
			}
		case c, ok := <-deviceChanges:
			if !ok {
				return // channel only closes when ctx is done
			}
			switch c.Type {
			case rx.Add, rx.Update:
				// On Update the Log trait may have appeared or disappeared; apply handles both.
				apply(c.New)
			case rx.Remove:
				delete(devs, c.Old.name)
				stop(c.Old.name)
			}
		}
	}
}

// runNodeStream repeatedly opens a PullLogMessages stream against conn (for the
// device named name), reconnecting with backoff until ctx is done. It gives up
// permanently if the node does not implement the Log trait. When updatesOnly is
// set it never requests the initial backlog — used when a previous stream for
// the same device already delivered it, so a restart doesn't replay duplicates.
func (a *logAggregator) runNodeStream(ctx context.Context, conn grpc.ClientConnInterface, name string, req *logpb.PullLogMessagesRequest, updatesOnly bool, out chan<- *logpb.PullLogMessagesResponse, logger *zap.Logger) {
	client := logpb.NewLogApiClient(conn)
	nodeReq := proto.Clone(req).(*logpb.PullLogMessagesRequest)
	nodeReq.Name = name
	if updatesOnly {
		nodeReq.UpdatesOnly = true
		nodeReq.InitialCount = 0
	}

	_ = task.Run(ctx, func(taskCtx context.Context) (task.Next, error) {
		received, err := streamLogsOnce(taskCtx, client, name, nodeReq, out)
		switch {
		case status.Code(err) == codes.Unimplemented:
			logger.Debug("node does not implement the Log trait, not aggregating its logs", zap.Error(err))
			return task.StopNow, err
		case received:
			// We've delivered this device's initial backlog once. A reconnect must
			// not replay it, or the merged stream would show duplicate lines, so ask
			// for live updates only from now on. A brief gap across the reconnect is
			// preferable to a flood of duplicates on every transient disconnect.
			// The stream stayed up long enough to deliver data, so reset the backoff.
			nodeReq.UpdatesOnly = true
			nodeReq.InitialCount = 0
			if taskCtx.Err() == nil {
				logger.Debug("log stream ended, will retry", zap.Error(err))
			}
			return task.ResetBackoff, err
		default:
			if taskCtx.Err() == nil {
				logger.Debug("log stream ended, will retry", zap.Error(err))
			}
			return task.Normal, err
		}
	}, task.WithRetry(task.RetryUnlimited), task.WithBackoff(10*time.Millisecond, 30*time.Second))
}

// streamLogsOnce opens a single PullLogMessages stream and forwards every
// response to out, stamping each message with name. It reports whether it
// received at least one response (so callers can stop requesting the initial
// backlog on reconnect) and returns the error that ended the stream.
func streamLogsOnce(ctx context.Context, client logpb.LogApiClient, name string, req *logpb.PullLogMessagesRequest, out chan<- *logpb.PullLogMessagesResponse) (received bool, err error) {
	stream, err := client.PullLogMessages(ctx, req)
	if err != nil {
		return false, err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			return received, err
		}
		received = true
		stampLogSource(resp, name)
		if err := chans.SendContext(ctx, out, resp); err != nil {
			return received, err
		}
	}
}

// stampLogSource records name as the source node on every message in resp.
func stampLogSource(resp *logpb.PullLogMessagesResponse, name string) {
	for _, change := range resp.Changes {
		for _, msg := range change.Messages {
			msg.SourceNode = name
		}
	}
}

// hasLogTrait reports whether md advertises the Log trait.
func hasLogTrait(md *metadatapb.Metadata) bool {
	for _, t := range md.GetTraits() {
		if t.GetName() == string(logpb.TraitName) {
			return true
		}
	}
	return false
}
