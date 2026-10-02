---
name: fix-security-alerts
description: Triage and fix open Dependabot security alerts and govulncheck findings: list the alerts, trace each dependency (direct or transitive), apply the fix (go get, yarn.lock re-resolve or a scoped resolution), run the gate, and dismiss alerts that don't apply, with a reason. Use when the user says "fix security alerts", "fix govulncheck", "dependabot alerts", "clear the vuln alerts" or similar. Pass --dry-run to report only.
argument-hint: [--dry-run]
---

# Fix security alerts $ARGUMENTS

Triage open Dependabot alerts for `smart-core-os/sc-bos` and govulncheck findings, then fix, dismiss or escalate each one. Two ecosystems carry alerts: **Go** (`go.mod`, one module at the root) and **npm** (`ui/yarn.lock`, a yarn 1 classic workspace root for `ops`, `panzoom-package`, `space`, `signage`, `ui-gen`).

If `--dry-run` is passed, stop after Phase 1 and only report.

## Efficiency rules

- **Combine related commands** with `&&` in a single Bash call, and **use parallel tool calls** for independent queries (`gh api`, govulncheck, `yarn why`).
- **Batch fixes.** Apply every safe change, then install and run the gate once.
- **Use `gh --jq`, not `jq`.** There is no `jq` on this host.
- **One fix per package range, not per alert.** Dependabot raises one alert per advisory, so a package often has several (fast-uri has seven). The fix is the highest patched version within each major range.

## Phase 1: Discover

### 1. List open alerts

```bash
gh api 'repos/:owner/:repo/dependabot/alerts?state=open&per_page=100' --paginate \
  --jq '.[] | "\(.number)\t\(.dependency.package.ecosystem)\t\(.dependency.package.name)\t\(.dependency.relationship)\t\(.security_advisory.severity)\t\(.security_vulnerability.vulnerable_version_range)\t\(.security_vulnerability.first_patched_version.identifier // "no fix")\t\(.security_advisory.ghsa_id)\t\(.security_advisory.summary)"'
```

Read `.security_vulnerability`, not `.security_advisory.vulnerabilities[0]`. The first is the range the alert matched against *our* installed version. The second is just the advisory's first range, which is the wrong fix for any package on more than one major (brace-expansion has separate 1.1.x and 5.0.x fixes).

