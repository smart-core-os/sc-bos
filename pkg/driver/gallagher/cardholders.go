package gallagher

import (
	"context"
	"encoding/json"
	"path"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/gallagher/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/gentrait/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

type CardholderList struct {
	Next *struct {
		Href string `json:"href"`
	} `json:"next,omitempty"`
	Results []CardholderPayload `json:"results"`
}

type LastSuccessfulAccessZone struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type CardholderUdmiInfo struct {
	Id                       string `json:"id,omitempty"`
	FirstName                string `json:"firstName,omitempty"`
	LastName                 string `json:"lastName,omitempty"`
	ShortName                string `json:"shortName,omitempty"`
	Description              string `json:"description,omitempty"`
	Authorised               bool   `json:"authorised,omitempty"`
	LastSuccessfulAccessTime string `json:"lastSuccessfulAccessTime,omitempty"`
	LastSuccessfulAccessZone string `json:"lastSuccessfulAccessZone,omitempty"`
}

type CardholderPayload struct {
	Href                     string                    `json:"href"`
	Id                       string                    `json:"id"`
	FirstName                string                    `json:"firstName"`
	LastName                 string                    `json:"lastName"`
	ShortName                string                    `json:"shortName"`
	Description              string                    `json:"description"`
	Authorised               bool                      `json:"authorised"`
	LastSuccessfulAccessTime string                    `json:"lastSuccessfulAccessTime"`
	LastSuccessfulAccessZone *LastSuccessfulAccessZone `json:"lastSuccessfulAccessZone"`
}

type Cardholder struct {
	gen.UnimplementedAccessApiServer
	gen.UnimplementedUdmiServiceServer
	config.ScDevice
	CardholderPayload
	lastAccessAttempt *resource.Value // gen.AccessAttempt
	udmiBus           minibus.Bus[*gen.PullExportMessagesResponse]
	undo              []node.Undo
}

type CardholderController struct {
	cardholders map[string]*Cardholder
	client      *client
	logger      *zap.Logger
	mu          sync.Mutex
	topicPrefix string
}

func newCardholderController(client *client, topicPrefix string, logger *zap.Logger) *CardholderController {
	return &CardholderController{
		cardholders: make(map[string]*Cardholder),
		client:      client,
		logger:      logger,
		topicPrefix: topicPrefix,
	}
}

// getCardholders gets top level list of all the cardholders from the Gallagher API
func (cc *CardholderController) getCardholders(ctx context.Context) (map[string]*Cardholder, error) {

	result := make(map[string]*Cardholder)
	url := cc.client.getUrl("cardholders")
	for {
		body, err := cc.client.doRequest(ctx, url)
		if err != nil {
			return result, err
		}

		var resultsList CardholderList
		err = json.Unmarshal(body, &resultsList)
		if err != nil {
			cc.logger.Error("failed to decode cardholder list", zap.Error(err))
			return result, err
		}

		for _, cardholder := range resultsList.Results {
			result[cardholder.Id] = &Cardholder{
				CardholderPayload: cardholder,
				lastAccessAttempt: resource.NewValue(resource.WithInitialValue(&gen.AccessAttempt{}), resource.WithNoDuplicates()),
			}
		}

		if resultsList.Next == nil || resultsList.Next.Href == "" {
			break
		} else {
			url = resultsList.Next.Href
		}
	}
	return result, nil
}

// getCardholderDetails gets & populates the full details for the given cardholder
func (cc *CardholderController) getCardholderDetails(ctx context.Context, cardholder *Cardholder) {
	resp, err := cc.client.doRequest(ctx, cardholder.Href)
	if err != nil {
		cc.logger.Error("failed to get cardholder details", zap.Error(err), zap.String("href", cardholder.Href))
		return
	}

	err = json.Unmarshal(resp, cardholder)
	if err != nil {
		cc.logger.Error("failed to decode cardholder", zap.Error(err))
		return
	}

	accessTime, err := time.Parse(time.RFC3339, cardholder.LastSuccessfulAccessTime)
	var accessTimePb *timestamppb.Timestamp
	if err == nil { // the time can be empty so don't fail if it doesn't parse
		accessTimePb = timestamppb.New(accessTime)
	}

	accessZone := ""
	if cardholder.LastSuccessfulAccessZone != nil {
		accessZone = cardholder.LastSuccessfulAccessZone.Name
	}

	_, _ = cardholder.lastAccessAttempt.Set(
		&gen.AccessAttempt{
			Actor: &gen.Actor{
				Name:          cardholder.FirstName + " " + cardholder.LastName,
				Title:         cardholder.Description,
				LastGrantTime: accessTimePb,
				LastGrantZone: accessZone,
			},
		})
}

// refreshCardholders get the list of cardholders and compare it to the previous list. Announce any new cardholders
// and undo (unannounce) any that are no longer present. Then update the cardholder details.
func (cc *CardholderController) refreshCardholders(ctx context.Context, announcer node.Announcer, scNamePrefix string) error {

	cc.mu.Lock()
	defer cc.mu.Unlock()
	cardholders, err := cc.getCardholders(ctx)
	if err != nil {
		return err
	}

	// look for new cardholders, add & announce them
	for id, c := range cardholders {
		if _, ok := cc.cardholders[id]; !ok {
			c.ScName = path.Join(scNamePrefix, "cardholders", c.Id)
			c.Meta = &traits.Metadata{
				Appearance: &traits.Metadata_Appearance{
					Title:       "Cardholder: " + c.FirstName + " " + c.LastName,
					Description: c.Description,
				},
				Membership: &traits.Metadata_Membership{
					Subsystem: "acs",
				},
			}
			c.undo = append(c.undo, announcer.Announce(c.ScName, node.HasTrait(accesspb.TraitName, node.WithClients(gen.WrapAccessApi(c)))))
			c.undo = append(c.undo, announcer.Announce(c.ScName, node.HasTrait(udmipb.TraitName, node.WithClients(gen.WrapUdmiService(c)))))
			c.undo = append(c.undo, announcer.Announce(c.ScName, node.HasMetadata(c.Meta)))
			cc.cardholders[id] = c
		}
		cc.getCardholderDetails(ctx, c)
	}

	// look for cardholders that have been removed, unannounce them
	for id, c := range cc.cardholders {
		if _, ok := cardholders[id]; !ok {
			cc.logger.Info("unannouncing cardholder", zap.String("id", id))
			for _, undo := range c.undo {
				undo()
			}
			delete(cc.cardholders, id)
		}
	}
	return nil
}

// run is the main loop for the cardholder controller, it refreshes the cardholders on a schedule
func (cc *CardholderController) run(ctx context.Context, schedule *jsontypes.Schedule, announcer node.Announcer, scNamePrefix string) error {

	err := cc.refreshCardholders(ctx, announcer, scNamePrefix)
	if err != nil {
		cc.logger.Error("failed to refresh cardholders, will try again on next run...", zap.Error(err))
	}

	t := time.Now()
	for {
		next := schedule.Next(t)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Until(next)):
			t = next
		}

		err := cc.refreshCardholders(ctx, announcer, scNamePrefix)
		if err != nil {
			cc.logger.Error("failed to refresh cardholders, will try again on next run...", zap.Error(err))
		}
	}
}

