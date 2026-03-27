package history

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/auto/history/config"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/airqualitysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	gen_airtemperaturepb "github.com/smart-core-os/sc-bos/pkg/proto/airtemperaturepb"
	"github.com/smart-core-os/sc-bos/pkg/proto/devicespb"
	"github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	gen_electricpb "github.com/smart-core-os/sc-bos/pkg/proto/electricpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/occupancysensorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/trait"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

func Test_automation_applyConfig(t *testing.T) {
	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	t.Cleanup(cancel)

	logger := zap.NewNop()
	occupancy := occupancysensorpb.NewModel()
	airQuality := airqualitysensorpb.NewModel()
	airTemperature := gen_airtemperaturepb.NewModel()
	electric := gen_electricpb.NewModel()
	meter := meterpb.NewModel()
	status := statuspb.NewModel()

	announcer := node.New("test")

	announcer.Logger = logger

	announcer.Announce("occupancy",
		node.HasTrait(trait.OccupancySensor),
		node.HasServer(
			occupancysensorpb.RegisterOccupancySensorApiServer,
			occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(occupancy)),
		),
	)

	announcer.Announce("airquality",
		node.HasTrait(trait.AirQualitySensor),
		node.HasServer(
			airqualitysensorpb.RegisterAirQualitySensorApiServer,
			airqualitysensorpb.AirQualitySensorApiServer(airqualitysensorpb.NewModelServer(airQuality)),
		),
	)

	announcer.Announce("airtemperature",
		node.HasTrait(trait.AirTemperature),
		node.HasServer(
			airtemperaturepb.RegisterAirTemperatureApiServer,
			airtemperaturepb.AirTemperatureApiServer(gen_airtemperaturepb.NewModelServer(airTemperature)),
		),
	)

	announcer.Announce("electric",
		node.HasTrait(trait.Electric),
		node.HasServer(
			electricpb.RegisterElectricApiServer,
			electricpb.ElectricApiServer(gen_electricpb.NewModelServer(electric)),
		),
	)

	announcer.Announce("meter",
		node.HasTrait(meterpb.TraitName),
		node.HasServer(
			meterpb.RegisterMeterApiServer,
			meterpb.MeterApiServer(meterpb.NewModelServer(meter)),
		),
	)

	announcer.Announce("status",
		node.HasTrait(statuspb.TraitName),
		node.HasServer(
			statuspb.RegisterStatusApiServer,
			statuspb.StatusApiServer(statuspb.NewModelServer(status)),
		),
	)

	for _, cfg := range cfgs {
		a := &automation{
			clients:   announcer,
			announcer: node.NewReplaceAnnouncer(announcer),
			logger:    logger,
		}

		err := a.applyConfig(ctx, cfg)

		if err != nil {
			t.Fatal(err)
		}
	}

	// many events to each model server
	for range 10 {
		if _, err := occupancy.SetOccupancy(&occupancysensorpb.Occupancy{
			State:       occupancysensorpb.Occupancy_OCCUPIED,
			PeopleCount: int32(rand.Intn(10)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := airQuality.UpdateAirQuality(&airqualitysensorpb.AirQuality{
			CarbonDioxideLevel:       new(rand.Float32()),
			VolatileOrganicCompounds: new(rand.Float32()),
			AirPressure:              new(rand.Float32()),
			InfectionRisk:            new(rand.Float32()),
			Score:                    new(rand.Float32()),
			ParticulateMatter_1:      new(rand.Float32()),
			ParticulateMatter_25:     new(rand.Float32()),
			ParticulateMatter_10:     new(rand.Float32()),
			AirChangePerHour:         new(rand.Float32()),
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := airTemperature.UpdateAirTemperature(&airtemperaturepb.AirTemperature{
			AmbientTemperature: &typespb.Temperature{ValueCelsius: rand.Float64()},
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := electric.UpdateDemand(&electricpb.ElectricDemand{
			Voltage:       new(rand.Float32()),
			Current:       rand.Float32(),
			ReactivePower: new(rand.Float32()),
			ApparentPower: new(rand.Float32()),
			PowerFactor:   new(rand.Float32()),
			RealPower:     new(rand.Float32()),
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := meter.UpdateMeterReading(&meterpb.MeterReading{
			Usage:    rand.Float32(),
			Produced: rand.Float32(),
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := status.UpdateProblem(&statuspb.StatusLog_Problem{
			Name: randString(16),
		}); err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second)
	}

	aqCli := airqualitysensorpb.NewAirQualitySensorHistoryClient(announcer.ClientConn())
	occupancyCli := occupancysensorpb.NewOccupancySensorHistoryClient(announcer.ClientConn())
	airTempCli := gen_airtemperaturepb.NewAirTemperatureHistoryClient(announcer.ClientConn())
	electricCli := gen_electricpb.NewElectricHistoryClient(announcer.ClientConn())
	meterCli := meterpb.NewMeterHistoryClient(announcer.ClientConn())
	statusCli := statuspb.NewStatusHistoryClient(announcer.ClientConn())

	aqHist, err := aqCli.ListAirQualityHistory(ctx, &airqualitysensorpb.ListAirQualityHistoryRequest{Name: "airquality", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(int32(2), aqHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "airquality")
	}

	occHist, err := occupancyCli.ListOccupancyHistory(ctx, &occupancysensorpb.ListOccupancyHistoryRequest{Name: "occupancy", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(int32(2), occHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "occupancy")
	}

	airTempHist, err := airTempCli.ListAirTemperatureHistory(ctx, &gen_airtemperaturepb.ListAirTemperatureHistoryRequest{Name: "airtemperature", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(int32(2), airTempHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "airtemperature")
	}

	electricHist, err := electricCli.ListElectricDemandHistory(ctx, &gen_electricpb.ListElectricDemandHistoryRequest{Name: "electric", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(int32(2), electricHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "electric")
	}

	meterHist, err := meterCli.ListMeterReadingHistory(ctx, &meterpb.ListMeterReadingHistoryRequest{Name: "meter", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(int32(2), meterHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "meter")
	}

	statusHist, err := statusCli.ListCurrentStatusHistory(ctx, &statuspb.ListCurrentStatusHistoryRequest{Name: "status", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(int32(2), statusHist.GetTotalSize()); diff != "" {
		t.Fatal(diff, "status")
	}

}

var cfgs = []config.Root{
	{
		Source: &config.Source{
			Name:            "occupancy",
			Trait:           trait.OccupancySensor,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
	{
		Source: &config.Source{
			Name:            "airquality",
			Trait:           trait.AirQualitySensor,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
	{
		Source: &config.Source{
			Name:            "airtemperature",
			Trait:           trait.AirTemperature,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
	{
		Source: &config.Source{
			Name:            "electric",
			Trait:           trait.Electric,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
	{
		Source: &config.Source{
			Name:            "meter",
			Trait:           meterpb.TraitName,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
	{
		Source: &config.Source{
			Name:            "status",
			Trait:           statuspb.TraitName,
			PollingSchedule: jsontypes.MustParseExtendedSchedule("*/5 * * * * *"),
		},
		Storage: &config.Storage{
			Type: "memory",
			TTL: &config.TTL{
				MaxAge:   jsontypes.Duration{Duration: time.Minute * 3},
				MaxCount: 10,
			},
		},
	},
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func Test_automation_applyConfigDevices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logger := zap.NewNop()
		occ1 := occupancysensorpb.NewModel()
		occ2 := occupancysensorpb.NewModel()

		announcer := node.New("test")
		announcer.Logger = logger

		announcer.Announce("occ1",
			node.HasTrait(trait.OccupancySensor),
			node.HasServer(
				occupancysensorpb.RegisterOccupancySensorApiServer,
				occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(occ1)),
			),
		)
		announcer.Announce("occ2",
			node.HasTrait(trait.OccupancySensor),
			node.HasServer(
				occupancysensorpb.RegisterOccupancySensorApiServer,
				occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(occ2)),
			),
		)

		a := &automation{
			clients:   announcer,
			announcer: node.NewReplaceAnnouncer(announcer),
			logger:    logger,
			devices: &streamingDevicesClient{
				devices: []*devicespb.Device{{Name: "occ1"}, {Name: "occ2"}},
			},
		}

		cfg := config.Root{
			Source:  &config.Source{Trait: trait.OccupancySensor},
			Storage: &config.Storage{Type: "memory"},
		}

		go a.applyConfigDevices(t.Context(), cfg)
		synctest.Wait()

		if _, err := occ1.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 1}); err != nil {
			t.Fatal(err)
		}
		if _, err := occ2.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 2}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		cli := occupancysensorpb.NewOccupancySensorHistoryClient(announcer.ClientConn())

		hist1, err := cli.ListOccupancyHistory(t.Context(), &occupancysensorpb.ListOccupancyHistoryRequest{Name: "occ1", PageSize: 100})
		if err != nil {
			t.Fatal(err)
		}
		if hist1.GetTotalSize() == 0 {
			t.Fatal("expected occ1 to have history records")
		}

		hist2, err := cli.ListOccupancyHistory(t.Context(), &occupancysensorpb.ListOccupancyHistoryRequest{Name: "occ2", PageSize: 100})
		if err != nil {
			t.Fatal(err)
		}
		if hist2.GetTotalSize() == 0 {
			t.Fatal("expected occ2 to have history records")
		}
	})
}

func Test_automation_applyConfigDevices_remove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logger := zap.NewNop()
		occ1 := occupancysensorpb.NewModel()

		announcer := node.New("test")
		announcer.Logger = logger
		announcer.Announce("occ1",
			node.HasTrait(trait.OccupancySensor),
			node.HasServer(
				occupancysensorpb.RegisterOccupancySensorApiServer,
				occupancysensorpb.OccupancySensorApiServer(occupancysensorpb.NewModelServer(occ1)),
			),
		)

		devicesCh := make(chan *devicespb.PullDevicesResponse, 1)
		a := &automation{
			clients:   announcer,
			announcer: node.NewReplaceAnnouncer(announcer),
			logger:    logger,
			devices: &streamingDevicesClient{
				devices: []*devicespb.Device{{Name: "occ1"}},
				ch:      devicesCh,
			},
		}

		cfg := config.Root{
			Source:  &config.Source{Trait: trait.OccupancySensor},
			Storage: &config.Storage{Type: "memory"},
		}

		go a.applyConfigDevices(t.Context(), cfg)
		synctest.Wait()

		if _, err := occ1.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 1}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		cli := occupancysensorpb.NewOccupancySensorHistoryClient(announcer.ClientConn())
		hist, err := cli.ListOccupancyHistory(t.Context(), &occupancysensorpb.ListOccupancyHistoryRequest{Name: "occ1", PageSize: 100})
		if err != nil {
			t.Fatal(err)
		}
		countBeforeRemove := hist.GetTotalSize()
		if countBeforeRemove == 0 {
			t.Fatal("expected occ1 to have history records before remove")
		}

		devicesCh <- &devicespb.PullDevicesResponse{Changes: []*devicespb.PullDevicesResponse_Change{
			{Type: typespb.ChangeType_REMOVE, Name: "occ1"},
		}}
		synctest.Wait()

		if _, err := occ1.SetOccupancy(&occupancysensorpb.Occupancy{State: occupancysensorpb.Occupancy_OCCUPIED, PeopleCount: 2}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		_, err = cli.ListOccupancyHistory(t.Context(), &occupancysensorpb.ListOccupancyHistoryRequest{Name: "occ1", PageSize: 100})
		if status.Code(err) != codes.NotFound {
			t.Fatalf("expected NotFound after remove, got %v", err)
		}
	})
}

// streamingDevicesClient implements devicespb.DevicesApiClient for tests.
// PullDevices immediately sends the configured devices as ADD events, then blocks until the context is cancelled.
// If ch is set, it is used as the stream channel and the caller controls what is sent after the initial events.
type streamingDevicesClient struct {
	devices []*devicespb.Device
	ch      chan *devicespb.PullDevicesResponse // optional; if nil, a local buffered channel is used
}

func (s *streamingDevicesClient) ListDevices(_ context.Context, _ *devicespb.ListDevicesRequest, _ ...grpc.CallOption) (*devicespb.ListDevicesResponse, error) {
	panic("not implemented")
}

func (s *streamingDevicesClient) PullDevices(ctx context.Context, _ *devicespb.PullDevicesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[devicespb.PullDevicesResponse], error) {
	ch := s.ch
	if ch == nil {
		ch = make(chan *devicespb.PullDevicesResponse, 1)
	}
	if len(s.devices) > 0 {
		changes := make([]*devicespb.PullDevicesResponse_Change, len(s.devices))
		for i, d := range s.devices {
			changes[i] = &devicespb.PullDevicesResponse_Change{
				Type:     typespb.ChangeType_ADD,
				NewValue: d,
				Name:     d.Name,
			}
		}
		ch <- &devicespb.PullDevicesResponse{Changes: changes}
	}
	return &pullDevicesStream{ctx: ctx, ch: ch}, nil
}

func (s *streamingDevicesClient) GetDevicesMetadata(_ context.Context, _ *devicespb.GetDevicesMetadataRequest, _ ...grpc.CallOption) (*devicespb.DevicesMetadata, error) {
	panic("not implemented")
}

func (s *streamingDevicesClient) PullDevicesMetadata(_ context.Context, _ *devicespb.PullDevicesMetadataRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[devicespb.PullDevicesMetadataResponse], error) {
	panic("not implemented")
}

func (s *streamingDevicesClient) GetDownloadDevicesUrl(_ context.Context, _ *devicespb.GetDownloadDevicesUrlRequest, _ ...grpc.CallOption) (*devicespb.DownloadDevicesUrl, error) {
	panic("not implemented")
}

type pullDevicesStream struct {
	ctx context.Context
	ch  chan *devicespb.PullDevicesResponse
}

func (s *pullDevicesStream) Recv() (*devicespb.PullDevicesResponse, error) {
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case resp, ok := <-s.ch:
		if !ok {
			return nil, status.Error(codes.Canceled, "stream closed")
		}
		return resp, nil
	}
}

func (s *pullDevicesStream) Header() (metadata.MD, error) { return nil, nil }
func (s *pullDevicesStream) Trailer() metadata.MD         { return nil }
func (s *pullDevicesStream) CloseSend() error             { return nil }
func (s *pullDevicesStream) Context() context.Context     { return s.ctx }
func (s *pullDevicesStream) SendMsg(_ any) error          { return nil }
func (s *pullDevicesStream) RecvMsg(_ any) error          { return nil }
