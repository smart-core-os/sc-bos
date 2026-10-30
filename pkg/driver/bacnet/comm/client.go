package comm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
)

// Client wraps a gobacnet.Client, capping how long each confirmed request may take.
//
// gobacnet waits for a response until the request ctx is done.
// It doesn't notify the waiter when a device answers with an Abort or Reject, and it has no timeout of its own,
// so a request whose ctx has no deadline can block forever while holding one of the client's shared transaction slots.
// Client bounds each request by RequestTimeout, so the effective deadline is the earlier of the caller's and the cap.
// Every exported gobacnet method that sends a confirmed request is overridden here,
// add an override for any new ones or they will bypass the cap.
type Client struct {
	*gobacnet.Client
	RequestTimeout time.Duration
}

// NewClient returns a Client that caps each request made via client to requestTimeout.
func NewClient(client *gobacnet.Client, requestTimeout time.Duration) *Client {
	return &Client{Client: client, RequestTimeout: requestTimeout}
}

// RequestTimeoutError is returned when a request is cut short by Client.RequestTimeout
// while the caller's ctx was still live.
//
// It deliberately doesn't match context.DeadlineExceeded via errors.Is:
// callers treat a ctx error as "my ctx is done, stop", which is not the case here.
// It converts to a gRPC status with codes.DeadlineExceeded.
type RequestTimeoutError struct {
	Timeout time.Duration
	Err     error // the error reported by gobacnet, for detail only
}

func (e *RequestTimeoutError) Error() string {
	return fmt.Sprintf("bacnet request timed out after %s: %v", e.Timeout, e.Err)
}

func (e *RequestTimeoutError) GRPCStatus() *status.Status {
	return status.New(codes.DeadlineExceeded, e.Error())
}

// IsRequestTimeout reports whether err contains a RequestTimeoutError.
func IsRequestTimeout(err error) bool {
	var timeoutErr *RequestTimeoutError
	return errors.As(err, &timeoutErr)
}

func (c *Client) ReadProperty(ctx context.Context, dest bactypes.Device, rp bactypes.ReadPropertyData) (bactypes.ReadPropertyData, error) {
	return capped(c, ctx, func(ctx context.Context) (bactypes.ReadPropertyData, error) {
		return c.Client.ReadProperty(ctx, dest, rp)
	})
}

func (c *Client) ReadMultiProperty(ctx context.Context, dev bactypes.Device, rp bactypes.ReadMultipleProperty) (bactypes.ReadMultipleProperty, error) {
	return capped(c, ctx, func(ctx context.Context) (bactypes.ReadMultipleProperty, error) {
		return c.Client.ReadMultiProperty(ctx, dev, rp)
	})
}

func (c *Client) ReadProperties(ctx context.Context, dev bactypes.Device, rp bactypes.ReadMultipleProperty) (bactypes.ReadMultipleProperty, error) {
	return capped(c, ctx, func(ctx context.Context) (bactypes.ReadMultipleProperty, error) {
		return c.Client.ReadProperties(ctx, dev, rp)
	})
}

func (c *Client) WriteProperty(ctx context.Context, dest bactypes.Device, wp bactypes.ReadPropertyData, priority uint) error {
	_, err := capped(c, ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.Client.WriteProperty(ctx, dest, wp, priority)
	})
	return err
}

func (c *Client) RemoteDevices(ctx context.Context, addr bactypes.Address, ids ...bactypes.ObjectInstance) ([]bactypes.Device, error) {
	return capped(c, ctx, func(ctx context.Context) ([]bactypes.Device, error) {
		return c.Client.RemoteDevices(ctx, addr, ids...)
	})
}

// Objects caps the whole object list walk, which is many requests, not each request within it.
func (c *Client) Objects(ctx context.Context, dev bactypes.Device) (bactypes.Device, error) {
	return capped(c, ctx, func(ctx context.Context) (bactypes.Device, error) {
		return c.Client.Objects(ctx, dev)
	})
}

func capped[T any](c *Client, ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	if c.RequestTimeout <= 0 {
		return fn(ctx)
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.RequestTimeout)
	defer cancel()
	res, err := fn(reqCtx)
	if err != nil && ctx.Err() == nil && reqCtx.Err() != nil {
		// gobacnet errors often don't wrap the ctx error, so judge by the ctx rather than err
		return res, &RequestTimeoutError{Timeout: c.RequestTimeout, Err: err}
	}
	return res, err
}
