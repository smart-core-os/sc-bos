// Package healthdb provides a SQLite-based implementation of a health check record store.
package healthdb

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

const appID = 0x5C0503

//go:embed schema/*.sql
var schemaVersionsFS embed.FS
var schema = sqlite.MustLoadVersionedSchema(schemaVersionsFS, "schema")

// DB is a store for Records
type DB struct {
	db *sqlite.Database

	trimAfterWriteOption trimAfterWriteOption
}

func Open(ctx context.Context, path string, options ...Option) (*DB, error) {
	o := &opts{}
	for _, option := range options {
		option(o)
	}
	if o.logger == nil {
		o.logger = zap.NewNop()
	}

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	db, err := sqlite.Open(ctx, path,
		sqlite.WithApplicationID(appID),
		sqlite.WithLogger(o.logger),
	)
	if err != nil {
		return nil, err
	}

	err = db.Migrate(ctx, schema)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) Insert(ctx context.Context, record Record) (Record, error) {
	toInsert := []Record{record}
	err := d.InsertBulk(ctx, toInsert)
	if err != nil {
		return Record{}, err
	}
	return toInsert[0], nil
}

// InsertBulk inserts multiple records into the database in a single transaction.
// Mutates records to set the ID field for each inserted record.
func (d *DB) InsertBulk(ctx context.Context, records []Record) error {
	err := d.db.WriteTx(ctx, func(tx *sql.Tx) (err error) {
		rowIDs, err := newRecordIDAllocator(ctx, tx)
		if err != nil {
			return err
		}
		checkIDs, err := newCheckIDAllocator(ctx, tx)
		if err != nil {
			return err
		}
		auxIDs, err := newAuxAllocator(ctx, tx)
		if err != nil {
			return err
		}

		stmt, err := tx.PrepareContext(ctx, "INSERT INTO health_check_history (id, check_id, aux_id, payload) VALUES (?, ?, ?, ?);")
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Join(err, stmt.Close())
		}()

		for i, record := range records {
			rowID, err := rowIDs.allocateRecordID(ctx, record.CreateTime)
			if err != nil {
				return err
			}
			checkID, err := checkIDs.getOrInsertCheckID(ctx, record.Name, record.CheckID)
			if err != nil {
				return err
			}
			auxID, err := auxIDs.getOrInsertAux(ctx, record.Aux)
			if err != nil {
				return err
			}

			_, err = stmt.ExecContext(ctx, rowID, checkID, auxID, record.Main)
			if err != nil {
				return err
			}
			records[i].ID = rowID
			records[i].CreateTime = rowID.Timestamp() // truncated and without time zone
		}
		err = stmt.Close()
		if err != nil {
			return err
		}

		return d.trimAfterWrite(ctx, tx, records)
	})
	return err
}

// Read reads records from the database into the provided slice.
// If id is zero, all checks are considered.
// If id has a zero id field, all checks for the given name are considered.
// It is an error to provide an id with no name, but an id.
// If from and to are non-zero, it filters records by ID range, greater-or-equal-to from, and less-than to.
// Returns number of records read and any error encountered.
// At most len(into) records will be read.
func (d *DB) Read(ctx context.Context, id CheckID, from, to RecordID, desc bool, into []Record) (n int, err error) {
	if id.Name == "" && id.ID != "" {
		return 0, errors.New("name required for non-zero id")
	}
	err = d.read(ctx, id, from, to, desc, func(record Record) bool {
		if n >= len(into) {
			return false
		}
		into[n] = record
		n++
		return true
	})
	return n, err
}

