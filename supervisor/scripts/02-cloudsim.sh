#!/usr/bin/env bash
# 02-cloudsim.sh — start cloudsim as the update server and provision the test node.
#
# Starts cloudsim bound to all interfaces (so the BOS container reaches it on the bridge gateway),
# creates a site + node, runs the real /v1/device/register exchange to obtain valid client
# credentials, uploads the v1/v2/v2bad artefacts, and records everything 03-install.sh /
# 04-deploy.sh need in $STATE_ENV.
#
# Requires: jq, curl. Re-runnable: reuses a still-running cloudsim, re-creates the node/artefacts.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
# pre: the artefact tarballs and supervisor RPMs exist (01-build.sh must have run).
test -s "$BUILD/v1.tar"
test -s "$BUILD/v2.tar"
test -s "$BUILD/v2bad.tar"
for v in "$SUP_V2" "$SUP_V2BAD"; do
  compgen -G "$SUP_RPM_DIR/$v/*.rpm" >/dev/null || { echo "missing supervisor rpm $v — run 01-build.sh" >&2; exit 1; }
done

# --- 1. build + start cloudsim --------------------------------------------------------------------
# post: $CLOUDSIM_BIN is current.
echo "==> building cloudsim -> $CLOUDSIM_BIN"
( cd "$REPO" && go build -o "$CLOUDSIM_BIN" ./cmd/cloudsim )

# Reuse an already-running instance if the pid file points at a live process; else start a fresh one.
# post: cloudsim is listening on $CLOUDSIM_LISTEN with logs at $CLOUDSIM_LOG.
if [[ -f "$CLOUDSIM_PID" ]] && kill -0 "$(cat "$CLOUDSIM_PID")" 2>/dev/null; then
  echo "==> cloudsim already running (pid $(cat "$CLOUDSIM_PID"))"
else
  echo "==> starting cloudsim on $CLOUDSIM_LISTEN (log: $CLOUDSIM_LOG)"
  nohup "$CLOUDSIM_BIN" -listen "$CLOUDSIM_LISTEN" -data "$CLOUDSIM_DB" -cleanup >"$CLOUDSIM_LOG" 2>&1 &
  echo $! >"$CLOUDSIM_PID"
fi

# Wait until the management API answers, so subsequent calls don't race startup.
# post: GET $MGMT/sites returns 2xx.
echo -n "==> waiting for cloudsim"
for _ in $(seq 1 30); do
  if curl -fsS -o /dev/null "$MGMT/sites"; then echo " ok"; break; fi
  echo -n "."; sleep 0.5
done
curl -fsS -o /dev/null "$MGMT/sites" || { echo " cloudsim did not come up; see $CLOUDSIM_LOG" >&2; exit 1; }

# --- 2. create a site + node ----------------------------------------------------------------------
# post: SITE_ID names a fresh site.
SITE_ID=$(curl -fsS -X POST "$MGMT/sites" -H 'Content-Type: application/json' \
  -d '{"name":"supervisor-test"}' | jq -r '.id')
echo "==> site id=$SITE_ID"
test -n "$SITE_ID" && [[ "$SITE_ID" != null ]]

# post: NODE_ID names a podman node in that site (its initial secret is unused — we register below).
NODE_ID=$(curl -fsS -X POST "$MGMT/nodes" -H 'Content-Type: application/json' \
  -d "{\"hostname\":\"$NODE_HOSTNAME\",\"siteId\":\"$SITE_ID\",\"platform\":\"$PLATFORM\"}" | jq -r '.id')
echo "==> node id=$NODE_ID"
test -n "$NODE_ID" && [[ "$NODE_ID" != null ]]

