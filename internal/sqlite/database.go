// Package sqlite provides an easy way to work with SQLite databases.
//
// The database is opened with Open (for file DBs) or OpenMemory (for in-memory DBs).
// A Database automatically uses connection pooling for best performance.
//
// Supports a basic schema migration system. Migrations can be loaded from a directory using LoadVersionedSchema,
// and applied to the database. The package keeps track of the current schema version in the database's
// user_version PRAGMA.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"go.uber.org/zap"
)

// Database represents a SQLite database.
// To access the database, use the ReadTx and WriteTx methods to run transactions.
//
// Database uses a connection pool. It is safe to use the same Database instance from multiple goroutines.
//
// Call Close to release resources when the database is no longer needed.
type Database struct {
	// use separate connection pools for reading and writing, to avoid unnecessary contention
	writer    *sql.DB
	reader    *sql.DB
	logger    *zap.Logger
	freeMemDB string // if set, memdb with this name will be deleted when the Database is closed.
}

type Option func(*opts)

// Open opens a Database on the local filesystem.
// The database is created if it does not already exist.
func Open(ctx context.Context, path string, options ...Option) (*Database, error) {
	o := resolveOpts(options...)

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// The database/sql.DB is a connection pool. We want to have certain PRAGMAs run at the start of each connection.
	// By putting them in the URI, they are automatically run for each new connection.
	// If we just ran the PRAGMAs after opening the connection, they would only be on one of the connections.
	readerURI := buildFileURI(path, false, o.readerPragmas)
	writerURI := buildFileURI(path, true, o.writerPragmas)
	return openURI(ctx, readerURI, writerURI, &o)
}

// OpenMemory opens an in-memory Database.
// When the Database is closed, all data is lost.
func OpenMemory(options ...Option) *Database {
	name := fmt.Sprintf("anonymous-memdb-%d", memDBCounter.Add(1))
	memdb.Create(name, nil)
	o := resolveOpts(options...)

	readerURI := buildMemoryURI(name, false, o.readerPragmas)
	writerURI := buildMemoryURI(name, true, o.writerPragmas)
	db, err := openURI(context.Background(), readerURI, writerURI, &o)
	if err != nil {
		// opening an in-memory database doesn't access any external resources, so this should never happen
		panic("failed to open memory database: " + err.Error())
	}
	db.freeMemDB = name
	return db
}

var memDBCounter atomic.Uint64

func openURI(ctx context.Context, readerURI, writerURI string, o *opts) (_ *Database, err error) {
	writer, err := sql.Open("sqlite3", writerURI)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = writer.Close()
		}
	}()
	// one connection in the writer pool, never closed
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	writer.SetConnMaxIdleTime(0)
	// early check to see if the database is actually available
	err = writer.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	reader, err := sql.Open("sqlite3", readerURI)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = reader.Close()
		}
	}()
	reader.SetMaxOpenConns(max(4, runtime.NumCPU()))

	db := &Database{
		reader: reader,
		writer: writer,
		logger: o.logger,
	}
	if o.expectedAppID != 0 {
		// check the application ID
		// - for a fresh DB (withAppID=0, version=0), set the application ID to the expected value
		// - for an existing DB, check the application ID matches the expected value
		err = db.WriteTx(ctx, func(tx *sql.Tx) error {
			appID, err := getApplicationID(ctx, tx)
			if err != nil {
				return err
			}
			version, err := getUserVersion(ctx, tx)
			if err != nil {
				return err
			}

			if appID == 0 && version == 0 {
				err = setApplicationID(ctx, tx, o.expectedAppID)
			} else if appID != o.expectedAppID {
				err = fmt.Errorf("%w: expected %d, got %d", ErrApplicationIDMismatch, o.expectedAppID, appID)
			}
			return err
		})
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func resolveOpts(options ...Option) opts {
	var o opts
	for _, option := range options {
		option(&o)
	}
	if o.logger == nil {
		o.logger = zap.NewNop()
	}
	return o
}

func WithLogger(logger *zap.Logger) Option {
	return func(o *opts) {
		o.logger = logger
	}
}

// ApplicationID is a globally unique identifier for the application that will use the database.
type ApplicationID uint32

// WithApplicationID sets the expected application ID for the database.
// When the database is opened, the application ID is compared to the given value.
// If both the application ID and the database version are 0, as is the case for a new empty database, the
// application ID is initialised to the given value.
// Otherwise, if the application ID does not match the given value, an error is returned.
func WithApplicationID(appID ApplicationID) Option {
	return func(o *opts) {
		o.expectedAppID = appID
	}
}

// WithReaderPragma sets a PRAGMA for the reader connections only.
func WithReaderPragma(key, value string) Option {
	return func(o *opts) {
		o.readerPragmas = append(o.readerPragmas, pragma{Key: key, Value: value})
	}
}

// WithWriterPragma sets a PRAGMA for the writer connection only.
func WithWriterPragma(key, value string) Option {
	return func(o *opts) {
		o.writerPragmas = append(o.writerPragmas, pragma{Key: key, Value: value})
	}
}

// WithPragma sets a PRAGMA for both the reader and writer connections.
func WithPragma(key, value string) Option {
	return func(o *opts) {
		o.readerPragmas = append(o.readerPragmas, pragma{Key: key, Value: value})
		o.writerPragmas = append(o.writerPragmas, pragma{Key: key, Value: value})
	}
}

type opts struct {
	logger        *zap.Logger
	expectedAppID ApplicationID
	readerPragmas []pragma
	writerPragmas []pragma
}

type pragma struct {
	Key   string
	Value string
}

func (db *Database) Close() error {
	if db.freeMemDB != "" {
		defer memdb.Delete(db.freeMemDB)
	}
	err := db.writer.Close()
	if err != nil {
		return err
	}
	err = db.reader.Close()
	return err
}

func (db *Database) ReadTx(ctx context.Context, f func(tx *sql.Tx) error) (err error) {
	tx, err := db.reader.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		return err
	}
	return db.runTX(tx, f)
}

