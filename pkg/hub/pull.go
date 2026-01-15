package hub

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
	"github.com/smart-core-os/sc-bos/pkg/util/pull"
)

type Node struct {
	Conn *grpc.ClientConn
	*hubpb.HubNode
}

type Child struct {
	*Node
	*traits.Child
}

type Change[T any] struct {
	Old *T
	New *T
	Err error
}

// PullNodes returns a channel of changes to the nodes in the hub.
// Node.Conn is nil, you should connect using information in Node.HubNode if desired.
func PullNodes(ctx context.Context, conn *grpc.ClientConn) <-chan Change[Node] {
	out := make(chan Change[Node])
	client := hubpb.NewHubApiClient(conn)
	changes := make(chan *hubpb.PullHubNodesResponse_Change)
	go func() {
		defer close(changes)

		// this only returns on ctx cancel or if conn returns Unimplemented for the apis we need.
		// Both cases we're happy to silently stop this routine
		_ = pull.Changes[*hubpb.PullHubNodesResponse_Change](ctx, &NodeFetcher{HubApiClient: client}, changes)
	}()
	go func() {
		defer close(out)
		for change := range changes {
			outChange := Change[Node]{}
			if change.OldValue != nil {
				outChange.Old = &Node{HubNode: change.OldValue}
			}
			if change.NewValue != nil {
				outChange.New = &Node{HubNode: change.NewValue}
			}
			if err := chans.SendContext(ctx, out, outChange); err != nil {
				return
			}
		}
	}()
	return out
}

// PullChildren returns a channel of changes to the children of all nodes enrolled in the hub conn.
// The conn argument should be a connection to the hub.
// For each node a new connection (via grpc.NewClient) will be made using the nodes address and the given dialOpts.
// Children sent via the returned channel will have their Conn field set to the new connection which can be used so long
// as that child is still a member of the hub cohort.
func PullChildren(ctx context.Context, conn *grpc.ClientConn, dialOpts ...grpc.DialOption) <-chan Change[Child] {
	out := make(chan Change[Child])
	go func() {
		defer close(out)
		nodes := activeNodes{}
		for nodeChange := range PullNodes(ctx, conn) {
			if nodeChange.Err != nil {
				if err := chans.SendContext(ctx, out, Change[Child]{Err: nodeChange.Err}); err != nil {
					return
				}
				continue
			}

			switch {
			case nodeChange.Old == nil && nodeChange.New == nil: // shouldn't happen, but just in case
			case nodeChange.New == nil:
				nodes.remove(nodeChange.Old)
			case nodes.exists(nodeChange.New):
				nodes.update(nodeChange.New)
			default: // add
				nodes.add(ctx, out, nodeChange.New, dialOpts...)
			}
		}
	}()
	return out
}

type activeNodes map[string]*activeNode

func (a activeNodes) exists(node *Node) bool {
	_, ok := a[node.Name]
	return ok
}

func (a activeNodes) remove(node *Node) {
	if n, ok := a[node.Name]; ok {
		n.stop()
		delete(a, node.Name)
	}
}

func (a activeNodes) update(node *Node) {
	n, ok := a[node.Name]
	if !ok {
		panic("calling update before checking exists")
	}
	n.update(node)
}

func (a activeNodes) add(ctx context.Context, out chan Change[Child], n *Node, dialOpts ...grpc.DialOption) {
	if _, ok := a[n.Name]; ok {
		panic("calling add before checking exists")
	}

	nodeCtx, stopper := context.WithCancel(ctx)
	addrUpdate := make(chan string)
	remote := node.DialChan(nodeCtx, addrUpdate, dialOpts...)
	stopper = func() {
		remote.Close()
		stopper()
		close(addrUpdate)
	}
	conn, err := remote.Connect(nodeCtx)
	if err != nil {
		// this error should not be related to connecting to the node itself, but rather to the dial options.
		if err != nil {
			if err := chans.SendContext(ctx, out, Change[Child]{Err: err}); err != nil {
				return
			}
		}
	}
	a[n.Name] = &activeNode{
		stop:       stopper,
		addrUpdate: addrUpdate,
		conn:       conn,
	}

	parentClient := traits.NewParentApiClient(conn)
	childChanges := make(chan *traits.PullChildrenResponse_Change)
	go func() {
		defer close(childChanges)
		fetcher := &childFetcher{
			ParentApiClient: parentClient,
			name:            n.Name,
		}
		// The error here is either because ctx is canceled or because the node doesn't support the parent trait.
		// In either case there's not much we can do.
		// Connection errors are retired by pull.Changes.
		_ = pull.Changes[*traits.PullChildrenResponse_Change](nodeCtx, fetcher, childChanges)
	}()
	go func() {
		for change := range childChanges {
			outChange := Change[Child]{}
			if change.OldValue != nil {
				outChange.Old = &Child{Node: n, Child: change.OldValue}
			}
			if change.NewValue != nil {
				outChange.New = &Child{Node: n, Child: change.NewValue}
			}
			if err := chans.SendContext(ctx, out, outChange); err != nil {
				return
			}
		}
	}()
}

