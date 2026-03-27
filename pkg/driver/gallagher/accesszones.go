package gallagher

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
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
	ZoneCount   int      `json:"zoneCount,omitempty"`
}

type AccessZone struct {
	accesspb.UnimplementedAccessApiServer
	config.ScDevice
	AccessZonePayload
	lastAccessAttempt *resource.Value // of *accesspb.AccessAttempt
	undo              []node.Undo
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
				lastAccessAttempt: resource.NewValue(resource.WithInitialValue(&accesspb.AccessAttempt{}), resource.WithNoDuplicates()),
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
}

// statusFlagsToGrant maps Gallagher access zone status flags to an AccessAttempt Grant value.
// A zone in lockDown denies entry; all other states (free, secure, codeOrCard, dualAuth) grant access.
func statusFlagsToGrant(flags []string) accesspb.AccessAttempt_Grant {
	for _, f := range flags {
		if f == "lockDown" {
			return accesspb.AccessAttempt_DENIED
		}
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
			z.ScName = path.Join(scNamePrefix, "access_zones", z.Id)
			z.Meta = &traits.Metadata{
				Appearance: &traits.Metadata_Appearance{
					Title:       z.Name,
					Description: z.Description,
				},
				Membership: &traits.Metadata_Membership{
					Subsystem: "acs",
				},
			}
			z.undo = append(z.undo, announcer.Announce(z.ScName, node.HasTrait(accesspb.TraitName, node.WithClients(accesspb.WrapApi(z)))))
			z.undo = append(z.undo, announcer.Announce(z.ScName, node.HasMetadata(z.Meta)))
			azc.zones[id] = z
		}
		azc.getAccessZoneDetails(azc.zones[id])
	}

	// unannounce removed zones
	for id, z := range azc.zones {
		if _, ok := zones[id]; !ok {
			azc.logger.Info("unannouncing access zone", zap.String("id", id))
			for _, undo := range z.undo {
				undo()
			}
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
