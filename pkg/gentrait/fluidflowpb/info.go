package fluidflowpb

import "github.com/smart-core-os/sc-bos/pkg/proto/fluidflowpb"

type InfoServer struct {
	fluidflowpb.UnimplementedFluidFlowInfoServer
	FluidFlowSupport *fluidflowpb.FluidFlowSupport
}

func (i *InfoServer) DescribeFluidFlow(_ *fluidflowpb.DescribeFluidFlowRequest) (*fluidflowpb.FluidFlowSupport, error) {
	return i.FluidFlowSupport, nil
}
