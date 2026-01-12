package standard

import (
	"sync"

	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

type index struct {
	all           []*healthpb.HealthCheck_ComplianceImpact_Standard
	byDisplayName map[string]*healthpb.HealthCheck_ComplianceImpact_Standard
}

func (idx *index) add(s *healthpb.HealthCheck_ComplianceImpact_Standard) {
	idx.all = append(idx.all, s)
	if v := s.GetDisplayName(); v != "" && idx.byDisplayName != nil {
		idx.byDisplayName[v] = s
	}
}

func (idx *index) FindByDisplayName(name string) *healthpb.HealthCheck_ComplianceImpact_Standard {
	if idx.byDisplayName == nil {
		idx.byDisplayName = make(map[string]*healthpb.HealthCheck_ComplianceImpact_Standard, len(idx.all))
		for _, s := range idx.all {
			if v := s.GetDisplayName(); v != "" {
				idx.byDisplayName[v] = s
			}
		}
	}
	return idx.byDisplayName[name]
}

var (
	globalMy  sync.Mutex
	standards = new(index)
)

// FindByDisplayName looks up a standard by its display name.
// If not found, returns nil.
func FindByDisplayName(name string) *healthpb.HealthCheck_ComplianceImpact_Standard {
	if name == "" {
		return nil
	}
	globalMy.Lock()
	defer globalMy.Unlock()
	return standards.FindByDisplayName(name)
}

// Register registers a standard.
// If a standard with the same display name already exists, it is overwritten.
// Returns s.
func Register(s *healthpb.HealthCheck_ComplianceImpact_Standard) *healthpb.HealthCheck_ComplianceImpact_Standard {
	globalMy.Lock()
	defer globalMy.Unlock()
	standards.add(s)
	return s
}
