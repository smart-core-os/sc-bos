---
name: fix-deprecated-wrap
description: Refactor deprecated WrapXxx calls (WrapApi, WrapInfo, WrapHistory, WrapAdminApi, WrapService) from sc-bos generated proto packages. Use when the user asks to fix wrap deprecations, remove WrapApi/WrapInfo calls, or migrate from deprecated trait wrappers.
disable-model-invocation: true
---

# Fix Deprecated WrapXxx Calls

## Scope — only sc-bos proto packages

Only refactor `WrapXxx` calls where the package qualifier resolves to an import from `github.com/smart-core-os/sc-bos/pkg/proto/*` (e.g. `lightpb`, `meterpb`, `airqualitysensorpb`).

**Do not touch** `WrapApi`, `WrapInfo`, etc. defined in any other package — they are unrelated functions that happen to share the name.

To confirm a match, check the import block of the file: the package alias used at the call site must map to a `github.com/smart-core-os/sc-bos/pkg/proto/*` import path.

## Step 1 — Run the scfix tool

Run the automated fixer first. It handles trivial cases and reduces the number of files requiring manual attention:

```bash
go run github.com/smart-core-os/sc-bos/cmd/tools/scfix -only wrap
```

The tool cannot handle every case — in particular, it skips `WrapXxx` calls whose result is passed to `node.WithClients` or `node.HasClient`, since those require the `wrap.ServiceUnwrapper` interface that client types don't implement. Any remaining calls after this step require the manual patterns below.

## Step 2 — Find remaining affected files

```bash
grep -rln '\.WrapApi\|\.WrapInfo\|\.WrapHistory\|\.WrapAdminApi\|\.WrapService\b' --include='*.go' .
```

For each file, check its import block. Keep only files where at least one `WrapXxx` call's qualifier maps to a `github.com/smart-core-os/sc-bos/pkg/proto/*` import path. Discard files where all matches come from other packages.

If `$ARGUMENTS` specifies particular files or packages, scope the search accordingly.

## Step 3 — Refactor all files in parallel

Each file is independent. Launch one general-purpose agent per file **in a single message** so they all run concurrently — but cap concurrency at **8 agents at a time**. When there are more than 8 files, group related files (e.g. all files under one driver subdirectory) into a single agent until the total number of agents is 8 or fewer.

Each agent prompt should include:
1. The absolute path(s) of its assigned file(s)
2. The scope rule (only `github.com/smart-core-os/sc-bos/pkg/proto/*` qualifiers)
3. The classification rules and transformation patterns from the reference section below
4. Instructions to read, edit, and write each file
5. **"Do not summarize what you changed. Report only files you could not fix or any type errors you encountered."**

After all agents complete, proceed to Step 4.

## Step 4 — Verify

```bash
go build ./...
```

Fix any remaining type errors by re-checking the patterns below.

---

## Transformation reference (include in each agent prompt)

### Classifying a call site

For each qualifying call, determine which pattern applies by reading the surrounding code:

### Pattern 1 — Node registration (most common)

The result is passed to `node.WithClients(...)` or `node.HasClient(...)` to register a trait server with a node. Replace the whole construct with `node.HasServer(...)`.

**Old:**
```go
announcer.Announce(name,
    node.HasTrait(t, node.WithClients(pkg.WrapApi(server))),
)
```
**New:**
```go
announcer.Announce(name,
    node.HasServer(pkg.RegisterXxxApiServer, pkg.XxxApiServer(server)),
    node.HasTrait(t),
)
```

Multiple wraps inside one `WithClients` become multiple `node.HasServer` arguments:
```go
// Old:
node.HasTrait(t, node.WithClients(pkg.WrapApi(s), pkg.WrapInfo(s2)))
// New (passed separately to Announce, not to HasTrait):
node.HasServer(pkg.RegisterXxxApiServer, pkg.XxxApiServer(s)),
node.HasServer(pkg.RegisterXxxInfoServer, pkg.XxxInfoServer(s2)),
node.HasTrait(t)
```

