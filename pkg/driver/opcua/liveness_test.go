package opcua

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

const testLiveNode = "ns=2;s=Tag1"

func TestPointLiveness(t *testing.T) {
	l := newPointLiveness(nil, testLiveNode)
	require.False(t, l.vouched(testLiveNode), "nothing delivered yet")

	l.good(testLiveNode, func() {})
	require.True(t, l.vouched(testLiveNode))

	l.unusable(testLiveNode)
	require.False(t, l.vouched(testLiveNode))
	// a good read does not stand in for the monitored item recovering
	l.readGood(testLiveNode, l.beginRead(testLiveNode), func() {})
	require.False(t, l.vouched(testLiveNode))
	l.good(testLiveNode, func() {})
	require.True(t, l.vouched(testLiveNode))

	l.connectionSuspect()
	require.False(t, l.vouched(testLiveNode))
	l.readGood(testLiveNode, l.beginRead(testLiveNode), func() {})
	require.True(t, l.vouched(testLiveNode), "a good read after a blip restores a delivered point")

	l.subscriptionLost()
	require.False(t, l.vouched(testLiveNode))
	l.connectionSuspect()
	l.readGood(testLiveNode, l.beginRead(testLiveNode), func() {})
	require.False(t, l.vouched(testLiveNode), "a new subscription has to deliver before it is vouched for")
}

// TestPointLiveness_readRacesNotification checks a live read that comes back after a
// notification landed leaves the cache to the notification, which is at least as fresh.
func TestPointLiveness_readRacesNotification(t *testing.T) {
	l := newPointLiveness(nil, testLiveNode)
	gen := l.beginRead(testLiveNode)
	l.good(testLiveNode, func() {})

	applied := false
	l.readGood(testLiveNode, gen, func() { applied = true })
	require.False(t, applied)

	l.readGood(testLiveNode, l.beginRead(testLiveNode), func() { applied = true })
	require.True(t, applied, "a read with nothing landing in between updates the cache")
}

func TestPointLiveness_connected(t *testing.T) {
	up := true
	l := newPointLiveness(func() bool { return up }, testLiveNode)
	l.good(testLiveNode, func() {})
	require.True(t, l.vouched(testLiveNode))
	up = false
	require.False(t, l.vouched(testLiveNode), "nothing is vouched for while the connection is down")
	up = true
	require.True(t, l.vouched(testLiveNode))
}

// TestPointLiveness_slowApply checks a cache update that is held up, as resource.Value.Set is
// by a slow Pull listener, blocks neither gopcua's connection goroutine nor a Get.
func TestPointLiveness_slowApply(t *testing.T) {
	l := newPointLiveness(nil, testLiveNode)
	applying, release := make(chan struct{}), make(chan struct{})
	go l.good(testLiveNode, func() {
		close(applying)
		<-release
	})
	<-applying
	defer close(release)

	done := make(chan struct{})
	go func() {
		defer close(done)
		l.connectionSuspect()
		l.subscriptionLost()
		l.unusable(testLiveNode)
		l.vouched(testLiveNode)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("blocked behind a cache update")
	}
}

func TestPointLiveness_unknownNode(t *testing.T) {
	l := newPointLiveness(nil, testLiveNode)
	l.good("ns=2;s=Other", func() { t.Error("applied a value for a node the trait does not read") })
	l.unusable("ns=2;s=Other")
	require.False(t, l.vouched("ns=2;s=Other"))
}

func TestLiveValue(t *testing.T) {
	v, err := liveValue(goodValue(float32(7)))
	require.NoError(t, err)
	require.Equal(t, float32(7), v)

	v, err = liveValue(&ua.DataValue{Value: ua.MustVariant(float32(7)), Status: ua.StatusUncertainLastUsableValue})
	require.NoError(t, err, "uncertain is usable, as it is from the subscription")
	require.Equal(t, float32(7), v)

	for _, dv := range []*ua.DataValue{nil, {Status: ua.StatusBadNoCommunication}, {Status: ua.StatusOK}} {
		_, err := liveValue(dv)
		require.Error(t, err, "%v", dv)
	}
}
