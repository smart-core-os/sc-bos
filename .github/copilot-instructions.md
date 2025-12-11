# SC-BOS Development Guide for AI Coding Agents

## Repository Overview

**Smart Core Building Operating System (SC BOS)** is a platform for connecting building systems (HVAC, lighting, security), running automations, hosting applications, and securing access to building data and control. This is a large (~300MB), actively-developed codebase under BETA status with no backwards compatibility guarantees.

**Tech Stack:**
- **Backend:** Go 1.25.x, gRPC, Protocol Buffers
- **Frontend:** Vue.js 3, Vite 6, Vuetify 3, Node.js 22.x, Yarn 1.22.x
- **Infrastructure:** PostgreSQL 14, Keycloak 21 (OAuth2/OIDC), Docker/Podman
- **Key Dependencies:** `github.com/smart-core-os/sc-api` (Smart Core API), `github.com/smart-core-os/sc-golang`

## Agent Behavior Guidelines

**Assume a working development environment.** The toolchain (Go 1.25.x, Node.js 22.x, Yarn 1.22.x) is already installed and configured. Only troubleshoot environment setup if you encounter specific errors that indicate missing dependencies or configuration issues.

**Test using unit tests, not ad-hoc commands.** Write or run existing unit tests to validate changes. Avoid one-off CLI commands or manual testing during development. Use `go test` for Go code and existing test suites for UI code.

**Be concise.** Do not over-comment code or generate verbose documentation. Code should be self-explanatory. Only add comments for complex logic or non-obvious decisions. Keep explanations of changes brief and focused on what matters.

## Critical Build Requirements

### Environment Setup

**ALWAYS required before ANY Go operations:**
```bash
# Private repo access is REQUIRED - builds will fail without this
git config --global url."https://<TOKEN>:x-oauth-basic@github.com/smart-core-os".insteadOf "https://github.com/smart-core-os"
export GOPRIVATE=github.com/smart-core-os/*
```

**For local development (database & auth):**
```bash
podman compose up -d  # or docker-compose up -d
# This starts PostgreSQL, Keycloak, and pgAdmin
# Keycloak: http://localhost:8888 (admin/admin)
# PostgreSQL: localhost:5432 (postgres/postgres)
```

### Build & Test Commands

**Go Backend:**
```bash
# Test (ALWAYS run this before committing)
go test -short ./...                    # Fast tests only
go test -skip '_e2e$' ./...            # All cacheable tests
go test -count=1 -run '_e2e$' ./...    # End-to-end tests (uncached)
go test -short -race ./...             # Race detector on short tests

# Build main application
go build -o sc-bos ./cmd/bos

# Generate protobuf code (run after modifying .proto files)
go run ./cmd/tools/genproto
```

**UI Frontend (ops and space):**
```bash
cd ui/ops  # or ui/space

# ALWAYS run yarn install before building if dependencies changed
yarn install --frozen-lockfile --check-files

# Lint (ALWAYS run before committing)
yarn lint:nofix  # Check only
yarn lint        # Auto-fix

# Build
yarn build       # Output to dist/

# Dev server
yarn dev         # Usually runs on port 5173
```

**Important:** UI workspaces use a monorepo structure at `ui/package.json` with workspaces for `ops`, `space`, `ui-gen`, `panzoom-package`, and `keycloak-login-prototype`.

## GitHub CI Workflows

All workflows are in `.github/workflows/` and run on push/PR. **Make changes that pass these checks:**

1. **go-test.yml** - Runs on Go file changes (`cmd/`, `internal/`, `pkg/`, `go.mod`, `go.sum`)
   - Go 1.25.x
   - Tests with `-skip '_e2e$'` (cached)
   - Tests with `-count=1 -run '_e2e$'` (e2e)
   - Race detector with `-short -race`

2. **ui.yml** - Runs on all pushes
   - Node.js 22.x
   - Checks for private Nexus registry URLs in `ui/yarn.lock`
   - Auto-commits cleaned `yarn.lock` if private URLs found
   - **Critical:** Uses `registry.npmjs.org`, NOT private Nexus

3. **go-security.yml** - Runs `govulncheck` on all pushes
   - Go 1.25.x vulnerability scanning

4. **build-sc-bos.yml** - Release workflow (tags only)
   - Builds for linux-amd64, linux-arm64, freebsd-amd64, win-amd64
   - Builds ops-ui with Vite
   - Creates GitHub releases

**Docker build:** Uses multi-stage Dockerfile requiring `.npmrc` secret:
```bash
docker build --secret=id=npmrc,src=$HOME/.npmrc .
```

## Project Structure

```
cmd/bos/main.go              # Main application entry point
cmd/tools/                   # CLI tools (genproto, scfix, etc.)
pkg/
  ├── app/                   # Application bootstrap & config (appconf, sysconf)
  ├── driver/                # Device drivers (bacnet, opcua, modbus, etc.)
  ├── auto/                  # Automation systems (lights, export, bms, etc.)
  ├── zone/                  # Zone management
  ├── system/                # System-level services
  ├── node/                  # API routing & publication
  ├── gen/                   # Generated protobuf code (DO NOT EDIT)
  └── gentrait/              # Protobuf utilities
internal/                    # Internal packages (account, auth, etc.)
proto/                       # Protocol buffer definitions
ui/
  ├── ops/                   # Operations UI (Vue 3 + Vuetify)
  ├── space/                 # Space/tenant UI (Vue 3 + Vuetify)
  └── ui-gen/                # Generated UI code from protos
example/config/              # Example configs
  ├── vanti-ugs/             # Main dev example (use this for testing)
  └── hub/                   # Multi-controller cohort example
scripts/gen-proto.sh         # Protobuf code generation
docker-compose.yml           # Dev services (Postgres, Keycloak)
.run/                        # JetBrains IDE run configurations
```

