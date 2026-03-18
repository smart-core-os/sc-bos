package modepb

import (
	"time"

	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
)

// tOrST returns t if m.StartTime is nil, or m.StartTime.AsTime() otherwise.
func tOrST(t time.Time, m *electricpb.ElectricMode) time.Time {
	if m.GetStartTime() == nil {
		return t
	}
	return m.GetStartTime().AsTime()
}
