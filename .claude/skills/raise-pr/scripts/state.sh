#!/usr/bin/env bash
# state.sh — emit a structured snapshot of the branch state for /raise-pr.
# Usage: scripts/state.sh [base-branch]   default base: main
#
# Output is key=value lines (one per metric) followed by a `---` separator and
# `uncommitted_files:` / `committed_files:` blocks with two-space indented paths.
# The skill body reads this output instead of issuing several ad-hoc git calls.
set -eo pipefail

BASE="${1:-main}"
BRANCH="$(git branch --show-current)"

# Fetch makes ahead/behind accurate; if offline, fall through with stale data
# and surface that via base_fetch so the caller can decide whether to retry.
if git fetch --quiet origin "$BASE" 2>/dev/null; then
  BASE_FETCH="fresh"
else
  BASE_FETCH="stale"
fi

AHEAD="$(git rev-list --count "origin/$BASE..HEAD")"
BEHIND="$(git rev-list --count "HEAD..origin/$BASE")"

# Uncommitted: tracked changes (staged + unstaged) from porcelain — `ls-files
# --modified` alone misses staged additions — plus untracked files from
# `ls-files --others` (which doesn't directory-collapse like porcelain does).
TRACKED_CHANGES="$(git status --porcelain --untracked-files=no | sed -E 's/^.{2} //' | sed -E 's/.* -> //')"
UNTRACKED="$(git ls-files --others --exclude-standard)"
UNCOMMITTED_FILES="$(printf '%s\n%s\n' "$TRACKED_CHANGES" "$UNTRACKED" | grep -v '^$' | sort -u || true)"
UNCOMMITTED_COUNT="$(printf '%s\n' "$UNCOMMITTED_FILES" | grep -c . || true)"

# Triple-dot: changes from merge-base(origin/BASE, HEAD) to HEAD — i.e. only
# what this branch added. Double-dot would include changes that landed on BASE.
COMMITTED_FILES="$(git diff --name-only "origin/$BASE...HEAD")"

ALL_FILES="$(printf '%s\n%s\n' "$COMMITTED_FILES" "$UNCOMMITTED_FILES" | grep -v '^$' | sort -u || true)"

docs_only=true
has_code=false
has_proto=false
while IFS= read -r f; do
  [ -z "$f" ] && continue
  case "$f" in
    # vendored / installed / build-output dirs are never our change — skip so a
    # stray file inside them (e.g. an untracked node_modules) can't skew the class.
    */node_modules/*|node_modules/*|*/vendor/*|vendor/*|*/dist/*|dist/*) continue ;;
    *.md) ;;
    .github/*) ;;
    .claude/*) ;;
    .gitignore) ;;
    *.proto)
      has_code=true; has_proto=true; docs_only=false ;;
    *.go|*.rego|*.ts|*.tsx|*.vue|*.js|*.jsx|*.mjs|*.cjs)
      has_code=true; docs_only=false ;;
    *)
      docs_only=false ;;
  esac
done <<< "$ALL_FILES"

if $docs_only; then
  CHANGE_CLASS="docs-only"
elif $has_code; then
  CHANGE_CLASS="code"
else
  CHANGE_CLASS="mixed"
fi

RECS=""
REASONS=""
add_rec() {
  RECS="${RECS}${RECS:+ }$1"
  REASONS="${REASONS}${REASONS:+; }$2"
}
plural() { [ "$1" -eq 1 ] && echo "" || echo "s"; }

[ "$BRANCH" = "$BASE" ] && add_rec "on-base-branch-stop" "current branch is the base branch"
[ "$AHEAD" -eq 0 ] && [ "$UNCOMMITTED_COUNT" -eq 0 ] && add_rec "nothing-to-raise-stop" "no commits ahead and no uncommitted work"
[ "$BEHIND" -gt 0 ] && add_rec "rebase" "behind $BASE by $BEHIND commit$(plural "$BEHIND")"
[ "$CHANGE_CLASS" = "docs-only" ] && add_rec "skip-build" "diff is docs/config only"
$has_proto && add_rec "proto-regen" "diff touches .proto files — regenerate before raising"

if [ -z "$RECS" ]; then
  RECS="none"
  REASONS="no action needed"
fi

echo "branch=$BRANCH"
echo "base=$BASE"
echo "base_fetch=$BASE_FETCH"
echo "ahead=$AHEAD"
echo "behind=$BEHIND"
echo "uncommitted_count=$UNCOMMITTED_COUNT"
echo "change_class=$CHANGE_CLASS"
echo "recommend=$RECS  // $REASONS"
echo "---"
echo "uncommitted_files:"
if [ -n "$UNCOMMITTED_FILES" ]; then
  printf '%s\n' "$UNCOMMITTED_FILES" | sed 's/^/  /'
fi
echo "committed_files:"
if [ -n "$COMMITTED_FILES" ]; then
  printf '%s\n' "$COMMITTED_FILES" | sed 's/^/  /'
fi
