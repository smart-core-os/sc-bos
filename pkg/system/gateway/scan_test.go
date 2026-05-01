package gateway

import (
	"context"
	"errors"
	"net"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/hubpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/onoffpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/rx"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/task/serviceapi"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/masks"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
)

func TestSystem_scanRemoteHub(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, hub := newMockCohort(t)
		gw1 := env.newGatewayNode("gw1")
		env.newGatewayNode("gw2")
		ac1 := env.newNode("ac1")
		ac2 := env.newNode("ac2")

		// create some devices on non-gateway nodes
		ac1.announceDeviceTraits("ac1/dev1", meterpb.TraitName, trait.OnOff)
		ac1.announceDeviceTraits("ac1/dev2", trait.OnOff)
		ac2.announceDeviceTraits("ac2/dev1", meterpb.TraitName)
		hub.announceDeviceTraits("hub/dev1", trait.OnOff)
		ac1.announceDeviceHealth("ac1/dev1", "-working", ">overheat")
		ac2.announceDeviceHealth("ac2/dev1", "+working")
		hub.announceDeviceHealth("hub/dev1", "+working")

		gw1Sys := &System{
			logger:     zaptest.NewLogger(t).With(zap.String("server", "gw1")),
			self:       gw1.node,
			hub:        hub,
			reflection: gw1.reflect,
			newClient:  env.newClient,
		}

		gw1Cohort := newCohort()
		go gw1Sys.scanRemoteHub(t.Context(), gw1Cohort, hub.conn)
		synctest.Wait() // all scanning done
		gw1CohortTester := newCohortTester(t, gw1Cohort)
		gw1CohortTester.assertNodes("hub", "gw2", "ac1", "ac2")
		hubNode := gw1CohortTester.node("hub")
		hubNode.assertDevices("hub/dev1")
		hubNode.assertDeviceTraits("hub/dev1", trait.OnOff)
		hubNode.assertDeviceHealth("hub/dev1", "+working")
		gw2Node := gw1CohortTester.node("gw2")
		gw2Node.assertDevices()
		ac1Node := gw1CohortTester.node("ac1")
		ac1Node.assertDevices("ac1/dev1", "ac1/dev2")
		ac1Node.assertDeviceTraits("ac1/dev1", meterpb.TraitName, trait.OnOff)
		ac1Node.assertDeviceHealth("ac1/dev1", "-working", ">overheat")
		ac1Node.assertDeviceTraits("ac1/dev2", trait.OnOff)
		ac2Node := gw1CohortTester.node("ac2")
		ac2Node.assertDevices("ac2/dev1")
		ac2Node.assertDeviceTraits("ac2/dev1", meterpb.TraitName)
		ac2Node.assertDeviceHealth("ac2/dev1", "+working")

		// test node modifications
		_, nodeChanges := gw1Cohort.Nodes.Sub(t.Context())
		env.newNode("ac3")
		synctest.Wait()
		assertChanVal(t, nodeChanges, func(ch rx.Change[*remoteNode]) {
			if ch.Type != rx.Add || ch.New.addr != "ac3" {
				t.Fatalf("unexpected node change for ac3 addition: %+v", ch)
			}
		})

		// test device modifications
		_, ac1DeviceChanges := ac1Node.node.Devices.Sub(t.Context())
		ac1.announceDeviceTraits("ac1/dev2", meterpb.TraitName) // a new trait for an existing device
		synctest.Wait()
		assertChanVal(t, ac1DeviceChanges, func(c rx.Change[remoteDesc]) {
			if c.Type != rx.Update {
				t.Fatalf("device update: want Update, got %v", c.Type)
			}
			if want := "ac1/dev2"; c.Old.name != want || c.New.name != want {
				t.Fatalf("device update: unexpected names: want=%q, got old=%q new=%q", want, c.Old.name, c.New.name)
			}
			wantOldMd := &metadatapb.Metadata{
				Name: "ac1/dev2",
				Traits: []*metadatapb.TraitMetadata{
					{Name: string(trait.Metadata)},
					{Name: string(trait.OnOff)},
				},
			}
			wantNewMd := &metadatapb.Metadata{
				Name: "ac1/dev2",
				Traits: []*metadatapb.TraitMetadata{
					{Name: string(meterpb.TraitName)},
					{Name: string(trait.Metadata)},
					{Name: string(trait.OnOff)},
				},
			}
			if diff := cmp.Diff(wantOldMd, c.Old.md, protocmp.Transform()); diff != "" {
				t.Fatalf("unexpected old metadata for device update (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantNewMd, c.New.md, protocmp.Transform()); diff != "" {
				t.Fatalf("unexpected new metadata for device update (-want +got):\n%s", diff)
			}
		})
		ac1.announceDeviceHealth("ac1/dev1", "+working") // was -working
		synctest.Wait()
		assertChanVal(t, ac1DeviceChanges, func(c rx.Change[remoteDesc]) {
			if c.Type != rx.Update {
				t.Fatalf("device update: want Update, got %v", c.Type)
			}
			if want := "ac1/dev1"; c.Old.name != want || c.New.name != want {
				t.Fatalf("device update: unexpected names: want=%q, got old=%q new=%q", want, c.Old.name, c.New.name)
			}
			assertDeviceHealth(t, c.Old, "-working", ">overheat")
			assertDeviceHealth(t, c.New, "+working", ">overheat")
		})
	})
}

