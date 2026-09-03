# Auto - UDMI

This auto implements (some of) the [UDMI](https://faucetsdn.github.io/udmi/) spec for data export and control, via MQTT. Drivers are responsible for conversion to [UDMI data structures](https://faucetsdn.github.io/udmi/gencode/docs/), and expose this by implementing the [UdmiService](../../../proto/udmi.proto). That same service also allows for the [config](https://faucetsdn.github.io/udmi/docs/messages/config.html) UDMI flow, which is how control is implemented.

Two settings bound how often pointset events publish: `minSendInterval` is the floor between two publishes on a topic, `heartbeatInterval` the ceiling on how long a source may stay silent.

## Heartbeat

Drivers publish a pointset only when its values change, so a device whose readings are steady goes silent — and because event topics are published unretained, it disappears from the broker entirely. A consumer then can't tell "unchanged" from "dead".

To close that gap the auto asks a source for a current message, via the `UdmiService.GetExportMessage` unary RPC, once it has been quiet for `heartbeatInterval` (default `4h`; set `"0s"` to disable). That RPC's contract is to collect data explicitly to return, so the message that comes back is a reading the driver actually took and stamped itself — the auto never replays or restamps anything. This mirrors the UDMI `pointset.sample_rate_sec` idea of emitting a sample periodically regardless of change.

Crucially, a driver with nothing to report answers `Unavailable` and the auto publishes nothing. Silence therefore still means dead: a heartbeat never asserts that a device was read when it wasn't.

The deadline is tracked per source, not per topic, and is reset by any pointset event that source publishes — so a heartbeat only ever arrives in place of traffic, never on top of it. Per source rather than per topic because `GetExportMessage` is addressed by source name: one call answers for the source as a whole. State and metadata don't reset the deadline; they are published retained, so the broker already holds the latest, and drivers re-announce them on every reconnect.

A source must implement `GetExportMessage` to be heartbeated. Today that means the BACnet merge driver; Steinel HPD, Xovis and HikCentral return `Unimplemented`, which permanently disarms the heartbeat for that source rather than being retried (see SCB-1441). Drivers that publish on a ticker regardless of change — OPC UA, HelvarNet light, Gallagher, mock — never go quiet, so the heartbeat doesn't apply to them.

## Minimum send interval

The same on-change publishing has the opposite failure mode. A driver's change detection is bit-exact, so a value that never settles counts as a change every time it is read — at the BACnet driver's default 10s poll period that's a publish per device every 10s, indefinitely. And because a pointset event carries the whole device, one restless point republishes every point on it.

`minSendInterval` puts a floor under that: at most one publish per pointset event topic per interval. A change arriving inside the interval isn't dropped, it's held, and the newest held payload for the topic goes out as soon as the interval expires. Consumers therefore see the current value at a bounded rate rather than a decimated sample of the changes. Intermediate values are lost, which is what a rate limit means — don't set this where every sample matters.

It defaults to `0`, which is off. Some details worth knowing:

- The first message on a topic always publishes on arrival; no consumer waits out an interval for a value that has never been sent.
- A released payload keeps the `timestamp` the source gave it. The reading was taken when it was reported, possibly most of an interval ago, and restamping it would misreport when it was observed.
- This is a floor on publishing, not a change-of-value deadband. It bounds how often a value may be reported, not how far it must move to be worth reporting, so a point that drifts by a hair still publishes once per interval.
- It applies to pointset event topics only. State and metadata are rare, retained, and describe the device rather than sampling it, so they publish on arrival.
- Set it below `heartbeatInterval`, or the intervals no longer mean what they say. A heartbeat's reply is a sample like any other and goes through the floor too, so it can't breach `minSendInterval`; if one arrives while a payload is held, the fresher reading supersedes the held one.

```json
{
  "name": "udmi", "type": "udmi",
  "heartbeatInterval": "4h",
  "minSendInterval": "5m",
  "qos": 1
}
```

Note that heartbeats are published at `qos` like any other event, and the default of `0` is at-most-once — a site relying on heartbeats to detect dead devices should set `"qos": 1` so the broker acknowledges them.
