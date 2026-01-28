// Package store is a SQLite-based persistent store that has a mimimal representation of sites, nodes and configs
// for local development of the BOS to Smart Core Connect integration.
package store

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

const appID = 0x5C0502

// Store provides access to the cloudsim SQLite database.
type Store struct {
	db *sqlite.Database
}

// OpenStore opens a SQLite database at the given path and applies migrations.
func OpenStore(ctx context.Context, path string, logger *zap.Logger) (*Store, error) {
	db, err := sqlite.Open(ctx, path,
		sqlite.WithLogger(logger),
		sqlite.WithApplicationID(appID),
	)
	if err != nil {
		return nil, err
	}

	err = db.Migrate(ctx, schema)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,
	}, nil
}

// NewMemoryStore creates an in-memory SQLite database for testing.
func NewMemoryStore(logger *zap.Logger) *Store {
	db := sqlite.OpenMemory(
		sqlite.WithLogger(logger),
		sqlite.WithApplicationID(appID),
	)

	err := db.Migrate(context.Background(), schema)
	if err != nil {
		// this can only happen if the migrations are broken
		panic(err)
	}

	return &Store{db: db}
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Read executes a read-only transaction.
func (s *Store) Read(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.ReadTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

// Write executes a read-write transaction.
func (s *Store) Write(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.WriteTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

// Tx wraps the generated queries with additional transaction methods.
type Tx struct {
	*queries.Queries
}
