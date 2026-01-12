package meter

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

type InfoServer struct {
	meterpb.UnimplementedMeterInfoServer
	MeterReading *meterpb.MeterReadingSupport
}

func (i *InfoServer) DescribeMeterReading(_ context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	return i.MeterReading, nil
}
