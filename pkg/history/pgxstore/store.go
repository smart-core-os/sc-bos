// Package pgxstore provides an implementation of history.Store backed by a Postgres database.
// The historical records for all stores are stored in a single table disambiguated via the source parameter.
package pgxstore

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/pkg/history"
)

//go:embed schema.sql
var schemaSql string

func SetupDB(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.BeginTxFunc(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, schemaSql)
		return err
	})
}

func New(ctx context.Context, source, connStr string, opts ...Option) (*Store, error) {
	pool, err := pgxpool.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect %w", err)
	}

	return SetupStoreFromPool(ctx, source, pool, opts...)
}

func SetupStoreFromPool(ctx context.Context, source string, pool *pgxpool.Pool, opts ...Option) (*Store, error) {
	err := SetupDB(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("setup %w", err)
	}

	return NewStoreFromPool(source, pool, opts...), nil
}

const LargeMaxCount = 1e7

func NewStoreFromPool(source string, pool *pgxpool.Pool, opts ...Option) *Store {
	s := &Store{
		slice:  slice{pool: pool, source: source},
		now:    time.Now,
		logger: zap.NewNop(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if count := s.maxCount; count > LargeMaxCount {
		s.logger.Warn("maxCount is high, this may cause performance issues", zap.Int64("maxCount", count))
	}
	return s
}

type Store struct {
	slice

	now    func() time.Time
	logger *zap.Logger

	maxAge   time.Duration
	maxCount int64
}

func (s *Store) Insert(ctx context.Context, at time.Time, payload []byte) (history.Record, int64, error) {
	r := history.Record{
		CreateTime: at,
		Payload:    payload,
	}

	row := s.pool.QueryRow(ctx, "INSERT INTO history (source, create_time, payload) VALUES ($1, $2, $3) RETURNING id",
		s.source, at, payload)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		return history.Record{}, -1, err
	}

	return r, id, nil
}

func (s *Store) Append(ctx context.Context, payload []byte) (history.Record, error) {
	now := s.now()
	r := history.Record{
		CreateTime: now,
		Payload:    payload,
	}

	r, id, err := s.Insert(ctx, now, payload)
	if err != nil {
		return history.Record{}, err
	}

	r.ID = strconv.FormatInt(id, 10)

	if err := s.gc(now); err != nil {
		// gc failure is not critical to the Append call, so just log it.
		// The next Append will have another chance to gc.
		s.logger.Warn("gc failed", zap.Error(err))
	}
	return r, nil
}

func (s *Store) gc(now time.Time) error {
	if s.maxAge == 0 && s.maxCount == 0 {
		return nil
	}

	if s.maxAge > 0 {
		t := now.Add(-s.maxAge)
		_, err := s.pool.Exec(context.Background(), "DELETE FROM history WHERE source = $1 AND create_time < $2", s.source, t)
		if err != nil {
			return err
		}
	}
	if s.maxCount > 0 {
		// We use create_time here as a substitute for a strict incremental id.
		// At most we leak records equal to the collisions of create_time, which should be minimal.
		sql := fmt.Sprintf(`DELETE FROM history WHERE source = $1 AND create_time < (SELECT create_time FROM history WHERE source = $1 ORDER BY create_time DESC LIMIT 1 OFFSET %d)`, s.maxCount)
		_, err := s.pool.Exec(context.Background(), sql, s.source)
		if err != nil {
			return err
		}
	}
	return nil
}

type slice struct {
	pool     *pgxpool.Pool
	source   string // distinguishes between this store and other stores that use the same table
	from, to history.Record
}

func (s slice) Slice(from, to history.Record) history.Slice {
	return slice{
		pool:   s.pool,
		source: s.source,
		from:   from,
		to:     to,
	}
}

func (s slice) Read(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, false)
}

func (s slice) ReadDesc(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, true)
}