// ReadLastRecord reads the most recent record for the given CheckID.
// If id is zero, all checks are considered.
// If id has a zero id field, all checks for the given name are considered.
// It is an error to provide an id with no name, but an id.
func (d *DB) ReadLastRecord(ctx context.Context, id CheckID) (Record, error) {
	if id.Name == "" && id.ID != "" {
		return Record{}, errors.New("name required for non-zero id")
	}
	var r Record
	err := d.read(ctx, id, 0, 0, true, func(record Record) bool {
		r = record
		return false // only need the first (last) record
	})
	if err != nil {
		return Record{}, err
	}
	if r.IsZero() {
		return Record{}, sql.ErrNoRows
	}
	return r, nil
}

func (d *DB) read(ctx context.Context, id CheckID, from, to RecordID, desc bool, cb func(Record) bool) error {
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		filters, args := buildFilters(id, from, to)

		order := "ASC"
		if desc {
			order = "DESC"
		}

		query := fmt.Sprintf(`
			SELECT health_check_history.id, health_check_ids.name, health_check_ids.check_id, health_check_aux.payload, health_check_history.payload
			FROM health_check_history
			INNER JOIN health_check_ids on health_check_history.check_id = health_check_ids.id
			INNER JOIN health_check_aux on health_check_history.aux_id = health_check_aux.id
			WHERE %s
			ORDER BY health_check_history.id %s;
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
				id            RecordID
				name, checkID string
				aux, main     []byte
			)
			err = rows.Scan(&id, &name, &checkID, &aux, &main)
			if err != nil {
				return err
			}
			r := Record{
				ID:         id,
				Name:       name,
				CheckID:    checkID,
				CreateTime: id.Timestamp(),
				Main:       main,
				Aux:        aux,
			}
			if !cb(r) {
				// Callback returned false, stop reading more records
				break
			}
		}
		return rows.Err()
	})
	return err
}

func (d *DB) Count(ctx context.Context, id CheckID, from, to RecordID) (int, error) {
	var count int
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		filters, args := buildFilters(id, from, to)

		query := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM health_check_history
			INNER JOIN health_check_ids ON health_check_history.check_id = health_check_ids.id
			WHERE %s;
		`, filters)

		err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
		return err
	})
	return count, err
}

func (d *DB) Size(ctx context.Context) (int64, error) {
	var size int64
	err := d.db.ReadTx(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&size)
	})
	return size, err
}

func (d *DB) Compact(ctc context.Context) error {
	// Compacting the database by vacuuming it
	return d.db.WriteTx(ctc, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctc, "PRAGMA incremental_vacuum(0)")
		return err
	})
}

// TrimOptions specifies criteria for trimming records from the database.
type TrimOptions struct {
	// If non-zero, ensures at least this many records are retained.
	MinCount int64
	// If non-zero, ensures at most this many records are retained.
	// If MaxCount is less than MinCount, MinCount takes precedence.
	MaxCount int64
	// When non-zero, records before this time are eligible for deletion.
	Before time.Time
}

func (o TrimOptions) IsZero() bool {
	return o.MinCount == 0 && o.MaxCount == 0 && o.Before.IsZero()
}

func (d *DB) trimAfterWrite(ctx context.Context, tx *sql.Tx, records []Record) error {
	tOpts := d.trimAfterWriteOption.toTrimOptions()
	if tOpts.IsZero() {
		return nil
	}
	// we only trim the modified checks so we aren't doing excess query work
	seenIDs := make(map[CheckID]struct{})
	for _, record := range records {
		id := CheckID{Name: record.Name, ID: record.CheckID}
		if _, seen := seenIDs[id]; seen {
			continue
		}
		seenIDs[id] = struct{}{}
		_, err := d.trim(ctx, tx, id, tOpts)
		if err != nil {
			return err
		}
	}
	return nil
}

// Trim removes records from the database according to the provided options.
func (d *DB) Trim(ctx context.Context, id CheckID, opts TrimOptions) (int64, error) {
	if id.Name == "" && id.ID != "" {
		return 0, errors.New("name required for non-zero id")
	}
	if opts.IsZero() || opts.MinCount > 0 && opts.MaxCount == 0 && opts.Before.IsZero() {
		// nothing to delete
		return 0, nil
	}
	var deleted int64
	err := d.db.WriteTx(ctx, func(tx *sql.Tx) error {
		n, err := d.trim(ctx, tx, id, opts)
		deleted = n
		return err
	})
	return deleted, err
}

