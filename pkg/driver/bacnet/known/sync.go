package known

import (
	"sync"

	bactypes "github.com/smart-core-os/gobacnet/types"
)

// SyncContext returns a Context that is safe for concurrent use.
// If the Context allows mutation, maybe because it is really a Map then mu should be used to protect the mutation.
func SyncContext(mu sync.Locker, ctx Context) Context {
	return &syncMap{impl: ctx, mu: mu}
}

type syncMap struct {
	impl Context
	mu   sync.Locker
}

func (s *syncMap) ListObjects(device bactypes.Device) ([]bactypes.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.ListObjects(device)
}

func (s *syncMap) LookupDeviceByID(id bactypes.ObjectInstance) (bactypes.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.LookupDeviceByID(id)
}

func (s *syncMap) LookupDeviceByName(name string) (bactypes.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.LookupDeviceByName(name)
}

func (s *syncMap) LookupObjectByID(device bactypes.Device, id bactypes.ObjectID) (bactypes.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.LookupObjectByID(device, id)
}

func (s *syncMap) LookupObjectByName(device bactypes.Device, name string) (bactypes.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.LookupObjectByName(device, name)
}

func (s *syncMap) GetDeviceDefaultWritePriority(id bactypes.ObjectInstance) uint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.impl.GetDeviceDefaultWritePriority(id)
}
