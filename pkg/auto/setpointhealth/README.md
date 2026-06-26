# Auto - Set Point Health

This automation watches a device's measured value against its set point and raises a health check
when the measurement fails to track the set point for a sustained period. It automatically tracks
devices matching specified conditions and creates a fault-based health check for each device.

Unlike [`healthbounds`](../healthbounds), which checks a single value against fixed bounds and reacts
instantly, this automation compares **two** values (measured vs set point) and only reports a fault
once the deviation has persisted for a configured duration. This detects equipment that is failing
to do its job — for example a fan coil unit whose return-air temperature stays at 24 °C an hour
after its set point was lowered to 18 °C indicates a cooling fault, even though 24 °C is a perfectly
normal room temperature.

## How it works

For each matching device the automation:

1. Pulls the configured trait resource, reading both the measured value and the set point from the
   same message.
2. Computes the deviation `abs(measured - setPoint)`.
3. Starts an on-delay timer when the deviation exceeds `tolerance`.
4. Marks the health check abnormal (a fault) once the deviation has continuously exceeded the
   tolerance, against the same set point, for `duration`.
5. Clears the fault when the deviation returns within `tolerance`.

### Timer behaviour

- The countdown is **reset when the deviation returns within tolerance** — only a continuous breach
  trips the check.
- The countdown is **restarted when the set point changes** — the equipment is given a fresh window
  to reach the new target. Consequently, repeatedly adjusting the set point faster than `duration`
  never raises a fault from this countdown alone, because there is no stable target to converge on.
- An optional **`maxDuration` backstop** closes that gap: when set, the check goes abnormal once the
  deviation has stayed outside `tolerance` continuously for `maxDuration`, **regardless of how often
  the set point changes**. This catches equipment that never tracks while its set point is being
  churned. The backstop clock starts on the first breach and is cleared only by a return to
  tolerance — set point changes do not reset it. Leave `maxDuration` unset to disable it.
- Once a fault is raised, only a return to tolerance clears it; a later set point change cannot
  un-confirm a sustained fault.
- If the measured value or set point cannot be read (unset, or a connection problem), the check
  reports an unreliable state and the countdown is frozen — a data gap is never treated as a fault.

## Supported traits

The trait must expose both the measured value and the set point as fields of the same resource, and
be registered in the shared trait registry (`pkg/auto/internal/anytrait`). Suitable traits include:

- `smartcore.traits.AirTemperature` — `ambientTemperature.valueCelsius` vs
  `temperatureSetPoint.valueCelsius`
- `smartcore.bos.Temperature` — `measured.valueCelsius` vs `setPoint.valueCelsius`

## Configuration

- `devices` - device query conditions identifying which devices to monitor (matches the DevicesApi).
- `source` - the `trait`, optional `resource`, and the `measured` and `setPoint` field paths.
- `tolerance` - the maximum allowed absolute difference, in the value's native unit. Must be `> 0`.
- `duration` - how long the deviation must persist before a fault is raised. Must be `> 0`.
- `maxDuration` - optional absolute backstop. When set, a fault is raised once the deviation has
  persisted continuously for this long irrespective of set point changes. Must be `> duration`.
  Omit to disable.
- `check` - health check metadata (display name, description, impacts). Bounds are not used.

Example monitoring a fan coil unit's return-air set point tracking:

```json
{
  "type": "setpointhealth",
  "name": "site/autos/health/fcu-setpoint-tracking",
  "devices": [{
    "field": "metadata.traits",
    "matches": {"conditions": [{"field": "name", "stringEqual": "smartcore.traits.AirTemperature"}]}
  }],
  "source": {
    "trait": "smartcore.traits.AirTemperature",
    "measured": "ambientTemperature.valueCelsius",
    "setPoint": "temperatureSetPoint.valueCelsius"
  },
  "tolerance": 1.5,
  "duration": "1h",
  "check": {
    "displayName": "Return-air set point tracking",
    "description": "Unhealthy when measured temperature stays >1.5°C from set point for over an hour."
  }
}
```

## Notes

- Each matching device gets its own independent health check, created when the device appears and
  removed when it disappears.
- Field paths in `source.measured` and `source.setPoint` use camelCase, automatically converted to
  snake_case for protobuf.
