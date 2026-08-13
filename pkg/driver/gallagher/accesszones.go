package gallagher

import (
	"context"
	"encoding/json"
	"math"
	"path"
	"slices"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type AccessZoneList struct {
	Next *struct {
		Href string `json:"href"`
	} `json:"next,omitempty"`
	Results []AccessZonePayload `json:"results"`
}

type AccessZonePayload struct {
	Id          string   `json:"id"`
	Href        string   `json:"href"`
	Name        string   `json:"name"`
	ShortName   string   `json:"shortName,omitempty"`
	Description string   `json:"description,omitempty"`
	StatusFlags []string `json:"statusFlags,omitempty"`
	Status      string   `json:"status,omitempty"`
	// ZoneCount is how many cardholders Gallagher currently counts inside the zone.
	// It's a pointer because Command Centre only sends it for zones with zone counting
	// enabled, and an absent count has to be distinguishable from a count of zero.
	ZoneCount *int `json:"zoneCount,omitempty"`
}

type AccessZone struct {
	accesspb.UnimplementedAccessApiServer
	config.ScDevice
	AccessZonePayload
	lastAccessAttempt *resource.Value          // of *accesspb.AccessAttempt
	occupancy         *occupancysensorpb.Model // nil until Gallagher reports a zone count
	// occupancyUndo is held apart from undo because occupancy is the one feature that can
	// come and go while the zone stays, so it has to be revocable on its own.
	occupancyUndo node.Undo
	undo          []node.Undo
}

// unannounce withdraws everything this zone announced, occupancy included.
func (z *AccessZone) unannounce() {
	for _, undo := range z.undo {
		undo()
	}
	z.undo = nil
	z.withdrawOccupancy()
}

type AccessZoneController struct {
	client *Client
	cc     *CardholderController
	zones  map[string]*AccessZone
	logger *zap.Logger
}

func newAccessZoneController(client *Client, cc *CardholderController, logger *zap.Logger) *AccessZoneController {
	return &AccessZoneController{
		client: client,
		cc:     cc,
		zones:  make(map[string]*AccessZone),
		logger: logger,
	}
}

// getAccessZones fetches the paginated list of access zones from the Gallagher API.
func (azc *AccessZoneController) getAccessZones() (map[string]*AccessZone, error) {
	result := make(map[string]*AccessZone)
	url := azc.client.getUrl("access_zones")
	for {
		body, err := azc.client.doRequest(url)
		if err != nil {
			return nil, err
		}

		var list AccessZoneList
		if err = json.Unmarshal(body, &list); err != nil {
			azc.logger.Error("failed to decode access zone list", zap.Error(err))
			return nil, err
		}

		for _, z := range list.Results {
			result[z.Id] = &AccessZone{
				AccessZonePayload: z,
			}
		}

		if list.Next == nil || list.Next.Href == "" {
			break
		}
		url = list.Next.Href
	}
	return result, nil
}

// getAccessZoneDetails fetches and populates full details for the given access zone.
func (azc *AccessZoneController) getAccessZoneDetails(zone *AccessZone) {
	resp, err := azc.client.doRequest(zone.Href)
	if err != nil {
		azc.logger.Error("failed to get access zone details", zap.Error(err), zap.String("href", zone.Href))
		return
	}

	// unmarshalling into the existing zone leaves absent properties at their previous value,
	// so clear the count first: if counting is turned off in Command Centre we want to stop
	// publishing rather than serve the last figure forever.
	zone.ZoneCount = nil
	if err = json.Unmarshal(resp, zone); err != nil {
		azc.logger.Error("failed to decode access zone details", zap.Error(err))
		return
	}

	attempt := &accesspb.AccessAttempt{
		Grant:             statusFlagsToGrant(zone.StatusFlags),
		Reason:            zone.Status,
		AccessAttemptTime: timestamppb.Now(),
	}

	if ch := azc.cc.lastCardholderForZoneHref(zone.Href); ch != nil {
		if t, err := time.Parse(time.RFC3339, ch.LastSuccessfulAccessTime); err == nil {
			attempt.AccessAttemptTime = timestamppb.New(t)
		}
		attempt.Actor = &actorpb.Actor{
			DisplayName: ch.FirstName + " " + ch.LastName,
		}
	}

	_, _ = zone.lastAccessAttempt.Set(attempt)
	zone.setOccupancy()
	// Reached only on a successful fetch and decode, which matters: a transient
	// error must not be read as "counting was turned off".
	azc.withdrawOccupancy(zone)
}

// setOccupancy publishes Gallagher's own zone count as an occupancy reading.
// Command Centre maintains this figure, so unlike counting turnstile events it needs no
// starting reference and survives a driver restart.
//
// StateChangeTime is only moved when the occupied state flips; a steady count carries the
// previous timestamp forward so consumers don't read every poll as a fresh change.
func (z *AccessZone) setOccupancy() {
	if z.ZoneCount == nil || z.occupancy == nil {
		return // zone counting isn't enabled for this zone
	}

	// Clamped rather than converted straight through. PeopleCount is an int32 and the
	// count arrives as a plain JSON number, so a garbled response could otherwise wrap
	// a large value round to a negative headcount.
	count := min(max(*z.ZoneCount, 0), math.MaxInt32)

	occupancy := &occupancysensorpb.Occupancy{
		PeopleCount:     int32(count),
		State:           occupancysensorpb.Occupancy_UNOCCUPIED,
		Confidence:      1,
		StateChangeTime: timestamppb.Now(),
	}
	if count > 0 {
		occupancy.State = occupancysensorpb.Occupancy_OCCUPIED
	}
	if prev, err := z.occupancy.GetOccupancy(); err == nil &&
		prev.State == occupancy.State && prev.StateChangeTime != nil {
		occupancy.StateChangeTime = prev.StateChangeTime
	}

	_, _ = z.occupancy.SetOccupancy(occupancy)
}

// announceOccupancy adds the OccupancySensor trait to a zone the first time Gallagher reports
// a zone count for it. Zones without zone counting enabled never get the trait, which keeps
// the estate free of sensors that would sit at zero forever.
//
// It's called after the details fetch rather than alongside the zone's other features because
// only the details response tells us whether a count exists.
func (azc *AccessZoneController) announceOccupancy(zone *AccessZone, announcer node.Announcer) {
	if zone.ZoneCount == nil || zone.occupancy != nil {
		return
	}

	zone.occupancy = occupancysensorpb.NewModel(resource.WithNoDuplicates())
	zone.setOccupancy() // seed the model with the count we just fetched
	zone.occupancyUndo = announcer.Announce(zone.ScName,
		node.HasServer(occupancysensorpb.RegisterOccupancySensorApiServer,
			occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(zone.occupancy))),
		node.HasTrait(trait.OccupancySensor),
	)
	azc.logger.Debug("announced occupancy for access zone",
		zap.String("name", zone.ScName), zap.Int("zoneCount", *zone.ZoneCount))
}