func TestSystem_zoneServicesWhenHubAlsoGateway(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gw1 := newMockRemoteNode(t, "gw1")
		hub := newMockRemoteNode(t, "hub")
		ac1 := newMockRemoteNode(t, "ac1")
		ac1.announceZone("ac1/zone1")

		devs := devicespb.NewCollection()
		gw1Sys := &System{
			logger:     zaptest.NewLogger(t).With(zap.String("server", "gw1")),
			self:       gw1.node,
			reflection: gw1.reflect,
			checks:     devicesToHealthCheckCollection(devs),
			announcer:  gw1.node,
		}

		gw1Cohort := newCohort()

		hubRemote := newRemoteNode("hub", hub.conn)
		hubRemote.isHub = true
		hubRemote.Self.Set(remoteDesc{name: "hub"})
		hubRemote.Systems.Set(remoteSystems{msgRecvd: true})
		for _, name := range []string{
			"hub", "hub/zones",
			"hub/automations", "hub/drivers", "hub/systems",
			"automations", "drivers", "systems", "zones",
			"ac1", "ac1/zones", "ac1/zone1",
			"ac1/automations", "ac1/drivers", "ac1/systems",
		} {
			hubRemote.Devices.Set(remoteDesc{name: name})
		}

		ac1Remote := newRemoteNode("ac1", ac1.conn)
		ac1Remote.Self.Set(remoteDesc{name: "ac1"})
		ac1Remote.Systems.Set(remoteSystems{msgRecvd: true})

		gw1Cohort.Nodes.Set(hubRemote)
		gw1Cohort.Nodes.Set(ac1Remote)

		go gw1Sys.announceCohort(t.Context(), gw1Cohort)
		synctest.Wait()

		for _, name := range []string{
			"ac1", "ac1/zones", "ac1/zone1",
			"ac1/automations", "ac1/drivers", "ac1/systems",
			"automations", "drivers", "systems", "zones",
		} {
			ac1Remote.Devices.Set(remoteDesc{name: name})
		}
		synctest.Wait()

		hubRemote.Systems.Set(remoteSystems{
			msgRecvd: true,
			gateway:  &servicespb.Service{Active: true},
		})
		synctest.Wait()

		svcClient := servicespb.NewServicesApiClient(gw1.conn)

		res, err := svcClient.ListServices(t.Context(), &servicespb.ListServicesRequest{Name: "ac1/zones"})
		if err != nil {
			t.Fatalf("ListServices(name=ac1/zones) via gateway after hub gateway switch: %v", err)
		}
		var zoneFound bool
		for _, svc := range res.Services {
			if svc.Id == "ac1/zone1" {
				zoneFound = true
				break
			}
		}
		if !zoneFound {
			ids := make([]string, len(res.Services))
			for i, svc := range res.Services {
				ids[i] = svc.Id
			}
			t.Fatalf("zone ac1/zone1 not found in ListServices(name=ac1/zones) via gateway; got %v", ids)
		}
	})
}

