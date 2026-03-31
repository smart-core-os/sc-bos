package fluidflowpb

type InfoServer struct {
	UnimplementedFluidFlowInfoServer
	FluidFlowSupport *FluidFlowSupport
}

func (i *InfoServer) DescribeFluidFlow(_ *DescribeFluidFlowRequest) (*FluidFlowSupport, error) {
	return i.FluidFlowSupport, nil
}
