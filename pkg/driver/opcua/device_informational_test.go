package opcua

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

// The node ids these tests use, named for the real ones the split exists for: a metering
// device's usage point, which the Meter trait reads, and one of the ~35 per-phase registers
// that only exist so the UDMI export can publish them.
const (
	usageNode = "ns=2;s=LDB_X_00_01_L\\MMTR1\\TotWh"
	spareNode = "ns=2;s=LDB_X_00_01_L\\MMXU1\\A\\phsB"
)

// recordedChecks holds both of a device's point checks, which is what these tests are about:
// not just that a failure lands somewhere, but that it stays off the other check.
//
// Read back through the registry rather than through a create/update callback, because one
// registry drives both checks and a callback would have to demultiplex them.
type recordedChecks struct {
	reg                    *healthpb.Registry
	primary, informational *healthpb.FaultCheck
}

func newRecordedChecks(t *testing.T) *recordedChecks {
	t.Helper()
	out := &recordedChecks{reg: healthpb.NewRegistry()}
	checks := out.reg.ForOwner("test")

	var err error
	out.primary, err = checks.NewFaultCheck("test-device",
		getDeviceHealthCheck(healthpb.HealthCheck_OCCUPANT_IMPACT_UNSPECIFIED, healthpb.HealthCheck_EQUIPMENT_IMPACT_UNSPECIFIED))
	require.NoError(t, err)
	t.Cleanup(out.primary.Dispose)

	out.informational, err = checks.NewFaultCheck("test-device", getInformationalPointCheck())
	require.NoError(t, err)
	t.Cleanup(out.informational.Dispose)
	return out
}

// primaryCheck is the device's deviceStatusCheck as it stands now.
func (rc *recordedChecks) primaryCheck(t *testing.T) *healthpb.HealthCheck {
	t.Helper()
	// ids are registered absolute, as owner:checkId - see healthpb.AbsID
	c := rc.reg.GetCheck("test-device", healthpb.AbsID("test", deviceStatusCheckId))
	require.NotNil(t, c, "the device status check should be registered")
	return c
}

// informationalCheck is the device's informationalPointCheck as it stands now.
func (rc *recordedChecks) informationalCheck(t *testing.T) *healthpb.HealthCheck {
	t.Helper()
	c := rc.reg.GetCheck("test-device", healthpb.AbsID("test", informationalPointCheckId))
	require.NotNil(t, c, "the informational point check should be registered")
	return c
}

// faultOn returns the current fault on c carrying code, or nil when there is none.
func faultOn(c *healthpb.HealthCheck, code string) *healthpb.HealthCheck_Error {
	for _, f := range c.GetFaults().GetCurrentFaults() {
		if f.GetCode().GetCode() == code {
			return f
		}
	}
	return nil
}

// newSplitTestDevice builds a device carrying both point checks, marking the nodes named in
// informational as such. Otherwise the same shape as newTestDevice, stagger zeroed so the
// retry delays are the only thing on the clock.
func newSplitTestDevice(t *testing.T, sub subscriber, informational map[string]bool, nodeIds ...string) (*device, *recordedChecks) {
	t.Helper()
	rc := newRecordedChecks(t)
	cfg := &config.Device{Name: "test-device"}
	for _, nodeId := range nodeIds {
		cfg.Variables = append(cfg.Variables, &config.Variable{
			NodeId:        nodeId,
			ParsedNodeId:  mustParseNodeID(nodeId),
			Informational: informational[nodeId],
		})
	}
	dev := newDevice(cfg, zaptest.NewLogger(t), sub, rc.primary, rc.informational, nil)
	dev.maxStagger = 0
	return dev, rc
}

