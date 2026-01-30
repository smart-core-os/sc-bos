package stores

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/history/sqlitestore"
)

var (
	ErrStoreClosed        = errors.New("closed")
	ErrStoreNotConfigured = errors.New("not configured")
)

// Config configures shared storage (dbs) for systems on this node.
type Config struct {
	Postgres *PostgresConfig `json:"postgres,omitempty"`
	// Local directory for storing database files.
	DataDir string      `json:"-"`
	Logger  *zap.Logger `json:"-"`
}

type PostgresConfig struct {
	pgxutil.ConnectConfig
}

const retryConnectDelay = 100 * time.Millisecond

// New creates a new Stores instance based on cfg, which must be non-nil.
func New(cfg *Config) *Stores {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Stores{
		sqliteHistoryStore: sqliteHistoryStore{
			path:   filepath.Join(cfg.DataDir, defaultSqliteHistoryFile),
			logger: logger.Named("sqlite"),
		},
	}
	if cfg.Postgres != nil {
		s.postgresStore.cfg = cfg.Postgres
	}
	return s
}

const defaultSqliteHistoryFile = "history.sqlite3"

// Stores provides access to shared storage connections/clients.
type Stores struct {
	postgresStore
	sqliteHistoryStore
}

// Close closes all stores.
func (s *Stores) Close() error {
	return multierr.Combine(
		s.postgresStore.close(),
		s.sqliteHistoryStore.close(),
	)
}

type postgresStore struct {
	cfg *PostgresConfig

	mu            sync.Mutex
	r, w, admin   *pgxpool.Pool
	err           error
	latestErrTime time.Time
}

// Postgres returns shared postgres connection pools.
// The pools can be used for read, write, and admin (alter table) operations.
func (s *postgresStore) Postgres() (r, w, admin *pgxpool.Pool, _ error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fail := func(err error) (_, _, _ *pgxpool.Pool, _ error) {
		s.err = err
		return nil, nil, nil, err
	}

	// during shutdown, a caller may sporadically try to get a store
	// after close has been called
	if errors.Is(s.err, ErrStoreClosed) {
		return nil, nil, nil, s.err
	}

	if s.r != nil {
		return s.r, s.w, s.admin, nil
	}

	if s.cfg == nil {
		return fail(fmt.Errorf("postgres: %w", ErrStoreNotConfigured))
	}

	if time.Since(s.latestErrTime) < retryConnectDelay {
		// prevent rapid reconnect attempts
		return nil, nil, nil, fmt.Errorf("%w [cached]", s.err)
	}

	// todo: support r, w, and admin pools
	pool, err := pgxutil.Connect(context.Background(), s.cfg.ConnectConfig)
	if err != nil {
		s.latestErrTime = time.Now()
		return fail(err)
	}

	s.r, s.w, s.admin = pool, pool, pool
	return s.r, s.w, s.admin, nil
}

func (s *postgresStore) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = fmt.Errorf("postgres: %w", ErrStoreClosed)
	if s.r == nil {
		return nil
	}

	s.r.Close()
	s.w.Close()
	s.admin.Close()
	s.r = nil
	s.w = nil
	s.admin = nil

	return nil
}

type sqliteHistoryStore struct {
	path   string
	logger *zap.Logger

	mu sync.Mutex
	db *sqlitestore.Database
}

// SqliteHistory returns a shared sqlite history database.
// The database is lazily opened on the first call.
// Do not close the database - it will be closed when the Stores are closed.
func (s *sqliteHistoryStore) SqliteHistory(ctx context.Context) (*sqlitestore.Database, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		db, err := sqlitestore.Open(ctx, s.path, sqlitestore.WithLogger(s.logger))
		if err != nil {
			return nil, err
		}
		s.db = db
	}

	return s.db, nil
}

func (s *sqliteHistoryStore) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
