package transportpb

import (
	"context"
)

type InfoServer struct {
	UnimplementedTransportInfoServer
	Transport *TransportSupport
}

func (i *InfoServer) DescribeTransport(_ context.Context, _ *DescribeTransportRequest) (*TransportSupport, error) {
	return i.Transport, nil
}
