package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// makeSiteAndNode creates a site and a node (default podman platform) and returns their IDs.
func makeSiteAndNode(t *testing.T, s *Store) (siteID, nodeID int64) {
	t.Helper()
	ctx := context.Background()
	err := s.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID
		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname:   "node-1",
			SiteID:     siteID,
			Platform:   "podman",
			SecretHash: []byte("0123456789abcdef0123456789abcdef"),
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		t.Fatalf("setup site/node: %v", err)
	}
	return siteID, nodeID
}

func readArtefactPayload(t *testing.T, s *Store, id int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	err := s.ReadUpdateArtefactPayload(context.Background(), id, func(file *os.File, size int64) error {
		_, err := io.Copy(&buf, file)
		return err
	})
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	return buf.Bytes()
}

func TestStore_UpdateArtefact_StreamRoundTrip(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()
	siteID, _ := makeSiteAndNode(t, s)

	payload := []byte("this is a fake podman-save tarball")
	wantSum := sha256.Sum256(payload)

	desc := "v1.2.3 build"
	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		SiteID:      &siteID,
		Platform:    "podman",
		Version:     "1.2.3",
		Description: &desc,
	}, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create artefact: %v", err)
	}
	id, sum := created.ID, created.Sha256.String
	if sum != hex.EncodeToString(wantSum[:]) {
		t.Errorf("sha256 = %s, want %s", sum, hex.EncodeToString(wantSum[:]))
	}

	// Payload round-trips byte-for-byte.
	if got := readArtefactPayload(t, s, id); !bytes.Equal(got, payload) {
		t.Errorf("payload round-trip mismatch: got %q want %q", got, payload)
	}

	// Metadata reflects the stored artefact, with sha256 recorded and size = len(payload).
	var meta queries.UpdateArtefact
	err = s.Read(ctx, func(tx *Tx) error {
		var e error
		meta, e = tx.GetUpdateArtefact(ctx, id)
		return e
	})
	if err != nil {
		t.Fatalf("get artefact: %v", err)
	}
	if !meta.SiteID.Valid || meta.SiteID.Int64 != siteID {
		t.Errorf("siteID = %v, want %d", meta.SiteID, siteID)
	}
	if meta.Platform != "podman" || meta.Version != "1.2.3" {
		t.Errorf("platform/version = %s/%s", meta.Platform, meta.Version)
	}
	if !meta.Sha256.Valid || meta.Sha256.String != sum {
		t.Errorf("stored sha256 = %v, want %s", meta.Sha256, sum)
	}
	if meta.Size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", meta.Size, len(payload))
	}
}

func TestStore_UpdateArtefact_LargePayloadStreams(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()

	// ~8 MiB of random data to exercise the chunked streaming path (1 MiB blocks).
	payload := make([]byte, 8<<20)
	if _, err := rand.Read(payload); err != nil {
		t.Fatalf("rand: %v", err)
	}
	wantSum := sha256.Sum256(payload)

	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "freebsd",
		Version:  "9.9.9",
	}, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create large artefact: %v", err)
	}
	id, sum := created.ID, created.Sha256.String
	if sum != hex.EncodeToString(wantSum[:]) {
		t.Errorf("large sha mismatch")
	}
	got := readArtefactPayload(t, s, id)
	if !bytes.Equal(got, payload) {
		t.Errorf("large payload round-trip mismatch (got %d bytes)", len(got))
	}
}

func TestStore_UpdateArtefact_GenericHasNullSite(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()

	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "podman",
		Version:  "generic-1",
	}, bytes.NewReader([]byte("abc")))
	if err != nil {
		t.Fatalf("create generic artefact: %v", err)
	}
	id := created.ID
	var meta queries.UpdateArtefact
	_ = s.Read(ctx, func(tx *Tx) error {
		var e error
		meta, e = tx.GetUpdateArtefact(ctx, id)
		return e
	})
	if meta.SiteID.Valid {
		t.Errorf("generic artefact should have NULL site_id, got %d", meta.SiteID.Int64)
	}
}