func newMockCohort(t *testing.T) (_ *mockCohort, hub *mockRemoteNode) {
	t.Helper()
	hub = newMockRemoteNode(t, "hub")
	hubServer := hub.makeHub()
	return &mockCohort{
		t:         t,
		nodes:     map[string]*mockRemoteNode{"hub": hub},
		hubServer: hubServer,
	}, hub
}

type mockCohort struct {
	t     *testing.T
	nodes map[string]*mockRemoteNode // will always include the hub node at "hub"

	hubServer *mockHubServer
}

func (c *mockCohort) newClient(address string) (*grpc.ClientConn, error) {
	c.t.Helper()
	n, exists := c.nodes[address]
	if !exists {
		c.t.Fatalf("mock cohort node %q does not exist", address)
	}
	return n.Connect(c.t.Context())
}

func (c *mockCohort) newNode(name string) *mockRemoteNode {
	c.t.Helper()
	_, exists := c.nodes[name]
	if exists {
		c.t.Fatalf("mock cohort node %q already exists", name)
	}
	n := newMockRemoteNode(c.t, name)
	c.nodes[name] = n
	c.hubServer.AddHubNode(n)
	return n
}

func (c *mockCohort) newGatewayNode(name string) *mockRemoteNode {
	c.t.Helper()
	n := c.newNode(name)
	n.makeGateway()
	return n
}

func newMockRemoteNode(t *testing.T, name string) *mockRemoteNode {
	t.Helper()
	devs := devicespb.NewCollection()
	n := node.New(name, nodeopts.WithStore(devs))
	lis, conn := newLocalConn(t)
	server := grpc.NewServer(grpc.UnknownServiceHandler(n.ServerHandler()))

	reflectionServer := reflectionapi.NewServer(server, n)
	reflectionServer.Register(server)

	devicespb.RegisterDevicesApiServer(server, devices.NewServer(n))

	rn := &mockRemoteNode{
		t:        t,
		lis:      lis,
		conn:     conn,
		server:   server,
		reflect:  reflectionServer,
		node:     n,
		checks:   devicesToHealthCheckCollection(devs),
		services: make(map[serviceId]service.Lifecycle),
	}
	rn.systems = service.NewMap(rn.newService, service.IdIsRequired)
	rn.autos = service.NewMap(rn.newService, service.IdIsRequired)
	rn.drivers = service.NewMap(rn.newService, service.IdIsRequired)
	rn.zones = service.NewMap(rn.newService, service.IdIsRequired)

	services := []struct {
		base  string
		store *service.Map
	}{
		{"systems", rn.systems},
		{"automations", rn.autos},
		{"drivers", rn.drivers},
		{"zones", rn.zones},
	}
	for _, svc := range services {
		svcServer := serviceapi.NewApi(svc.store)
		n.Announce(svc.base, node.HasServer(servicespb.RegisterServicesApiServer, servicespb.ServicesApiServer(svcServer)))
		n.Announce(path.Join(name, svc.base), node.HasServer(servicespb.RegisterServicesApiServer, servicespb.ServicesApiServer(svcServer)))
	}

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("mock remote node %q server stopped: %v", name, err)
		}
	}()
	t.Cleanup(func() {
		server.Stop()
	})

	return rn
}

type mockRemoteNode struct {
	t *testing.T

	lis     *bufconn.Listener
	conn    *grpc.ClientConn
	server  *grpc.Server
	reflect *reflectionapi.Server
	// named trait apis, including each of the service types
	node *node.Node
	// underlying health check management
	checks system.HealthCheckCollection
	// different types of service
	systems, autos, drivers, zones *service.Map
	// running services
	services map[serviceId]service.Lifecycle
}

func (n *mockRemoteNode) makeHub() *mockHubServer {
	n.t.Helper()
	hubServer := newMockHubServer(n.t)
	hubSvc, err := node.RegistryService(hubpb.HubApi_ServiceDesc, hubServer)
	if err != nil {
		n.t.Fatalf("failed to create hub service: %v", err)
	}
	_, err = n.node.AnnounceService(hubSvc)
	if err != nil {
		n.t.Fatalf("failed to announce hub service: %v", err)
	}
	return hubServer
}