func (db *Database) WriteTx(ctx context.Context, f func(tx *sql.Tx) error) (err error) {
	tx, err := db.writer.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // uses an IMMEDIATE transaction lock
	})
	if err != nil {
		return err
	}
	return db.runTX(tx, f)
}

func (db *Database) runTX(tx *sql.Tx, f func(tx *sql.Tx) error) (err error) {
	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				db.logger.Error("failed to rollback transaction", zap.Error(rollbackErr),
					zap.NamedError("originalErr", err))
				err = errors.Join(err, rollbackErr)
			}
		}
	}()
	err = f(tx)
	if err != nil {
		// f failed, automatic rollback in the defer above
		return
	}
	return tx.Commit()
}

func (db *Database) Migrate(ctx context.Context, schema Schema) error {
	err := schema.validate()
	if err != nil {
		return err
	}

	target := schema.migrations[len(schema.migrations)-1].Version
	done := false
	for !done {
		err := db.WriteTx(ctx, func(tx *sql.Tx) error {
			version, err := getUserVersion(ctx, tx)
			if err != nil {
				return err
			}
			if version == target {
				done = true
				return nil
			} else if version > target {
				return fmt.Errorf("can't migrate down from schema version %d to %d", version, target)
			}

			next := version + 1
			m, ok := schema.find(next)
			if !ok {
				return fmt.Errorf("no migration found for version %d", next)
			}

			_, err = tx.ExecContext(ctx, m.SQL)
			if err != nil {
				return fmt.Errorf("failed to migrate from schema version %d -> %d: %w", version, next, err)
			}

			err = setUserVersion(ctx, tx, next)
			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func IsForeignKeyError(err error) bool {
	var sqlErr *sqlite3.Error
	if !errors.As(err, &sqlErr) {
		return false
	}

	return errors.Is(sqlErr.ExtendedCode(), sqlite3.CONSTRAINT_FOREIGNKEY)
}

func IsUniqueConstraintError(err error) bool {
	var sqlErr *sqlite3.Error
	if !errors.As(err, &sqlErr) {
		return false
	}

	return errors.Is(sqlErr.ExtendedCode(), sqlite3.CONSTRAINT_UNIQUE)
}

var ErrApplicationIDMismatch = errors.New("database application ID mismatch")

type Timestamp time.Time

func (ft *Timestamp) Scan(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot parse %T as timestamp", v)
	}
	t, err := time.Parse(DateTimeFormat, str)
	if err != nil {
		return err
	}
	*(*time.Time)(ft) = t
	return nil
}

func buildFileURI(path string, writer bool, pragmas []pragma) string {
	query := make(url.Values)
	if writer {
		query.Add("mode", "rwc")
	} else {
		query.Add("mode", "ro")
	}
	query.Add("_timefmt", DateTimeFormat)
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	query.Add("_pragma", "foreign_keys(ON)")
	for _, p := range pragmas {
		query.Add("_pragma", fmt.Sprintf("%s(%s)", p.Key, p.Value))
	}
	uri := &url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(path),
		RawQuery: query.Encode(),
	}

	uriString := uri.String()

	if runtime.GOOS == "windows" {
		uriString = strings.Replace(uriString, "file://", "file:", 1)
	}

	return uriString
}

func buildMemoryURI(name string, writer bool, pragmas []pragma) string {
	query := make(url.Values)
	if writer {
		query.Add("mode", "rwc")
	} else {
		query.Add("mode", "ro")
	}
	query.Add("_timefmt", DateTimeFormat)
	query.Add("_pragma", "foreign_keys(ON)")
	for _, p := range pragmas {
		query.Add("_pragma", fmt.Sprintf("%s(%s)", p.Key, p.Value))
	}
	query.Add("vfs", "memdb")
	uri := &url.URL{
		Scheme:   "file",
		Path:     fmt.Sprintf("/%s", name),
		RawQuery: query.Encode(),
	}
	return uri.String()
}

// DateTimeFormat is the default format for encoding/decoding time.Time values in the database.
// It uses the SQLite format with millisecond precision.
const DateTimeFormat = string(sqlite3.TimeFormat4)
