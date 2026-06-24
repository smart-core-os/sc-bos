package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// CreateBinaryArtefactParams carries the metadata for a new binary artefact. The payload itself is
// streamed separately so it is never held in memory in full.
type CreateBinaryArtefactParams struct {
	SiteID      *int64 // nil = generic artefact, available to all sites
	OS          string
	Arch        string
	Version     string
	Description *string
}

// BinaryArtefact is an artefact's stored metadata together with its payload Size. Size is the length of
// the external payload file, read with stat, rather than a stored column.
type BinaryArtefact struct {
	queries.BinaryArtefact
	Size int64
}

// artefactPath returns the on-disk path of an artefact's payload file, named by its id.
func (s *Store) artefactPath(id int64) string {
	return filepath.Join(s.artefactsDir, strconv.FormatInt(id, 10))
}

// CreateBinaryArtefact inserts a new binary artefact, copying src completely into an external
// payload file while computing its SHA-256 and counting the bytes read. The payload is never buffered
// in full. It returns the created artefact (the generated id, the computed sha256, the create time, and
// the payload Size) so the caller need not read it back.
//
// The payload is first streamed to a temp file, then the row is inserted and the file renamed into
// place (named by the new id) before the transaction commits. Two edge cases follow from this:
//
//   - Reused id: SQLite reuses the rowid of a deleted artefact whose file may still be on disk (file
//     deletion is row-only; see SweepOrphanArtefacts). The rename overwrites that stale file
//     atomically, so the new row always points at the new contents - no special handling needed.
//   - Commit fails after the rename: the row is rolled back but the file is left in place named by an
//     id that no longer exists. This is just another orphan file: it is invisible to reads (the row
//     is gone, so downloads 404) and SweepOrphanArtefacts reclaims it. Per the storage design there
//     is no inline file deletion, so we deliberately do not remove it here.
func (s *Store) CreateBinaryArtefact(ctx context.Context, p CreateBinaryArtefactParams, src io.Reader) (BinaryArtefact, error) {
	tmp, err := os.CreateTemp(s.artefactsDir, "tmp-*")
	if err != nil {
		return BinaryArtefact{}, err
	}
	tmpName := tmp.Name()
	// Remove the temp file unless it is successfully renamed into place below.
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpName)
		}
	}()

	h := sha256.New()
	size, copyErr := io.Copy(tmp, io.TeeReader(src, h))
	if copyErr != nil {
		_ = tmp.Close()
		return BinaryArtefact{}, fmt.Errorf("stream payload: %w", copyErr)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return BinaryArtefact{}, err
	}
	if err := tmp.Close(); err != nil {
		return BinaryArtefact{}, err
	}

	qp := queries.CreateBinaryArtefactParams{
		Os:      p.OS,
		Arch:    p.Arch,
		Version: p.Version,
		Sha256:  h.Sum(nil),
	}
	if p.SiteID != nil {
		qp.SiteID = sql.NullInt64{Int64: *p.SiteID, Valid: true}
	}
	if p.Description != nil {
		qp.Description = sql.NullString{String: *p.Description, Valid: true}
	}

	var row queries.BinaryArtefact
	err = s.Write(ctx, func(tx *Tx) error {
		var err error
		row, err = tx.CreateBinaryArtefact(ctx, qp)
		if err != nil {
			return err
		}
		if err := os.Rename(tmpName, s.artefactPath(row.ID)); err != nil {
			return err
		}
		renamed = true
		return nil
	})
	if err != nil {
		return BinaryArtefact{}, err
	}
	return BinaryArtefact{BinaryArtefact: row, Size: size}, nil
}

// GetBinaryArtefact returns an artefact's metadata, including its payload Size read from the external
// file. A missing artefact row yields an error wrapping sql.ErrNoRows.
func (s *Store) GetBinaryArtefact(ctx context.Context, id int64) (BinaryArtefact, error) {
	var row queries.BinaryArtefact
	if err := s.Read(ctx, func(tx *Tx) error {
		var err error
		row, err = tx.GetBinaryArtefact(ctx, id)
		return err
	}); err != nil {
		return BinaryArtefact{}, err
	}
	size, err := s.artefactSize(id)
	if err != nil {
		return BinaryArtefact{}, err
	}
	return BinaryArtefact{BinaryArtefact: row, Size: size}, nil
}

// ListBinaryArtefacts returns a page of artefact metadata, each including its payload Size read from the
// external file.
func (s *Store) ListBinaryArtefacts(ctx context.Context, params queries.ListBinaryArtefactsParams) ([]BinaryArtefact, error) {
	var rows []queries.BinaryArtefact
	if err := s.Read(ctx, func(tx *Tx) error {
		var err error
		rows, err = tx.ListBinaryArtefacts(ctx, params)
		return err
	}); err != nil {
		return nil, err
	}
	out := make([]BinaryArtefact, len(rows))
	for i, row := range rows {
		size, err := s.artefactSize(row.ID)
		if errors.Is(err, fs.ErrNotExist) {
			// The payload file is gone but the metadata row remains; list it with size 0 rather than
			// failing the whole page, so the orphaned row stays visible (and deletable).
			size = 0
		} else if err != nil {
			return nil, err
		}
		out[i] = BinaryArtefact{BinaryArtefact: row, Size: size}
	}
	return out, nil
}

// ReadBinaryArtefactPayload opens the payload file of the given artefact read-only and passes it,
// with its size, to f. f typically sets Content-Length from size and streams the file out. The file
// is closed when f returns. If the payload file does not exist, os.Open returns an error that wraps
// os.ErrNotExist.
func (s *Store) ReadBinaryArtefactPayload(ctx context.Context, id int64, f func(file *os.File, size int64) error) error {
	file, err := os.Open(s.artefactPath(id))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return f(file, info.Size())
}

// artefactSize returns the byte length of an artefact's external payload file.
func (s *Store) artefactSize(id int64) (int64, error) {
	info, err := os.Stat(s.artefactPath(id))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// SweepOrphanArtefacts removes payload files that no longer correspond to a binary_artefact row,
// plus any leftover temp files from interrupted uploads, and returns the number of files removed.
// Orphans arise from artefact deletions (which are row-only) and from silent cascade deletes (removing
// a site removes its artefact rows), as well as from crashes mid-upload. SQLite cannot delete files
// on cascade, so this sweep is the sole reclaimer of artefact payload files.
func (s *Store) SweepOrphanArtefacts(ctx context.Context) (removed int, err error) {
	var ids []int64
	if err := s.Read(ctx, func(tx *Tx) error {
		var e error
		ids, e = tx.ListBinaryArtefactIDs(ctx)
		return e
	}); err != nil {
		return 0, err
	}
	live := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		live[strconv.FormatInt(id, 10)] = struct{}{}
	}

	entries, err := os.ReadDir(s.artefactsDir)
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		switch {
		case strings.HasPrefix(name, "tmp-"):
			// Leftover temp file from an interrupted upload.
		case isLiveArtefactFile(name, live):
			continue
		case !isArtefactFileName(name):
			// Not one of ours (numeric id); leave it alone.
			continue
		}
		if err := os.Remove(filepath.Join(s.artefactsDir, name)); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func isLiveArtefactFile(name string, live map[string]struct{}) bool {
	_, ok := live[name]
	return ok
}

func isArtefactFileName(name string) bool {
	_, err := strconv.ParseInt(name, 10, 64)
	return err == nil
}