func (n *mockRemoteNode) makeGateway() {
	id, _, err := n.systems.Create(Name, Name, service.State{
		Active: true,
		Config: []byte("cfg"),
	})
	if err != nil {
		return
	}
	n.t.Cleanup(func() {
		_, err := n.systems.Delete(id)
		if err != nil {
			n.t.Errorf("failed to delete gateway system service: %v", err)
		}
	})
}

func (n *mockRemoteNode) Close() error {
	return nil
}

func (n *mockRemoteNode) Target() string {
	return n.node.Name()
}

func (n *mockRemoteNode) Connect(_ context.Context) (*grpc.ClientConn, error) {
	return n.conn, nil
}

type serviceId struct {
	id, kind string
}

func (n *mockRemoteNode) newService(id, kind string) (service.Lifecycle, error) {
	n.t.Helper()
	svc := service.New(func(ctx context.Context, config string) error {
		return nil
	}, service.WithParser(func(data []byte) (string, error) {
		return string(data), nil
	}))
	n.services[serviceId{id, kind}] = svc
	return svc, nil
}

func (n *mockRemoteNode) announceDeviceTraits(name string, tns ...trait.Name) {
	n.t.Helper()
	if len(tns) == 0 {
		n.t.Fatalf("no traits provided for device %q", name)
	}
	var opts []node.Feature
	for _, tn := range tns {
		switch tn {
		case meterpb.TraitName:
			srv := meterpb.NewModelServer(meterpb.NewModel())
			opts = append(opts,
				node.HasServer(meterpb.RegisterMeterApiServer, meterpb.MeterApiServer(srv)),
				node.HasTrait(tn),
			)
		case trait.OnOff:
			srv := onoffpb.NewModelServer(onoffpb.NewModel())
			opts = append(opts,
				node.HasServer(onoffpb.RegisterOnOffApiServer, onoffpb.OnOffApiServer(srv)),
				node.HasTrait(tn),
			)
		default:
			n.t.Fatalf("unsupported trait %q", tn)
		}
	}
	n.node.Announce(name, opts...)
}

// announceDeviceHealth announces health checks for the named device.
// Each check becomes the id and display name of the health check.
// If the check starts with:
//   - '+' it is normal (the default)
//   - '-' it is abnormal
//   - '>' it is high
//   - '<' it is low
//
// announceZone adds a zone to this node, mirroring what pkg/zone/area/area.go does in production.
// The zone appears in the zones service map (so ListServices(name: "<node>/zones") includes it)
// and is announced as a device with its own ServicesApi.
func (n *mockRemoteNode) announceZone(name string) {
	n.t.Helper()
	// Register in the zones service map so it shows up in ListServices(name: "<node>/zones").
	id, _, err := n.zones.Create(name, "area", service.State{
		Active: true,
		Config: []byte("{}"),
	})
	if err != nil {
		n.t.Fatalf("failed to create zone %q in service map: %v", name, err)
	}
	n.t.Cleanup(func() {
		_, _ = n.zones.Delete(id)
	})
	zoneServices := service.NewMap(n.newService, service.IdIsRequired)
	zoneSvcServer := serviceapi.NewApi(zoneServices)
	n.node.Announce(name, node.HasServer(servicespb.RegisterServicesApiServer, servicespb.ServicesApiServer(zoneSvcServer)))
}

func (n *mockRemoteNode) announceDeviceHealth(name string, checks ...string) {
	n.t.Helper()
	if len(checks) == 0 {
		n.t.Fatalf("no health checks provided for device %q", name)
	}
	var hc []*healthpb.HealthCheck
	for _, desc := range checks {
		hc = append(hc, makeHealthCheck(desc))
	}
	err := n.checks.MergeHealthChecks(name, hc...)
	if err != nil {
		n.t.Fatalf("failed to announce health checks for device %q: %v", name, err)
	}
}

