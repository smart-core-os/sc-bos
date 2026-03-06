package pubcache

import (
	"context"

	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"

	"github.com/smart-core-os/sc-bos/sc-api/go/traits"
)

// NewBoltStorage create a Storage using a Bolt database as the backend.
// The passed database must have been opened read-write.
// This implementation is not context aware; contexts passed to methods will be ignored.
func NewBoltStorage(db *bbolt.DB, bucket []byte) Storage {
	if db == nil {
		panic("db must not be nil")
	}

	// copy the bucket identifier to guard against external mutation
	bucketCopy := make([]byte, len(bucket))
	copy(bucketCopy, bucket)

	return &boltStorage{db: db, bucket: bucketCopy}
}

type boltStorage struct {
	db     *bbolt.DB
	bucket []byte
}

func (b *boltStorage) LoadPublication(_ context.Context, pubID string) (*traits.Publication, error) {
	var serialized []byte

	err := b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			return ErrPublicationNotFound
		}

		serialized = bucket.Get([]byte(pubID))
		return nil
	})
	if err != nil {
		return nil, err
	}

	if serialized == nil {
		return nil, ErrPublicationNotFound
	}

	// deserialize the binary protobuf
	pub := &traits.Publication{}
	err = proto.Unmarshal(serialized, pub)
	return pub, err
}

func (b *boltStorage) StorePublication(_ context.Context, pub *traits.Publication) error {
	if pub.GetId() == "" {
		return ErrPublicationInvalid
	}

	serialized, err := proto.Marshal(pub)
	if err != nil {
		return err
	}

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(b.bucket)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(pub.Id), serialized)
		return err
	})
}

func (b *boltStorage) ListPublications(_ context.Context) (pubIDs []string, err error) {
	err = b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			// absence of the bucket simply means that no publications are stored
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			pubIDs = append(pubIDs, string(k))
			return nil
		})
	})
	return
}

func (b *boltStorage) DeletePublication(_ context.Context, pubID string) (present bool, err error) {
	err = b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			// no bucket means there are no publications stored
			present = false
			return nil
		}

		key := []byte(pubID)
		existing := bucket.Get(key)
		if existing != nil {
			present = true
			return bucket.Delete(key)
		} else {
			present = false
			return nil
		}
	})
	return
}
