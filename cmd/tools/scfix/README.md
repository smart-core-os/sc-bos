# scfix

Applies automated code transformations to modernize sc-bos and dependent repositories.

## Usage

```bash
# List available fixes
scfix -list

# Apply all fixes (default)
scfix

# Dry run (recommended first)
scfix -dry-run -v

# Apply specific fixes
scfix -only=wrap,optclients

# Skip specific fixes
scfix -skip=wrap
```

## Flags

- `-list` - List available fixes and exit
- `-dry-run` - Show changes without applying them
- `-only` - Run only specified fixes by ID (comma-separated or repeated)
- `-skip` - Skip specified fixes by ID (comma-separated or repeated)
- `-v` - Verbose output
- `-q` - Quiet mode (errors only)

Note: Runs from the git repository root automatically.

## Fixes

### optclients

Replaces deprecated `node.WithOptClients` and `node.HasOptClient` functions.

```go
// Before
node.Announce("example",
    node.HasTrait("foo", node.WithOptClients(client1)),
    node.HasOptClient(client2),
)

// After
node.Announce("example",
    node.HasTrait("foo", node.WithClients(client1)),
    node.HasClient(client2),
)
```

### wrap

Converts generated `pkg.WrapFoo` calls to `New*Client` with `wrap.ServerToClient`.

```go
// Before
client := airqualitysensorpb.WrapApi(server)

// After  
client := traits.NewAirQualitySensorApiClient(wrap.ServerToClient(traits.AirQualitySensorApi_ServiceDesc, server))
```

