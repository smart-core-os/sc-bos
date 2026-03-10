package meterpb

import (
	"context"
)

type InfoServer struct {
	UnimplementedMeterInfoServer
	MeterReading *MeterReadingSupport
}

func (i *InfoServer) DescribeMeterReading(_ context.Context, _ *DescribeMeterReadingRequest) (*MeterReadingSupport, error) {
	return i.MeterReading, nil
}
