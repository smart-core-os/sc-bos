package modepb

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
)

type InfoServer struct {
	traits.UnimplementedModeInfoServer
	Modes *traits.ModesSupport
}

func (i *InfoServer) DescribeModes(context.Context, *traits.DescribeModesRequest) (*traits.ModesSupport, error) {
	return i.Modes, nil
}
