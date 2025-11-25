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
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/internal/manage/devices"
	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/meter"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/system/gateway/internal/rx"
	"github.com/smart-core-os/sc-bos/pkg/task/service"
	"github.com/smart-core-os/sc-bos/pkg/task/serviceapi"
	"github.com/smart-core-os/sc-bos/pkg/util/resources"
	"github.com/smart-core-os/sc-golang/pkg/masks"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
	"github.com/smart-core-os/sc-golang/pkg/trait/onoffpb"
	"github.com/smart-core-os/sc-golang/pkg/wrap"
)

func TestSystem_scanRemoteHub(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, hub := newMockCohort(t)
		gw1 := env.newGatewayNode("gw1")
		env.newGatewayNode("gw2")
		ac1 := env.newNode("ac1")
		ac2 := env.newNode("ac2")

		// create some devices on non-gateway nodes
		ac1.announceDeviceTraits("ac1/dev1", meter.TraitName, trait.OnOff)
		ac1.announceDeviceTraits("ac1/dev2", trait.OnOff)
		ac2.announceDeviceTraits("ac2/dev1", meter.TraitName)
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
		ac1Node.assertDeviceTraits("ac1/dev1", meter.TraitName, trait.OnOff)
		ac1Node.assertDeviceHealth("ac1/dev1", "-working", ">overheat")
		ac1Node.assertDeviceTraits("ac1/dev2", trait.OnOff)
		ac2Node := gw1CohortTester.node("ac2")
		ac2Node.assertDevices("ac2/dev1")
		ac2Node.assertDeviceTraits("ac2/dev1", meter.TraitName)
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
		ac1.announceDeviceTraits("ac1/dev2", meter.TraitName) // a new trait for an existing device
		synctest.Wait()
		assertChanVal(t, ac1DeviceChanges, func(c rx.Change[remoteDesc]) {
			if c.Type != rx.Update {
				t.Fatalf("device update: want Update, got %v", c.Type)
			}
			if want := "ac1/dev2"; c.Old.name != want || c.New.name != want {
				t.Fatalf("device update: unexpected names: want=%q, got old=%q new=%q", want, c.Old.name, c.New.name)
			}
			wantOldMd := &traits.Metadata{
				Name: "ac1/dev2",
				Traits: []*traits.TraitMetadata{
					{Name: string(trait.Metadata)},
					{Name: string(trait.OnOff)},
				},
			}
			wantNewMd := &traits.Metadata{
				Name: "ac1/dev2",
				Traits: []*traits.TraitMetadata{
					{Name: string(meter.TraitName)},
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
	return n.Connect(nil)
}

func (c *mockCohort) newNode(name string) *mockRemoteNode {
	c.t.Helper()
	n, exists := c.nodes[name]
	if exists {
		c.t.Fatalf("mock cohort node %q already exists", name)
	}
	n = newMockRemoteNode(c.t, name)
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

	gen.RegisterDevicesApiServer(server, devices.NewServer(n))

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
		client := gen.WrapServicesApi(serviceapi.NewApi(svc.store))
		n.Announce(svc.base, node.HasClient(client))
		n.Announce(path.Join(name, svc.base), node.HasClient(client))
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
	hubSvc, err := node.RegistryService(gen.HubApi_ServiceDesc, hubServer)
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

func (n *mockRemoteNode) newAuto(id, kind string) service.Lifecycle {
	n.t.Helper()
	id, _, err := n.autos.Create(id, kind, service.State{
		Active: true,
		Config: []byte("cfg"),
	})
	if err != nil {
		n.t.Fatalf("failed to create automation service %q/%q: %v", id, kind, err)
	}
	n.t.Cleanup(func() {
		_, err := n.autos.Delete(id)
		if err != nil {
			n.t.Errorf("failed to delete automation service %q/%q: %v", id, kind, err)
		}
	})
	r := n.autos.Get(id)
	if r == nil {
		n.t.Fatalf("automation service %q/%q not found after creation", id, kind)
	}
	return r.Service
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
		var client wrap.ServiceUnwrapper
		switch tn {
		case meter.TraitName:
			client = gen.WrapMeterApi(meter.NewModelServer(meter.NewModel()))
		case trait.OnOff:
			client = onoffpb.WrapApi(onoffpb.NewModelServer(onoffpb.NewModel()))
		default:
			n.t.Fatalf("unsupported trait %q", tn)
		}
		opts = append(opts, node.HasTrait(tn, node.WithClients(client)))
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
func (n *mockRemoteNode) announceDeviceHealth(name string, checks ...string) {
	n.t.Helper()
	if len(checks) == 0 {
		n.t.Fatalf("no health checks provided for device %q", name)
	}
	var hc []*gen.HealthCheck
	for _, desc := range checks {
		hc = append(hc, makeHealthCheck(desc))
	}
	err := n.checks.MergeHealthChecks(name, hc...)
	if err != nil {
		n.t.Fatalf("failed to announce health checks for device %q: %v", name, err)
	}
}

func makeHealthCheck(desc string) *gen.HealthCheck {
	normality := gen.HealthCheck_NORMAL
	if len(desc) > 0 {
		switch desc[0] {
		case '+': // normal
			desc = strings.TrimSpace(desc[1:])
		case '-': // abnormal
			normality = gen.HealthCheck_ABNORMAL
			desc = strings.TrimSpace(desc[1:])
		case '>': // high
			normality = gen.HealthCheck_HIGH
			desc = strings.TrimSpace(desc[1:])
		case '<': // low
			normality = gen.HealthCheck_LOW
			desc = strings.TrimSpace(desc[1:])
		}
	}
	// simple check, just needs to have enough info to identify it
	return &gen.HealthCheck{
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
	gen.UnimplementedHubApiServer
	t     *testing.T
	nodes *resource.Collection // of *gen.HubNode, keyed by address
}

func (h *mockHubServer) AddHubNode(n *mockRemoteNode) {
	h.t.Helper()
	addr := n.node.Name()
	_, err := h.nodes.Add(addr, &gen.HubNode{
		Name:    addr,
		Address: addr,
	})
	if err != nil {
		h.t.Fatalf("failed to add hub node %q: %v", addr, err)
	}
}

func (h *mockHubServer) GetHubNode(_ context.Context, req *gen.GetHubNodeRequest) (*gen.HubNode, error) {
	res, ok := h.nodes.Get(req.GetAddress())
	if !ok {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return res.(*gen.HubNode), nil
}

func (h *mockHubServer) ListHubNodes(_ context.Context, _ *gen.ListHubNodesRequest) (*gen.ListHubNodesResponse, error) {
	var nodes []*gen.HubNode
	for _, r := range h.nodes.List() {
		nodes = append(nodes, r.(*gen.HubNode))
	}
	return &gen.ListHubNodesResponse{Nodes: nodes}, nil
}

func (h *mockHubServer) PullHubNodes(req *gen.PullHubNodesRequest, g grpc.ServerStreamingServer[gen.PullHubNodesResponse]) error {
	for c := range resources.PullCollection[*gen.HubNode](g.Context(), h.nodes.Pull(g.Context(), resource.WithUpdatesOnly(req.GetUpdatesOnly()))) {
		err := g.Send(&gen.PullHubNodesResponse{Changes: []*gen.PullHubNodesResponse_Change{
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
	want := make([]*gen.HealthCheck, 0, len(wantChecks))
	for _, desc := range wantChecks {
		want = append(want, makeHealthCheck(desc))
	}
	if diff := cmp.Diff(want, got.health, protocmp.Transform(), cmpopts.SortSlices(func(a, b *gen.HealthCheck) int {
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

func (d *devicesHealthCheckCollection) MergeHealthChecks(name string, checks ...*gen.HealthCheck) error {
	_, err := (*devicespb.Collection)(d).Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, src proto.Message) {
		dstDev := dst.(*gen.Device)
		dstDev.HealthChecks = healthpb.MergeChecks(mask.Merge, dstDev.HealthChecks, checks...)
	}), resource.WithCreateIfAbsent())
	return err
}

func (d *devicesHealthCheckCollection) RemoveHealthChecks(name string, ids ...string) error {
	_, err := (*devicespb.Collection)(d).Update(&gen.Device{Name: name}, resource.WithMerger(func(mask *masks.FieldUpdater, dst, _ proto.Message) {
		dstDev := dst.(*gen.Device)
		for _, id := range ids {
			dstDev.HealthChecks = healthpb.RemoveCheck(dstDev.HealthChecks, id)
		}
	}))
	if code := status.Code(err); code == codes.NotFound {
		err = nil
	}
	return err
}
