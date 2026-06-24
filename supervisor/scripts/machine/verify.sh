#!/usr/bin/env bash
# machine/verify.sh (runs as root INSIDE rocky-10) — print the assertion surface for the test.
# Read-only. Invoked by 05-verify.sh.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../config.sh"

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
