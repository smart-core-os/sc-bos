// Package cloud exposes a client for talking to a cloud service like Smart Core Connect.
//
// The cloudsim tool in this repo is a test implementation of this API, useful for local development.
//
// The term 'enrollment' in this package relates to the initial connection of a BOS node to the cloud, and is totally
// independent of the enrollment of a BOS node in a cohort, as in pkg/manage/enrollment.
//
// # Config distribution
//
// The cloud distributes configuration to a node via a pull model. On each poll BOS checks in with the
// cloud, reporting its current and any installing config deployment; the cloud replies with the latest
// config deployment it wants the node to run. A new config deployment is downloaded, staged as
// "installing", and applied on the next clean restart, after which BOS reports the deployment as
// current. This is the config channel, handled by DeploymentUpdater.
//
// # Software update distribution (Supervisor integration)
//
// Alongside config, the same check-in carries a software-update channel (handled by SoftwareUpdater).
// The cloud may return a latestUpdate naming an update deployment plus its artefact (target version,
// SHA-256, payload URL). Unlike config, BOS cannot apply a software update itself — applying it
// recreates the BOS process. Instead BOS drives a separate host process, the Supervisor, over a
// Unix-socket gRPC API:
//
//  1. On a new latestUpdate, BOS persists the in-flight intent, reports installingUpdate to the cloud,
//     and calls the Supervisor's InstallUpdate(version, payloadURL, sha256).
//  2. The Supervisor downloads, verifies, applies, and restarts BOS. The new instance calls the
//     Supervisor's Commit(version) on startup; the Supervisor auto-rolls-back if no matching Commit
//     arrives within its deadline.
//  3. After the startup Commit settles, BOS compares its running version to the in-flight target and
//     reports the outcome to the cloud: currentUpdate on success, failedUpdate (with the Supervisor's
//     reason) on a rollback, then clears the intent. See SoftwareUpdater.ReconcileStartup.
//
// The Supervisor integration is optional and disabled by default. When disabled (no Supervisor, as in
// dev), the update channel reports nothing, no install is attempted, and the config channel is
// unaffected. Enable it via the system config:
//
//	{
//	  "supervisor": {
//	    "enabled": true,
//	    "socketPath": "/run/sc-bos-supervisor/supervisor.sock"
//	  }
//	}
//
// socketPath and the per-RPC timeout have built-in defaults (see pkg/app/sysconf); only enabled is
// required to turn the integration on.
//
// The "running version" BOS commits and compares is its build-info main module version. For
// development it can be overridden by the BOS_VERSION_OVERRIDE environment variable; this override is
// for development only. In production the BOS build-info version MUST equal the artefact/image tag
// version that the Supervisor installs, otherwise a successful update is indistinguishable from a
// rollback and will be reported as failedUpdate.
package cloud
