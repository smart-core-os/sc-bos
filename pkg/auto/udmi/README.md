# Auto - UDMI

This auto implements (some of) the [UDMI](https://faucetsdn.github.io/udmi/) spec for data export and control, via MQTT. Drivers are responsible for conversion to [UDMI data structures](https://faucetsdn.github.io/udmi/gencode/docs/), and expose this by implementing the [UdmiService](../../../proto/udmi.proto). That same service also allows for the [config](https://faucetsdn.github.io/udmi/docs/messages/config.html) UDMI flow, which is how control is implemented.

## Heartbeat

Drivers publish a pointset only when its values change, so a device whose readings are steady goes silent — and because event topics are published unretained, it disappears from the broker entirely. A consumer then can't tell "unchanged" from "dead".

To close that gap the auto republishes a pointset event topic's last message once it has been quiet for `heartbeatInterval` (default `4h`; set `"0s"` to disable). The replayed payload is identical except for its `timestamp`, which is refreshed to the time of the heartbeat so ingest records the sample as observed now rather than hours ago. This mirrors the UDMI `pointset.sample_rate_sec` idea of emitting a sample periodically regardless of change.

Deadlines are tracked per topic and reset by any publish on that topic, so a heartbeat only ever arrives in place of traffic, never on top of it. Only pointset event topics are heartbeated: state and metadata are published retained, so the broker already holds the latest.

```json
{
  "name": "udmi", "type": "udmi",
  "heartbeatInterval": "4h",
  "qos": 1
}
```

Note that heartbeats are published at `qos` like any other event, and the default of `0` is at-most-once — a site relying on heartbeats to detect dead devices should set `"qos": 1` so the broker acknowledges them.
