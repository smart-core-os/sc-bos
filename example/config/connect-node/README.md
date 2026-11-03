# Connect Node Example

A minimal, self-contained BOS node for testing the cloud connection against Smart Core Connect (SCC) or a local
stand-in such as [`cmd/cloudsim`](../../../cmd/cloudsim). It needs no postgres, keycloak or UI build, and its ports
are chosen so it can run beside the other example nodes.

| API   | Address  |
|-------|----------|
| gRPC  | `:23560` |
| HTTPS | `:8460`  |

The app config contains:

- A `mock` driver with three `smartcore.bos.Meter` devices, one `smartcore.bos.Health` device, and a driver-level
  health check that flaps to a fault 15% of the time (`healthCheck.faultProbability`).
- A `healthbounds` automation that raises a health check on each meter whose usage leaves `0..10000`.
- A `history` automation recording meter readings in memory.
- A [`connecttelemetry`](../../../pkg/auto/connecttelemetry) automation that publishes the meters and the node's own
  host resources (`smartcore.bos.ResourceUse`) to the Connect telemetry broker every minute, and the node's health
  checks as they change, using the node's enrolled certificate (`useCloudCredential`).

The system config turns on the `resourceUse` system, which announces the node's CPU, memory and disk use as
`smartcore.bos.ResourceUse` on the `connect-node` device. Without it there is no host telemetry to send.

The `connecttelemetry` broker host is a placeholder, `tls://REPLACE-WITH-BROKER-HOST:8883`. Until you replace it the
automation logs a connection warning on every retry, which is harmless. Everything else works without it.

Device names deliberately contain no `/`. `connecttelemetry` puts the device name straight into the topic
(`tlm/devices/<name>/events/pointset`), so a name with a slash spans several topic segments and won't match a
single-level `+` subscription.

## Running

Copy the config somewhere you can edit it, then boot. The `--data` directory holds the node's keys and cloud
registration, so give each test its own.

```shell
mkdir -p .data/connect-node
cp example/config/connect-node/*.json .data/connect-node/
go run ./cmd/bos --policy-mode=off \
  --sysconf .data/connect-node/system.conf.json \
  --appconf .data/connect-node/app.conf.json \
  --data .data/connect-node/data
```

`--policy-mode=off` permits unauthenticated requests, so the `grpcurl` calls below need no token. Don't use it outside
local testing.

Check the node is up and not yet enrolled:

```shell
grpcurl -insecure -d '{"name":"connect-node"}' localhost:23560 \
  smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi/GetCloudConnection
# "state": "UNCONFIGURED"
```

## Enrolling

Get an enrolment code for the target node from the SCC management console (or cloudsim, below). Codes are single
use and short-lived. Then register, passing the SCC registration endpoint:

```shell
grpcurl -insecure -d '{
  "name": "connect-node",
  "enrollment_code": {"code": "ABC123", "register_url": "https://<scc-host>/v1/device/register"}
}' localhost:23560 smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi/RegisterCloudConnection
```

The response carries the `nodeId` SCC assigned. `GetCloudConnection` should then report `CONNECTED` with a
`lastCheckInTime`. If you have the Ops UI pointed at this node, its "Link to Cloud" dialog does the same thing.

`system.conf.json` has no `cloud.registerUrl`, so `register_url` is required here. Setting `cloud.registerUrl` makes it
the default.

There is no need to restart after enrolling, unlinking or re-enrolling. `connecttelemetry` connects to the broker
once the node is enrolled, using the node ID as its MQTT client ID and `nodeId` property, reconnects as the new node
when it is re-enrolled, and disconnects when it is unlinked. The registration is stored under `--data`, so it
survives a restart.

Unlink when you're done:

```shell
grpcurl -insecure -d '{"name":"connect-node"}' localhost:23560 \
  smartcore.bos.ops.cloud.v1alpha.CloudConnectionApi/UnlinkCloudConnection
```

## Things that will catch you out

- **A config deployment replaces this app config.** Once enrolled, if SCC has a config deployment for the node, BOS
  loads that instead of `app.conf.json` and the mock devices disappear. Enrol into a node with no deployment. The log
  line `in cloud config mode, but no active config available - using local config instead` confirms the local config
  is in use.
- **Each SCC environment has its own CA.** A registration made against one environment, or against cloudsim, is not
  valid against another. Use a fresh `--data` directory when switching. Unlinking isn't enough: it clears only the
  registration, and a config deployment already downloaded under `<data>/cloud` stays and loads after the next
  enrolment. cloudsim regenerates its CA every time it starts, so re-enrol after restarting it.
- **`connecttelemetry` needs TLS.** It rejects `mqtt://` hosts, so it can't talk to a plain-text broker on 1883. Point
  it at a TLS listener.

## Against cloudsim

cloudsim stands in for SCC's device API (registration, check-in, renewal, config deployment). It has no telemetry
broker, so `connecttelemetry` keeps warning.

cloudsim serves a self-signed certificate, so add `"cloud": {"insecureSkipVerify": true}` to your copy of
`system.conf.json`. Never set this against a real SCC.

Start from an empty database. The `-data` file persists between runs, and cloudsim regenerates its CA anyway, so
keeping it buys nothing here:

```shell
rm -f .data/cloudsim.db
go run ./cmd/cloudsim -listen 127.0.0.1:9443 -data .data/cloudsim.db
```

Create a site, a node and an enrolment code, either in the web UI at <https://127.0.0.1:9443> or through the API.
The `1`s below assume an empty database. Otherwise use the `id` from each create response:

```shell
curl -sk -X POST https://127.0.0.1:9443/api/v1/management/sites -d '{"name":"test"}'
curl -sk -X POST https://127.0.0.1:9443/api/v1/management/nodes -d '{"hostname":"connect-node","siteId":"1"}'
curl -sk -X POST https://127.0.0.1:9443/api/v1/management/nodes/1/enrollment-codes -d '{}'
```

Enrol with `"register_url": "https://127.0.0.1:9443/v1/device/register"`. Check-ins appear at
`/api/v1/management/nodes/1/check-ins`.

## Health scenarios

The driver's `systemStatusCheck` flaps by itself. To push a meter out of bounds on demand, first stop its simulation.
The mock meters take their first reading 30 seconds to 2 minutes after start, then a new one every 1 to 30 minutes,
and each overwrites a forced value:

```shell
grpcurl -insecure -d '{"name":"meter-01","automations":[{"trait":"smartcore.bos.Meter","active":false}]}' \
  localhost:23560 smartcore.bos.mock.v1.MockDeviceApi/SetDeviceAutomation
```

Then force the value:

```shell
grpcurl -insecure -d '{"name":"meter-01","values":[
  {"trait":"smartcore.bos.Meter","value_protojson":"{\"usage\":50000}"}
]}' localhost:23560 smartcore.bos.mock.v1.MockDeviceApi/ForceTraitValue
```

The meter's `healthbounds:connect-node/auto/meter-health` check goes `HIGH` and stays there, and
`connecttelemetry` sends the change to Connect, where it shows held at the meter's ref until the device is
adopted. List every device carrying a health check with:

```shell
grpcurl -insecure -d '{"query":{"conditions":[{"field":"health_checks.id","present":{}}]}}' \
  localhost:23560 smartcore.bos.devices.v1.DevicesApi/ListDevices
```