`.dependency.scope` is `runtime` for every entry in `ui/yarn.lock` (yarn 1 lockfiles don't record dev vs prod), so don't trust it. Work out the scope with `yarn why`. `relationship` is `transitive`, `direct` or `inconclusive`. Treat `inconclusive` as "go and look" (vite is one).

If there are no alerts and govulncheck is clean, say so and stop.

### 2. Go: govulncheck

```bash
go version
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Pure Go, so it runs natively on Windows. It adds two things Dependabot can't give:

- **Symbol-level reachability.** "Your code is affected" with example traces means the vulnerable symbol is reachable. The "doesn't appear to call" footer lists the unreachable findings (`-show verbose` for detail). Only that second group can ever be dismissed as `not_used`.
- **Stdlib and toolchain vulns,** which Dependabot never raises.

Stdlib findings are relative to the **toolchain that ran the scan**, not `go.mod`. So the local `go version` can report vulns that CI never sees, because `go-security.yml` resolves `go-version-input: 1.26` to the latest 1.26.x. What ships is the `golang:` tag in the `Dockerfile` (and `demo/vanti-ugs/Dockerfile-*`). Compare a stdlib finding's "Fixed in" against that tag: if the tag is older, the release image is affected and the tag needs to move, whatever CI says.

For each Go module alert, `go mod why -m <module>` gives the import path that pulls it in. Every OTel module is `// indirect`, pulled in through OPA (`opa/v1/rego` → `opa/v1/plugins` → `otel/sdk/trace`).

### 3. npm: trace each package

From `ui/`:

```bash
yarn why <pkg>
```

For each installed version, record the parent chain and the semver range the parent asks for (in `ui/yarn.lock` the entry header lists every range that resolved to it, e.g. `brace-expansion@^1.1.7:`). Then decide:

- **In range:** the patched version satisfies every requesting range. Re-resolving the lock fixes it with no pin.
- **Out of range:** at least one parent pins below the fix. It needs a `resolutions` entry, or a bump of the parent.
- **Dev-only:** only reached through build, lint or test tooling (eslint, svgo via vite-svg-loader, vite plugins). It is still worth fixing, but it is a candidate for `tolerable_risk` if the fix is hard.

Check the existing `resolutions` in `ui/package.json` too. A pin there may already cover the fix, or may be what's holding the package back.

### 4. Summarise

Present one row per package range, with the alert numbers folded in:

| Alerts | Package | Installed | Severity | Via | Fix | Method |
|---|---|---|---|---|---|---|
| 177, 190, 191 | brace-expansion | 1.1.13 | high | minimatch@3 | 1.1.18 | re-resolve (`^1.1.7`, in range) |
| 167, 175, 187, 188 | brace-expansion | 5.0.5 | high | minimatch ← glob | 5.0.9 | re-resolve (`^5.0.5`, in range) |
| 210 | go.opentelemetry.io/otel/sdk | 1.44.0 | low | opa/v1/rego | 1.45.0 | go get (OTel family) |
| (govulncheck) | stdlib net/http | go1.26.2 (Dockerfile) | n/a | many traces | go1.26.6 | bump Go tag |

Categorise:

- **Auto-fix:** a patched version exists and is in range, or is a patch/minor bump of a direct dependency or a Go module.
- **Dismiss:** not exploitable here. For Go, only when govulncheck says the symbol is unreachable. Explain why.
- **Needs attention:** a major bump, no fix available, a resolution that would cross a major, or a chain you can't untangle.
- **Already fixed:** the lock or `go.mod` already carries the patched version and the alert just hasn't closed yet. Dependabot re-scans on push to `main`.

If `--dry-run`, stop here.

## Phase 2: Apply

### Go module

```bash
go get <module>@v<fixed> && go mod tidy
```

Move OpenTelemetry as a family, never one module alone: `go get go.opentelemetry.io/otel@v<fixed> go.opentelemetry.io/otel/sdk@v<fixed> go.opentelemetry.io/otel/metric@v<fixed> go.opentelemetry.io/otel/trace@v<fixed>`. That is the same lockstep as the `opentelemetry` group in `.github/dependabot.yml`. Check `git diff go.mod` afterwards: anything that moved besides the target is MVS pulling it forward, and belongs in the summary.

### Go stdlib (govulncheck only)

Bump every place that pins Go in one go, to the "Fixed in" release or later:

- `Dockerfile` and `demo/vanti-ugs/Dockerfile-*`: the `golang:<ver>-alpine<ver>` tag. This is what ships.
- `go.mod`: the `go` line only if a fix needs a newer minimum. Don't raise the `go` line just to force a patch release on everyone.
- The workflows already float (`'^1.26.x'`, `go-version-input: 1.26`). Only touch them on a minor bump.

Re-run govulncheck with a matching local toolchain to confirm: `GOTOOLCHAIN=go1.26.<n> go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.

### npm, fix in range

Delete the package's entries from `ui/yarn.lock` (each whole block, header to blank line) and reinstall. yarn re-resolves those ranges to the newest matching version and leaves the rest of the lock alone. No permanent pin. Prefer this to a resolution whenever it works.

### npm, fix out of range

Add `resolutions` to `ui/package.json`. A blanket `"<pkg>": "^<fixed>"` forces every consumer onto one major, which breaks anything on another major (brace-expansion 1.x and 5.x both live in this tree). Scope it to the parent chain instead:

```json
"resolutions": {
  "<parent>/**/<pkg>": "^<fixed>"
}
```

Pick a parent that only reaches the one major. A name alone may not do it: both brace-expansion majors here come through a package called `minimatch` (3.x and 10.x), so `minimatch/**/brace-expansion` would still cross the line. `yarn why <pkg>` only shows one level, so walk up with `yarn why <parent>` until the chains diverge. If they never do, bump the parent instead.

package.json can't hold a comment, so give the reason for each resolution in the PR description. Remember that a resolution also silently holds future bumps, so the summary should name it as something to remove once the parent catches up.

### Install

`yarn.lock` (v1) is platform-independent, but `ui/node_modules` isn't. Install on the platform that last installed it, or you swap the native binaries out from under the other one:

```bash
ls ui/node_modules/@rolldown/    # binding-win32-* → native, binding-linux-* → WSL
```

```powershell
# Native (win32 binding)
Set-Location ui; yarn install
```

```bash
# WSL (linux binding). Source nvm or WSL picks up the Windows yarn and finds no node.
wsl.exe -e bash -lc 'export NVM_DIR="$HOME/.nvm"; . "$NVM_DIR/nvm.sh"; cd /mnt/c/Users/DeanRedfern/dev/sc-bos/ui && yarn install'
```

After the install, check the lock:

- `yarn why <pkg>` now reports only patched versions.
- No private registry URLs crept in. `ui-lint.yml` fails the PR on `nexus.vanti.co.uk/repository/` in `ui/yarn.lock`.
- `yarn install --frozen-lockfile` passes. That is what CI and the `Dockerfile` run.

### Dismissals

```bash
gh api repos/:owner/:repo/dependabot/alerts/<number> -X PATCH \
  -f state=dismissed -f dismissed_reason=<reason> \
  -f dismissed_comment='<explanation>'
```

Valid `dismissed_reason` values: `fix_started`, `inaccurate`, `no_bandwidth`, `not_used`, `tolerable_risk`.

- **Go `not_used`:** only when govulncheck lists the finding as unreachable. Quote it in the comment, e.g. `govulncheck @<commit>: module required but vulnerable symbol <pkg.Func> not reachable`. A finding with example traces is reachable, even if the traces only reach it from `init`. Fix it instead.
- **npm `tolerable_risk`:** a dev-only path (build or lint tooling that never runs on untrusted input and isn't in the shipped bundle) where the fix is out of reach. Name the chain in the comment.
- **Never dismiss just to make the count go down.** If a fix exists and applies cleanly, apply it. Dismissals are public on this repo.

Dismissals are outward-facing. List them and confirm with the user before sending.

## Phase 3: Verify

The local gate from `.claude/CLAUDE.md`. On Windows, `go test ./...` has known host failures, so run the full suite from WSL.

**Go** (if `go.mod`, a Dockerfile or the toolchain moved):

```bash
go build ./... && go vet -lostcancel=false ./... && staticcheck ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
wsl.exe -e bash -lc 'cd /mnt/c/Users/DeanRedfern/dev/sc-bos && go test ./...'
```

govulncheck should show no remaining reachable findings for what you fixed.

**UI** (if `ui/yarn.lock` or `ui/package.json` moved). Lint every workspace, as CI does, from WSL:

```bash
wsl.exe -e bash -lc 'export NVM_DIR="$HOME/.nvm"; . "$NVM_DIR/nvm.sh"; cd /mnt/c/Users/DeanRedfern/dev/sc-bos/ui && for d in ops panzoom-package space signage ui-gen; do yarn --cwd "$d" lint:nofix || exit 1; done'
```

Then `yarn --cwd <ws> build` for `ops`, `space` and `signage`, on whichever platform matches `node_modules` (see Install). A lock change can move a transitive build dependency under any of them, not just the one you were thinking of.

If something fails:

- A resolution too broad? Scope it to the parent chain.
- A major incompatibility? Revert that fix and move the alert to Needs attention, or to `tolerable_risk` if it's dev-only.
- Re-run the gate after each revert.

## Phase 4: Summary

Report:

- **Fixed:** each package range, from → to, with its alert numbers and method (go get, re-resolve, resolution, Go tag).
- **Dismissed:** alert numbers, reason and comment.
- **Needs attention:** what's blocking each one, and the suggested next step.
- **Already fixed:** alerts that should close on the next scan of `main`.

Show the `go.mod` require diff and any `resolutions` added or removed. Hand off to `/raise-pr`, and keep fixes separate from any change to `.github/dependabot.yml` so each can be reviewed on its own. Alerts close on their own once the fix reaches `main`.
