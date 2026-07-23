package paxton

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// processEvents receives events newest-first (orderBy=eventTime DESC); the most recent
// must be applied last so it is the value GetLastAccessAttempt returns.
func TestProcessEvents_NewestEventBecomesCurrentState(t *testing.T) {
	d := &Driver{logger: zap.NewNop(), seen: newSeenEvents()}
	door := &Door{id: 7, name: "Door 7", deviceName: "doors/7", accessState: resource.NewValue(resource.WithNoDuplicates())}
	d.doors.Store(7, door)

	now := time.Now().UTC().Truncate(time.Second)
	events := []Event{
		{ID: 3, EventTime: now, Address: 7, EventType: 15, EventDescription: "Access permitted"},
		{ID: 2, EventTime: now.Add(-time.Second), Address: 7, EventType: 15, EventDescription: "Access permitted"},
		{ID: 1, EventTime: now.Add(-2 * time.Second), Address: 7, EventType: 15, EventDescription: "Access permitted"},
	}

	d.processEvents(context.Background(), events, "polling")

	attempt, ok := door.accessState.Get().(*accesspb.AccessAttempt)
	require.True(t, ok)
	assert.Equal(t, now.Unix(), attempt.GetAccessAttemptTime().AsTime().Unix())
}

func TestSourceEvent_MatchesDoorByAddress(t *testing.T) {
	d := &Driver{logger: zap.NewNop()}
	d.doors.Store(7, &Door{id: 7, name: "Front Door"})
	ec := NewEventsController(d)

	src := &securityeventpb.SecurityEvent_Source{}
	ok := ec.sourceEvent(Event{Address: 7, DeviceName: "Front Door"}, src)

	assert.True(t, ok)
	assert.Equal(t, "7", src.Id)
	assert.Equal(t, "Front Door", src.Name)
}

func TestSourceEvent_CardholderSourceUsesCardholderName(t *testing.T) {
	d := &Driver{logger: zap.NewNop()}
	d.cardholders.Store(42, &Cardholder{id: 42, deviceName: "cardholder/42"})
	ec := NewEventsController(d)

	src := &securityeventpb.SecurityEvent_Source{}
	// DeviceName here is a door name; the cardholder source must not adopt it.
	ok := ec.sourceEvent(Event{UserID: 42, FirstName: "Jane", Surname: "Doe", DeviceName: "Front Door"}, src)

	assert.True(t, ok)
	assert.Equal(t, "42", src.Id)
	assert.Equal(t, "Jane Doe", src.Name)
}

func TestSourceEvent_UnknownSourceReturnsFalse(t *testing.T) {
	d := &Driver{logger: zap.NewNop()}
	ec := NewEventsController(d)

	src := &securityeventpb.SecurityEvent_Source{}
	ok := ec.sourceEvent(Event{Address: 99, UserID: 99}, src)

	assert.False(t, ok)
}

func TestSignalRLiveEvent_ToEvent_ReportsBadTime(t *testing.T) {
	e := &SignalRLiveEvent{EventTime: "not-a-time", UserID: 5, Details: "x"}
	ev, err := e.toEvent()
	require.Error(t, err)
	assert.Equal(t, 5, ev.UserID) // other fields still populated despite the parse error
}

func TestSignalRLiveEvent_ToEvent_ParsesTimeAndSplitsName(t *testing.T) {
	e := &SignalRLiveEvent{EventTime: "2026-03-30T12:00:00Z", UserName: "Jane Doe"}
	ev, err := e.toEvent()
	require.NoError(t, err)
	assert.Equal(t, "Jane", ev.FirstName)
	assert.Equal(t, "Doe", ev.Surname)
	assert.Equal(t, 2026, ev.EventTime.Year())
}
