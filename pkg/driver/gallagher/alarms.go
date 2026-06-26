package gallagher

import (
	"container/ring"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type AlarmPayload struct {
	Href    string    `json:"href"`
	Id      string    `json:"id"`
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
	Source  struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Href string `json:"href"`
	} `json:"source"`
	Type      string `json:"type"`
	EventType struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"eventType"`
	Priority int    `json:"priority"`
	State    string `json:"state"`
	Active   bool   `json:"active"`
}

type AlarmList struct {
	Alarms []AlarmPayload `json:"alarms"`
	Next   *struct {
		Href string `json:"href"`
	} `json:"next,omitempty"`
}

type Alarm struct {
	AlarmPayload
}

type SecurityEventController struct {
	securityeventpb.UnimplementedSecurityEventApiServer

	client        *Client
	logger        *zap.Logger
	mu            sync.Mutex
	lastAlarmTime time.Time // cursor for alarms API
	lastEventTime time.Time // cursor for events API
	// security events is a circular buffer, it always points to the oldest security event
	securityEvents *ring.Ring // *securityeventpb.SecurityEvent
	updates        minibus.Bus[*securityeventpb.PullSecurityEventsResponse_Change]
}

func newSecurityEventController(client *Client, logger *zap.Logger, n int) *SecurityEventController {
	return &SecurityEventController{
		client:         client,
		logger:         logger,
		lastAlarmTime:  time.Now().Add(-24 * time.Hour),
		lastEventTime:  time.Now().Add(-24 * time.Hour),
		securityEvents: ring.New(n),
	}
}

// getAlarms gets the top level list of alarms, the returned list is sorted in oldest first order
func (sc *SecurityEventController) getAlarms() ([]*Alarm, error) {
	var result []*Alarm
	url := sc.client.getUrl("alarms")

	for {
		body, err := sc.client.doRequest(url)
		if err != nil {
			sc.logger.Error("failed to get alarms", zap.Error(err))
			return nil, err
		}

		var resultsList AlarmList
		err = json.Unmarshal(body, &resultsList)
		if err != nil {
			sc.logger.Error("failed to decode alarm list", zap.Error(err))
			return nil, err
		}

		for _, alarm := range resultsList.Alarms {

			a := &Alarm{
				AlarmPayload: alarm,
			}
			sc.getAlarmDetails(a)
			result = append(result, a)
		}

		if resultsList.Next == nil || resultsList.Next.Href == "" {
			break
		} else {
			url = resultsList.Next.Href
		}
	}
	slices.SortFunc(result, func(i, j *Alarm) int {
		if i.Time.Before(j.Time) {
			return -1
		} else if i.Time.After(j.Time) {
			return 1
		} else {
			return 0
		}
	})

	return result, nil
}

// getAlarmDetails gets & populates the full details for the given alarms
func (sc *SecurityEventController) getAlarmDetails(alarm *Alarm) {
	resp, err := sc.client.doRequest(alarm.Href)
	if err != nil {
		sc.logger.Error("failed to get alarm", zap.Error(err))
		return
	}

	err = json.Unmarshal(resp, &alarm)
	if err != nil {
		sc.logger.Error("failed to decode alarm", zap.Error(err))
	}
}

func newSecurityEvent(t time.Time, id, message string, priority int, sourceId, sourceName string) *securityeventpb.SecurityEvent {
	return &securityeventpb.SecurityEvent{
		SecurityEventTime: timestamppb.New(t),
		Description:       message,
		Id:                id,
		Priority:          int32(priority),
		Source: &securityeventpb.SecurityEvent_Source{
			Id:        sourceId,
			Name:      sourceName,
			Subsystem: "acs",
		},
	}
}

func (sc *SecurityEventController) addSecurityEvent(ctx context.Context, event *securityeventpb.SecurityEvent) {
	sc.securityEvents.Value = event
	sc.securityEvents = sc.securityEvents.Next()
	sc.updates.Send(ctx, &securityeventpb.PullSecurityEventsResponse_Change{
		ChangeTime: timestamppb.Now(),
		OldValue:   nil,
		NewValue:   event,
	})
}

// refreshAlarms call the Gallagher alarms API and add any new ones to the sc that are newer than our current newest
func (sc *SecurityEventController) refreshAlarms(ctx context.Context) error {
	alarms, err := sc.getAlarms()
	if err != nil {
		return fmt.Errorf("failed to get alarms: %w", err)
	}

	for _, alarm := range alarms {
		if !alarm.Time.After(sc.lastAlarmTime) {
			break
		}
		event := newSecurityEvent(alarm.Time, alarm.Id, alarm.Message, alarm.Priority, alarm.Source.Id, alarm.Source.Name)
		sc.addSecurityEvent(ctx, event)
		// the events in alarms are always oldest first, so this is fine
		sc.lastAlarmTime = alarm.Time
		sc.logger.Info("adding new security event", zap.Time("time", alarm.Time), zap.String("message", alarm.Message))
	}
	return nil
}

// run the alarm controller schedule to refresh the alarms and events
func (sc *SecurityEventController) run(ctx context.Context, schedule *jsontypes.Schedule) error {

	if err := sc.refreshAlarms(ctx); err != nil {
		sc.logger.Error("failed to refresh alarms, will try again on next run...", zap.Error(err))
	}
	if err := sc.refreshEvents(ctx); err != nil {
		sc.logger.Error("failed to refresh events, will try again on next run...", zap.Error(err))
	}

	t := time.Now()
	for {
		next := schedule.Next(t)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
			t = next
		}

		sc.mu.Lock()
		if err := sc.refreshAlarms(ctx); err != nil {
			sc.logger.Error("failed to refresh alarms, will try again on next run...", zap.Error(err))
		}
		if err := sc.refreshEvents(ctx); err != nil {
			sc.logger.Error("failed to refresh events, will try again on next run...", zap.Error(err))
		}
		sc.mu.Unlock()
	}
}

// maxSecurityEventPageSize caps how many events a single ListSecurityEvents
// page may return, regardless of the requested page size.
const maxSecurityEventPageSize = 1000

func (sc *SecurityEventController) ListSecurityEvents(_ context.Context, req *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {
	if req.PageSize < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid page size")
	}
	pageSize := int(req.PageSize)
	if pageSize == 0 || pageSize > maxSecurityEventPageSize {
		pageSize = maxSecurityEventPageSize
	}

	// start is the index, counting from the newest event, of the first event to
	// return. The page token is just this offset.
	start := 0
	if req.PageToken != "" {
		s, err := strconv.Atoi(req.PageToken)
		if err != nil || s < 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
		start = s
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	response := &securityeventpb.ListSecurityEventsResponse{}
	// The ring head points at the oldest slot (the next write position), so the
	// newest event is one step back from it. Walk backwards so the most recent
	// events come first, as the UI expects. When the buffer isn't full yet the
	// unwritten slots are nil and sit past the oldest event, so the first nil
	// marks the end of the events.
	node := sc.securityEvents
	for n := 0; n < sc.securityEvents.Len(); n++ {
		node = node.Prev()
		se, ok := node.Value.(*securityeventpb.SecurityEvent)
		if !ok {
			break
		}
		// TotalSize counts every event, not just those on this page.
		response.TotalSize++
		if int(response.TotalSize) <= start {
			continue // before the requested page
		}
		if len(response.SecurityEvents) < pageSize {
			response.SecurityEvents = append(response.SecurityEvents, se)
		}
	}

	// Only hand out a token when there are more events past this page.
	if next := start + len(response.SecurityEvents); next < int(response.TotalSize) {
		response.NextPageToken = strconv.Itoa(next)
	}
	return response, nil
}

func (sc *SecurityEventController) PullSecurityEvents(_ *securityeventpb.PullSecurityEventsRequest, server grpc.ServerStreamingServer[securityeventpb.PullSecurityEventsResponse]) error {
	for msg := range sc.updates.Listen(server.Context()) {
		var response securityeventpb.PullSecurityEventsResponse
		response.Changes = append(response.Changes, msg)
		err := server.Send(&response)
		if err != nil {
			return err
		}
	}
	return nil
}
