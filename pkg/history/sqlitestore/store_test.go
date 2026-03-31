package sqlitestore

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/smart-core-os/sc-bos/pkg/history"
)

func TestDatabase_Insert(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	// Using fixed origin time for test determinism
	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	source := "test-source"

	// Insert multiple records sequentially
	records := []Record{
		{Source: source, Payload: []byte("test-payload-1"), CreateTime: originTime.Add(-2 * time.Hour)},
		{Source: source, Payload: []byte("test-payload-2"), CreateTime: originTime.Add(-1 * time.Hour)},
		{Source: source, Payload: []byte("test-payload-3"), CreateTime: originTime},
	}

	// Use zero-sized slice with capacity
	insertedRecords := make([]Record, 0, len(records))

	// Insert all records and collect the results
	for _, r := range records {
		record, err := db.Insert(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error inserting record: %v", err)
		}

		if record.ID == 0 {
			t.Errorf("expected valid record ID, got record.ID=%q", record.ID)
		}

		if record.Payload == nil || string(record.Payload) != string(r.Payload) {
			t.Errorf("expected payload to match %q, got %q", r.Payload, record.Payload)
		}

		insertedRecords = append(insertedRecords, record)
	}

	// Verify all records were stored correctly
	verifyRecords(t, db, ctx, records)
}

// verifyRecords retrieves all records for a source from the database and verifies they match expected records.
func verifyRecords(t testing.TB, db *Database, ctx context.Context, expected []Record) {
	t.Helper()

	buf := make([]Record, 1000)

	var next RecordID
	for len(expected) > 0 {
		clear(buf)
		// Fetch all records using zero-value Records to indicate full range
		count, err := db.Read(ctx, "", next, 0, false, buf)
		if err != nil {
			t.Fatalf("unexpected error reading records: %v", err)
		}

		// either we should fill the buffer or get all remaining records
		if count != len(expected) && count != len(buf) {
			t.Errorf("expected %d records, got %d", len(expected), count)
			return
		}

		// Verify each record has correct data
		for i := range count {
			if buf[i].ID == 0 {
				t.Errorf("record %d: missing ID", i)
			}
		}
		diff := cmp.Diff(expected[:count], buf[:count],
			cmpopts.IgnoreFields(Record{}, "ID"),
			cmpopts.EquateApproxTime(time.Millisecond),
		)
		if diff != "" {
			t.Errorf("data mismatch (-want +got):\n%s", diff)
		}
		expected = expected[count:]
		if count > 0 {
			next = buf[count-1].ID + 1
		}
	}

}