## Configuration Files

- **go.mod/go.sum** - Go dependencies (module: `github.com/smart-core-os/sc-bos`)
- **proto.mod** - Proto dependencies
- **ui/package.json** - Yarn workspace root
- **ui/ops/package.json**, **ui/space/package.json** - UI app configs
- **ui/ops/eslint.config.js**, **ui/space/eslint.config.js** - ESLint configs (flat config format)
- **ui/ops/vite.config.js**, **ui/space/vite.config.js** - Vite build configs
- **docker-compose.yml** - PostgreSQL, Keycloak, pgAdmin services
- **Dockerfile** - Multi-stage build (Node.js 22 + Go 1.25)

## Key Architectural Concepts

### Smart Core Terminology

**Traits** - Standard gRPC service interfaces that define device capabilities (e.g., `OnOff`, `BrightnessSensor`, `OccupancySensor`). Defined in both `github.com/smart-core-os/sc-api` and locally in `proto/`. Trait requests always have an associated `name` field identifying the target device.

**Resources** - The data exposed by a trait. For example, `OccupancySensor` is the trait, and `Occupancy` is the resource it exposes. Resources represent the actual state or data being managed.

**Aspects** - Different views into a trait, each serving a specific purpose:
- **Api aspect** (e.g., `OccupancySensorApi`) - Primary access to trait resource data via RPCs using verbs like `Get` or `Pull` (e.g., `OccupancySensorApi.GetOccupancy`)
- **Info aspect** (e.g., `OccupancySensorInfo`) - Optional metadata and discovery information like expected range, units, or capabilities
- **History aspect** (e.g., `OccupancySensorHistory`) - Historical records for the resource (e.g., past `Occupancy` values)

**Named Entities** - Devices, automations, and zones have unique resource names in the system (e.g., `devices/light-1`, `zones/floor-2`). These names are used in the `name` field of trait requests.

### SC BOS Architecture

**Controllers** - Each SC BOS instance. Roles: Area Controller (floor devices), Building Controller (global automation, accounts, CA), Gateway (external API, no device integrations).

**Services** - Plugin system with 4 types:
- **Drivers** (`pkg/driver/*`) - Device integrations (BACnet, OPC UA, etc.)
- **Autos** (`pkg/auto/*`) - Automation logic (lights, alerts, export)
- **Zones** (`pkg/zone/*`) - Logical groupings of devices
- **Systems** (`pkg/system/*`) - System-level services

**APIs** - gRPC services defined in `proto/` + imported from `github.com/smart-core-os/sc-api`

**Config Loading** - JSON configs in `example/config/` support includes via glob patterns. Drivers/automation/zones merge by name (first-come, first-served).

## Common Pitfalls & Workarounds

1. **Go build fails with "410 Gone"** - Missing private repo auth. Set `GOPRIVATE` and git config (see above).

2. **Yarn install fails with 404** - Check `.npmrc` has GitHub token. For CI, ensure `NODE_AUTH_TOKEN` secret is set.

3. **UI yarn.lock has Nexus URLs** - The ui.yml workflow auto-fixes this. Locally: `sed -i -e 's/nexus\.vanti\.co\.uk\/repository\/npm-public/registry\.npmjs\.org/g' ui/yarn.lock`

4. **Protobuf changes not reflected** - Run `go run ./cmd/tools/genproto` after modifying `.proto` files. Generated code goes in `pkg/gen/` and `ui/ui-gen/`.

5. **Tests fail with database errors** - Ensure `podman compose up -d` is running for tests needing PostgreSQL/Keycloak.

6. **CGO errors** - Backend builds use `CGO_ENABLED=0` for static binaries (see Dockerfile and workflows).

## Making Changes

**For Go changes:**
1. Edit code in `cmd/`, `internal/`, or `pkg/`
2. If touching `.proto` files: run `go run ./cmd/tools/genproto`
3. Run `go test -short ./...` - MUST pass
4. Build: `go build ./cmd/bos`
5. For driver/auto/zone/system plugins, register factory in `pkg/*/all*/factories.go`

**For UI changes:**
1. Edit code in `ui/ops/` or `ui/space/`
2. Run `yarn lint:nofix` - MUST pass
3. Run `yarn build` - MUST succeed
4. Test in browser with `yarn dev`

**For proto changes:**
1. Edit `.proto` files in `proto/`
2. Run `go run ./cmd/tools/genproto` - regenerates `pkg/gen/` and `ui/ui-gen/`
3. Test affected Go and UI code

## Validation Checklist

Before committing, ALWAYS verify:
- [ ] `go test -short ./...` passes (no database needed)
- [ ] `go test -short -race ./...` passes (race detector)
- [ ] `cd ui/ops && yarn lint:nofix` passes
- [ ] `cd ui/space && yarn lint:nofix` passes
- [ ] If proto modified: `go run ./cmd/tools/genproto` executed
- [ ] Code follows existing patterns in similar files
- [ ] No `console.log` in UI code (use proper logging)
- [ ] No hardcoded credentials or secrets

## Trust These Instructions

These instructions are comprehensive and validated. Only perform additional searches if:
1. You need to understand implementation details of a specific feature
2. These instructions are incomplete for your specific task
3. You encounter an error not documented here

When exploring, focus on:
- Similar existing implementations (e.g., other drivers in `pkg/driver/`)
- Tests in `*_test.go` files for usage patterns
- Config examples in `example/config/vanti-ugs/`
- Documentation in `docs/` for architecture decisions

