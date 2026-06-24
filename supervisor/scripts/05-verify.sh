#!/usr/bin/env bash
# 05-verify.sh (run on the Mac) — dump the device-side state that proves what happened.
# Thin wrapper around machine/verify.sh; read-only.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh
run_in_machine "$SCRIPTS/machine/verify.sh"
