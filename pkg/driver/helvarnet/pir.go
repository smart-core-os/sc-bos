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

	"github.com/smart-core-os/sc-bos/pkg/auto/udmi"
	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/udmipb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// Pir represents a single PIR sensor within the HelvarNet system.
type Pir struct {
	occupancysensorpb.UnimplementedOccupancySensorApiServer
	udmipb.UnimplementedUdmiServiceServer

	client    *tcpClient
	conf      *config.Device
	logger    *zap.Logger
	occupancy *resource.Value // *traits.Occupancy
}

func newPir(client *tcpClient, l *zap.Logger, conf *config.Device) *Pir {
	return &Pir{
		client:    client,
		conf:      conf,
		logger:    l,
		occupancy: resource.NewValue(resource.WithInitialValue(&occupancysensorpb.Occupancy{}), resource.WithNoDuplicates()),
	}
}

// refreshOccupancyStatus queries the input state of the sensor, stores it, and returns what was read.
func (p *Pir) refreshOccupancyStatus(ctx context.Context) (*occupancysensorpb.Occupancy, error) {
	command := queryInputState(p.conf.Address)
	want := "?" + command[1:len(command)-1]

	r, err := p.client.sendAndReceive(ctx, command, want)
	if err != nil {
		return nil, err
	}

	split := strings.Split(r, "=")
	if len(split) < 2 {
		return nil, fmt.Errorf("invalid response in queryInputState: %s", r)
	}

	s := strings.TrimSuffix(split[1], "#")
	statusInt, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}

	state := occupancysensorpb.Occupancy_UNOCCUPIED
	if statusInt == 1 {
		state = occupancysensorpb.Occupancy_OCCUPIED
	}
	occupancy := &occupancysensorpb.Occupancy{
		State:      state,
		Confidence: 1,
	}

	_, _ = p.occupancy.Set(occupancy)
	return occupancy, nil
}

func (p *Pir) GetOccupancy(context.Context, *occupancysensorpb.GetOccupancyRequest) (*occupancysensorpb.Occupancy, error) {
	value := p.occupancy.Get().(*occupancysensorpb.Occupancy)
	return value, nil
}

func (p *Pir) PullOccupancy(_ *occupancysensorpb.PullOccupancyRequest, server occupancysensorpb.OccupancySensorApi_PullOccupancyServer) error {
	for value := range p.occupancy.Pull(server.Context()) {
		occupancy := value.Value.(*occupancysensorpb.Occupancy)
		err := server.Send(&occupancysensorpb.PullOccupancyResponse{Changes: []*occupancysensorpb.PullOccupancyResponse_Change{
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

	_, err := p.refreshOccupancyStatus(ctx)
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
			_, err := p.refreshOccupancyStatus(ctx)
			if err != nil {
				p.logger.Error("failed to refresh occupancy status", zap.Error(err))
			}
		}
	}
}

func (p *Pir) udmiPointsetFromData(occupancy *occupancysensorpb.Occupancy) (*udmipb.MqttMessage, error) {
	points := make(udmi.PointsEvent)
	points["OccupancyStatus"] = udmi.PointValue{PresentValue: occupancy.State.String()}

	b, err := json.Marshal(points)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal udmi points: %w", err)
	}
	return &udmipb.MqttMessage{
		Topic:   p.conf.TopicPrefix + "/event/pointset/points",
		Payload: string(b),
	}, nil
}

// GetExportMessage reads the sensor now rather than replaying the cached value, so a heartbeat
// never vouches for a PIR that has stopped answering (see pkg/auto/udmi/README.md).
func (p *Pir) GetExportMessage(ctx context.Context, _ *udmipb.GetExportMessageRequest) (*udmipb.MqttMessage, error) {
	occupancy, err := p.refreshOccupancyStatus(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, status.Errorf(codes.Unavailable, "failed to read occupancy: %v", err)
	}
	m, err := p.udmiPointsetFromData(occupancy)
	if err != nil {
		p.logger.Error("failed to create udmi pointset message", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create udmi pointset message")
	}
	return m, nil
}

// PullExportMessages sends a pointset each time the occupancy changes. The poller reads the
// sensor every few seconds, so sending on every poll, as the light does, would flood the export.
//
// A new stream opens with a live read rather than the cached value. The cache outlives the
// device (a failed poll leaves it in place) and the udmi auto resubscribes whenever its export
// task retries, so replaying the cache would let a dead PIR keep publishing and keep resetting
// its heartbeat. If the read fails the stream starts silent and waits for the next real change.
func (p *Pir) PullExportMessages(_ *udmipb.PullExportMessagesRequest, server udmipb.UdmiService_PullExportMessagesServer) error {
	ctx := server.Context()
	// Subscribe before reading so a change the read itself sets can't slip between the two.
	changes := p.occupancy.Pull(ctx, resource.WithUpdatesOnly(true))

	var lastSent occupancysensorpb.Occupancy_State
	send := func(occupancy *occupancysensorpb.Occupancy) error {
		// UNSPECIFIED is the placeholder before the first read; a real read is always OCCUPIED or UNOCCUPIED.
		// lastSent drops the change that the opening read pushes through changes as well.
		if occupancy.GetState() == occupancysensorpb.Occupancy_STATE_UNSPECIFIED || occupancy.GetState() == lastSent {
			return nil
		}
		m, err := p.udmiPointsetFromData(occupancy)
		if err != nil {
			p.logger.Error("failed to create udmi pointset message", zap.Error(err))
			return nil
		}
		if err := server.Send(&udmipb.PullExportMessagesResponse{Name: p.conf.Name, Message: m}); err != nil {
			return err
		}
		lastSent = occupancy.GetState()
		return nil
	}

	if occupancy, err := p.refreshOccupancyStatus(ctx); err != nil {
		p.logger.Debug("no current occupancy for new export stream", zap.Error(err))
	} else if err := send(occupancy); err != nil {
		return err
	}
	for value := range changes {
		if err := send(value.Value.(*occupancysensorpb.Occupancy)); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pir) PullControlTopics(*udmipb.PullControlTopicsRequest, grpc.ServerStreamingServer[udmipb.PullControlTopicsResponse]) error {
	return status.Error(codes.Unimplemented, "PullControlTopics is not implemented for Pir")
}

func (p *Pir) OnMessage(context.Context, *udmipb.OnMessageRequest) (*udmipb.OnMessageResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OnMessage is not implemented for Pir")
}
