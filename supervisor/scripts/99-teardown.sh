#!/usr/bin/env bash
# 99-teardown.sh — undo everything 02/03 set up on this host. Idempotent.
# Leaves the built binaries/images in place by default (re-runs are cheap); pass --all to wipe $BUILD.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

# --- services + units + state ---------------------------------------------------------------------
# post: both services are stopped and disabled, the Quadlet unit is gone, the Supervisor RPM (which
#       owns its binary/unit/config) is removed, and systemd has forgotten the generated units.
sudo systemctl disable --now sc-bos.service 2>/dev/null || true
sudo systemctl disable --now sc-bos-supervisor.service 2>/dev/null || true
sudo rm -f "$QUADLET_DIR/sc-bos.container"
sudo dnf -y remove sc-bos-supervisor 2>/dev/null || true
sudo systemctl daemon-reload

# post: any config left by the RPM, durable state (incl. the rpm store), BOS data dir, and the demo
#       users file are removed.
sudo rm -rf "$SUP_CONF_DIR" "$SUP_STATE_DIR" "$BOS_DATA" "$BOS_USERS_FILE"

# post: the test image tags are dropped (ignore errors if already gone). :base is the heavy
#       production-like image; teardown removes it too, so a later run rebuilds it.
for t in "$V1" "$V2" "$V2BAD" current previous base; do
  sudo podman rmi -f "$IMAGE_REPO:$t" 2>/dev/null || true
done

# --- cloudsim ------------------------------------------------------------------------------------
# post: no cloudsim process; the db + artefacts dir + captured state are removed.
if [[ -f "$CLOUDSIM_PID" ]] && kill -0 "$(cat "$CLOUDSIM_PID")" 2>/dev/null; then
  echo "==> stopping cloudsim (pid $(cat "$CLOUDSIM_PID"))"
  kill "$(cat "$CLOUDSIM_PID")" 2>/dev/null || true
fi
rm -f "$CLOUDSIM_PID" "$CLOUDSIM_DB" "$STATE_ENV"
rm -rf "${CLOUDSIM_DB%.db}-artefacts"   # cloudsim names the artefacts dir from the db basename

# --- optional: remove all build outputs -----------------------------------------------------------
if [[ "${1:-}" == "--all" ]]; then
  echo "==> removing build outputs ($BUILD)"
  rm -rf "$BUILD"
fi
echo "==> 99-teardown.sh done"
