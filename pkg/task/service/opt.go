package service

import (
	"context"
	"encoding/json"
	"time"
)

type Option[T any] interface {
	apply(l *Service[T])
}

func DefaultOpts[C any]() []Option[C] {
	return []Option[C]{
		WithNow[C](time.Now),
		WithParser(func(data []byte) (C, error) {
			var c C
			err := json.Unmarshal(data, &c)
			return c, err
		}),
	}
}

// OptionFunc adapts a func of the correct signature to implement Option.
type OptionFunc[T any] func(l *Service[T])

//lint:ignore U1000 this is used via interface Option, but staticcheck can't see it (because of generics?)
func (o OptionFunc[T]) apply(l *Service[T]) {
	o(l)
}

// WithParser configures a Service to use the given parse func instead of the default json.Unmarshaler.
func WithParser[T any](parse ParseFunc[T]) Option[T] {
	return OptionFunc[T](func(l *Service[T]) {
		l.parse = parse
	})
}

// WithNow configures a service with a custom time functions instead of the default time.Now.
// Useful for testing.
func WithNow[T any](now func() time.Time) Option[T] {
	return OptionFunc[T](func(l *Service[T]) {
		l.now = now
	})
}

// WithOnStop sets a function on the Service that will be called each time Service.Stop is executed.
// The onStop func should not invoke any lifecycle methods on the created service as this may result in a deadlock.
func WithOnStop[T any](onStop func()) Option[T] {
	return OptionFunc[T](func(l *Service[T]) {
		l.onStop = onStop
	})
}

// Disposable is implemented by resources whose lifetime should match the service lifetime.
type Disposable interface {
	Dispose()
}

// SystemCheck is a health check whose lifecycle is managed by the service framework.
// Use [WithSystemCheck] to register one with a service.
type SystemCheck interface {
	Disposable
	// MarkRunning is called automatically when applyConfig returns nil.
	MarkRunning()
	// MarkFailed is called automatically when applyConfig returns a non-nil error.
	MarkFailed(err error)
}

// WithSystemCheck registers a health check whose lifecycle matches the service lifetime.
//
// Unlike resources created inside applyConfig, the check is NOT disposed between retry
// attempts, so it remains visible (e.g. in the health registry) during connection failures.
//
// The check is updated automatically:
//   - MarkFailed(err) is called when applyConfig returns an error.
//   - MarkRunning() is called when applyConfig returns nil.
//
// Drivers may call MarkFailed before returning an error to provide richer context; the
// framework will call it again as a fallback if the driver does not.
// Dispose is called exactly once when the service stops, after any WithOnStop handlers.
// A nil check is silently ignored.
func WithSystemCheck[T any](check SystemCheck) Option[T] {
	if check == nil {
		return OptionFunc[T](func(*Service[T]) {})
	}
	return OptionFunc[T](func(l *Service[T]) {
		orig := l.apply
		l.apply = func(ctx context.Context, cfg T) error {
			err := orig(ctx, cfg)
			if err != nil {
				check.MarkFailed(err)
			} else {
				check.MarkRunning()
			}
			return err
		}
		// Append to onStop so disposal runs AFTER driver cleanup (WithOnStop handlers).
		existing := l.onStop
		l.onStop = func() {
			if existing != nil {
				existing()
			}
			check.Dispose()
		}
	})
}

// WithRetry configures a service to retry ApplyFunc when it returns an error.
func WithRetry[T any](opts ...RetryOption) Option[T] {
	return OptionFunc[T](func(l *Service[T]) {
		retry := defaultRetryOptions
		for _, opt := range opts {
			opt(&retry)
		}
		l.retry = &retry
	})
}
