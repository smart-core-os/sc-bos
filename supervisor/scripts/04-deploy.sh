#!/usr/bin/env bash
# 04-deploy.sh <v1|v2|v2bad> — create an update deployment and watch it settle.
#
#   ./04-deploy.sh v2      # happy path: expect the deployment to reach COMPLETED
#   ./04-deploy.sh v2bad   # rollback:   expect IN_PROGRESS then FAILED (node stays on the prior good version)
#   ./04-deploy.sh v1      # reverse a v2 update back to v1, so you can run the cycle again
#
# Deploy the version the node is NOT currently running (deploying the running version is a no-op the
# Supervisor can't confirm, so it would roll back). The normal cycle is: v2 -> v1 -> v2 -> ...
#
# Polls the deployment status until it reaches a terminal state or the timeout elapses. The actual
# install runs device-side: BOS picks up latestUpdate on its next ~30s check-in, calls the Supervisor,
# which downloads/loads/tag-swaps/restarts; outcome is reported back on a later check-in.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
# pre: cloud state captured.
test -s "$STATE_ENV" || { echo "missing $STATE_ENV — run 02-cloudsim.sh" >&2; exit 1; }
source "$STATE_ENV"

# --- resolve the requested artefact ---------------------------------------------------------------
case "${1:-}" in
  v1)    ART_ID="$ART_V1_ID";    EXPECT="completed" ;;
  v2)    ART_ID="$ART_V2_ID";    EXPECT="completed" ;;
  v2bad) ART_ID="$ART_V2BAD_ID"; EXPECT="failed" ;;
  *) echo "usage: $0 <v1|v2|v2bad>" >&2; exit 2 ;;
esac
test -n "${ART_ID:-}" && [[ "$ART_ID" != null ]] || { echo "no artefact id for '$1' in $STATE_ENV — re-run 02-cloudsim.sh" >&2; exit 1; }
echo "==> deploying artefact $1 (id=$ART_ID) to node $NODE_ID; expecting '$EXPECT'"

# --- create the deployment ------------------------------------------------------------------------
# pre:  no other non-terminal deployment for this node (cloudsim 409s otherwise; a pending one is
#       auto-cancelled). post: DEP_ID is PENDING and will be offered to the node on its next check-in.
DEP_ID=$(curl -fsS -X POST "$MGMT/update-deployments" -H 'Content-Type: application/json' \
  -d "{\"updateArtefactId\":\"$ART_ID\",\"nodeId\":\"$NODE_ID\"}" | jq -r '.id')
echo "==> deployment id=$DEP_ID"
test -n "$DEP_ID" && [[ "$DEP_ID" != null ]]

# --- poll until terminal --------------------------------------------------------------------------
# A happy-path install needs: one check-in to fetch it, install+restart, one check-in to report.
# A rollback additionally waits out the 60s commitDeadline plus the rollback restart. Budget ~5 min.
# post: prints the final status; exits non-zero if it didn't match EXPECT (or timed out).
deadline=$(( SECONDS + 300 ))
last=""
while (( SECONDS < deadline )); do
  body=$(curl -fsS "$MGMT/update-deployments/$DEP_ID")
  status=$(jq -r '.status' <<<"$body")
  if [[ "$status" != "$last" ]]; then
    echo "    [$(date +%H:%M:%S)] status=$status $(jq -rc '{reason}' <<<"$body")"
    last="$status"
  fi
  case "$status" in
    completed|failed|cancelled) break ;;
  esac
  sleep 5
done

echo "==> final status: $last"
if [[ "$last" == "$EXPECT" ]]; then
  echo "==> PASS (got expected '$EXPECT')"
else
  echo "==> FAIL (expected '$EXPECT', got '$last') — run 05-verify.sh for device-side detail" >&2
  exit 1
fi
