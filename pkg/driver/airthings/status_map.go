package airthings

import (
	"context"
	"sync"

	statuspb2 "github.com/smart-core-os/sc-bos/pkg/gentrait/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
)

// statusMap tracks the status of multiple named status Model.
// Any names used to update/delete problems will be announced as Status traits using the given announcer.
type statusMap struct {
	mu        sync.Mutex
	known     map[string]model // keyed by sc name used to announce
	announcer node.Announcer

	watchEvents *minibus.Bus[statusWatchEvent]
}

type statusWatchEvent struct {
	Name string
	Ctx  context.Context
}

type model struct {
	*statuspb2.Model
	unannounce node.Undo
}

func newStatusMap(announcer node.Announcer) *statusMap {
	return &statusMap{
		known:       make(map[string]model),
		announcer:   announcer,
		watchEvents: &minibus.Bus[statusWatchEvent]{},
	}
}

func (m *statusMap) UpdateProblem(name string, problem *statuspb.StatusLog_Problem) {
	m.getOrCreateModel(name).UpdateProblem(problem)
}

func (m *statusMap) DeleteProblem(name, problem string) {
	mod, ok := m.getModel(name)
	if !ok {
		return // nothing to do anyway
	}
	mod.DeleteProblem(problem)
}

func (m *statusMap) Forget(name string) {
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
func (m *statusMap) WatchEvents(ctx context.Context) <-chan statusWatchEvent {
	return m.watchEvents.Listen(ctx)
}

func (m *statusMap) getOrCreateModel(name string) model {
	m.mu.Lock()
	mod, ok := m.known[name]
	if !ok {
		nm := statuspb2.NewModel()
		srv := &watchEventServer{
			ModelServer: statuspb2.NewModelServer(nm),
			m:           m,
		}
		client := statuspb.WrapApi(srv)
		mod = model{
			Model:      nm,
			unannounce: m.announcer.Announce(name, node.HasTrait(statuspb2.TraitName, node.WithClients(client))),
		}
		m.known[name] = mod
	}
	m.mu.Unlock()
	return mod
}

func (m *statusMap) getModel(name string) (model, bool) {
	m.mu.Lock()
	mod, ok := m.known[name]
	m.mu.Unlock()
	return mod, ok
}

type watchEventServer struct {
	*statuspb2.ModelServer
	m *statusMap
}

func (s *watchEventServer) PullCurrentStatus(request *statuspb.PullCurrentStatusRequest, server statuspb.StatusApi_PullCurrentStatusServer) error {
	go s.m.watchEvents.Send(server.Context(), statusWatchEvent{
		Name: request.Name,
		Ctx:  server.Context(),
	})
	return s.ModelServer.PullCurrentStatus(request, server)
}
