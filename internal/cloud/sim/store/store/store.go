// Package store provides a SQLite-based data storage layer for the cloudsim application.
//
// # Entities and Relationships
//
// Sites represent physical locations. Nodes are BOS controllers associated with a site.
// ConfigVersions store versioned binary configurations, associated with a node.
// ConfigDeployments track the status of deploying a config version to a node
// (pending -> in_progress -> completed/failed/cancelled).
// BinaryArtefacts store versioned BOS software payloads (podman save tarballs), scoped to a
// platform and optionally a site. Because the payloads are large, they are stored as external files
// on disk (named by artefact id, in a directory beside the database file) rather than as BLOBs; the
// row holds only metadata. CreateBinaryArtefact streams a payload to disk and ReadBinaryArtefactPayload
// streams it back, so neither buffers it in full.
// BinaryDeployments roll out an artefact to a specific node, sharing the same state machine as
// config deployments.
//
// Deleting a site cascades to all associated nodes, config versions, and deployments. Artefact rows
// also cascade, but their files are removed only by SweepOrphanArtefacts (run at startup behind a
// flag), since SQLite cascade deletes cannot touch the filesystem.
//
// # Usage
//
// All operations must occur within Read or Write transactions. List queries use page token
// pagination (ordered by ID). This package uses sqlc to generate type-safe code from SQL.
//
// To regenerate after modifying queries/queries.sql or migrations/*.sql, use `go generate`
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

const appID = 0x5C0502

//go:embed migrations/*.sql
var migrationsFS embed.FS

var schema = sqlite.MustLoadVersionedSchema(migrationsFS, "migrations")

// Store provides access to the cloudsim SQLite database.
// Use OpenStore for persistent storage or NewMemoryStore for non-persistent uses e.g. testing.
// All operations must be performed within Read or Write transactions.
type Store struct {
	db *sqlite.Database
	// artefactsDir holds binary-artefact payload files, named by artefact id. It lives next to the
	// database file (see artefactsDirForDBPath) for persistent stores, or in a temp dir for memory
	// stores.
	artefactsDir string
	// ownsArtefactsDir is true when artefactsDir is a temp dir created by NewMemoryStore, which Close
	// should remove. Persistent stores leave their directory in place.
	ownsArtefactsDir bool
}

// artefactsDirForDBPath derives the binary-artefact directory that sits next to the database file,
// named from its basename: foo/bar.db -> foo/bar-artefacts.
func artefactsDirForDBPath(dbPath string) string {
	base := filepath.Base(dbPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(filepath.Dir(dbPath), name+"-artefacts")
}

// OpenStore opens a SQLite database at the given path and applies migrations to update to the latest schema version.
// The database file will be created if it doesn't exist, as will the sibling directory holding update
// artefact payload files.
// Returns an error if the database cannot be opened or migrations fail.
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

	artefactsDir := artefactsDirForDBPath(path)
	if err := os.MkdirAll(artefactsDir, 0o755); err != nil {
		return nil, err
	}

	return &Store{
		db:           db,
		artefactsDir: artefactsDir,
	}, nil
}

// NewMemoryStore creates an in-memory SQLite database for testing.
// The database is fully functional but exists only in memory and will be
// lost when the store is closed. Binary artefact payloads are written to a temporary directory that
// Close removes. Panics if migrations or temp-dir creation fail.
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

	artefactsDir, err := os.MkdirTemp("", "cloudsim-artefacts-")
	if err != nil {
		panic(err)
	}

	return &Store{db: db, artefactsDir: artefactsDir, ownsArtefactsDir: true}
}

// Close closes the database connection, and removes the artefacts directory if it is a temporary one
// owned by the store (memory stores only).
func (s *Store) Close() error {
	err := s.db.Close()
	if s.ownsArtefactsDir && s.artefactsDir != "" {
		err = errors.Join(err, os.RemoveAll(s.artefactsDir))
	}
	return err
}

// Read executes a read-only transaction.
// Multiple concurrent read transactions are allowed.
// The transaction is automatically rolled back when the function returns.
func (s *Store) Read(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.ReadTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

// Write executes a read-write transaction.
// Only one write transaction can execute at a time.
// The transaction is automatically committed if the function returns nil,
// or rolled back if it returns an error.
func (s *Store) Write(ctx context.Context, f func(tx *Tx) error) error {
	return s.db.WriteTx(ctx, func(tx *sql.Tx) error {
		storeTx := &Tx{Queries: queries.New(tx)}
		return f(storeTx)
	})
}

// Tx wraps the generated queries with additional transaction methods.
// Access to Tx is only available within Read or Write transaction callbacks.
// All query methods are generated by sqlc from queries/queries.sql.
type Tx struct {
	*queries.Queries
}
