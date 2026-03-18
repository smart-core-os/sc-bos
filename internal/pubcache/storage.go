package pubcache

import (
	"context"
	"errors"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/pkg/proto/publicationpb"
)

var (
	ErrPublicationNotFound = errors.New("publication not found in storage")
	ErrPublicationInvalid  = errors.New("invalid publication cannot be stored")
)

// A Storage implementation is used by the cache to persist a single version for each publication it caches.
type Storage interface {
	// LoadPublication retrieves a cached publication from storage.
	// If a publication with the given ID is not cached, returns ErrPublicationNotFound
	LoadPublication(ctx context.Context, pubID string) (*publicationpb.Publication, error)
	// StorePublication will cache a publication in storage, silently replacing any existing publication with the same ID.
	// If pub cannot be stored because it is invalid, returns ErrPublicationInvalid
	StorePublication(ctx context.Context, pub *publicationpb.Publication) error
	// ListPublications will list the publication IDs of all publications currently stored in the cache.
	ListPublications(ctx context.Context) (pubIDs []string, err error)
	// DeletePublication will remove the publication with the given ID from the cache storage.
	// If the publication was not in the storage, no error is returned but return value present will be false.
	DeletePublication(ctx context.Context, pubID string) (present bool, err error)
}

// NewMemoryStorage returns a non-persistent Storage implementation, which stores all publications in memory.
// Publications are copied when inserted or retrieved, so cached publications cannot be accidentally modified.
// The returned Storage is empty.
func NewMemoryStorage() Storage {
	return &memoryStorage{store: make(map[string]*publicationpb.Publication)}
}

type memoryStorage struct {
	m     sync.RWMutex
	store map[string]*publicationpb.Publication
}

func (m *memoryStorage) LoadPublication(_ context.Context, pubID string) (*publicationpb.Publication, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	pub, ok := m.store[pubID]
	if !ok {
		return nil, ErrPublicationNotFound
	}
	// prevent the client from modifying the copy inside the storage
	pub = clonePublication(pub)
	return pub, nil
}

func (m *memoryStorage) StorePublication(_ context.Context, pub *publicationpb.Publication) error {
	// a publication with an empty ID is almost certainly a mistake
	if pub.GetId() == "" {
		return ErrPublicationInvalid
	}

	// to prevent the client from modifying the publication in the store, we must do a deep clone
	pub = clonePublication(pub)

	m.m.Lock()
	defer m.m.Unlock()
	m.store[pub.Id] = pub
	return nil
}

func (m *memoryStorage) ListPublications(_ context.Context) (pubIDs []string, err error) {
	m.m.RLock()
	defer m.m.RUnlock()

	pubIDs = make([]string, 0, len(m.store))
	for id := range m.store {
		pubIDs = append(pubIDs, id)
	}
	return pubIDs, nil
}

func (m *memoryStorage) DeletePublication(_ context.Context, pubID string) (present bool, err error) {
	m.m.Lock()
	defer m.m.Unlock()

	_, present = m.store[pubID]
	delete(m.store, pubID)
	return
}

func clonePublication(pub *publicationpb.Publication) *publicationpb.Publication {
	return proto.Clone(pub).(*publicationpb.Publication)
}