When `node.HasClient(pkg.WrapApi(s))` is passed directly to `Announce`:
```go
// Old:
announcer.Announce(name, node.HasClient(pkg.WrapApi(server)))
// New:
announcer.Announce(name, node.HasServer(pkg.RegisterXxxApiServer, pkg.XxxApiServer(server)))
```

### Pattern 2 — Client creation

The result is stored in a variable for making RPC calls (not passed to node at all, or the variable is an interface type). Replace with `NewXxxClient(wrap.ServerToClient(...))`.

```go
// Old:
client := pkg.WrapApi(server)
// New:
client := pkg.NewXxxApiClient(wrap.ServerToClient(pkg.XxxApi_ServiceDesc, server))
```

### Naming conventions — Register/Server function names

The `WrapXxx` suffix maps to these naming conventions:

| WrapXxx suffix | Register function | Server interface | ServiceDesc |
|---|---|---|---|
| `WrapApi` | `RegisterXxxApiServer` | `XxxApiServer` | `XxxApi_ServiceDesc` |
| `WrapInfo` | `RegisterXxxInfoServer` | `XxxInfoServer` | `XxxInfo_ServiceDesc` |
| `WrapHistory` | `RegisterXxxHistoryServer` | `XxxHistoryServer` | `XxxHistory_ServiceDesc` |
| `WrapAdminApi` | `RegisterXxxAdminApiServer` | `XxxAdminApiServer` | `XxxAdminApi_ServiceDesc` |
| `WrapService` | `RegisterXxxServiceServer` | `XxxServiceServer` | `XxxService_ServiceDesc` |

Where `Xxx` is derived from the package name (e.g. `lightpb` → `Light`, `airqualitysensorpb` → `AirQualitySensor`, `meterpb` → `Meter`).

**When uncertain about the exact name**, grep the package:
```bash
grep -n 'func Register' pkg/proto/PKGNAME/*.go
```

### Import updates

- Remove `"github.com/smart-core-os/sc-bos/pkg/wrap"` from files where `WrapXxx` calls are removed and `wrap` is no longer used.
- Add `"github.com/smart-core-os/sc-bos/pkg/wrap"` if switching to Pattern 2 (`wrap.ServerToClient`).
- Add `"github.com/smart-core-os/sc-bos/pkg/node"` if not already present.

### Helper functions accepting `wrap.ServiceUnwrapper`

If a file has a helper function with signature like:
```go
func helper(n *node.Node, name string, client wrap.ServiceUnwrapper) node.Undo {
    return n.Announce(name, node.HasClient(client))
}
```
Refactor it to accept `node.Feature` (or inline the announce call) and remove the `wrap.ServiceUnwrapper` parameter:
```go
func helper(n *node.Node, name string, feature node.Feature) node.Undo {
    return n.Announce(name, feature)
}
```

### Gotchas

- `node.WithClients` and `node.HasClient` are both removed — neither has a direct `node.HasServer` equivalent in terms of signature, but `node.HasServer(...)` is passed directly to `Announce` alongside `node.HasTrait`.
- `node.HasTrait(t)` no longer needs any options when the `WithClients` option is removed; drop the options argument entirely.
- When the same server value is used for both `WrapApi` and `WrapInfo`, it can be passed to both `node.HasServer` calls:
  ```go
  node.HasServer(pkg.RegisterXxxApiServer, pkg.XxxApiServer(server)),
  node.HasServer(pkg.RegisterXxxInfoServer, pkg.XxxInfoServer(server)),
  ```
- The explicit interface cast (`pkg.XxxApiServer(server)`) is required for Go's type inference with `node.HasServer[S any]`.
- Do NOT add helpers to `pkg/wrap`, and do NOT revert to `WrapXxx` functions.