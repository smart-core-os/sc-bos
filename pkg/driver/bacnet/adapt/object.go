package adapt

import (
	"fmt"

	"github.com/smart-core-os/gobacnet"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/rpc"
	gen_healthpb "github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/node"
)

type errFn func(health *gen_healthpb.FaultCheck, name, request string, err error)

// Object adapts a bacnet object into one or more smart core named traits.
func Object(prefix string, client *gobacnet.Client, device bactypes.Device, object config.Object, deviceHealth *gen_healthpb.FaultCheck, errFn errFn) (node.SelfAnnouncer, error) {
	switch object.ID.Type {
	case objecttype.BinaryValue, objecttype.BinaryOutput, objecttype.BinaryInput:
		return BinaryObject(prefix, client, device, object, deviceHealth, errFn)
	}

	if object.Trait == "" {
		return nil, ErrNoDefault
	}
	return nil, ErrNoAdaptation
}

// DeviceName returns the smart core name we should use for the configured object.
func DeviceName(o config.Device) string {
	if o.Name != "" {
		return o.Name
	}
	return fmt.Sprintf("%d", o.ID)
}

// ObjectName returns the smart core name we should use for the configured object.
func ObjectName(o config.Object) string {
	if o.Name != "" {
		return o.Name
	}
	return o.ID.String()
}

func ObjectIDFromProto(identifier *rpc.ObjectIdentifier) bactypes.ObjectID {
	return bactypes.ObjectID{
		Type:     objecttype.ObjectType(identifier.Type),
		Instance: bactypes.ObjectInstance(identifier.Instance),
	}
}

func ObjectIDToProto(id bactypes.ObjectID) *rpc.ObjectIdentifier {
	return &rpc.ObjectIdentifier{
		Type:     uint32(id.Type),
		Instance: uint32(id.Instance),
	}
}