func (c *Cardholder) GetLastAccessAttempt(context.Context, *gen.GetLastAccessAttemptRequest) (*gen.AccessAttempt, error) {
	value := c.lastAccessAttempt.Get()
	access := value.(*gen.AccessAttempt)
	return access, nil
}

func (c *Cardholder) PullAccessAttempts(_ *gen.PullAccessAttemptsRequest, server gen.AccessApi_PullAccessAttemptsServer) error {
	for value := range c.lastAccessAttempt.Pull(server.Context()) {
		accessAttempt := value.Value.(*gen.AccessAttempt)
		err := server.Send(&gen.PullAccessAttemptsResponse{Changes: []*gen.PullAccessAttemptsResponse_Change{
			{
				Name:          c.ScName,
				ChangeTime:    timestamppb.New(value.ChangeTime),
				AccessAttempt: accessAttempt,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cardholder) PullExportMessages(_ *gen.PullExportMessagesRequest, server gen.UdmiService_PullExportMessagesServer) error {
	for msg := range c.udmiBus.Listen(server.Context()) {
		err := server.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cardholder) PullControlTopics(_ *gen.PullControlTopicsRequest, topicsServer gen.UdmiService_PullControlTopicsServer) error {
	<-topicsServer.Context().Done()
	return nil
}

func (c *Cardholder) OnMessage(context.Context, *gen.OnMessageRequest) (*gen.OnMessageResponse, error) {
	return &gen.OnMessageResponse{}, nil
}

func (cc *CardholderController) sendUdmiMessages(ctx context.Context) {

	cc.mu.Lock()
	snapshot := make([]*Cardholder, 0, len(cc.cardholders))
	for _, c := range cc.cardholders {
		snapshot = append(snapshot, c)
	}
	cc.mu.Unlock()

	for _, c := range snapshot {
		zoneName := ""
		if c.LastSuccessfulAccessZone != nil {
			zoneName = c.LastSuccessfulAccessZone.Name
		}

		payload, err := json.Marshal(CardholderUdmiInfo{
			Id:                       c.Id,
			FirstName:                c.FirstName,
			LastName:                 c.LastName,
			ShortName:                c.ShortName,
			Description:              c.Description,
			Authorised:               c.Authorised,
			LastSuccessfulAccessTime: c.LastSuccessfulAccessTime,
			LastSuccessfulAccessZone: zoneName,
		})

		if err != nil {
			cc.logger.Error("failed to marshal cardholder udmi", zap.Error(err))
			continue
		}

		c.udmiBus.Send(ctx, &gen.PullExportMessagesResponse{
			Name: c.ScName,
			Message: &gen.MqttMessage{
				Topic:   cc.topicPrefix + config.PointsEventTopicSuffix,
				Payload: string(payload),
			},
		})
	}
}
