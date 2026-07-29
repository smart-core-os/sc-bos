package paxton

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/securityeventpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type EventsController struct {
	securityeventpb.UnimplementedSecurityEventApiServer

	driver *Driver
	// client is captured at construction rather than read live from driver.client.
	// applyConfig reassigns driver.client on every reconfigure, and ListSecurityEvents
	// is driven by external gRPC callers that are not fenced by the prevDone handshake,
	// so reaching through driver.client here would race that write. The controller is
	// rebuilt per generation alongside the client, so this reference stays consistent.
	client *Client

	latestState *resource.Value
}

func NewEventsController(d *Driver) *EventsController {
	return &EventsController{
		driver:      d,
		client:      d.client,
		latestState: resource.NewValue(resource.WithNoDuplicates()),
	}
}

func (e *EventsController) processSecurityEvents(ctx context.Context, events []Event) error {
	for _, event := range events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		source := &securityeventpb.SecurityEvent_Source{Subsystem: DriverName}
		eventType := convertEventToEventTypeEnum(event)
		if !e.sourceEvent(event, source) {
			e.driver.logger.Debug("cannot source event", zap.String("desc", event.EventDescription))
		}
		se := paxtonToSmartcoreEvent(event, source, eventType)
		if _, err := e.latestState.Set(&securityeventpb.PullSecurityEventsResponse_Change{
			ChangeTime: timestamppb.New(event.EventTime),
			NewValue:   se,
			Type:       typespb.ChangeType_ADD,
		}); err != nil {
			e.driver.logger.Error("failed to set latest event", zap.Error(err))
			return err
		}
	}

	return nil
}

// ListSecurityEvents returns a slice of the security events belonging to doors or cardholders
// sorted by time ascending; returning the latest events in the first page as that's what the UI expects.
// Use the response.NextPageToken to retrieve older event pages
func (e *EventsController) ListSecurityEvents(ctx context.Context, request *securityeventpb.ListSecurityEventsRequest) (*securityeventpb.ListSecurityEventsResponse, error) {
	returnLength := 50

	if request.GetPageSize() > 0 {
		returnLength = min(int(request.GetPageSize()), 1000)
	}

	sendPageToken := ""

	timeFilter := time.Now()

	if request.GetPageToken() != "" {
		var err error
		timeFilter, err = time.Parse(time.RFC3339, request.GetPageToken())

		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}

	}

	var ret []*securityeventpb.SecurityEvent
	totalLength := 0
	var oldestReturned time.Time

	appendEvent := func(event Event) {
		source := &securityeventpb.SecurityEvent_Source{Subsystem: DriverName}
		eventType := convertEventToEventTypeEnum(event)
		if !e.sourceEvent(event, source) {
			e.driver.logger.Debug("cannot source event", zap.String("desc", event.EventDescription))
		}
		ret = append(ret, paxtonToSmartcoreEvent(event, source, eventType))
		oldestReturned = event.EventTime
	}

	for {
		events, err := e.client.GetEvents(ctx, timeFilter, true)

		if err != nil {
			return nil, err
		}

		if len(events) < 1 {
			break
		}
		totalLength += len(events)

		full := false
		for i := range events {
			appendEvent(events[i])
			if len(ret) >= returnLength {
				// The API time filter has second granularity, so the next page queries
				// strictly older than oldestReturned's second. Drain any remaining events
				// in this batch that share that second so they are not skipped at the
				// boundary. (A second spanning a batch boundary is a residual limitation.)
				sec := events[i].EventTime.Truncate(time.Second)
				for j := i + 1; j < len(events) && events[j].EventTime.Truncate(time.Second).Equal(sec); j++ {
					appendEvent(events[j])
				}
				full = true
				break
			}
		}

		// Prevent an infinite loop when the whole batch shares timeFilter's timestamp.
		oldestInBatch := events[len(events)-1].EventTime
		if oldestInBatch.Equal(timeFilter) {
			break
		}
		if full {
			break
		}

		// GetEvents returns events in time descending order, so page to older events.
		timeFilter = oldestInBatch
	}

	// The page token is the oldest event actually returned (not merely fetched) so the
	// next page resumes exactly where this one ended without skipping events between the
	// page boundary and the end of the last batch.
	if !oldestReturned.IsZero() {
		sendPageToken = oldestReturned.Format(time.RFC3339)
	}

	return &securityeventpb.ListSecurityEventsResponse{
		SecurityEvents: ret,
		NextPageToken:  sendPageToken,
		TotalSize:      int32(totalLength),
	}, nil
}

// sources an event
// returns false if event can't be sourced into a door or cardholder
func (e *EventsController) sourceEvent(event Event, source *securityeventpb.SecurityEvent_Source) bool {
	if event.EventDescription == "Operator" ||
		event.EventDescription == "Server" ||
		event.EventDescription == "Backup succeeded" ||
		event.EventDescription == "User details" ||
		event.EventDescription == "Access Level" {
		// these are known internal events
		source.Name = "Paxton Net2"
		return true
	}

	// Prefer the door the event occurred at, matched by address (door ID).
	if door, ok := e.driver.doorByAddress(event.Address); ok {
		source.Id = strconv.Itoa(door.id)
		source.Name = door.name
		// TODO: other source fields to support filtering
		return true
	}

	// Otherwise source it to the cardholder who triggered the event.
	if _, ok := e.driver.cardholders.Load(event.UserID); ok {
		source.Id = strconv.Itoa(event.UserID)
		source.Name = strings.TrimSpace(fmt.Sprintf("%s %s", event.FirstName, event.Surname))
		// TODO: other fields to support filtering
		return true
	}

	return false
}

func (e *EventsController) PullSecurityEvents(request *securityeventpb.PullSecurityEventsRequest, server securityeventpb.SecurityEventApi_PullSecurityEventsServer) error {
	changes := e.latestState.Pull(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly()), resource.WithBackpressure(false))

	for {
		select {
		case latest := <-changes:
			val, ok := latest.Value.(*securityeventpb.PullSecurityEventsResponse_Change)

			if !ok {
				return status.Error(codes.Internal, "alarm received unknown value")
			}

			if err := server.Send(&securityeventpb.PullSecurityEventsResponse{
				Changes: []*securityeventpb.PullSecurityEventsResponse_Change{
					val,
				},
			}); err != nil {
				return err
			}
		case <-server.Context().Done():
			return server.Context().Err()
		}
	}

}

func paxtonToSmartcoreEvent(pe Event, s *securityeventpb.SecurityEvent_Source, eventType securityeventpb.SecurityEvent_EventType) *securityeventpb.SecurityEvent {
	description := ""
	if pe.FirstName != "" {
		description = pe.FirstName + " " + pe.Surname + " - "
	}
	description += pe.EventDescription
	return &securityeventpb.SecurityEvent{
		SecurityEventTime: timestamppb.New(pe.EventTime),
		Description:       description,
		Id:                fmt.Sprintf("%d", pe.ID),
		EventType:         eventType,
		Priority:          int32(pe.Priority),
		Source:            s,
	}
}
