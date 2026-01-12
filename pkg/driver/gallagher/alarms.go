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

	client *Client
	logger *zap.Logger
	mu     sync.Mutex
	// security events is a circular buffer, it always points to the oldest security event
	lastEventTime  time.Time  // *securityeventpb.SecurityEvent
	securityEvents *ring.Ring // *securityeventpb.SecurityEvent
	updates        minibus.Bus[*securityeventpb.PullSecurityEventsResponse_Change]
}

func newSecurityEventController(client *Client, logger *zap.Logger, n int) *SecurityEventController {
	return &SecurityEventController{
		client:         client,
		logger:         logger,
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

// refreshAlarms call the Gallagher alarms API and add any new ones to the sc that are newer than our current newest
func (sc *SecurityEventController) refreshAlarms(ctx context.Context) error {
	alarms, err := sc.getAlarms()
	if err != nil {
		sc.logger.Error("failed to get alarms", zap.Error(err))
		return err
	}

	for _, alarm := range alarms {
		// we only want to add new alarms
		if alarm.Time.After(sc.lastEventTime) {
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
			sc.securityEvents.Value = event
			sc.securityEvents = sc.securityEvents.Next()
			sc.updates.Send(ctx, &securityeventpb.PullSecurityEventsResponse_Change{
				ChangeTime: timestamppb.Now(),
				OldValue:   nil,
				NewValue:   event,
			})
			// the events in alarms are always oldest first, so this is fine
			sc.lastEventTime = alarm.Time
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

		sc.mu.Lock()
		err := sc.refreshAlarms(ctx)
		sc.mu.Unlock()
		if err != nil {
			sc.logger.Error("failed to refresh alarms, will try again on next run...", zap.Error(err))
		}
	}
}

func (sc *SecurityEventController) ListSecurityEvents(_ context.Context, req *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {

	nextPageToken := ""
	start := sc.securityEvents.Len() - 1

	if req.PageToken != "" {
		s, err := strconv.ParseInt(req.PageToken, 10, 64)
		if err != nil {
			return nil, err
		}
		start = int(s)
		if start < 0 || start >= sc.securityEvents.Len() {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
	}

	if req.PageSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid page size")
	} else if req.PageSize > int32(sc.securityEvents.Len()) {
		req.PageSize = int32(sc.securityEvents.Len())
	}

	if req.PageSize > 1000 {
		req.PageSize = 1000
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()
	var response securityeventpb.ListSecurityEventsResponse
	// get the most recent ones first as that is what the UI expects
	for i := start; i >= 0; i-- {
		se := sc.securityEvents.Move(i)
		if se.Value != nil {
			response.SecurityEvents = append(response.SecurityEvents, se.Value.(*securityeventpb.SecurityEvent))
		}
		if len(response.SecurityEvents) >= int(req.PageSize) {
			nextPageToken = strconv.FormatInt(int64(i-1), 10)
			break
		}
	}

	response.NextPageToken = nextPageToken
	response.TotalSize = int32(sc.securityEvents.Len())
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
