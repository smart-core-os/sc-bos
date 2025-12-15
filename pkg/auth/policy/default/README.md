# Policy files for the Smart Core BOS

TL;DR admins can do anything anywhere, zones are restricted by name, roles restrict permissions.

Test these rules by running `opa test . --v0-compatible` from this directory.

We've split the policies into rules and utilities:

- token.rego contains token related utils like "does the token have this role"
- rpc.rego contains utils for working with requests, for example "is this a write request"

The bulk of the rules are in `smartcore.rego`, which applies these rules to the standard verbs and conventions of Smart
Core apis. Exceptions are encoded into specific rpc rego files.

## Global Data Available to All Policies

The policy package injects some global data into the policy evaluation context. 

### `data.system` - Information about the Smart Core system
- `data.system.known_traits` - A list of fully qualified traits recognised by BOS. 
  Each entry is an object like `{"name": "smartcore.bos.SoundSensor", "grpc_services": ["smartcore.bos.SoundSensorApi"]}`.
  This can be used by the policies to tell which gRPC requests are for trait APIs.