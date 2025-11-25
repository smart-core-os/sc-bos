package gateway

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/smart-core-os/sc-golang/pkg/trait"
)

func TestSystem_announceCohort(t *testing.T) {
	newAnnounceTest("preloaded node", t, func(th *announceTester) {
		th.addNode("ac1", "ac1/d1", "ac1/d2")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1", "ac1/d2")
	})
	newAnnounceTest("preloaded gateway", t, func(th *announceTester) {
		th.addGateway("gw1",
			"gw1", "systems", "gw1/systems",
			"ac1/d1", "ac1/d2")
		th.runAnnounceCohort()
		th.assertSimpleDevices("gw1", "gw1/systems") // only the node and full service names get proxied
	})
	newAnnounceTest("preloaded hub", t, func(th *announceTester) {
		th.addHub("hub", "hub/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices("hub/d1")
	})
	newAnnounceTest("delayed node name", t, func(th *announceTester) {
		ac1 := th.newRemoteNode("ac1", remoteDesc{}, remoteSystems{msgRecvd: true}, rds("ac1/d1")...)
		th.c.Nodes.Set(ac1)
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices yet because no name for node
		ac1.Self.Set(rd("ac1"))
		synctest.Wait()
		th.assertSimpleDevices("ac1/d1") // now we have devices
	})
	newAnnounceTest("delayed node name timeout", t, func(th *announceTester) {
		ac1 := th.newRemoteNode("ac1", remoteDesc{}, remoteSystems{msgRecvd: true}, rds("ac1/d1")...)
		th.c.Nodes.Set(ac1)
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices yet because no name for node
		time.Sleep(waitTimeout)
		synctest.Wait()
		th.assertSimpleDevices("ac1/d1") // now we have devices
	})
	newAnnounceTest("delayed gateway", t, func(th *announceTester) {
		gw1 := th.newRemoteNode("gw1", rd("gw1"), remoteSystems{}, rds("gw1/d1")...)
		th.c.Nodes.Set(gw1)
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices yet

		gw1.Systems.Set(remoteSystems{msgRecvd: true, gateway: &gen.Service{Active: true}})
		synctest.Wait()
		th.assertSimpleDevices() // still no devices because it's a gateway
	})
	newAnnounceTest("delayed gateway timeout", t, func(th *announceTester) {
		gw1 := th.newRemoteNode("gw1", rd("gw1"), remoteSystems{}, rds("gw1/d1")...)
		th.c.Nodes.Set(gw1)
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices yet
		time.Sleep(waitTimeout)
		synctest.Wait()
		th.assertSimpleDevices("gw1/d1") // now we have devices
	})
	newAnnounceTest("node becomes gateway", t, func(th *announceTester) {
		gw1 := th.addNode("gw1", "gw1/d1",
			// devices with special handling
			"gw1", "drivers", "gw1/drivers")
		th.runAnnounceCohort()
		th.assertSimpleDevices("gw1/d1", "gw1", "gw1/drivers") // we have devices

		gw1.Systems.Set(remoteSystems{msgRecvd: true, gateway: &gen.Service{Active: true}})
		synctest.Wait()
		th.assertSimpleDevices("gw1", "gw1/drivers") // devices are removed
	})
	newAnnounceTest("gateway stops being gateway", t, func(th *announceTester) {
		gw1 := th.addGateway("gw1", "gw1/d1",
			// devices with special handling
			"gw1", "zones", "gw1/zones")
		th.runAnnounceCohort()
		th.assertSimpleDevices("gw1", "gw1/zones") // no devices because it's a gateway

		gw1.Systems.Set(remoteSystems{msgRecvd: true})
		synctest.Wait()
		th.assertSimpleDevices("gw1/d1", "gw1", "gw1/zones") // now we have devices
	})
	newAnnounceTest("simple service names aren't proxied", t, func(th *announceTester) {
		th.addNode("ac1", "drivers", "automations", "zones", "systems")
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices because these names are special
	})
	newAnnounceTest("expanded service names are proxied", t, func(th *announceTester) {
		th.addNode("ac1", "ac1/drivers", "ac1/automations", "ac1/zones", "ac1/systems")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/drivers", "ac1/automations", "ac1/zones", "ac1/systems")
	})
	newAnnounceTest("node added", t, func(th *announceTester) {
		th.addNode("ac1", "ac1/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1")
		th.addNode("ac2", "ac2/d1", "ac2/d2")
		synctest.Wait()
		th.assertSimpleDevices("ac1/d1", "ac2/d1", "ac2/d2")
	})
	newAnnounceTest("node removed", t, func(th *announceTester) {
		ac1 := th.addNode("ac1", "ac1/d1", "ac1/d2")
		th.addNode("ac2", "ac2/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac2/d1")
		th.c.Nodes.Remove(ac1)
		synctest.Wait()
		th.assertSimpleDevices("ac2/d1")
	})
	newAnnounceTest("device added", t, func(th *announceTester) {
		ac1 := th.addNode("ac1", "ac1/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1")
		ac1.Devices.Set(rd("ac1/d2"))
		ac1.Devices.Set(rd("ac1/d3"))
		synctest.Wait()
		th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac1/d3")
	})
	newAnnounceTest("device removed", t, func(th *announceTester) {
		ac1 := th.addNode("ac1", "ac1/d1", "ac1/d2", "ac1/d3")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac1/d3")
		ac1.Devices.Remove(rd("ac1/d2"))
		synctest.Wait()
		th.assertSimpleDevices("ac1/d1", "ac1/d3")
	})
	newAnnounceTest("device updated", t, func(th *announceTester) {
		ac1 := th.addNode("ac1", "ac1/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices("ac1/d1")

		stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))
		now := time.Now()
		ac1.Devices.Set(remoteDesc{name: "ac1/d1", md: &traits.Metadata{
			Name:       "ac1/d1",
			Membership: &traits.Metadata_Membership{Subsystem: "test devices"},
		}})

		wantOldDevice := &gen.Device{
			Name:     "ac1/d1",
			Metadata: md("ac1/d1", ts(trait.Metadata)...),
		}
		wantNewDevice := &gen.Device{
			Name: "ac1/d1",
			Metadata: &traits.Metadata{
				Name:       "ac1/d1",
				Membership: &traits.Metadata_Membership{Subsystem: "test devices"},
				Traits:     ts(trait.Metadata),
			},
		}
		assertDeviceUpdate(th.T, stream, wantOldDevice, wantNewDevice, now)
	})
	newAnnounceTest("gateway device added", t, func(th *announceTester) {
		gw1 := th.addGateway("gw1")
		th.runAnnounceCohort()
		th.assertSimpleDevices()
		// should not be added
		gw1.Devices.Set(rd("gw1/d1"))
		synctest.Wait()
		th.assertSimpleDevices()
		// should be added
		gw1.Devices.Set(rd("gw1"))
		gw1.Devices.Set(rd("gw1/zones"))
		synctest.Wait()
		th.assertSimpleDevices("gw1", "gw1/zones")
	})
	newAnnounceTest("gateway device updated", t, func(th *announceTester) {
		gw1 := th.addGateway("gw1", "ac1/d1")
		th.runAnnounceCohort()
		th.assertSimpleDevices() // no devices because it's a gateway

		stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))
		gw1.Devices.Set(remoteDesc{name: "ac1/d1", md: &traits.Metadata{
			Name:       "ac1/d1",
			Membership: &traits.Metadata_Membership{Subsystem: "test devices"},
		}})
		synctest.Wait()
		select {
		case c := <-stream:
			th.Fatalf("unexpected device update received: %+v", c)
		default:
			// expected, no update should be sent
		}
	})
	// health checks
	newAnnounceTest("preloaded node health", t, func(th *announceTester) {
		rn := th.newRemoteNode("ac1", rd("ac1"), remoteSystems{msgRecvd: true},
			rd("ac1/d1", hcs("+online", ">hot")...),
			rd("ac1/d2", hcs("-off")...),
		)
		th.c.Nodes.Set(rn)
		th.runAnnounceCohort()
		th.assertDevices(
			newSimpleDevice("ac1/d1", hcs("+online", ">hot")...),
			newSimpleDevice("ac1/d2", hcs("-off")...),
		)
	})
	newAnnounceTest("health check update", t, func(th *announceTester) {
		rn := th.newRemoteNode("ac1", rd("ac1"), remoteSystems{msgRecvd: true},
			rd("ac1/d1", hcs("-online", ">hot")...),
		)
		th.c.Nodes.Set(rn)
		th.runAnnounceCohort()
		stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))
		rn.Devices.Set(rd("ac1/d1", hcs("+online", "+cold")...))

		wantOld := newSimpleDevice("ac1/d1", hcs("-online", ">hot")...)
		wantNew := newSimpleDevice("ac1/d1", hcs("+online", "+cold")...)
		assertDeviceUpdate(th.T, stream, wantOld, wantNew, time.Now())
	})
	newAnnounceTest("health device removed", t, func(th *announceTester) {
		rn := th.newRemoteNode("ac1", rd("ac1"), remoteSystems{msgRecvd: true},
			rd("ac1/d1", hcs("+online", ">hot")...),
		)
		th.c.Nodes.Set(rn)
		th.runAnnounceCohort()
		th.assertDevices(
			newSimpleDevice("ac1/d1", hcs("+online", ">hot")...),
		)
		rn.Devices.Remove(remoteDesc{name: "ac1/d1"})
		synctest.Wait()
		th.assertDevices()
	})
}

