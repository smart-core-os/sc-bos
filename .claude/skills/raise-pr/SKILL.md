---
name: raise-pr
description: Raise a pull request from the current branch — assesses whether the change has been validated to the depth its size warrants, runs the build/test/lint gate that mirrors CI, commits, pushes, opens the PR as draft, watches CI to green, then marks it ready and prompts you for who to request review from (this repo has no CODEOWNERS auto-request). Use when the user signals end-of-task intent ("raise a PR", "open a PR", "ship this", "create the PR", "let's PR this"). Do NOT fire on "review my changes" (that's /code-review) or post-PR feedback (that's /read-pr-feedback).
argument-hint: [base-branch]
---

# Raise PR

Take a branch from "implementation done" to "PR open, CI green, review requested". Default base branch is `main` if no argument given. Repo: `github.com/smart-core-os/sc-bos` (use the `gh` CLI).

## Goal

A pull request is the author saying: *"I want this change to land in the product in exactly this shape, and I'm confident enough in it to ask for a second pair of eyes."* The review process is a sanity check on that commitment, not a substitute for it. Quality and confidence are the author's responsibility before raising.

The skill exists to make the commitment honest. Confidence is earned by checking the change outcome against several criteria, weighted by the change's size and impact:

- **Mechanical** — lint, vet, build. Cheap, deterministic, run as a pre-flight.
- **Correctness** — unit/integration tests, binary/API exercise, proto regenerated. Should have happened *during* the work.
- **Judgement** — design alignment, UI assessment, review of some form. Proportionate to scope.

The skill verifies these signals; it does not perform the underlying work at PR time. Missing evidence means the change is not yet ready to be committed to — surface the gap rather than paper over it by doing the work last-minute.

The skill makes no assumption about *what* upstream workflow was used. It does not require `/grab-ticket`, `/code-review`, or any specific predecessor; it assesses what's actually in the session and the diff.

## Efficiency rules

- **Single `gh pr create` call** — never fail-then-retry. Resolve title, body, base, and remote tracking *before* invoking. Use `--body-file` (not `--body`) for prose bodies.
- **Don't ask for confirmation on a clean run** of the mechanical gates. The one interactive step that always happens is choosing reviewers in Phase 7 — that's the point of this skill's variant.
- **Use parallel tool calls** for independent reads and independent checks (see Phase 2).
- **Never hot-loop CI checks.** Either stream events via `Monitor` on `gh pr checks <pr> --watch`, or wait at least 30 seconds between polls. CI runs typically take a minute or longer to settle.

## Sanity check

Run `bash .claude/skills/raise-pr/scripts/state.sh [base-branch]` (default base: `main`) to get a structured snapshot of the branch in one call. The script fetches the base and emits key=value lines:

- `branch`, `base`, `ahead`, `behind`, `uncommitted_count`
- `base_fetch` — `fresh` if the fetch of `origin/<base>` succeeded; `stale` if it failed (likely offline). Stale numbers may be wrong; retry or proceed cautiously.
- `change_class` — `docs-only` | `code` | `mixed` (heuristic on file paths/extensions; `.go/.proto/.rego/.vue/.js/.ts` count as code)
- `recommend` — space-separated flags, optionally followed by ` // ` and a human-readable rationale. Flags: `on-base-branch-stop`, `nothing-to-raise-stop`, `rebase`, `skip-build`, `proto-regen`. Ignore everything from ` // ` onward if parsing.
- Lists of uncommitted and committed-ahead files

Act on the flags before doing anything else:

