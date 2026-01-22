package adapt

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/onoffpb"
)

// BinaryObject adapts a binary bacnet object as smart core traits.
func BinaryObject(prefix string, client *gobacnet.Client, device bactypes.Device, object config.Object, deviceHealth *gen_healthpb.FaultCheck, errFn errFn) (node.SelfAnnouncer, error) {
	switch object.Trait {
	case "":
		return nil, ErrNoDefault
	case trait.OnOff:
		model := onoffpb.NewModel()
		return &binaryOnOff{
			prefix: prefix,
			client: client,
			device: device,
			object: object,

			deviceHealth: deviceHealth,
			errFn:        errFn,

			model:       model,
			ModelServer: onoffpb.NewModelServer(model),
		}, nil
	}

	return nil, ErrNoAdaptation
}

type binaryOnOff struct {
	prefix string
	client *gobacnet.Client
	device bactypes.Device
	object config.Object

	deviceHealth *gen_healthpb.FaultCheck
	errFn        errFn

	model *onoffpb.Model
	*onoffpb.ModelServer
}

func (b *binaryOnOff) GetOnOff(ctx context.Context, request *traits.GetOnOffRequest) (*traits.OnOff, error) {
	read, err := b.client.ReadProperty(ctx, b.device, bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: bactypes.ObjectID(b.object.ID),
			Properties: []bactypes.Property{
				{ID: property.PresentValue, ArrayIndex: bactypes.ArrayAll},
			},
		},
	})

	b.errFn(b.deviceHealth, b.name(), "getOnOff", err)
	if err != nil {
		return nil, err
	}

	var value bool
	switch v := read.Object.Properties[0].Data.(type) {
	case bool:
		value = v
	case uint32: // YABE room simulator uses this, not sure if that is "normal"
		value = v == 1
	default:
		return nil, status.Errorf(codes.Internal, "expected bool or uint32 return type for binary value, got %v", v)
	}

	var state traits.OnOff_State
	if value {
		state = traits.OnOff_ON
	} else {
		state = traits.OnOff_OFF
	}
	return b.model.UpdateOnOff(&traits.OnOff{State: state})
}

func (b *binaryOnOff) UpdateOnOff(ctx context.Context, request *traits.UpdateOnOffRequest) (*traits.OnOff, error) {
	return b.UnimplementedOnOffApiServer.UpdateOnOff(ctx, request)
}

func (b *binaryOnOff) PullOnOff(request *traits.PullOnOffRequest, server traits.OnOffApi_PullOnOffServer) error {
	return b.ModelServer.PullOnOff(request, server)
}

func (b *binaryOnOff) AnnounceSelf(a node.Announcer) node.Undo {
	return a.Announce(b.name(),
		node.HasTrait(trait.OnOff, node.WithClients(onoffpb.WrapApi(b))),
	)
}

func (b *binaryOnOff) name() string {
	return b.prefix + ObjectName(b.object)
}
