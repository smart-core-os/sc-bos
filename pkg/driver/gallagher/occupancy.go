package gallagher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
)

type OccupancyEventController struct {
	traits.OccupancySensorApiServer

	client   *client
	interval time.Duration

	lastRefreshCycle time.Time
	bootupTime       time.Time

	logger *zap.Logger

	mu               sync.Mutex
	totalPeopleCount int32

	notifyPull chan struct{}
}

func newOccupancyEventController(client *client, logger *zap.Logger, interval time.Duration) *OccupancyEventController {
	return &OccupancyEventController{
		client:     client,
		bootupTime: time.Now(),
		interval:   interval,
		logger:     logger,
		notifyPull: make(chan struct{}),
	}
}

func (o *OccupancyEventController) run(ctx context.Context) error {
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case latest := <-ticker.C:
			if err := o.refresh(ctx); err != nil {
				continue
			}
			o.lastRefreshCycle = latest

			select {
			case o.notifyPull <- struct{}{}:
			default:
			}
		}
	}
}

func (o *OccupancyEventController) refresh(ctx context.Context) error {

	reqUrl := fmt.Sprintf("%s?after=%s&fields=source&top=1000", o.client.getUrl("events"), o.lastRefreshCycle.Format(time.RFC3339))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		bytes, err := o.client.doRequest(ctx, reqUrl)

		if err != nil {
			o.logger.Error("failed to fetch events", zap.Error(err))
			return err
		}

		resp := &EventUpdateResponse{}

		err = json.Unmarshal(bytes, resp)
		if err != nil {
			o.logger.Error("failed to unmarshal events", zap.Error(err))
			return err
		}

		for _, ev := range resp.Events {
			if strings.Contains(strings.ToLower(ev.Source.Name), "speedgate") {
				o.mu.Lock()
				if strings.Contains(strings.ToLower(ev.Source.Name), "- out") {
					o.totalPeopleCount--
				}
				if strings.Contains(strings.ToLower(ev.Source.Name), "- in") {
					o.totalPeopleCount++
				}
				o.mu.Unlock()
			}
		}

		if resp.Next.Href == "" {
			break
		} else {
			reqUrl = resp.Next.Href
		}
	}

	return nil
}

type EventUpdateResponse struct {
	Events   []EventUpdate   `json:"events"`
	Previous EventUpdateLink `json:"previous"`
	Next     EventUpdateLink `json:"next"`
	Updates  EventUpdateLink `json:"updates"`
}

// EventUpdate isn't complete, I have only included what this functionality requires for occupancy counting
type EventUpdate struct {
	Source EventUpdateSource `json:"source"`
}

type EventUpdateLink struct {
	Href string `json:"href"`
}

type EventUpdateSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Href string `json:"href"`
}

func (o *OccupancyEventController) loadOccupancyCount() (int32, traits.Occupancy_State, float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	occ := o.totalPeopleCount
	state := traits.Occupancy_OCCUPIED
	if occ <= 0 {
		occ = 0
		state = traits.Occupancy_UNOCCUPIED
		// this is a hack to get around the building being occupied by people when the controller boots up
		// as people keep leaving, the count will converge to 0
		// then as people start entering the count increases until they leave
		o.totalPeopleCount = 0
	}

	confidence := 1.

	if time.Now().Before(o.bootupTime.Add(24 * time.Hour)) {
		// if we're in the first day of running
		// then we want to check the boot-up time
		// otherwise it's assumed the convergence to 0 has worked,
		// and we are counting people entering/leaving the building correctly against the total count
		if o.bootupTime.Hour() > 6 && o.bootupTime.Hour() < 20 {
			// the driver's boot-up time was not at night.
			// We therefore, assume at least someone is occupying the building
			// and a starting count of 0 is false (low confidence)
			confidence = 0.
		}
	}

	return occ, state, confidence
}

func (o *OccupancyEventController) GetOccupancy(_ context.Context, _ *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	count, state, confidence := o.loadOccupancyCount()

	return &traits.Occupancy{
		State:           state,
		PeopleCount:     count,
		StateChangeTime: timestamppb.New(o.lastRefreshCycle),
		Confidence:      confidence,
	}, nil
}
func (o *OccupancyEventController) PullOccupancy(_ *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	for {
		select {
		case <-o.notifyPull:
			count, state, confidence := o.loadOccupancyCount()

			if err := server.Send(&traits.PullOccupancyResponse{
				Changes: []*traits.PullOccupancyResponse_Change{
					{
						Occupancy: &traits.Occupancy{
							State:           state,
							PeopleCount:     count,
							Confidence:      confidence,
							StateChangeTime: timestamppb.New(o.lastRefreshCycle),
						},
					},
				}}); err != nil {
				o.logger.Error("failed to send occupancy", zap.Error(err))
				return err
			}
		case <-server.Context().Done():
			return server.Context().Err()
		}
	}
}
