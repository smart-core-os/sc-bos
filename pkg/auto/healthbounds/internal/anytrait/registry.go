package anytrait

import (
	"context"
	"errors"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/emergencylightpb"
	meterpb "github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

var ErrNotFound = errors.New("not found")

type Resolver struct {
	byName map[trait.Name]Trait
}

func (r *Resolver) FindByName(name trait.Name) (Trait, error) {
	t, ok := r.byName[name]
	if !ok {
		return Trait{}, ErrNotFound
	}
	return t, nil
}

func (r *Resolver) add(name trait.Name, resources ...Resource) {
	r.byName[name] = Trait{
		name:      name,
		resources: resources,
	}
}

var (
	knownTraits     *Resolver
	initKnownTraits = sync.OnceFunc(func() {
		knownTraits = &Resolver{
			byName: make(map[trait.Name]Trait),
		}
		knownTraits.add(trait.AirTemperature, Resource{
			name: "AirTemperature",
			desc: (&traits.AirTemperature{}).ProtoReflect().Descriptor(),
			get:  getter(traits.NewAirTemperatureApiClient, traits.AirTemperatureApiClient.GetAirTemperature),
			pull: puller(traits.NewAirTemperatureApiClient, traits.AirTemperatureApiClient.PullAirTemperature, (*traits.PullAirTemperatureResponse_Change).GetAirTemperature),
		})
		knownTraits.add(emergencylightpb.TraitName, Resource{
			name: "TestResultSet",
			desc: (&gen.TestResultSet{}).ProtoReflect().Descriptor(),
			get:  getter(gen.NewEmergencyLightApiClient, gen.EmergencyLightApiClient.GetTestResultSet),
			pull: puller(gen.NewEmergencyLightApiClient, gen.EmergencyLightApiClient.PullTestResultSets, (*gen.PullTestResultsResponse_Change).GetTestResult),
		})
		knownTraits.add(meterpb.TraitName, Resource{
			name: "MeterReading",
			desc: (&gen.MeterReading{}).ProtoReflect().Descriptor(),
			get:  getter(gen.NewMeterApiClient, gen.MeterApiClient.GetMeterReading),
			pull: puller(gen.NewMeterApiClient, gen.MeterApiClient.PullMeterReadings, (*gen.PullMeterReadingsResponse_Change).GetMeterReading),
		})
		knownTraits.add(trait.OnOff, Resource{
			name: "OnOff",
			desc: (&traits.OnOff{}).ProtoReflect().Descriptor(),
			get:  getter(traits.NewOnOffApiClient, traits.OnOffApiClient.GetOnOff),
			pull: puller(traits.NewOnOffApiClient, traits.OnOffApiClient.PullOnOff, (*traits.PullOnOffResponse_Change).GetOnOff),
		})
	})
)

// FindByName looks up a trait by its name.
// If not found, returns ErrNotFound.
func FindByName(name trait.Name) (Trait, error) {
	initKnownTraits()
	return knownTraits.FindByName(name)
}

// reqPT forces the implementation of proto.Message to also have a pointer receiver.
// The type is intended for use as a constraint in generic functions.
type reqPT[R any] interface {
	*R
	proto.Message
}

// newClient creates a new gRPC client using the given connection.
type newClient[Client any] func(cc grpc.ClientConnInterface) Client

// doGet should call c.GetFoo(ctx, req, opts...), where Foo is the resource name of the trait.
type doGet[Client, ReqPT, Res any] func(c Client, ctx context.Context, req ReqPT, opts ...grpc.CallOption) (Res, error)

// doPull should call c.PullFoo(ctx, req, opts...), where Foo is the resource name of the trait.
type doPull[Client, ReqPT, Res any] func(c Client, ctx context.Context, req ReqPT, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Res], error)

// getVal should call c.GetFoo(), where Foo is the resource name of the trait.
type getVal[Change, V any] func(c Change) V

// getter returns a function that executes the Get verb against a trait resource.
func getter[Client, Req any, Res proto.Message, ReqPT reqPT[Req]](newClient newClient[Client], get doGet[Client, ReqPT, Res]) getFunc {
	pr := ReqPT(new(Req)).ProtoReflect()
	return func(ctx context.Context, conn grpc.ClientConnInterface, r GetRequest) (Value, error) {
		reqMsg := pr.New()
		getReqToProto(reqMsg, r)
		client := newClient(conn)
		resp, err := get(client, ctx, reqMsg.Interface().(ReqPT))
		if err != nil {
			return Value{}, err
		}
		return Value{pb: resp}, nil
	}
}

// pullChange is the common methods of pull response change messages.
type pullChange interface {
	GetChangeTime() *timestamppb.Timestamp
}

// pullResPT represents common pull response methods.
// The pull response type must be a pointer as grpc.ServerStreamingClient returns a pointer to its generic type.
type pullResPT[Res, C any] interface {
	*Res
	GetChanges() []C
}

// puller returns a function that executes the Pull verb against a trait resource.
func puller[Client, Req, Res any, Change pullChange, V proto.Message, ReqPT reqPT[Req], ResPT pullResPT[Res, Change]](newClient newClient[Client], pull doPull[Client, ReqPT, Res], changeVal getVal[Change, V]) pullFunc {
	pr := ReqPT(new(Req)).ProtoReflect()
	return func(ctx context.Context, conn grpc.ClientConnInterface, r PullRequest) (Stream, error) {
		reqMsg := pr.New()
		pullReqToProto(reqMsg, r)
		client := newClient(conn)
		stream, err := pull(client, ctx, reqMsg.Interface().(ReqPT))
		if err != nil {
			return Stream{}, err
		}
		res := Stream{
			recv: func() (PullResponse, error) {
				res, err := stream.Recv()
				if err != nil {
					return PullResponse{}, err
				}
				resPT := ResPT(res)
				resp := PullResponse{}
				for _, change := range resPT.GetChanges() {
					resp.Changes = append(resp.Changes, ValueChange{
						ChangeTime: change.GetChangeTime(),
						Value:      Value{pb: changeVal(change)},
					})
				}
				return resp, nil
			},
		}
		return res, nil
	}
}

func readReqToProto(dst protoreflect.Message, req ReadRequest) {
	if f := dst.Descriptor().Fields().ByName("name"); f != nil && f.Kind() == protoreflect.StringKind {
		dst.Set(f, protoreflect.ValueOfString(req.Name))
	}
	if f := dst.Descriptor().Fields().ByName("read_mask"); f != nil && f.Kind() == protoreflect.MessageKind && f.Message().Name() == "google.protobuf.FieldMask" {
		if req.ReadMask != nil {
			dst.Set(f, protoreflect.ValueOfMessage(req.ReadMask.ProtoReflect()))
		} else {
			dst.Clear(f)
		}
	}
}

func getReqToProto(dst protoreflect.Message, req GetRequest) {
	readReqToProto(dst, req.ReadRequest)
}

func pullReqToProto(dst protoreflect.Message, req PullRequest) {
	readReqToProto(dst, req.ReadRequest)
	if f := dst.Descriptor().Fields().ByName("read_mask"); f != nil && f.Kind() == protoreflect.MessageKind && f.Message().Name() == "google.protobuf.FieldMask" {
		if req.ReadMask != nil {
			dst.Set(f, protoreflect.ValueOfMessage(req.ReadMask.ProtoReflect()))
		} else {
			dst.Clear(f)
		}
	}
}
