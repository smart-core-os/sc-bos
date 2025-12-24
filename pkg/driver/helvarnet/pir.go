package helvarnet

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-bos/pkg/gen"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// Pir represents a single PIR sensor within the HelvarNet system.
type Pir struct {
	traits.UnimplementedOccupancySensorApiServer
	gen.UnimplementedUdmiServiceServer

	client    *tcpClient
	conf      *config.Device
	logger    *zap.Logger
	occupancy *resource.Value // *traits.Occupancy
	udmiBus   minibus.Bus[*gen.PullExportMessagesResponse]
}

func newPir(client *tcpClient, l *zap.Logger, conf *config.Device) *Pir {
	return &Pir{
		client:    client,
		conf:      conf,
		logger:    l,
		occupancy: resource.NewValue(resource.WithInitialValue(&traits.Occupancy{}), resource.WithNoDuplicates()),
	}
}

// refreshOccupancyStatus refreshes the occupancy by querying the input state of the sensor
func (p *Pir) refreshOccupancyStatus(ctx context.Context) error {
	command := queryInputState(p.conf.Address)
	want := "?" + command[1:len(command)-1]

	r, err := p.client.sendAndReceive(ctx, command, want)
	if err != nil {
		return err
	}

	split := strings.Split(r, "=")
	if len(split) < 2 {
		return fmt.Errorf("invalid response in getLastScene: %s", r)
	}

	s := strings.TrimSuffix(split[1], "#")
	statusInt, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	occupancy := traits.Occupancy_UNOCCUPIED
	if statusInt == 1 {
		occupancy = traits.Occupancy_OCCUPIED
	}

	// Update the occupancy status
	_, _ = p.occupancy.Set(&traits.Occupancy{
		State:      occupancy,
		Confidence: 1,
	})

	return nil
}

func (p *Pir) GetOccupancy(context.Context, *traits.GetOccupancyRequest) (*traits.Occupancy, error) {
	value := p.occupancy.Get().(*traits.Occupancy)
	return value, nil
}

func (p *Pir) PullOccupancy(_ *traits.PullOccupancyRequest, server traits.OccupancySensorApi_PullOccupancyServer) error {
	for value := range p.occupancy.Pull(server.Context()) {
		occupancy := value.Value.(*traits.Occupancy)
		err := server.Send(&traits.PullOccupancyResponse{Changes: []*traits.PullOccupancyResponse_Change{
			{
				Name:       p.conf.Name,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Occupancy:  occupancy,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Pir) runUpdateState(ctx context.Context, t time.Duration) error {

	err := p.refreshOccupancyStatus(ctx)
	if err != nil {
		p.logger.Error("failed to refresh occupancy status", zap.Error(err))
	}
	ticker := time.NewTicker(t)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := p.refreshOccupancyStatus(ctx)
			if err != nil {
				p.logger.Error("failed to refresh occupancy status", zap.Error(err))
			}
		}
	}
}

func (p *Pir) udmiPointsetFromData() (*gen.MqttMessage, error) {
	points := make(udmi.PointsEvent)
	occupancy := p.occupancy.Get().(*traits.Occupancy)
	points["OccupancyStatus"] = udmi.PointValue{PresentValue: occupancy.State.String()}

	b, err := json.Marshal(points)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal udmi points: %w", err)
	}
	return &gen.MqttMessage{
		Topic:   p.conf.TopicPrefix + "/event/pointset/points",
		Payload: string(b),
	}, nil
}

func (p *Pir) sendUdmiMessage(ctx context.Context) {
	m, err := p.udmiPointsetFromData()
	if err != nil {
		p.logger.Error("failed to create udmi pointset message", zap.Error(err))
		return
	}

	p.udmiBus.Send(ctx, &gen.PullExportMessagesResponse{
		Name:    p.conf.Name,
		Message: m,
	})
}

func (p *Pir) GetExportMessage(context.Context, *gen.GetExportMessageRequest) (*gen.MqttMessage, error) {
	m, err := p.udmiPointsetFromData()
	if err != nil {
		p.logger.Error("failed to create udmi pointset message", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create udmi pointset message")
	}
	return m, nil
}

func (p *Pir) PullExportMessages(_ *gen.PullExportMessagesRequest, server gen.UdmiService_PullExportMessagesServer) error {
	for msg := range p.udmiBus.Listen(server.Context()) {
		err := server.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Pir) PullControlTopics(*gen.PullControlTopicsRequest, grpc.ServerStreamingServer[gen.PullControlTopicsResponse]) error {
	return status.Error(codes.Unimplemented, "PullControlTopics is not implemented for Pir")
}

func (p *Pir) OnMessage(context.Context, *gen.OnMessageRequest) (*gen.OnMessageResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OnMessage is not implemented for Pir")
}
