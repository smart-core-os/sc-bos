#!/usr/bin/env bash
# Pre-commit / pre-PR gate: runs the CI checks that apply to what changed on this branch.
#
# By default only the changed Go packages (and the packages that depend on them) and the changed
# UI workspaces are checked. Use --full to check everything, including the race and e2e tests CI runs.
#
# Logs for each step are written to .tmp/gate/<step>.log; grep those rather than re-running.
set -o pipefail

usage() {
  cat <<'EOF'
Usage: scripts/gate.sh [--base <ref>] [--race] [--full]

  --base <ref>  compare against <ref> instead of the merge base with origin/main
  --race        also run the changed Go packages' short tests under the race detector
  --full        check everything as CI does: all Go packages (including e2e and race),
                every UI workspace, rego policy tests, and proto generation
EOF
}

BASE=""
RACE=0
FULL=0
while [ $# -gt 0 ]; do
  case "$1" in
    --base)
      [ $# -ge 2 ] || { echo "--base needs a ref" >&2; usage >&2; exit 2; }
      BASE="$2"; shift 2 ;;
    --race) RACE=1; shift ;;
    --full) FULL=1; RACE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT" || exit 2
MODULE=$(go list -m)
LOGDIR="$ROOT/.tmp/gate"
mkdir -p "$LOGDIR"
rm -f "$LOGDIR"/*.log

if [ -z "$BASE" ]; then
  BASE=$(git merge-base HEAD origin/main 2>/dev/null) || {
    echo "could not find the merge base with origin/main; pass --base <ref>" >&2
    exit 2
  }
fi

# Committed since BASE, plus uncommitted and untracked changes. --no-renames lists a moved file under both
# paths, so the package it left (and that package's importers) is checked too.
CHANGED=$( {
  git diff --no-renames --name-only "$BASE"...HEAD
  git diff --no-renames --name-only HEAD
  git ls-files --others --exclude-standard
} | sort -u)

FAILED=()
PASSED=()
SKIPPED=()

# run <name> <cmd...>: runs cmd with output to .tmp/gate/<name>.log and records the result.
run() {
  local name="$1"; shift
  local log="$LOGDIR/$name.log"
  printf '%-14s ' "$name"
  if "$@" >"$log" 2>&1; then
    echo "ok"
    PASSED+=("$name")
  else
    echo "FAIL (.tmp/gate/$name.log)"
    # Prefer the lines that say what failed; go test output buries them under passing packages.
    if grep -qE '^(--- FAIL|FAIL[[:space:]]|panic:|SIGSEGV)' "$log"; then
      grep -E '^(--- FAIL|FAIL[[:space:]]|panic:|SIGSEGV)' "$log" | head -n 20 | sed 's/^/    /'
    else
      tail -n 20 "$log" | sed 's/^/    /'
    fi
    FAILED+=("$name")
  fi
}

skip() {
  printf '%-14s skipped: %s\n' "$1" "$2"
  SKIPPED+=("$1")
}

changed_under() {
  echo "$CHANGED" | grep -qE "$1"
}

# ---- Go ----

# go_pkgs prints the import paths of the changed Go packages. A changed non-Go file (testdata,
# embedded assets) counts against the nearest enclosing directory that holds Go files.
go_pkgs() {
  echo "$CHANGED" | grep -E '^(cmd|internal|pkg|supervisor)/' | while read -r f; do
    d=$(dirname "$f")
    while [ "$d" != "." ]; do
      if ls "$d"/*.go >/dev/null 2>&1; then
        echo "$MODULE/$d"
        break
      fi
      d=$(dirname "$d")
    done
  done | sort -u
}

# affected_pkgs prints the packages in this module that are changed, depend on a changed package,
# or whose tests import one. -e keeps packages that fail to load, so a broken one still reaches vet and fails
# there rather than silently dropping out.
affected_pkgs() {
  local changed="$1"
  go list -e -f '{{.ImportPath}} {{join .Deps " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./... |
    GATE_CHANGED="$changed" awk '
      BEGIN { n = split(ENVIRON["GATE_CHANGED"], c, "\n"); for (i = 1; i <= n; i++) if (c[i] != "") want[c[i]] = 1 }
      { for (i = 1; i <= NF; i++) if ($i in want) { print $1; next } }
    '
}

GO_PKGS=""
if [ $FULL -eq 1 ] || changed_under '^go\.(mod|sum)$'; then
  GO_PKGS="./..."
elif CHANGED_PKGS=$(go_pkgs) && [ -n "$CHANGED_PKGS" ]; then
  GO_PKGS=$(affected_pkgs "$CHANGED_PKGS")
fi

if [ -n "$GO_PKGS" ]; then
  # shellcheck disable=SC2086 # GO_PKGS is a whitespace separated package list
  set -- $GO_PKGS
  [ "$GO_PKGS" = "./..." ] && echo "Go: all packages" || echo "Go: $# affected package(s)"
  run go-vet go vet -lostcancel=false "$@"
  if command -v staticcheck >/dev/null 2>&1; then
    run staticcheck staticcheck "$@"
  else
    skip staticcheck "not installed (go install honnef.co/go/tools/cmd/staticcheck@2026.1); CI will run it"
  fi
  run go-test go test -skip '_e2e$' "$@"
  [ $FULL -eq 1 ] && run go-test-e2e go test -count=1 -run '_e2e$' "$@"
  [ $RACE -eq 1 ] && run go-test-race go test -short -race "$@"
  if command -v govulncheck >/dev/null 2>&1; then
    run govulncheck govulncheck "$@"
  else
    skip govulncheck "not installed (go install golang.org/x/vuln/cmd/govulncheck@latest); CI will run it"
  fi
fi

# ---- Proto ----

# The same triggers as CI's proto-check, except that hand-written helpers in pkg/proto don't count.
if [ $FULL -eq 1 ] || changed_under '^(proto/.*\.proto$|pkg/proto/.*\.pb\.go$|ui/ui-gen/proto/|cmd/tools/(genproto|protoc-gen-router|protoc-gen-wrapper)/|scripts/(gen-proto\.sh|dev-container/install-protoc\.sh)$|go\.(mod|sum)$)'; then
  if command -v protoc >/dev/null 2>&1; then
    gen_snapshot() {
      { git diff HEAD -- pkg/proto ui/ui-gen/proto
        git ls-files -o --exclude-standard -z -- pkg/proto ui/ui-gen/proto | xargs -0 cat 2>/dev/null
      } | shasum
    }
    before=$(gen_snapshot)
    run gen-proto bash scripts/gen-proto.sh
    if [ "$before" != "$(gen_snapshot)" ]; then
      echo "               generated code changed: review and commit pkg/proto and ui/ui-gen/proto"
      FAILED+=("gen-proto-stale")
    fi
  else
    skip gen-proto "protoc not installed (scripts/dev-container/install-protoc.sh); CI proto-check will verify"
  fi
fi

# ---- Rego ----

if [ $FULL -eq 1 ] || changed_under '^pkg/auth/policy/default/'; then
  if command -v opa >/dev/null 2>&1; then
    run rego-test sh -c 'cd pkg/auth/policy/default && opa test -v .'
  else
    skip rego-test "opa not installed; CI will run it"
  fi
fi

# ---- UI ----

if changed_under '^ui/yarn\.lock$' && grep -qE 'nexus\.vanti\.co\.uk/repository/' ui/yarn.lock; then
  echo "ui/yarn.lock   FAIL: contains private nexus URLs; CI rejects these"
  FAILED+=("yarn-lock")
fi

WORKSPACES=$(node -p "require('./ui/package.json').workspaces.join('\n')")
if [ $FULL -eq 1 ] || changed_under '^ui/(package\.json|yarn\.lock)$'; then
  UI_WS="$WORKSPACES"
else
  UI_WS=$(for ws in $WORKSPACES; do changed_under "^ui/$ws/" && echo "$ws"; done)
fi

if [ -n "$UI_WS" ]; then
  # ui/ is a Yarn 1 workspace; fall back to a pinned yarn via npx when none is installed.
  if command -v yarn >/dev/null 2>&1; then YARN="yarn"; else YARN="npx --yes yarn@1.22.22"; fi
  if [ ! -d ui/node_modules ]; then
    echo "ui             FAIL: run '$YARN install --frozen-lockfile' in ui/ first"
    FAILED+=("ui-install")
  else
    # CI installs with a frozen lockfile, which fails when a package.json change isn't reflected in yarn.lock.
    if changed_under '^ui/(.*/)?package\.json$|^ui/yarn\.lock$'; then
      run ui-install sh -c "cd ui && $YARN install --frozen-lockfile"
    fi
    for ws in $UI_WS; do
      run "ui-$ws" sh -c "cd ui && $YARN --cwd '$ws' lint:nofix"
    done
  fi
fi

# ---- Summary ----

echo
if [ ${#PASSED[@]} -eq 0 ] && [ ${#FAILED[@]} -eq 0 ]; then
  echo "Nothing to check for the changes since $(git rev-parse --short "$BASE")."
elif [ ${#FAILED[@]} -eq 0 ]; then
  echo "Gate passed (${#PASSED[@]} ok, ${#SKIPPED[@]} skipped)."
else
  echo "Gate FAILED: ${FAILED[*]}"
  exit 1
fi