func TestStore_UpdateArtefact_ListFilters(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()
	siteID, _ := makeSiteAndNode(t, s)

	mk := func(site *int64, platform, version string) {
		_, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
			SiteID: site, Platform: platform, Version: version,
		}, bytes.NewReader([]byte("hi")))
		if err != nil {
			t.Fatalf("create %s: %v", version, err)
		}
	}
	mk(&siteID, "podman", "site-podman")
	mk(&siteID, "freebsd", "site-freebsd")
	mk(nil, "podman", "generic-podman")

	list := func(platform string, site int64) []queries.UpdateArtefact {
		var rows []queries.UpdateArtefact
		err := s.Read(ctx, func(tx *Tx) error {
			var e error
			rows, e = tx.ListUpdateArtefacts(ctx, queries.ListUpdateArtefactsParams{
				AfterID:  0,
				Platform: platform,
				SiteID:   site,
				Limit:    100,
			})
			return e
		})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		return rows
	}

	// No filter: all three.
	if got := list("", 0); len(got) != 3 {
		t.Errorf("no filter: got %d artefacts, want 3", len(got))
	}
	// Filter by site: the site-specific two plus the generic one.
	if got := list("", siteID); len(got) != 3 {
		t.Errorf("site filter: got %d, want 3 (2 site + 1 generic)", len(got))
	}
	// Filter by platform podman: site-podman + generic-podman.
	if got := list("podman", 0); len(got) != 2 {
		t.Errorf("platform filter: got %d, want 2", len(got))
	}
	// Site + platform freebsd: just site-freebsd (generic is podman).
	if got := list("freebsd", siteID); len(got) != 1 {
		t.Errorf("site+platform filter: got %d, want 1", len(got))
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("stat %s: %v", path, err)
	return false
}

func TestStore_UpdateArtefact_DeleteIsRowOnlySweepReclaims(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()

	// Create the kept artefact first so its id differs from the one we delete (SQLite reuses the
	// rowid of a deleted row when the table is otherwise empty).
	kept, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "podman", Version: "2.0.0",
	}, bytes.NewReader([]byte("hi")))
	if err != nil {
		t.Fatalf("create kept artefact: %v", err)
	}
	keepID := kept.ID

	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "podman", Version: "1.0.0",
	}, bytes.NewReader([]byte("abc")))
	if err != nil {
		t.Fatalf("create artefact: %v", err)
	}
	id := created.ID
	if !fileExists(t, s.artefactPath(id)) {
		t.Fatalf("payload file should exist after create")
	}

	// Deleting the row is row-only: the file is intentionally left behind for the sweep.
	err = s.Write(ctx, func(tx *Tx) error {
		_, e := tx.DeleteUpdateArtefact(ctx, id)
		return e
	})
	if err != nil {
		t.Fatalf("delete artefact: %v", err)
	}
	if !fileExists(t, s.artefactPath(id)) {
		t.Errorf("payload file should remain after row-only delete")
	}

	// Stray files the sweep should and should not touch.
	strayTmp := filepath.Join(s.artefactsDir, "tmp-leftover")
	if err := os.WriteFile(strayTmp, []byte("partial"), 0o644); err != nil {
		t.Fatalf("write stray temp: %v", err)
	}
	strayNonNumeric := filepath.Join(s.artefactsDir, "notes.txt")
	if err := os.WriteFile(strayNonNumeric, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("write stray non-numeric: %v", err)
	}

	removed, err := s.SweepOrphanArtefacts(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	// Removed: the deleted artefact's orphan file and the stray temp file.
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	if fileExists(t, s.artefactPath(id)) {
		t.Errorf("orphan file should be removed by sweep")
	}
	if fileExists(t, strayTmp) {
		t.Errorf("stray temp file should be removed by sweep")
	}
	if !fileExists(t, s.artefactPath(keepID)) {
		t.Errorf("referenced artefact file should be kept by sweep")
	}
	if !fileExists(t, strayNonNumeric) {
		t.Errorf("non-numeric file should be left alone by sweep")
	}
}

