package stores

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	// StorageHealthHighPercent is the disk utilisation percentage (0-100) at or above which
	// a store's storage HealthCheck reports HIGH. When zero, a default of 90 is used.
	StorageHealthHighPercent float32 `json:"storageHealthHighPercent,omitempty"`
	// Local directory for storing database files.
	DataDir string      `json:"-"`
	Logger  *zap.Logger `json:"-"`
}

type PostgresConfig struct {
	pgxutil.RoleConfig
	// MaxSizeBytes is the maximum storage size, in bytes, the Postgres history store is
	// allowed to use. Postgres does not report a disk capacity itself, so this must be set
	// to enable the storage utilisation HealthCheck and the bytes.capacity DataRetention
	// field for the Postgres store. When zero, no capacity is reported and no check is raised.
	MaxSizeBytes uint64 `json:"maxSizeBytes,omitempty"`
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
	pools         pgxutil.Pools
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

	pools, err := pgxutil.ConnectRoles(context.Background(), s.cfg.RoleConfig)
	if err != nil {
		s.latestErrTime = time.Now()
		return fail(err)
	}

	s.pools = pools
	s.r, s.w, s.admin = pools.Read, pools.Write, pools.Admin
	return s.r, s.w, s.admin, nil
}

// PostgresPoolsFor returns the postgres connection pools a subsystem should use
// given its storage config rc.
//
// If rc is zero the shared node pools are returned; these are owned by the
// Stores and must not be closed by the caller. Otherwise a dedicated set of
// pools is opened from rc and closed automatically when ctx is done.
func (s *postgresStore) PostgresPoolsFor(ctx context.Context, rc pgxutil.RoleConfig) (pgxutil.Pools, error) {
	if rc.IsZero() {
		r, w, admin, err := s.Postgres()
		if err != nil {
			return pgxutil.Pools{}, err
		}
		return pgxutil.Pools{Read: r, Write: w, Admin: admin}, nil
	}
	pools, err := pgxutil.ConnectRoles(ctx, rc)
	if err != nil {
		return pgxutil.Pools{}, err
	}
	context.AfterFunc(ctx, pools.Close)
	return pools, nil
}

func (s *postgresStore) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = fmt.Errorf("postgres: %w", ErrStoreClosed)
	if s.r == nil {
		return nil
	}

	// Close each distinct pool once; roles may share an underlying pool.
	s.pools.Close()
	s.pools = pgxutil.Pools{}
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
