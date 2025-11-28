package gateway

import (
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/rx"
	"github.com/smart-core-os/sc-bos/pkg/util/slices"
)

// cohort describes the hub and enrolled Nodes.
type cohort struct {
	ignore []string // a list of addresses to ignore in the cohort
	Nodes  *rx.Set[*remoteNode]
}

func newCohort(ignore ...string) *cohort {
	nodeCmp := func(a, b *remoteNode) int { return strings.Compare(a.addr, b.addr) }
	return &cohort{
		ignore: ignore,
		Nodes:  rx.NewSet(slices.NewSortedFunc(nodeCmp)),
	}
}

func (c *cohort) ShouldIgnore(addr string) bool {
	return slices.Contains(addr, c.ignore)
}

// remoteNode describes a remote node enrolled in the cohort.
type remoteNode struct {
	conn  *grpc.ClientConn
	addr  string // the network address for the node, used to identify the node. Will not change after creation.
	isHub bool   // true if the node is the hub

	Self     *rx.Val[remoteDesc]
	Systems  *rx.Val[remoteSystems]
	Services *rx.Set[protoreflect.ServiceDescriptor]
	Devices  *rx.Set[remoteDesc]
}

func newRemoteNode(addr string, conn *grpc.ClientConn) *remoteNode {
	return &remoteNode{
		conn:    conn,
		addr:    addr,
		Self:    rx.NewVal(remoteDesc{}),
		Systems: rx.NewVal(remoteSystems{}),
		Services: rx.NewSet(slices.NewSortedFunc[protoreflect.ServiceDescriptor](func(a, b protoreflect.ServiceDescriptor) int {
			return strings.Compare(string(a.FullName()), string(b.FullName()))
		})),
		Devices: rx.NewSet(slices.NewSortedFunc[remoteDesc](func(a, b remoteDesc) int {
			return strings.Compare(a.name, b.name)
		})),
	}
}

// remoteDesc describes the name and metadata for a remote entity; node or name.
type remoteDesc struct {
	name   string           // the announced name, this is the routing key
	md     *traits.Metadata // used to support the DevicesApi locally
	health []*gen.HealthCheck
}

// remoteSystems describes relevant systems a remote node has.
type remoteSystems struct {
	msgRecvd bool         // true if we've heard from the remote node
	gateway  *gen.Service // a description of the gateway system
}
