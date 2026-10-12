# connecttelemetry automation

The `connecttelemetry` automation streams device telemetry from an on-premise Smart Core
instance to Smart Core Connect (SCC) as **UDMI**, over the Connect telemetry
(Azure Event Grid) **MQTT v5** broker.

It discovers devices by trait, polls the typed trait API on a schedule, and publishes
each device's data per-device as:

- **Telemetry** — a UDMI pointset (`{timestamp, version, points{<name>{present_value}}}`).
- **Discovery** — a UDMI device-metadata message (`{system{name, description, tags,
  location}, pointset{points{<name>{units, writable, ref}}}}`), on the first cycle and then
  every `metadataInterval` cycles.

See [`docs/connect-telemetry-ingest.md`](../../../docs/connect-telemetry-ingest.md) for
the topic grammar and the payload contract agreed with the Connect ingest side.

## Supported traits

Two traits are exported: **Meter** (`smartcore.bos.Meter`) for every device that implements
it, and **ResourceUse** (`smartcore.bos.ResourceUse`) for the local node only.

### Meter

Point names are **Google Digital
Buildings Ontology (DBO) standard field names** (mapped in `pkg/dbo`), so the building-config
translation is an identity mapping — see `docs/connect-telemetry-ingest.md` and
`.claude/plans/dbo-conformance-plan.md`. The meter reading maps to:

| DBO point (field)              | Source                  | Units (discovery)                   |
|--------------------------------|-------------------------|-------------------------------------|
| `energy_accumulator`           | `MeterReading.usage`    | `MeterReadingSupport.usage_unit`    |
| `exported_energy_accumulator`  | `MeterReading.produced` | `MeterReadingSupport.produced_unit` |

`exported_energy_accumulator` is only emitted when the meter reports production (its support
declares a `produced_unit`), so consumption-only meters don't publish a constant-zero series.
The telemetry timestamp is `MeterReading.end_time` (the reading instant), falling back to now.
Discovery units carry the **raw** device unit string; the raw→DBO unit-name mapping (e.g.
`kWh`→`kilowatt_hours`) is a building-config concern. Meters are read-only, so no point is
marked `writable`.

### ResourceUse

The node's host CPU, memory and disk use, published as ordinary telemetry on the node's own
entity: Connect gives each node an entity whose `aka.bos.ref` is the node name, so
`tlm/devices/<nodeName>/events/pointset` binds to it. Connect judges the values with a
`healthbounds` check rather than a status snapshot.

Requires the `resourceUse` system to be enabled in `system.conf.json`; it announces the trait on
the node name. The collector attaches **only to the local node** (`Node.Name()`): a gateway sees
its cohort's ResourceUse too, but each node publishes its own.

DBO has no host resource fields, so these names are our own proposal (Connect's catalogue
fields are SCC-978). They are the same in both `pointNaming` modes.

| Point                       | Source                 | Units (discovery) |
|-----------------------------|------------------------|-------------------|
| `cpu_utilization`           | `cpu.utilization`      | `percent`         |
| `memory_utilization`        | `memory.utilization`   | `percent`         |
| `memory_usage`              | `memory.usage`         | `bytes`           |
| `memory_limit`              | `memory.limit`         | `bytes`           |
| `disk_<mount>_utilization`  | `disks[].utilization`  | `percent`         |
| `disk_<mount>_usage`        | `disks[].usage`        | `bytes`           |
| `disk_<mount>_limit`        | `disks[].limit`        | `bytes`           |

- Unset fields produce no point. `core_percent`, `free_bytes` and `network.connection_count`
  are not exported.
