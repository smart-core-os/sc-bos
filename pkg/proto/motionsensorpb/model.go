package motionsensorpb

import (
	"context"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/resource"
)

type Model struct {
	motionDetection *resource.Value // of *MotionDetection
}

func NewModel(opts ...resource.Option) *Model {
	defaultOpts := []resource.Option{resource.WithInitialValue(&MotionDetection{})}
	opts = append(defaultOpts, opts...)
	return &Model{
		motionDetection: resource.NewValue(opts...),
	}
}

// SetMotionDetection updates the known motion detection state for this device.
func (m *Model) SetMotionDetection(motionDetection *MotionDetection, opts ...resource.WriteOption) (*MotionDetection, error) {
	v, err := m.motionDetection.Set(motionDetection, opts...)
	if err != nil {
		return nil, err
	}
	return v.(*MotionDetection), nil
}

func (m *Model) GetMotionDetection(opts ...resource.ReadOption) (*MotionDetection, error) {
	return m.motionDetection.Get(opts...).(*MotionDetection), nil
}

func (m *Model) PullMotionDetections(ctx context.Context, opts ...resource.ReadOption) <-chan PullMotionDetectionChange {
	send := make(chan PullMotionDetectionChange)
	recv := m.motionDetection.Pull(ctx, opts...)
	go func() {
		defer close(send)
		for change := range recv {
			send <- PullMotionDetectionChange{
				Value:      change.Value.(*MotionDetection),
				ChangeTime: change.ChangeTime,
			}
		}
	}()
	return send
}

type PullMotionDetectionChange struct {
	Value      *MotionDetection
	ChangeTime time.Time
}
