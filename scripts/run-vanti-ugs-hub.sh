#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
cd "$ROOT_DIR"

# ANSI colour codes for log prefixes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RESET='\033[0m'

pids=()

cleanup() {
  echo "Shutting down..."
  for pid in "${pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait "${pids[@]}" 2>/dev/null || true
}
trap cleanup INT TERM

# prefix_log <label> <colour> <cmd> [args...]
prefix_log() {
  local label="$1"
  local colour="$2"
  shift 2
  "$@" 2>&1 | while IFS= read -r line; do
    printf "${colour}[%-8s]${RESET} %s\n" "$label" "$line"
  done &
  pids+=($!)
}

echo "Building .bin/bos..."
go build -o .bin/bos ./cmd/bos

echo "Starting BC-01, EG-01, and vanti-ugs AC..."

prefix_log "BC-01" "$GREEN" \
  .bin/bos --policy-mode=off \
    --sysconf example/config/vanti-ugs-cohort/bc-01/system.json \
    --data .data/vanti-ugs-hub/bc-01

prefix_log "EG-01" "$BLUE" \
  .bin/bos --policy-mode=off \
    --sysconf example/config/vanti-ugs-cohort/eg-01/system.json \
    --data .data/vanti-ugs-hub/eg-01

prefix_log "AC" "$RED" \
  .bin/bos --policy-mode=off \
    --appconf example/config/vanti-ugs-cohort/ac-01/app.conf.json \
    --sysconf example/config/vanti-ugs-cohort/ac-01/system.json \
    --data .data/vanti-ugs-hub/ac-01

prefix_log "UI" "$YELLOW" \
  yarn --cwd "$ROOT_DIR/ui/ops" run dev --mode=vanti-ugs-cohort

echo "All processes started. Press Ctrl+C to stop."
wait
