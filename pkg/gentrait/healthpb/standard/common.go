package standard

import (
	"sync"

	"github.com/vanti-dev/sc-bos/pkg/gen"
)

type index struct {
	all           []*gen.HealthCheck_ComplianceImpact_Standard
	byDisplayName map[string]*gen.HealthCheck_ComplianceImpact_Standard
}

func (idx *index) add(s *gen.HealthCheck_ComplianceImpact_Standard) {
	idx.all = append(idx.all, s)
	if v := s.GetDisplayName(); v != "" && idx.byDisplayName != nil {
		idx.byDisplayName[v] = s
	}
}

func (idx *index) FindByDisplayName(name string) *gen.HealthCheck_ComplianceImpact_Standard {
	if idx.byDisplayName == nil {
		idx.byDisplayName = make(map[string]*gen.HealthCheck_ComplianceImpact_Standard, len(idx.all))
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
func FindByDisplayName(name string) *gen.HealthCheck_ComplianceImpact_Standard {
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
func Register(s *gen.HealthCheck_ComplianceImpact_Standard) *gen.HealthCheck_ComplianceImpact_Standard {
	globalMy.Lock()
	defer globalMy.Unlock()
	standards.add(s)
	return s
}