// TestDevice_subscribe_informationalPointStaysOffPrimaryCheck is the point of the whole split:
// a spare register nothing reads going away must not drag the device onto a dashboard scoped
// to whether it is doing its job.
func TestDevice_subscribe_informationalPointStaysOffPrimaryCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(map[string][]error{spareNode: {ua.StatusBadTimeout}})
		dev, rc := newSplitTestDevice(t, sub, map[string]bool{spareNode: true}, usageNode, spareNode)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		fault := faultOn(rc.informationalCheck(t), PointSubscribeError)
		require.NotNil(t, fault, "a failing informational point should raise %s on its own check", PointSubscribeError)
		require.Contains(t, fault.DetailsText, spareNode)

		require.Nil(t, faultOn(rc.primaryCheck(t), PointSubscribeError), "a point no trait reads is not a functional failure")
		require.Nil(t, faultOn(rc.primaryCheck(t), DeviceConfigError))
		require.Equal(t, healthpb.HealthCheck_NORMAL, rc.primaryCheck(t).GetNormality(),
			"deviceStatusCheck should stay normal while the device's usage point is delivering")

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_informationalPointGivenUpStaysOffPrimaryCheck is the same for a point
// the server refuses outright, which is the other fault code pointHealth owns.
func TestDevice_subscribe_informationalPointGivenUpStaysOffPrimaryCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(map[string][]error{spareNode: {ua.StatusBadNodeIDUnknown}})
		dev, rc := newSplitTestDevice(t, sub, map[string]bool{spareNode: true}, usageNode, spareNode)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		fault := faultOn(rc.informationalCheck(t), DeviceConfigError)
		require.NotNil(t, fault, "a refused informational point should raise %s on its own check", DeviceConfigError)
		require.Contains(t, fault.DetailsText, spareNode)

		require.Nil(t, faultOn(rc.primaryCheck(t), DeviceConfigError))
		require.Nil(t, faultOn(rc.primaryCheck(t), PointSubscribeError))
		require.Equal(t, healthpb.HealthCheck_NORMAL, rc.primaryCheck(t).GetNormality())

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_traitBackedPointFaultsPrimaryCheck is the reverse, and the case that
// must keep working: the usage point failing is exactly what the Metering dashboard is for.
func TestDevice_subscribe_traitBackedPointFaultsPrimaryCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(map[string][]error{usageNode: {ua.StatusBadTimeout}})
		dev, rc := newSplitTestDevice(t, sub, map[string]bool{spareNode: true}, usageNode, spareNode)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		fault := faultOn(rc.primaryCheck(t), PointSubscribeError)
		require.NotNil(t, fault, "the point backing the Meter trait belongs on deviceStatusCheck")
		require.Contains(t, fault.DetailsText, usageNode)
		require.NotContains(t, fault.DetailsText, spareNode, "a point that is delivering is not a fault")

		require.Nil(t, faultOn(rc.informationalCheck(t), PointSubscribeError),
			"the informational check should say nothing about a trait-backed point")
		require.Equal(t, healthpb.HealthCheck_ABNORMAL, rc.primaryCheck(t).GetNormality())

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_deadSubscriptionSplitsAcrossChecks checks failAllPoints splits the same
// way applyResults does. A subscription that never opened takes both kinds of point with it,
// and the informational ones must still not be reported as a functional failure.
func TestDevice_subscribe_deadSubscriptionSplitsAcrossChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(nil, ua.StatusBadTimeout)
		dev, rc := newSplitTestDevice(t, sub, map[string]bool{spareNode: true}, usageNode, spareNode)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		primary := faultOn(rc.primaryCheck(t), PointSubscribeError)
		require.NotNil(t, primary, "the whole device is waiting on the subscription")
		require.Contains(t, primary.DetailsText, usageNode)
		require.NotContains(t, primary.DetailsText, spareNode,
			"an informational point should be named on its own check, not this one")

		informational := faultOn(rc.informationalCheck(t), PointSubscribeError)
		require.NotNil(t, informational, "the informational points are still waiting on it too")
		require.Contains(t, informational.DetailsText, spareNode)
		require.NotContains(t, informational.DetailsText, usageNode)

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_subscribe_informationalPointRecovers checks the informational check clears the
// way the primary one does, rather than latching on the first failure.
func TestDevice_subscribe_informationalPointRecovers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sub := newFakeSubscriber(map[string][]error{spareNode: {ua.StatusBadTimeout, nil}})
		dev, rc := newSplitTestDevice(t, sub, map[string]bool{spareNode: true}, usageNode, spareNode)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := goSubscribe(ctx, dev)

		synctest.Wait()
		require.NotNil(t, faultOn(rc.informationalCheck(t), PointSubscribeError))

		// the straggler rejoins on the live subscription
		time.Sleep(subscribeRetryInitial)
		synctest.Wait()
		require.Nil(t, faultOn(rc.informationalCheck(t), PointSubscribeError), "a point that came good is not a fault")
		require.Equal(t, healthpb.HealthCheck_NORMAL, rc.informationalCheck(t).GetNormality())

		cancel()
		requireSubscribeReturns(t, done)
	})
}

