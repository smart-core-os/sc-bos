package meterpb

import (
	"context"

	"github.com/smart-core-os/sc-api/go/traits"
)

type InfoServer struct {
	traits.UnimplementedMeterInfoServer
	MeterReading *traits.MeterReadingSupport
}

func (i *InfoServer) DescribeMeterReading(_ context.Context, _ *traits.DescribeMeterReadingRequest) (*traits.MeterReadingSupport, error) {
	return i.MeterReading, nil
}