func (d *DB) trim(ctx context.Context, tx *sql.Tx, id CheckID, opts TrimOptions) (_ int64, err error) {
	// Simple case when there are no count-based limits
	if opts.MinCount == 0 && opts.MaxCount == 0 {
		oldestIDToKeep := MakeRecordID(opts.Before, 0)
		return d.deleteChecksBefore(ctx, tx, id, oldestIDToKeep)
	}

	var tsID RecordID
	if !opts.Before.IsZero() {
		tsID = MakeRecordID(opts.Before, 0)
	}

	// when there are counts, we have to delete checks per matching id,
	// which means querying for each id to work out the cutoff point.
	filter, args := buildFilters(id, 0, 0)
	checkIDs, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT id from health_check_ids WHERE %s`, filter), args...)
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Join(err, checkIDs.Close())
	}()

	var totalDeleted int64
	for checkIDs.Next() {
		var idRow int64
		err := checkIDs.Scan(&idRow)
		if err != nil {
			return 0, err
		}
		var oldestID RecordID
		switch {
		case opts.Before.IsZero():
			oldestID, err = d.nthNewestID(ctx, tx, idRow, max(opts.MaxCount, opts.MinCount)-1)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, err
			}
		case opts.MinCount > 0 && opts.MaxCount == 0:
			oldestID, err = d.nthNewestID(ctx, tx, idRow, opts.MinCount-1)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, err
			}
			oldestID = min(oldestID, tsID)
		case opts.MaxCount > 0 && opts.MinCount == 0:
			oldestID, err = d.nthNewestID(ctx, tx, idRow, opts.MaxCount-1)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, err
			}
			oldestID = max(oldestID, tsID)
		default:
			minID, err := d.nthNewestID(ctx, tx, idRow, opts.MinCount-1)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, err
			}
			var maxID RecordID
			if opts.MaxCount >= opts.MinCount {
				var err error
				maxID, err = d.nthNewestID(ctx, tx, idRow, opts.MaxCount-1)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return 0, err
				}
			} else {
				maxID = minID
			}
			switch {
			case minID < tsID:
				oldestID = minID
			case maxID < tsID:
				oldestID = tsID
			default:
				oldestID = maxID
			}
		}
		if oldestID == 0 {
			// nothing to delete for this check ID
			continue
		}
		nDeleted, err := d.deleteChecksBeforeWithID(ctx, tx, idRow, oldestID)
		if err != nil {
			return 0, err
		}
		totalDeleted += nDeleted
	}
	if err := checkIDs.Err(); err != nil {
		return 0, err
	}

	return totalDeleted, nil
}

func (d *DB) deleteChecksBefore(ctx context.Context, tx *sql.Tx, id CheckID, before RecordID) (int64, error) {
	if id.Name == "" {
		res, err := tx.ExecContext(ctx, "DELETE FROM health_check_history WHERE id < ?", before)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	filters, args := buildFilters(id, 0, 0)
	args = append(args, before)
	query := fmt.Sprintf(`
		DELETE FROM health_check_history
		WHERE health_check_history.check_id IN (SELECT id FROM health_check_ids WHERE %s) AND health_check_history.id < ?;
	`, filters)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DB) deleteChecksBeforeWithID(ctx context.Context, tx *sql.Tx, id int64, before RecordID) (int64, error) {
	res, err := tx.ExecContext(ctx, "DELETE FROM health_check_history WHERE check_id = ? AND id < ?", id, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// nthNewestID returns the RecordID for the nth newest check with check_id of id.
func (d *DB) nthNewestID(ctx context.Context, tx *sql.Tx, id, n int64) (RecordID, error) {
	query := fmt.Sprintf(`
		SELECT id
		FROM health_check_history
		WHERE check_id = ?
		ORDER BY id DESC
		LIMIT 1 OFFSET %d;
	`, n)
	var res RecordID
	err := tx.QueryRowContext(ctx, query, id).Scan(&res)
	if err != nil {
		return 0, err
	}
	return res, nil
}

// builds an SQL term for filtering records based on a CheckID and row ID range.
// Also returns a slice of parameters to be passed when executing the query.
// If no filtering is to be performed, returns a dummy condition to maintain valid SQL syntax.
func buildFilters(id CheckID, from, to RecordID) (string, []any) {
	var filters []string
	var args []any
	if id.Name != "" {
		filters = append(filters, "health_check_ids.name = ?")
		args = append(args, id.Name)
	}
	if id.ID != "" {
		filters = append(filters, "health_check_ids.check_id = ?")
		args = append(args, id.ID)
	}

	if from != 0 {
		filters = append(filters, "health_check_history.id >= ?")
		args = append(args, from)
	}

	if to != 0 {
		filters = append(filters, "health_check_history.id < ?")
		args = append(args, to)
	}

	if len(filters) > 0 {
		return strings.Join(filters, " AND "), args
	} else {
		return "1 = 1", nil // no filtering, return a dummy condition
	}
}

type Record struct {
	ID            RecordID
	Name, CheckID string
	CreateTime    time.Time
	Main, Aux     []byte // payloads
}

func (r Record) IsZero() bool {
	return r.ID == 0 &&
		r.Name == "" &&
		r.CheckID == "" &&
		r.CreateTime.IsZero() &&
		len(r.Main) == 0 &&
		len(r.Aux) == 0
}

// RecordID is a combination of Timestamp and deduplicating serial for use as the primary key for history records.
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

type recordIDAllocator struct {
	maxRecordIDByTS mapping[int64, RecordID] // map from millisecond timestamp to the max record ID within that ts
	stmt            *sql.Stmt                // prepared statement for querying the max record ID for a given millisecond timestamp
}

func newRecordIDAllocator(ctx context.Context, tx *sql.Tx) (*recordIDAllocator, error) {
	stmt, err := tx.PrepareContext(ctx, "SELECT MAX(id) FROM health_check_history WHERE id >= (1000000 * ?1) AND id < (1000000 * (?1 + 1))")
	if err != nil {
		return nil, err
	}

	return &recordIDAllocator{
		stmt: stmt,
	}, nil
}

func (ra *recordIDAllocator) allocateRecordID(ctx context.Context, ts time.Time) (RecordID, error) {
	var (
		anyExist bool
		highest  RecordID
	)

	milli := ts.UnixMilli()
	highest, hasEntry := ra.maxRecordIDByTS.find(milli)
	if hasEntry {
		anyExist = true
	} else {
		// Query the highest record ID for this millisecond timestamp
		var id sql.NullInt64
		err := ra.stmt.QueryRowContext(ctx, milli).Scan(&id)
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
		var ok bool
		next, ok = highest.Next()
		if !ok {
			return 0, ErrTooManyRecords
		}
	} else {
		next = MakeRecordID(ts, 0)
	}
	if hasEntry {
		ra.maxRecordIDByTS.set(milli, next)
	} else {
		ra.maxRecordIDByTS.add(milli, next)
	}
	return next, nil
}

// CheckID matches one or more health checks, depending on non-zero fields.
// A zero CheckID matches all checks for all devices.
// An absent id matches all checks for the named device.
// An absent name, but a present id, is unsupported as ids are scoped to names.
type CheckID struct {
	Name, ID string
}

type checkIDAllocator struct {
	ids        mapping[CheckID, int64] // values are row PKs from the id table
	insertStmt *sql.Stmt
	queryStmt  *sql.Stmt
}

func newCheckIDAllocator(ctx context.Context, tx *sql.Tx) (*checkIDAllocator, error) {
	insertStmt, err := tx.PrepareContext(ctx, "INSERT INTO health_check_ids (name, check_id) VALUES (?, ?) RETURNING id")
	if err != nil {
		return nil, err
	}
	queryStmt, err := tx.PrepareContext(ctx, "SELECT id FROM health_check_ids WHERE name = ? AND check_id = ?")
	if err != nil {
		return nil, err
	}
	return &checkIDAllocator{
		insertStmt: insertStmt,
		queryStmt:  queryStmt,
	}, nil
}

func (a *checkIDAllocator) getOrInsertCheckID(ctx context.Context, name, id string) (int64, error) {
	return getOrInsert(ctx, a.ids, CheckID{name, id}, a.queryStmt, a.insertStmt, name, id)
}

type auxAllocator struct {
	rows       mapping[string, int64] // from string(data) to row id in aux table
	insertStmt *sql.Stmt
	queryStmt  *sql.Stmt
}

func newAuxAllocator(ctx context.Context, tx *sql.Tx) (*auxAllocator, error) {
	insertStmt, err := tx.PrepareContext(ctx, "INSERT INTO health_check_aux(payload) VALUES (?) RETURNING id")
	if err != nil {
		return nil, err
	}
	queryStmt, err := tx.PrepareContext(ctx, "SELECT id FROM health_check_aux WHERE payload = ?")
	if err != nil {
		return nil, err
	}
	return &auxAllocator{
		insertStmt: insertStmt,
		queryStmt:  queryStmt,
	}, nil
}

func (a *auxAllocator) getOrInsertAux(ctx context.Context, payload []byte) (int64, error) {
	return getOrInsert(ctx, a.rows, string(payload), a.queryStmt, a.insertStmt, payload)
}

func getOrInsert[K comparable](ctx context.Context, cache mapping[K, int64], k K, queryStmt, insertStmt *sql.Stmt, args ...any) (int64, error) {
	if pk, ok := cache.find(k); ok {
		return pk, nil
	}

	var pk int64
	err := queryStmt.QueryRowContext(ctx, args...).Scan(&pk)
	if err == nil {
		cache.add(k, pk)
		return pk, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	err = insertStmt.QueryRowContext(ctx, args...).Scan(&pk)
	if err != nil {
		return 0, err
	}
	cache.add(k, pk)
	return pk, nil
}

var (
	ErrTooManyRecords  = errors.New("too many records")
	ErrInvalidRecordID = errors.New("invalid record ID format")
)

// mapping is copied from the http package to represent a k-v mapping, with optimisations for few entries.
type mapping[K comparable, V any] struct {
	s []entry[K, V] // for few mappings
	m map[K]V       // for many mappings
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// taken from http package impl based on benchmarks
const maxSlice = 8

// add adds a new k-v pair to the mapping.
func (h *mapping[K, V]) add(k K, v V) {
	if h.m == nil && len(h.s) < maxSlice {
		h.s = append(h.s, entry[K, V]{k, v})
	} else {
		if h.m == nil {
			h.m = map[K]V{}
			for _, e := range h.s {
				h.m[e.key] = e.value
			}
			h.s = nil
		}
		h.m[k] = v
	}
}

// set updates an existing key with the provided value.
func (h *mapping[K, V]) set(k K, v V) {
	if h.m == nil {
		for i, e := range h.s {
			if e.key == k {
				e.value = v
				h.s[i] = e
				return
			}
		}
	} else {
		h.m[k] = v
	}
}

// find returns the value corresponding to the given key.
// The second return value is false if there is no value
// with that key.
func (h *mapping[K, V]) find(k K) (v V, found bool) {
	if h == nil {
		return v, false
	}
	if h.m != nil {
		v, found = h.m[k]
		return v, found
	}
	for _, e := range h.s {
		if e.key == k {
			return e.value, true
		}
	}
	return v, false
}
