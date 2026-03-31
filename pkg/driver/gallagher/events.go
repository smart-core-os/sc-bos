package gallagher

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"go.uber.org/zap"
)

type EventPayload struct {
	Href     string    `json:"href"`
	Id       string    `json:"id"`
	Time     time.Time `json:"time"`
	Message  string    `json:"message"`
	Priority int       `json:"priority"`
	Source   struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Href string `json:"href"`
	} `json:"source"`
	Type struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"type"`
	EventType struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"eventType"`
}

type EventList struct {
	Events []EventPayload `json:"events"`
	Next   *struct {
		Href string `json:"href"`
	} `json:"next,omitempty"`
}

// getEvents fetches events from the Gallagher /events API after lastEventTime, returning oldest-first.
func (sc *SecurityEventController) getEvents() ([]*EventPayload, error) {
	var result []*EventPayload
	url := sc.client.getUrl("api/events") + "?after=" + sc.lastEventTime.Format(time.RFC3339)

	for {
		body, err := sc.client.doRequest(url)
		if err != nil {
			sc.logger.Error("failed to get events", zap.Error(err))
			return nil, err
		}

		var eventList EventList
		err = json.Unmarshal(body, &eventList)
		if err != nil {
			sc.logger.Error("failed to decode event list", zap.Error(err))
			return nil, err
		}

		for _, e := range eventList.Events {
			ep := e
			result = append(result, &ep)
		}

		if eventList.Next == nil || eventList.Next.Href == "" {
			break
		}
		url = eventList.Next.Href
	}

	slices.SortFunc(result, func(i, j *EventPayload) int {
		if i.Time.Before(j.Time) {
			return -1
		} else if i.Time.After(j.Time) {
			return 1
		}
		return 0
	})

	return result, nil
}

// refreshEvents fetches new events from the Gallagher API and adds them to the shared ring buffer.
func (sc *SecurityEventController) refreshEvents(ctx context.Context) error {
	events, err := sc.getEvents()
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	for _, e := range events {
		if !e.Time.After(sc.lastEventTime) {
			break
		}
		event := newSecurityEvent(e.Time, e.Id, e.Message, e.Priority, e.Source.Id, e.Source.Name)
		sc.addSecurityEvent(ctx, event)
		sc.lastEventTime = e.Time
		sc.logger.Info("adding new security event from events API", zap.Time("time", e.Time), zap.String("message", e.Message))
	}
	return nil
}
