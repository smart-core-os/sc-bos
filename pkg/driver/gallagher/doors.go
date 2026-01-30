package gallagher

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type DoorList struct {
	Next *struct {
		Href string `json:"href"`
	} `json:"next,omitempty"`
	Results []DoorPayload `json:"results"`
}

type DoorUdmiInfo struct {
	Id   string `json:"id"`
	Href string `json:"href"`
	Name string `json:"name"`
}

type DoorPayload struct {
	Id          string `json:"id"`
	Href        string `json:"href"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Door struct {
	config.ScDevice
	DoorPayload
	Undo []node.Undo
}

type DoorController struct {
	client      *Client
	topicPrefix string
	doors       map[string]*Door
	logger      *zap.Logger
}

func newDoorController(client *Client, topicPrefix string, logger *zap.Logger) *DoorController {
	return &DoorController{
		client:      client,
		doors:       make(map[string]*Door),
		logger:      logger,
		topicPrefix: topicPrefix,
	}
}

// getDoors gets top level list of all the doors from the Gallagher API
func (dc *DoorController) getDoors() (map[string]*Door, error) {

	result := make(map[string]*Door)
	url := dc.client.getUrl("doors")
	for {
		body, err := dc.client.doRequest(url)
		if err != nil {
			return nil, err
		}

		var resultsList DoorList
		err = json.Unmarshal(body, &resultsList)
		if err != nil {
			dc.logger.Error("failed to decode door list", zap.Error(err))
			return nil, err
		}

		for _, door := range resultsList.Results {
			result[door.Id] = &Door{
				DoorPayload: door,
			}
			dc.getDoorDetails(result[door.Id])
		}

		if resultsList.Next == nil || resultsList.Next.Href == "" {
			break
		} else {
			url = resultsList.Next.Href
		}
	}

	return result, nil
}

// getDoorDetails gets the full details for each door
func (dc *DoorController) getDoorDetails(door *Door) {

	resp, err := dc.client.doRequest(door.Href)
	if err != nil {
		dc.logger.Error("failed to get door", zap.Error(err))
		return
	}

	err = json.Unmarshal(resp, door)
	if err != nil {
		dc.logger.Error("failed to decode door", zap.Error(err))
	}
}

// refreshDoors get the list of doors and compare it to the previous list. Announce any new doors
// and undo (unannounce) any that are no longer present. Then update the doors details.
func (dc *DoorController) refreshDoors(announcer node.Announcer, scNamePrefix string) error {

	doors, err := dc.getDoors()
	if err != nil {
		return err
	}

	// look for new doors, add & announce them
	for id, d := range doors {
		if _, ok := dc.doors[id]; !ok {

			d.ScName = path.Join(scNamePrefix, "doors", d.Id)
			d.Meta = &traits.Metadata{
				Appearance: &traits.Metadata_Appearance{
					Title:       "Door: " + d.Name,
					Description: d.Description,
				},
				Membership: &traits.Metadata_Membership{
					Subsystem: "acs",
				},
			}

			d.Undo = append(d.Undo, announcer.Announce(d.ScName, node.HasMetadata(d.Meta)))
			dc.doors[id] = d
		}
	}

	// look for doors that have been removed (unlikely but possible), unannounce them
	for id, c := range dc.doors {
		if _, ok := doors[id]; !ok {
			dc.logger.Info("unannouncing door", zap.String("id", id))
			for _, undo := range c.Undo {
				undo()
			}
			delete(dc.doors, id)
		}
	}
	return nil
}

// run is the main loop for the door controller, it refreshes the doors on a schedule
func (dc *DoorController) run(ctx context.Context, schedule *jsontypes.Schedule, announcer node.Announcer, scNamePrefix string) error {

	t := time.Now()
	for {
		next := schedule.Next(t)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
			t = next
		}

		err := dc.refreshDoors(announcer, scNamePrefix)
		if err != nil {
			dc.logger.Error("failed to refresh doors, will try again on next run...", zap.Error(err))
		}
	}
}