// withdrawOccupancy removes the OccupancySensor trait from a zone that has stopped
// reporting a count, which is what happens when zone counting is turned off in Command
// Centre.
//
// This is the other half of clearing ZoneCount before unmarshalling the details response.
// That clear exists so an absent count is distinguishable from an unchanged one, but on
// its own it achieved nothing: setOccupancy simply returns when the count is nil, so the
// model kept its last value and the trait stayed announced. A zone that had seven people
// in it when counting was disabled went on reporting seven people indefinitely, which is
// worse than reporting nothing because it looks like a live figure.
//
// announceOccupancy is lazy and idempotent, so a count that comes back is picked up again
// on the next refresh.
func (azc *AccessZoneController) withdrawOccupancy(zone *AccessZone) {
	if zone.ZoneCount != nil || zone.occupancy == nil {
		return
	}

	zone.withdrawOccupancy()
	azc.logger.Debug("withdrew occupancy for access zone, no zone count reported",
		zap.String("name", zone.ScName))
}

// withdrawOccupancy drops the zone's occupancy model and unannounces its trait, if it has
// one. Safe to call on a zone that never had one.
func (z *AccessZone) withdrawOccupancy() {
	if z.occupancyUndo != nil {
		z.occupancyUndo()
		z.occupancyUndo = nil
	}
	z.occupancy = nil
}

