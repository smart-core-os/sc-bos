// Package local implements a local model of the AirThings api.
// The types in this package decouple the needs of Smart Core from any limitations in the AirThings api.
package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/olebedev/emitter"

	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
)

// Location holds information about devices in a particular location.
type Location struct {
	mu                    sync.RWMutex // guards the below
	latestSamplesByDevice map[string]api.DeviceSampleResponseEnriched
	bus                   *emitter.Emitter
}

func NewLocation() *Location {
	return &Location{
		latestSamplesByDevice: make(map[string]api.DeviceSampleResponseEnriched),
		bus:                   emitter.New(0),
	}
}

// GetLatestSample returns the latest sample for the given device, or false if there is none.
func (m *Location) GetLatestSample(deviceID string) (api.DeviceSampleResponseEnriched, bool) {
	m.mu.RLock()
	sample, ok := m.latestSamplesByDevice[deviceID]
	m.mu.RUnlock()
	return sample, ok
}

// PullLatestSamples subscribes to changes to the latest sample for the given device.
// Changes will be published to the returned chan, the current value will be returned immediately.
// The chan will be closed when the context is cancelled.
// The goroutine will clean up automatically when ctx is done.
func (m *Location) PullLatestSamples(ctx context.Context, deviceID string) (api.DeviceSampleResponseEnriched, <-chan api.DeviceSampleResponseEnriched) {
	topic := fmt.Sprintf("sample/%s/change", deviceID)
	m.mu.RLock()
	latestSample, _ := m.latestSamplesByDevice[deviceID]
	stream := m.bus.On(topic)
	m.mu.RUnlock()

	ch := make(chan api.DeviceSampleResponseEnriched)

	go func() {
		defer close(ch)
		defer m.bus.Off(topic, stream)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-stream:
				if !ok {
					return
				}
				sample, ok := event.Args[0].(api.DeviceSampleResponseEnriched)
				if !ok {
					continue
				}
				select {
				case ch <- sample:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return latestSample, ch
}

// UpdateLatestSamples writes updates and notifies subscribers.
// The returned chan will be closed when all subscribers have been notified.
func (m *Location) UpdateLatestSamples(samples api.GetLocationSamplesResponseEnriched) <-chan struct{} {
	var emitChans []<-chan struct{}

	m.mu.Lock()
	for _, sample := range samples.GetDevices() {
		id := m.sampleDeviceID(sample)
		m.latestSamplesByDevice[id] = sample
		ch := m.emitSampleChange(sample)
		emitChans = append(emitChans, ch)
	}
	m.mu.Unlock()

	ch := make(chan struct{})
	go func() {
		for _, emitChan := range emitChans {
			<-emitChan
		}
		close(ch)
	}()
	return ch
}

func (m *Location) emitSampleChange(sample api.DeviceSampleResponseEnriched) <-chan struct{} {
	id := m.sampleDeviceID(sample)
	return m.bus.Emit(fmt.Sprintf("sample/%s/change", id), sample)
}

func (*Location) sampleDeviceID(sample api.DeviceSampleResponseEnriched) string {
	segment, ok := sample.GetSegmentOk()
	if !ok {
		return ""
	}
	return segment.GetId()
}
