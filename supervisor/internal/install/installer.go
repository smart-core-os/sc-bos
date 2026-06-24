// Package install applies BOS update artefacts on a Rocky Linux / Podman host.
//
// The mechanism is described in cmd/supervisor/docs/supervisor-mechanism-rocky.md: load the artefact
// image, move the stable :current tag onto it (recording :previous as a rollback pointer), and restart
// only the BOS systemd unit. Confirmation that the new version is good is obtained out-of-band: BOS
// asserts its running version via the Supervisor's Commit RPC (see supervisor/docs/commit-protocol.md),
// so the Installer itself only applies and rolls back.
package install

import "context"

// Installer applies a downloaded update artefact, or rolls back to the previous one, for a single BOS
// node. Confirmation is not the Installer's concern: BOS asserts its running version via Commit, so the
// platform surface is just apply and rollback.
type Installer interface {
	// Apply loads the artefact at artefactPath, records the rollback pointer, repoints the stable tag
	// onto version, and restarts the BOS unit so it is recreated from the new image. It is idempotent:
	// re-applying an already-applied version is a no-op, so recovery can re-run it safely.
	Apply(ctx context.Context, artefactPath, version string) error
	// Rollback repoints the stable tag back to the previous image and restarts the BOS unit. It returns
	// an error if there is no previous image to roll back to (e.g. a failed first install).
	Rollback(ctx context.Context) error
}
