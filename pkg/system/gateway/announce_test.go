package gateway

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-bos/internal/node/nodeopts"
	"github.com/smart-core-os/sc-bos/internal/util/grpc/reflectionapi"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/servicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

func TestSystem_announceCohort(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, th *announceTester)
	}{
		{
			name: "preloaded node",
			run: func(t *testing.T, th *announceTester) {
				th.addNode("ac1", "ac1/d1", "ac1/d2")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1", "ac1/d2")
			},
		},
		{
			name: "preloaded gateway",
			run: func(t *testing.T, th *announceTester) {
				th.addGateway("gw1",
					"gw1", "systems", "gw1/systems",
					"ac1/d1", "ac1/d2")
				th.runAnnounceCohort()
				th.assertSimpleDevices("gw1", "gw1/systems") // only the node and full service names get proxied
			},
		},
		{
			name: "preloaded hub",
			run: func(t *testing.T, th *announceTester) {
				th.addHub("hub", "hub/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("hub/d1")
			},
		},
		{
			name: "delayed node name",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.newRemoteNode("ac1", remoteDesc{}, remoteSystems{msgRecvd: true}, rds("ac1/d1")...)
				th.c.Nodes.Set(ac1)
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices yet because no name for node
				ac1.Self.Set(rd("ac1"))
				synctest.Wait()
				th.assertSimpleDevices("ac1/d1") // now we have devices
			},
		},
		{
			name: "delayed node name timeout",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.newRemoteNode("ac1", remoteDesc{}, remoteSystems{msgRecvd: true}, rds("ac1/d1")...)
				th.c.Nodes.Set(ac1)
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices yet because no name for node
				time.Sleep(waitTimeout)
				synctest.Wait()
				th.assertSimpleDevices("ac1/d1") // now we have devices
			},
		},
		{
			name: "delayed gateway",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.newRemoteNode("gw1", rd("gw1"), remoteSystems{}, rds("gw1/d1")...)
				th.c.Nodes.Set(gw1)
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices yet

				gw1.Systems.Set(remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}})
				synctest.Wait()
				th.assertSimpleDevices() // still no devices because it's a gateway
			},
		},
		{
			name: "delayed gateway timeout",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.newRemoteNode("gw1", rd("gw1"), remoteSystems{}, rds("gw1/d1")...)
				th.c.Nodes.Set(gw1)
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices yet
				time.Sleep(waitTimeout)
				synctest.Wait()
				th.assertSimpleDevices("gw1/d1") // now we have devices
			},
		},
		{
			name: "node becomes gateway",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.addNode("gw1", "gw1/d1",
					// devices with special handling
					"gw1", "drivers", "gw1/drivers")
				th.runAnnounceCohort()
				th.assertSimpleDevices("gw1/d1", "gw1", "gw1/drivers") // we have devices

				gw1.Systems.Set(remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}})
				synctest.Wait()
				th.assertSimpleDevices("gw1", "gw1/drivers") // devices are removed
			},
		},
		{
			name: "gateway stops being gateway",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.addGateway("gw1", "gw1/d1",
					// devices with special handling
					"gw1", "zones", "gw1/zones")
				th.runAnnounceCohort()
				th.assertSimpleDevices("gw1", "gw1/zones") // no devices because it's a gateway

				gw1.Systems.Set(remoteSystems{msgRecvd: true})
				synctest.Wait()
				th.assertSimpleDevices("gw1/d1", "gw1", "gw1/zones") // now we have devices
			},
		},
		{
			// Hub nodes always proxy all devices regardless of whether they also have an active gateway system.
			// This is the core behaviour added by the hub-as-gateway change.
			name: "hub with active gateway system proxies all devices",
			run: func(t *testing.T, th *announceTester) {
				hub := th.newRemoteNode("hub", rd("hub"), remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}}, rds("hub/d1", "hub/d2")...)
				hub.isHub = true
				th.c.Nodes.Set(hub)
				th.runAnnounceCohort()
				// shouldProxyDevice must return true for all hub devices even when isGateway() is true.
				th.assertSimpleDevices("hub/d1", "hub/d2")
			},
		},
		{
			// When a hub's gateway status changes, no device removal/re-add events should be emitted.
			// Without the hub guard in switchGatewayMode, renewDevicesSub() would be called and would
			// briefly remove all devices before re-adding them, producing spurious PullDevices events.
			name: "hub gateway status change emits no device churn events",
			run: func(t *testing.T, th *announceTester) {
				hub := th.addHub("hub", "hub/d1", "hub/d2")
				th.runAnnounceCohort()
				th.assertSimpleDevices("hub/d1", "hub/d2")

				stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))

				hub.Systems.Set(remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}})
				synctest.Wait()
				select {
				case c := <-stream:
					t.Fatalf("unexpected device change when hub became gateway: %+v", c)
				default:
				}

				hub.Systems.Set(remoteSystems{msgRecvd: true}) // gateway deactivated
				synctest.Wait()
				select {
				case c := <-stream:
					t.Fatalf("unexpected device change when hub stopped being gateway: %+v", c)
				default:
				}
			},
		},
		{
			// When a hub's gateway system becomes active, it must continue proxying services.
			// Without the hub guard in switchGatewayMode, closeServiceSub() would be called and
			// hub services would stop being proxied.
			name: "hub becoming gateway keeps services proxied",
			run: func(t *testing.T, th *announceTester) {
				desc := findServiceDesc(t, "smartcore.bos.onoff.v1.OnOffApi")
				hub := th.addHub("hub", "hub/d1")
				hub.Services.Set(desc)

				obsCore, logs := observer.New(zapcore.DebugLevel)
				th.sys.logger = zap.New(zapcore.NewTee(th.sys.logger.Core(), obsCore))

				th.runAnnounceCohort()

				initialCount := logs.FilterMessage("routable service announced").Len()
				if initialCount != 1 {
					t.Fatalf("expected 1 initial service announcement, got %d", initialCount)
				}

				// Hub gains gateway system — services must remain, no re-announcement.
				hub.Systems.Set(remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}})
				synctest.Wait()

				afterCount := logs.FilterMessage("routable service announced").Len()
				if afterCount != initialCount {
					t.Errorf("hub gateway status change caused %d spurious service re-announcement(s)", afterCount-initialCount)
				}
			},
		},
		{
			// A device added to a hub that has an active gateway system must still be proxied.
			name: "device added to hub with active gateway is proxied",
			run: func(t *testing.T, th *announceTester) {
				hub := th.newRemoteNode("hub", rd("hub"), remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}}, rds("hub/d1")...)
				hub.isHub = true
				th.c.Nodes.Set(hub)
				th.runAnnounceCohort()
				th.assertSimpleDevices("hub/d1")

				hub.Devices.Set(rd("hub/d2"))
				synctest.Wait()
				th.assertSimpleDevices("hub/d1", "hub/d2")
			},
		},
		{
			// shouldProxyDevice: non-hub gateway with self.name "" must not proxy any device.
			name: "gateway with unknown name proxies nothing",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.newRemoteNode("gw1",
					remoteDesc{}, // name not yet known
					remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}},
					rds("gw1", "gw1/drivers", "ac1/d1")...)
				th.c.Nodes.Set(gw1)
				th.runAnnounceCohort()
				// name is empty so shouldProxyDevice must return false for everything
				th.assertSimpleDevices()
			},
		},
		{
			// shouldProxyDevice: only the gateway's own name and its fixed-service children are proxied.
			name: "gateway proxies only own name and fixed service children",
			run: func(t *testing.T, th *announceTester) {
				th.addGateway("gw1",
					"gw1",             // own name → proxied
					"gw1/drivers",     // fixed service child → proxied
					"gw1/automations", // fixed service child → proxied
					"gw1/systems",     // fixed service child → proxied
					"gw1/zones",       // fixed service child → proxied
					"gw1/d1",          // arbitrary child → NOT proxied
					"gw1/zone1",       // non-fixed child → NOT proxied
					"ac1/d1",          // foreign device → NOT proxied
				)
				th.runAnnounceCohort()
				th.assertSimpleDevices("gw1", "gw1/drivers", "gw1/automations", "gw1/systems", "gw1/zones")
			},
		},
		{
			name: "simple service names aren't proxied",
			run: func(t *testing.T, th *announceTester) {
				th.addNode("ac1", "drivers", "automations", "zones", "systems")
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices because these names are special
			},
		},
		{
			name: "expanded service names are proxied",
			run: func(t *testing.T, th *announceTester) {
				th.addNode("ac1", "ac1/drivers", "ac1/automations", "ac1/zones", "ac1/systems")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/drivers", "ac1/automations", "ac1/zones", "ac1/systems")
			},
		},
		{
			name: "zone device proxied for non-gateway node",
			run: func(t *testing.T, th *announceTester) {
				th.addNode("ac1", "ac1/zones", "ac1/zone1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/zones", "ac1/zone1")
			},
		},
		{
			name: "zone device not proxied for gateway node",
			run: func(t *testing.T, th *announceTester) {
				th.addGateway("gw1", "gw1", "gw1/zones", "gw1/zone1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("gw1", "gw1/zones") // "gw1/zone1" not proxied
			},
		},
		{
			name: "node added",
			run: func(t *testing.T, th *announceTester) {
				th.addNode("ac1", "ac1/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1")
				th.addNode("ac2", "ac2/d1", "ac2/d2")
				synctest.Wait()
				th.assertSimpleDevices("ac1/d1", "ac2/d1", "ac2/d2")
			},
		},
		{
			name: "node removed",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.addNode("ac1", "ac1/d1", "ac1/d2")
				th.addNode("ac2", "ac2/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac2/d1")
				th.c.Nodes.Remove(ac1)
				synctest.Wait()
				th.assertSimpleDevices("ac2/d1")
			},
		},
		{
			name: "device added",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.addNode("ac1", "ac1/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1")
				ac1.Devices.Set(rd("ac1/d2"))
				ac1.Devices.Set(rd("ac1/d3"))
				synctest.Wait()
				th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac1/d3")
			},
		},
		{
			name: "device removed",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.addNode("ac1", "ac1/d1", "ac1/d2", "ac1/d3")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1", "ac1/d2", "ac1/d3")
				ac1.Devices.Remove(rd("ac1/d2"))
				synctest.Wait()
				th.assertSimpleDevices("ac1/d1", "ac1/d3")
			},
		},
		{
			name: "device updated",
			run: func(t *testing.T, th *announceTester) {
				ac1 := th.addNode("ac1", "ac1/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices("ac1/d1")

				stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))
				now := time.Now()
				ac1.Devices.Set(remoteDesc{name: "ac1/d1", md: &metadatapb.Metadata{
					Name:       "ac1/d1",
					Membership: &metadatapb.Metadata_Membership{Subsystem: "test devices"},
				}})

				wantOldDevice := &devicespb.Device{
					Name:     "ac1/d1",
					Metadata: md("ac1/d1", ts(trait.Metadata)...),
				}
				wantNewDevice := &devicespb.Device{
					Name: "ac1/d1",
					Metadata: &metadatapb.Metadata{
						Name:       "ac1/d1",
						Membership: &metadatapb.Metadata_Membership{Subsystem: "test devices"},
						Traits:     ts(trait.Metadata),
					},
				}
				assertDeviceUpdate(th.T, stream, wantOldDevice, wantNewDevice, now)
			},
		},
		{
			name: "gateway device added",
			run: func(t *testing.T, th *announceTester) {
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
			},
		},
		{
			name: "gateway device updated",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.addGateway("gw1", "ac1/d1")
				th.runAnnounceCohort()
				th.assertSimpleDevices() // no devices because it's a gateway

				stream := th.n.PullDevices(th.Context(), resource.WithUpdatesOnly(true))
				gw1.Devices.Set(remoteDesc{name: "ac1/d1", md: &metadatapb.Metadata{
					Name:       "ac1/d1",
					Membership: &metadatapb.Metadata_Membership{Subsystem: "test devices"},
				}})
				synctest.Wait()
				select {
				case c := <-stream:
					th.Fatalf("unexpected device update received: %+v", c)
				default:
					// expected, no update should be sent
				}
			},
		},
		{
			name: "preloaded node health",
			run: func(t *testing.T, th *announceTester) {
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
			},
		},
		{
			name: "health check update",
			run: func(t *testing.T, th *announceTester) {
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
			},
		},
		{
			name: "health device removed",
			run: func(t *testing.T, th *announceTester) {
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
			},
		},
		{
			name: "systems resolves before self.name",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.newRemoteNode("gw1",
					remoteDesc{}, // name not yet known
					remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}},
					rds("gw1", "gw1/drivers", "gw1/automations", "gw1/systems")...)
				th.c.Nodes.Set(gw1)
				th.runAnnounceCohort()

				time.Sleep(waitTimeout)  // force waitForFunc to time out with name still empty
				synctest.Wait()          // switchGatewayMode(true) has now run with self.name == ""
				th.assertSimpleDevices() // name unknown, can't proxy gateway devices yet

				gw1.Self.Set(rd("gw1"))
				synctest.Wait() // renewDevicesSub must re-evaluate with the now-known name
				th.assertSimpleDevices("gw1", "gw1/drivers", "gw1/automations", "gw1/systems")
			},
		},
		{
			// Regression test: reflectNode polls every 10s and calls node.Services.Replace
			// with fresh service descriptor objects. rx.Set.Replace emits an Update event for
			// every item already in the set — even when the value is logically unchanged — because
			// it can't distinguish a same-valued replacement from a real update. The Update handler
			// in announceRemoteNode un-announces and re-announces the service on every poll, causing
			// log spam and briefly removing routes.
			name: "service not re-announced when Replace is called with unchanged descriptors",
			run: func(t *testing.T, th *announceTester) {
				desc := findServiceDesc(t, "smartcore.bos.onoff.v1.OnOffApi")
				ac1 := th.addNode("ac1", "ac1/d1")
				ac1.Services.Set(desc)

				// Instrument the logger with an observer so we can count announcements.
				obsCore, logs := observer.New(zapcore.DebugLevel)
				th.sys.logger = zap.New(zapcore.NewTee(th.sys.logger.Core(), obsCore))

				th.runAnnounceCohort()

				initialCount := logs.FilterMessage("routable service announced").Len()
				if initialCount != 1 {
					t.Fatalf("expected 1 initial announcement, got %d", initialCount)
				}

				// Simulate what reflectNode does on every poll: replace with the same descriptors.
				// This should NOT trigger a re-announcement.
				ac1.Services.Replace([]protoreflect.ServiceDescriptor{desc})
				synctest.Wait()

				afterCount := logs.FilterMessage("routable service announced").Len()
				if afterCount != initialCount {
					t.Errorf("Services.Replace with unchanged descriptors caused %d spurious re-announcement(s)", afterCount-initialCount)
				}
			},
		},
		{
			name: "multiple gateway nodes with hub and delayed self.name",
			run: func(t *testing.T, th *announceTester) {
				gw1 := th.newRemoteNode("gw1",
					remoteDesc{}, // name not yet known
					remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}},
					rds("gw1", "gw1/drivers", "gw1/automations", "gw1/systems")...)
				gw2 := th.newRemoteNode("gw2",
					remoteDesc{}, // name not yet known
					remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}},
					rds("gw2", "gw2/drivers", "gw2/automations", "gw2/systems")...)

				th.addHub("hub", "hub", "hub/d1", "hub/drivers", "hub/automations", "hub/systems")
				for _, gw := range []*remoteNode{gw1, gw2} {
					for _, name := range []string{"hub", "hub/drivers", "hub/automations", "hub/systems"} {
						gw.Devices.Set(rd(name))
					}
				}

				th.c.Nodes.Set(gw1)
				th.c.Nodes.Set(gw2)
				th.runAnnounceCohort()

				time.Sleep(waitTimeout) // force waitForFunc to time out with name still empty
				synctest.Wait()         // switchGatewayMode(true) has now run for both with self.name == ""
				th.assertSimpleDevices("hub", "hub/d1", "hub/drivers", "hub/automations", "hub/systems")

				gw1.Self.Set(rd("gw1"))
				gw2.Self.Set(rd("gw2"))
				synctest.Wait() // renewDevicesSub re-evaluates with the now-known names

				th.assertSimpleDevices(
					"hub", "hub/d1", "hub/drivers", "hub/automations", "hub/systems",
					"gw1", "gw1/drivers", "gw1/automations", "gw1/systems",
					"gw2", "gw2/drivers", "gw2/automations", "gw2/systems",
				)
			},
		},
		{
			// Regression test for route-clobbering: when gw1's systems info arrives quickly
			// (msgRecvd=true) but without an active gateway service, gw1 is immediately treated
			// as a non-gateway and proxies all its devices — including hub re-exports and
			// downstream ac1 devices.  Those routes overwrite hub's routes.  When gw1 later
			// confirms it is a gateway, its non-gateway proxy is undone.  The undo must
			// *restore* hub's routes (via RestoreRouteIfConn) rather than deleting them.
			// At the announce layer this is visible as:
			//   - "ac1/d1" appears while gw1 is non-gateway, then disappears after gateway mode.
			//   - Hub devices persist throughout because hub announces them independently.
			name: "gateway with delayed gateway status",
			run: func(t *testing.T, th *announceTester) {
				// gw1 re-exports hub devices and has a downstream ac1/d1 device.
				gw1 := th.newRemoteNode("gw1",
					rd("gw1"),
					remoteSystems{msgRecvd: true}, // systems received, gateway status not yet known
					append(
						rds("gw1", "gw1/drivers", "gw1/automations", "gw1/systems"),
						rds("hub", "hub/d1", "hub/drivers", "hub/automations", "ac1/d1")...,
					)...)

				th.addHub("hub", "hub", "hub/d1", "hub/drivers", "hub/automations")
				th.c.Nodes.Set(gw1)
				th.runAnnounceCohort()

				// systems.msgRecvd=true → the systems wait resolves immediately →
				// isGateway()=false → switchGatewayMode(false) → all gw1 devices proxied.
				th.assertSimpleDevices(
					"gw1", "gw1/drivers", "gw1/automations", "gw1/systems",
					"hub", "hub/d1", "hub/drivers", "hub/automations",
					"ac1/d1",
				)

				// Gateway status arrives.
				gw1.Systems.Set(remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}})
				synctest.Wait()
				// gw1 switches to gateway mode:
				//   - ac1/d1 is no longer proxied (not a fixed-service device under gw1)
				//   - hub devices remain because hub announced them independently
				th.assertSimpleDevices(
					"gw1", "gw1/drivers", "gw1/automations", "gw1/systems",
					"hub", "hub/d1", "hub/drivers", "hub/automations",
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				th := newAnnounceTester(t)
				tt.run(t, th)
			})
		})
	}
}

