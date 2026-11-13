package sqlite

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A write that meets a lock held by another connection must wait for it rather than fail with
// "database is locked" (SCB-1534).
func TestOpen_WaitsForLock(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "db.sqlite3")
	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	err = db.WriteTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `CREATE TABLE test ( id INTEGER PRIMARY KEY );`)
		return err
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	other, err := sql.Open("sqlite3", buildFileURI(dbPath, true, nil))
	if err != nil {
		t.Fatalf("open blocking connection: %v", err)
	}
	defer other.Close()
	blocker, err := other.Conn(ctx)
	if err != nil {
		t.Fatalf("blocking Conn: %v", err)
	}
	defer blocker.Close()
	if _, err := blocker.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("BEGIN IMMEDIATE: %v", err)
	}

	const holdFor = 200 * time.Millisecond
	start := time.Now()
	writeErr := make(chan error, 1)
	go func() {
		writeErr <- db.WriteTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO test (id) VALUES (1);`)
			return err
		})
	}()

	select {
	case err := <-writeErr:
		t.Fatalf("WriteTx returned while the lock was held: %v", err)
	case <-time.After(holdFor):
	}
	if _, err := blocker.ExecContext(ctx, "COMMIT"); err != nil {
		t.Fatalf("COMMIT: %v", err)
	}

	select {
	case err := <-writeErr:
		if err != nil {
			t.Fatalf("WriteTx: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WriteTx did not finish after the lock was released")
	}
	if waited := time.Since(start); waited < holdFor {
		t.Errorf("WriteTx finished after %v, before the lock was released", waited)
	}
}

func TestBuildURI_BusyTimeoutFirst(t *testing.T) {
	uris := map[string]string{
		"file_reader":   buildFileURI("db.sqlite3", false, nil),
		"file_writer":   buildFileURI("db.sqlite3", true, []pragma{{Key: "busy_timeout", Value: "5"}}),
		"memory_reader": buildMemoryURI("mem", false, nil),
		"memory_writer": buildMemoryURI("mem", true, []pragma{{Key: "busy_timeout", Value: "5"}}),
	}
	for name, uri := range uris {
		t.Run(name, func(t *testing.T) {
			_, rawQuery, _ := strings.Cut(uri, "?")
			query, err := url.ParseQuery(rawQuery)
			if err != nil {
				t.Fatalf("parse %q: %v", uri, err)
			}
			pragmas := query["_pragma"]
			if len(pragmas) == 0 || pragmas[0] != busyTimeoutPragma {
				t.Errorf("first _pragma = %v, want %s", pragmas, busyTimeoutPragma)
			}
			if strings.HasSuffix(name, "_writer") && pragmas[len(pragmas)-1] != "busy_timeout(5)" {
				t.Errorf("caller busy_timeout should come last so it overrides the default, got %v", pragmas)
			}
		})
	}
}
