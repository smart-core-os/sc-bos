package pubcache

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/sc-golang/pkg/resource"
)

const (
	MinBackoff   = 10 * time.Second
	MaxBackoff   = 5 * time.Minute
	PollInterval = 5 * time.Minute
)

// A Cache will keep a local copy of a Publication in sync with a publication server. Updates are one-directional -
// it is not possible to apply updates locally to the cache.
// Once Pull is called, the Cache will continue updating its local mirror of the publication in the background,
// until ctx is cancelled.
// If Storage is provided, then the Cache will load its initial value from the Storage if present, and store all
// received publication updates into the Cache.
// Do not copy a Cache or modify its fields once any methods have been called.
type Cache struct {
	logger  *zap.Logger
	ctx     context.Context // Updating will stop when this ctx ends.
	source  traits.PublicationApiClient
	device  string
	pubID   string
	storage Storage // Optional; when nil, storage is not used.

	startBackgroundOnce sync.Once
	initialised         chan struct{}   // closed once a valid publication value is available in latest
	latest              *resource.Value // do not attempt to read until initialised is closed
}

// New constructs a Cache.
// Background tasks are not started immediately, they will begin once Pull is called for the first time.
// The Context ctx can be used to stop the Cache's background tasks.
// Storage is optional; if nil, storage won't be used.
func New(ctx context.Context, source traits.PublicationApiClient, device string, pubID string, opts ...CacheOption) *Cache {
	if ctx == nil {
		panic("parameter ctx is required")
	}
	if source == nil {
		panic("parameter source is required")
	}

	c := &Cache{
		logger: zap.NewNop(),
		ctx:    ctx,
		source: source,
		device: device,
		pubID:  pubID,

		initialised: make(chan struct{}),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Pull will return the current cached value of the publication, followed by all updates.
// When ctx ends, the returned channel will be closed.
// If no current value is available (for example, because the server is offline and the publication is not present in
// Storage), then the initial value may be delayed indefinitely.
// The returned channel does not apply backpressure, so if the consumer is slow, some intermediate values of the
// publication may be missed.
func (c *Cache) Pull(ctx context.Context) <-chan *traits.Publication {
	ch := make(chan *traits.Publication)
	c.startBackground()

	go func() {
		defer close(ch)

		// c.latest may be nil or invalid until c.initialised is closed, so wait until then
		select {
		case <-ctx.Done():
			return
		case <-c.initialised:
		}

		for change := range c.latest.Pull(ctx) {
			select {
			case ch <- change.Value.(*traits.Publication):
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// starts the background updater task
// idempotent - does nothing if called again
func (c *Cache) startBackground() {
	c.startBackgroundOnce.Do(func() {
		go c.runBackground()
	})
}

func (c *Cache) runBackground() {
	// if the publication is cached in storage, pre-initialise latest to that
	if c.storage != nil {
		ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
		initial, err := c.storage.LoadPublication(ctx, c.pubID)
		cancel()

		if err == nil {
			err = c.commitPublication(initial, false)
			if err != nil {
				c.logger.Error("failed to commit publication from storage", zap.Error(err), zap.String("pub", c.pubID))
			}
		} else if !errors.Is(err, ErrPublicationNotFound) {
			c.logger.Error("unexpected error looking up publication in storage", zap.Error(err), zap.String("pub", c.pubID))
		}
	}

	_ = c.runUpdate()
}

func (c *Cache) commitPublication(pub *traits.Publication, store bool) error {
	// the job value might not be initialised yet
	if c.latest == nil {
		c.latest = resource.NewValue(
			resource.WithInitialValue(pub),
			resource.WithNoDuplicates(),
		)
		close(c.initialised) // signal that c.latest is valid
	} else {
		_, err := c.latest.Set(pub)
		if err != nil {
			return err
		}
	}

	// commit the publication to storage
	if c.storage != nil && store {
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		defer cancel()
		return c.storage.StorePublication(ctx, pub)
	}
	return nil
}

// pulls publication updates from the source and stores them in the job's value.
// when the first value is received, initialises job.value and closes job.initialised
// started will be true if at least one value was successfully received from the server
// all publication values retrieved are also stored in the cache's Storage
func (c *Cache) pullPublication() (err error, started bool) {
	stream, err := c.source.PullPublication(c.ctx, &traits.PullPublicationRequest{
		Name: c.device,
		Id:   c.pubID,
	})
	if err != nil {
		return err, false
	}

	for {
		res, err := stream.Recv()
		if err != nil {
			return err, started
		}

		for _, change := range res.Changes {
			err = c.commitPublication(change.Publication, true)
			if err != nil {
				return err, started
			}
			// we managed to receive and commit at least one change
			started = true
		}
	}
}

func (c *Cache) pollPublication() (err error, started bool) {
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	// ticker only fires for the first time after PollInterval has elapsed, so we must poll once outside the loop
	err = c.pollPublicationOnce()
	if err != nil {
		return
	}
	started = true

	for range ticker.C {
		err = c.pollPublicationOnce()
		if err != nil {
			return
		}
	}
	panic("unreachable") // ticker will never stop by itself
}

// fetches and commits a publication once, using the GetPublication gRPC call
func (c *Cache) pollPublicationOnce() error {
	ctx, cancel := context.WithTimeout(c.ctx, time.Minute)
	defer cancel()

	pub, err := c.source.GetPublication(ctx, &traits.GetPublicationRequest{
		Name: c.device,
		Id:   c.pubID,
	})

	if err != nil {
		return err
	}

	return c.commitPublication(pub, true)
}

// calls pullPublication in a loop with exponential backoff
// if pullPublication returns an error with gRPC status code Unimplemented, switches to polling
// this function only returns once c.ctx has completed
func (c *Cache) runUpdate() error {
	ctx := c.ctx
	backoff := MinBackoff
	for {
		err, started := c.pullPublication()
		if status.Code(err) == codes.Unimplemented {
			c.logger.Warn("device does not support PullPublication; switching to polling", zap.Duration("interval", PollInterval))
			err, started = c.pollPublication()
		}
		if err != nil {
			c.logger.Error("Pull publication failed", zap.Error(err),
				zap.String("pub", c.pubID), zap.String("device", c.device))
		}

		c.logger.Debug("pull publication backoff", zap.String("pub", c.pubID), zap.Duration("backoff", backoff))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		if started {
			backoff = MinBackoff
		} else {
			// back off with an exponential timeout
			backoff *= 2
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
		}
	}
}

type CacheOption func(cache *Cache)

func WithLogger(logger *zap.Logger) CacheOption {
	return func(cache *Cache) {
		cache.logger = logger
	}
}

func WithStorage(storage Storage) CacheOption {
	return func(cache *Cache) {
		cache.storage = storage
	}
}
