# Auto - UDMI

This auto implements (some of) the [UDMI](https://faucetsdn.github.io/udmi/) spec for data export and control, via MQTT. Drivers are responsible for conversion to [UDMI data structures](https://faucetsdn.github.io/udmi/gencode/docs/), and expose this by implementing the [UdmiService](../../../proto/udmi.proto). That same service also allows for the [config](https://faucetsdn.github.io/udmi/docs/messages/config.html) UDMI flow, which is how control is implemented.

## Heartbeat

Drivers publish a pointset only when its values change, so a device whose readings are steady goes silent — and because event topics are published unretained, it disappears from the broker entirely. A consumer then can't tell "unchanged" from "dead".

To close that gap the auto asks a source for a current message, via the `UdmiService.GetExportMessage` unary RPC, once it has been quiet for `heartbeatInterval` (default `4h`; set `"0s"` to disable). That RPC's contract is to collect data explicitly to return, so the message that comes back is a reading the driver actually took and stamped itself — the auto never replays or restamps anything. This mirrors the UDMI `pointset.sample_rate_sec` idea of emitting a sample periodically regardless of change.

Crucially, a driver with nothing to report answers `Unavailable` and the auto publishes nothing. Silence therefore still means dead: a heartbeat never asserts that a device was read when it wasn't.

The deadline is tracked per source, not per topic, and is reset by any pointset event that source publishes — so a heartbeat only ever arrives in place of traffic, never on top of it. Per source rather than per topic because `GetExportMessage` is addressed by source name: one call answers for the source as a whole. State and metadata don't reset the deadline; they are published retained, so the broker already holds the latest, and drivers re-announce them on every reconnect.

A source must implement `GetExportMessage` to be heartbeated. Today that means the BACnet merge driver; Steinel HPD, Xovis and HikCentral return `Unimplemented`, which permanently disarms the heartbeat for that source rather than being retried (see SCB-1441). Drivers that publish on a ticker regardless of change — OPC UA, HelvarNet light, Gallagher, mock — never go quiet, so the heartbeat doesn't apply to them.

```json
{
  "name": "udmi", "type": "udmi",
  "heartbeatInterval": "4h",
  "qos": 1
}
```

Note that heartbeats are published at `qos` like any other event, and the default of `0` is at-most-once — a site relying on heartbeats to detect dead devices should set `"qos": 1` so the broker acknowledges them.
