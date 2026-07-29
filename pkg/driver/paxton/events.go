package paxton

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
)

func (d *Driver) linkEvents(ctx context.Context, events []Event) error {
	for _, event := range events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		grant, code := convertEventToGrant(event)

		// Resolve the cardholder so the door's access attempt actor references the SmartCore node.
		var cardholder *Cardholder
		if event.UserID != 0 {
			if val, ok := d.cardholders.Load(event.UserID); ok {
				cardholder, ok = val.(*Cardholder)
				if !ok {
					return status.Error(codes.Internal, "invalid card holder")
				}
			}
		}

		// Build actor: DisplayName is the human-readable "FirstName Surname".
		// Name is the cardholder's SmartCore device name for cross-trait linking (omitted when unknown).
		actor := &actorpb.Actor{
			DisplayName: fmt.Sprintf("%s %s", event.FirstName, event.Surname),
		}
		if cardholder != nil {
			actor.Name = cardholder.deviceName
		}

		attempt := &accesspb.AccessAttempt{
			Grant:             grant,
			Reason:            fmt.Sprintf("[code]: %s\n[description]: %s", code, event.EventDescription),
			Actor:             actor,
			AccessAttemptTime: timestamppb.New(event.EventTime),
		}

		// Update the door that this event occurred at, matched by address (door ID).
		if door, ok := d.doorByAddress(event.Address); ok {
			if _, err := door.accessState.Set(attempt); err != nil {
				d.logger.Error("failed to set door access attempt", zap.Error(err))
			}
		}

		// Update the cardholder who made the access attempt.
		if cardholder == nil {
			continue
		}
		if _, err := cardholder.state.Set(attempt); err != nil {
			d.logger.Error("failed to set access attempt", zap.Error(err))
		}
	}

	return nil
}

func (c *Client) GetEvents(ctx context.Context, timeFilter time.Time, olderThan bool) ([]Event, error) {
	reqUrl, err := url.JoinPath(c.baseUrl, "api", "v1", "events")

	if err != nil {
		return nil, err
	}

	u, err := url.Parse(reqUrl)

	if err != nil {
		return nil, err
	}

	// The API does not like the timezone being added to the time, so we format it without the timezone.
	filter := fmt.Sprintf("eventTime>='%s'", timeFilter.Format("2006-01-02T15:04:05"))

	if olderThan {
		filter = fmt.Sprintf("eventTime<'%s'", timeFilter.Format("2006-01-02T15:04:05"))
	}

	query := u.Query()
	query.Set("where", filter)
	query.Set("orderBy", "eventTime DESC")
	u.RawQuery = query.Encode()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)

	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req)

	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	var events []Event

	err = json.NewDecoder(resp.Body).Decode(&events)

	if errors.Is(err, io.EOF) {
		// An empty body means there are no events in the queried window; not an error.
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return events, nil
}

// eventKey returns a fingerprint string for an Event used for cross-channel deduplication.
// REST-polled events are also registered under this key so that a SignalR liveEvent carrying
// the same data is recognised as a duplicate even though liveEvents carry no integer ID.
//
// The time is truncated to the second because the REST and SignalR representations may differ
// in sub-second precision. A consequence is that two genuinely distinct events with the same
// (second, type, user, address) collapse to one; this is accepted as rare in practice.
func eventKey(e Event) string {
	return fmt.Sprintf("%s|%d|%d|%d",
		e.EventTime.UTC().Truncate(time.Second).Format(time.RFC3339),
		e.EventType,
		e.UserID,
		e.Address,
	)
}

type Event struct {
	EventTime        time.Time `json:"eventTime"`
	ID               int       `json:"id"`
	DeviceName       string    `json:"deviceName"`
	CardNo           int       `json:"cardNo"`
	EventType        int       `json:"eventType"`
	EventDescription string    `json:"eventDescription"`
	EventSubType     int       `json:"eventSubType"`
	EventDetails     string    `json:"eventDetails"`
	LinkedEvent      int       `json:"linkedEvent"`
	FirstName        string    `json:"firstName"`
	MiddleName       string    `json:"middleName"`
	Surname          string    `json:"surname"`
	UserID           int       `json:"userID"`
	Priority         int       `json:"priority"`
	Address          int       `json:"address"`
	PeripheralID     int       `json:"peripheralID"`
	IOBoardID        int       `json:"ioBoardID"`
	DoorGroupID      int       `json:"doorGroupID"`
	DeviceDeleted    bool      `json:"deviceDeleted"`
}
