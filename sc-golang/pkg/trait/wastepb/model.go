package wastepb

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

var disposalMethods = [3]traits.WasteRecord_DisposalMethod{
	traits.WasteRecord_DISPOSAL_METHOD_UNSPECIFIED,
	traits.WasteRecord_MIXED_RECYCLING,
	traits.WasteRecord_GENERAL_WASTE,
}

var areas = [3]string{"Area 1", "Area 2", "Area 3"}
var systems = [3]string{"System 1", "System 2", "System 3"}
var streams = [3]string{"Stream 1", "Stream 2", "Stream 3"}

type Model struct {
	mu              sync.Mutex // guards allWasteRecords and genId
	allWasteRecords []*traits.WasteRecord
	genId           int

	lastWasteRecord *resource.Value // of *traits.WasteRecord
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&traits.WasteRecord{})}
	opts = append(defaultOpts, opts...)

	m := &Model{
		lastWasteRecord: resource.NewValue(opts...),
	}

	// let's add some records to start with so we can test the list method without waiting
	startTime := time.Now().Add(-100 * time.Minute)
	for i := 0; i < 100; i++ {
		_, _ = m.GenerateWasteRecord(timestamppb.New(startTime))
		startTime = startTime.Add(time.Minute)
	}
	return m
}

// AddWasteRecord manually adds a waste record to the model
func (m *Model) AddWasteRecord(wr *traits.WasteRecord, opts ...resource.WriteOption) (*traits.WasteRecord, error) {
	v, err := m.lastWasteRecord.Set(wr, opts...)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allWasteRecords = append(m.allWasteRecords, wr)
	return v.(*traits.WasteRecord), nil
}

// GenerateWasteRecord generates a new waste record with the given timestamp and adds it to the model
func (m *Model) GenerateWasteRecord(ts *timestamppb.Timestamp) (*traits.WasteRecord, error) {
	m.mu.Lock()
	wr := &traits.WasteRecord{
		WasteCreateTime:  ts,
		RecordCreateTime: ts,
		Id:               strconv.Itoa(m.genId),
		Weight:           rand.Float32() * 1000,
		DisposalMethod:   disposalMethods[m.genId%len(disposalMethods)],
		Area:             areas[m.genId%len(areas)],
		System:           systems[m.genId%len(systems)],
		Stream:           streams[m.genId%len(streams)],
		Co2Saved:         rand.Float32() * 100,
		LandSaved:        rand.Float32() * 100,
		TreesSaved:       rand.Float32() * 5,
	}
	m.genId++
	m.mu.Unlock()
	return m.AddWasteRecord(wr)
}

func (m *Model) GetWasteRecordCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.allWasteRecords)
}

func (m *Model) ListWasteRecords(start, count int) []*traits.WasteRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var wasteRecords []*traits.WasteRecord
	// reverse to retrieve the latest wasteRecords first
	for i := start - 1; i >= 0; i-- {
		wasteRecords = append(wasteRecords, m.allWasteRecords[i])
		if len(wasteRecords) >= count {
			break
		}
	}
	return wasteRecords
}

func (m *Model) pullWasteRecordsWrapper(request *traits.PullWasteRecordsRequest, server traits.WasteApi_PullWasteRecordsServer) error {
	if !request.UpdatesOnly {
		m.mu.Lock()
		i := len(m.allWasteRecords) - 50
		if i < 0 {
			i = 0
		}
		for ; i < len(m.allWasteRecords)-1; i++ {
			change := &traits.PullWasteRecordsResponse_Change{
				Name:       request.Name,
				NewValue:   m.allWasteRecords[i],
				ChangeTime: m.allWasteRecords[i].WasteCreateTime,
				Type:       types.ChangeType_ADD,
			}
			if err := server.Send(&traits.PullWasteRecordsResponse{Changes: []*traits.PullWasteRecordsResponse_Change{change}}); err != nil {
				m.mu.Unlock()
				return err
			}
		}
		m.mu.Unlock()
	}
	for change := range m.PullWasteRecords(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		msg := &traits.PullWasteRecordsResponse{}
		msg.Changes = append(msg.Changes, change)
		if err := server.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) PullWasteRecords(ctx context.Context, opts ...resource.ReadOption) <-chan *traits.PullWasteRecordsResponse_Change {
	send := make(chan *traits.PullWasteRecordsResponse_Change)
	recv := m.lastWasteRecord.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			wr := change.Value.(*traits.WasteRecord)
			change := &traits.PullWasteRecordsResponse_Change{
				Name:       "Waste Record",
				NewValue:   wr, // the mock driver only generates new waste records and does not delete them
				ChangeTime: wr.WasteCreateTime,
				Type:       types.ChangeType_ADD,
			}
			send <- change
		}
	}()

	return send
}
