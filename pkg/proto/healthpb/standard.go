package healthpb

import (
	"sync"
)

type stdIndex struct {
	all           []*HealthCheck_ComplianceImpact_Standard
	byDisplayName map[string]*HealthCheck_ComplianceImpact_Standard
}

func (idx *stdIndex) add(s *HealthCheck_ComplianceImpact_Standard) {
	idx.all = append(idx.all, s)
	if v := s.GetDisplayName(); v != "" && idx.byDisplayName != nil {
		idx.byDisplayName[v] = s
	}
}

func (idx *stdIndex) FindByDisplayName(name string) *HealthCheck_ComplianceImpact_Standard {
	if idx.byDisplayName == nil {
		idx.byDisplayName = make(map[string]*HealthCheck_ComplianceImpact_Standard, len(idx.all))
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
	standards = new(stdIndex)
)

// FindStandardByDisplayName looks up a standard by its display name.
// If not found, returns nil.
func FindStandardByDisplayName(name string) *HealthCheck_ComplianceImpact_Standard {
	if name == "" {
		return nil
	}
	globalMy.Lock()
	defer globalMy.Unlock()
	return standards.FindByDisplayName(name)
}

// RegisterStandard registers a standard.
// If a standard with the same display name already exists, it is overwritten.
// Returns s.
func RegisterStandard(s *HealthCheck_ComplianceImpact_Standard) *HealthCheck_ComplianceImpact_Standard {
	globalMy.Lock()
	defer globalMy.Unlock()
	standards.add(s)
	return s
}
