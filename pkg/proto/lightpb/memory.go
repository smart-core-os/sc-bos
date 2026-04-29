package lightpb

import (
	"context"
	"time"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-bos/pkg/proto/typespb"
	"github.com/smart-core-os/sc-bos/pkg/resource"
)

// MemoryDevice implements the LightApiServer interface for a single device by storing state in memory.
type MemoryDevice struct {
	UnimplementedLightApiServer
	brightness     *resource.Value
	brightnessTick time.Duration // duration between updates when tweening brightness

	// todo: support presets
}

func NewMemoryDevice() *MemoryDevice {
	return &MemoryDevice{
		brightnessTick: time.Second / 15,
		brightness: resource.NewValue(
			resource.WithInitialValue(InitialBrightness()),
			resource.WithWritablePaths(&Brightness{},
				"level_percent",
				"brightness_tween.total_duration",
				"preset",
			),
		),
	}
}

func InitialBrightness() *Brightness {
	return &Brightness{}
}

func (s *MemoryDevice) GetBrightness(_ context.Context, req *GetBrightnessRequest) (*Brightness, error) {
	return s.brightness.Get(resource.WithReadMask(req.ReadMask)).(*Brightness), nil
}

func (s *MemoryDevice) UpdateBrightness(ctx context.Context, request *UpdateBrightnessRequest) (*Brightness, error) {
	if request.GetBrightness().GetPreset() != nil {
		res, err := s.brightness.Set(request.GetBrightness())
		return res.(*Brightness), err
	}

	if err := resource.ValidateTweenOnUpdate("brightness", request.GetBrightness().GetBrightnessTween()); err != nil {
		return nil, err
	}

	duration := request.Brightness.GetBrightnessTween().GetTotalDuration().AsDuration()
	if duration > 0 {
		startTime := time.Now()
		lastObj, err := s.brightness.Set(request.Brightness,
			resource.WithUpdatePaths("level_percent", "brightness_tween", "target_level_percent"),
			resource.WithMoreWritablePaths("brightness_tween", "target_level_percent"),
			resource.InterceptBefore(func(old, new proto.Message) {
				current := old.(*Brightness)
				next := new.(*Brightness)
				if request.Delta {
					next.LevelPercent += current.LevelPercent
				}
				capLevelPercent(next)
				// move properties into their tween equivalents
				next.TargetLevelPercent = next.LevelPercent
				next.LevelPercent = current.LevelPercent
				next.BrightnessTween.Progress = 0
			}),
		)
		if err != nil {
			return nil, err
		}

		startVal := lastObj.(*Brightness)
		tween := gween.New(startVal.LevelPercent, startVal.TargetLevelPercent, float32(duration.Milliseconds()), ease.Linear)

		go func() {
			ticker := time.NewTicker(s.brightnessTick)
			defer ticker.Stop()
			for {
				now := <-ticker.C
				playTime := now.Sub(startTime)
				newValue, finished := tween.Set(float32(playTime.Milliseconds()))
				if finished {
					// the tween has completed, reset the tween data
					_, err := s.brightness.Set(&Brightness{LevelPercent: newValue},
						resource.WithUpdatePaths("level_percent"),
						resource.WithResetPaths("target_level_percent", "brightness_tween"),
						resource.WithExpectedValue(lastObj),
					)
					if err != nil && err != resource.ExpectedValuePreconditionFailed {
						panic(err) // programmer error
					}
					return
				}

				// calculate using time, not value, which leave room for easing (and is mentioned in the tween spec)
				progress := 100 * float32(playTime.Milliseconds()) / float32(duration.Milliseconds())
				lastObj, err = s.brightness.Set(&Brightness{LevelPercent: newValue, BrightnessTween: &typespb.Tween{Progress: progress}},
					resource.WithUpdatePaths("level_percent", "brightness_tween.progress"),
					resource.WithMoreWritablePaths("brightness_tween.progress"),
					resource.WithExpectedValue(lastObj),
				)
				switch {
				case err == resource.ExpectedValuePreconditionFailed:
					// somebody else changed the value, tweening is done
					return
				case err != nil:
					panic(err) // programmer error
				}
			}
		}()

		return startVal, nil
	}

	res, err := s.brightness.Set(
		request.Brightness,
		// if there's a tween in progress, clear the tween props
		resource.WithResetPaths("target_level_percent", "brightness_tween"),
		resource.InterceptBefore(func(old, change proto.Message) {
			oldVal := old.(*Brightness)
			newVal := change.(*Brightness)
			if request.Delta {
				newVal.LevelPercent += oldVal.LevelPercent
			}
			capLevelPercent(newVal)
		}))
	if err != nil {
		return nil, err
	}
	return res.(*Brightness), nil
}

func (s *MemoryDevice) PullBrightness(request *PullBrightnessRequest, server LightApi_PullBrightnessServer) error {
	for event := range s.brightness.Pull(server.Context(), resource.WithReadMask(request.ReadMask), resource.WithUpdatesOnly(request.UpdatesOnly)) {
		brightness := event.Value.(*Brightness)
		// don't emit progress if the caller doesn't want it
		if request.ExcludeRamping {
			progress := brightness.GetBrightnessTween().GetProgress()
			if progress != 0 && progress != 100 {
				continue
			}
		}

		change := &PullBrightnessResponse_Change{
			Name:       request.Name,
			Brightness: brightness,
			ChangeTime: timestamppb.New(event.ChangeTime),
		}
		err := server.Send(&PullBrightnessResponse{
			Changes: []*PullBrightnessResponse_Change{change},
		})
		if err != nil {
			return err
		}
	}

	return server.Context().Err()
}

func capLevelPercent(next *Brightness) {
	if next.LevelPercent < 0 {
		next.LevelPercent = 0
	}
	if next.LevelPercent > 100 {
		next.LevelPercent = 100
	}
}
