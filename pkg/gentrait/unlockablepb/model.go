package unlockablepb

import (
	"github.com/smart-core-os/sc-bos/pkg/gen"
)

type Model struct {
	unlockableBanks []*gen.UnlockableBank
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) AddUnlockableBank(bank *gen.UnlockableBank) {
	m.unlockableBanks = append(m.unlockableBanks, bank)
}
