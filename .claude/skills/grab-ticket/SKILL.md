---
name: grab-ticket
description: Work a YouTrack ticket end-to-end — fetch it, verify the situation against the actual codebase (don't trust the ticket body), present options with trade-offs, then build and validate the fix. TRIGGER whenever the user opens a prompt with a SCB- ticket reference and an investigate / look-at / work-on / pick-up / grab / read-and-fix intent — e.g. "look at SCB-123", "work SCB-456", "pick up SCB-789", "grab SCB-NNN", "read SCB-123 and tell me how we'd fix it", "investigate SCB-NNN", "what's going on with SCB-XYZ". Covers the full loop from triage through PR handoff and composes with /verify, /code-review, and /raise-pr.
argument-hint: <SCB-NNN>
---

# Grab YouTrack ticket: $ARGUMENTS

Work a YouTrack ticket end-to-end — fetch it, verify the situation against the actual codebase, agree the approach with the user, then build and validate the fix.

The skill's core assumption: **tickets are a starting point, not a spec.** The body may be wrong, the proposed solution may be wrong, the scope may be wider than enumerated. Trust the codebase, not the ticket. SCB tickets are "things to pick up" — the body doesn't track product state (code and PRs are the source of truth for that), but the **assignee** and **workflow state** do matter for team coordination, and this skill keeps those in sync as it works.

The YouTrack project is **SCB** (Smart Core BOS) at `https://vanti.youtrack.cloud`. Access it through the `mcp__youtrack__*` MCP tools.

## Phase 1 — Fetch the ticket

### Pre-flight — branch check

Before doing anything else, check the current branch:

```
git rev-parse --abbrev-ref HEAD
```

If on `main`, **stop immediately**. Investigation, categorisation, and the option-discussion conversation in Phases 2–4 should happen in the session that's going to do the implementation, on the branch that will hold the work — otherwise the work is wasted or has to be manually transferred.

Tell the user: create a feature branch for this ticket (this repo's convention is `feat/<short-slug>` or `fix/<short-slug>`; some contributors run one worktree per ticket), then re-invoke the skill there. Don't soldier on with a read-only triage pass on `main` — the assign + In-Progress claim later in this phase only makes sense once we're committed to working the ticket from this session.

If on a feature branch (anything other than `main`), proceed. The skill assumes the branch was prepared for this ticket, but doesn't enforce a particular naming convention.

### Determine the ticket ID

- If `$ARGUMENTS` matches `SCB-\d+`, use that.
- Otherwise check the current branch name (`git rev-parse --abbrev-ref HEAD`) for an embedded ticket ID like `feat/SCB-123-foo`.
- If still unknown, ask the user.

Fetch the ticket and its context in parallel:

- `mcp__youtrack__get_issue` — title, body, type, status, reporter, assignee
- `mcp__youtrack__get_issue_comments` — prior triage and discussion
- Linked issues from the issue payload (`links` field, if any) — duplicates, parents, follow-ups worth fetching too

Surface a short summary back to the user (one paragraph): what the ticket asks for, who reported it, current state, and anything notable in the comments (e.g. "Alice already debugged this and pointed at package X"). **Don't yet say what you think we should do** — that comes after Phase 2. The user can interject if the fetched ticket isn't the one they meant; otherwise proceed straight to claiming it.

### Claim the ticket

Right after the summary, take ownership in YouTrack — deciding what to do with a ticket already counts as working on it, so the state should reflect that immediately:

- **Assign to the current user** via `mcp__youtrack__change_issue_assignee`. Resolve the current user with `mcp__youtrack__get_current_user` if you don't already know their login. Note: the YouTrack MCP can set or change an assignee but cannot clear the field — reassigning to a real person is fine, but unassigning requires the YouTrack UI.
- **Transition to "In Progress"** via `mcp__youtrack__update_issue`, unless the ticket is already at or past In Progress (don't backstep).

State name discovery: if you don't already know the project's workflow values, call `mcp__youtrack__get_issue_fields_schema` once to learn the valid `State` options for SCB, then pick the closest match to "In Progress" (and later "To Verify"). For reference, SCB's states are: Submitted, To Do, In Progress, Can't Reproduce, Duplicate, Done, Won't fix, Deployed, To Verify, Blocked.

**Edge cases that should stop the skill here**:

- Ticket is already **Done / Deployed / Won't fix / Duplicate** — don't reopen it. Surface the closed state loudly and ask the user whether they actually want to work it (likely the wrong ticket).
- Ticket is **already assigned to someone else** — don't steal it. Show who has it and ask the user how to proceed.
- Ticket is an **epic / placeholder / meta** — the workflow assumes a single workable unit. Flag it and ask which child issue to actually pick up.

## Phase 2 — Verify against the codebase

Treat the ticket body and any proposed solution in it as untrusted. For every factual claim the ticket makes — "when X happens, Y is shown", "the API returns Z when called with W", "feature F is missing" — find the corresponding code path and confirm or refute the claim. Read the code, don't theorise about it.

Concretely:

- **Bugs.** Locate the code path, reason through it, and where practical **reproduce the symptom locally**. For Go, write or run a focused test (`go test -run <name> ./pkg/...`) or exercise the relevant binary under `cmd/`. For the ops UI, run the dev server (`yarn dev` in `ui/ops`) and drive the affected screen. Confirm the bug actually exists on the current branch *before* discussing fixes — a fix proposed on a theory often turns out to be aimed at the wrong root cause and wastes a review round.
- **Missing feature.** Find where it would live and what scaffolding is already in place. Note any partial work (half-built RPC, proto message with no server impl, config field with no UI, etc.) — these change the size of the work materially.
- **Refactor / cleanup.** Read the cited code and trace its callers; understand what depends on it before proposing changes.

Watch for these specific outcomes, because they change the next step:

- **Already fixed.** Behaviour doesn't reproduce. Maybe it was fixed in a later commit, maybe the reporter's environment was stale. Skip to Phase 3 with category *Invalid*.
- **Ticket says one thing, code does another.** Ticket asserts X is the bug; code shows Y. Don't silently fix Y under the heading of X — name the divergence so the user can decide what to do with it.
- **Scope wider than enumerated.** A bug class with multiple sites; a missing field that breaks more than the single example the reporter gave. Default to fixing the **full class**, not just the named case — leaving siblings broken usually just produces a duplicate ticket. But surface it as a choice; sometimes there's a reason to stay narrow (urgency, blast radius, a separate piece of work already planned for the rest).
- **Multiple plausible root causes.** When two code paths could each produce the reported symptom, instrument or read enough to know which one — don't pick the more convenient candidate.

### When verification stalls — ask the user first

If the repro fails, the symptom doesn't appear, or the ticket's wording leaves you genuinely unsure what behaviour to test, **ask the user before declaring "needs more info"**. They're usually closer to the missing context than either the ticket or the codebase: a recent change on another branch, an env flag the test data needs, a related ticket, a sibling feature that interacts, or just a translation of what the reporter probably meant.

Concrete things to ask for when stuck:

- "I can't reproduce on `main` — do you have a branch / env where it shows up?"
- "The ticket says X but I see code Y — which behaviour are we treating as the bug?"
- "I'm seeing two code paths that could produce this symptom — does one of them ring a bell?"
- "What was the reporter's setup (which site, which driver, which node)?"

Continue Phase 2 with whatever context the user provides. Only escalate to the **Needs more info** category in Phase 3 if the user can't unstick the repro either — at that point the question genuinely belongs with the reporter, and drafting a comment back is the right move.

### Report findings

Once verification settles (with or without help from the user), report findings with a short summary before moving to Phase 3. Frame it as *what you found*, not *what you'd do* — Phase 3 is for the recommendation.

If verification surfaces an issue that's a strict superset of the ticket (a class-wide bug, a missing layer of a feature), say so explicitly. The user expects to be told.

## Phase 3 — Categorise & engage

Based on Phase 2, classify the situation into one of these and tell the user which one fits, with one-sentence reasoning:

| Category | What it means | Next step |
|---|---|---|
| **Invalid / already fixed / duplicate** | Cannot reproduce; fixed in a later commit; covered by another ticket | Skill ends. Optionally post a closing comment (Phase 8). |
| **Needs more info** | Reporter's description isn't enough, codebase doesn't disambiguate, **and** the user couldn't fill the gap when asked in Phase 2 | Skill ends. Draft a "what would help" comment back to the reporter. |
| **Workable** | A concrete change we can plan and build — bug fix, refactor, new RPC, new config field, new driver capability, even a sizeable addition to an existing area. Most tickets land here. | Continue to Phase 4. |
| **New product domain** | The ticket asks for a whole new product concept the codebase doesn't already model (a new trait, a new subsystem, a data model invented from scratch). The bar is "we need to invent terminology and a data model", not "this is a bigger change than usual". | Recommend stepping back into a design/plan pass first (write a short design doc under `docs/design/`, or use plan mode) and stop. |

Most non-trivial work is still **Workable** — we don't write a design doc just because the change is medium-sized, touches several packages, or crosses the Go/UI boundary. The design gate is *new domain*, not *non-trivial*. If you find yourself reaching for a design doc, articulate which new domain concept the ticket introduces before recommending it; if you can't name one, the answer is Workable.

Scope decisions (narrow vs full-class) were already discussed in Phase 2 — by Phase 3 you're categorising the **agreed scope**, not re-opening the conversation. A ticket where the user picked a wider scope is still Workable; broader doesn't mean design-shaped.

For categories that end the skill, summarise clearly so the user can act on it (close the ticket themselves, ping the reporter, start a design pass). **For an ended skill, do not update ticket status, fields, or assignee beyond what Phase 1 already did** — leave further changes to the user.

## Phase 4 — Propose options & decide

Present **1–3 viable approaches** with concrete trade-offs: effort, risk, regression surface, reach. Don't jump straight to one — the user's call. A worked option includes:

- What you'd change (which files / which layers — Go packages, proto, ops UI)
- What the user-facing outcome is
- The notable trade-off vs. the alternatives (perf, scope creep, blast radius, etc.)

Use `AskUserQuestion` when the options map cleanly onto multi-choice. When the choice is interlinked with other decisions, free-text discussion is fine. Ask one focused question at a time rather than dropping a numbered wall of questions — it's faster to converge that way.

After the user picks, **mirror the chosen plan back** as 2–3 bullet points before writing code.

The chosen option will almost always have follow-up details that need pinning down (naming, config defaults, edge cases, failure modes). Surface those one at a time — the same interview rule applies. Don't dump them as a numbered list of "open questions" and expect the user to reply with a numbered block; that's slower and easier to lose track of. Pick the most consequential open question, ask it on its own, apply the answer, then ask the next. If a couple of small details cluster naturally, batching those two is fine — but the default is one at a time.

## Phase 5 — Implement

Branch state was already vetted in the Phase 1 pre-flight, so proceed directly to implementation. Follow the conventions already present in the code you're touching (match the surrounding package's style, error handling, and test patterns).

- **Proto changes.** After editing any `.proto`, regenerate the Go/JS code with `bash scripts/gen-proto.sh` (it runs `go run ./cmd/tools/genproto`), and commit the regenerated output alongside the proto.
- **Go.** Keep it building as you go: `go build ./...`. Add/adjust tests next to the code.
- **Ops UI.** Lives under `ui/ops` (Vue 3 + Vuetify, Vite, yarn workspaces rooted at `ui/`). Lint from `ui/` (the workspace root, as CI does): `yarn --cwd ops lint`.

If the work turns out to be materially different from the chosen option (e.g. you discover a needed proto change mid-edit, or the change implies a new trait/domain that wasn't in the plan), stop and re-engage the user — don't quietly expand the scope. Bigger-than-expected is usually still Workable; only step out to a design pass if what you discovered genuinely crosses into a new product domain.

## Phase 6 — Validate end-to-end

The ticket symptom should be **gone**, and **nothing adjacent should have regressed**. A green build alone is **not** validation — the change needs to be observed working. Consider driving this with the `/verify` skill.

- **Same reproduction as Phase 2.** If you reproduced the bug via a Go test, re-run it. If via the ops UI, re-walk the same screen. The smoke must hit the same surface the ticket complained about.
- **Adjacent surface.** Mirror what CI will run (see `/raise-pr` for the full gate): `go build ./...`, `go vet -lostcancel=false ./...`, `go test ./...`, and for UI changes `yarn --cwd ops lint:nofix` run from `ui/`. If the fix touched a shared helper, smoke-test the other callers. If it touched UI, click around related screens — visual regressions don't show up in unit tests, so observe the change rendered before claiming success.
- **Driver / integration work.** Where the change touches a driver or external integration, exercise it against a mock or a real device as available, and say which you used.

If you genuinely can't validate (no environment, missing hardware), say so explicitly rather than claiming success. Report results back to the user with a short summary: what you tested, what passed, anything you couldn't verify and why.

Local validation passing isn't yet a To Verify state — that transition is tied to PR publication, not to a green build on the user's machine. It happens in Phase 8.

## Phase 7 — Self-review (when significant)

Significant changes benefit from a fresh-eyes pass before they reach a teammate's PR queue. Use the built-in **`/code-review`** skill to review the working diff for correctness bugs and cleanups.

**Invoke `/code-review` when the change is significant**, e.g.:

- The diff touches a lot of files, or spans several packages
- The change lands in heavily-used / load-bearing code — a shared helper, an auth path, a proto/trait definition many drivers consume
- Cross-cutting impact — a change to `pkg/proto/*`, `pkg/node`, config schemas, anything other features rely on
- You're not fully confident the implementation lands cleanly

**Skip it for:**

- A CSS / styling tweak in a single Vue component
- Markdown or other doc updates
- A small refactor with no behavioural change
- A one-line bug fix that Phase 6 already exercised end-to-end

When in doubt, lean toward running it — a review round inside this session is cheaper than another round of PR feedback later. Move on to Phase 8 when the review loop settles.

## Phase 8 — Wrap up

### Ticket comment (usually skip)

Default: **no comment.** SCB tickets aren't reliably read again after they're picked up, so a comment is overhead with no audience. Only post one when there's something load-bearing for future readers:

- "Closing as not-a-bug — X is intentional, see <link>"
- "Surfaced a wider issue while picking this up, tracked as SCB-XYZ"
- "Partial fix landed; broader rework deferred — see PR #NNN for context"
- "Cannot reproduce — closing. If it comes back, please attach <Y>"
- (For the Phase 3 "Needs more info" exit) "Couldn't reproduce — would help to know <X, Y, Z>."

When one of these applies, just write the final wording and post it via `mcp__youtrack__add_issue_comment` — no explicit approval step. The comment is narrow in scope and posted to a ticket the team rarely revisits, so a confirmation gate buys nothing. The substantive call is *whether* to comment at all (gated above); once that's settled, the post is mechanical. Mention what you posted in the end-of-skill summary so the user sees it.

The skill already handled assignee and state transitions in earlier phases — don't touch them again here, and don't change priority, type, sprint, subsystem, or other ticket fields (those stay manual).

### PR handoff

If code changed and the user is happy, recommend they invoke `/raise-pr`. Don't auto-invoke it — that skill has its own pre-flight (validation depth, build gate, branch hygiene) that's better entered fresh. Don't commit during the iteration loop unless the user explicitly asks; iteration on review feedback is faster when changes accumulate uncommitted, and `/raise-pr` owns commit timing at the end.

### Move the ticket to To Verify

The To Verify transition is tied to **PR publication**, not local validation — until the change is on GitHub for a teammate to look at, there's nothing for anyone to verify. Once `/raise-pr` has successfully published the PR (pushed, opened, and marked ready-for-review per that skill's flow), transition the ticket from In Progress to **"To Verify"** via `mcp__youtrack__update_issue`, using the project's exact state name as discovered in Phase 1.

To Verify is the *in-review* state; the closing transition happens on **merge**, carried by the `#SCB-NNNN State Done` command `/raise-pr` puts in the PR body (processed by YouTrack's GitHub VCS integration). So don't set Done here — leave the merge to resolve it. If that integration isn't enabled, the ticket stays at To Verify after merge and someone closes it manually.

Skip the transition if:

- The user declined `/raise-pr` or `/raise-pr` did not complete (no published PR yet — state stays In Progress).
- The ticket is already past To Verify (don't backstep).
- No code change was made (Invalid / Needs more info / New-domain handed to a design pass) — there's nothing to verify, leave the state where it is.

### When there's nothing to PR

If verification ended with no code change (Invalid / Needs more info / New-domain), there's nothing to PR and no To Verify transition — say so and stop.
