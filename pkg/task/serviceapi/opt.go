package serviceapi

import (
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type Option func(a *Api)

func WithKnownTypes(t ...string) Option {
	return func(a *Api) {
		a.knownTypes = append(a.knownTypes, t...)
	}
}

func WithKnownTypesFromMapKeys[M ~map[string]T, T any](m M) Option {
	return func(a *Api) {
		a.knownTypes = slices.Grow(a.knownTypes, len(m))
		for k := range m {
			a.knownTypes = append(a.knownTypes, k)
		}
	}
}

func WithLogger(l *zap.Logger) Option {
	return func(a *Api) {
		a.logger = l
	}
}

func WithStore(s Store) Option {
	return func(a *Api) {
		a.store = s
	}
}
