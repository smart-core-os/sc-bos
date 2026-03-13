package router

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	occupancysensorpb2 "github.com/smart-core-os/sc-bos/pkg/trait/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/wrap"
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

// TestSwapRoute_RestoreRouteIfConn tests that SwapRoute + RestoreRouteIfConn correctly restores
// the previous route when the current owner undoes its proxy.
// This is important for the gateway system: when a non-gateway fallback announcer temporarily
// proxies a device (overwriting the hub's route), undoing the proxy should restore the hub's route.
func TestSwapRoute_RestoreRouteIfConn(t *testing.T) {
	r := New()

	// Distinct sentinel values for three different connections.
	hubConn := &fakeConn{"hub"}
	eg02Conn := &fakeConn{"eg02"}

	// Step 1: hub sets the route.
	prev, err := r.SwapRoute("", "BC-01/automations", hubConn)
	if err != nil {
		t.Fatalf("hub SwapRoute: %v", err)
	}
	if prev != nil {
		t.Errorf("hub SwapRoute prev: got %v, want nil", prev)
	}

	// Step 2: EG-02 (non-gateway fallback) overwrites the hub's route.
	prev, err = r.SwapRoute("", "BC-01/automations", eg02Conn)
	if err != nil {
		t.Fatalf("eg02 SwapRoute: %v", err)
	}
	if prev != hubConn {
		t.Errorf("eg02 SwapRoute prev: got %v, want hubConn", prev)
	}

	// Step 3: EG-02 switches to gateway mode and undoes its proxy.
	// RestoreRouteIfConn should restore the hub's connection.
	restored := r.RestoreRouteIfConn("", "BC-01/automations", eg02Conn, hubConn)
	if !restored {
		t.Error("RestoreRouteIfConn: expected true (route was eg02Conn), got false")
	}

	// Step 4: hub's route should now be active — a hub undo should succeed.
	deleted := r.DeleteRouteIfConn("", "BC-01/automations", hubConn)
	if !deleted {
		t.Error("hub DeleteRouteIfConn: expected true (route should be hubConn after restore), got false")
	}
}

// TestRestoreRouteIfConn_NopWhenOverwritten verifies that RestoreRouteIfConn is a no-op if
// the route has already been replaced by a third party.
func TestRestoreRouteIfConn_NopWhenOverwritten(t *testing.T) {
	r := New()
	connA := &fakeConn{"A"}
	connB := &fakeConn{"B"}
	connC := &fakeConn{"C"}

	_, _ = r.SwapRoute("", "key", connA)
	_, _ = r.SwapRoute("", "key", connB) // B overwrites A

	// B's undo: try to restore A (but C has since overwritten B).
	_, _ = r.SwapRoute("", "key", connC) // C overwrites B

	restored := r.RestoreRouteIfConn("", "key", connB, connA)
	if restored {
		t.Error("RestoreRouteIfConn should be a no-op when current conn is not connB")
	}
}

// TestRestoreRouteIfConn_DeleteWhenNoPrev verifies that RestoreRouteIfConn deletes the route
// when prev is nil (i.e., there was no previous route).
func TestRestoreRouteIfConn_DeleteWhenNoPrev(t *testing.T) {
	r := New()
	conn := &fakeConn{"A"}

	prev, _ := r.SwapRoute("", "key", conn)
	if prev != nil {
		t.Fatalf("expected no prev route, got %v", prev)
	}

	deleted := r.RestoreRouteIfConn("", "key", conn, nil)
	if !deleted {
		t.Error("RestoreRouteIfConn with nil prev should delete the route")
	}

	// verify the route is gone
	if r.DeleteRouteIfConn("", "key", conn) {
		t.Error("route should have been deleted")
	}
}

// fakeConn is a minimal grpc.ClientConnInterface used only for route identity checks.
type fakeConn struct{ name string }

func (f *fakeConn) Invoke(_ context.Context, _ string, _, _ any, _ ...grpc.CallOption) error {
	return nil
}

func (f *fakeConn) NewStream(_ context.Context, _ *grpc.StreamDesc, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