func makeHealthCheck(desc string) *healthpb.HealthCheck {
	normality := healthpb.HealthCheck_NORMAL
	if len(desc) > 0 {
		switch desc[0] {
		case '+': // normal
			desc = strings.TrimSpace(desc[1:])
		case '-': // abnormal
			normality = healthpb.HealthCheck_ABNORMAL
			desc = strings.TrimSpace(desc[1:])
		case '>': // high
			normality = healthpb.HealthCheck_HIGH
			desc = strings.TrimSpace(desc[1:])
		case '<': // low
			normality = healthpb.HealthCheck_LOW
			desc = strings.TrimSpace(desc[1:])
		}
	}
	// simple check, just needs to have enough info to identify it
	return &healthpb.HealthCheck{
		Id:          desc,
		Normality:   normality,
		DisplayName: desc,
	}
}

func newMockHubServer(t *testing.T) *mockHubServer {
	t.Helper()
	return &mockHubServer{
		t:     t,
		nodes: resource.NewCollection(),
	}
}

type mockHubServer struct {
	hubpb.UnimplementedHubApiServer
	t     *testing.T
	nodes *resource.Collection // of *hubpb.HubNode, keyed by address
}

func (h *mockHubServer) AddHubNode(n *mockRemoteNode) {
	h.t.Helper()
	addr := n.node.Name()
	_, err := h.nodes.Add(addr, &hubpb.HubNode{
		Name:    addr,
		Address: addr,
	})
	if err != nil {
		h.t.Fatalf("failed to add hub node %q: %v", addr, err)
	}
}