- `on-base-branch-stop` — the branch is the base. Stop and ask which branch should hold the work.
- `nothing-to-raise-stop` — no commits ahead **and** no uncommitted work. Nothing to raise. Stop.
- `rebase` — branch is behind base. Rebase onto `origin/<base>` now, before Phase 1, so confidence assessment and pre-flight operate on the final state (`git rebase origin/<base>`; stash first if there's uncommitted work). Resolve conflicts, then continue.
- `skip-build` — pre-flight (Phase 2) can skip the Go build/test chain; see that phase for the fast path.
- `proto-regen` — the diff touches `.proto` files. Confirm the generated code is up to date (`bash scripts/gen-proto.sh`) and that the regenerated output is committed, before proceeding.

## Phase 1 — Confidence assessment

Read the session and the diff. Judge whether the evidence supports committing to this change in this shape. The depth required scales with the size and impact of the change.

### Signals to assess

**Session continuity** — Scan the conversation for items raised by the user that were parked, deferred, or forgotten. Phrases like "we'll do X later", "also handle Y", "remind me about Z", or direct asks that drifted out of focus. The change should not leave session-level expectations unmet without an explicit decision to defer them.

**Tests** — Does the diff include tests covering the new behaviour? For a new package/service, expect a `_test.go`. For a bug fix, expect a regression test that would have caught the bug. For a tiny refactor, existing tests may suffice. The signal: would a reviewer reasonably expect tests here, and are they present?

**Binary / API exercise** — If the diff adds or changes an RPC, driver, or command, look for evidence in the session that it was actually run — a `go test` against the new path, the relevant `cmd/` binary exercised, or a gRPC call made.

**Proto regeneration** — If `.proto` files changed, the generated Go (`pkg/proto/...`, `pkg/gen`) and JS (`ui/ui-gen/proto`) must be regenerated and committed. Confirm the generated files are in the diff and consistent with the proto change.

**Frontend functional flow** — If the diff changes ops-UI behaviour, look for evidence the affected flow was driven end-to-end in the session (dev server, navigate, exercise, confirm outcome).

**UI visual signoff** — How much pixel surface did the change move?

- *Small* — a CSS tweak, a copy change, a fix targeted at a specific visual bug. Look for evidence the change was visually confirmed in the session (a screenshot or a browser navigation that loaded the changed surface after the edit). If it went in blind, that's a gap.
- *Non-trivial* — new views, restyled components, layout shifts, new visual states. Look for evidence the user has engaged with the visual in this session — a shared screenshot, feedback given and addressed, or direct sign-off. If the user hasn't seen it, ask for a walkthrough before raising.

**Design alignment** — If a design doc under `docs/` describes the change, the diff should match it on structure, RPCs, config, fields.

**Review** — Proportionate to size. A two-line fix needs no formal review beyond its own clarity. A new feature should have evidence of *some* review pass: `/code-review`, direct user feedback that was engaged with, or substantive back-and-forth in the session. Absent any review trace on a substantial change, that's a confidence gap.

### On a gap

Stop. Name the gap concretely — which signal is weak, what specifically is missing, what would close it. Pause for the user.

Example:

> The diff adds a new `MeterInfo.DescribeMeterReading` handler and wires it into the mock driver. The session shows a `go build` but no test covering the new handler, and no review trace for a change of this size. Before raising: add the test, or confirm you've reviewed the handler separately.

If the user supplies plausible out-of-session evidence ("I tested it from another terminal", "I reviewed it in another session"), take it at face value and continue. The gate catches genuinely-missed validation, not ceremony. Do not volunteer to *do* the validation yourself — that defeats the gate.

## Phase 2 — Pre-flight gate

Mirror what CI runs so nothing is discovered only after pushing.

If sanity check returned `skip-build` (docs/config-only diff), run only the relevant linters and skip ahead to Phase 3.

### Go (when the diff touches `go.mod`, `cmd/`, `internal/`, or `pkg/`)

Run in parallel (separate Bash calls in one message):

- `go build ./...`
- `go vet -lostcancel=false ./...`
- `go test ./...` (CI additionally runs `-short -race ./...` and the `_e2e$` split; run the race pass too for concurrency-touching changes)

CI also runs `staticcheck ./...` (`honnef.co/go/tools/cmd/staticcheck@2026.1`). Run it if available locally; if not installed, note that staticcheck will run in CI and watch it in Phase 6.

### Ops UI (when the diff touches `ui/**`)

- Run from `ui/` (the yarn workspace root, as CI does): `yarn --cwd ops lint:nofix` — exactly what CI runs per-workspace. (`yarn --cwd ui/ops …` from the repo root does **not** work — the repo root isn't a workspace, so the hoisted binaries don't resolve.)
- For a build-affecting change, also `yarn --cwd ops build` (from `ui/`).

Note: the `lint` script auto-fixes; `lint:nofix` only checks. If you run `lint` to auto-fix, review the resulting `git status` and include the fixes in the commit, then confirm `lint:nofix` passes.

All gates must succeed. If anything fails, fix it locally and rerun — do not push a broken branch and let CI catch it.

## Phase 3 — Commit

Iteration tweaks may have been held uncommitted during review loops. Roll them into one meaningful commit, or a small number of meaningful commits — not per-file noise.

**Title format:** this repo uses `<area>: <summary>` with the ticket ID in parentheses at the end. Real examples from the history:

```
ui/ops: escape CSV special characters in downloads (SCB-1375)
node/alltraits: drop status trait from service registry
auto/history: drop status trait recorder
```

- `<area>` is the package/path touched (`ui/ops`, `node/alltraits`, `pkg/auto/bms`, `driver/mock`, …).
- Append ` (SCB-NNNN)` when the work tracks a ticket. Unlike some repos, sc-bos **keeps the ticket ID in the subject line** — don't strip it.
- Imperative mood, lower-case area, no trailing full stop.

Plain `git commit` — never `--no-verify`.

## Phase 4 — Push

First push to a new remote branch:

```
git push -u origin <branch>
```

Subsequent push (history rewritten via amend, rebase, fixup):

```
git push --force-with-lease --force-if-includes
```

Never plain `--force` or bare `--force-with-lease` (without `--force-if-includes`) — both can silently overwrite upstream changes.

If push is rejected for "fetch first" — and sanity check didn't already prompt a rebase — rebase onto `origin/<base>` and push again.

## Phase 5 — Create the PR as draft

Resolve everything before invoking `gh pr create`:

1. **Title** — already decided in Phase 3 (`area: summary (SCB-NNNN)`).
2. **Body** — prose explaining *why*, not what. The diff explains what. End with a verification line stating how the change was exercised. If the work tracks a ticket, end with a YouTrack command footer `#SCB-NNNN State Done` — this both links the ticket and, when the PR merges, transitions it to Done via YouTrack's GitHub VCS integration (the merge is the resolving event; a bare `SCB-NNNN` only links). If the integration isn't enabled the footer is harmless text and the ticket stays where `/grab-ticket` left it (To Verify), to be closed manually. Use the project's resolved-state command if it isn't `State Done`.
3. **Base** — `$ARGUMENTS` or `main`.
4. **Draft mode** — open as draft. CI still runs on drafts; mergeable status is computed regardless. Opening as draft keeps reviewers out of it until CI is green and the PR is mergeable — the flip and the reviewer request happen together in Phase 7.

Write the body to a temp file, then create the PR:

```
mkdir -p .tmp
# write the body to .tmp/pr-body.md  (.tmp is gitignored; create it if missing)
gh pr create --draft --base <base> --title "<title>" --body-file .tmp/pr-body.md
```

Capture the PR number and URL from the output.

### Body shape

```
<one short paragraph: why this change exists — the trigger, the user pain. Not "this PR adds X">

<optional second paragraph: non-obvious design choice, trade-off, or scope note>

Verified by <go test ./pkg/... / exercised cmd/bos with <config> / browser walk-through of <flow> / proto regenerated and built / ...>.

#SCB-NNNN State Done
```

Keep it short. Reviewers can read the diff. The body's job is to communicate intent, confirm the change was witnessed, and link the ticket.

## Phase 6 — Watch CI and merge state

Watch the PR through to green without polling tightly. Either:

- Stream events via `Monitor` on `gh pr checks <pr> --watch` (event-driven, no polling), or
- Poll `gh pr checks <pr>` and `gh pr view <pr> --json mergeable,mergeStateStatus`, waiting at least 30 seconds between polls.

sc-bos CI includes Go Test, Go Lint (vet + staticcheck), Go Security, Rego Test, and UI Lint — each only runs when its paths are touched, so expect a subset relevant to your diff.

### On `mergeable: CONFLICTING`

Rebase onto `origin/<base>`, resolve, then push (`--force-with-lease --force-if-includes`) and resume watching.

### On red CI

Identify the failing run, then pull its failed-step logs:

```
gh run list --branch <branch> --limit 5 --json databaseId,name,conclusion,headSha
gh run view <run-id> --log-failed
```

Classify the failure:

- **Mechanical** — lint/format, vet/staticcheck finding, a test broken by a refactor miss, a missing import. The fix is local and obvious.
- **Behavioural** — a test failed because the *behaviour* is wrong, or the failure is in code you don't recognise. Stop and escalate: failing job name, log snippet, your hypothesis. Don't guess your way through a real bug.

When CI fails and a fix is needed, the fix is in-session work — witness it the same way you'd witness any change: re-run the failing test locally, exercise the affected path. "It compiles" is mechanical confidence, not enough for a behavioural failure. Apply the matching validation, re-run the Phase 2 pre-flight, then push.

### On green CI and no conflicts

Proceed to Phase 7.

## Phase 7 — Mark ready and request review

This repo has **no CODEOWNERS auto-request**, so reviewers must be named explicitly. Do **not** auto-assign anyone or auto-invoke a review skill — **ask the user who should review.**

1. Flip the PR from draft to ready:

   ```
   gh pr ready <pr>
   ```

2. **Prompt the user for the reviewer(s).** Ask who they'd like to request review from. To make the choice easy, offer a short list of candidates from recent history rather than a blank prompt:

   ```
   git log -n 30 --format='%an' -- <top-level dirs the diff touched> | sort | uniq -c | sort -rn
   gh api repos/smart-core-os/sc-bos/collaborators --jq '.[].login'   # if permitted
   ```

   Present a few likely names (recent authors in the touched areas) and let the user pick one or more — or type a different GitHub login. Use `AskUserQuestion` if the candidate list maps cleanly onto options; otherwise a plain free-text ask is fine. Don't proceed by guessing.

3. Once the user names the reviewer(s), request them by GitHub login:

   ```
   gh pr edit <pr> --add-reviewer <login1>,<login2>
   ```

Then print:

- PR URL
- Reviewer(s) requested

Suggest the next move:

> CI is green and `<reviewer(s)>` have been requested. When their review lands, run `/read-pr-feedback`.

Do **not** merge the PR. Merging is the user's decision (or a separate step).

## Tone

You are taking the change from "implementation done" to "PR open, validated, CI green, review requested". Be decisive on mechanical fixes; surface confidence gaps honestly rather than papering over them; treat reviewer time as a cost that earned confidence is the price of. The one deliberate pause is choosing reviewers — that's a human call, so ask. When CI fails, the fix is held to the same confidence bar as the original change — a red CI doesn't lower the standard, it just triggers another round of it.
