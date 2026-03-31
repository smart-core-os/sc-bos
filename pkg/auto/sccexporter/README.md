## sccexporter automation 

The sccexporter automation provides a quick way of exporting trait data from devices to Smart Core Connect (SCC) via MQTT.
For all the traits provided in the config, the automation will discover all the devices which implement those traits,
grab the data at a given interval, and publish the data to SCC via MQTT.

### Supported Traits

The automation currently supports the following traits:
- **Meter** (`smartcore.bos.Meter`) - Energy/resource metering data
- **AirQualitySensor** (`smartcore.traits.AirQualitySensor`) - Air quality measurements
- **AirTemperature** (`smartcore.traits.AirTemperature`) - Temperature data
- **OccupancySensor** (`smartcore.traits.OccupancySensor`) - Occupancy information

### Data Collection

- **Trait Data**: Collected and published at the configured interval for all devices as separate resources (e.g., `meterReading`, `airQuality`)
- **Trait Info**: For traits that have info/support data (e.g., Meter), the info is collected once during startup and published as a separate resource (e.g., `meterReadingInfo`) alongside the data in each payload
- **Device Metadata**: Collected on startup and included as a separate trait (`smartcore.trait.Metadata`) at a configurable interval (default: every 100 data collection cycles)

### Payload Structure

Data is organized in a **nested structure**:
```
trait → resource → data
```

For example:
- `smartcore.bos.Meter` → `meterReading` → meter reading data
- `smartcore.bos.Meter` → `meterReadingInfo` → unit information
- `smartcore.traits.AirQualitySensor` → `airQuality` → air quality measurements

This structure separates data from metadata and allows for future extensibility.

## Message Structure

The `message` structure represents the data format published to the MQTT broker:

```go
type message struct {
    Agent     string    `json:"agent"`
    Device    Device    `json:"device"`
    Timestamp time.Time `json:"timestamp"`
}
```

### Fields

- **Agent** (`string`): Identifier for the agent/system sending the data (configured in automation settings)
- **Device** (`Device`): Contains device information and trait data or metadata
- **Timestamp** (`time.Time`): When the message was created

### Device Structure

```go
type TraitData map[string]json.RawMessage

type Device struct {
    Name string                   `json:"name"`
    Data map[trait.Name]TraitData `json:"data,omitempty"`
}
```

- **Name** (`string`): The unique Smart Core name of the device
- **Data** (`map[trait.Name]TraitData`): Nested map structure for trait data
  - **First level keys** are trait names of type `trait.Name` (e.g., "smartcore.bos.Meter", "smartcore.traits.AirQualitySensor")
  - **First level values** are `TraitData` maps containing resource data for that trait
  - **TraitData** (`map[string]json.RawMessage`): Maps resource names to their JSON-encoded data
    - **Keys** are resource names within the trait (e.g., "meterReading", "meterReadingInfo", "airQuality")
    - **Values** are `json.RawMessage` (byte arrays) containing JSON-encoded resource data
  - Protobuf messages are serialized using `protojson.Marshal`, which produces JSON with camelCase field names
  - Multiple traits can be included in a single message
  - Each trait can contain multiple resources (data and info objects)

### Data Messages

The `Data` map uses a **nested structure** where:
- **Level 1**: Trait name (e.g., "smartcore.bos.Meter")
- **Level 2**: Resource name within that trait (e.g., "meterReading", "meterReadingInfo")
- **Level 3**: JSON-encoded resource data

This structure provides:
- **Separation of concerns**: Data and info objects are kept separate
- **Extensibility**: Easy to add new resources to a trait without breaking existing consumers
- **Clarity**: The structure explicitly shows trait → resource → data

#### Resource Naming Conventions

- **Data resources**: Named after the resource type (e.g., `meterReading`, `airQuality`, `occupancy`, `airTemperature`)
- **Info/Support resources**: Append "Info" to the data resource name (e.g., `meterReadingInfo`)
- **Metadata**: Uses the resource name `metadata` under the special `smartcore.trait.Metadata` trait

#### Encoding Details

- **Protobuf Messages**: All trait data and info is encoded using `protojson.Marshal`, which produces JSON with:
  - camelCase field names (e.g., `carbonDioxideLevel` instead of `carbon_dioxide_level`)
  - RFC 3339 timestamps (e.g., `2025-11-27T10:30:00Z`)
  - Numeric enums as string names (e.g., `"OCCUPIED"` instead of `1`)

- **Meter Data**: Now uses separate resources:
  1. `meterReading`: Contains the `MeterReading` protobuf message (usage, produced, timestamps)
  2. `meterReadingInfo`: Contains the `MeterReadingSupport` protobuf message (units)
  - Both are serialized independently using `protojson.Marshal`

- **Multiple Traits**: A single message can contain data for multiple traits that the device implements
- **Metadata**: Device metadata is included at a configurable interval (default: every 100 data collection cycles) under the `smartcore.trait.Metadata` trait with resource name `metadata`

### Example: Message with Multiple Traits

```json
{
  "agent": "van/uk/brum/ugs/building/scc-exporter",
  "device": {
    "name": "van/uk/brum/ugs/sensors/multi-sensor-01",
    "data": {
      "smartcore.traits.AirQualitySensor": {
        "airQuality": {
          "carbonDioxideLevel": 450.5,
          "score": 75.5
        }
      },
      "smartcore.traits.AirTemperature": {
        "airTemperature": {
          "ambientTemperature": {
            "valueCelsius": 22.5
          }
        }
      },
      "smartcore.traits.OccupancySensor": {
        "occupancy": {
          "state": "OCCUPIED",
          "peopleCount": 5,
          "stateChangeTime": "2025-11-27T10:25:00Z"
        }
      }
    }
  },
  "timestamp": "2025-11-27T10:30:00Z"
}
```

