package udmi

import (
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// nodeExports announces UdmiExportApi against each node's own name, following the systems
// that expose node-scoped APIs that way (see pkg/system/history, pkg/system/log).
//
// A node normally runs a single udmi automation, but nothing stops it running more — the
// reason to would be publishing to two brokers. Only one server can hold the route for a
// name, so all of a node's automations share one, and its snapshot merges their collectors.
// That way the export covers the whole node however many automations produced it, rather
// than whichever automation registered first.
var nodeExports = &exportRegistry{}

type exportRegistry struct {
	mu     sync.Mutex
	byNode map[*node.Node]*nodeExport
}

type nodeExport struct {
	server *exportServer
	undo   node.Undo
}

// Add registers c as a source of exported points for n, announcing UdmiExportApi against
// n's name if this is the node's first collector. The returned func deregisters c, undoing
// the announcement once the node's last collector goes.
//
// Registering a collector that is already registered for n adds it a second time; pair each
// Add with exactly one call to the func it returns.
func (r *exportRegistry) Add(n *node.Node, c *exportCollector) (remove func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.byNode[n]
	if !ok {
		if r.byNode == nil {
			r.byNode = make(map[*node.Node]*nodeExport)
		}
		server := &exportServer{}
		entry = &nodeExport{
			server: server,
			// The node already announces its own metadata and Parent trait, so this
			// announcement contributes the API alone.
			undo: n.Announce(n.Name(),
				node.HasNoAutoMetadata(),
				node.HasServer(udmipb.RegisterUdmiExportApiServer, udmipb.UdmiExportApiServer(server)),
			),
		}
		r.byNode[n] = entry
	}
	entry.server.addCollector(c)

	return func() { r.remove(n, c) }
}

func (r *exportRegistry) remove(n *node.Node, c *exportCollector) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.byNode[n]
	if !ok {
		return
	}
	if entry.server.removeCollector(c) > 0 {
		return
	}
	entry.undo()
	delete(r.byNode, n)
}
