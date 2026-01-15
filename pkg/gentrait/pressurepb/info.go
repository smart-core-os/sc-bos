package pressurepb

import "github.com/smart-core-os/sc-bos/pkg/proto/pressurepb"

type InfoServer struct {
	pressurepb.UnimplementedPressureInfoServer
	PressureSupport *pressurepb.PressureSupport
}

func (i *InfoServer) DescribePressure(_ *pressurepb.DescribePressureRequest) (*pressurepb.PressureSupport, error) {
	return i.PressureSupport, nil
}
