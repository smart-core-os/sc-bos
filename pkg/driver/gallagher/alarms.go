package gallagher

import (
	"container/ring"
	"context"
	"encoding/json"
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

	client *client
	logger *zap.Logger
	mu     sync.Mutex
	// security events is a circular buffer, it always points to the oldest security event
	lastEventTime  time.Time  // *securityeventpb.SecurityEvent
	securityEvents *ring.Ring // *securityeventpb.SecurityEvent
	// eventsList maintains events in a slice for efficient pagination (newest first)
	eventsList []*securityeventpb.SecurityEvent
	maxEvents  int
	updates    minibus.Bus[*securityeventpb.PullSecurityEventsResponse_Change]
}

func newSecurityEventController(client *client, logger *zap.Logger, n int) *SecurityEventController {
	return &SecurityEventController{
		client:         client,
		logger:         logger,
		lastEventTime:  time.Now().Add(-24 * time.Hour),
		securityEvents: ring.New(n),
		eventsList:     make([]*securityeventpb.SecurityEvent, 0, n),
		maxEvents:      n,
	}
}

// getAlarms gets the top level list of alarms, the returned list is sorted in oldest first order
func (sc *SecurityEventController) getAlarms(ctx context.Context) ([]*Alarm, error) {
	var result []*Alarm
	url := sc.client.getUrl("alarms")

	for {
		body, err := sc.client.doRequest(ctx, url)
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
			sc.getAlarmDetails(ctx, a)
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
func (sc *SecurityEventController) getAlarmDetails(ctx context.Context, alarm *Alarm) {
	resp, err := sc.client.doRequest(ctx, alarm.Href)
	if err != nil {
		sc.logger.Error("failed to get alarm", zap.Error(err))
		return
	}

	err = json.Unmarshal(resp, &alarm)
	if err != nil {
		sc.logger.Error("failed to decode alarm", zap.Error(err))
	}
}

// refreshAlarms call the Gallagher alarms API and add any new ones to the sc that are newer than our current newest
func (sc *SecurityEventController) refreshAlarms(ctx context.Context) error {
	alarms, err := sc.getAlarms(ctx)
	if err != nil {
		sc.logger.Error("failed to get alarms", zap.Error(err))
		return err
	}

	for _, alarm := range alarms {
		// Create event object outside the lock
		event := &securityeventpb.SecurityEvent{
			SecurityEventTime: timestamppb.New(alarm.Time),
			Description:       alarm.Message,
			Id:                alarm.Id,
			Priority:          int32(alarm.Priority),
			Source: &securityeventpb.SecurityEvent_Source{
				Id:        alarm.Source.Id,
				Name:      alarm.Source.Name,
				Subsystem: "acs",
			},
		}

		// Atomically check and update within single critical section
		sc.mu.Lock()
		shouldAdd := alarm.Time.After(sc.lastEventTime)
		if shouldAdd {
			sc.securityEvents.Value = event
			sc.securityEvents = sc.securityEvents.Next()
			sc.lastEventTime = alarm.Time

			// Add to front of list (newest first) for efficient pagination
			sc.eventsList = append([]*securityeventpb.SecurityEvent{event}, sc.eventsList...)
			// Trim to maxEvents if exceeded
			if len(sc.eventsList) > sc.maxEvents {
				sc.eventsList = sc.eventsList[:sc.maxEvents]
			}
		}
		sc.mu.Unlock()

		// Send to channel outside the lock to avoid blocking while holding mutex
		if shouldAdd {
			sc.updates.Send(ctx, &securityeventpb.PullSecurityEventsResponse_Change{
				ChangeTime: timestamppb.Now(),
				OldValue:   nil,
				NewValue:   event,
			})

			sc.logger.Info("adding new security event", zap.Time("time", alarm.Time), zap.String("message", alarm.Message))
		}
	}
	return nil
}

// run the alarm controller schedule to refresh the alarms
func (sc *SecurityEventController) run(ctx context.Context, schedule *jsontypes.Schedule) error {

	err := sc.refreshAlarms(ctx)
	if err != nil {
		sc.logger.Error("failed to refresh alarms, will try again on next run...", zap.Error(err))
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

		err := sc.refreshAlarms(ctx)
		if err != nil {
			sc.logger.Error("failed to refresh alarms, will try again on next run...", zap.Error(err))
		}
	}
}

func (sc *SecurityEventController) ListSecurityEvents(_ context.Context, req *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {

	// Validate page size first (before acquiring lock)
	if req.PageSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid page size")
	}

	pageSize := int(req.PageSize)
	if pageSize > 1000 {
		pageSize = 1000
	}

	// Parse page token (offset into the list)
	offset := 0
	if req.PageToken != "" {
		parsed, err := strconv.ParseInt(req.PageToken, 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = int(parsed)
		if offset < 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	totalEvents := len(sc.eventsList)

	// Validate offset
	if offset > totalEvents {
		return nil, status.Error(codes.InvalidArgument, "invalid page token")
	}

	// Calculate end position
	end := offset + pageSize
	if end > totalEvents {
		end = totalEvents
	}

	// Efficiently slice the events (O(1) access)
	var response securityeventpb.ListSecurityEventsResponse
	response.SecurityEvents = make([]*securityeventpb.SecurityEvent, end-offset)
	copy(response.SecurityEvents, sc.eventsList[offset:end])
	response.TotalSize = int32(totalEvents)

	// Set next page token if there are more events
	if end < totalEvents {
		response.NextPageToken = strconv.FormatInt(int64(end), 10)
	}

	return &response, nil
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
