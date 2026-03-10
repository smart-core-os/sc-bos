package client

import (
	"context"
	"net/url"
	"time"

	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/smart-core-os/sc-api/go/info"
)

type Client struct {
	conn        *grpc.ClientConn
	retryPolicy []grpc.CallOption
	logger      *zap.Logger
}

func NewClient(processorAddr *url.URL, auth AuthCredentials, logger *zap.Logger) Client {
	// connect to processor
	conn, err := grpc.Dial(
		processorAddr.Host,
		grpc.WithTransportCredentials(auth.Creds),
		grpc.WithUnaryInterceptor(grpcretry.UnaryClientInterceptor()),
	)
	if err != nil {
		logger.Fatal("Could not create connection to server", zap.Error(err))
	}

	return Client{
		conn,
		[]grpc.CallOption{
			grpcretry.WithMax(5),
			grpcretry.WithPerRetryTimeout(2 * time.Second),
			grpcretry.WithBackoff(grpcretry.BackoffExponentialWithJitter(100*time.Millisecond, 0.01)),
		},
		logger,
	}
}

// Shutdown this connection to the client
func (c *Client) Shutdown() error {
	return c.conn.Close()
}

// Info service functions
func (c *Client) GetDeviceList(ctx context.Context) ([]*info.Device, error) {
	i := info.NewInfoClient(c.conn)
	resp, err := i.ListDevices(ctx, &info.ListDevicesRequest{})
	if err != nil {
		return nil, err
	}
	devices := resp.Devices
	// loop through to get all known devices for this controller
	for resp.NextPageToken != "" {
		resp, err = i.ListDevices(ctx, &info.ListDevicesRequest{PageToken: resp.NextPageToken})
		if err != nil {
			return nil, err
		}
		devices = append(devices, resp.Devices...)
	}

	return devices, nil
}

/*func (c *Client) Devices(ctx context.Context) (<- chan *info.Device, error) {

}*/

// Health service functions
