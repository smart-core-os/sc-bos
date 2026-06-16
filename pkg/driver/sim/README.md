# sim driver

The `sim` driver simulates a whole building with **physically-coupled** behaviour. Where the
[`mock`](../mock) driver gives every trait its own independent random automation, `sim` runs a
single engine in which the pieces influence each other the way a real building does:

- **People** arrive, move between rooms and leave over the working day. Their presence drives
  occupancy sensors, motion/PIR sensors and enter/leave counts.
- **Lights** come on in occupied rooms during working hours and dim when there's more daylight.
- **FCUs** (fan coil units) ramp their fans with occupancy; room temperature rises with the
  number of people and is pulled back toward the setpoint by the fan.
- **Energy** is *derived*: the building's electrical demand is the base load plus the actual
  lighting and FCU loads, and the meter accumulates that demand over time. CO₂ rises with how
  full a room is.

## Configuration

The driver is configured at the building level rather than as a device list. You describe
floors, the rooms on each floor, and the device *archetypes* in each room; the driver expands
each archetype into concrete devices named `<namePrefix>/<floor>/<room>/<type>-NN`.

```jsonc
{
  "type": "sim",
  "namePrefix": "sim/demo",
  "timeMultiplier": 288,           // optional: 1 = real time, 288 = a day every 5 minutes
  "tickInterval": "5s",            // optional: wall-clock simulation cadence (default 5s)
  "baseLoadW": 5000,               // optional: always-on building load (default 5000W)
  "workingHours": {"start": 8, "end": 18, "days": ["Mon","Tue","Wed","Thu","Fri"]},
  "healthCheck": {"faultProbability": 0.05}, // optional driver-level connectivity sim
  "floors": [
    {
      "name": "ground", "title": "Ground Floor", "maxOccupancy": 60,
      "rooms": [
        {
          "name": "open-plan", "title": "Open Plan",
          "archetypes": [
            {"type": "lighting", "count": 6},
            {"type": "fcu", "count": 2, "setPointC": 21},
            {"type": "pir"},
            {"type": "brightness"},
            {"type": "airquality"}
          ]
        }
      ]
    }
  ],
  "buildingDevices": [             // devices reporting whole-building aggregates
    {"type": "meter"},
    {"type": "electric"}
  ]
}
```

A room's `maxOccupancy` defaults to an even share of its floor's `maxOccupancy`.

### Archetypes

An archetype is a named device *type* defined as a set of SmartCore traits (some,
like `fcu`, are composites of several), together with the simulation coupling that
drives them. The trait set is the archetype's authoritative capability list — the
table below mirrors what each `type` declares in the driver's archetype registry.

| `type`       | Traits                                  | Coupling |
|--------------|-----------------------------------------|----------|
| `lighting`   | Light, Status                           | level follows occupancy + working hours, dimmed by daylight; contributes lighting watts |
| `fcu`        | AirTemperature, FanSpeed, OnOff, Status | fan ramps with occupancy; temperature responds to occupant heat vs. fan cooling; contributes FCU watts |
| `pir` / `occupancy` | OccupancySensor                  | people count / state from room occupants |
| `motion`     | MotionSensor                            | detected while the room is occupied |
| `brightness` | BrightnessSensor                        | daylight through windows + electric light contribution |
| `airquality` | AirQualitySensor                        | CO₂ rises with occupancy |
| `enterleave` | EnterLeaveSensor                        | enter/leave totals as occupancy changes (whole-building when in `buildingDevices`) |
| `meter`      | Meter                                   | accumulating kWh from whole-building demand |
| `electric`   | Electric                                | instantaneous demand derived from lighting + FCU + base load |

`ratedPowerW` overrides a lighting/fcu device's full-load power (defaults: 60W per light, 400W
per FCU). `setPointC` sets an FCU's target temperature (default 21°C).

## Time acceleration

By default the simulation runs in real time, so values look right for the current time of day.
Set `timeMultiplier` above 1 to compress time — e.g. `288` runs a full 24-hour cycle every five
wall-clock minutes, which is useful for demos. State changes are clamped per tick so they stay
stable under large time steps.

## Example

A two-floor example lives at
[`example/config/sim-building/app.conf.json`](../../../example/config/sim-building/app.conf.json),
including zones so the data aggregates in the ops UI. It also configures `history` automations
and an ops dashboard (`ui-config.json`) so the simulated data is visible as power, occupancy,
air-temperature and air-quality charts. With `timeMultiplier: 288` a simulated day passes in
roughly five wall-clock minutes, so the history charts fill quickly.

History is stored in Postgres (matching the other example configs), so the example needs the
dev database running (`podman compose up -d`) — see the project README for details.

## Forcing values

Lighting, FCU and occupancy devices expose the `MockDeviceApi` (`smartcore.bos.mock.v1`), the same
service the mock driver uses, so you can drive the simulation live:

- `ForceTraitValue` sets a room input — occupancy (people count), air-temperature set point,
  lighting level or fan speed. Unlike the mock driver, the forced value becomes a **simulation
  input**: the engine keeps running and the rest of the building responds to it (forcing a room's
  occupancy still drives its lighting, FCU load, CO2, metered energy and so on).
- `SetDeviceAutomation` toggles whether the engine drives that input: `active=false` freezes it at
  its current value, `active=true` releases it back to the simulation.

Derived outputs (electrical demand, metering, brightness, air quality, motion, enter/leave) follow
from those inputs and cannot be forced directly — `ForceTraitValue` rejects them and points at the
input that drives them.

## Notes

- The simulation owns all device state and recomputes it every tick; forced values are inputs to
  that model (see above) rather than overwrites of a device's output.
- The random source is seeded (config `seed`, default fixed) so runs are reproducible. The
  optional `healthCheck` fault simulation is intentionally not seeded — fault timing varies
  from run to run.
- The optional `metadata` block is announced on the driver's own `name` as a device.
