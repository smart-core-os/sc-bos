package airthings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/local"
	"github.com/smart-core-os/sc-bos/pkg/util/chans"
)

func (d *Driver) listLocations(ctx context.Context) (api.GetLocationsResponse, error) {
	url := d.cfg.URL("/v1/locations")
	var locations api.GetLocationsResponse
	err := d.get(ctx, url, &locations)
	if err != nil {
		return locations, fmt.Errorf("list locations %w", err)
	}
	return locations, nil
}

func (d *Driver) hydrateLocation(ctx context.Context, location config.Location) (config.Location, error) {
	if location.ID != "" {
		return location, nil
	}
	if location.Name == "" {
		return location, fmt.Errorf("one of ID or Name must be set")
	}

	d.listLocationsOnce.Do(func() {
		d.locations, d.locationsErr = d.listLocations(ctx)
		if d.locationsErr != nil {
			d.listLocationsOnce = sync.Once{} // try again next time
		}
	})

	if d.locationsErr != nil {
		return location, d.locationsErr
	}

	clean := func(s string) string {
		return strings.TrimSpace(strings.ToLower(s))
	}
	cn := clean(location.Name)
	for _, l := range d.locations.GetLocations() {
		if clean(l.Name) == cn {
			location.ID = l.Id
			break
		}
	}

	return location, nil
}

// pollLocationsLatestSamples polls the latest samples for the given location.
// Results will be placed into dst.
func (d *Driver) pollLocationLatestSamples(ctx context.Context, location config.Location, dst *local.Location) error {
	location, err := d.hydrateLocation(ctx, location)
	if err != nil {
		return fmt.Errorf("hydrate location %w", err)
	}
	if location.ID == "" {
		return fmt.Errorf("location not found")
	}
	url := d.cfg.URL("/v1/locations/%v/latest-samples", location.ID)

	delay := location.Poll.Or(config.DefaultPoll)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		var samples api.GetLocationSamplesResponseEnriched
		err := d.get(ctx, url, &samples)
		switch {
		case err != nil:
			d.Logger.Warn("get location samples failed",
				zap.String("location.id", location.ID), zap.String("location.name", location.Name), zap.Error(err))
		default:
			c := dst.UpdateLatestSamples(samples)
			// ensure backpressure
			if _, err := chans.RecvContext(ctx, c); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (d *Driver) get(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request %w", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("do get %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("bad response %v %v, failed to read body %v", res.StatusCode, res.Status, err)
		}
		return fmt.Errorf("bad response %v %v\n%v", res.StatusCode, res.Status, string(body))
	}
	return json.NewDecoder(res.Body).Decode(v)
}
