# Supervisor update — end-to-end test harness

Repeatable scripts that exercise the real BOS software-update flow against actual podman + systemd.
**cloudsim** is the update server and the local podman/systemd installation is the update target;
everything runs on **one Rocky Linux 10 host** that you provision. They verify both the happy path
(update applied and committed) and the Supervisor's local auto-rollback (a target that never commits
is reverted).

This is dev/test tooling — not part of any production deployment.

## What it does

cloudsim hands the node a `latestUpdate` on check-in → BOS forwards the artefact ref + download URL to
the Supervisor over a Unix socket → the Supervisor downloads, verifies, `podman load`s, swaps the
`localhost/smartcore/bos:current` tag and restarts the `sc-bos` Quadlet unit → the new BOS calls
`Commit(version)` when healthy. No `Commit` within `commitDeadline` ⇒ rollback to `:previous`. BOS
reports the outcome (COMPLETED / FAILED) on its next check-in.

The base image is built from the repo's main `Dockerfile` (Ops UI included). Two versions (`v1`, `v2`)
extend it, baking a distinct `BOS_VERSION_OVERRIDE` so the Supervisor can tell a successful update from
a rollback; `v2bad` is an inert image that never commits, to drive rollback.

## Prerequisites

Run these directly on a **Rocky Linux 10 host** (a VM or bare metal that you have set up), as a user
with passwordless `sudo`, or as root. The host needs:

- `go` (builds the Supervisor binary and cloudsim),
- `podman` (rootful) + `systemd`,
- `jq` and `curl`.

**Networking.** cloudsim runs on the host bound to `0.0.0.0:8080`. The BOS container uses default
(bridge) networking and reaches cloudsim at the Podman bridge gateway IP, which `03-install.sh`
discovers and seeds into `registration.json`. The Supervisor runs as a host service and downloads the
same payload URL — it reaches cloudsim at that same gateway address, so nothing on the host filesystem
is modified. Ensure the host firewall allows the Podman bridge to reach `:8080`. The Ops UI and gRPC
API are published to the host (ports 443, 23557).

## Run order

```sh
cd supervisor/scripts
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

The deployed image is production-like — built straight from the repo's main `Dockerfile` with the Ops
UI baked in, then extended with the demo node's config. That config is the **vanti-ugs example**
(`example/config/vanti-ugs`: mock devices/zones plus the ops + Cohort Ops UI config), adapted by
`01-build.sh` to run without a database:

- history automations write to **sqlite** (`/data/history.sqlite3`) instead of Postgres;
- the postgres-only systems (alerts, tenants, accounts) are dropped from `system.conf`;
- the Ops UI is set to **local-auth** login (vanti-ugs ships auth disabled for a keycloak dev setup),
  because policy enforcement is left **on** (BOS default) so calls must carry a token.

After `03-install.sh`, browse to **`https://localhost`** on the box (or `https://<box-ip>` from
elsewhere — port 443 is published). Accept the self-signed cert, then log in with the demo account
printed by `03` — by default **`admin` / `admin`** (change via `ADMIN_USER`/`ADMIN_PASSWORD` in
`config.sh`). The account is generated with `cmd/pash` and bind-mounted over `/cfg/users.json`.

## Layout

| Path | Purpose |
|---|---|
| `config.sh` | shared variables, sourced by every script |
| `NN-*.sh` | the ordered driver scripts you invoke |
| `conf/` | the BOS `system.conf` overlay, Supervisor `config.json`, Quadlet unit, Containerfiles, build ignorefile |
| `.build/` | gitignored scratch: binaries, tarballs, `state.env`, `users.json`, cloudsim db/logs |

## Notes

- The BOS base image is built from the repo `Dockerfile` with `conf/dockerignore` (keeps the context
  small without touching the repo's committed `.dockerignore`).
- The Supervisor's `config.json` sets `allowInsecureDownloads: true` so it accepts cloudsim's plain-HTTP
  payload URL (production defaults to HTTPS-only). The `commitDeadline` is shortened to `60s`.
- BOS enrollment is pre-seeded into `<data>/cloud/registration.json` using credentials from a real
  `/v1/device/register` exchange (run by `02`), so no interactive enrollment is needed.
- No config version is deployed; BOS falls back to its baked-in local app config, which is enough to
  boot, check in, commit, and serve the UI.
