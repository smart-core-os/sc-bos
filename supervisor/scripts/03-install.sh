#!/usr/bin/env bash
# 03-install.sh (run on the Mac) — install + start the Supervisor and BOS v1 inside rocky-10.
# Thin wrapper: all host mutation happens as root in the machine via machine/install.sh.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
# pre: build outputs and captured cloud state exist (01-build.sh and 02-cloudsim.sh have run).
test -s "$SUP_BIN"   || { echo "missing $SUP_BIN — run 01-build.sh" >&2; exit 1; }
test -s "$STATE_ENV" || { echo "missing $STATE_ENV — run 02-cloudsim.sh" >&2; exit 1; }

# --- generate the demo local account (fixed password) -------------------------------------------
# cmd/pash reads the password on stdin and prints a bcrypt hash (BOS's local-account secret format).
# post: $USERS_JSON is a one-account users.json (superAdmin) BOS will validate logins against.
echo "==> generating demo admin account ($ADMIN_USER) via cmd/pash"
HASH=$(printf '%s\n' "$ADMIN_PASSWORD" | ( cd "$REPO" && go run ./cmd/pash ))
test -n "$HASH"
jq -n --arg id "$ADMIN_USER" --arg h "$HASH" \
  '[{id:$id, roles:["superAdmin","admin"], title:"Demo admin", secrets:[{hash:$h}]}]' > "$USERS_JSON"

echo "==> installing supervisor + BOS v1 in $MACHINE"
run_in_machine "$SCRIPTS/machine/install.sh"
echo "==> 03-install.sh done (Ops UI login: $ADMIN_USER / $ADMIN_PASSWORD)"
