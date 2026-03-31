// Package boltstore provides an implementation of history.Store with records stored in a bolthold database.
package boltstore

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/timshannon/bolthold"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/history"
)

type Store struct {
	slice // sorted by id, which is createTime+dedupe index
	now   func() time.Time

	maxAge   time.Duration
	maxCount int64

	logger *zap.Logger
}

func NewFromDb(ctx context.Context, db *bolthold.Store, source string, opts ...Option) (history.Store, error) {
	b := []byte(source)

	s := &Store{
		now:    time.Now,
		logger: zap.NewNop(),
	}
	for _, opt := range opts {
		opt(s)
	}

	err := db.Bolt().Update(func(tx *bolt.Tx) error {
		var err error
		_, err = tx.CreateBucketIfNotExists(b)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.slice = slice{
		db:     db,
		bucket: b,
	}

	// clean out old entries on startup
	err = s.gc(s.now())
	if err != nil {
		s.logger.Warn("gc failed", zap.Error(err))
	}

	return s, nil
}

func (s *Store) Append(ctx context.Context, payload []byte) (history.Record, error) {
	now := s.now()
	r := history.Record{
		ID:         createTimeToID(now),
		CreateTime: now,
		Payload:    payload,
	}

	err := s.db.Bolt().Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return s.db.InsertIntoBucket(b, r.ID, r)
	})
	if err != nil {
		return history.Record{}, err
	}

	if err := s.gc(now); err != nil {
		// gc failure is not critical to the Append call, so just log it.
		// The next Append will have another chance to gc.
		s.logger.Warn("gc failed", zap.Error(err))
	}

	return r, nil
}

func createTimeToID(now time.Time) string {
	return strconv.FormatInt(now.UnixNano(), 10)
}

// gc removes records older than now minus the specified maxAge period, or records over maxCount.
func (s *Store) gc(now time.Time) error {
	if s.maxAge == 0 && s.maxCount == 0 {
		return nil
	}
	var ageErr, countErr error
	if s.maxAge > 0 {
		before := now.Add(-s.maxAge)
		// remove records older than `maxAge`
		ageErr = s.db.Bolt().Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.bucket)
			return s.db.DeleteMatchingFromBucket(b, &history.Record{}, bolthold.Where("CreateTime").Lt(before))
		})
	}
	if s.maxCount > 0 {
		// remove records over maxCount
		countErr = s.db.Bolt().Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.bucket)
			q := bolthold.Query{}
			var r []history.Record
			err := s.db.FindInBucket(b, &r, q.SortBy("CreateTime").Reverse().Limit(1).Skip(int(s.maxCount)))
			if err != nil {
				return err
			}
			if len(r) == 0 {
				return nil
			}
			return s.db.DeleteMatchingFromBucket(b, &history.Record{}, bolthold.Where("CreateTime").Le(r[0].CreateTime))
		})
	}

	return errors.Join(ageErr, countErr)
}

type slice struct {
	db *bolthold.Store

	bucket   []byte // distinguishes between this store and other stores that use the same db
	from, to history.Record
}

func (s slice) Slice(from, to history.Record) history.Slice {
	return slice{
		db:     s.db,
		bucket: s.bucket,
		from:   from,
		to:     to,
	}
}

func (s slice) getQuery() *bolthold.Query {
	var query *bolthold.Query
	if !s.from.IsZero() {
		if s.from.ID != "" {
			query = bolthold.Where("ID").Ge(s.from.ID)
		} else if !s.from.CreateTime.IsZero() {
			query = bolthold.Where("CreateTime").Ge(s.from.CreateTime)
		}
	}
	if !s.to.IsZero() {
		if s.to.ID != "" {
			if query == nil {
				query = bolthold.Where("ID").Lt(s.to.ID)
			} else {
				query = query.And("ID").Lt(s.to.ID)
			}
		} else if !s.to.CreateTime.IsZero() {
			if query == nil {
				query = bolthold.Where("CreateTime").Lt(s.to.CreateTime)
			} else {
				query = query.And("CreateTime").Lt(s.to.CreateTime)
			}
		}
	}
	if query == nil {
		query = &bolthold.Query{}
	}
	return query.SortBy("CreateTime")
}

// Read reads records and places them into dst, with the oldest records at index 0.
func (s slice) Read(ctx context.Context, dst []history.Record) (int, error) {
	return s.read(ctx, dst, false)
}

// ReadDesc reads records and places them into dst, with the newest records at index 0.
func (s slice) ReadDesc(ctx context.Context, dst []history.Record) (int, error) {
	return s.read(ctx, dst, true)
}

// read returns up to len(into) records, between from and to.
// When reverse is false, record 0 will be the oldest, when true it will be the newest.
func (s slice) read(ctx context.Context, into []history.Record, reverse bool) (int, error) {
	query := s.getQuery()
	if reverse {
		query = query.Reverse()
	}
	maxLen := len(into)
	query = query.Limit(maxLen)

	var records []history.Record

	err := s.db.Bolt().View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return s.db.FindInBucket(b, &records, query)
	})
	if err != nil {
		return 0, err
	}

	copy(into, records)

	// todo: this should be upgraded to use the new min() function in Go 1.21 when the project is updated
	return int(math.Min(float64(maxLen), float64(len(records)))), nil
}

func (s slice) Len(ctx context.Context) (int, error) {
	query := s.getQuery()
	var count int
	err := s.db.Bolt().View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		var err error
		count, err = s.db.CountInBucket(b, &history.Record{}, query)
		return err
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}