func TestStore_UpdateArtefact_ReusedIDOverwritesOrphanFile(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()

	first := []byte("first payload")
	firstArt, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "podman", Version: "1.0.0",
	}, bytes.NewReader(first))
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	firstID := firstArt.ID

	// Delete the row; the file is left behind as an orphan.
	if err := s.Write(ctx, func(tx *Tx) error {
		_, e := tx.DeleteUpdateArtefact(ctx, firstID)
		return e
	}); err != nil {
		t.Fatalf("delete first: %v", err)
	}

	// The next create reuses the rowid (the table is empty). Its rename must overwrite the orphan
	// file so the new row points at the new contents.
	second := []byte("second, different payload")
	wantSum := sha256.Sum256(second)
	secondArt, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		Platform: "freebsd", Version: "2.0.0",
	}, bytes.NewReader(second))
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	secondID, sum := secondArt.ID, secondArt.Sha256.String
	if secondID != firstID {
		t.Fatalf("expected rowid reuse: firstID=%d secondID=%d", firstID, secondID)
	}
	if sum != hex.EncodeToString(wantSum[:]) {
		t.Errorf("sha256 = %s, want %s", sum, hex.EncodeToString(wantSum[:]))
	}
	if got := readArtefactPayload(t, s, secondID); !bytes.Equal(got, second) {
		t.Errorf("payload = %q, want %q (orphan file not overwritten)", got, second)
	}
}

func TestStore_UpdateArtefact_SiteCascadeLeavesFileForSweep(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()
	siteID, _ := makeSiteAndNode(t, s)

	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		SiteID: &siteID, Platform: "podman", Version: "1.0.0",
	}, bytes.NewReader([]byte("ok")))
	if err != nil {
		t.Fatalf("create artefact: %v", err)
	}
	id := created.ID

	// Deleting the site cascades to its artefact rows, but SQLite cannot remove the file.
	err = s.Write(ctx, func(tx *Tx) error {
		_, e := tx.DeleteSite(ctx, siteID)
		return e
	})
	if err != nil {
		t.Fatalf("delete site: %v", err)
	}
	err = s.Read(ctx, func(tx *Tx) error {
		_, e := tx.GetUpdateArtefact(ctx, id)
		return e
	})
	if err == nil {
		t.Fatalf("artefact row should be cascade-deleted with its site")
	}
	if !fileExists(t, s.artefactPath(id)) {
		t.Fatalf("payload file should remain after cascade delete")
	}

	removed, err := s.SweepOrphanArtefacts(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if fileExists(t, s.artefactPath(id)) {
		t.Errorf("cascade-orphaned file should be removed by sweep")
	}
}

func TestStore_UpdateDeployments_Lifecycle(t *testing.T) {
	s := NewMemoryStore(zap.NewNop())
	defer s.Close()
	ctx := context.Background()
	siteID, nodeID := makeSiteAndNode(t, s)

	created, err := s.CreateUpdateArtefact(ctx, CreateUpdateArtefactParams{
		SiteID: &siteID, Platform: "podman", Version: "1.0.0",
	}, bytes.NewReader([]byte("ok")))
	if err != nil {
		t.Fatalf("create artefact: %v", err)
	}
	artefactID := created.ID

	var dep queries.UpdateDeployment
	err = s.Write(ctx, func(tx *Tx) error {
		var e error
		dep, e = tx.CreateUpdateDeployment(ctx, queries.CreateUpdateDeploymentParams{
			UpdateArtefactID: artefactID,
			NodeID:           nodeID,
			Status:           "pending",
		})
		return e
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	// Active deployment by node finds the pending one.
	var active queries.UpdateDeployment
	err = s.Read(ctx, func(tx *Tx) error {
		var e error
		active, e = tx.GetActiveUpdateDeploymentByNode(ctx, nodeID)
		return e
	})
	if err != nil || active.ID != dep.ID {
		t.Fatalf("active by node: id=%d err=%v", active.ID, err)
	}

	// Set status to completed sets finished_time.
	var done queries.UpdateDeployment
	err = s.Write(ctx, func(tx *Tx) error {
		var e error
		done, e = tx.SetUpdateDeploymentStatus(ctx, queries.SetUpdateDeploymentStatusParams{
			ID:     dep.ID,
			Status: "completed",
		})
		return e
	})
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
	if done.Status != "completed" || !done.FinishedTime.Valid {
		t.Errorf("completed deployment: status=%s finished=%v", done.Status, done.FinishedTime)
	}

	// No active deployment remains.
	err = s.Read(ctx, func(tx *Tx) error {
		_, e := tx.GetActiveUpdateDeploymentByNode(ctx, nodeID)
		return e
	})
	if err == nil {
		t.Error("expected no active deployment after completion")
	}
}
