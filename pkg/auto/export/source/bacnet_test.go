package source

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
	"github.com/smart-core-os/sc-bos/pkg/auto"
	"github.com/smart-core-os/sc-bos/pkg/auto/export/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/adapt"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/rpc"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

func TestBacnet_PublishAll(t *testing.T) {
	pub := &testPublisher{}
	b := &bacnet{
		services: Services{
			Services: auto.Services{
				Logger: zap.NewNop(),
			},
			Publisher: pub,
		},
		Lifecycle: task.NewLifecycle[config.BacnetSource](nil),
	}

	server := &testBacnetApiServer{
		Map: known.NewMap(),
	}
	device := bactypes.Device{ID: bactypes.ObjectID{Type: objecttype.Device, Instance: 12}}
	server.StoreDevice("dev01", device, 0)
	server.StoreObject(device, "obj01", bactypes.Object{
		ID: bactypes.ObjectID{Type: objecttype.AnalogValue, Instance: 1},
		Properties: []bactypes.Property{
			{ID: property.PresentValue, ArrayIndex: bactypes.ArrayAll, Data: float32(1.2)},
		},
	})
	cfg := config.BacnetSource{
		Devices: []config.BacnetDevice{
			{Name: "dev01"},
		},
	}
	err := b.publishAll(context.Background(), cfg, rpc.WrapBacnetDriverService(server), nil)
	if err != nil {
		t.Fatal(err)
	}

	want := &testPublisher{
		{Topic: "dev01/obj/AnalogValue:1/prop/PresentValue", Payload: `{"real":1.2}`},
	}
	if diff := cmp.Diff(want, pub); diff != "" {
		t.Fatalf("published messages (-want,+got)\n%s", diff)
	}

}

type testPublisher []publication

func (t *testPublisher) Publish(ctx context.Context, topic string, payload any) error {
	*t = append(*t, publication{topic, payload})
	return nil
}

type publication struct {
	Topic   string
	Payload any
}

type testBacnetApiServer struct {
	rpc.UnimplementedBacnetDriverServiceServer

	*known.Map
}

func (t *testBacnetApiServer) ReadProperty(ctx context.Context, request *rpc.ReadPropertyRequest) (*rpc.ReadPropertyResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (t *testBacnetApiServer) ReadPropertyMultiple(ctx context.Context, request *rpc.ReadPropertyMultipleRequest) (*rpc.ReadPropertyMultipleResponse, error) {
	device, err := t.LookupDeviceByName(request.Name)
	if err != nil {
		return nil, err
	}
	res := &rpc.ReadPropertyMultipleResponse{}
	for _, specification := range request.ReadSpecifications {
		object, err := t.LookupObjectByID(device, adapt.ObjectIDFromProto(specification.ObjectIdentifier))
		if err != nil {
			return nil, err
		}
		if len(specification.PropertyReferences) == 0 {
			specification.PropertyReferences = []*rpc.PropertyReference{
				{Identifier: uint32(property.PresentValue)},
			}
		}

		objRes := &rpc.ReadPropertyMultipleResponse_ReadResult{
			ObjectIdentifier: adapt.ObjectIDToProto(object.ID),
		}
	found:
		for _, reference := range specification.PropertyReferences {
			for _, p := range object.Properties {
				if p.ID == property.ID(reference.Identifier) {
					rr, err := adapt.PropertyToProtoReadResult(p)
					if err != nil {
						return nil, err
					}
					objRes.Results = append(objRes.Results, rr)
					break found
				}
			}

			// property not found
			return nil, fmt.Errorf("%v.%v does not have property %v", request.Name, object.ID, property.ID(reference.Identifier))
		}
		res.ReadResults = append(res.ReadResults, objRes)
	}
	return res, nil
}

func (t *testBacnetApiServer) ListObjects(_ context.Context, request *rpc.ListObjectsRequest) (*rpc.ListObjectsResponse, error) {
	device, err := t.LookupDeviceByName(request.Name)
	if err != nil {
		return nil, err
	}
	objects, err := t.Map.ListObjects(device)
	if err != nil {
		return nil, err
	}
	res := &rpc.ListObjectsResponse{}
	for _, object := range objects {
		res.Objects = append(res.Objects, adapt.ObjectIDToProto(object.ID))
	}
	return res, nil
}
