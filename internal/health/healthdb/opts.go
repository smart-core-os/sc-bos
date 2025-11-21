package healthdb

import (
	"time"

	"go.uber.org/zap"
)

type opts struct {
	logger      *zap.Logger
	trimOnWrite trimAfterWriteOption
}

type Option func(*opts)

// WithLogger is an option to set the logger used by the store.
func WithLogger(logger *zap.Logger) Option {
	return func(s *opts) {
		s.logger = logger
	}
}

type trimAfterWriteOption struct {
	minCount int64
	maxCount int64
	maxAge   time.Duration
}

func (o trimAfterWriteOption) toTrimOptions() TrimOptions {
	dst := TrimOptions{
		MinCount: o.minCount,
		MaxCount: o.maxCount,
	}
	if o.maxAge > 0 {
		dst.Before = time.Now().Add(-o.maxAge)
	}
	return dst
}

// WithTrimOnWrite is an option to set how records are removed after writing.
func WithTrimOnWrite(minCount, maxCount int64, maxAge time.Duration) Option {
	return func(o *opts) {
		o.trimOnWrite = trimAfterWriteOption{
			minCount: minCount,
			maxCount: maxCount,
			maxAge:   maxAge,
		}
	}
}