func newAnnounceTest(name string, t *testing.T, f func(t *announceTester)) {
	t.Run(name, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			th := newAnnounceTester(t)
			f(th)
		})
	})
}

// assertDeviceUpdate asserts that device updates are received that transition a device from wantOld to wantNew.
// Any sequence of updates that connect together to join the two states are accepted.
func assertDeviceUpdate(t *testing.T, stream <-chan devicespb.DevicesChange, wantOld, wantNew *gen.Device, now time.Time) {
	t.Helper()
	// all updates look a bit like this, with different old/new values
	want := devicespb.DevicesChange{
		Id:         "ac1/d1",
		ChangeType: types.ChangeType_UPDATE,
		ChangeTime: now,
		OldValue:   wantOld,
		NewValue:   wantNew,
	}
	for {
		synctest.Wait()
		got, ok := <-stream
		if !ok {
			t.Fatal("device update stream closed")
		}
		if diff := cmp.Diff(want, got, protocmp.Transform()); diff == "" {
			return // we reached the desired state
		}
		// We are in an intermediate update state.
		// We can't predict the exact state transition, but we can verify the old state
		wantIntermediate := devicespb.DevicesChange{
			Id:         "ac1/d1",
			ChangeType: types.ChangeType_UPDATE,
			ChangeTime: now,
			OldValue:   want.OldValue,
		}
		want.OldValue = got.NewValue
		got.NewValue = nil // because we can't predict it, clear for comparison
		if diff := cmp.Diff(wantIntermediate, got, protocmp.Transform()); diff != "" {
			t.Fatalf("intermediate update breaks chain on old value (-want +got):\n%s", diff)
		}
	}
}

