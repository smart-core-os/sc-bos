package anytrait

import (
	"context"
	"errors"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/emergencylightpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/soundsensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/temperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
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
		knownTraits.add(trait.AirQualitySensor, Resource{
			name: "AirQualitySensor",
			desc: (&airqualitysensorpb.AirQuality{}).ProtoReflect().Descriptor(),
			get:  getter(airqualitysensorpb.NewAirQualitySensorApiClient, airqualitysensorpb.AirQualitySensorApiClient.GetAirQuality),
			pull: puller(airqualitysensorpb.NewAirQualitySensorApiClient, airqualitysensorpb.AirQualitySensorApiClient.PullAirQuality, (*airqualitysensorpb.PullAirQualityResponse_Change).GetAirQuality),
		})
		knownTraits.add(trait.AirTemperature, Resource{
			name: "AirTemperature",
			desc: (&airtemperaturepb.AirTemperature{}).ProtoReflect().Descriptor(),
			get:  getter(airtemperaturepb.NewAirTemperatureApiClient, airtemperaturepb.AirTemperatureApiClient.GetAirTemperature),
			pull: puller(airtemperaturepb.NewAirTemperatureApiClient, airtemperaturepb.AirTemperatureApiClient.PullAirTemperature, (*airtemperaturepb.PullAirTemperatureResponse_Change).GetAirTemperature),
		})
		knownTraits.add(emergencylightpb.TraitName, Resource{
			name: "TestResultSet",
			desc: (&emergencylightpb.TestResultSet{}).ProtoReflect().Descriptor(),
			get:  getter(emergencylightpb.NewEmergencyLightApiClient, emergencylightpb.EmergencyLightApiClient.GetTestResultSet),
			pull: puller(emergencylightpb.NewEmergencyLightApiClient, emergencylightpb.EmergencyLightApiClient.PullTestResultSets, (*emergencylightpb.PullTestResultsResponse_Change).GetTestResult),
		})
		knownTraits.add(meterpb.TraitName, Resource{
			name: "MeterReading",
			desc: (&meterpb.MeterReading{}).ProtoReflect().Descriptor(),
			get:  getter(meterpb.NewMeterApiClient, meterpb.MeterApiClient.GetMeterReading),
			pull: puller(meterpb.NewMeterApiClient, meterpb.MeterApiClient.PullMeterReadings, (*meterpb.PullMeterReadingsResponse_Change).GetMeterReading),
		})
		knownTraits.add(trait.OnOff, Resource{
			name: "OnOff",
			desc: (&onoffpb.OnOff{}).ProtoReflect().Descriptor(),
			get:  getter(onoffpb.NewOnOffApiClient, onoffpb.OnOffApiClient.GetOnOff),
			pull: puller(onoffpb.NewOnOffApiClient, onoffpb.OnOffApiClient.PullOnOff, (*onoffpb.PullOnOffResponse_Change).GetOnOff),
		})
		knownTraits.add(soundsensorpb.TraitName, Resource{
			name: "SoundLevel",
			desc: (&soundsensorpb.SoundLevel{}).ProtoReflect().Descriptor(),
			get:  getter(soundsensorpb.NewSoundSensorApiClient, soundsensorpb.SoundSensorApiClient.GetSoundLevel),
			pull: puller(soundsensorpb.NewSoundSensorApiClient, soundsensorpb.SoundSensorApiClient.PullSoundLevel, (*soundsensorpb.PullSoundLevelResponse_Change).GetSoundLevel),
		})
		knownTraits.add(temperaturepb.TraitName, Resource{
			name: "Temperature",
			desc: (&temperaturepb.Temperature{}).ProtoReflect().Descriptor(),
			get:  getter(temperaturepb.NewTemperatureApiClient, temperaturepb.TemperatureApiClient.GetTemperature),
			pull: puller(temperaturepb.NewTemperatureApiClient, temperaturepb.TemperatureApiClient.PullTemperature, (*temperaturepb.PullTemperatureResponse_Change).GetTemperature),
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
