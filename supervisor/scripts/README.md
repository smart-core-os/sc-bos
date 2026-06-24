# Supervisor update — end-to-end test harness

Repeatable scripts that exercise the real BOS software-update flow against actual podman + systemd:
**cloudsim** (on the Mac) is the update server, and an Apple **container machine `rocky-10`** is the
Rocky/Podman update target. They verify both the happy path (update applied and committed) and the
Supervisor's local auto-rollback (a target that never commits is reverted).

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

- The `rocky-10` container machine running (`container machine ls`), with the Mac home bind-mounted.
- `go`, `jq`, `curl`, and the `container` CLI on the Mac.
- **Host redirect** (one-time, needs admin): the macOS Application Firewall blocks the VM from
  reaching host services directly, so cloudsim (on the Mac, loopback) is exposed to the machine via
  Apple container's documented mechanism:

  ```sh
  sudo container system dns create host.container.internal --localhost 203.0.113.113
  ```

  This maps `203.0.113.113` → the host's localhost; the machine and the (host-networked) BOS container
  reach cloudsim at `http://203.0.113.113:8080`. Re-run it after a host/machine restart (the rule does
  not survive a reboot) and note it disables iCloud Private Relay while active. Change `HOST_REDIRECT_IP`
  in `config.sh` if you pick a different documentation-range IP.

## Run order

```sh
cd supervisor/scripts
./01-build.sh            # cross-build binaries (Mac); build v1/v2/v2bad images + tarballs (machine)
./02-cloudsim.sh         # start cloudsim, create site+node, enroll, upload artefacts -> .build/state.env
./03-install.sh          # install Supervisor + BOS v1 in rocky-10, seed :current=v1, start both
./04-deploy.sh v2        # happy path  — expect COMPLETED (:current -> v2, :previous -> v1)
./05-verify.sh           # dump tag indirection, Supervisor state.json, and logs
./04-deploy.sh v2bad     # rollback    — expect FAILED   (:current back to v2 after the deadline)
./05-verify.sh
./04-deploy.sh v1        # reverse back to v1 so you can run the cycle again (v2 -> v1 -> v2 -> ...)
./99-teardown.sh         # stop + remove everything (add --all to also wipe .build/)
```

Re-running is safe: `01` skips outputs that exist (`FORCE=1` to rebuild), `03` clears prior durable
state for a clean baseline, and `99` is idempotent.

## Layout

| Path | Side | Purpose |
|---|---|---|
| `config.sh` | both | shared variables + machine-exec helpers (sourced everywhere) |
| `NN-*.sh` | Mac | the ordered driver scripts you invoke |
| `machine/*.sh` | rocky-10 | root-side steps, run via `container machine run` |
| `conf/` | — | the BOS `system.conf` overlay, Supervisor `config.json`, Quadlet unit, Containerfiles |
| `.build/` | both | gitignored scratch: binaries, tarballs, `state.env`, cloudsim db/logs |

## Loading the Ops UI

The deployed image is production-like — built straight from the repo's main `Dockerfile` with the Ops
UI baked in, then extended with the demo node's config. That config is the **vanti-ugs example**
(`example/config/vanti-ugs`: mock devices/zones plus the ops + Cohort Ops UI config), adapted by
`01-build.sh` to run without a database:

- history automations write to **sqlite** (`/data/history.sqlite3`) instead of Postgres;
- the postgres-only systems (alerts, tenants, accounts) are dropped from `system.conf`;
- the Ops UI is set to **local-auth** login (vanti-ugs ships auth disabled for a keycloak dev setup),
  because policy enforcement is left **on** (BOS default) so calls must carry a token.

After `03-install.sh`, browse to **`https://<machine-ip>`** (e.g. `https://192.168.64.5`; get the IP
with `container machine inspect rocky-10`). Accept the self-signed cert, then log in with the demo
account printed by `03` — by default **`admin` / `admin`** (change via `ADMIN_USER`/`ADMIN_PASSWORD`
in `config.sh`). The account is generated with `cmd/pash` and bind-mounted over `/cfg/users.json`.

## Layout

| Path | Side | Purpose |
|---|---|---|
| `config.sh` | both | shared variables + machine-exec helpers (sourced everywhere) |
| `NN-*.sh` | Mac | the ordered driver scripts you invoke |
| `machine/*.sh` | rocky-10 | root-side steps, run via `container machine run` |
| `conf/` | — | `system.conf` overlay, Supervisor `config.json`, Quadlet unit, Containerfile, build ignorefile |
| `.build/` | both | gitignored scratch: binary, tarballs, `state.env`, `users.json`, cloudsim db/logs |

## Notes

- The BOS base image is built from the repo `Dockerfile` inside the machine with `--network=host` (the
  nested podman bridge is unavailable) and `conf/dockerignore` (keeps the context small without touching
  the repo's committed `.dockerignore`). The BOS container also runs with `Network=host`.
- The Supervisor's `config.json` sets `allowInsecureDownloads: true` so it accepts cloudsim's plain-HTTP
  payload URL (production defaults to HTTPS-only). The `commitDeadline` is shortened to `60s`.
- BOS enrollment is pre-seeded into `<data>/cloud/registration.json` using credentials from a real
  `/v1/device/register` exchange (run by `02`), so no interactive enrollment is needed.
- No config version is deployed; BOS falls back to its baked-in local app config, which is enough to
  boot, check in, commit, and serve the UI.
