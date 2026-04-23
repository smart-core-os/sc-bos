package service

import (
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

// SystemCheck is the driver's persistent top-level health indicator, created by the framework
// and passed via driver.Services.SystemCheck. Drivers call MarkFailed when connectivity is
// lost and MarkRunning when it is restored. Drivers must call Dispose in their stop handler.
type SystemCheck interface {
	Dispose()
	MarkRunning()
	MarkFailed(err error)
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
