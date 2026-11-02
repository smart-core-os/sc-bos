// Package install applies BOS update artefacts on a single node. An installer tracks two installed
// releases: current, the release BOS runs, and previous, the release to roll back to. Applying a release
// points previous at the old current release and current at the new one. Rolling back points current at
// the previous release. Confirmation that the new release is good is obtained out-of-band: BOS asserts
// its running version via the Supervisor's Commit RPC (see supervisor/docs/state-model.md), so the
// Installer itself only applies, restarts and rolls back.
package install

import "context"

// Installer applies a downloaded update artefact, restarts BOS, or rolls back to the previous release,
// for a single BOS node.
type Installer interface {
	// Apply installs the artefact at artefactPath as version, then points previous at the old current
	// release and current at version, without restarting BOS. It is idempotent: when current already
	// points at version, previous is left unchanged, so recovery can re-run it safely.
	Apply(ctx context.Context, artefactPath, version string) error
	// Restart restarts BOS so it runs the release current points at.
	Restart(ctx context.Context) error
	// Rollback points current at the previous release and restarts BOS. It returns an error if there is
	// no previous release to roll back to (e.g. a failed first install).
	Rollback(ctx context.Context) error
}