type activeNode struct {
	stop       func()
	addrUpdate chan<- string
	conn       *grpc.ClientConn
}

func (a *activeNode) update(n *Node) {
	a.addrUpdate <- n.Address
}

type NodeFetcher struct {
	hubpb.HubApiClient
	known map[string]*hubpb.HubNode // in case of polling, this tracks seen nodes so we correctly send changes
}

func (c *NodeFetcher) Pull(ctx context.Context, changes chan<- *hubpb.PullHubNodesResponse_Change) error {
	stream, err := c.PullHubNodes(ctx, &hubpb.PullHubNodesRequest{})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			if err := chans.SendContext(ctx, changes, change); err != nil {
				return err
			}
		}
	}
}

func (c *NodeFetcher) Poll(ctx context.Context, changes chan<- *hubpb.PullHubNodesResponse_Change) error {
	nodes, err := c.ListHubNodes(ctx, &hubpb.ListHubNodesRequest{})
	if err != nil {
		return err
	}
	if c.known == nil {
		c.known = make(map[string]*hubpb.HubNode)
	}
	unseen := make(map[string]struct{}, len(c.known))
	for s := range c.known {
		unseen[s] = struct{}{}
	}

	for _, node := range nodes.Nodes {
		// we do extra work here to try and send out more accurate changes to make callers lives easier
		change := &hubpb.PullHubNodesResponse_Change{
			Type:     types.ChangeType_ADD,
			NewValue: node,
		}
		if old, ok := c.known[node.Name]; ok {
			change.Type = types.ChangeType_UPDATE
			change.OldValue = old
			delete(unseen, node.Name)
		}
		if proto.Equal(change.OldValue, change.NewValue) {
			continue
		}

		c.known[node.Name] = node
		if err := chans.SendContext(ctx, changes, change); err != nil {
			return err
		}
	}

	for name := range unseen {
		node := c.known[name]
		delete(c.known, name)
		change := &hubpb.PullHubNodesResponse_Change{
			Type:     types.ChangeType_REMOVE,
			OldValue: node,
		}
		if err := chans.SendContext(ctx, changes, change); err != nil {
			return err
		}
	}
	return nil
}

type childFetcher struct {
	traits.ParentApiClient
	name  string
	known map[string]*traits.Child // in case of polling, this tracks seen children so we correctly send changes
}

func (c *childFetcher) Pull(ctx context.Context, changes chan<- *traits.PullChildrenResponse_Change) error {
	stream, err := c.PullChildren(ctx, &traits.PullChildrenRequest{Name: c.name})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			if err := chans.SendContext(ctx, changes, change); err != nil {
				return err
			}
		}
	}
}

func (c *childFetcher) Poll(ctx context.Context, changes chan<- *traits.PullChildrenResponse_Change) error {
	if c.known == nil {
		c.known = make(map[string]*traits.Child)
	}
	unseen := make(map[string]struct{}, len(c.known))
	for s := range c.known {
		unseen[s] = struct{}{}
	}

	req := &traits.ListChildrenRequest{Name: c.name, PageSize: 1000}
	for {
		res, err := c.ListChildren(ctx, req)
		if err != nil {
			return err
		}

		for _, node := range res.Children {
			// we do extra work here to try and send out more accurate changes to make callers lives easier
			change := &traits.PullChildrenResponse_Change{
				Type:     types.ChangeType_ADD,
				NewValue: node,
			}
			if old, ok := c.known[node.Name]; ok {
				change.Type = types.ChangeType_UPDATE
				change.OldValue = old
				delete(unseen, node.Name)
			}
			if proto.Equal(change.OldValue, change.NewValue) {
				continue
			}

			c.known[node.Name] = node
			if err := chans.SendContext(ctx, changes, change); err != nil {
				return err
			}
		}

		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	for name := range unseen {
		node := c.known[name]
		delete(c.known, name)
		change := &traits.PullChildrenResponse_Change{
			Type:     types.ChangeType_REMOVE,
			OldValue: node,
		}
		if err := chans.SendContext(ctx, changes, change); err != nil {
			return err
		}
	}
	return nil
}
