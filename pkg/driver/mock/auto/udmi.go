package auto

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/rand"

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
)

type UdmiServer struct {
	udmipb.UnimplementedUdmiServiceServer
	log        *zap.Logger
	deviceName string
}

func NewUdmiServer(log *zap.Logger, deviceName string) *UdmiServer {
	return &UdmiServer{
		log:        log,
		deviceName: deviceName,
	}
}

func (u *UdmiServer) GetExportMessage(ctx context.Context, request *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	return u.randMqttMessage()
}

func (u *UdmiServer) PullExportMessages(request *udmipb.PullExportMessagesRequest, messagesServer udmipb.UdmiService_PullExportMessagesServer) error {

	if request.IncludeLast {
		msg, err := u.randMqttMessage()
		if err != nil {
			u.log.Warn("Failed to marshal UDMI payload", zap.Error(err))
		} else if err := u.sendMqttMessage(messagesServer, msg); err != nil {
			return err
		}
	}

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-messagesServer.Context().Done():
			return messagesServer.Context().Err()
		case <-ticker.C:
			msg, err := u.randMqttMessage()
			if err != nil {
				u.log.Warn("Failed to marshal UDMI payload", zap.Error(err))
				continue
			}
			if err := u.sendMqttMessage(messagesServer, msg); err != nil {
				return err
			}
		}
	}
}

func (u *UdmiServer) sendMqttMessage(stream udmipb.UdmiService_PullExportMessagesServer, msg *udmipb.MqttMessage) error {
	return stream.Send(&udmipb.PullExportMessagesResponse{Name: u.deviceName, Message: msg})
}

func (u *UdmiServer) randMqttMessage() (*udmipb.MqttMessage, error) {
	points := &udmi.PointsEvent{
		"ClgDmnd":          udmi.PointValue{PresentValue: 0},
		"ClgOrrideCmd":     udmi.PointValue{PresentValue: 0},
		"ClgOvrd":          udmi.PointValue{PresentValue: 0},
		"FanOrrideCmd":     udmi.PointValue{PresentValue: 50},
		"FanOvrd":          udmi.PointValue{PresentValue: 0},
		"FanSpd":           udmi.PointValue{PresentValue: 0},
		"FanStat":          udmi.PointValue{PresentValue: 0},
		"HtgDmnd":          udmi.PointValue{PresentValue: 0},
		"HtgOrrideCmd":     udmi.PointValue{PresentValue: 0},
		"HtgOvrd":          udmi.PointValue{PresentValue: 0},
		"MaxFanSpdStPt":    udmi.PointValue{PresentValue: 51},
		"MinFanSpdStPt":    udmi.PointValue{PresentValue: 20},
		"NOccDb":           udmi.PointValue{PresentValue: 25},
		"OccCoolStPt":      udmi.PointValue{PresentValue: 20.5},
		"OccDb":            udmi.PointValue{PresentValue: 1},
		"OccHtgStPt":       udmi.PointValue{PresentValue: 19.5},
		"Occupation Relay": udmi.PointValue{PresentValue: 0},
		"RATemp":           udmi.PointValue{PresentValue: 16 + rand.Float32()*8.0},
		"RemFanSpd":        udmi.PointValue{PresentValue: 0},
		"RemFanSpdSlct":    udmi.PointValue{PresentValue: 0},
		"RemOcc":           udmi.PointValue{PresentValue: 1},
		"RemShutdwn":       udmi.PointValue{PresentValue: 0},
		"SATemp":           udmi.PointValue{PresentValue: 16 + rand.Float32()*8.0},
		"SlctdFanSpd":      udmi.PointValue{PresentValue: 0},
		"Unocc":            udmi.PointValue{PresentValue: 1},
		"ZnTempStPt":       udmi.PointValue{PresentValue: 23},
	}
	body, err := json.Marshal(points)
	if err != nil {
		return nil, err
	}
	return u.mqttMessage(body), nil
}

func (u *UdmiServer) mqttMessage(body []byte) *udmipb.MqttMessage {
	return &udmipb.MqttMessage{
		Topic:   "test/mock/" + u.deviceName + "/event/pointset/points",
		Payload: string(body),
	}
}
