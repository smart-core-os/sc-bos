package store

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

func TestStore_BackfillConfigVersionChecksums(t *testing.T) {
	ctx := t.Context()
	store := NewMemoryStore(zap.NewNop())
	defer func() { _ = store.Close() }()

	var nodeID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Os: "linux", Arch: "arm64", Hostname: "NODE-01", SiteID: site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// A legacy config version with no checksum (NULL sha256), plus one that already has its checksum.
	legacyPayload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	keepPayload := []byte("already-checksummed")
	keepSum := sha256.Sum256(keepPayload)
	var legacyID, keepID int64
	err = store.Write(ctx, func(tx *Tx) error {
		legacy, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID: nodeID, Payload: legacyPayload, // Sha256 left nil -> NULL
		})
		if err != nil {
			return err
		}
		legacyID = legacy.ID
		keep, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID: nodeID, Payload: keepPayload, Sha256: keepSum[:],
		})
		if err != nil {
			return err
		}
		keepID = keep.ID
		return nil
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	updated, err := store.BackfillConfigVersionChecksums(ctx)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}

	err = store.Read(ctx, func(tx *Tx) error {
		legacy, err := tx.GetConfigVersion(ctx, legacyID)
		if err != nil {
			return err
		}
		wantLegacy := sha256.Sum256(legacyPayload)
		if !bytes.Equal(legacy.Sha256, wantLegacy[:]) {
			t.Errorf("legacy sha256 = %x, want %x", legacy.Sha256, wantLegacy)
		}
		keep, err := tx.GetConfigVersion(ctx, keepID)
		if err != nil {
			return err
		}
		if !bytes.Equal(keep.Sha256, keepSum[:]) {
			t.Errorf("already-checksummed row changed to %x, want %x", keep.Sha256, keepSum)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	// Idempotent: a second run has nothing left to backfill.
	if updated, err := store.BackfillConfigVersionChecksums(ctx); err != nil || updated != 0 {
		t.Errorf("second backfill = %d, %v; want 0, nil", updated, err)
	}
}
