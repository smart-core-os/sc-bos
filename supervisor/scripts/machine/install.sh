#!/usr/bin/env bash
# machine/install.sh (runs as root INSIDE rocky-10) — install units/binary/config, seed v1, start.
# Invoked by 03-install.sh; not run directly. Idempotent: safe to re-run for a clean baseline.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../config.sh"
# pre: 02-cloudsim.sh wrote the cloud credentials we seed into registration.json.
source "$STATE_ENV"

# pre: the good v1 image is in the store, the supervisor binary and the demo users.json are on the mount.
sudo podman image exists "$IMAGE_REPO:$V1" || { echo "missing $IMAGE_REPO:$V1 — run 01-build.sh" >&2; exit 1; }
test -s "$SUP_BIN"
test -s "$USERS_JSON"

# --- 0. stop any prior run + clear durable state for a clean baseline ------------------------------
# post: neither service is running; no stale Supervisor state.json or BOS update.json survives.
sudo systemctl stop sc-bos.service 2>/dev/null || true
sudo systemctl stop sc-bos-supervisor.service 2>/dev/null || true
sudo rm -rf "$SUP_STATE_DIR" "$BOS_DATA"

# --- 1. install the Supervisor binary, config, and unit -------------------------------------------
# post: the binary is at the stable ExecStart path; config enables HTTP downloads + a 60s deadline.
sudo install -m 0755 "$SUP_BIN" "$SUP_INSTALL"
sudo install -d "$SUP_CONF_DIR"
sudo install -m 0644 "$CONF/supervisor-config.json" "$SUP_CONF_DIR/config.json"
sudo install -m 0644 "$REPO/supervisor/deploy/sc-bos-supervisor.service" "$SYSTEMD_DIR/sc-bos-supervisor.service"

# --- 2. install the BOS Quadlet unit + demo login account -----------------------------------------
# post: Quadlet will generate sc-bos.service from this unit on the next daemon-reload.
sudo install -d "$QUADLET_DIR"
sudo install -m 0644 "$CONF/sc-bos.container" "$QUADLET_DIR/sc-bos.container"
# post: the demo users.json sits at the host path the unit bind-mounts over /cfg/users.json, so the
#       baked default account is replaced by one with our known password (policy enforcement stays on).
sudo install -D -m 0644 "$USERS_JSON" "$BOS_USERS_FILE"

# --- 3. seed the stable :current tag to v1 --------------------------------------------------------
# pre:  the unit references $IMAGE_REPO:current; nothing resolves it yet.
# post: :current -> v1, so the first `systemctl start sc-bos` runs the v1 image.
sudo podman tag "$IMAGE_REPO:$V1" "$IMAGE_REPO:current"
sudo podman image exists "$IMAGE_REPO:current"

# --- 4. pre-seed the cloud registration so BOS checks in without interactive enrollment -----------
# post: <data>/cloud/registration.json (mode 0600) carries the credentials from step 02; BOS loads it
#       on startup and moves to the Connecting state, then polls /v1/device/check-in.
sudo install -d -m 0750 "$BOS_DATA/cloud"
sudo tee "$BOS_DATA/cloud/registration.json" >/dev/null <<EOF
{"client_id":"$CLIENT_ID","client_secret":"$CLIENT_SECRET","bosapi_root":"$BOSAPI_ROOT"}
EOF
sudo chmod 0600 "$BOS_DATA/cloud/registration.json"

# --- 5. start the Supervisor first (it creates the socket BOS dials), then BOS --------------------
# post: sc-bos-supervisor is active and its socket exists before BOS starts, so initSupervisor dials
#       successfully and the update integration is live.
sudo systemctl daemon-reload
sudo systemctl enable --now sc-bos-supervisor.service
echo -n "==> waiting for supervisor socket"
for _ in $(seq 1 20); do
  if sudo test -S /run/sc-bos-supervisor/supervisor.sock; then echo " ok"; break; fi
  echo -n "."; sleep 0.5
done
sudo test -S /run/sc-bos-supervisor/supervisor.sock || { echo " socket not created" >&2; exit 1; }

# post: sc-bos (v1) is started; it will check in within the poll interval and Commit(v1).
sudo systemctl start sc-bos.service

echo "==> services:"
sudo systemctl --no-pager --no-legend list-units 'sc-bos*' || true
echo "==> machine/install.sh done"
