package source

import (
	"context"

	"github.com/smart-core-os/sc-bos/pkg/auto"
)

type Services struct {
	auto.Services
	Publisher Publisher
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type PublisherFunc func(ctx context.Context, topic string, payload any) error

func (p PublisherFunc) Publish(ctx context.Context, topic string, payload any) error {
	return p(ctx, topic, payload)
}
