package modepb

import (
	"context"
)

type InfoServer struct {
	UnimplementedModeInfoServer
	Modes *ModesSupport
}

func (i *InfoServer) DescribeModes(context.Context, *DescribeModesRequest) (*ModesSupport, error) {
	return i.Modes, nil
}