// statusFlagsToGrant maps Gallagher access zone status flags to an AccessAttempt Grant value.
// A zone in lockDown denies entry; all other states (free, secure, codeOrCard, dualAuth) grant access.
func statusFlagsToGrant(flags []string) accesspb.AccessAttempt_Grant {
	if slices.Contains(flags, "lockDown") {
		return accesspb.AccessAttempt_DENIED
	}
	return accesspb.AccessAttempt_GRANTED
}

// refreshAccessZones fetches the current zone list, announces new zones, updates existing ones,
// and unannounces zones that have been removed.
func (azc *AccessZoneController) refreshAccessZones(announcer node.Announcer, scNamePrefix string) error {
	zones, err := azc.getAccessZones()
	if err != nil {
		return err
	}

	// announce new zones
	for id, z := range zones {
		if _, ok := azc.zones[id]; !ok {
			z.lastAccessAttempt = resource.NewValue(resource.WithInitialValue(&accesspb.AccessAttempt{}), resource.WithNoDuplicates())
			z.ScName = path.Join(scNamePrefix, "access_zones", z.Id)
			z.Meta = &metadatapb.Metadata{
				Appearance: &metadatapb.Metadata_Appearance{
					Title:       z.Name,
					Description: z.Description,
				},
				Membership: &metadatapb.Metadata_Membership{
					Subsystem: "acs",
				},
			}
			z.undo = append(z.undo, announcer.Announce(z.ScName,
				node.HasServer(accesspb.RegisterAccessApiServer, accesspb.AccessApiServer(z)),
				node.HasTrait(accesspb.TraitName),
			))
			z.undo = append(z.undo, announcer.Announce(z.ScName, node.HasMetadata(z.Meta), node.HasDeviceType(metadatapb.Metadata_VIRTUAL)))
			azc.zones[id] = z
		}
		azc.getAccessZoneDetails(azc.zones[id])
		azc.announceOccupancy(azc.zones[id], announcer)
	}

	// unannounce removed zones
	for id, z := range azc.zones {
		if _, ok := zones[id]; !ok {
			azc.logger.Info("unannouncing access zone", zap.String("id", id))
			z.unannounce()
			delete(azc.zones, id)
		}
	}
	return nil
}

// run is the main loop for the access zone controller, refreshing zones on a cron schedule.
func (azc *AccessZoneController) run(ctx context.Context, schedule *jsontypes.Schedule, announcer node.Announcer, scNamePrefix string) error {
	t := time.Now()
	for {
		next := schedule.Next(t)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
			t = next
		}

		if err := azc.refreshAccessZones(announcer, scNamePrefix); err != nil {
			azc.logger.Error("failed to refresh access zones, will try again on next run...", zap.Error(err))
		}
	}
}

func (z *AccessZone) GetLastAccessAttempt(_ context.Context, _ *accesspb.GetLastAccessAttemptRequest) (*accesspb.AccessAttempt, error) {
	return z.lastAccessAttempt.Get().(*accesspb.AccessAttempt), nil
}

func (z *AccessZone) PullAccessAttempts(_ *accesspb.PullAccessAttemptsRequest, server accesspb.AccessApi_PullAccessAttemptsServer) error {
	for value := range z.lastAccessAttempt.Pull(server.Context()) {
		err := server.Send(&accesspb.PullAccessAttemptsResponse{Changes: []*accesspb.PullAccessAttemptsResponse_Change{
			{
				Name:          z.ScName,
				ChangeTime:    timestamppb.New(value.ChangeTime),
				AccessAttempt: value.Value.(*accesspb.AccessAttempt),
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}