- Disks are keyed by mount point, not list position. `<mount>` is the mount point lowercased,
  with each run of characters outside `[a-z0-9]` replaced by `_` and trimmed; empty becomes
  `root`. So `/` → `root`, `/var/lib/bos` → `var_lib_bos`, `C:\` → `c`. If two mounts reduce to
  the same name (`/var-lib` and `/var/lib`), each gets a `_<fnv32a hex>` suffix of its raw
  mount point, so the names don't depend on disk order.
- Each disk point's discovery entry carries the raw mount point as `ref`.
- ResourceUse has no timestamp, so the telemetry is stamped with the time of the fetch.

### Other traits

Other traits configured in `traits` are logged as unsupported and skipped. Add a trait by
implementing a `traitCollector` in `device.go` and wiring it in `newCollector` (`auto.go`).

## Topics

Published per-device under the fixed intent prefix `tlm/`:

- Telemetry: `tlm/devices/<deviceRef>/events/pointset`
- Discovery: `tlm/devices/<deviceRef>/events/discovery`

> **Known gap:** `deviceRef` is currently the raw Smart Core device name, which contains
> `/` and therefore spans multiple topic segments. This is deliberately unresolved for now
> — see the "Open issues" section of the decision doc. Publisher (node) identity is **not**
> in the topic: on the Event Grid path it rides the mTLS credential (broker enrichment); on
> a local MQTT v5 broker it is set as the `nodeId` user property.

## Authentication

Two mutually-exclusive credential modes:

- **`useCloudCredential: true`** — presents the node's Connect leaf certificate (mTLS) via
  `GetClientCertificate`, so renewals are picked up live. The broker (server) is verified
  against system/public roots (Event Grid presents a public Azure TLS cert). *Pending PR
  #890, which provides the leaf credential to the automation; until then use the file-path
  mode.*
- **File-path certs** (`clientCertPath` + `clientKeyPath`, optional `caCertPath`) — a
  dev/test fallback. `caCertPath` verifies the broker; empty falls back to system roots.

## Configuration

```json
{
  "type": "connecttelemetry",
  "traits": ["smartcore.bos.Meter", "smartcore.bos.ResourceUse"],
  "fetchTimeout": "5s",
  "mqtt": {
    "host": "tls://telemetry.example.com:8883",
    "topicPrefix": "tlm",
    "clientId": "scc-exporter-1",
    "useCloudCredential": true,
    "sendInterval": "*/15 * * * *",
    "metadataInterval": 100,
    "qos": 1,
    "connectTimeout": "5s",
    "publishTimeout": "5s"
  }
}
```

For the file-path dev mode, drop `useCloudCredential` and supply certs instead:

```json
"mqtt": {
  "host": "tls://localhost:8883",
  "clientCertPath": "/path/to/client.crt",
  "clientKeyPath": "/path/to/client.key",
  "caCertPath": "/path/to/ca.crt"
}
```

### Options

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `traits` | yes | — | Traits to export: `smartcore.bos.Meter` and/or `smartcore.bos.ResourceUse` |
| `fetchTimeout` | no | `5s` | Per-device trait-fetch timeout |
| `mqtt.host` | yes | — | MQTT v5 broker URL; must use a TLS scheme (`tls://`, `ssl://`, `mqtts://`, `wss://`) — non-TLS schemes are rejected so certs are never silently dropped |
| `mqtt.topicPrefix` | no | `tlm` | Fixed intent prefix (publish-authz scope) |
| `mqtt.clientId` | no | node id | MQTT client id |
| `mqtt.useCloudCredential` | no | `false` | Use the Connect leaf credential (mutually exclusive with file-path certs) |
| `mqtt.clientCertPath` / `mqtt.clientKeyPath` | file mode | — | Client cert + key for mTLS |
| `mqtt.caCertPath` | no | system roots | CA to verify the broker |
| `mqtt.sendInterval` | no | `*/15 * * * *` | Telemetry poll schedule |
| `mqtt.metadataInterval` | no | `100` | Publish discovery every N cycles (and cycle 0) |
| `mqtt.qos` | no | `1` | MQTT QoS (0, 1, or 2) |
| `mqtt.connectTimeout` | no | `5s` | Connect timeout |
| `mqtt.publishTimeout` | no | `5s` | Per-publish timeout (includes awaiting connection) |
