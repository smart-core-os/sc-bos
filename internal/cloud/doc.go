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
// Alongside config, the same check-in carries binary-update metadata (implemented by BinaryUpdater).
// The cloud may return a latestBinary object naming a deployment that the cloud wants the node to be
// running. Unlike config, BOS cannot apply a software update itself - it lacks the necessary privileges and wouldn't
// be able to roll back in the event of a failure. When there's a software update to install, BOS will:
//
//  1. Check-in again to mark the new version as INSTALLING
//  2. Ask the supervisor to install the new update
//  3. Upon restarting, BOS will check in with the new version
//
// The "running version" BOS commits and compares is embedded at build time, and can be overridden by the
// BOS_VERSION_OVERRIDE environment variable for development and testing. In production this version MUST equal the
// artefact/image tag version, otherwise updates may not install correctly.
package cloud