func newFakeClientConn() *grpc.ClientConn {
	return &grpc.ClientConn{} // can do nothing with this
}

func rd(name string, checks ...*gen.HealthCheck) remoteDesc {
	return remoteDesc{
		name:   name,
		md:     md(name),
		health: checks,
	}
}

func rds(names ...string) []remoteDesc {
	var rds []remoteDesc
	for _, n := range names {
		rds = append(rds, rd(n))
	}
	return rds
}

func hcs(descs ...string) []*gen.HealthCheck {
	var hcs []*gen.HealthCheck
	for _, d := range descs {
		hcs = append(hcs, makeHealthCheck(d))
	}
	slices.SortFunc(hcs, func(a, b *gen.HealthCheck) int {
		return strings.Compare(a.Id, b.Id)
	})
	return hcs
}

func md(name string, traitList ...*traits.TraitMetadata) *traits.Metadata {
	return &traits.Metadata{
		Name:       name,
		Appearance: &traits.Metadata_Appearance{Title: name},
		Traits:     traitList,
	}
}

func ts[S ~string](name ...S) []*traits.TraitMetadata {
	var ts []*traits.TraitMetadata
	for _, n := range name {
		ts = append(ts, &traits.TraitMetadata{Name: string(n)})
	}
	return ts
}

