package paxton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

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

type Cardholder struct {
	accesspb.UnimplementedAccessApiServer

	id         int
	deviceName string

	undo []node.Undo

	state *resource.Value
}

func (c *Cardholder) GetLastAccessAttempt(_ context.Context, request *accesspb.GetLastAccessAttemptRequest) (*accesspb.AccessAttempt, error) {
	value := c.state.Get(resource.WithReadMask(request.GetReadMask()))

	access, ok := value.(*accesspb.AccessAttempt)

	if !ok {
		return nil, status.Error(codes.Internal, "cannot convert state to AccessAttempt")
	}

	return access, nil
}

func (c *Cardholder) PullAccessAttempts(request *accesspb.PullAccessAttemptsRequest, server accesspb.AccessApi_PullAccessAttemptsServer) error {
	for value := range c.state.Pull(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly())) {
		accessAttempt, ok := value.Value.(*accesspb.AccessAttempt)

		if !ok {
			return status.Error(codes.Internal, "cannot convert state to AccessAttempt")
		}

		if err := server.Send(&accesspb.PullAccessAttemptsResponse{
			Changes: []*accesspb.PullAccessAttemptsResponse_Change{
				{
					Name:          c.deviceName,
					ChangeTime:    timestamppb.New(value.ChangeTime),
					AccessAttempt: accessAttempt,
				},
			},
		}); err != nil {
			return err
		}
	}

	return nil
}

func (d *Driver) refreshCardholders(ctx context.Context, announcer node.Announcer, cardholderPrefix string) error {
	users, err := d.client.GetUsers(ctx)

	if err != nil {
		return err
	}

	// announce cardholders found in the API and add to map
	for _, user := range users {
		if _, ok := d.cardholders.Load(user.ID); !ok {
			scName := path.Join(cardholderPrefix, "cardholder", strconv.Itoa(user.ID))

			meta := &metadatapb.Metadata{
				Appearance: &metadatapb.Metadata_Appearance{
					Title:       fmt.Sprintf("Cardholder: %s %s", user.FirstName, user.LastName),
					Description: user.Telephone,
				},
				Membership: &metadatapb.Metadata_Membership{
					Subsystem: "acs",
				},
			}

			cardholder := &Cardholder{
				id:         user.ID,
				deviceName: scName,
				state:      resource.NewValue(resource.WithNoDuplicates()),
			}

			undo := announcer.Announce(scName, node.HasServer(accesspb.RegisterAccessApiServer, accesspb.AccessApiServer(cardholder)), node.HasTrait(accesspb.TraitName))
			cardholder.undo = append(cardholder.undo, undo)

			undo = announcer.Announce(scName, node.HasMetadata(meta))
			cardholder.undo = append(cardholder.undo, undo)

			d.cardholders.Store(user.ID, cardholder)
		}
	}

	// cardholders that exist in the map but not found in the API response need to be un-announced
	d.cardholders.Range(func(key, value any) bool {
		id := key.(int)
		cardHolder := value.(*Cardholder)

		if _, ok := users[id]; !ok {
			d.logger.Debug("un-announcing cardholder", zap.String("name", cardHolder.deviceName))

			for _, undo := range cardHolder.undo {
				undo()
			}

			d.cardholders.Delete(id)
		}

		return true
	})

	return nil
}

func (c *Client) GetUsers(ctx context.Context) (map[int]User, error) {
	reqUrl, err := url.JoinPath(c.baseUrl, "api", "v1", "users")

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

	var users []User

	err = json.NewDecoder(resp.Body).Decode(&users)

	if err != nil {
		return nil, err
	}

	foundUsers := make(map[int]User)

	for _, user := range users {
		foundUsers[user.ID] = user
	}

	return foundUsers, nil
}

// CustomField represents a single custom field.
type CustomField struct {
	ID    int    `json:"Id"`
	Value string `json:"Value"`
}

// User represents the main structure for user information.
type User struct {
	ID                 int           `json:"Id"`
	FirstName          string        `json:"FirstName"`
	LastName           string        `json:"LastName"`
	ExpiryDate         time.Time     `json:"ExpiryDate"`
	ActivateDate       time.Time     `json:"ActivateDate"`
	PIN                string        `json:"PIN"`
	Telephone          string        `json:"Telephone"`
	Extension          string        `json:"Extension"`
	Fax                string        `json:"Fax"`
	IsAntiPassbackUser bool          `json:"IsAntiPassbackUser"`
	IsAlarmUser        bool          `json:"IsAlarmUser"`
	IsLockdownExempt   bool          `json:"IsLockdownExempt"`
	HasImage           bool          `json:"HasImage"`
	CustomFields       []CustomField `json:"CustomFields"`
}