### Example: Meter Message with Separate Data and Info

```json
{
  "agent": "van/uk/brum/ugs/building/scc-exporter",
  "device": {
    "name": "van/uk/brum/ugs/meters/elec-main",
    "data": {
      "smartcore.bos.Meter": {
        "meterReading": {
          "usage": 123.45,
          "produced": 67.89,
          "startTime": "2025-11-27T09:30:00Z",
          "endTime": "2025-11-27T10:30:00Z"
        },
        "meterReadingInfo": {
          "usageUnit": "kWh",
          "producedUnit": "kWh"
        }
      }
    }
  },
  "timestamp": "2025-11-27T10:30:00Z"
}
```

Note how `meterReading` and `meterReadingInfo` are now separate resources within the `smartcore.bos.Meter` trait, making it clear which fields are data vs. metadata about the data.

### Example: Message with Metadata

```json
{
  "agent": "van/uk/brum/ugs/building/scc-exporter",
  "device": {
    "name": "van/uk/brum/ugs/meters/elec-main",
    "data": {
      "smartcore.bos.Meter": {
        "meterReading": {
          "usage": 123.45,
          "produced": 67.89,
          "startTime": "2025-11-27T09:30:00Z",
          "endTime": "2025-11-27T10:30:00Z"
        },
        "meterReadingInfo": {
          "usageUnit": "kWh",
          "producedUnit": "kWh"
        }
      },
      "smartcore.trait.Metadata": {
        "metadata": {
          "name": "van/uk/brum/ugs/meters/elec-main",
          "appearance": {
            "title": "Main Electrical Meter",
            "description": "Building main power meter"
          },
          "location": {
            "floor": "B1",
            "zone": "electrical-room"
          }
        }
      }
    }
  },
  "timestamp": "2025-11-27T10:30:00Z"
}
```

### Notes

- **Nested Structure**: The `Data` field uses a two-level map:
  - Level 1: Trait names (e.g., "smartcore.bos.Meter", "smartcore.traits.AirQualitySensor")
  - Level 2: Resource names within each trait (e.g., "meterReading", "meterReadingInfo", "airQuality")
- **Resource Separation**: Data and info/metadata are kept as separate resources for clarity and extensibility
- **Protobuf Encoding**: All resources use `protojson.Marshal`, which produces:
  - camelCase field names (e.g., `carbonDioxideLevel`, `ambientTemperature`)
  - String representations for enums (e.g., `"OCCUPIED"`)
  - RFC 3339 formatted timestamps (e.g., `"2025-11-27T10:30:00Z"`)
- **Meter Data Structure**: 
  - `meterReading`: Contains the actual meter reading data (usage, produced, timestamps)
  - `meterReadingInfo`: Contains metadata about the readings (units)
  - This separation makes it clear which fields are measurements vs. metadata
- **Multiple Traits**: All traits from the same device are combined into a single message, appearing as separate entries in the `data` map
- **Metadata Trait**: Device metadata is included at a configurable interval (default: every 100 cycles) under the `smartcore.trait.Metadata` trait with resource name `metadata`
- **Extensibility**: The nested structure allows easy addition of new resources to existing traits without breaking consumers (e.g., could add `meterHistory` or `meterConfig` resources to the Meter trait in the future)

## Configuration

### Example Configuration

```json
{
  "type": "sccexporter",
  "traits": ["smartcore.bos.Meter", "smartcore.traits.AirQualitySensor"],
  "fetchTimeout": "5s",
  "mqtt": {
    "agent": "van/uk/site/building/exporter",
    "host": "ssl://mqtt.example.com:8883",
    "topic": "scc/data",
    "clientId": "scc-exporter-1",
    "clientCertPath": "/path/to/client.crt",
    "clientKeyPath": "/path/to/client.key",
    "caCertPath": "/path/to/ca.crt",
    "connectTimeout": "5s",
    "publishTimeout": "5s",
    "sendInterval": "*/15 * * * *",
    "metadataInterval": 100,
    "qos": 1
  }
}
```

### Configuration Options

#### General Settings

- **fetchTimeout** (duration, optional): Maximum time to wait for a single device's trait data fetch. If a device takes longer than this, the fetch is cancelled and that device is skipped for this cycle. This prevents slow or hanging devices from blocking the entire collection cycle. Default: "5s"

#### Mqtt Settings

- **agent** (string, required): Identifier for the agent/system sending the data
- **host** (string, required): MQTT broker address (e.g., "ssl://mqtt.example.com:8883")
- **topic** (string, required): MQTT topic to publish messages to
- **clientId** (string, required): Unique client identifier for MQTT connection
- **clientCertPath** (string, required): Path to client certificate file for TLS
- **clientKeyPath** (string, required): Path to client key file for TLS
- **caCertPath** (string, required): Path to CA certificate file for TLS
- **connectTimeout** (duration, optional): Timeout for connecting to MQTT broker, default: "5s"
- **publishTimeout** (duration, optional): Timeout for publishing to MQTT, default: "5s"
- **sendInterval** (schedule, optional): Cron schedule for data collection, default: "*/15 * * * *" (every 15 minutes)
- **metadataInterval** (int, optional): Include metadata every N data collection cycles, default: 100
- **qos** (int, optional): MQTT Quality of Service level (0, 1, or 2), default: 1



