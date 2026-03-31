// Package sqlitestore implements a history store using SQLite as the backend database.
//
// Implements the history.Store interface (call Database.OpenStore).
// Many sources can share the same Database instance.
//
// Limitations:
//   - Maximum of 1,000,000 records can be created within the same millisecond timestamp, due to the combined
//     timestamp+serial RecordID format.
package sqlitestore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/sqlite"
	"github.com/smart-core-os/sc-bos/pkg/history"
)

const appID = 0x5C0502

//go:embed migrations/*.sql
var migrationFS embed.FS

var schema = sqlite.MustLoadVersionedSchema(migrationFS, "migrations")

type Database struct {
	db     *sqlite.Database
	logger *zap.Logger
}

func Open(ctx context.Context, path string, options ...Option) (*Database, error) {
	o := resolveOptions(options...)

	db, err := sqlite.Open(ctx, path,
		sqlite.WithApplicationID(appID),
		sqlite.WithLogger(o.logger),
		// automatically shrink the database file when data is deleted
		sqlite.WithWriterPragma("auto_vacuum", "FULL"),
	)
	if err != nil {
		return nil, err
	}

	return open(ctx, db, o)
}

func OpenMemory(ctx context.Context, options ...Option) (*Database, error) {
	o := resolveOptions(options...)
	db := sqlite.OpenMemory(sqlite.WithLogger(o.logger))
	return open(ctx, db, o)
}

func open(ctx context.Context, db *sqlite.Database, o *opts) (*Database, error) {
	err := db.Migrate(ctx, schema)
	if err != nil {
		return nil, err
	}

	return &Database{
		db:     db,
		logger: o.logger,
	}, nil
}

func resolveOptions(options ...Option) *opts {
	o := &opts{}
	for _, option := range options {
		option(o)
	}
	if o.logger == nil {
		o.logger = zap.NewNop()
	}
	return o
}

