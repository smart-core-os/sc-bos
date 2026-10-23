# sc profile

<!-- What varies in sc-bos for the Smart Core team's shared Claude Code workflow skills. Conventions that apply to
everyone are in AGENTS.md; this file only adds what the workflow needs on top. -->

## Gate

`scripts/gate.sh` (see `--help`). It runs CI's checks for the Go packages, protos, Rego policy and UI workspaces the
change touches, and skips a step with a note when its tool isn't installed. A docs-only change needs no gate. After a
rebase that brings in `go.mod` changes, run `scripts/gate.sh --full`.

## Validation

- **`go`**: tests for the new behaviour; a bug fix gets a regression test. For changes touching goroutines, locks or
  streams, the gate's `--race`. Where the change is observable in a running controller (an RPC, driver, automation
  or config option), evidence it was exercised: run `cmd/bos` against `example/config/vanti-ugs` (command in
  AGENTS.md) and call the API, or drive it from the Ops UI.
- **`proto`**: generated code regenerated with `bash scripts/gen-proto.sh` and committed (`pkg/proto/<name>pb`,
  `ui/ui-gen/proto`). A new trait is also registered in `pkg/node/alltraits/services.go`. Blocks ready: CI's
  proto-check fails on stale output.
- **`rego`**: `test_*` rules in `pkg/auth/policy/default/test_general.rego` proving the allow and deny paths, run
  with `opa test -v .` in that directory (the gate does this when `opa` is installed). The Go tests in
  `pkg/auth/policy` also load the default policy.
- **Permission-gated work**: there are no seeded personas. Prove it in tests with requests carrying different roles
  or permissions (the `test_general.rego` cases, or `Validate` in `pkg/auth/policy`), not by logging in as users.
- **`vue`, `js`** under `ui/`: the affected flow driven in the browser against a running controller
  (`yarn --cwd ops dev --mode=vanti-ugs` from `ui/`), with a screenshot of each changed surface. No phone-width pass.

## Pull requests

- **Title**: the `<area>: <summary> (SCB-NNNN)` format in AGENTS.md, not conventional commits. **The ticket stays in
  the title**: `ui/ops: escape CSV special characters in downloads (SCB-1375)`.
- **Ticket footer**: `#SCB-NNNN State Done`.
- **CI** runs only on PRs (and on `main`), filtered by path, so a branch needs a draft PR to get CI.
- **Reviewers**: there is no CODEOWNERS file; ask the user who should review.

## Tickets

YouTrack project: `SCB`

Branches are `<initials>/<slug>`, `feat/<slug>` or `fix/<slug>`; some people use one worktree per ticket. Design work
for a new domain goes under `docs/` (`docs/design/` for cross-cutting design, a top-level doc for a feature).

## Review

- **Proto and trait design**, when a `.proto` changes: against `proto/AGENTS.md` (trait naming, aspects, history
  records, no derivable fields).
- **Authorisation**, when `pkg/auth/policy/**` or a new non-standard RPC verb is added: does the Rego policy cover it,
  and do `test_general.rego` cases prove it.
- **Generated code**, always: no hand edits to `*.pb.go` or `ui/ui-gen/proto/**`; the `.proto` changes with them.
- Further convention sources: `AGENTS.md` and the `AGENTS.md` nearest each changed file (each `CLAUDE.md` here only
  imports its sibling `AGENTS.md`), and `.claude/review-voice.md` for how review text reads.