func (s slice) read(ctx context.Context, into []history.Record, desc bool) (int, error) {
	var where []string
	var args []any
	where, args = s.sourceClause(where, args)
	where, args, err := s.readRangeClause(where, args)
	if err != nil {
		return 0, err
	}

	orderBy := "id ASC"
	if desc {
		orderBy = "id DESC"
	}

	sql := fmt.Sprintf("SELECT id, create_time, payload FROM history WHERE %s ORDER BY %s LIMIT %v", strings.Join(where, " AND "), orderBy, len(into))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var id int64
		err := rows.Scan(&id, &into[i].CreateTime, &into[i].Payload)
		if err != nil {
			return 0, err
		}
		into[i].ID = idFromSql(id)
		i++
	}
	return i, nil
}

func (s slice) Len(ctx context.Context) (int, error) {
	var where []string
	var args []any
	where, args = s.sourceClause(where, args)
	where, args, err := s.lenRangeClause(where, args)
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("SELECT COUNT(*) FROM history WHERE %s", strings.Join(where, " AND "))
	row := s.pool.QueryRow(ctx, sql, args...)
	var count int
	err = row.Scan(&count)
	return count, err
}

func (s slice) sourceClause(clauses []string, args []any) ([]string, []any) {
	return append(clauses, fmt.Sprintf("source = $%d", len(args)+1)), append(args, s.source)
}

// lenRangeClause returns the where clauses and arguments for the Len operation.
func (s slice) lenRangeClause(clauses []string, args []any) ([]string, []any, error) {
	switch {
	case s.from.ID != "":
		id, err := idToSql(s.from.ID)
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, fmt.Sprintf("id >= $%d", len(args)+1))
		args = append(args, id)
	case !s.from.CreateTime.IsZero():
		clauses = append(clauses, fmt.Sprintf("create_time >= $%d", len(args)+1))
		args = append(args, s.from.CreateTime)
	}
	switch {
	case s.to.ID != "":
		id, err := idToSql(s.to.ID)
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, fmt.Sprintf("id < $%d", len(args)+1))
		args = append(args, id)
	case !s.to.CreateTime.IsZero():
		clauses = append(clauses, fmt.Sprintf("create_time < $%d", len(args)+1))
		args = append(args, s.to.CreateTime)
	}
	return clauses, args, nil
}

// readRangeClause returns the where clause and arguments for the Read operations.
func (s slice) readRangeClause(clauses []string, args []any) ([]string, []any, error) {
	switch {
	case s.from.ID != "":
		id, err := idToSql(s.from.ID)
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, fmt.Sprintf("id >= $%d", len(args)+1))
		args = append(args, id)
	case !s.from.CreateTime.IsZero():
		sourceIdx := len(args) + 1
		args = append(args, s.source)
		timeIdx := len(args) + 1
		args = append(args, s.from.CreateTime)
		clauses = append(clauses, fmt.Sprintf(`id >= (
	select min(id) from history where source = $%[1]d and create_time = (
		select min(create_time) from history where source = $%[1]d and create_time >= $%[2]d
	)
)`, sourceIdx, timeIdx))
	}
	switch {
	case s.to.ID != "":
		id, err := idToSql(s.to.ID)
		if err != nil {
			return nil, nil, err
		}
		clauses = append(clauses, fmt.Sprintf("id < $%d", len(args)+1))
		args = append(args, id)
	case !s.to.CreateTime.IsZero():
		sourceIdx := len(args) + 1
		args = append(args, s.source)
		timeIdx := len(args) + 1
		args = append(args, s.to.CreateTime)
		clauses = append(clauses, fmt.Sprintf(`id <= (
	select max(id) from history where source = $%[1]d and create_time = (
		select max(create_time) from history where source = $%[1]d and create_time < $%[2]d
	)
)`, sourceIdx, timeIdx))
	}
	return clauses, args, nil
}

func idToSql(id string) (int64, error) {
	return strconv.ParseInt(id, 16, 64)
}

func idFromSql(id int64) string {
	return strconv.FormatInt(id, 16)
}