func TestDatabase_InsertBulk(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: "source-2", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
	}

	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expect := []Record{
		{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: "source-2", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_InsertBulk_DuplicateSources(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	source := "duplicate-source"
	records := []Record{
		{Source: source, CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: source, CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
	}

	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expect := []Record{
		{Source: source, CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: source, CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_InsertBulk_DuplicateCreateTimes(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: "source-2", CreateTime: originTime, Payload: []byte("payload-2")},
	}

	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// for records with identical CreateTime, insertion order should be preserved
	expect := []Record{
		{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: "source-2", CreateTime: originTime, Payload: []byte("payload-2")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_TrimCount(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: "source-1", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("-3h")},
		{Source: "source-2", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("-3h")},
		{Source: "source-1", CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("-2h")},
		{Source: "source-2", CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("-2h")},
		{Source: "source-1", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
		{Source: "source-2", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
	}
	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Trim to 2 records for source-1, should delete the oldest record
	// source-2 should remain unaffected
	deleted, err := db.TrimCount(ctx, "source-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 record deleted, got %d", deleted)
	}
	expect := []Record{
		{Source: "source-2", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("-3h")},
		{Source: "source-1", CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("-2h")},
		{Source: "source-2", CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("-2h")},
		{Source: "source-1", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
		{Source: "source-2", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_TrimTime(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: "other-source", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("other-3h")},
		{Source: "time-source", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("-3h")},
		{Source: "time-source", CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("-2h")},
		{Source: "time-source", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
		{Source: "time-source", CreateTime: originTime, Payload: []byte("now")},
	}
	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Trim records older than -90 minutes
	cutoff := originTime.Add(-90 * time.Minute)
	deleted, err := db.TrimTime(ctx, "time-source", cutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 records deleted, got %d", deleted)
	}
	expect := []Record{
		{Source: "other-source", CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("other-3h")},
		{Source: "time-source", CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("-1h")},
		{Source: "time-source", CreateTime: originTime, Payload: []byte("now")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_InsertBulk_WithMaxCount(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	source := "trim-source"
	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: source, CreateTime: originTime.Add(-3 * time.Hour), Payload: []byte("old-payload-1")},
		{Source: source, CreateTime: originTime.Add(-2 * time.Hour), Payload: []byte("old-payload-2")},
		{Source: source, CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("old-payload-3")},
		{Source: source, CreateTime: originTime, Payload: []byte("new-payload")},
	}

	err := db.InsertBulk(ctx, records, WithMaxCount(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expect := []Record{
		{Source: source, CreateTime: originTime.Add(-1 * time.Hour), Payload: []byte("old-payload-3")},
		{Source: source, CreateTime: originTime, Payload: []byte("new-payload")},
	}
	verifyRecords(t, db, ctx, expect)
}

func TestDatabase_InsertBulk_WithEarliestTime(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	now := time.Now()
	source := "time-trim-source"
	records := []Record{
		{Source: source, CreateTime: now.Add(-3 * time.Hour), Payload: []byte("old-payload-1")},
		{Source: source, CreateTime: now.Add(-2 * time.Hour), Payload: []byte("old-payload-2")},
		{Source: source, CreateTime: now.Add(-1 * time.Hour), Payload: []byte("old-payload-3")},
		{Source: source, CreateTime: now, Payload: []byte("new-payload")},
	}

	trimBefore := now.Add(-90 * time.Minute) // Trim records older than 90 minutes

	err := db.InsertBulk(ctx, records, WithEarliestTime(trimBefore))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, err := db.Count(ctx, source, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error counting records: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 records after time-based trim, got %d", count)
	}
}

func TestDatabase_Read(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
		{Source: "source-2", CreateTime: originTime.Add(2 * time.Second), Payload: []byte("payload-3")},
		{Source: "source-2", CreateTime: originTime.Add(3 * time.Second), Payload: []byte("payload-4")},
	}
	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	type testCase struct {
		source string
		from   RecordID
		to     RecordID
		desc   bool
		expect []Record
	}
	tests := map[string]testCase{
		"all": {
			expect: []Record{
				{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
				{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
				{Source: "source-2", CreateTime: originTime.Add(2 * time.Second), Payload: []byte("payload-3")},
				{Source: "source-2", CreateTime: originTime.Add(3 * time.Second), Payload: []byte("payload-4")},
			},
		},
		"desc": {
			desc: true,
			expect: []Record{
				{Source: "source-2", CreateTime: originTime.Add(3 * time.Second), Payload: []byte("payload-4")},
				{Source: "source-2", CreateTime: originTime.Add(2 * time.Second), Payload: []byte("payload-3")},
				{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
				{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
			},
		},
		"source-1": {
			source: "source-1",
			expect: []Record{
				{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
				{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
			},
		},
		"from": {
			from: MakeRecordID(originTime.Add(time.Second), 0),
			expect: []Record{
				{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
				{Source: "source-2", CreateTime: originTime.Add(2 * time.Second), Payload: []byte("payload-3")},
				{Source: "source-2", CreateTime: originTime.Add(3 * time.Second), Payload: []byte("payload-4")},
			},
		},
		"to": {
			to: MakeRecordID(originTime.Add(2*time.Second), 0),
			expect: []Record{
				{Source: "source-1", CreateTime: originTime, Payload: []byte("payload-1")},
				{Source: "source-1", CreateTime: originTime.Add(time.Second), Payload: []byte("payload-2")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			buf := make([]Record, len(tc.expect)+1)
			n, err := db.Read(ctx, tc.source, tc.from, tc.to, tc.desc, buf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != len(tc.expect) {
				t.Errorf("expected %d records, got %d", len(tc.expect), n)
			}
			diff := cmp.Diff(tc.expect, buf[:n],
				cmpopts.IgnoreFields(Record{}, "ID"),
				cmpopts.EquateApproxTime(time.Millisecond),
			)
			if diff != "" {
				t.Errorf("data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDatabase_Count(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()

	source := "test-source"
	payload := []byte("test-payload")
	at := time.Now()

	_, err := db.Insert(ctx, Record{
		Source:     source,
		CreateTime: at,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Using zero-value Records to count the entire dataset
	count, err := db.Count(ctx, source, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestDatabase_Size(t *testing.T) {
	db := newTestDB(t)
	ctx := t.Context()

	size, err := db.Size(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if size <= 0 {
		t.Errorf("expected database size to be greater than 0, got %d", size)
	}
}

func BenchmarkDatabase_InsertBatch(b *testing.B) {
	db := newTestDB(b)
	ctx := b.Context()
	totalRecords := b.N

	const (
		numSources  = 50
		batchSize   = 10_000 // insert in batches for efficiency
		payloadSize = 100    // size of each payload in bytes
	)

	logger := zaptest.NewLogger(b)

	logger.Info("Starting large scale database test",
		zap.Int("total_records", totalRecords),
		zap.Int("num_sources", numSources),
		zap.Int("batch_size", batchSize))

	b.ResetTimer()
	_ = insertTestBatches(b, db, totalRecords)
	b.StopTimer()

	insertDuration := b.Elapsed()
	logger.Info("Insertion completed",
		zap.Duration("total_time", insertDuration),
		zap.Float64("avg_records_per_second", float64(totalRecords)/insertDuration.Seconds()))

	// Verify record count
	count, err := db.Count(ctx, "", 0, 0)
	if err != nil {
		b.Fatalf("failed to count records: %v", err)
	}

	if count != totalRecords {
		b.Errorf("expected %d records, got %d", totalRecords, count)
	}

	// Measure database size
	dbSize, err := db.Size(ctx)
	if err != nil {
		b.Fatalf("failed to get database size: %v", err)
	}

	avgSizePerRecord := float64(dbSize) / float64(totalRecords)
	overheadPerRecord := avgSizePerRecord - float64(payloadSize)

	logger.Info("Database size analysis",
		zap.Int64("total_size_bytes", dbSize),
		zap.Float64("total_size_mb", float64(dbSize)/(1024*1024)),
		zap.Float64("avg_bytes_per_record", avgSizePerRecord),
		zap.Float64("overhead_bytes_per_record", overheadPerRecord),
		zap.Int("total_records", count))
}

func BenchmarkDatabase_Read(b *testing.B) {
	db := newTestDB(b)
	ctx := b.Context()
	totalRecords := b.N

	expected := insertTestBatches(b, db, totalRecords)

	b.ResetTimer()
	verifyRecords(b, db, ctx, expected)
}

func insertTestBatches(tb testing.TB, db *Database, totalRecords int) (expected []Record) {
	const (
		numSources  = 50
		batchSize   = 10_000 // insert in batches for efficiency
		payloadSize = 100    // size of each payload in bytes
	)

	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Insert records in batches
	for batch := 0; len(expected) < totalRecords; batch++ {
		// Handle last batch which may be smaller
		thisBatchSize := batchSize
		if len(expected)+thisBatchSize > totalRecords {
			thisBatchSize = totalRecords - len(expected)
		}
		records := make([]Record, thisBatchSize)

		for i := 0; i < thisBatchSize; i++ {
			recordNum := batch*batchSize + i
			sourceID := recordNum % numSources
			record := Record{
				Source:     fmt.Sprintf("source-%d", sourceID),
				CreateTime: baseTime.Add(time.Duration(recordNum) * time.Millisecond),
				Payload:    generateRandomPayload(payloadSize),
			}
			records[i] = record
			expected = append(expected, record)
		}

		err := db.InsertBulk(tb.Context(), records)
		if err != nil {
			tb.Fatalf("failed to insert batch %d: %v", batch, err)
		}
	}
	return expected
}

func generateRandomPayload(size int) []byte {
	payload := make([]byte, size)
	_, err := rand.Read(payload)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random payload: %v", err))
	}
	return payload
}

func newTestDB(tb testing.TB) *Database {
	tb.Helper()
	dir := tb.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ctx := tb.Context()
	logger, err := zap.NewDevelopment()
	if err != nil {
		tb.Fatalf("failed to create logger: %v", err)
	}
	db, err := Open(ctx, dbPath, WithLogger(logger))
	if err != nil {
		tb.Fatalf("failed to open test database: %v", err)
	}
	tb.Logf("created test database %s", dbPath)
	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Errorf("failed to close test database: %v", err)
		}
		stat, err := os.Stat(dbPath)
		if err != nil {
			tb.Logf("failed to stat test database file: %v", err)
		} else {
			tb.Logf("database file size: %d bytes", stat.Size())
		}
	})
	return db
}

func newTestMemDB(tb testing.TB) *Database {
	tb.Helper()
	ctx := tb.Context()
	logger, err := zap.NewDevelopment()
	if err != nil {
		tb.Fatalf("failed to create logger: %v", err)
	}
	db, err := OpenMemory(ctx, WithLogger(logger))
	if err != nil {
		tb.Fatalf("failed to open test database: %v", err)
	}
	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Errorf("failed to close test database: %v", err)
		}
	})
	return db
}

// tests Read, Slice and Len functionality of a Store
func TestStore_Read(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()
	source := "test-source"
	originTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Source: source, CreateTime: originTime, Payload: []byte("payload-1")},
		{Source: source, CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-2")},
		{Source: source, CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-3")},
	}
	err := db.InsertBulk(ctx, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	type testCase struct {
		slice  bool
		from   history.Record
		to     history.Record
		desc   bool
		expect []history.Record
	}

	cases := map[string]testCase{
		"all": {
			expect: []history.Record{
				{ID: records[0].ID.String(), CreateTime: originTime, Payload: []byte("payload-1")},
				{ID: records[1].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-2")},
				{ID: records[2].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-3")},
			},
		},
		"desc": {
			desc: true,
			expect: []history.Record{
				{ID: records[2].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-3")},
				{ID: records[1].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-2")},
				{ID: records[0].ID.String(), CreateTime: originTime, Payload: []byte("payload-1")},
			},
		},
		"from_id": {
			slice: true,
			from:  history.Record{ID: records[2].ID.String()},
			expect: []history.Record{
				{ID: records[2].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-3")},
			},
		},
		"from_time": {
			slice: true,
			from:  history.Record{CreateTime: originTime.Add(time.Millisecond)},
			expect: []history.Record{
				{ID: records[1].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-2")},
				{ID: records[2].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-3")},
			},
		},
		"to_id": {
			slice: true,
			to:    history.Record{ID: records[2].ID.String()},
			expect: []history.Record{
				{ID: records[0].ID.String(), CreateTime: originTime, Payload: []byte("payload-1")},
				{ID: records[1].ID.String(), CreateTime: originTime.Add(time.Millisecond), Payload: []byte("payload-2")},
			},
		},
		"to_time": {
			slice: true,
			to:    history.Record{CreateTime: originTime.Add(time.Millisecond)},
			expect: []history.Record{
				{ID: records[0].ID.String(), CreateTime: originTime, Payload: []byte("payload-1")},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			store := db.OpenStore(source)
			var slice history.Slice = store
			if tc.slice {
				slice = slice.Slice(tc.from, tc.to)
			}
			buf := make([]history.Record, len(tc.expect)+1)
			var n int
			var err error
			if tc.desc {
				n, err = slice.ReadDesc(ctx, buf)
			} else {
				n, err = slice.Read(ctx, buf)
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != len(tc.expect) {
				t.Errorf("expected %d records, got %d", len(tc.expect), n)
			}
			diff := cmp.Diff(tc.expect, buf[:n])
			if diff != "" {
				t.Errorf("data mismatch (-want +got):\n%s", diff)
			}

			// also grab Len and verify it matches
			length, err := slice.Len(ctx)
			if err != nil {
				t.Errorf("unexpected Len error: %v", err)
			}
			if length != len(tc.expect) {
				t.Errorf("expected Len %d, got %d", len(tc.expect), length)
			}
		})
	}

}

func TestStore_Append(t *testing.T) {
	db := newTestMemDB(t)
	ctx := t.Context()
	source := "test-source"

	store := db.OpenStore(source)
	record, err := store.Append(ctx, []byte("test-payload"))
	if err != nil {
		t.Fatalf("unexpected Append error: %v", err)
	}
	diff := cmp.Diff(history.Record{
		CreateTime: time.Now(),
		Payload:    []byte("test-payload"),
	}, record,
		cmpopts.IgnoreFields(history.Record{}, "ID"),
		// we can't control exactly what time is used, but it shouldn't vary much from now
		cmpopts.EquateApproxTime(time.Second),
	)
	if diff != "" {
		t.Errorf("data mismatch (-want +got):\n%s", diff)
	}

	verifyRecords(t, db, ctx, []Record{
		{Source: source, CreateTime: record.CreateTime, Payload: []byte("test-payload")},
	})
}