// assertDeviceUpdate asserts that device updates are received that transition a device from wantOld to wantNew.
// Any sequence of updates that connect together to join the two states are accepted.
func assertDeviceUpdate(t *testing.T, stream <-chan devicespb.DevicesChange, wantOld, wantNew *devicespb.Device, now time.Time) {
	t.Helper()
	// all updates look a bit like this, with different old/new values
	want := devicespb.DevicesChange{
		Id:         "ac1/d1",
		ChangeType: typespb.ChangeType_UPDATE,
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
			ChangeType: typespb.ChangeType_UPDATE,
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

func rd(name string, checks ...*healthpb.HealthCheck) remoteDesc {
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

func hcs(descs ...string) []*healthpb.HealthCheck {
	var hcs []*healthpb.HealthCheck
	for _, d := range descs {
		hcs = append(hcs, makeHealthCheck(d))
	}
	slices.SortFunc(hcs, func(a, b *healthpb.HealthCheck) int {
		return strings.Compare(a.Id, b.Id)
	})
	return hcs
}

func md(name string, traitList ...*metadatapb.TraitMetadata) *metadatapb.Metadata {
	return &metadatapb.Metadata{
		Name:       name,
		Appearance: &metadatapb.Metadata_Appearance{Title: name},
		Traits:     traitList,
	}
}

func ts[S ~string](name ...S) []*metadatapb.TraitMetadata {
	var ts []*metadatapb.TraitMetadata
	for _, n := range name {
		ts = append(ts, &metadatapb.TraitMetadata{Name: string(n)})
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
	rn := t.newRemoteNode(addr, rd(addr), remoteSystems{msgRecvd: true, gateway: &servicespb.Service{Active: true}}, rds(devices...)...)
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
	var wantDevices []*devicespb.Device
	for _, name := range wantNames {
		wantDevices = append(wantDevices, newSimpleDevice(name))
	}
	t.assertDevices(wantDevices...)
}

func newSimpleDevice(name string, checks ...*healthpb.HealthCheck) *devicespb.Device {
	return &devicespb.Device{
		Name:         name,
		Metadata:     md(name, ts(trait.Metadata)...),
		HealthChecks: checks,
	}
}

func (t *announceTester) assertDevices(want ...*devicespb.Device) {
	t.Helper()
	slices.SortFunc(want, cmpDevices)
	// add in the self node t the right place keeping want sorted by name
	selfDevice := &devicespb.Device{Name: "self", Metadata: &metadatapb.Metadata{Name: "self", Traits: ts(trait.Metadata, trait.Parent), DeviceType: metadatapb.Metadata_NODE}}
	if i, ok := slices.BinarySearchFunc(want, selfDevice, cmpDevices); !ok {
		want = slices.Insert(want, i, selfDevice)
	}
	// health checks in devices should be ordered by id
	for _, device := range want {
		slices.SortFunc(device.HealthChecks, func(a, b *healthpb.HealthCheck) int {
			return strings.Compare(a.Id, b.Id)
		})
	}

	got := t.n.ListDevices()
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("announced devices mismatch (-want +got):\n%s", diff)
	}
}

func cmpDevices(a, b *devicespb.Device) int {
	return strings.Compare(a.Name, b.Name)
}

// findServiceDesc looks up a protoreflect.ServiceDescriptor by full name from the global registry.
// The relevant proto package must already be imported (and thus init()-registered) for this to succeed.
func findServiceDesc(t *testing.T, fullName string) protoreflect.ServiceDescriptor {
	t.Helper()
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(fullName))
	if err != nil {
		t.Fatalf("service %q not found in proto registry: %v", fullName, err)
	}
	sd, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("%q resolved to %T, want protoreflect.ServiceDescriptor", fullName, d)
	}
	return sd
}
