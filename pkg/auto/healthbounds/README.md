# Auto - Health Bounds

This automation monitors device trait values and creates health checks when those values exceed normal operational
bounds. It automatically tracks devices matching specified conditions and creates bounds-based health checks for each
device.

## How it works

The healthbounds automation:

1. Watches for devices matching configured conditions
2. For each matching device, pulls updates from a specified trait resource
3. Extracts a value from the trait's data structure
4. Compares that value against configured bounds
5. Updates a health check based on whether the value is within normal range

This is useful for monitoring environmental conditions (like temperature), equipment status, or any numeric or
comparable value exposed by a device trait.

## Currently Supported Traits

**Note:** Only a limited set of traits are currently supported. See `internal/anytrait/registry.go` for the complete
list.

Currently supported traits include:

- `smartcore.traits.AirTemperature` - Monitor air temperature readings
- `smartcore.bos.EmergencyLight` - Monitor emergency light test results
- `smartcore.bos.Meter` - Monitor meter readings
- `smartcore.traits.OnOff` - Monitor on/off state

More traits can be added by registering them in the internal trait registry.

## Configuration

The configuration is split into three main sections:

- `devices` - A list of device query conditions to identify which devices to monitor. The query matches those in the
  DevicesApi.
- `source` - Specifies the trait and field path to extract the value to monitor. The auto may use both pull and get
  verbs based on device support.
- `check` - Defines the health check properties and bounds to evaluate. Limit configuration to metadata fields and
  bounds. Reliability and other state fields should be omitted.

Here is a basic example that monitors ambient temperature and create a health check when it goes outside the comfortable
range:

```json
{
  "type": "healthbounds",
  "name": "site/autos/health/comfortable-air-temperature",
  "devices": [{"field": "metadata.traits.name", "stringEqual": "smartcore.traits.AirTemperature"}],
  "source": {"trait": "smartcore.traits.AirTemperature", "value": "ambientTemperature.valueCelsius"},
  "check": {
    "displayName": "Ambient Temperature",
    "description": "Checks the ambient air temperature is within a comfortable range",
    "occupantImpact": "COMFORT",
    "bounds": {
      "normalRange": {
        "low": {"floatValue": 15.0},
        "high": {"floatValue": 25.0},
        "deadband": {"floatValue": 2.0}
      },
      "displayUnit": "°C"
    }
  }
}
```

## Notes

- Each device matching the query will have its own independent health check instance
- Health checks are automatically created when devices appear and removed when they disappear
- The automation handles connection reliability and will update health check reliability status accordingly
- Field paths in `source.value` use camelCase, which are automatically converted to snake_case for protobuf
