package store

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// BackfillConfigVersionChecksums computes and stores a SHA-256 for every config version whose checksum
// is missing, returning the number of rows updated. Rows created before the sha256 column existed have a
// NULL checksum; the check-in endpoint now requires one, so this repairs legacy rows in place.
func (s *Store) BackfillConfigVersionChecksums(ctx context.Context) (updated int, err error) {
	err = s.Write(ctx, func(tx *Tx) error {
		rows, err := tx.ListConfigVersionsWithoutChecksum(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			sum := sha256.Sum256(row.Payload)
			if err := tx.SetConfigVersionChecksum(ctx, queries.SetConfigVersionChecksumParams{
				ID:     row.ID,
				Sha256: sum[:],
			}); err != nil {
				return fmt.Errorf("set checksum for config version %d: %w", row.ID, err)
			}
			updated++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return updated, nil
}