func (h *mockHubServer) GetHubNode(_ context.Context, req *hubpb.GetHubNodeRequest) (*hubpb.HubNode, error) {
	res, ok := h.nodes.Get(req.GetAddress())
	if !ok {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return res.(*hubpb.HubNode), nil
}

func (h *mockHubServer) ListHubNodes(_ context.Context, _ *hubpb.ListHubNodesRequest) (*hubpb.ListHubNodesResponse, error) {
	var nodes []*hubpb.HubNode
	for _, r := range h.nodes.List() {
		nodes = append(nodes, r.(*hubpb.HubNode))
	}
	return &hubpb.ListHubNodesResponse{Nodes: nodes}, nil
}

func (h *mockHubServer) PullHubNodes(req *hubpb.PullHubNodesRequest, g grpc.ServerStreamingServer[hubpb.PullHubNodesResponse]) error {
	for c := range resources.PullCollection[*hubpb.HubNode](g.Context(), h.nodes.Pull(g.Context(), resource.WithUpdatesOnly(req.GetUpdatesOnly()))) {
		err := g.Send(&hubpb.PullHubNodesResponse{Changes: []*hubpb.PullHubNodesResponse_Change{
			{Type: c.ChangeType, NewValue: c.NewValue, OldValue: c.OldValue, ChangeTime: timestamppb.New(c.ChangeTime)},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func newCohortTester(t *testing.T, c *cohort) *cohortTester {
	t.Helper()
	return &cohortTester{
		t: t,
		c: c,
	}
}

type cohortTester struct {
	t *testing.T
	c *cohort
}

func (ct *cohortTester) assertNodes(wantAddrs ...string) {
	ct.t.Helper()
	if got, want := ct.c.Nodes.Len(), len(wantAddrs); got != want {
		ct.t.Fatalf("unexpected number of nodes in cohort: got %d, want %d", got, want)
	}
	var foundAddrs []string
	for _, n := range ct.c.Nodes.All {
		foundAddrs = append(foundAddrs, n.addr)
	}
	if diff := cmp.Diff(wantAddrs, foundAddrs, cmpopts.SortSlices(strings.Compare)); diff != "" {
		ct.t.Fatalf("unexpected node addresses in cohort (-want +got):\n%s", diff)
	}
}

func (ct *cohortTester) node(addr string) *cohortTesterNode {
	ct.t.Helper()
	_, got, found := ct.c.Nodes.Find(&remoteNode{addr: addr})
	if !found {
		ct.t.Fatalf("node %q not found in cohort", addr)
	}
	return &cohortTesterNode{
		t:    ct.t,
		node: got,
	}
}

type cohortTesterNode struct {
	t    *testing.T
	node *remoteNode
}

func (ctn *cohortTesterNode) assertDevices(names ...string) {
	ctn.t.Helper()

	// add the built-in devices to the list
	serviceTypes := []string{"systems", "automations", "drivers", "zones"}
	for _, st := range serviceTypes {
		names = append(names, st)
		names = append(names, path.Join(ctn.node.addr, st))
	}
	names = append(names, ctn.node.addr) // self device

	if got, want := ctn.node.Devices.Len(), len(names); got != want {
		ctn.t.Fatalf("unexpected number of devices on node %q: got %d, want %d", ctn.node.addr, got, want)
	}
	var foundNames []string
	for _, d := range ctn.node.Devices.All {
		foundNames = append(foundNames, d.name)
	}
	if diff := cmp.Diff(names, foundNames, cmpopts.SortSlices(strings.Compare)); diff != "" {
		ctn.t.Fatalf("unexpected device names on node %q (-want +got):\n%s", ctn.node.addr, diff)
	}
}

func (ctn *cohortTesterNode) assertDeviceTraits(name string, wantTraits ...trait.Name) {
	ctn.t.Helper()
	_, got, found := ctn.node.Devices.Find(remoteDesc{name: name})
	if !found {
		ctn.t.Fatalf("device %q not found on node %q", name, ctn.node.addr)
	}

	// all devices will have Metadata
	wantTraits = append(wantTraits, trait.Metadata)
	slices.Sort(wantTraits)

	var gotTraits []trait.Name
	for _, tm := range got.md.Traits {
		gotTraits = append(gotTraits, trait.Name(tm.GetName()))
	}
	if diff := cmp.Diff(wantTraits, gotTraits, cmpopts.SortSlices(strings.Compare)); diff != "" {
		ctn.t.Fatalf("unexpected traits for device %q on node %q (-want +got):\n%s", name, ctn.node.addr, diff)
	}
}

func (ctn *cohortTesterNode) assertDeviceHealth(name string, wantChecks ...string) {
	ctn.t.Helper()
	_, got, found := ctn.node.Devices.Find(remoteDesc{name: name})
	if !found {
		ctn.t.Fatalf("device %q not found on node %q", name, ctn.node.addr)
	}
	assertDeviceHealth(ctn.t, got, wantChecks...)
}

func assertDeviceHealth(t *testing.T, got remoteDesc, wantChecks ...string) {
	t.Helper()
	want := make([]*healthpb.HealthCheck, 0, len(wantChecks))
	for _, desc := range wantChecks {
		want = append(want, makeHealthCheck(desc))
	}
	if diff := cmp.Diff(want, got.health, protocmp.Transform(), cmpopts.SortSlices(func(a, b *healthpb.HealthCheck) int {
		return strings.Compare(a.Id, b.Id)
	})); diff != "" {
		t.Fatalf("unexpected health checks for device %q(-want +got):\n%s", got.name, diff)
	}
}

func assertChanVal[T any](t *testing.T, ch <-chan T, fn func(T)) {
	t.Helper()
	select {
	case v, ok := <-ch:
		if !ok {
			t.Fatalf("expected value from channel, but channel was closed")
		}
		fn(v)
	default:
		t.Fatalf("expected value from channel, but none available")
	}
}

func TestServicesEqual(t *testing.T) {
	// Helper to get a real service descriptor for testing
	getServiceDesc := func(t *testing.T, fullName string) protoreflect.ServiceDescriptor {
		t.Helper()
		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(fullName))
		if err != nil {
			t.Fatalf("failed to find service descriptor for %q: %v", fullName, err)
		}
		svcDesc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			t.Fatalf("%q is not a service descriptor", fullName)
		}
		return svcDesc
	}

	tests := []struct {
		name           string
		nodeServices   []string // service full names in the node
		remoteServices []string // service full names to compare
		want           bool
	}{
		{
			name:           "empty sets are equal",
			nodeServices:   []string{},
			remoteServices: []string{},
			want:           true,
		},
		{
			name: "identical services - prevents hot loop on repeated reflection polling",
			// This is the PRIMARY test case for the hot loop fix.
			// When reflectNode() is called every 10 seconds (via poll), it fetches service
			// descriptors via gRPC reflection. For custom project traits (e.g., "smartcore.vanti.UKPowerApi"),
			// the reflection API returns file descriptors each time. Without servicesEqual(),
			// node.Services.Replace() would be called every poll cycle, triggering service
			// change events even though nothing changed, causing continuous re-announcements.
			// servicesEqual() compares by FullName() which is stable across reflection calls.
			nodeServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.meter.v1.MeterApi",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.meter.v1.MeterApi",
			},
			want: true,
		},
		{
			name: "order doesn't matter - services can be in any order",
			// Custom traits returned from reflection may come in different orders.
			// The comparison must be order-independent to avoid false negatives.
			nodeServices: []string{
				"smartcore.bos.meter.v1.MeterApi",
				"smartcore.bos.onoff.v1.OnOffApi",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.meter.v1.MeterApi",
			},
			want: true,
		},
		{
			name: "multiple services with similar names - tests FullName uniqueness",
			// Custom project traits often have similar naming patterns (e.g., FooApi, FooInfo, FooHistory).
			// The comparison must correctly distinguish between them.
			nodeServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.onoff.v1.OnOffInfo",
				"smartcore.bos.meter.v1.MeterApi",
				"smartcore.bos.meter.v1.MeterInfo",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.onoff.v1.OnOffInfo",
				"smartcore.bos.meter.v1.MeterApi",
				"smartcore.bos.meter.v1.MeterInfo",
			},
			want: true,
		},
		{
			name:           "different number of services",
			nodeServices:   []string{"smartcore.bos.onoff.v1.OnOffApi"},
			remoteServices: []string{"smartcore.bos.onoff.v1.OnOffApi", "smartcore.bos.meter.v1.MeterApi"},
			want:           false,
		},
		{
			name:           "different services",
			nodeServices:   []string{"smartcore.bos.onoff.v1.OnOffApi"},
			remoteServices: []string{"smartcore.bos.meter.v1.MeterApi"},
			want:           false,
		},
		{
			name: "extra service in remote - device gained new trait",
			// Simulates when a device adds a new custom trait.
			// This should trigger Replace() to announce the new service.
			nodeServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.devices.v1.DevicesApi",
			},
			want: false,
		},
		{
			name: "missing service in remote - device lost trait",
			// Simulates when a device removes a custom trait.
			// This should trigger Replace() to unannounce the removed service.
			nodeServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.devices.v1.DevicesApi",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
			},
			want: false,
		},
		{
			name: "single service mismatch in many - detects subtle differences",
			// With many custom traits, a single trait change must be detected.
			// This ensures we don't miss updates when dealing with devices that
			// have many custom project-specific traits.
			nodeServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.meter.v1.MeterApi",
				"smartcore.bos.devices.v1.DevicesApi",
			},
			remoteServices: []string{
				"smartcore.bos.onoff.v1.OnOffApi",
				"smartcore.bos.meter.v1.MeterApi",
				"smartcore.bos.emergency.v1.EmergencyApi", // Different service
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock node with the specified services
			rn := newRemoteNode("test-node", nil)
			for _, svcName := range tt.nodeServices {
				rn.Services.Set(getServiceDesc(t, svcName))
			}

			// Create remote services list
			remoteServices := make([]protoreflect.ServiceDescriptor, 0, len(tt.remoteServices))
			for _, svcName := range tt.remoteServices {
				remoteServices = append(remoteServices, getServiceDesc(t, svcName))
			}

			// Test
			got := servicesEqual(rn, remoteServices)
			if got != tt.want {
				t.Errorf("servicesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestServicesEqual_CustomTraitScenario tests the behavior with service descriptors that simulate
// custom project traits. While we can't create mock ServiceDescriptors due to unexported interface methods,
// this test documents the real-world scenario and validates the comparison logic works correctly.
//
// In production, custom project traits are discovered via gRPC reflection and follow the naming pattern:
//   - "smartcore.vanti.UKPowerApi" (Vanti project-specific UK power monitoring)
//   - "smartcore.customer.BuildingSystemApi" (Customer project-specific building system)
//   - "smartcore.projectrepo.SpecialSensorApi" (Generic project-specific sensor)
//
// The key insight is that servicesEqual() compares services by FullName() which remains stable across
// reflection calls, preventing the hot loop even though the ServiceDescriptor instances differ.
func TestServicesEqual_CustomTraitScenario(t *testing.T) {
	// Simulate a device with both standard and "custom" traits
	// In reality, custom traits would have names like:
	//   - "smartcore.vanti.UKPowerApi"
	//   - "smartcore.customer.BuildingSystemApi"
	//   - "smartcore.projectrepo.SpecialSensorApi"
	// These are fetched via reflection and would have identical FullNames on repeated polls.

	// Get some real service descriptors
	onOffDesc, _ := protoregistry.GlobalFiles.FindDescriptorByName("smartcore.bos.onoff.v1.OnOffApi")
	meterDesc, _ := protoregistry.GlobalFiles.FindDescriptorByName("smartcore.bos.meter.v1.MeterApi")
	devicesDesc, _ := protoregistry.GlobalFiles.FindDescriptorByName("smartcore.bos.devices.v1.DevicesApi")

	onOffSvc := onOffDesc.(protoreflect.ServiceDescriptor)
	meterSvc := meterDesc.(protoreflect.ServiceDescriptor)
	devicesSvc := devicesDesc.(protoreflect.ServiceDescriptor)

	rn := newRemoteNode("test-node", nil)

	// Initial state: device has standard traits
	rn.Services.Set(onOffSvc)
	rn.Services.Set(meterSvc)

	// First reflection poll - same services returned
	remoteServices := []protoreflect.ServiceDescriptor{onOffSvc, meterSvc}
	if !servicesEqual(rn, remoteServices) {
		t.Error("Expected services to be equal on first poll (hot loop prevention)")
	}

	// Second reflection poll - SAME services but potentially different descriptor instances
	// In production with custom traits, reflection would return new instances with same FullName
	// We simulate this by calling FindDescriptorByName again (gets same descriptor from registry)
	onOffDesc2, _ := protoregistry.GlobalFiles.FindDescriptorByName("smartcore.bos.onoff.v1.OnOffApi")
	meterDesc2, _ := protoregistry.GlobalFiles.FindDescriptorByName("smartcore.bos.meter.v1.MeterApi")
	remoteServices2 := []protoreflect.ServiceDescriptor{
		onOffDesc2.(protoreflect.ServiceDescriptor),
		meterDesc2.(protoreflect.ServiceDescriptor),
	}
	if !servicesEqual(rn, remoteServices2) {
		t.Error("Expected services to be equal on second poll even with potentially different instances")
	}

	// Device adds a new trait (simulates custom trait being added)
	remoteServices3 := []protoreflect.ServiceDescriptor{onOffSvc, meterSvc, devicesSvc}
	if servicesEqual(rn, remoteServices3) {
		t.Error("Expected services to be different when new trait added")
	}

	// Update node state
	rn.Services.Replace(remoteServices3)

	// Third poll - verify the new state is recognized as equal
	if !servicesEqual(rn, remoteServices3) {
		t.Error("Expected services to be equal after update")
	}
}


func newLocalConn(t *testing.T) (*bufconn.Listener, *grpc.ClientConn) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	c, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	t.Cleanup(func() {
		err := errors.Join(
			c.Close(),
			lis.Close(),
		)
		if err != nil {
			t.Logf("failed to close local connection: %v", err)
		}
	})
	return lis, c
}

// todo: reuse this code which is duplicated in pkg/app

func devicesToHealthCheckCollection(d *devicespb.Collection) system.HealthCheckCollection {
	return (*devicesHealthCheckCollection)(d)
}

type devicesHealthCheckCollection devicespb.Collection

func (d *devicesHealthCheckCollection) MergeHealthChecks(name string, checks ...*healthpb.HealthCheck) error {
	_, err := (*devicespb.Collection)(d).Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
		dstDev := dst.(*devicespb.Device)
		dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, checks...)
	}), resource.WithCreateIfAbsent())
	return err
}

func (d *devicesHealthCheckCollection) RemoveHealthChecks(name string, ids ...string) error {
	_, err := (*devicespb.Collection)(d).Update(&devicespb.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, _ proto.Message) {
		dstDev := dst.(*devicespb.Device)
		for _, id := range ids {
			dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
		}
	}))
	if code := status.Code(err); code == codes.NotFound {
		err = nil
	}
	return err
}
