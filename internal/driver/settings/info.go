package settings

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/modepb"
)

type infoServer struct {
	modepb.UnimplementedModeInfoServer
	Modes *modepb.ModesSupport
}

func (i *infoServer) DescribeModes(context.Context, *modepb.DescribeModesRequest) (*modepb.ModesSupport, error) {
	return i.Modes, nil
}
