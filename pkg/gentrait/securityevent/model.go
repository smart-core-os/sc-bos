package securityevent

import (
	"container/ring"
	"context"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

var subSystems = [2]string{"access control", "cctv"}

type Model struct {
	mu                sync.Mutex // guards allSecurityEvents and genId
	allSecurityEvents *ring.Ring // *securityeventpb.SecurityEvent
	genId             int

	lastSecurityEvent *resource.Value // of *securityeventpb.SecurityEvent
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&securityeventpb.SecurityEvent{})}
	opts = append(defaultOpts, opts...)

	m := &Model{
		lastSecurityEvent: resource.NewValue(opts...),
		allSecurityEvents: ring.New(100),
	}

	// let's add some events to start with so we can test the list method without waiting
	startTime := time.Now().Add(-100 * time.Minute)
	for range 100 {
		_, _ = m.GenerateSecurityEvent(timestamppb.New(startTime))
		startTime = startTime.Add(time.Minute)
	}

	return m
}

// AddSecurityEvent manually add a security event to the model
func (m *Model) AddSecurityEvent(se *securityeventpb.SecurityEvent, opts ...resource.WriteOption) (*securityeventpb.SecurityEvent, error) {
	v, err := m.lastSecurityEvent.Set(se, opts...)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allSecurityEvents.Value = se
	m.allSecurityEvents = m.allSecurityEvents.Next()
	return v.(*securityeventpb.SecurityEvent), nil
}

// GenerateSecurityEvent generates a security event and adds it to the model
func (m *Model) GenerateSecurityEvent(time *timestamppb.Timestamp) (*securityeventpb.SecurityEvent, error) {
	m.mu.Lock()
	se := &securityeventpb.SecurityEvent{
		SecurityEventTime: time,
		Description:       "Generated security event",
		Id:                strconv.Itoa(m.genId),
		Source: &securityeventpb.SecurityEvent_Source{
			Subsystem: subSystems[m.genId%2],
		},
		EventType: securityeventpb.SecurityEvent_EventType(m.genId % 22),
	}
	m.genId++
	m.mu.Unlock()
	return m.AddSecurityEvent(se)
}

func (m *Model) GetSecurityEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.allSecurityEvents.Len()
}

func (m *Model) ListSecurityEvents(start, count int) []*securityeventpb.SecurityEvent {

	var events []*securityeventpb.SecurityEvent
	// reverse to retrieve the latest events first

	for i := start - 1; i >= 0; i-- {
		e := m.allSecurityEvents.Move(i)
		events = append(events, e.Value.(*securityeventpb.SecurityEvent))
		if len(events) >= count {
			break
		}
	}
	return events
}

func (m *Model) pullSecurityEventsWrapper(request *securityeventpb.PullSecurityEventsRequest, server securityeventpb.SecurityEventApi_PullSecurityEventsServer) error {
	if !request.UpdatesOnly {
		m.mu.Lock()
		i := m.allSecurityEvents.Len() - 50
		if i < 0 {
			i = 0
		}
		for ; i < m.allSecurityEvents.Len()-1; i++ {
			e := m.allSecurityEvents.Move(i)
			event := e.Value.(*securityeventpb.SecurityEvent)
			change := &securityeventpb.PullSecurityEventsResponse_Change{
				Name:       request.Name,
				NewValue:   event,
				ChangeTime: event.SecurityEventTime,
				Type:       types.ChangeType_ADD,
			}
			if err := server.Send(&securityeventpb.PullSecurityEventsResponse{Changes: []*securityeventpb.PullSecurityEventsResponse_Change{change}}); err != nil {
				m.mu.Unlock()
				return err
			}
		}
		m.mu.Unlock()
	}
	for change := range m.PullSecurityEvents(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &securityeventpb.PullSecurityEventsResponse{}
		msg.Changes = append(msg.Changes, change)
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) PullSecurityEvents(ctx context.Context, opts ...resource.ReadOption) <-chan *securityeventpb.PullSecurityEventsResponse_Change {
	send := make(chan *securityeventpb.PullSecurityEventsResponse_Change)
	recv := m.lastSecurityEvent.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			value := change.Value.(*securityeventpb.SecurityEvent)
			send <- &securityeventpb.PullSecurityEventsResponse_Change{
				OldValue:   nil,
				NewValue:   value, // the mock driver only generates new security events and does not delete them
				ChangeTime: value.SecurityEventTime,
				Type:       types.ChangeType_ADD,
			}
		}
	}()

	return send
}
