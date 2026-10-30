# Connect telemetry ingest — BOS publisher notes

This document describes how the `connecttelemetry` automation (`pkg/auto/connecttelemetry/`) publishes
device telemetry and discovery to Smart Core Connect (SCC), and records the points still open
with the Connect ingest side.

> **The authoritative ingest contract is owned by Connect**, not this document. The source of
> truth is the `smart-core-connect` repo: `docs/ingest.md` (SCC-592, the ingest design),
> `docs/bos-auth.md` (MQTT telemetry auth), `docs/bicep/ingest-infrastructure.md` and
> `.bicep/modules/event-grid-mqtt.bicep` (the Event Grid MQTT front-door). This file is the
> **BOS-side implementation notes** — what we emit and why — plus the questions to resolve
> jointly. Where the two disagree, Connect's docs win.

> **Status: Connect's ingest consumer is not built yet.** Today only the Event Grid infra
> (bicep) and a legacy Service-Bus JSON prototype (`smart-core-connect-worker`) exist; the MQTT
> topic parser, kind classification, UDMI pointset/discovery reader and `aka` resolution are
> **design spec only** (`ingest-infrastructure.md` notes the consumer code is "a future
> adapter"). So there is no live parser to conform to and no end-to-end integration target
> until Connect builds it. What follows is aligned to the spec's shapes + the concrete infra.

The exporter discovers devices **by trait**, polls the typed trait API on a schedule, and
publishes each device's data as UDMI. Meter is published for every device that implements it;
ResourceUse only for the local node, which publishes its own host resources.

It can also publish the node's **health checks**, over the same connection, in sc-bos's own
shape rather than UDMI. See [Health](#health).

## Transport & auth (owned by Connect — `bos-auth.md`, `ingest-infrastructure.md`)

- **Transport: MQTT v5 is required** (`bos-auth.md`). MQTT v3.1.1 is not mentioned as accepted.
  Broker hostname arrives out-of-band via the node's cloud configuration ("like the
  registration URL").
- **Auth:** mTLS with the node's Connect leaf certificate (`CN = nodeId`, `credentialId` in a
  URI SAN, EKU `clientAuth`). The client presents leaf + intermediate; the broker trusts the
  Connect **root** (the only cert the broker stores).
- **Broker server verification:** Connect's docs don't state how the client validates the
  broker's server cert. Since the front-door is Azure Event Grid (public Azure TLS), we verify
  the broker against **system/public roots** — *this is a BOS-side assumption*, not stated by
  Connect. (In file-path dev mode an optional CA can override this.)
- **Identity is not in the payload, nor the topic.** Event Grid stamps the publisher identity
  as routing enrichments from the authenticated client's registry entry — see
  [Enrichment attributes](#enrichment-attributes-owned-by-connect). The publisher sets none.

## Topic grammar

- **Telemetry:** `tlm/devices/<deviceRef>/events/pointset` — **confirmed.** The `tlm/` prefix
  and `tlm/#` publish topic space are defined by the front-door
  (`event-grid-mqtt.bicep`, `publishTopicPrefix` default `tlm`); everything below the prefix is
  source-native addressing parsed by the (future) ingest layout. `.../events/pointset` matches
  the UDMI telemetry example in `ingest.md` (the "Pier Point" worked example).
- **Discovery:** `tlm/devices/<deviceRef>/events/discovery` — **BOS-chosen, NOT confirmed.**
  There is no `events/discovery` literal anywhere in Connect. `ingest.md` treats discovery as
  UDMI *device metadata* — a distinct message *kind* that "need not be interleaved with data on
  one topic" and "may be delivered on its own topic or channel." The exact subfolder is
  undefined; `events/discovery` is our provisional choice, to confirm with Connect.
- `tlm/` is the fixed intent prefix — the publish-authz scope. The MQTT topic arrives
  downstream as the CloudEvent `subject`.
- **No nodeId in the topic.** Publisher identity rides the mTLS credential and is stamped via
  routing enrichments, not the topic path.
- `<deviceRef>` is the device (producer) segment. Connect is intended to resolve it as
  `identity=(system=bos, ref)` against `aka[bos].ref` — see the deviceRef open issue below.

### How kinds are distinguished

Connect now has a `classifyKind` step plus a per-kind layout/match mechanism (shipped in
SCC-592, merged 2026-07-06). A discovery announcement is "a distinct message *kind* the
Pipeline's `layout` recognises" — the per-kind match decides, in practice from the payload
shape (telemetry carries `points{…present_value}`; discovery carries `system{…}` +
`pointset.points{…}`) and/or the topic. We publish telemetry and discovery on distinct
subfolders (`events/pointset` vs our provisional `events/discovery`) so a layout can split on
topic; the payload shapes also differ, so a body test works too. The exact match rule (topic
vs body) is Connect's to configure — align the discriminator with `classifyKind`.

## Payloads

### Telemetry — UDMI pointset

```json
{
  "timestamp": "2026-06-22T10:15:00Z",
  "version": "1.5.2",
  "points": {
    "energy_accumulator":          { "present_value": 123.45 },
    "exported_energy_accumulator": { "present_value": 67.89 }
  }
}
```

- ISO-8601 `timestamp` is the reading instant (`MeterReading.end_time`, falling back to now).
- Point names are **DBO standard field names** (`pkg/dbo`) — see the meter mapping below and
  `.claude/plans/dbo-conformance-plan.md`.
- One `present_value` per present point; an absent point means "no update" (partial messages
  are first-class in `ingest.md`).
- `version` is the UDMI envelope schema version. Note: `ingest.md`'s telemetry example **omits
  `version`** — it is a UDMI extra, harmless and ignored, not a Connect-required field.
- Read-only: writable/command points never appear in telemetry.

### Discovery — UDMI device metadata

```json
{
  "timestamp": "2026-06-22T10:15:00Z",
  "version": "1.5.2",
  "system": {
    "name": "Main Electrical Meter",
    "description": "Building main power meter",
    "tags": ["hvac"],
    "location": { "site": "", "floor": "03" }
  },
  "pointset": {
    "points": {
      "energy_accumulator":          { "units": "kWh" },
      "exported_energy_accumulator": { "units": "kWh" }
    }
  }
}
```

Shape matches `ingest.md`'s device-metadata example. Built from the device's
`metadatapb.Metadata`:

- `system.name` = appearance title (falls back to the device name); `system.description` =
  appearance description.
- `system.tags` carries the building subsystem (`membership.subsystem`) — Connect derives the
  `subsystem=` selector from tags.
- `system.location.floor` = metadata location floor → Connect's `floor=` selector. **`site` is
  left empty** — org/site identity is supplied by broker enrichment, not the payload.
- `pointset.points` is the declared inventory: point name, optional `units`, optional
  `writable`, optional `ref` (the source's own name for the point, e.g. a disk's mount point).
  Writable/command points are announced here (`writable: true`) but never sent as telemetry.

### Meter point mapping

Point names are DBO standard fields (`pkg/dbo`); discovery `units` carry the **raw** device
unit string (the raw→DBO unit-name mapping, e.g. `kWh`→`kilowatt_hours`, is a building-config
concern):

| DBO field (point name)         | Value source            | Units (discovery, raw)               |
|--------------------------------|-------------------------|--------------------------------------|
| `energy_accumulator`           | `MeterReading.usage`    | `MeterReadingSupport.usage_unit`     |
| `exported_energy_accumulator`  | `MeterReading.produced` | `MeterReadingSupport.produced_unit`  |

`exported_energy_accumulator` is only emitted (telemetry and inventory) when the meter declares
a `produced_unit`, avoiding a spurious constant-zero series for consumption-only meters.
Meters are read-only, so no point is `writable`. NB an energy-only meter has **no canonical DBO
entity type** (all `METERS/EM_PWM*` require `power_sensor`) — see the DBO conformance plan.

On the trait-poll path there are no raw vendor point names, so the trait's semantic field
names become the UDMI point names. This is the trade-off for supporting devices that don't
implement the UDMI export trait, and for per-trait rollout granularity (meter-first).

### Node resources (ResourceUse)

Connect's health spec (`docs/health.md`, "Host resources are telemetry, not a snapshot") wants a
node's CPU, memory and disk use as ordinary telemetry on the node's entity, judged by a
`healthbounds` check. The node publishes its own `smartcore.bos.ResourceUse` under its node name,
which is the `aka.bos.ref` of the node entity Connect registers (`bos-auth.md`). A gateway does
not republish its cohort's resources.

```json
{
  "timestamp": "2026-10-12T09:00:00Z",
  "version": "1.5.2",
  "points": {
    "cpu_percentage_sensor":             { "present_value": 12.5 },
    "memory_percentage_sensor":          { "present_value": 41.3 },
    "memory_datasize_sensor":            { "present_value": 6979321856 },
    "memory_datasize_capacity":          { "present_value": 16899117056 },
    "disk_percentage_sensor_705468254":  { "present_value": 63.2 },
    "disk_datasize_sensor_705468254":    { "present_value": 63215742976 },
    "disk_datasize_capacity_705468254":  { "present_value": 100000000000 },
    "disk_percentage_sensor_3086875876": { "present_value": 18.4 },
    "disk_datasize_sensor_3086875876":   { "present_value": 92000000000 },
    "disk_datasize_capacity_3086875876": { "present_value": 500000000000 }
  }
}
```

Discovery declares `units` (`percent` for utilisation, `bytes` for usage and limit) and, for each
disk point, the raw mount point as `ref`:

```json
"pointset": {
  "points": {
    "cpu_percentage_sensor":           { "units": "percent" },
    "memory_datasize_sensor":          { "units": "bytes" },
    "disk_datasize_sensor_3086875876": { "units": "bytes", "ref": "/var/lib/bos" }
  }
}
```

- ResourceUse carries no timestamp, so the event is stamped with the time the node last sampled
  it (the model's change time).
- Unset fields are omitted rather than sent as zero.
- The names are fields in Connect's catalogue, which extends DBO with the `cpu`, `memory` and
  `disk` components and the `datasize` measurement (DBO has no host resource fields). They are
  identical in both `pointNaming` modes. Connect's `ingest-transform.spec.ts` lands this exact
  payload through the BOS pipeline template.
- A disk's instance suffix is the fnv32a hash of its raw mount point, in decimal: `705468254` is
  `/` and `3086875876` is `/var/lib/bos`. See the automation README for why.

### Exclusions

- No control/command points as telemetry (ingest is read-only w.r.t. the building).
- No org/site/node identity in the body (comes from the cert/enrichment).
- No client-side aggregation (Connect lands at full resolution).

## Health

Enabled with `"health": {"enabled": true}`. The contract is Connect's: `docs/health.md` in
`smart-core-connect` ("Receiving health"). Where this section and that one disagree, Connect's wins.

### What is published

Only the checks this node evaluates itself, from its health registry
(`auto.Services.LocalHealthChecks`). A gateway also re-announces its cohort's checks into its
device store; those are never published, since each node publishes its own and Connect judges a
check's staleness by the node that sent it.

### Messages

All health messages go to two fixed topics, at **QoS 1** whatever `mqtt.qos` says.

A resource's complete set, on `tlm/bos/health`:

```json
{
  "version": 1,
  "timestamp": "2026-10-13T09:12:04.317Z",
  "resource": "pier-point/hvac/ctrl-3",
  "subject": {"kind": "device"},
  "hash": "9f2c41d07a6be853",
  "checks": [{"id": "...", "display_name": "...", "normality": "ABNORMAL"}]
}
```

- `checks` is every check the node holds for `resource`, each a `smartcore.bos.health.v1.HealthCheck`
  encoded with `protojson`, proto field names and enum names. An empty list (`[]`) removes them all.
  Connect counts a check as healthy only if its `reliability.state` is `RELIABLE`; an unspecified
  state is unhealthy ("could not tell is never fine"). Checks get `RELIABLE` on their first
  evaluation, so only a check that has never been evaluated reads unhealthy.
- `resource` rides in the envelope, not the topic, because resource names contain `/`.
- `timestamp` is when the message was built. Connect orders a resource's sets by it, to the
  millisecond, and discards a set older than the one it holds, or a different set with the same
  timestamp. So each resource's timestamps strictly increase at millisecond resolution: a set built
  in the same millisecond as the last, or after the clock steps back, is stamped a millisecond after
  the last. That holds within a process; a restart after the clock steps back can still lose sets
  until the clock catches up, or until the daily resend. Connect orders a timestamp more than five
  minutes ahead of its own clock by when it was received instead.

A manifest of every resource with at least one check, on `tlm/bos/health-manifest`:

```json
{
  "version": 1,
  "timestamp": "2026-10-13T09:15:00Z",
  "resources": [{"nameHash": "f5b04fb7c61623cf", "hash": "9f2c41d07a6be853"}]
}
```

Connect disposes of any resource the manifest leaves out, and marks one whose hash disagrees as
out of step until a matching set arrives. Out of step turns the node's `connect:health-intake`
check abnormal, since Connect takes it to mean a set was lost after the broker accepted it. Sets
and manifests aren't processed in publish order, so Connect only judges a resource whose set is
stamped before the manifest. The manifest therefore lists the hash of each resource's **last set
built**, not its current one: the set waiting for its `PUBACK` if there is one, otherwise the
acknowledged one. A set that changed but is still queued behind the rate limit isn't stamped yet,
so Connect won't judge it against the manifest, and listing its new hash would flag the intake
check on every manifest built during a burst of changes. A resource with no acknowledged set yet,
whose last send failed, or whose current set is skipped as too large, is listed with its current
hash. Connect then shows it as out of step (which is true, or may be) rather than disposing of
checks the node still has.

Connect also flags `connect:health-intake` when a node has been online for **30 minutes** without
a manifest, and marks every one of its checks stale until a set or an agreeing manifest brings it
current. The window counts from the later of the last manifest and the node coming online, and a
restart too quick to take the node offline doesn't reset it. So `health.manifestInterval` is at
most 15 minutes, and the first manifest goes 5 minutes after start rather than a whole interval.
A manifest over 512 KB is skipped (see [Limits](#limits)), so a node whose manifest has outgrown
that stalls its stream until it shrinks.

### Subject kind

From the resource's `Metadata.device_type`:

| `device_type` | `subject.kind` |
|---|---|
| `NODE`, `GATEWAY`, `HUB` | `node` |
| `SERVICE`, `AUTOMATION`, `ZONE` | `service` |
| anything else, or no metadata | `device` |

Drivers' system checks (on the driver's id) and the history stores' checks (on the store's name)
are announced as `SERVICE` for this. Automations and zones carry their own type, but every check on
their name is the service's own, so they go as `service` too. Connect files a `service` as a
component of the node's entity rather than as a device, and only knows these three kinds, so the
resource name is all that tells a driver, an automation and a zone apart there.

### Hashes

- **Set hash:** SHA-256 over the kind and each check, sorted by id and deterministically
  marshalled, with `bounds.current_value` and `deviation` cleared; first 16 hex digits. A value
  moving within the same state is therefore not a change. Connect only compares it for
  equality, so the recipe is ours and may change between releases (costing one resend of each set).
- **Name hash:** the first 16 hex digits of the SHA-256 of the resource name's UTF-8 bytes.
  This one **is** part of the contract: Connect computes it too.

### When messages are sent

- A set is sent whenever its hash differs from the last one the broker acknowledged (`PUBACK`)
  for that resource. Sends wait for the connection, so a set that changed while the node was cut
  off goes when it reconnects. An unacknowledged send is retried with backoff from 1s to 60s,
  reset when the connection comes back. It may still have landed (a late `PUBACK`, or resent from
  the MQTT session), so after a failure the next set is sent even if it matches the last one
  acknowledged.
- A resource that loses all its checks is sent as an empty set, and forgotten once that is
  acknowledged.
- Every set is also resent once a day, at a minute of the UTC day derived from the name, so
  resends spread across the day and a set Connect lost is recovered within a day.
- The manifest goes every `health.manifestInterval` (15 minutes, at most 15). The first goes 5
  minutes after start (or after one interval, if that is shorter), so Connect doesn't dispose of
  checks that are still being created, while a quick restart stays inside Connect's 30-minute
  [stall](#messages) window. An unacknowledged manifest is retried with backoff from 1s to 60s,
  rather than left for the next interval, and each retry is built afresh.
- Sending starts 10s after the automation, and runs no faster than `health.maxPublishRate`
  messages a second. The default of 10 is 6,000 per 10 minutes, inside Connect's 10,000; the
  most accepted is 16.

### Limits

Connect drops what is over its per-node limits and flags the node's `connect:health-intake`
check. The publisher warns about each once, so the cause shows up on the node too:

| Limit | Connect | Publisher |
|---|---|---|
| Resources with checks | 5,000 | warns |
| Checks per set | 50 | warns, still sends |
| Set size | 64 KB | warns, still sends; over 256 KB skips it until it changes |
| Manifest size | 512 KB | warns, skips it, so the stream [stalls](#messages) |
| Messages | 10,000 per 10 minutes | `health.maxPublishRate`, 10 a second |

A message Connect can't parse (another `version`, an unparseable `timestamp`, an unknown subject
kind) is dropped without flagging `connect:health-intake`, so an encoding bug is silent on the
Connect side. The publisher logs encode failures at error level and skips the set until it changes.

## Enrichment attributes (owned by Connect)

The Event Grid front-door stamps the publisher identity onto every routed event from the
authenticated client's registry entry. **The publisher sets none of these** — they ride the
credential.

- **Stamped CloudEvent extension attributes (lowercase)**, per `event-grid-mqtt.bicep`
  `routingEnrichments`: `nodeid`, `orgsiteid`, `siteid`, `organisationid`, `credentialid`
  (note British spelling `organisation`). These are what the consumer reads to route.
- **Source client attributes (camelCase)**, provisioned per credential
  (`smart-core-connect-eventgrid/src/eg-client.service.ts`):
  `{ type: "bos_node", nodeId, organisationId, siteId, orgSiteId }`; `credentialid` derives
  from the cert's `credentialId` (the `authenticationName`).

## Lifecycle

Discovery is published on the first cycle that reaches the broker and then every
`metadataInterval` poll cycles, and a discovery that fails to send is retried each cycle until
it goes; telemetry every cycle. Each cycle waits for the broker connection, up to the next
scheduled send, and is skipped if it can't connect by then.

Per `ingest.md`, matching the UDMI shape is not "zero-config" landing: UDMI is a **future
built-in template** (a starting layout + mappings an operator adopts), and even then an
operator must ratify the stream (bind `(bos, ref)` → entity) and author the point-name →
catalogue-field mapping; pre-ratification telemetry is held as `unresolved` and
replayed after binding. So the exporter needs no store-and-forward for correctness; in-session
assurance is MQTT QoS.

## Open issues (to resolve jointly with Connect — SCC-592)

1. **`deviceRef` contains slashes — deliberately unresolved for now.** On the trait-poll path
   there is no UDMI device id, so `deviceRef` is the **raw Smart Core device name** (e.g.
   `van/uk/brum/ugs/meters/elec-main`), which contains `/` and spans multiple topic segments.
   The spec neither forbids this (a BOS `aka[bos].ref` demonstrably contains slashes in
   `entity.md`) nor defines how a layout reduces a multi-segment topic back to one `ref`. We
   ship it as-is; **this must be revisited** (options: percent-encode into one segment,
   trailing-segment only, or a dedicated metadata ref field), and agreed with Connect alongside
   how the ref resolves to `aka[bos].ref`.
2. **Discovery subfolder.** `events/discovery` is our provisional choice; Connect has not
   defined a discovery subfolder (discovery may be its own topic/channel). Confirm the
   subfolder (or channel) Connect's layout will recognise, and that telemetry is `events/pointset`.
3. **`type` selector source.** Connect matches on a `type=` selector sourced from "the device's
   TypeName or the Pipeline's layout"; UDMI's metadata shape doesn't show where a device type
   lives, and BOS discovery currently emits **no** device type (only `tags`/`floor`). Agree
   where the `type` selector should come from (a discovery field vs layout config) and whether
   BOS should emit one.
4. **Connect leaf credential — landed (PR #890).** `internal/cloud` now exposes the X.509
   `Credential`/`NodeID()`/`TLSCertificate()` surface, wired to the automation via
   `auto.CloudCredentialSource` (`pkg/app/services.go` `startAutomations`). The exporter uses
   it in `useCloudCredential` mode; file-path mTLS remains the dev/test fallback.
5. **`nodeId` MQTT v5 user property.** The publisher sets a `nodeId` v5 user property when the
   node identity is known. **Nothing in Connect reads it today** (no local/plain-MQTT adapter
   exists); on the Event Grid path identity comes from enrichment. Kept only as a possible
   future/local-dev hook — not a read contract. Confirm if/when a local adapter needs it.

## Known limitations

- **Meter value precision (float32).** The Meter trait carries readings as `float32`, so the
  exporter forwards them at that precision. For a cumulative accumulator this is a resolution
  ceiling (~7 significant digits) once the running total gets large — reached sooner for meters
  reporting in `Wh`. This is upstream of the exporter (the trait's field type); the fix is to
  widen the Meter reading to `double` (or report deltas) at the trait level. Until then the
  exporter keeps the value as `float32` on the wire deliberately: `encoding/json` emits the
  shortest decimal that round-trips the float32, whereas widening to `float64` here would
  surface the float32's binary artifacts instead of improving fidelity.
