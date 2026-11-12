# connecttelemetry automation

The `connecttelemetry` automation streams device telemetry from an on-premise Smart Core
instance to Smart Core Connect (SCC) as **UDMI**, over the Connect telemetry
(Azure Event Grid) **MQTT v5** broker.

It discovers devices by trait, polls the typed trait API on a schedule, and publishes
each device's data per-device (every cycle, or for a trait in `sendOnChange` only on change and
heartbeat, see [Sending on change](#sending-on-change)) as:

- **Telemetry** — a UDMI pointset (`{timestamp, version, points{<name>{present_value}}}`).
- **Discovery** — a UDMI device-metadata message (`{system{name, description, tags,
  location}, pointset{points{<name>{units, writable, ref}}}}`), on the first cycle that
  reaches the broker and then every `metadataInterval` cycles. A discovery that fails to
  send is retried each cycle until it goes.

Each cycle waits for the broker connection, up to the next scheduled send, before it fetches
and publishes. A cycle that can't connect in that time is skipped, not caught up.

With `health.enabled` it also publishes the health checks this node evaluates, one complete
set per resource on `tlm/bos/health` whenever the set changes, plus a periodic manifest of every
set's hash on `tlm/bos/health-manifest`. Zones' checks are left out unless `health.includeZones`
is set. Which sets the broker has acknowledged is saved in the node's database (`db.bolt`) once a
minute, so a restart resends only the sets that changed while it was down or in the minute before.
Sends wait after start until the node's checks stop being created, for up to 2 minutes. See the
Health section of the ingest doc.

See [`docs/connect-telemetry-ingest.md`](../../../docs/connect-telemetry-ingest.md) for
the topic grammar and the payload contract agreed with the Connect ingest side.

## Supported traits

These traits are exported for every device that implements them:

- **Meter** (`smartcore.bos.Meter`)
- **Temperature** (`smartcore.bos.Temperature`), whose child devices can be sent as points of
  their parent, see [Grouping children](#grouping-children)
- **Light** (`smartcore.traits.Light`), a lamp's brightness, see [Light](#light)
- **Transport** (`smartcore.bos.Transport`), a lift's or escalator's floor, direction, doors, load,
  speed and operating mode, see [Transport](#transport)
- the environmental sensor traits **AirTemperature** (`smartcore.traits.AirTemperature`),
  **AirQualitySensor** (`smartcore.traits.AirQualitySensor`) and **OccupancySensor**
  (`smartcore.traits.OccupancySensor`)

**ResourceUse** (`smartcore.bos.ResourceUse`) is exported for the local node only.

Zones (device type `ZONE`) are skipped unless `includeZones` is set. A zone's reading is built from
devices the node already publishes, such as the sum of its meters, and Connect models zones itself, so
publishing them would count those devices twice. The same goes for the groups a zone announces as
`<zone>/<key>`, such as its `lightGroups`, `meterGroups` and `thermostatGroups`: a device named under a
zone with no device type is taken to be one of these. `devices` and `zones` can name a group as they
name a zone. Zone health is separate: see `health.includeZones`.

`devices` limits a trait to the devices it names, and admits a named zone whatever `includeZones`
says. Use it for a zone whose reading is its own measurement rather than a sum of devices already
sent. For example, a zone that counts people with an enter/leave counter at its door has a count
that no member device carries. Listing it alone under `smartcore.traits.OccupancySensor` sends that
zone without the floor's PIRs or any other zone:

```json
"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
"devices": {"smartcore.traits.OccupancySensor": ["site/zones/meeting-suite"]}
```

Meters stay unrestricted here, since `smartcore.bos.Meter` has no entry.

`zones` adds named zones to a trait without restricting it. A trait with an entry exports its devices
as it would with no entry, other zones still skipped unless `includeZones` is set, plus the zones it
names. To send the floor's PIRs as well as the counted zone above, without naming each PIR:

```json
"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
"zones": {"smartcore.traits.OccupancySensor": ["site/zones/meeting-suite"]}
```

A trait can't have both a `devices` and a `zones` entry, since `devices` already sends the zones it
names. A name in `zones` that isn't an announced zone implementing the trait is logged as
`configured zone not found for trait` at each device refresh.

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

DBO has no host resource fields, so the names come from Connect's field catalogue, which extends
DBO with the `cpu`, `memory` and `disk` components and the `datasize` measurement (standard unit
bytes) beside DBO's own `percentage`. They are the same in both `pointNaming` modes.

| Point                         | Source                 | Units (discovery) |
|-------------------------------|------------------------|-------------------|
| `cpu_percentage_sensor`       | `cpu.utilization`      | `percent`         |
| `memory_percentage_sensor`    | `memory.utilization`   | `percent`         |
| `memory_datasize_sensor`      | `memory.usage`         | `bytes`           |
| `memory_datasize_capacity`    | `memory.limit`         | `bytes`           |
| `disk_percentage_sensor_<n>`  | `disks[].utilization`  | `percent`         |
| `disk_datasize_sensor_<n>`    | `disks[].usage`        | `bytes`           |
| `disk_datasize_capacity_<n>`  | `disks[].limit`        | `bytes`           |

- Unset fields produce no point. `core_percent`, `free_bytes` and `network.connection_count`
  are not exported.
- Disks are keyed by mount point, not list position. `<n>` is DBO's numeric instance suffix: the
  fnv32a hash of the raw mount point, in decimal, so `/` → `705468254` and `/var/lib/bos` →
  `3086875876`. A mount can't be spelt into the name, as every token must be a catalogue
  subfield, and the hash keeps each disk's points the same whatever order the disks are listed in.
- Each disk point's discovery entry carries the raw mount point as `ref`.
- ResourceUse has no timestamp, so the telemetry is stamped with the time the node last sampled
  it (the model's change time). A stalled sampler shows up as stale telemetry rather than fresh.

### Environmental sensors

Each reading lands on the DBO generic room-sensor field that Connect's BOS CSV import lands the
same column on, so a sensor's imported history and its live telemetry share a series. VOC by mass
has no import column, so it lands through Connect's generic DBO field mapping instead. Units are
fixed by the trait protos and declared in discovery as DBO unit names, which Connect converts by:
it holds a numeric reading whose quantity has several units until discovery names one.

| DBO point (field)                   | Source                                          | Units (discovery)            |
|-------------------------------------|-------------------------------------------------|------------------------------|
| `zone_air_temperature_sensor`       | `AirTemperature.ambient_temperature`            | `degrees_celsius`            |
| `zone_air_relative_humidity_sensor` | `AirTemperature.ambient_humidity`               | `percent_relative_humidity`  |
| `zone_air_co2_concentration_sensor` | `AirQuality.carbon_dioxide_level`               | `parts_per_million`          |
| `zone_air_voc_concentration_sensor` | `AirQuality.volatile_organic_compounds`         | `parts_per_million`          |
| `zone_air_voc_density_sensor`       | `AirQuality.volatile_organic_compounds_density` | `micrograms_per_cubic_meter` |
| `air_pressure_sensor`               | `AirQuality.air_pressure`                       | `hectopascals`               |
| `zone_air_pm1pt0_density_sensor`    | `AirQuality.particulate_matter_1`               | `micrograms_per_cubic_meter` |
| `zone_air_pm2pt5_density_sensor`    | `AirQuality.particulate_matter_25`              | `micrograms_per_cubic_meter` |
| `zone_air_pm10pt0_density_sensor`   | `AirQuality.particulate_matter_10`              | `micrograms_per_cubic_meter` |
| `zone_occupancy_status`             | `Occupancy.state`                               | none (multistate)            |
| `zone_occupancy_count`              | `Occupancy.people_count`                        | none (headcount)             |

- Under `pointNaming: raw` each point takes its source field's name (`carbon_dioxide_level`, `state`).
- Unset optional fields produce no point, and discovery declares only the points the sensor's
  last reading populated.
- VOC in ppm and VOC by mass are separate points, never converted into each other: that needs a
  reference gas the sensor doesn't report. A sensor sends whichever it measures; a BMS that reports
  TVOC in mg/m³ should be configured onto `volatile_organic_compounds_density` (the BACnet driver's
  `vocDensity`, with `scale: 1000`), not onto the ppm field.
- Set points, `mode`, dew point, and AirQuality's `comfort`, `score`, `infection_risk` and
  `air_change_per_hour` are not exported: set points and modes are writable or enum-valued, and the
  rest are vendor indices or have no DBO field.
- `zone_occupancy_status` is sent as its canonical DBO state, a string, since Connect holds a number
  on a multistate field: `OCCUPIED`→`occupied`, `UNOCCUPIED`→`unoccupied`, `IDLE`→`inactive`
  ("likely occupied but no activity") and `STATE_UNSPECIFIED`→`unknown`. Raw naming sends the
  trait's state name.
- The trait reports a people count of 0 from a sensor that can't count, such as a lighting PIR, so
  which counts go depends on what the sensor says about itself (`OccupancySupport.max_people`, read
  when the device is found):
  - `max_people` above 0, a counter: every count, except a 0 beside an occupied or idle state, which
    Connect's import also reads as no reading.
  - `max_people` 0: no count.
  - no answer, which is every driver today: only a count above 0, so a presence detector never
    claims an empty room holds 0 people. A counter's true zeros wait for its driver to describe it.
    A zone counted only by `enterLeaveOccupancySensors` describes itself as a counter, so its zeros
    are sent; one that also lists `occupancySensors` stays undescribed.
- The reading comes from Get, which every driver serves (some poll the device on it), and is stamped
  with the time it was fetched, as a meter's is: a value the driver can still read is current. A
  model's change time would make a sensor whose value doesn't change, such as an empty room's PIR,
  look dead. A driver that can't read the device fails the Get, so nothing is sent.
- A device implementing several of these traits publishes one pointset with all their points.

### Temperature

`smartcore.bos.Temperature` says nothing of what it measures, a room, a pipe or a tank, so its
reading goes on DBO's generic temperature field, and Connect's per-source mappings land it on a
more specific one.

| DBO point (field)    | Raw point  | Source                 | Units (discovery) |
|----------------------|------------|------------------------|-------------------|
| `temperature_sensor` | `measured` | `Temperature.measured` | `degrees_celsius` |

The set point is writable, not a reading, and isn't exported. As for the environmental sensors, the
reading comes from Get and is stamped with the time it was fetched, and an unset `measured` sends nothing.

### Light

A `smartcore.traits.Light` device sends the lamp's brightness level, 0 when it is off.

| DBO point (field)              | Raw point       | Source                     | Units (discovery) |
|--------------------------------|-----------------|----------------------------|-------------------|
| `brightness_percentage_sensor` | `level_percent` | `Brightness.level_percent` | `percent`         |

- The field is DBO's brightness monitoring field, not `lighting_percentage_sensor`, which is a
  photocell's input to a lighting controller.
- A `confidence` of 0 means the driver doesn't know the level, as when a driver that reports
  confidence hasn't heard from a lamp since start-up, so nothing is sent until it does. A low but
  non-zero confidence is still sent. A driver that doesn't report confidence is taken at its word,
  so a lamp it hasn't read yet may be sent as 0.
- Discovery declares the point even while the level is unknown, so Connect has its unit before the
  first reading.
- The preset (a scene name) and the target level (where a fade is heading) aren't readings and
  aren't sent. The proto's rule that a zero level is ignored when a preset is present isn't applied:
  some drivers, such as the BACnet merge light, always set a preset, empty when no scene matches, and
  following it would drop lamps that are off.
- As for the environmental sensors, the reading comes from Get and is stamped with the time it was
  fetched, including a heartbeat's.

A site with many lamps whose levels hold steady can send them on change, see
[Sending on change](#sending-on-change):

```json
"traits": ["smartcore.traits.Light"],
"sendOnChange": {"smartcore.traits.Light": {"heartbeat": "1h"}}
```

### Transport

A `smartcore.bos.Transport` device, a lift or escalator, sends each value that has a field agreed
with Connect in SCC-1209. Multistate values are lower-case strings, Connect's vocabulary for them, in
both namings.

| DBO point (field)        | Raw point          | Source                  | Value / units (discovery)                                                       |
|--------------------------|--------------------|-------------------------|---------------------------------------------------------------------------------|
| `floor_count`            | `actual_position`  | `actual_position.floor` | integer, the driver's floor number                                               |
| `direction_status`       | `moving_direction` | `moving_direction`      | `up`, `down`, `stopped` (`NO_DIRECTION`), or another direction's lower-case name |
| `position_status`        | `doors`            | `doors[].status`        | `open`, `opening`, `closed`, `closing`, `safety_locked`, `limited_open`         |
| `load_percentage_sensor` | `load`             | `load`                  | `percent`, when Describe's `load_unit` is a percentage                          |
| `mass_sensor`            | `load`             | `load`                  | `kilograms`, `pounds_mass` or `metric_tons`, when Describe's `load_unit` is a weight |
| `linearvelocity_sensor`  | `speed`            | `speed`                 | Describe's `speed_unit` as its DBO name, such as `mm/s` as `millimeters_per_second` |
| `operating_mode`         | `operating_mode`   | `operating_mode`        | the full enum name in lower case, such as `normal` or `out_of_service`          |

- An unspecified direction, door state or operating mode, or an unset load or speed, sends no point.
- Connect reads `floor_count` as 0 for the ground floor and negative for a basement. The floor is
  sent as the driver gives it, so a controller that numbers floors otherwise, from 1 or from the
  lowest basement, needs its driver to map them, as the OPC UA driver's `actualPosition.enum` does.
- The floor is sent only when it is a whole number. The OPC UA driver gives a numeric node's value as
  `12`, or `12.00` from a float node, and both are sent as 12; a floor named through an enum, such as
  `GF`, isn't sent.
- A transport with one door sends it as `position_status` (`doors` raw). One with more sends each as
  `position_status_<n>` (`doors_<n>`), `n` from 1 in the trait's door order, since Connect takes only
  a numeric instance suffix. The OPC UA driver keeps its doors in config order.
- The load goes on the field its unit says and is never converted: Connect converts each unit to its
  quantity's standard one itself. Speed units are `mm/s`, `m/s`, `m/min`, `km/h`, `ft/s`, `ft/min` and
  `mph`, and weights `kg`, `lb` and `t`, or their DBO names. Connect fails a point whose unit it can't
  resolve, so a load or speed in any other unit, or from a device whose driver doesn't describe it,
  isn't sent.
- The units are read from Describe each time the devices are listed. Discovery declares the load and
  speed points with their units then, so Connect has them before the first value. A driver that
  learns its units late, such as the BACnet merge transport, whose first Describe is `unspecified`
  until its first poll, sends no load until the devices are next listed.
- Next destinations, the journey, ETA, payloads, the passenger alarm, the active state and the stopped
  reason have no agreed field and aren't sent. Faults belong to health, not telemetry.
- As for the environmental sensors, the reading comes from Get and is stamped with the time it was
  fetched.

### Grouping children

Some drivers announce each point of a piece of plant as its own device. A BACnet heat exchanger, for
example, is a device holding its set points and alarms, with each water temperature a Temperature
device beneath it: `…/PHX-S-01-01/chw-primary-flow`, `…/PHX-S-01-01/chw-primary-return` and so on.
Connect would see one device per point. `groupChildren` sends them as one:

```json
"traits": ["smartcore.bos.Temperature"],
"groupChildren": ["smartcore.bos.Temperature"]
```

- A device of a grouped trait named `<parent>/<leaf>`, whose parent is an announced device, is sent
  as a point of `<parent>`: one pointset and one discovery for the parent, with the parent's metadata.
  The parent needn't implement the trait.
- The point is keyed by the leaf in lower case, each run of characters other than letters and digits
  becoming `_`: `chw-primary-flow` → `chw_primary_flow`, `DHW-N-09-PHX-primary-lthw-flow` →
  `dhw_n_09_phx_primary_lthw_flow`, in either `pointNaming`. Connect's per-source mappings turn the
  keys into fields. A parent with the trait itself keeps its own reading on `temperature_sensor`.
- A child whose key is taken, by another child or by a parent that has the trait itself, is logged
  and not sent.
- With health on, each child's checks go in its parent's set, ids prefixed with the leaf
  (`chw-primary-flow/healthbounds:chw-return`), so the same automation's check on several children
  stays unique. The child's own ref gets no set. Health doesn't depend on `traits` or `devices`, so
  this holds for a child whose reading isn't sent too. See the ingest doc's Health section.
- Each change resends the parent's whole set, so a parent whose checks change waits half a second
  before its set is built, and children changing together (one automation's checks on every point,
  which change in the same pass) send it once. A device with no children goes at once, as ungrouped.
  A check that flips often, such as a bound set where a point normally runs, now costs its parent's
  whole set each time, so give bounds a deadband wider than the reading's normal swing.
- Neither a zone nor a zone's child is grouped.
- A parent, once found, stays one until the config changes, for telemetry and health alike: a driver
  reconfiguring removes and re-announces its devices, and a child caught without its parent would
  otherwise go out as itself until the next device refresh.

Only `smartcore.bos.Temperature` can be grouped, and grouping is off unless configured, so other
sites see no change.

### Sending on change

Every device is sent on every cycle by default, whether or not its reading changed. For traits
whose readings rarely change, such as a PIR, nearly every one of those pointsets repeats
the last. `sendOnChange` sends a trait's points only when they change, or when its heartbeat has
passed since they were last sent:

```json
"traits": ["smartcore.bos.Meter", "smartcore.traits.OccupancySensor"],
"sendOnChange": {"smartcore.traits.OccupancySensor": {"heartbeat": "1h"}}
```

- Each device is still polled every `sendInterval`; only what is published is reduced. A change is
  sent on the next cycle after it happens, so it is never more than one interval late.
- Any difference in a point's value is a change: there is no deadband. A continuous reading such as
  a temperature or CO2 level moves a little on nearly every cycle, so sending it on change saves
  little. It suits readings that hold steady, such as occupancy or a lamp's level.
- The unit of change is one trait of one device, or one grouped child. Its points are left out of the
  device's pointset while they are unchanged and not yet due, and Connect reads an absent point as no
  update. A device whose other traits are on the grid still sends those points every cycle, so its
  pointsets are partial between heartbeats. A device with nothing to send sends no pointset.
- The pointset's timestamp is the latest reading among the points it carries.
- `heartbeat` defaults to `1h` and must be positive. It is measured between the cycles' scheduled
  times, so `1h` on a `*/5` grid sends an unchanged reading on exactly every 12th cycle. It counts from
  the last send, change or heartbeat, and isn't staggered across devices.
- A failed publish, or a trait that couldn't be read, sends its points again on the next cycle that
  reads them.
- What was last sent is held in memory and kept across a device refresh, but not across a restart
  or a config change: the first cycle after either sends every on-change trait, changed or not.
- `smartcore.bos.Meter` can't be sent on change: meters stay on the send grid, which Connect's meter
  liveness relies on.

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
  against system/public roots (Event Grid presents a public Azure TLS cert).
  The automation follows the node's enrolment: it connects once the node is enrolled, and
  disconnects and resends everything when the node is re-enrolled under another node id or
  endpoint. While the node is not enrolled it publishes nothing, and its system check reports why.
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
  },
  "health": {
    "enabled": true,
    "manifestInterval": "15m",
    "maxPublishRate": 10
  }
}
```

To publish health only, leave out `traits`.

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
| `traits` | unless health is enabled | — | Traits to export, any of `smartcore.bos.Meter`, `smartcore.bos.ResourceUse`, `smartcore.bos.Temperature`, `smartcore.bos.Transport`, `smartcore.traits.Light`, `smartcore.traits.AirTemperature`, `smartcore.traits.AirQualitySensor` and `smartcore.traits.OccupancySensor` |
| `groupChildren` | no | — | Traits whose `<parent>/<leaf>` devices are sent as points of their parent, telemetry and health alike, see [Grouping children](#grouping-children). Only `smartcore.bos.Temperature`, which must be in `traits` |
| `sendOnChange` | no | — | Map of trait name to `{"heartbeat": "<duration>"}`. Those traits' points are sent only when they change or when `heartbeat` (default `1h`) has passed since they were last sent, see [Sending on change](#sending-on-change). Each key must be in `traits`; `smartcore.bos.Meter` is rejected |
| `pointNaming` | no | `dbo` | `dbo` emits DBO field names; `raw` emits the trait's own names (`usage`/`produced` for meters) and leaves the mapping to the consumer. ResourceUse points are the same in both |
| `fetchTimeout` | no | `5s` | Per-device trait-fetch timeout |
| `includeZones` | no | `false` | Also publish zones that implement `traits`, and their groups such as `lightGroups`. Off by default because a zone's readings are built from devices already published |
| `devices` | no | — | Map of trait name to the device names to export for it. A trait with an entry exports only those devices, zones included whatever `includeZones` says; traits without one are unchanged. Each key must be in `traits` and each list non-empty |
| `zones` | no | — | Map of trait name to zone names to export alongside the trait's other devices, see [Supported traits](#supported-traits). Has no effect with `includeZones` set. Each key must be in `traits` and have no `devices` entry, and each list non-empty |
| `mqtt.host` | yes | — | MQTT v5 broker URL; must use a TLS scheme (`tls://`, `ssl://`, `mqtts://`, `wss://`) — non-TLS schemes are rejected so certs are never silently dropped |
| `mqtt.topicPrefix` | no | `tlm` | Fixed intent prefix (publish-authz scope) |
| `mqtt.clientId` | no | node id | MQTT client id |
| `mqtt.useCloudCredential` | no | `false` | Use the Connect leaf credential (mutually exclusive with file-path certs) |
| `mqtt.clientCertPath` / `mqtt.clientKeyPath` | file mode | — | Client cert + key for mTLS |
| `mqtt.caCertPath` | no | system roots | CA to verify the broker |
| `mqtt.sendInterval` | no | `*/15 * * * *` | Telemetry poll schedule |
| `mqtt.metadataInterval` | no | `100` | Publish discovery, and refresh the device list, every N cycles (and cycle 0); either is retried each cycle until it succeeds |
| `mqtt.qos` | no | `1` | MQTT QoS (0, 1, or 2) |
| `mqtt.connectTimeout` | no | `5s` | Connect timeout |
| `mqtt.publishTimeout` | no | `5s` | Per-publish timeout. Each cycle has already waited for the connection, up to the next send |
| `health.enabled` | no | `false` | Publish this node's own health checks |
| `health.manifestInterval` | no | `15m` | Time between health manifests, at most `15m`: Connect flags a node that goes 30 minutes without one |
| `health.maxPublishRate` | no | `10` | Most health messages a second, at most `16` (Connect accepts 10,000 per 10 minutes) |
| `health.includeZones` | no | `false` | Also publish the checks on zones, as services of the node. Off by default because Connect associates building checks with the devices in a space itself, and files a service's checks as node health. Independent of `includeZones` |
