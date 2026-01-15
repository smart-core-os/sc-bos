package statuspb

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

// Map tracks the status of multiple named status Model.
// Any names used to update/delete problems will be announced as Status traits using the given announcer.
type Map struct {
	mu        sync.Mutex
	known     map[string]model // keyed by sc name used to announce
	announcer node.Announcer

	watchEvents *minibus.Bus[WatchEvent]
}

type WatchEvent struct {
	Name string
	Ctx  context.Context
}

type model struct {
	*Model
	unannounce node.Undo
}

func NewMap(announcer node.Announcer) *Map {
	return &Map{
		known:       make(map[string]model),
		announcer:   announcer,
		watchEvents: &minibus.Bus[WatchEvent]{},
	}
}

func (m *Map) UpdateProblem(name string, problem *statuspb.StatusLog_Problem) {
	m.getOrCreateModel(name).UpdateProblem(problem)
}

func (m *Map) DeleteProblem(name, problem string) {
	mod, ok := m.getModel(name)
	if !ok {
		return // nothing to do anyway
	}
	mod.DeleteProblem(problem)
}

func (m *Map) Forget(name string) {
	m.mu.Lock()
	mod, ok := m.known[name]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.known, name)
	m.mu.Unlock()
	mod.unannounce()
}

// WatchEvents returns a chan that emits when a client starts pulling the status for a given name.
// The context of the event is the context of the client's request, so will be cancelled when the client disconnects.
func (m *Map) WatchEvents(ctx context.Context) <-chan WatchEvent {
	return m.watchEvents.Listen(ctx)
}

func (m *Map) getOrCreateModel(name string) model {
	m.mu.Lock()
	mod, ok := m.known[name]
	if !ok {
		nm := NewModel()
		srv := &watchEventServer{
			ModelServer: NewModelServer(nm),
			m:           m,
		}
		client := statuspb.WrapApi(srv)
		mod = model{
			Model:      nm,
			unannounce: m.announcer.Announce(name, node.HasTrait(TraitName, node.WithClients(client))),
		}
		m.known[name] = mod
	}
	m.mu.Unlock()
	return mod
}

func (m *Map) getModel(name string) (model, bool) {
	m.mu.Lock()
	mod, ok := m.known[name]
	m.mu.Unlock()
	return mod, ok
}

type watchEventServer struct {
	*ModelServer
	m *Map
}

func (s *watchEventServer) PullCurrentStatus(request *statuspb.PullCurrentStatusRequest, server statuspb.StatusApi_PullCurrentStatusServer) error {
	go s.m.watchEvents.Send(server.Context(), WatchEvent{
		Name: request.Name,
		Ctx:  server.Context(),
	})
	return s.ModelServer.PullCurrentStatus(request, server)
}
