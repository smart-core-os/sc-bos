package proxy

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

// deviceFetcher implements pull.Fetcher to pull or poll devices from a client
type deviceFetcher struct {
	client devicespb.DevicesApiClient
	known  map[string]*devicespb.Device // in case of polling, this tracks seen devices so we correctly send changes
}

func (c *deviceFetcher) Pull(ctx context.Context, changes chan<- *devicespb.PullDevicesResponse_Change) error {
	stream, err := c.client.PullDevices(ctx, &devicespb.PullDevicesRequest{
		ReadMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "metadata"},
		},
	})
	if err != nil {
		return err
	}
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		for _, change := range msg.Changes {
			if err := chans.SendContext(ctx, changes, change); err != nil {
				return err
			}
		}
	}
}

func (c *deviceFetcher) Poll(ctx context.Context, changes chan<- *devicespb.PullDevicesResponse_Change) error {
	if c.known == nil {
		c.known = make(map[string]*devicespb.Device)
	}
	unseen := make(map[string]struct{}, len(c.known))
	for s := range c.known {
		unseen[s] = struct{}{}
	}

	req := &devicespb.ListDevicesRequest{
		ReadMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "metadata"},
		},
	}
	for {
		res, err := c.client.ListDevices(ctx, req)
		if err != nil {
			return err
		}

		for _, node := range res.Devices {
			// we do extra work here to try and send out more accurate changes to make callers lives easier
			change := &devicespb.PullDevicesResponse_Change{
				Type:     types.ChangeType_ADD,
				NewValue: node,
			}
			if old, ok := c.known[node.Name]; ok {
				change.Type = types.ChangeType_UPDATE
				change.OldValue = old
				delete(unseen, node.Name)
			}
			if proto.Equal(change.OldValue, change.NewValue) {
				continue
			}

			c.known[node.Name] = node
			if err := chans.SendContext(ctx, changes, change); err != nil {
				return err
			}
		}

		req.PageToken = res.NextPageToken
		if req.PageToken == "" {
			break
		}
	}

	for name := range unseen {
		node := c.known[name]
		delete(c.known, name)
		change := &devicespb.PullDevicesResponse_Change{
			Type:     types.ChangeType_REMOVE,
			OldValue: node,
		}
		if err := chans.SendContext(ctx, changes, change); err != nil {
			return err
		}
	}
	return nil
}
