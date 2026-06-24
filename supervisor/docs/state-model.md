Supervisor operation
====================

In this document: How the Supervisor applies a BOS update and correctly tracks state. See `supervisor/internal/server/server.go`
`supervisor/internal/install/podman.go`, for implementation details.

The Supervisor must accurately track the state of an update, even if it is interrupted at any point in the process.

Communication between BOS and Supervisor
----------------------------------------
The Supervisor exposes a local gRPC server, which BOS connects to. On boot, BOS will determine if it is healthy.
If it is, it will call the Commit RPC. If a new BOS update has just been installed, the Supervisor will mark the update
as completed. BOS may also call Commit on every boot, which is idempotent if the version was already marked as completed.

If BOS is unhealthy, it will not call Commit, and the Supervisor will time out waiting, and trigger the rollback
to the previous version automatically.

The same gRPC API can be used to trigger an update and check its status. All the logic of deciding when to install an
update, and which version, lives in BOS (we expect it to come from the connection to Smart Core Cloud, which is 
implemented in BOS).

State reported by the Supervisor to BOS
---------------------------------------

BOS reads the update status via `GetUpdateStatus`. The status is derived from the stored goal plus the
running phase, not kept as a separate progress field:

| State         | Meaning                                                                            |
|---------------|------------------------------------------------------------------------------------|
| `IDLE`        | no update in progress or recorded                                                  |
| `DOWNLOADING` | fetching and verifying the artefact; the host has not been changed                 |
| `INSTALLING`  | the host is being or has been changed; we don't know if the new version is healthy |
| `COMPLETED`   | the target version is running and confirmed good                                   |
| `FAILED`      | the target is not running the version we want                                      |

Host state managed by the Supervisor
------------------------------------

On Rocky / Podman the host holds one image per applied version, plus two moving tags the Supervisor
owns: `:current` (what the BOS unit boots) and `:previous` (the rollback pointer). The BOS service is
recreated from `:current` on restart. 

### FreeBSD support (planned, not yet implemented)
As BOS does not run in a container on FreeBSD, the supervisor must track an applied version, a recorded previous
version and the version actually running itself. The overall installation flow will however be similar.

Update Completion
-----------------

An update is completed when the Supervisor knows that the target version is the one actually
running and that it is healthy. This happens when the newly-updated BOS calls `Commit(version)` over the socket once
it considers itself healthy. Note that even if BOS starts up, it may consider itself unhealthy if a configured 
cloud connection fails.

Applying a Podman container update
----------------------------------

1. **Download and verify**: binary artefact is downloaded to disk and its checksum verified. 
2. **Load** the artefact image into Podman
3. **Record the rollback pointer** — point `:previous` at whatever `:current` resolves to now. A no-op
   on a first install.
4. **Select the new version** — point `:current` at the new image. This is the moment the version the
   unit will boot changes.
5. **Restart BOS**, which is recreated from `:current`.
6. **Await confirmation** — the new BOS calls `Commit` once healthy, within `commitDeadline`.

N.B.: the tag swap (steps 3–4) is skipped when the target version is the same as `:current`. Rollback
after a failed confirmation is the symmetric tail: point `:current` back at `:previous`, restart, and
await confirmation of the restored version.


Crashes and recovery
--------------------

A crash can interrupt an update at any step above. When the Supervisor restarts, it reloads its current goal from disk
and continues working to install and verify / rollback an update.

| When the crash happens                          | State of the host                                     | Recovery                                                |
|-------------------------------------------------|-------------------------------------------------------|---------------------------------------------------------|
| During download, before the image is loaded     | unchanged; still running the old version              | re-fetch, continue                                      |
| After loading, before the tag swap              | new image present, `:current` still old; running old  | update current, continue                                |
| After selecting the new version, before restart | `:current` points at the new image; still running old | update current (no-op), continue                        |
| After restart, while awaiting confirmation      | running the new version, not yet confirmed            | await commit                                            |
| Midway through a rollback                       | running the unhealthy target                          | unhealthy target fails to commit, rollback re-triggered |
| After the update has settled                    | already in desired state                              | no action required                                      |

Scenarios
---------

Precondition: running v1, goal is to install v2.

**Happy path:** A fresh installation downloads and verifies v2, applies it, and restarts; v2 boots and calls
`Commit(v2)`; the Supervisor reports `COMPLETED`. `:current` points at v2, `:previous` at v1.

**New version unhealthy:** v2 is applied but never calls `Commit` within the deadline. The Supervisor
points `:current` back at v1 and restarts; v1 boots and calls `Commit(v1)`; the update is `FAILED` with
"update v2 not confirmed within ...". The node runs v1, healthy.

**Reboot during load:** The host reboots while v2 is being loaded. The image never finished loading
and `:current` still points at v1, so v1 comes back up. Supervisor remembers its goal is to switch to v1, so it
starts the whole update process again (from download).

**Reboot after selecting v2, before restart:** `:current` already points at v2 but BOS is still running v1.
The supervisor continues by restarting BOS.

**First install ever, fails confirmation:** There is no previous version, so the rollback-pointer step is
a no-op and, when confirmation fails, the rollback finds no previous image: the update is `FAILED` with
"no previous image to roll back to". This is one case where we deliberately leave an unhealthy version installed. The
other is if, after a rollback, the old version reports as unhealthy (we don't implement double-rollback).

State Storage
-------------

The durable state is a JSON file at `<StateDir>/state.json`: the goal (target version, URL, checksum), the last committed 
version (what to roll back to), and start/finish times plus any error. This data allows the Supervisor to report update
state correctly even if it's restarted in the middle of an update.

The goal also carries an opaque deployment id supplied on `InstallUpdate`. The Supervisor never interprets it; it
persists it with the goal and echoes it back in `GetUpdateStatus`, so the caller can correlate the Supervisor's
version-keyed outcome with its own deployment-keyed record (for example, to tell a fresh request to re-install a
version apart from a stale failed attempt at that same version).