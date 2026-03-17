package settings

import (
	"context"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

type infoServer struct {
	traits.UnimplementedModeInfoServer
	Modes *traits.ModesSupport
}

func (i *infoServer) DescribeModes(context.Context, *traits.DescribeModesRequest) (*traits.ModesSupport, error) {
	return i.Modes, nil
}
