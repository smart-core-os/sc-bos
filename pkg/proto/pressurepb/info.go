package pressurepb

type InfoServer struct {
	UnimplementedPressureInfoServer
	PressureSupport *PressureSupport
}

func (i *InfoServer) DescribePressure(_ *DescribePressureRequest) (*PressureSupport, error) {
	return i.PressureSupport, nil
}
