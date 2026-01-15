package opcua

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/driver/opcua/config"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

// Udmi implements the Smart Core UDMI (Universal Device Management Interface) trait for OPC UA devices.
// It collects OPC UA variable values and publishes them as UDMI point events to an MQTT-style message bus.
type Udmi struct {
	config.Trait
	udmipb.UnimplementedUdmiServiceServer

	logger *zap.Logger
	// the points that have been configured to be monitored. node id -> value source
	monitoredPoints map[string]*config.ValueSource
	pointEvents     udmi.PointsEvent
	pointsMu        sync.Mutex
	scName          string
	udmiBus         minibus.Bus[*udmipb.PullExportMessagesResponse]
	udmiConfig      config.UdmiConfig
}

func readUdmiConfig(raw []byte) (cfg config.UdmiConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

func newUdmi(n string, c config.RawTrait, l *zap.Logger) (*Udmi, error) {
	cfg, err := readUdmiConfig(c.Raw)
	if err != nil {
		return nil, err
	}
	u := &Udmi{
		logger:          l,
		monitoredPoints: make(map[string]*config.ValueSource),
		pointEvents:     make(udmi.PointsEvent),
		scName:          n,
		udmiConfig:      cfg,
	}
	// the points in the config are defined as a map of name -> value source to match the pattern we use
	// we get a callback with the node id, so convert to a map of node id -> value source for fast lookup
	for _, p := range cfg.Points {
		u.monitoredPoints[p.NodeId] = p
	}
	return u, nil
}

func (u *Udmi) sendUdmiMessage(ctx context.Context, node *ua.NodeID, value any) {

	if p, ok := u.monitoredPoints[node.String()]; ok {

		pointName := p.Name
		presetValue := p.GetValueFromIntKey(value)
		u.pointsMu.Lock()
		defer u.pointsMu.Unlock()
		u.pointEvents[pointName] = udmi.PointValue{PresentValue: presetValue}

		body, err := json.Marshal(u.pointEvents)
		if err != nil {
			u.logger.Error("failed to marshal points event", zap.String("device", u.scName), zap.Error(err))
			return
		}

		u.udmiBus.Send(ctx, &udmipb.PullExportMessagesResponse{
			Name: u.scName,
			Message: &udmipb.MqttMessage{
				Topic:   u.udmiConfig.TopicPrefix + config.PointsEventTopicSuffix,
				Payload: string(body),
			},
		})
	}
}

func (u *Udmi) PullControlTopics(_ *udmipb.PullControlTopicsRequest, topicsServer udmipb.UdmiService_PullControlTopicsServer) error {
	// we don't have any control topics, yet
	<-topicsServer.Context().Done()
	return nil
}

func (u *Udmi) OnMessage(context.Context, *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	// we don't support doing anything here, yet
	return &udmipb.OnMessageResponse{}, nil
}

func (u *Udmi) PullExportMessages(_ *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	for msg := range u.udmiBus.Listen(server.Context()) {
		err := server.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *Udmi) GetExportMessage(context.Context, *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	u.pointsMu.Lock()
	defer u.pointsMu.Unlock()

	body, err := json.Marshal(u.pointEvents)
	if err != nil {
		u.logger.Error("failed to marshal points event", zap.String("device", u.scName), zap.Error(err))
		return nil, err
	}
	return &udmipb.MqttMessage{
		Topic:   u.udmiConfig.TopicPrefix + config.PointsEventTopicSuffix,
		Payload: string(body),
	}, nil
}