func (d *Database) OpenStore(source string, opts ...WriteOption) *Store {
	return &Store{
		database: d,
		source:   source,
		opts:     opts,
	}
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Insert(ctx context.Context, record Record, opts ...WriteOption) (Record, error) {
	toInsert := []Record{record}
	err := d.InsertBulk(ctx, toInsert, opts...)
	if err != nil {
		return Record{}, err
	}
	return toInsert[0], nil
}

// InsertBulk inserts multiple records into the database in a single transaction.
// Mutates records to set the ID field for each inserted record.
func (d *Database) InsertBulk(ctx context.Context, records []Record, opts ...WriteOption) error {
	o := writeOpts{}
	for _, option := range opts {
		option(&o)
	}

	txTime := time.Now()
	err := d.db.WriteTx(ctx, func(tx *sql.Tx) (err error) {
		sources, err := newSourceAllocator(ctx, tx)
		if err != nil {
			return err
		}
		idAlloc, err := newRecordIDAllocator(ctx, tx)
		if err != nil {
			return err
		}

		stmt, err := tx.PrepareContext(ctx, "INSERT INTO history (id, source_id, payload) VALUES (?, ?, ?);")
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Join(err, stmt.Close())
		}()

		modifiedSources := make(map[string]struct{})
		for i, record := range records {
			modifiedSources[record.Source] = struct{}{}
			srcID, err := sources.getOrInsertSource(ctx, record.Source)
			if err != nil {
				return err
			}
			recordID, err := idAlloc.allocateRecordID(ctx, record.CreateTime)
			if err != nil {
				return err
			}

			_, err = stmt.ExecContext(ctx, recordID, srcID, record.Payload)
			if err != nil {
				return err
			}
			records[i].ID = recordID
			records[i].CreateTime = recordID.Timestamp() // truncated and without time zone
		}
		for source := range modifiedSources {
			if o.enableMaxCount {
				_, err = d.trimCount(ctx, tx, source, o.maxCount)
				if err != nil {
					return err
				}
			}
			// only one of trimTime or trimAge can be set
			// calculate what time t, if any, to trim to
			t := o.trimTime
			if d := o.trimAge; t.IsZero() && d > 0 {
				t = txTime.Add(-d)
			}
			if !t.IsZero() {
				_, err = d.trimTime(ctx, tx, source, t)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	return err
}

// Read reads records from the database into the provided slice.
// If source is non-empty, it filters records by source.
// If from and to are non-zero, it filters records by ID range, greater-or-equal-to from, and less-than to.
// Returns number of records read and any error encountered.
// Size of the slice limits the number of records to read.
func (d *Database) Read(ctx context.Context, source string, from, to RecordID, desc bool, into []Record) (n int, err error) {
	err = d.read(ctx, source, from, to, desc, func(record Record) bool {
		if n >= len(into) {
			return false
		}
		into[n] = record
		n++
		return true
	})
	return n, err
}

func (d *Database) read(ctx context.Context, source string, from, to RecordID, desc bool, cb func(Record) bool) error {
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		filters, args := buildFilters(source, from, to)

		order := "ASC"
		if desc {
			order = "DESC"
		}

		query := fmt.Sprintf(`
			SELECT history.id, history_sources.source, payload
            FROM history
            INNER JOIN history_sources ON history.source_id = history_sources.id
			WHERE %s
            ORDER BY history.id %s;
            `, filters, order)

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var (
				id      RecordID
				src     string
				payload []byte
			)
			err = rows.Scan(&id, &src, &payload)
			if err != nil {
				return err
			}
			if !cb(Record{ID: id, CreateTime: id.Timestamp(), Source: src, Payload: payload}) {
				// Callback returned false, stop reading more records
				break
			}
		}
		return rows.Err()
	})
	return err
}

func (d *Database) Count(ctx context.Context, source string, from, to RecordID) (int, error) {
	var count int
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		filters, args := buildFilters(source, from, to)

		query := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM history
			INNER JOIN history_sources ON history.source_id = history_sources.id
			WHERE %s;
		`, filters)

		err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
		return err
	})
	return count, err
}

func (d *Database) Size(ctx context.Context) (int64, error) {
	var size int64
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&size)
	})
	return size, err
}

// TrimTime deletes all records with a timestamp before the specified time.
// If source is non-empty, only deletes records from that source. If source is empty, deletes records from all sources.
// Returns the number of records deleted.
func (d *Database) TrimTime(ctx context.Context, source string, before time.Time) (int64, error) {
	var deleted int64
	err := d.db.WriteTx(ctx, func(tx *sql.Tx) (err error) {
		deleted, err = d.trimTime(ctx, tx, source, before)
		return
	})
	return deleted, err
}

func (d *Database) trimTime(ctx context.Context, tx *sql.Tx, source string, before time.Time) (deleted int64, err error) {
	recordID := MakeRecordID(before, 0)
	if source == "" {
		deleted, err = d.deleteAllBefore(ctx, tx, recordID)
	} else {
		deleted, err = d.deleteSourceBefore(ctx, tx, source, recordID)
	}
	return deleted, err
}

// deletes all records with ID less than the specified ID, for all sources
func (d *Database) deleteAllBefore(ctx context.Context, tx *sql.Tx, id RecordID) (int64, error) {
	res, err := tx.ExecContext(ctx, "DELETE FROM history WHERE id < ?", id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// deletes all records for the specified source name with ID less than the specified ID
func (d *Database) deleteSourceBefore(ctx context.Context, tx *sql.Tx, source string, id RecordID) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		DELETE FROM history WHERE id < ? AND source_id IN (
			SELECT id FROM history_sources WHERE source = ?
		)
	`, id, source)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// deletes all records for the specified source name
func (d *Database) deleteSource(ctx context.Context, tx *sql.Tx, source string) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		DELETE FROM history WHERE source_id IN (SELECT id FROM history_sources WHERE source = ?)
    `, source)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// get record ID for the nth newest record within the specified source
// e.g. n=0 returns the newest record, n=1 the second newest, etc.
// Returns sql.ErrNoRows if there are fewer than n+1 records for the source.
func (d *Database) nthNewestRecordID(ctx context.Context, tx *sql.Tx, source string, n int64) (RecordID, error) {
	var boundary RecordID
	err := tx.QueryRowContext(ctx, `
		SELECT history.id FROM history
		INNER JOIN history_sources ON history.source_id = history_sources.id
		WHERE history_sources.source = ?
		ORDER BY history.id DESC LIMIT 1 OFFSET ?
   	`, source, n).Scan(&boundary)
	if err != nil {
		return 0, err
	}
	return boundary, nil
}

// TrimCount deletes the oldest records for the specified source if the total number of records exceeds the given limit.
// After deletion, the total number of records for that source will be at most 'limit'.
// Source must be non-empty.
// Returns the number of records deleted.
func (d *Database) TrimCount(ctx context.Context, source string, limit int64) (int64, error) {
	if source == "" {
		return 0, errors.New("source must be non-empty")
	}
	if limit < 0 {
		return 0, errors.New("limit must be non-negative")
	}
	var deleted int64
	err := d.db.WriteTx(ctx, func(tx *sql.Tx) (err error) {
		deleted, err = d.trimCount(ctx, tx, source, limit)
		return err
	})
	return deleted, err
}

func (d *Database) trimCount(ctx context.Context, tx *sql.Tx, source string, limit int64) (int64, error) {
	// we either need to delete all records older than boundary, or all records
	var (
		boundary  RecordID
		deleteAll bool
		err       error
	)
	if limit == 0 {
		deleteAll = true
	} else {
		boundary, err = d.nthNewestRecordID(ctx, tx, source, limit-1)
		if errors.Is(err, sql.ErrNoRows) {
			deleteAll = true
		} else if err != nil {
			return 0, err
		}
	}

	var deleted int64
	if deleteAll {
		deleted, err = d.deleteSource(ctx, tx, source)
	} else {
		deleted, err = d.deleteSourceBefore(ctx, tx, source, boundary)
	}
	return deleted, err
}

type Record struct {
	ID         RecordID
	Source     string
	CreateTime time.Time
	Payload    []byte
}

type Store struct {
	database *Database
	source   string
	opts     []WriteOption // passed to every write operation

	from, to history.Record
}

func (s *Store) Append(ctx context.Context, payload []byte) (history.Record, error) {
	now := time.Now()
	record, err := s.database.Insert(ctx, Record{
		Source:     s.source,
		CreateTime: now,
		Payload:    payload,
	}, s.opts...)
	if err != nil {
		return history.Record{}, err
	}
	return history.Record{
		ID:         record.ID.String(),
		CreateTime: record.CreateTime,
		Payload:    record.Payload,
	}, nil
}

func (s *Store) Slice(from, to history.Record) history.Slice {
	return &Store{
		database: s.database,
		source:   s.source,
		from:     from,
		to:       to,
		opts:     s.opts,
	}
}

func (s *Store) Read(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, false)
}

func (s *Store) ReadDesc(ctx context.Context, into []history.Record) (int, error) {
	return s.read(ctx, into, true)
}

func (s *Store) read(ctx context.Context, into []history.Record, desc bool) (int, error) {
	if len(into) == 0 {
		return 0, nil
	}

	fromBound, err := calcBound(s.from)
	if err != nil {
		return 0, err
	}
	toBound, err := calcBound(s.to)
	if err != nil {
		return 0, err
	}

	var n int
	err = s.database.read(ctx, s.source, fromBound, toBound, desc, func(record Record) bool {
		into[n] = history.Record{
			ID:         record.ID.String(),
			CreateTime: record.CreateTime,
			Payload:    record.Payload,
		}
		n++
		return n < len(into)
	})
	return n, err
}

func (s *Store) Len(ctx context.Context) (int, error) {
	fromBound, err := calcBound(s.from)
	if err != nil {
		return 0, err
	}
	toBound, err := calcBound(s.to)
	if err != nil {
		return 0, err
	}

	return s.database.Count(ctx, s.source, fromBound, toBound)
}

func calcBound(limit history.Record) (RecordID, error) {
	if limit.ID != "" {
		id, err := ParseRecordID(limit.ID)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	if !limit.CreateTime.IsZero() {
		id := MakeRecordID(limit.CreateTime, 0)
		return id, nil
	}

	// zero value is sentinel meaning "no bound"
	return 0, nil
}

// builds an SQL term for filtering records based on source and ID range.
// Also returns a slice of parameters to be passed when executing the query.
// If no filtering is to be performed, returns a dummy condition to maintain valid SQL syntax.
func buildFilters(source string, from, to RecordID) (string, []any) {
	var filters []string
	var args []any
	if source != "" {
		filters = []string{"history_sources.source = ?"}
		args = []any{source}
	}

	if from != 0 {
		filters = append(filters, "history.id >= ?")
		args = append(args, from)
	}

	if to != 0 {
		filters = append(filters, "history.id < ?")
		args = append(args, to)
	}

	if len(filters) > 0 {
		return strings.Join(filters, " AND "), args
	} else {
		return "1 = 1", nil // no filtering, return a dummy condition
	}
}

type RecordID int64

func MakeRecordID(ts time.Time, serial int) RecordID {
	if serial < 0 || serial >= 1_000_000 {
		panic("serial must be in range [0, 999999]")
	}
	return RecordID(ts.UnixMilli()*1_000_000 + int64(serial))
}

func ParseRecordID(s string) (RecordID, error) {
	id, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, ErrInvalidRecordID
	}
	if id < 0 {
		return 0, ErrInvalidRecordID
	}
	return RecordID(id), nil
}

func (id RecordID) Timestamp() time.Time {
	return time.UnixMilli(int64(id) / 1_000_000)
}

func (id RecordID) Serial() int {
	return int(int64(id) % 1_000_000)
}

func (id RecordID) Next() (RecordID, bool) {
	if id.Serial() == 1_000_000-1 {
		// incrementing the serial number would overflow the millisecond timestamp
		return 0, false
	}
	return RecordID(int64(id) + 1), true
}

func (id RecordID) String() string {
	return fmt.Sprintf("%016X", int64(id))
}

// sourceAllocator maintains a cache of source name to ID mappings, so that we don't have to query the database for
// it each time we insert.
// Can only be used within a single transaction (e.g. bulk insert) because the sources table, which this caches,
// can be modified between transactions.
type sourceAllocator struct {
	sources    map[string]int64 // source name to ID mapping
	queryStmt  *sql.Stmt        // prepared statement for querying source IDs
	insertStmt *sql.Stmt        // prepared statement for inserting new sources if they don't exist
}

func newSourceAllocator(ctx context.Context, tx *sql.Tx) (*sourceAllocator, error) {
	sc := &sourceAllocator{
		sources: make(map[string]int64),
	}

	var err error
	sc.queryStmt, err = tx.PrepareContext(ctx, "SELECT id FROM history_sources WHERE source = ?")
	if err != nil {
		return nil, err
	}

	sc.insertStmt, err = tx.PrepareContext(ctx, "INSERT INTO history_sources (source) VALUES (?) RETURNING id")
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// getOrInsertSource returns the ID in the sources table for the given source name.
// If the source doesn't exist within the table, it is inserted.
// Results are cached so calling getOrInsertSource for the same source multiple times is efficient.
func (sa *sourceAllocator) getOrInsertSource(ctx context.Context, source string) (int64, error) {
	if id, ok := sa.sources[source]; ok {
		return id, nil
	}

	// Query the ID of the source
	var id int64
	err := sa.queryStmt.QueryRowContext(ctx, source).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = sa.insertStmt.QueryRowContext(ctx, source).Scan(&id)
	}
	if err != nil {
		return 0, err
	}

	sa.sources[source] = id
	return id, nil
}

// recordIDAllocator is used to efficiently allocate RecordID within a single transaction.
// Each RecordID encodes both a timestamp and a serial number. When a new row is to be inserted, we need to determine
// the next free serial number for the given timestamp.
// recordIDAllocator caches the highest serial number used for each seen timestamp, so we don't have to query it
// on every INSERT.
// Can only be used within a single transaction (e.g. bulk insert) because new records can be inserted between transactions.
type recordIDAllocator struct {
	maxRecordIDByTS map[int64]RecordID // map from millisecond timestamp to the max record ID within that ts
	stmt            *sql.Stmt          // prepared statement for querying the max record ID for a given millisecond timestamp
}

func newRecordIDAllocator(ctx context.Context, tx *sql.Tx) (*recordIDAllocator, error) {
	ra := &recordIDAllocator{
		maxRecordIDByTS: make(map[int64]RecordID),
	}

	var err error
	ra.stmt, err = tx.PrepareContext(ctx, "SELECT MAX(id) FROM history WHERE id >= (1000000 * ?1) AND id < (1000000 * (?1 + 1))")
	if err != nil {
		return nil, err
	}

	return ra, nil
}

// allocateRecordID returns the next free RecordID for the given timestamp.
// Queries the database if this timestamp hasn't been seen before. The returned RecordID is considered allocated,
// so subsequent calls with the same timestamp will return incrementing RecordIDs.
func (ra *recordIDAllocator) allocateRecordID(ctx context.Context, ts time.Time) (RecordID, error) {
	var (
		anyExist bool
		highest  RecordID
	)

	highest, ok := ra.maxRecordIDByTS[ts.UnixMilli()]
	if ok {
		anyExist = true
	} else {
		// Query the highest record ID for this millisecond timestamp
		var id sql.NullInt64
		err := ra.stmt.QueryRowContext(ctx, ts.UnixMilli()).Scan(&id)
		if err != nil {
			return 0, err
		}
		if id.Valid {
			highest = RecordID(id.Int64)
			anyExist = true
		}
	}

	var next RecordID
	if anyExist {
		next, ok = highest.Next()
		if !ok {
			// would overflow into the next timestamp
			return 0, ErrTooManyRecords
		}
	} else {
		next = MakeRecordID(ts, 0)
	}
	ra.maxRecordIDByTS[ts.UnixMilli()] = next
	return next, nil
}

var (
	ErrTooManyRecords  = errors.New("too many records")
	ErrInvalidRecordID = errors.New("invalid record ID format")
)
