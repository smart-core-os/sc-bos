package resource

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidateTweenOnUpdate(name string, tween *types.Tween) error {
	if tween == nil {
		return nil
	}
	if err := ValidateNoProgress(name, tween); err != nil {
		return err
	}
	if err := ValidateNonNegativeDuration(name, tween); err != nil {
		return err
	}

	return nil
}

func ValidateNoProgress(name string, tween *types.Tween) error {
	if tween.GetProgress() != 0 {
		return status.Errorf(codes.InvalidArgument, "%v_tween.progress should not be set, got %v", name, tween.GetProgress())
	}
	return nil
}

func ValidateNonNegativeDuration(name string, tween *types.Tween) error {
	if tween.GetTotalDuration().AsDuration() < 0 {
		return status.Errorf(codes.InvalidArgument, "%v_tween.total_duration should be non-negative, got %v", name, tween.GetTotalDuration())
	}
	return nil
}

var (
	TweenComplete = io.EOF
	TweenStopped  = errors.New("tween stopped")
)

type Tween struct {
	Start           float32
	End             float32
	Duration        time.Duration
	Easing          ease.TweenFunc
	FramesPerSecond float32

	tween *gween.Tween
}

func NewTween(opts ...TweenOption) *Tween {
	t := &Tween{}
	for _, opt := range DefaultTweenOptions {
		opt(t)
	}
	for _, opt := range opts {
		opt(t)
	}

	t.tween = gween.New(t.Start, t.End, float32(t.Duration.Milliseconds()), t.Easing)

	return t
}

func (t *Tween) NextFrame() (float32, error) {
	return 0, nil
}

type TweenOption func(*Tween)

var DefaultTweenOptions = []TweenOption{
	WithBounds(0, 100),
	WithFramesPerSecond(10),
	WithDuration(time.Second),
	WithEasing(ease.Linear),
}

func WithBounds(start, end float32) TweenOption {
	if start == end {
		panic(fmt.Errorf("start cannot equal end: %v == %v", start, end))
	}
	return func(tween *Tween) {
		tween.Start = start
		tween.End = end
	}
}

func WithFramesPerSecond(fps float32) TweenOption {
	return func(tween *Tween) {
		tween.FramesPerSecond = fps
	}
}

func WithDuration(duration time.Duration) TweenOption {
	return func(tween *Tween) {
		tween.Duration = duration
	}
}

func WithEasing(easing ease.TweenFunc) TweenOption {
	return func(tween *Tween) {
		tween.Easing = easing
	}
}
