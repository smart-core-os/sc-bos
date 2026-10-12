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
publishes each device's data as UDMI. The first rollout is **meter-only**.

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
    "cpu_utilization":              { "present_value": 12.5 },
    "memory_utilization":           { "present_value": 41.3 },
    "memory_usage":                 { "present_value": 6979321856 },
    "memory_limit":                 { "present_value": 16899117056 },
    "disk_root_utilization":        { "present_value": 63.2 },
    "disk_root_usage":              { "present_value": 63215742976 },
    "disk_root_limit":              { "present_value": 100000000000 },
    "disk_var_lib_bos_utilization": { "present_value": 18.4 },
    "disk_var_lib_bos_usage":       { "present_value": 92000000000 },
    "disk_var_lib_bos_limit":       { "present_value": 500000000000 }
  }
}
```

Discovery declares `units` (`percent` for utilisation, `bytes` for usage and limit) and, for each
disk point, the raw mount point as `ref`:

```json
"pointset": {
  "points": {
    "cpu_utilization":        { "units": "percent" },
    "memory_usage":           { "units": "bytes" },
    "disk_var_lib_bos_usage": { "units": "bytes", "ref": "/var/lib/bos" }
  }
}
```

- The timestamp is the fetch time: ResourceUse carries none.
- Unset fields are omitted rather than sent as zero.
- `<mount>` in disk point names is the sanitised mount point (`/` → `root`); see the automation
  README for the rule and how colliding mounts are disambiguated.
- **These names await Connect's catalogue.** DBO has no host resource fields, so the names are
  our proposal and identical in both `pointNaming` modes. SCC-978 adds the matching fields to
  Connect's catalogue; until it lands, the points have no catalogue field to bind to.

### Exclusions

- No control/command points as telemetry (ingest is read-only w.r.t. the building).
- No org/site/node identity in the body (comes from the cert/enrichment).
- No client-side aggregation (Connect lands at full resolution).

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

Discovery is published on the first cycle and then every `metadataInterval` poll cycles;
telemetry every cycle. Per `ingest.md`, matching the UDMI shape is not "zero-config" landing:
UDMI is a **future built-in template** (a starting layout + mappings an operator adopts), and
even then an operator must ratify the stream (bind `(bos, ref)` → entity) and author the
point-name → catalogue-field mapping; pre-ratification telemetry is held as `unresolved` and
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
