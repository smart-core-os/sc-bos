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
// current.
//
// # Software update distribution (Supervisor integration)
//
// Alongside config, the same check-in carries software-update metadata (implemented by SoftwareUpdater).
// The cloud may return a latestUpdate object naming an update deployment that the cloud wants the node to be
// running. Unlike config, BOS cannot apply a software update itself - it lacks the necessary privileges and wouldn't
// be able to roll back in the event of a failure. When there's a software update to install, BOS will:
//
//  1. Check-in again to mark the new version as INSTALLING
//  2. Ask the supervisor to install the new update
//  3. Upon restarting, BOS will check in with the new version
//
// The "running version" BOS commits and compares is its build-info main module version. For
// development it can be overridden by the BOS_VERSION_OVERRIDE environment variable; this override is
// for development only. In production the BOS build-info version MUST equal the artefact/image tag
// version that the Supervisor installs, otherwise a successful update is indistinguishable from a
// rollback and will be reported as failedUpdate.
package cloud
