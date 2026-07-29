package paxton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/metadatapb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Door struct {
	accesspb.UnimplementedAccessApiServer

	id         int
	name       string
	deviceName string
	undo       []node.Undo

	accessState *resource.Value // AccessAttempt — last access attempt at this door
}

func (d *Door) GetLastAccessAttempt(_ context.Context, request *accesspb.GetLastAccessAttemptRequest) (*accesspb.AccessAttempt, error) {
	value := d.accessState.Get(resource.WithReadMask(request.GetReadMask()))
	attempt, ok := value.(*accesspb.AccessAttempt)
	if !ok {
		return nil, status.Error(codes.Internal, "cannot convert state to AccessAttempt")
	}
	return attempt, nil
}

func (d *Door) PullAccessAttempts(request *accesspb.PullAccessAttemptsRequest, server accesspb.AccessApi_PullAccessAttemptsServer) error {
	for value := range d.accessState.Pull(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		attempt, ok := value.Value.(*accesspb.AccessAttempt)
		if !ok {
			return status.Error(codes.Internal, "cannot convert state to AccessAttempt")
		}
		if err := server.Send(&accesspb.PullAccessAttemptsResponse{
			Changes: []*accesspb.PullAccessAttemptsResponse_Change{
				{
					Name:          d.deviceName,
					ChangeTime:    timestamppb.New(value.ChangeTime),
					AccessAttempt: attempt,
				},
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

// doorByAddress returns the announced Door for an event's address, if known.
//
// The Paxton event "address" field, the door Id returned by /doors, and the SignalR
// deviceId are all assumed to be the same identifier — this is the single place that
// assumption lives, shared by event linking and security-event sourcing. If door access
// attempts fail to link in the field, this assumption is the first thing to verify.
func (d *Driver) doorByAddress(address int) (*Door, bool) {
	if address == 0 {
		return nil, false
	}
	val, ok := d.doors.Load(address)
	if !ok {
		return nil, false
	}
	door, ok := val.(*Door)
	return door, ok
}

func (d *Driver) refreshDoors(ctx context.Context, announcer node.Announcer, deviceNamePrefix string) error {
	doors, err := d.client.GetDoors(ctx)

	if err != nil {
		return err
	}

	// announce doors found in the API and add to map
	for id, door := range doors {
		if _, ok := d.doors.Load(id); !ok {
			scName := path.Join(deviceNamePrefix, "doors", strconv.Itoa(door.Id))

			meta := &metadatapb.Metadata{
				Appearance: &metadatapb.Metadata_Appearance{
					Title: fmt.Sprintf("Door: %s", door.Name),
				},
				Membership: &metadatapb.Metadata_Membership{
					Subsystem: "acs",
				},
			}

			doorServer := &Door{
				id:          door.Id,
				name:        door.Name,
				deviceName:  scName,
				accessState: resource.NewValue(resource.WithNoDuplicates()),
			}

			undo := announcer.Announce(scName, node.HasServer(accesspb.RegisterAccessApiServer, accesspb.AccessApiServer(doorServer)), node.HasTrait(accesspb.TraitName))
			doorServer.undo = append(doorServer.undo, undo)

			undo = announcer.Announce(scName, node.HasMetadata(meta))
			doorServer.undo = append(doorServer.undo, undo)

			d.doors.Store(id, doorServer)
		}
	}

	// doors that exist in the map but not found in the API response need to be un-announced
	d.doors.Range(func(key, value any) bool {
		id := key.(int)
		doorServer := value.(*Door)

		if _, ok := doors[id]; !ok {
			d.logger.Info("un-announcing door", zap.String("name", doorServer.deviceName))

			for _, undo := range doorServer.undo {
				undo()
			}

			d.doors.Delete(id)
		}

		return true
	})

	return nil
}

func (c *Client) GetDoors(ctx context.Context) (map[int]DoorResponse, error) {
	reqUrl, err := url.JoinPath(c.baseUrl, "api", "v1", "doors")

	if err != nil {
		return nil, err
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)

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

	var doors []DoorResponse

	err = json.NewDecoder(resp.Body).Decode(&doors)

	if err != nil {
		return nil, err
	}

	foundDoors := make(map[int]DoorResponse)

	for _, door := range doors {
		foundDoors[door.Id] = door
	}

	return foundDoors, nil
}

type DoorResponse struct {
	Name string `json:"Name"`
	Id   int    `json:"Id"`
}
