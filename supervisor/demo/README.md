# Supervisor update demo

Scripts that demonstrate the real BOS software-update flow against actual podman + systemd.
**cloudsim** is the update server and the local podman/systemd installation is the update target;
everything runs on one Rocky Linux 10. The demo shows a successful update and a failed update followed by the 
Supervisor's local auto-rollback (a target that never commits is reverted). See [`../docs/state-model.md`](../docs/state-model.md) for the 
update mechanism and crash-safety model behind it.

## Deployed images

The base image is built from the repo's main `Dockerfile` (Ops UI included). Two versions (`v1`, `v2`)
extend it, baking a distinct `BOS_VERSION_OVERRIDE` so the Supervisor can tell a successful update from
a rollback; `v2bad` is an inert image that never commits, to drive rollback.

Images include example configs based on vanti-ugs, and the Ops UI.

## Prerequisites

Run on a Rocky Linux 10 VM you have prepared for the demo. I recommend enabling passwordless sudo.
Ensure the following packages are installed, and the repo is on the filesystem.

- `go` 
- `podman` (rootful) + `systemd`,
- `jq` and `curl`.

**Networking.** cloudsim runs (non-containerised) on the host bound to `0.0.0.0:8080`. The BOS container uses default
(bridge) networking and reaches cloudsim at the Podman bridge gateway IP, which `03-install.sh`
discovers and seeds into `registration.json`. The Supervisor runs as a (non-containerised) service and downloads the
same payload URL - it reaches cloudsim at that same gateway address, so nothing on the host filesystem
is modified. 

## Run order

```sh
cd supervisor/demo
./01-build.sh            # build the supervisor binary, the v1/v2/v2bad images, and artefact tarballs
./02-cloudsim.sh         # start cloudsim, create site+node, enroll, upload artefacts -> .build/state.env
./03-install.sh          # install Supervisor + BOS v1, seed :current=v1, start both
./04-deploy.sh v2        # happy path  — expect COMPLETED (:current -> v2, :previous -> v1)
./05-verify.sh           # dump tag indirection, Supervisor state.json, and logs
./04-deploy.sh v2bad     # rollback    — expect FAILED   (:current back to v2 after the deadline)
./05-verify.sh
./04-deploy.sh v1        # reverse back to v1 so you can run the cycle again (v2 -> v1 -> v2 -> ...)
./99-teardown.sh         # stop + remove everything (add --all to also wipe .build/)
```

Re-running is safe: `01` skips outputs that exist (`FORCE=1` to rebuild), `03` clears prior durable
state for a clean baseline, and `99` is idempotent.

## Loading the Ops UI

After `03-install.sh`, BOS will be available on port 443. Accept the self-signed cert, then log in with the demo account
printed by `03` - by default **`admin` / `admin`** (change via `ADMIN_USER`/`ADMIN_PASSWORD` in
`config.sh`). 

## Notes

- The Supervisor's `config.json` sets `allowInsecureDownloads: true` so it accepts cloudsim's plain-HTTP
  payload URL, and shortens `commitDeadline` to `60s` make the demo more responsive.
- BOS enrollment is pre-seeded into `cloud/registration.json` on the named `sc-bos-data` volume using
  credentials from a real `/v1/device/register` exchange (run by script 2), so no interactive enrollment is
  needed.
- No config version is deployed; BOS falls back to its baked-in local app config, which is enough to
  boot, check in, commit, and serve the UI.

## Appendix: running on macOS using Container Machines

On macOS, you can use
[Apple container machine](https://github.com/apple/container/blob/main/docs/container-machine.md) to run the Rocky Linux
10 VM. The [`machine.Containerfile`](machine.Containerfile) builds a machine image with every
prerequisite baked in (rootful podman, Go, `jq`, `curl`) and passwordless sudo for your mapped user.

```sh
container build -t local/sc-bos-demo-machine:latest -f supervisor/demo/machine.Containerfile .
container machine create local/sc-bos-demo-machine:latest --name sc-bos-demo
```

Your macOS home is mounted at the same path inside the machine, so the repo checkout is already there:
`container machine run -n sc-bos-demo`, then `cd` to `supervisor/demo` and follow the run order above.
