// Package memstore provides an implementation of history.Store with records stored in memory.
package memstore

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/smart-core-os/sc-bos/pkg/history"
)

type Store struct {
	// mtx protects slice during calls which read or modify the underlying slice
	mtx sync.Mutex
	// slice is sorted by id, which is createTime+dedupe index
	slice
	now func() time.Time

	maxAge   time.Duration
	maxCount int64
}

func New(opts ...Option) *Store {
	s := &Store{now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func SetNow(s *Store, now func() time.Time) func() {
	old := s.now
	s.now = now
	return func() {
		s.now = old
	}
}

func (s *Store) Append(_ context.Context, payload []byte) (history.Record, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	now := s.now()
	l := len(s.slice)
	if l > 0 && s.slice[l-1].CreateTime.After(now) {
		return history.Record{}, errors.New("time is running backwards")
	}

	r := history.Record{Payload: payload, CreateTime: now, ID: createTimeToID(now)}
	if l > 0 && s.slice[l-1].CreateTime == now {
		// todo: cap this memory growth. Should be ok for now as this isn't likely in production
		// 000 sorts after 00 after 0, so adding a 0 to the time maintains the sort order
		r.ID = s.slice[l-1].ID + "0"
	}
	s.slice = append(s.slice, r)
	s.gc(now)
	return r, nil
}

func (s *Store) Slice(from, to history.Record) history.Slice {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.slice.Slice(from, to)
}

func (s *Store) Read(ctx context.Context, into []history.Record) (int, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.slice.Read(ctx, into)
}

func (s *Store) ReadDesc(ctx context.Context, into []history.Record) (int, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.slice.ReadDesc(ctx, into)
}

func (s *Store) Len(ctx context.Context) (int, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.slice.Len(ctx)
}

func (s *Store) gc(now time.Time) {
	if s.maxAge == 0 && s.maxCount == 0 {
		return
	}

	if s.maxAge > 0 {
		// find the first record that is "not older" than maxAge and drop everything before it
		if i, ok := s.indexOf(history.Record{CreateTime: now.Add(-s.maxAge)}); ok {
			s.slice = s.slice[i:]
		}
	}
	if s.maxCount > 0 {
		if l := int64(len(s.slice)); l > s.maxCount {
			s.slice = s.slice[l-s.maxCount:]
		}
	}
}

type slice []history.Record // sorted by id, which is createTime+dedupe

func (rs slice) indexOf(r history.Record) (int, bool) {
	id := computeId(r)
	return sort.Find(len(rs), func(i int) int {
		return strings.Compare(id, rs[i].ID)
	})
}

func (rs slice) indexFrom(from history.Record) (int, bool) {
	if from.IsZero() {
		return 0, true
	}
	return rs.indexOf(from)
}

func (rs slice) indexTo(to history.Record) (int, bool) {
	if to.IsZero() {
		return len(rs), true
	}
	return rs.indexOf(to)
}

func (rs slice) Slice(from, to history.Record) history.Slice {
	fromIndex, _ := rs.indexFrom(from)
	toIndex, _ := rs.indexTo(to)
	return rs[fromIndex:toIndex]
}

func (rs slice) Read(_ context.Context, into []history.Record) (int, error) {
	return copy(into, rs), nil
}

func (rs slice) ReadDesc(_ context.Context, into []history.Record) (int, error) {
	i := copy(into, rs[max(len(rs)-len(into), 0):])
	slices.Reverse(into[:i])
	return i, nil
}

func (rs slice) Len(_ context.Context) (int, error) {
	return len(rs), nil
}

func computeId(r history.Record) string {
	if r.ID != "" {
		return r.ID
	}
	return createTimeToID(r.CreateTime)
}

func createTimeToID(t time.Time) string {
	return t.In(time.UTC).Format(time.RFC3339Nano)
}
