#!/usr/bin/env bash
# 01-build.sh (run on the Mac) — build everything the test needs.
#
# Cross-compiles the Supervisor binary on the Mac (it runs as a host systemd service), then drives
# podman in the machine to build the production-like BOS base image from the repo's main Dockerfile
# (UI included), the v1/v2/v2bad images, and the v1/v2/v2bad artefact tarballs.
#
# Idempotent: existing outputs are reused unless FORCE=1 is set.
#   ./01-build.sh          # build only what's missing
#   FORCE=1 ./01-build.sh  # rebuild everything (incl. the full base image)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

FORCE="${FORCE:-0}"
mkdir -p "$BUILD"

# --- 1. cross-compile the Supervisor binary (host binary, runs as a systemd service) --------------
# pre:  Go toolchain on the Mac; the supervisor package compiles.
# post: $SUP_BIN is a linux/arm64 executable the machine can run at /usr/local/bin.
if [[ "$FORCE" == 1 || ! -x "$SUP_BIN" ]]; then
  echo "==> building supervisor -> $SUP_BIN"
  ( cd "$REPO" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -o "$SUP_BIN" ./supervisor )
else
  echo "==> supervisor binary present, skipping (FORCE=1 to rebuild)"
fi
# assert: the binary exists and is non-empty.
test -s "$SUP_BIN"

# --- 2. stage the demo node config (vanti-ugs, adapted to run DB-less) ----------------------------
# The deployed images ship the vanti-ugs example (mock devices + ops/cohort Ops UI), adapted so the
# node needs no Postgres: history automations write to sqlite, and the Ops UI uses local-auth login
# (required because policy enforcement stays on). Staged here rather than committed into conf/ to avoid
# duplicating the large example. post: $BOS_CTX is a self-contained build context for the extension.
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
echo "==> staging demo config from $VANTI_UGS -> $BOS_CTX"
rm -rf "$BOS_CTX"; mkdir -p "$BOS_CTX/ui-config"
# app.conf: vanti-ugs, with every history automation's postgres storage switched to sqlite (the only
# non-postgres storage the history automation supports besides bolt/memory; sqlite auto-opens at
# /data/history.sqlite3). The only "postgres" storage refs in app.conf are these history automations.
sed 's/"type": "postgres"/"type": "sqlite"/g' "$VANTI_UGS/app.conf.json" > "$BOS_CTX/app.conf.json"
# assert: no postgres storage remains.
! grep -q '"type": "postgres"' "$BOS_CTX/app.conf.json"
# system.conf: our DB-less overlay (supervisor + cloud + staticHosting + fileAccounts; no postgres
# systems). post: copied verbatim into the context.
cp "$CONF/system.conf.json" "$BOS_CTX/system.conf.json"
cp "$CONF/Containerfile.bos" "$BOS_CTX/Containerfile"
# ui-config: vanti-ugs (hub:true -> Cohort page, ops:true), but enable local-auth login so the UI
# authenticates under policy enforcement (vanti-ugs ships auth disabled for a keycloak/dev setup).
jq '.config.auth.disabled = false | .config.auth.providers = ["localAuth"]' \
  "$VANTI_UGS/ui-config.json" > "$BOS_CTX/ui-config/ui-config.json"
# assets referenced by ui-config as ./assets/... resolve relative to /__/scos/, so co-locate them.
cp -R "$VANTI_UGS/assets" "$BOS_CTX/ui-config/assets"
# assert: the context is self-contained.
test -f "$BOS_CTX/app.conf.json" && test -f "$BOS_CTX/ui-config/ui-config.json"

# --- 3. build images + save tarballs inside the machine -------------------------------------------
# pre:  the supervisor binary is staged; podman is available in the machine; the repo (incl. the main
#       Dockerfile) is visible at $REPO via the bind mount.
# post: localhost/smartcore/bos:{base,v1,v2,v2bad} exist in the machine's root image store, and
#       $BUILD/{v1,v2,v2bad}.tar are world-readable podman-save tarballs cloudsim can upload.
# FORCE is passed as an argument because env vars do not survive `container machine run` + sudo.
echo "==> building images + saving tarballs in $MACHINE"
run_in_machine "$SCRIPTS/machine/build.sh" "$FORCE"

echo "==> 01-build.sh done"
