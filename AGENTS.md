# sc-bos

Smart Core Building Operating System: a Go application that integrates building systems (HVAC, lighting,
security, metering), runs automations over them, and serves a gRPC API and web UIs. Concepts and layout are in
[README.md](README.md); feature and design docs are in [docs/](docs).

Keep this file to what applies across the repo. Guidance for one area goes in an `AGENTS.md` in that directory.

## Layout

- `cmd/bos`: the main binary. `cmd/tools/*`: dev and support tools, including `genproto`.
- `pkg/app`: bootstrap and config (`appconf`, `sysconf`). `pkg/node`: API publication and routing.
- `pkg/driver`, `pkg/auto`, `pkg/zone`, `pkg/system`: the plugin services, one per sub-package. Each kind is
  registered in its `all*/factory.go` (`pkg/driver/alldrivers`, `pkg/auto/allautos`, `pkg/zone/allzones`,
  `pkg/system/allsystems`); a new plugin is not reachable until it is added there.
- `proto/smartcore/bos/<name>/v1`: API definitions. Generated Go goes to `pkg/proto/<name>pb`, generated JS to
  `ui/ui-gen/proto`. See [proto/AGENTS.md](proto/AGENTS.md).
- `pkg/auth/policy/default`: Rego authorisation policy, one file per gRPC service.
- `ui/`: Yarn 1 workspace (`ops`, `space`, `signage`, `ui-gen`, `panzoom-package`). Vue 3 + Vuetify.
- `supervisor/`: the out-of-process update supervisor (same Go module).
- `example/config/*`: runnable configs. `vanti-ugs` is the usual dev setup; `hub` and `vanti-ugs-cohort` run a
  cohort; `connect-node` talks to Smart Core Connect.

## Concepts

- **Trait**: a gRPC service describing one device capability (`OnOff`, `Meter`, `Health`), named after the state
  it manages. Every request carries a `name` identifying the target device, zone or automation.
- **Aspects** of a trait: `<Trait>Api` (get/pull/update the resource), `<Trait>Info` (metadata such as units or
  supported values) and `<Trait>History` (past records of the resource).
- **Controller**: one running sc-bos. In a cohort, area controllers own devices, the building controller holds
  accounts and acts as CA (the hub), and gateways expose one API for the whole site.

## Commands

```bash
scripts/gate.sh                     # before every commit or PR: CI's checks for what changed (--help)
go test -skip '_e2e$' ./pkg/foo/... # one area; CI also runs the _e2e$ tests uncached and -short -race
go build ./cmd/bos
go run ./cmd/bos --policy-mode check --data .data/vanti-ugs \
  --appconf example/config/vanti-ugs/app.conf.json \
  --sysconf example/config/vanti-ugs/system.conf.json
bash scripts/gen-proto.sh           # after any .proto change; commit the generated output
```

- **Gate output** for each step is in `.tmp/gate/<step>.log`; grep those rather than re-running.
- **UI**: install and run yarn from `ui/`, never `npm` (npm ignores `yarn.lock` and trips on a peer-dependency
  conflict in `signage`). `yarn install --frozen-lockfile`, then `yarn --cwd ops dev` (add `--mode=vanti-ugs` to
  point at the example config), `yarn --cwd ops lint:nofix`, `yarn --cwd ops build`. `yarn --cwd ui/ops` from the
  repo root does not resolve the hoisted binaries.
- **Services**: `docker compose up -d` (or `podman compose`) starts PostgreSQL, Keycloak and pgAdmin for features
  that need them. Details in [docs/install/dev.md](docs/install/dev.md).
- **Proto generation** needs the pinned `protoc` toolchain; `genproto` rejects any other version. Install it with
  `scripts/dev-container/install-protoc.sh` using the versions in
  `cmd/tools/genproto/internal/toolchain/versions.go`.
- **Rego**: `opa test -v .` from `pkg/auth/policy/default`.

## Conventions

- **Never edit generated code by hand**: `*.pb.go` and `ui/ui-gen/proto/**`. Change the `.proto` and regenerate.
  Hand-written helpers for a trait (`model.go`, `model_server.go`, etc.) live beside the generated code in
  `pkg/proto/<name>pb`.
- **Prove behaviour with tests.** New packages get `_test.go`; bug fixes get a test that would have caught the bug.
  Where the change is observable in a running controller or the UI, also exercise it there.
- **Linting**: `go vet -lostcancel=false` and `staticcheck` (config in `staticcheck.conf`) as CI runs them; ESLint
  per UI workspace. No stray `console.log` in UI code (`console.warn`/`console.error` for real problems are fine).
- **`ui/yarn.lock` must only reference `registry.npmjs.org`.** CI rejects private Nexus URLs.
- **Comments** explain non-obvious decisions, not what the code does.

## Commits and PRs

- **Title**: `<area>: <summary>`, where area is the package or path touched (`ui/ops`, `auto/connecttelemetry`,
  `proto/rebootpb`). Imperative, lower-case area, no full stop. Append `(SCB-NNNN)` when the work tracks a ticket.
- **PR body**: why the change exists (the diff shows what), any non-obvious trade-off, and how it was verified.
- **Before raising**: `scripts/gate.sh` passes, and generated code is regenerated and committed.
- There is no CODEOWNERS file, so reviewers are requested explicitly.
