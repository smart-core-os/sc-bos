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

### historyimports

Updates JavaScript/TypeScript imports for history aspects extracted to separate proto files.

```js
// Before
import {ElectricHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/history_grpc_web_pb';
import {ListElectricDemandHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/history_pb';
/** @type {import('@smart-core-os/sc-bos-ui-gen/proto/history_pb').ElectricDemandRecord.AsObject} */

// After
import {ElectricHistoryPromiseClient} from '@smart-core-os/sc-bos-ui-gen/proto/electric_history_grpc_web_pb';
import {ListElectricDemandHistoryRequest} from '@smart-core-os/sc-bos-ui-gen/proto/electric_history_pb';
/** @type {import('@smart-core-os/sc-bos-ui-gen/proto/electric_history_pb').ElectricDemandRecord.AsObject} */
```

**Manual fixes required**

The tool will only replace imports when it _knows_ the target filename for the symbols being imported.
Some imports came from trait specific files like `health_pb`, which the tool uses to infer the new history file as `health_history_pb`.
When the symbols were defined in the common `history_pb` or `history_grpc_web_pb` files, this gets harder to determine.
The tool is able to infer the target import filename when imported clients only refer to a single trait e.g., `ElectricHistoryPromiseClient`.

In all other cases (no clients or multiple clients), the tool cannot determine the target filename and will require manual fixes.
The tool will show `! Manual import fix needed` or `! Manual JSDoc fix needed` in these cases.
You will need to inspect these files and manually replace the imported file path with the correct `_history_pb` or `_history_grpc_web_pb` file.
