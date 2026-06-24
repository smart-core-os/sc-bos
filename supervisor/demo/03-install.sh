#!/usr/bin/env bash
# 03-install.sh — install + start the Supervisor and BOS v1 on this host. Idempotent: safe to re-run
# for a clean baseline. Uses sudo for the host mutations (run as a sudo-capable user, or as root).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
# pre: build outputs, the good v1 image, and captured cloud state exist (01 + 02 have run).
test -s "$SUP_BIN"   || { echo "missing $SUP_BIN — run 01-build.sh" >&2; exit 1; }
test -s "$STATE_ENV" || { echo "missing $STATE_ENV — run 02-cloudsim.sh" >&2; exit 1; }
sudo podman image exists "$IMAGE_REPO:$V1" || { echo "missing $IMAGE_REPO:$V1 — run 01-build.sh" >&2; exit 1; }

# 02-cloudsim.sh wrote the cloud credentials we seed into registration.json.
source "$STATE_ENV"

# --- generate the demo local account (fixed password) -------------------------------------------
# cmd/pash reads the password on stdin and prints a bcrypt hash (BOS's local-account secret format).
# post: $USERS_JSON is a one-account users.json (superAdmin) BOS will validate logins against.
echo "==> generating demo admin account ($ADMIN_USER) via cmd/pash"
HASH=$(printf '%s\n' "$ADMIN_PASSWORD" | ( cd "$REPO" && go run ./cmd/pash ))
test -n "$HASH"
jq -n --arg id "$ADMIN_USER" --arg h "$HASH" \
  '[{id:$id, roles:["superAdmin","admin"], title:"Demo admin", secrets:[{hash:$h}]}]' > "$USERS_JSON"

# --- 0. stop any prior run + clear durable state for a clean baseline ------------------------------
# post: neither service is running; no stale Supervisor state.json survives.
sudo systemctl stop sc-bos.service 2>/dev/null || true
sudo systemctl stop sc-bos-supervisor.service 2>/dev/null || true
sudo rm -rf "$SUP_STATE_DIR"
sudo podman volume rm -f "$BOS_DATA_VOLUME" 2>/dev/null || true

# --- 1. install the Supervisor binary, config, and unit -------------------------------------------
# post: the binary is at the stable ExecStart path; config enables HTTP downloads + a 60s deadline.
sudo install -m 0755 "$SUP_BIN" "$SUP_INSTALL"
sudo install -d "$SUP_CONF_DIR"
sudo install -m 0644 "$CONF/supervisor-config.json" "$SUP_CONF_DIR/config.json"
sudo install -m 0644 "$CONF/sc-bos-supervisor.service" "$SYSTEMD_DIR/sc-bos-supervisor.service"

# --- 2. install the BOS Quadlet unit + demo login account -----------------------------------------
# post: Quadlet will generate sc-bos.service from this unit on the next daemon-reload.
sudo install -d "$QUADLET_DIR"
sudo install -m 0644 "$CONF/sc-bos.container" "$QUADLET_DIR/sc-bos.container"
# post: the demo users.json sits at the host path the unit bind-mounts over /cfg/users.json, so the
#       baked default account is replaced by one with our known password (policy enforcement stays on).
sudo install -D -m 0644 "$USERS_JSON" "$BOS_USERS_FILE"

# --- 3. seed the stable :current tag to v1 --------------------------------------------------------
# post: :current -> v1, so the first `systemctl start sc-bos` runs the v1 image.
sudo podman tag "$IMAGE_REPO:$V1" "$IMAGE_REPO:current"
sudo podman image exists "$IMAGE_REPO:current"

# --- 4. pre-seed the cloud registration so BOS checks in without interactive enrollment -----------
# bosapi_root points at cloudsim via the Podman bridge gateway, which both the in-container BOS and
# the host Supervisor (which performs the download) reach. post: cloud/registration.json (mode 0600)
# in the named data volume carries the credentials from step 02; BOS loads it on startup and moves to
# the Connecting state, then polls /v1/device/check-in via bosapi_root.
GATEWAY=$(sudo podman network inspect podman --format '{{(index .Subnets 0).Gateway}}')
test -n "$GATEWAY" || { echo "could not determine podman gateway ip" >&2; exit 1; }
BOSAPI_ROOT="http://${GATEWAY}:${CLOUDSIM_PORT}"
echo "==> bosapi_root=$BOSAPI_ROOT (podman bridge gateway)"
# Seed registration.json into the named data volume before BOS starts. Step 0 removed any prior
# volume, so create a fresh one and write into its mountpoint (rootful podman, so it is host-readable).
sudo podman volume create "$BOS_DATA_VOLUME" >/dev/null
BOS_DATA_MOUNT=$(sudo podman volume inspect "$BOS_DATA_VOLUME" --format '{{.Mountpoint}}')
sudo install -d -m 0750 "$BOS_DATA_MOUNT/cloud"
sudo tee "$BOS_DATA_MOUNT/cloud/registration.json" >/dev/null <<EOF
{"client_id":"$CLIENT_ID","client_secret":"$CLIENT_SECRET","bosapi_root":"$BOSAPI_ROOT"}
EOF
sudo chmod 0600 "$BOS_DATA_MOUNT/cloud/registration.json"

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
echo "==> 03-install.sh done (Ops UI login: $ADMIN_USER / $ADMIN_PASSWORD)"
