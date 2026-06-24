#!/usr/bin/env bash
# machine/teardown.sh (runs as root INSIDE rocky-10) — remove the test deployment from the machine.
# Invoked by 99-teardown.sh. Touches only this project's files (see the harness guardrails).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../config.sh"

# post: both services are stopped and disabled.
sudo systemctl disable --now sc-bos.service 2>/dev/null || true
sudo systemctl disable --now sc-bos-supervisor.service 2>/dev/null || true

# post: the units we installed are gone and systemd has forgotten the generated sc-bos.service.
sudo rm -f "$QUADLET_DIR/sc-bos.container" \
           "$SYSTEMD_DIR/sc-bos-supervisor.service"
sudo systemctl daemon-reload

# post: installed binary/config + durable state + BOS data dir + demo users file are removed.
sudo rm -rf "$SUP_INSTALL" "$SUP_CONF_DIR" "$SUP_STATE_DIR" "$BOS_DATA" "$BOS_USERS_FILE"

# post: the test image tags are dropped (ignore errors if already gone). :base is the heavy
#       production-like image; teardown removes it too, so a later run rebuilds it.
for t in "$V1" "$V2" "$V2BAD" current previous base; do
  sudo podman rmi -f "$IMAGE_REPO:$t" 2>/dev/null || true
done

echo "==> machine/teardown.sh done"
