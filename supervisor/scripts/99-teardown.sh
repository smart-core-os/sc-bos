#!/usr/bin/env bash
# 99-teardown.sh (run on the Mac) — undo everything 02/03 set up, on both the Mac and the machine.
# Leaves the built binaries/images in place by default (re-runs are cheap); pass --all to wipe $BUILD.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

# --- machine side: stop services, remove units/state, drop test image tags ------------------------
echo "==> tearing down inside $MACHINE"
run_in_machine "$SCRIPTS/machine/teardown.sh"

# --- Mac side: stop cloudsim ----------------------------------------------------------------------
# post: no cloudsim process; the db + artefacts dir are removed.
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
