#!/usr/bin/env bash
# 05-verify.sh — dump the device-side state that proves what happened. Read-only.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

# which_version TAG — print which of v1/v2/v2bad the given tag's image id matches (tag indirection).
which_version() {
  local id want
  id=$(sudo podman image inspect --format '{{.Id}}' "$IMAGE_REPO:$1" 2>/dev/null) || { echo "<absent>"; return; }
  for want in "$V1" "$V2" "$V2BAD"; do
    if [[ "$id" == "$(sudo podman image inspect --format '{{.Id}}' "$IMAGE_REPO:$want" 2>/dev/null)" ]]; then
      echo "$want"; return
    fi
  done
  echo "$id"
}

echo "===== tag indirection (what the Supervisor swapped) ====="
# post: shows which version :current (running) and :previous (rollback pointer) resolve to.
echo "  :current  -> $(which_version current)"
echo "  :previous -> $(which_version previous)"

echo
echo "===== running BOS container ====="
sudo podman ps --filter name=sc-bos --format 'table {{.Names}} {{.Image}} {{.Status}}' || true

echo
echo "===== installed Supervisor RPM (self-update result) ====="
# post: the installed version is sup2 after a successful self-update, or back to sup1 after a rollback.
rpm -q sc-bos-supervisor || echo "  (sc-bos-supervisor rpm not installed)"

echo
echo "===== Supervisor self-update state ====="
# post: self-update.json shows the most recent self-update's phase (completed/failed) and any reason.
if sudo test -f "$SUP_STATE_DIR/self-update.json"; then
  sudo cat "$SUP_STATE_DIR/self-update.json"; echo
else
  echo "  (no self-update.json yet)"
fi

echo
echo "===== Supervisor durable state (committed version / last error) ====="
# post: state.json shows the committed (running, healthy) version and any failure reason.
if sudo test -f "$SUP_STATE_DIR/state.json"; then
  sudo cat "$SUP_STATE_DIR/state.json"; echo
else
  echo "  (no state.json yet)"
fi

echo
echo "===== recent Supervisor log ====="
sudo journalctl -u sc-bos-supervisor.service --no-pager -n 40 || true

echo
echo "===== recent BOS log (commit / check-in lines) ====="
sudo journalctl -u sc-bos.service --no-pager -n 40 || true
