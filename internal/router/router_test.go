package router

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	occupancysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/wrap"
	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

// tests overall type behavior: registering services, adding routes, and routing requests with correct priority.
func TestRouter(t *testing.T) {
	r := New()
	check(t, r.AddService(routedRegistryService(t, traits.OnOffApi_ServiceDesc.ServiceName, "name")))
	check(t, r.AddService(routedRegistryService(t, traits.OccupancySensorApi_ServiceDesc.ServiceName, "name")))
	check(t, r.AddService(routedRegistryService(t, traits.AirQualitySensorApi_ServiceDesc.ServiceName, "name")))

	fooModel := onoffpb.NewModel(resource.WithInitialValue(&traits.OnOff{State: traits.OnOff_OFF}))
	defaultModel := onoffpb.NewModel(resource.WithInitialValue(&traits.OnOff{State: traits.OnOff_ON}))
	occupancyModel := occupancysensorpb2.NewModel(resource.WithInitialValue(&traits.Occupancy{State: traits.Occupancy_OCCUPIED}))

	// register a specific route for "foo"
	check(t, r.AddRoute("", "foo",
		wrap.ServerToClient(traits.OnOffApi_ServiceDesc, onoffpb.NewModelServer(fooModel))))
	// register a specific route for "foo" for the occupancy service - this should have higher priority
	check(t, r.AddRoute(traits.OccupancySensorApi_ServiceDesc.ServiceName, "foo",
		wrap.ServerToClient(traits.OccupancySensorApi_ServiceDesc, occupancysensorpb2.NewModelServer(occupancyModel))))
	// add a catch-all for all OnOffApi requests that are not to "foo"
	check(t, r.AddRoute(traits.OnOffApi_ServiceDesc.ServiceName, "",
		wrap.ServerToClient(traits.OnOffApi_ServiceDesc, onoffpb.NewModelServer(defaultModel))))

	conn := NewLoopback(r)
	onOffClient := traits.NewOnOffApiClient(conn)
	occupancyClient := traits.NewOccupancySensorApiClient(conn)
	airQualityClient := traits.NewAirQualitySensorApiClient(conn)
	modeClient := traits.NewModeApiClient(conn)
	// "foo" should route to the fooModel
	res, err := onOffClient.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "foo"})
	if err != nil {
		t.Errorf("failed to get onoff for foo: %v", err)
	} else if res.State != traits.OnOff_OFF {
		t.Errorf("expected OFF for foo, got %v", res.State)
	}
	// "bar" (or anything that's not "foo") should route to the defaultModel
	res, err = onOffClient.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "bar"})
	if err != nil {
		t.Errorf("failed to get onoff for bar: %v", err)
	} else if res.State != traits.OnOff_ON {
		t.Errorf("expected ON for bar, got %v", res.State)
	}
	// "foo" for the occupancy service should route to the occupancyModel
	res2, err := occupancyClient.GetOccupancy(context.Background(), &traits.GetOccupancyRequest{Name: "foo"})
	if err != nil {
		t.Errorf("failed to get occupancy for foo: %v", err)
	} else if res2.State != traits.Occupancy_OCCUPIED {
		t.Errorf("expected OCCUPIED for foo, got %v", res2.State)
	}
	// "bar" for the occupancy service should fail to resolve
	_, err = occupancyClient.GetOccupancy(context.Background(), &traits.GetOccupancyRequest{Name: "bar"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound for bar, got %v", err)
	}
	// there are no matching routes registered for the air quality service on device "bar", so it should fail to resolve
	_, err = airQualityClient.GetAirQuality(context.Background(), &traits.GetAirQualityRequest{Name: "bar"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound for air quality, got %v", err)
	}
	// the mode service isn't registered on the router so this should fail, even though there is an all-service route
	// for "foo"
	_, err = modeClient.GetModeValues(context.Background(), &traits.GetModeValuesRequest{Name: "foo"})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented for mode, got %v", err)
	}

}

func TestRouter_AddService(t *testing.T) {
	r := New()
	model := onoffpb.NewModel(resource.WithInitialValue(&traits.OnOff{State: traits.OnOff_OFF}))
	client := traits.NewOnOffApiClient(NewLoopback(r))

	// add a device route
	err := r.AddRoute("", "foo", wrap.ServerToClient(traits.OnOffApi_ServiceDesc, onoffpb.NewModelServer(model)))
	if err != nil {
		t.Fatalf("(1) failed to add route: %v", err)
	}

	// route shouldn't match because the service hasn't been added yet
	_, err = client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "foo"})
	if statusErr, _ := status.FromError(err); statusErr.Code() != codes.Unimplemented {
		t.Errorf("(2) expected Unimplemented, got %v", statusErr)
	}

	// service should not be registered, then after we add it, it should be
	srvName := traits.OnOffApi_ServiceDesc.ServiceName
	srv := r.GetService(srvName)
	if srv != nil {
		t.Errorf("(3) service %q should not exist yet", srvName)
	}
	srv = routedRegistryService(t, srvName, "name")
	if err = r.AddService(srv); err != nil {
		t.Errorf("(4) cannot add service %s: %v", srvName, err)
	}
	srv = r.GetService(srvName)
	if srv == nil {
		t.Errorf("(5) service %q should exist", srvName)
	}

	// route should now match
	res, err := client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "foo"})
	if err != nil {
		t.Errorf("(6) failed to get onoff for foo: %v", err)
	}
	expect := &traits.OnOff{State: traits.OnOff_OFF}
	if diff := cmp.Diff(expect, res, protocmp.Transform()); diff != "" {
		t.Errorf("(7) unexpected response (-want +got):\n%s", diff)
	}

	// delete the service and the route should stop matching
	if !r.DeleteService(srvName) {
		t.Errorf("(8) service %q should exist", srvName)
	}
	if r.GetService(srvName) != nil {
		t.Errorf("(9) service %q should not exist", srvName)
	}
	_, err = client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "foo"})
	if statusErr, _ := status.FromError(err); statusErr.Code() != codes.Unimplemented {
		t.Errorf("(10) expected Unimplemented, got %v", statusErr)
	}
}

func routedRegistryService(t *testing.T, serviceName, keyName string) *Service {
	t.Helper()
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		t.Fatalf("descriptor for service %q not in registry: %v", serviceName, err)
	}
	servDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("%q is not a service", serviceName)
	}
	s, err := NewRoutedService(servDesc, keyName)
	if err != nil {
		t.Fatalf("failed to create routed service: %v", err)
	}
	return s
}

func TestWithKeyInterceptor(t *testing.T) {
	r := New(WithKeyInterceptor(func(key string) (mappedKey string, err error) {
		return strings.ToLower(key), nil
	}))

	check(t, r.AddService(routedRegistryService(t, traits.OnOffApi_ServiceDesc.ServiceName, "name")))
	model := onoffpb.NewModel(resource.WithInitialValue(&traits.OnOff{State: traits.OnOff_OFF}))
	check(t, r.AddRoute("", "foo", wrap.ServerToClient(traits.OnOffApi_ServiceDesc, onoffpb.NewModelServer(model))))

	// interceptor should map the request to "FOO" to the handler for "foo"
	conn := NewLoopback(r)
	client := traits.NewOnOffApiClient(conn)
	res, err := client.GetOnOff(context.Background(), &traits.GetOnOffRequest{Name: "FOO"})
	if err != nil {
		t.Errorf("failed to get onoff for FOO: %v", err)
	}
	expect := &traits.OnOff{State: traits.OnOff_OFF}
	if diff := cmp.Diff(expect, res, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
