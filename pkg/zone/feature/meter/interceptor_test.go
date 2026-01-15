package meter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/smart-core-os/sc-bos/pkg/proto/meterpb"
)

type mockPullServer struct {
	ctx     context.Context
	changes chan *meterpb.PullMeterReadingsResponse
	mtx     sync.Mutex
}

func (m *mockPullServer) SetHeader(md metadata.MD) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockPullServer) SendHeader(md metadata.MD) error {
	// TODO implement me
	panic("implement me")
}

func (m *mockPullServer) SetTrailer(md metadata.MD) {
	// TODO implement me
	panic("implement me")
}

func (m *mockPullServer) SendMsg(a any) error {
	return nil
}

func (m *mockPullServer) RecvMsg(a any) error {
	return nil
}

func (m *mockPullServer) Context() context.Context {
	return m.ctx
}

func (m *mockPullServer) Send(resp *meterpb.PullMeterReadingsResponse) error {
	m.changes <- resp
	return nil
}

type timeInterceptor struct {
	current int
	base    time.Time
	step    time.Duration
}

func (t *timeInterceptor) now() time.Time {
	return t.base.Add(t.step * time.Duration(t.current))
}

func (t *timeInterceptor) time() time.Time {
	ti := t.base.Add(t.step * time.Duration(t.current))
	t.current++
	return ti
}

func (t *timeInterceptor) from(index int) time.Time {
	return t.base.Add(t.step * time.Duration(index))
}

type event struct {
	usage float32
	err   error
}
type meterModelInterceptor struct {
	meterpb.UnimplementedMeterApiServer
	meterpb.UnimplementedMeterInfoServer
	events map[string][]event

	mtx          sync.Mutex
	meterToIndex map[string]int
}

func (m *meterModelInterceptor) DescribeMeterReading(_ context.Context, _ *meterpb.DescribeMeterReadingRequest) (*meterpb.MeterReadingSupport, error) {
	return &meterpb.MeterReadingSupport{
		UsageUnit: "kWh",
	}, nil
}

func (m *meterModelInterceptor) GetMeterReading(_ context.Context, r *meterpb.GetMeterReadingRequest) (*meterpb.MeterReading, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if len(m.events) == 0 {
		return nil, errors.New("no events")
	}
	events, ok := m.events[r.Name]
	if !ok {
		return nil, fmt.Errorf("unknown meter %q", r.Name)
	}

	index, ok := m.meterToIndex[r.Name]
	if !ok {
		return nil, fmt.Errorf("unknown meter %q", r.Name)
	}

	if index >= len(events) {
		index = 0 // wrap around
	}

	m.meterToIndex[r.Name] = index + 1

	ev := events[index]
	if ev.err != nil {
		return nil, ev.err
	}

	return &meterpb.MeterReading{
		Usage: ev.usage,
	}, nil
}

func (m *meterModelInterceptor) PullMeterReadings(r *meterpb.PullMeterReadingsRequest, s meterpb.MeterApi_PullMeterReadingsServer) error {
	for {
		res, err := m.GetMeterReading(s.Context(), &meterpb.GetMeterReadingRequest{Name: r.Name})
		if err != nil {
			return err
		}

		if err = s.Send(&meterpb.PullMeterReadingsResponse{
			Changes: []*meterpb.PullMeterReadingsResponse_Change{
				{
					Name:         r.Name,
					MeterReading: res,
				},
			},
		}); err != nil {
			return err
		}
	}
}