func newAnnounceTester(t *testing.T) *announceTester {
	devs := devicespb.NewCollection()
	n := node.New("self", nodeopts.WithStore(devs))
	rs := reflectionapi.NewServer(n)
	sys := &System{
		self:       n,
		reflection: rs,
		announcer:  n,
		checks:     devicesToHealthCheckCollection(devs),
		logger:     zaptest.NewLogger(t),
	}
	c := newCohort()
	return &announceTester{
		T:   t,
		sys: sys,
		n:   n,
		rs:  rs,
		c:   c,
	}
}

type announceTester struct {
	*testing.T
	sys *System
	n   *node.Node
	rs  *reflectionapi.Server
	c   *cohort
}

func (t *announceTester) runAnnounceCohort() {
	go t.sys.announceCohort(t.T.Context(), t.c)
	synctest.Wait()
}

func (t *announceTester) addNode(addr string, devices ...string) *remoteNode {
	rn := t.newRemoteNode(addr, rd(addr), remoteSystems{msgRecvd: true}, rds(devices...)...)
	t.c.Nodes.Set(rn)
	return rn
}

func (t *announceTester) addGateway(addr string, devices ...string) *remoteNode {
	rn := t.newRemoteNode(addr, rd(addr), remoteSystems{msgRecvd: true, gateway: &gen.Service{Active: true}}, rds(devices...)...)
	t.c.Nodes.Set(rn)
	return rn
}

func (t *announceTester) addHub(addr string, devices ...string) *remoteNode {
	rn := t.newRemoteNode(addr, rd(addr), remoteSystems{msgRecvd: true}, rds(devices...)...)
	rn.isHub = true
	t.c.Nodes.Set(rn)
	return rn
}

func (t *announceTester) newRemoteNode(addr string, self remoteDesc, systems remoteSystems, devices ...remoteDesc) *remoteNode {
	rn := newRemoteNode(addr, newFakeClientConn())
	rn.Self.Set(self)
	rn.Systems.Set(systems)
	for _, d := range devices {
		rn.Devices.Set(d)
	}
	return rn
}

func (t *announceTester) assertSimpleDevices(wantNames ...string) {
	t.Helper()
	var wantDevices []*gen.Device
	for _, name := range wantNames {
		wantDevices = append(wantDevices, newSimpleDevice(name))
	}
	t.assertDevices(wantDevices...)
}

func newSimpleDevice(name string, checks ...*gen.HealthCheck) *gen.Device {
	return &gen.Device{
		Name:         name,
		Metadata:     md(name, ts(trait.Metadata)...),
		HealthChecks: checks,
	}
}

func (t *announceTester) assertDevices(want ...*gen.Device) {
	t.Helper()
	slices.SortFunc(want, cmpDevices)
	// add in the self node t the right place keeping want sorted by name
	selfDevice := &gen.Device{Name: "self", Metadata: &traits.Metadata{Name: "self", Traits: ts(trait.Metadata, trait.Parent)}}
	if i, ok := slices.BinarySearchFunc(want, selfDevice, cmpDevices); !ok {
		want = slices.Insert(want, i, selfDevice)
	}
	// health checks in devices should be ordered by id
	for _, device := range want {
		slices.SortFunc(device.HealthChecks, func(a, b *gen.HealthCheck) int {
			return strings.Compare(a.Id, b.Id)
		})
	}

	got := t.n.ListDevices()
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("announced devices mismatch (-want +got):\n%s", diff)
	}
}

func cmpDevices(a, b *gen.Device) int {
	return strings.Compare(a.Name, b.Name)
}