# --- 3. real device enrollment: code -> register -> client credentials ----------------------------
# Using the genuine /v1/device/register exchange (rather than seeding the raw node secret) guarantees
# the credentials validate at /v1/device/token, sidestepping any secret-encoding assumption.
# post: CODE is a single-use enrollment code for the node.
CODE=$(curl -fsS -X POST "$MGMT/nodes/$NODE_ID/enrollment-codes" | jq -r '.code')
echo "==> enrollment code=$CODE"
test -n "$CODE" && [[ "$CODE" != null ]]

# Register over loopback. We do not use the response's host-derived bosapi_root — 03-install.sh
# writes the bridge-gateway bosapi_root into registration.json.
# post: CLIENT_ID/CLIENT_SECRET are valid node credentials.
REG=$(curl -fsS -X POST "$CLOUDSIM_LOCAL/v1/device/register" \
  -H "Authorization: Bearer $CODE" -H 'Content-Type: application/json' \
  -d '{"client_name":"rocky-10-test"}')
CLIENT_ID=$(jq -r '.client_id' <<<"$REG")
CLIENT_SECRET=$(jq -r '.client_secret' <<<"$REG")
echo "==> registered client_id=$CLIENT_ID"
test -n "$CLIENT_ID" && [[ "$CLIENT_ID" != null ]]
test -n "$CLIENT_SECRET" && [[ "$CLIENT_SECRET" != null ]]

# --- 4. upload the two update artefacts -----------------------------------------------------------
# post: ART_V2_ID / ART_V2BAD_ID name artefacts scoped to this site+platform; cloudsim computed and
#       stored each tarball's sha256 (returned to BOS on check-in; the Supervisor verifies against it).
upload() {  # upload VERSION KIND PAYLOAD -> prints artefact id
  curl -fsS -X POST "$MGMT/update-artefacts" \
    -F "version=$1" -F "platform=$PLATFORM" -F "kind=$2" -F "siteId=$SITE_ID" -F "payload=@$3" | jq -r '.id'
}
# BOS images (kind defaults to bos-image). v1 is uploaded too so it can reverse a v2 update.
ART_V1_ID=$(upload "$V1" "bos-image" "$BUILD/v1.tar")
ART_V2_ID=$(upload "$V2" "bos-image" "$BUILD/v2.tar")
ART_V2BAD_ID=$(upload "$V2BAD" "bos-image" "$BUILD/v2bad.tar")
# Supervisor RPMs (kind=supervisor-rpm): the good self-update target and the broken one (-> rollback).
ART_SUP2_ID=$(upload "$SUP_V2" "supervisor-rpm" "$(sup_rpm_path "$SUP_V2")")
ART_SUP2BAD_ID=$(upload "$SUP_V2BAD" "supervisor-rpm" "$(sup_rpm_path "$SUP_V2BAD")")
echo "==> artefacts: v1=$ART_V1_ID v2=$ART_V2_ID v2bad=$ART_V2BAD_ID sup2=$ART_SUP2_ID sup2bad=$ART_SUP2BAD_ID"
for id in "$ART_V1_ID" "$ART_V2_ID" "$ART_V2BAD_ID" "$ART_SUP2_ID" "$ART_SUP2BAD_ID"; do
  test -n "$id" && [[ "$id" != null ]]
done

# --- 5. persist captured state for the later scripts ----------------------------------------------
# post: $STATE_ENV is a sourceable file with all ids/credentials.
cat >"$STATE_ENV" <<EOF
# Generated by 02-cloudsim.sh — runtime state for the supervisor update test.
SITE_ID="$SITE_ID"
NODE_ID="$NODE_ID"
CLIENT_ID="$CLIENT_ID"
CLIENT_SECRET="$CLIENT_SECRET"
ART_V1_ID="$ART_V1_ID"
ART_V2_ID="$ART_V2_ID"
ART_V2BAD_ID="$ART_V2BAD_ID"
ART_SUP2_ID="$ART_SUP2_ID"
ART_SUP2BAD_ID="$ART_SUP2BAD_ID"
EOF
echo "==> wrote $STATE_ENV"
echo "==> 02-cloudsim.sh done"
