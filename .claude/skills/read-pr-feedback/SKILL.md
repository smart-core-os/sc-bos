---
name: read-pr-feedback
description: Process review feedback on your PR — resolve comments, apply fixes, and reply to reviewers systematically. Use after a reviewer has left comments on a PR you raised (e.g. "read the PR feedback", "address the review comments", "work through the PR review"). Do NOT fire on "raise a PR" (that's /raise-pr) or "review my changes" (that's /code-review).
argument-hint: [pr-number]
---

# Process PR feedback $ARGUMENTS

Systematically work through review comments on your PR, resolving each thread by applying fixes, replying to reviewers, or discussing trade-offs. Repo: `github.com/smart-core-os/sc-bos` (use the `gh` CLI).

## Phase 1 — Detect PR

If no PR number argument given, detect from current branch:

```
gh pr view --json number,title,url
```

If a PR number is given, use it directly. Confirm the PR exists and show the title.

## Phase 2 — Fetch reviews & comments

Fetch all review data:

```
gh api repos/{owner}/{repo}/pulls/<number>/reviews --paginate
gh api repos/{owner}/{repo}/pulls/<number>/comments --paginate
gh api repos/{owner}/{repo}/issues/<number>/comments --paginate
```

Check thread resolution status via GraphQL (uses this skill's bundled query file to avoid permission prompts):

```
gh api graphql -F query=@.claude/skills/read-pr-feedback/graphql/review-threads.graphql -F owner=smart-core-os -F repo=sc-bos -F number=NUMBER
```

The thread `id` (node ID like `PRRT_...`) is used in Phase 6 to resolve threads. The comment `databaseId` is the numeric ID used for `in_reply_to` in REST API replies.

Filter out resolved threads. Note which reviewer authored each comment — this is needed in Phase 6 to determine per-reviewer re-request eligibility.

## Phase 3 — Present summary

Show:

- Number of reviewers with open comments
- Open comment count per reviewer
- Review verdicts (approved, changes requested, commented)

## Phase 4 — Triage comments

Read the code at each comment's location and classify every open comment into one of these categories based on what action is needed:

1. **Already addressed** — the current code already fixes the feedback (e.g., from a previous commit). Just needs a reply confirming.
2. **Observation / no change requested** — the reviewer is flagging context, asking a question, or confirming an intentional choice ("no action needed if intentional", "just flagging for later"). Needs a reply acknowledging the observation; no code change. Do not silently resolve these — a reply closes the loop.
3. **Straightforward fixes** — the reviewer's suggestion is correct and there is no genuine trade-off to decide. This includes but is not limited to typos, naming, imports, nil checks, added validation, error-wrapping tweaks, small refactors, or test coverage. The test is *"is there a real design choice the author should make?"*, not *"is this smaller than one line?"*.
4. **Needs author input** — multiple valid approaches or a trade-off the author should decide. Present options.
5. **Push-back candidates** — the current approach is deliberately chosen and the reviewer's suggestion would regress it or change scope. Flag the concrete reason (not vibes) so the author can decide whether to push back or accept. Pushback is a valid outcome — don't steer away from it when the existing code is right.
6. **Scope expansion** — addressing the comment would require work in areas the PR *did not already touch*: other packages, other drivers, unrelated concerns, or broad refactoring across untouched files. The test is *surface area*, not *size*. If the PR introduced a package, RPC, proto message, or pattern, then adding a sibling method, filling in a missing field, or completing the surface the PR itself defined is **not** scope expansion — it's finishing the work. Default to tackling it in this PR unless the added work genuinely reaches into unrelated code.

Present the triage as a numbered list per category, with each comment showing: reviewer, file:line, and a one-line summary of the feedback.

If all comments fall into categories 1–3 (already addressed, observations, and straightforward fixes), skip confirmation and proceed directly — there are no decisions for the user to make. Otherwise, wait for the user to confirm or adjust the categorisation before proceeding.

## Phase 5 — Process categories

Work through the categories in the order listed above. For each category, ask the user for a batch decision before proceeding:

- **Already addressed** — reply to these confirming they're fixed. No user confirmation needed.
- **Observation / no change requested** — reply acknowledging. If the observation flagged an intentional choice, state the rationale in one line. No user confirmation needed.
- **Straightforward fixes** — apply all changes and show a summary of what was done. No user confirmation needed.
- **Needs author input** — present each comment (or sub-group of related comments) with the options and trade-offs. Wait for the user's choice, then apply.
- **Push-back candidates** — for each, state the concrete reason the current approach is right (e.g. "scope is intentional — tracked in SCB-XXX", "the previous semantics were the bug"). Ask whether to push back or accept.
- **Scope expansion** — for each, state which untouched areas the work reaches into and why that makes it out-of-scope. If the work stays inside the surface the PR introduced or modified, reclassify as *Straightforward fixes* or *Needs author input* instead of deferring. Ask whether to tackle it in this PR, defer with a follow-up ticket, or push back.

Skip empty categories silently. If the user declines a whole category (e.g., "skip the scope expansion ones"), mark those comments as deferred.

### Applying fixes

- If a fix touches `.proto` files, regenerate the Go/JS afterwards (`bash scripts/gen-proto.sh`) and include the regenerated output.
- Keep the change building as you go (`go build ./...`); UI changes should still pass `yarn --cwd ops lint:nofix` (run from `ui/`).

### Reply to threads

After each category is processed, reply to the threads in that category.

**For code-level review comments** (comments attached to a file/line in the diff), reply inline using `in_reply_to` so the response appears in the same thread:

```
gh api repos/{owner}/{repo}/pulls/<number>/comments -f body="Reply text" -F in_reply_to=COMMENT_ID
```

The `COMMENT_ID` is the `id` of the review comment being replied to (from the `/pulls/<number>/comments` endpoint, not the `/issues/<number>/comments` endpoint).

**For general PR-level comments** (comments on the PR description, not attached to code), reply as an issue comment:

```
gh api repos/{owner}/{repo}/issues/<number>/comments -f body="Reply text"
```

Keep replies concise: what was done (with commit ref if applicable), or why the current approach is preferred.

## Phase 5b — Integration pass

Before wrapping up, re-read each file you changed **cold** — the whole region, not just the changed lines — and integrate the fixes so they read as if always part of the code, not bolted on in response to comments. The diff and your thread replies are the trace; the code itself must not narrate that it was revised (no "changed to address review" comments, no awkward seams around the edit). For a substantial or prose-heavy round a self-read won't catch this reliably — you carry the same focus that produced the bolt-on — so run the check through a fresh context (`/code-review`, or a `code-reviewer` subagent) instead.

## Phase 6 — Wrap up

Summarise:

- Comments addressed with code changes
- Comments replied to without code changes
- Comments deferred or skipped

If any code changes were made, re-run the relevant pre-flight gate (mirrors CI):

```
go build ./... && go vet -lostcancel=false ./... && go test ./...
# for UI changes (run from ui/):
(cd ui && yarn --cwd ops lint:nofix)
```

Report results. If everything passes, offer to commit the changes. Follow the repo's commit convention — `<area>: <summary>` with the ticket in parens when applicable, e.g.:

```
<area>: address review feedback (SCB-NNNN)
```

List the specific items addressed in the commit body.

### Refresh the PR description

Before resolving threads, check whether the changes invalidate anything in the PR description. The diff and inline replies tell most of the story, but reviewers read the description to find context that isn't in the code — and a stale description sends them down wrong paths.

Re-fetch the description with `gh pr view <number> --json body --jq .body` and scan for any of these signals:

- **New manual setup steps surfaced by feedback.** A "you also need to..." piece of config that lives outside the diff: a node config key, an env var, a driver setting, a one-time command. If the reviewer flagged it and the fix accepts the addition, the description needs the step.
- **Architecture or scope claims now wrong.** The description's "what changes" / "out of scope" notes sometimes describe choices the review revised. A defaulting change, a moved boundary, a renamed file or extracted helper: surface it.
- **Verification line invalidated.** A step that references a behaviour the fix removed, or a flow that now needs additional verification.
- **Sequencing or blocker notes the author wants surfaced.** If a fix turns out to depend on another PR landing first, add a callout at the top of the description.

Update the description with `gh pr edit <number> --body-file <file>`. Bundle changes into one edit rather than several. If you're unsure whether a tweak rises to "needs updating", err on the side of asking the user — small adds are fine, but rewriting a section the author wrote deliberately is a different decision.

If nothing in the description needs to change, say so explicitly and move on. Don't edit it just to feel busy.

### Resolve addressed threads

After committing (or if no code changes were needed) and the PR description is current, resolve threads that have been fully addressed:

```
gh api graphql -F query=@.claude/skills/read-pr-feedback/graphql/resolve-thread.graphql -f threadId=THREAD_NODE_ID
```

Thread node IDs come from the GraphQL query in Phase 2. Only resolve threads where:

- A code change was made that addresses the feedback, or
- The reply explains why no change is needed and the user approved the response

Do **not** resolve threads that were deferred or skipped.

### Re-request reviews

For each reviewer, check whether **all** of their open comments were addressed (by code change or approved reply). Re-request review from a reviewer when:

- **All** of their comments were addressed — none deferred or skipped
- Their most recent verdict was **"changes requested"** or **"commented"**

Treat a `"commented"` verdict the same as `"changes requested"` — if the PR needs an explicit approval to merge, a bare comment blocks it just as much, so re-request the reviewer once their feedback is addressed.

Do not re-request review when:

- Any of the reviewer's threads were deferred or skipped
- The reviewer already **approved** and the changes made were trivial — e.g. typos, formatting, or directly applying the reviewer's own one-line suggestion. If the changes were substantive (even if the reviewer approved), re-request so they can re-verify.

```
gh pr edit <number> --add-reviewer <login1>,<login2>
```

This repo has no CODEOWNERS auto-request, so the reviewers are whoever was requested when the PR was raised (see `/raise-pr` Phase 7). If it's unclear who should re-review, ask the user rather than guessing.