// TestDevice_handleStatusValue_routesReadsByCheck covers the read path rather than the
// subscribe path: a Bad read on a spare register must neither fault the primary check nor,
// when it comes back Good, vouch for it.
func TestDevice_handleStatusValue_routesReadsByCheck(t *testing.T) {
	dev, rc := newSplitTestDevice(t, nil, map[string]bool{spareNode: true}, usageNode, spareNode)
	ctx := context.Background()

	dev.handleStatusValue(ctx, mustParseNodeID(spareNode), ua.StatusBadNodeIDUnknown, nil)
	rel := rc.informationalCheck(t).GetReliability()
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rel.GetState())
	require.Contains(t, rel.GetLastError().GetDetailsText(), spareNode)
	require.NotEqual(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rc.primaryCheck(t).GetReliability().GetState(),
		"a bad read on a point no trait reads must not mark the device unreliable")

	// a good read on the same point clears its own check and says nothing about the other
	dev.handleStatusValue(ctx, mustParseNodeID(spareNode), ua.StatusOK, float64(1))
	require.Equal(t, healthpb.HealthCheck_Reliability_RELIABLE, rc.informationalCheck(t).GetReliability().GetState())
	require.NotEqual(t, healthpb.HealthCheck_Reliability_RELIABLE, rc.primaryCheck(t).GetReliability().GetState(),
		"a point no trait reads answering does not show the device is doing its job")

	// and a bad read on the trait-backed point lands where it always did
	dev.handleStatusValue(ctx, mustParseNodeID(usageNode), ua.StatusBadNodeIDUnknown, nil)
	require.Equal(t, healthpb.HealthCheck_Reliability_BAD_RESPONSE, rc.primaryCheck(t).GetReliability().GetState())
	require.Equal(t, healthpb.HealthCheck_Reliability_RELIABLE, rc.informationalCheck(t).GetReliability().GetState(),
		"the informational check keeps its own last word")
}

// TestDevice_newDevice_withoutInformationalCheck pins the compatibility promise: a config that
// marks nothing informational behaves exactly as it did before the flag existed.
func TestDevice_newDevice_withoutInformationalCheck(t *testing.T) {
	cfg := &config.Device{
		Name: "test-device",
		Variables: []*config.Variable{
			{NodeId: usageNode, ParsedNodeId: mustParseNodeID(usageNode)},
			{NodeId: spareNode, ParsedNodeId: mustParseNodeID(spareNode)},
		},
	}
	require.False(t, hasInformationalVariable(cfg), "no variable is marked, so no second check is needed")

	dev := newDevice(cfg, zaptest.NewLogger(t), nil, newSimpleFaultCheck(t), nil, nil)
	require.Nil(t, dev.informationalCheck)
	require.Nil(t, dev.informationalPoints)
	require.Empty(t, dev.informational)

	// every point still routes to the primary check, which is what commitBatches must honour
	primary, informational := newPointBatch(), newPointBatch()
	require.Same(t, primary, dev.batchFor(spareNode, primary, informational))
	require.Same(t, primary, dev.batchFor(usageNode, primary, informational))
}

// TestDevice_informationalSet checks the set is built from the parsed node, since that is what
// the monitor results and notifications name a point by.
func TestDevice_informationalSet(t *testing.T) {
	cfg := &config.Device{
		Name: "test-device",
		Variables: []*config.Variable{
			{NodeId: usageNode, ParsedNodeId: mustParseNodeID(usageNode)},
			{NodeId: spareNode, ParsedNodeId: mustParseNodeID(spareNode), Informational: true},
		},
	}
	require.True(t, hasInformationalVariable(cfg))

	rc := newRecordedChecks(t)
	dev := newDevice(cfg, zaptest.NewLogger(t), nil, rc.primary, rc.informational, nil)
	require.Equal(t, map[string]bool{mustParseNodeID(spareNode).String(): true}, dev.informational)
	require.NotNil(t, dev.informationalPoints)
}
