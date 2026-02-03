# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Smart Core Building Operating System (SC BOS)** - A platform for connecting building systems (HVAC, lighting, security), running automations, hosting web applications, and securing access to building data. Large (~300MB), actively-developed BETA codebase.

**Tech Stack:** Go 1.25.x backend with gRPC/Protocol Buffers, Vue 3/Vite 6/Vuetify 3 frontend (Node 22.x, Yarn 1.22.x), PostgreSQL 14, Keycloak 21 (OAuth2/OIDC).

## Build & Test Commands

### Go Backend
```bash
go test -short ./...                    # Fast tests (run before committing)
go test -short -race ./...              # Race detector
go test -skip '_e2e$' ./...             # All cacheable tests
go test -count=1 -run '_e2e$' ./...     # E2E tests (uncached)
go build -o sc-bos ./cmd/bos            # Build main application
go run ./cmd/tools/genproto             # Regenerate proto code after .proto changes
```

### UI Frontend (from ui/ops or ui/space)
```bash
yarn install --frozen-lockfile --check-files  # Install deps
yarn lint:nofix                               # Check lint (run before committing)
yarn lint                                     # Auto-fix lint
yarn build                                    # Production build
yarn dev                                      # Dev server (port 5173)
```

### Local Development Services
```bash
podman compose up -d   # Starts PostgreSQL, Keycloak, pgAdmin
# Keycloak: http://localhost:8888 (admin/admin)
# PostgreSQL: localhost:5432 (postgres/postgres)
```

## Critical Environment Setup

**Required for Go operations (private repo access):**
```bash
git config --global url."https://<TOKEN>:x-oauth-basic@github.com/smart-core-os".insteadOf "https://github.com/smart-core-os"
export GOPRIVATE=github.com/smart-core-os/*
```

## Architecture

### Core Concepts
- **Traits** - gRPC service interfaces defining device capabilities (OnOff, BrightnessSensor, etc.). Requests include a `name` field identifying the target device.
- **Resources** - Data exposed by traits (e.g., `Occupancy` is the resource for `OccupancySensor` trait)
- **Aspects** - Different views into a trait: Api (data access via Get/Pull), Info (metadata), History (historical records)
- **Named Entities** - Unique resource names like `devices/light-1`, `zones/floor-2`
- **Controllers** - SC BOS instances with roles: Area Controller, Building Controller, Gateway

### Plugin System (4 types)
- **Drivers** (`pkg/driver/*`) - Device integrations (BACnet, OPC UA, Modbus, etc.)
- **Autos** (`pkg/auto/*`) - Automation logic (lights, alerts, export)
- **Zones** (`pkg/zone/*`) - Logical device groupings
- **Systems** (`pkg/system/*`) - System-level services

New plugins must be registered in their respective `pkg/*/all*/factories.go` file.

### Key Directories
```
cmd/bos/main.go           # Application entry point
cmd/tools/                 # CLI tools (genproto, scfix, etc.)
pkg/app/                   # Bootstrap, config (appconf, sysconf)
pkg/driver/, auto/, zone/, system/  # Plugin implementations
pkg/gen/                   # Generated proto code (DO NOT EDIT)
pkg/node/                  # API routing & publication
internal/                  # Internal packages
proto/                     # Protocol buffer definitions
ui/ops/                    # Operations UI (Vue 3 + Vuetify)
ui/space/                  # Space/tenant UI
ui/ui-gen/                 # Generated UI code from protos
example/config/vanti-ugs/  # Main dev config (use for testing)
```

## Validation Before Committing

- `go test -short ./...` passes
- `go test -short -race ./...` passes
- `cd ui/ops && yarn lint:nofix` passes
- `cd ui/space && yarn lint:nofix` passes
- If proto modified: `go run ./cmd/tools/genproto` executed
- No `console.log` in UI code

## Common Issues

1. **Go build "410 Gone"** - Missing private repo auth (see environment setup above)
2. **Yarn.lock has Nexus URLs** - Replace with `registry.npmjs.org` (CI auto-fixes this)
3. **Proto changes not reflected** - Run `go run ./cmd/tools/genproto`
4. **Tests need database** - Run `podman compose up -d` first
