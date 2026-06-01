# Smart Core BACnet driver

This package integrates BACnet devices with Smart Core. It supports two data links:

- **BACnet/IP** (UDP) — the original transport.
- **BACnet/SC** (secure connect) — websocket-over-TLS to a hub, with mutual TLS, hub-routed VMAC addressing and failover. Set the `secureConnect` config block to switch.

NPDU/APDU encoding is shared between the two via a Vanti [fork of gobacnet](https://github.com/smart-core-os/gobacnet/tree/write) (the `write` branch). The driver depends on the [`bclient.Client`](bclient/bclient.go) interface so the same trait mapping, adapt and merge code drives both transports.

The driver interprets [config](config/root.go) and sets up connections to BACnet devices, associating BACnet objects with Smart Core traits and properties. A definitive sample config lives in [config/testdata/sample.json5](config/testdata/sample.json5) — the driver only parses JSON, but the json5 helps document the schema with comments.

## Quick start (BACnet/IP)

```json
{
  "type": "bacnet", "name": "MyDriverImpl",
  "devices": [
    {
      "id": 10002,
      "objects": [
        {"id": "BinaryValue:1", "trait": "smartcore.traits.OnOff"}
      ]
    }
  ],
  "traits": [
    {
      "name": "thermostat", "kind": "smartcore.traits.AirTemperature",
      "setPoint": {"device": 10002, "object": "AnalogInput:0"},
      "ambientTemperature": {"device": 10002, "object": "AnalogOutput:0"}
    }
  ]
}
```

## BACnet/SC (secure connect)

Setting `secureConnect` switches the driver to talk to a BACnet/SC hub over a secure websocket using mutual TLS instead of opening a UDP socket; `localInterface`/`localPort` are ignored. Devices are configured as usual but are normally located by Who-Is on their instance id (omit `comm.ip`) since the hub routes by VMAC, not IP.

Only the data link differs: NPDU/APDU encoding, trait mappings and adapt/merge are shared with BACnet/IP. The SC implementation (BVLC-SC framing, websocket/TLS transport, connect handshake, heartbeat, hub failover) lives in the [sc](sc) package; the driver selects between clients via [`bclient.Client`](bclient/bclient.go).

```json
{
  "type": "bacnet", "name": "MyDriverImpl",
  "secureConnect": {
    "primaryHubURI": "wss://hub.example.com:47808",
    "failoverHubURI": "wss://hub2.example.com:47808",
    "deviceUUID": "1b671a64-40d5-491e-99b0-da01ff1f3341",
    "tls": {
      "certificates": [{"certificate": "/run/secrets/bsc-cert.pem", "privateKey": "/run/secrets/bsc-key.pem"}],
      "rootCAs": "/run/secrets/bsc-ca.pem"
    }
  },
  "devices": [
    {"id": 10002, "objects": [{"id": "BinaryValue:1", "trait": "smartcore.traits.OnOff"}]}
  ]
}
```

Defaults: `deviceUUID` and `vmac` are random (and logged) when omitted; `maxBVLCLength` and `maxNPDULength` default to 1497; `heartbeatInterval` to 30s; `connectTimeout` to 10s. TLS 1.2 is enforced as the minimum version. Validation fails driver startup with an actionable message when `primaryHubURI` is missing, the scheme isn't `ws`/`wss`, or `wss` is requested without a client certificate / a CA (or `insecureSkipVerify`).

## Device metadata

Each device the driver announces carries metadata combining the operator's `device.metadata` with these driver-supplied extras:

- **`Appearance.Title`** — defaults to `device.title` (the operator-captured vendor/native name) and falls back to the announced device name. An operator-set Title in `device.metadata.appearance` is preserved.
- **`More["mqtt_topic"]`** — the base UDMI topic `<mqttTopicPrefix>/<deviceName>`, with **no suffix**. Consumers append `/events/pointset` etc themselves. Driven by the root `mqttTopicPrefix` config; omitted when that's empty.
- **`More["point_map"]`** — a JSON array catalogue of the MQTT points published for this device, one entry per point in any UDMI trait whose value source resolves to this device.

Operator-set `More` keys are never overwritten — the driver only fills entries that aren't already present.

Each `point_map` entry uses these stable field names (so consumers can parse every integration's point_map uniformly):

```
spec · mqtt · meaning · type · unit (omitempty) · access
```

Operators (or a generating pipeline) supply the rich fields via `pointSpecs` at driver root, keyed by **MQTT point name**:

```json
{
  "type": "bacnet",
  "mqttTopicPrefix": "ACME/BUILDING",
  "pointSpecs": {
    "supply-air-temp-c": {
      "spec":    "SAT",
      "meaning": "supply.air.temperature",
      "type":    "real",
      "unit":    "C",
      "access":  "read"
    }
  },
  "devices": [{"id": 10000, "name": "ahu-1", "title": "AHU-1"}],
  "traits": [{
    "name": "ahu-1-udmi", "kind": "udmi",
    "topicPrefix": "ACME/BUILDING/AHU-1",
    "points": {
      "supply-air-temp-c": {"device": "ahu-1", "object": "AnalogInput:0"}
    }
  }]
}
```

Points without a matching `pointSpec` still appear in `point_map`: `spec` defaults to the MQTT name, the descriptive fields stay empty, and `access` is inferred from the BACnet object type backing the point — `AnalogOutput`/`BinaryOutput`/`MultiStateOutput` and the `*Value` types resolve to `read/write`; everything else (including refs by object name where the type isn't known statically) resolves to `read`. The build logic lives in [pointmap](pointmap/pointmap.go).

## Commissioning tools

The [`cmd/tools/bacnet-sc/`](../../../cmd/tools/bacnet-sc) directory has single-shot CLIs that exercise a BACnet/SC hub directly, without standing up the driver:

| Tool | Purpose |
|---|---|
| `connect` | Dial the hub, complete the handshake, print the negotiated VMACs + UUID. Verifies URI, mutual-TLS certs and any allow-listing. |
| `whois`   | Broadcast Who-Is, print the I-Am responses. |
| `read`    | Read one property from a device located by Who-Is. |
| `write`   | Write one property (refuses without `-confirm`; refuses unsafe writes by default). |

All four share the same flag surface (`-hub`, `-cert`, `-key`, `-ca`, `-uuid`, `-vmac`, `-insecure`, `-timeout`) and accept `-json` for piping into other tooling.

For BACnet/IP there's [`cmd/tools/bacnet-whois`](../../../cmd/tools/bacnet-whois), [`bacnet-comm-test`](../../../cmd/tools/bacnet-comm-test) and [`bacnet-multi-comm-test`](../../../cmd/tools/bacnet-multi-comm-test); the multi tool supports an optional `-writes-file` JSON for executing WriteProperty operations alongside the read pass.

## BACnet - Smart Core Mapping

The driver does not make assumptions about which objects implement which traits. If an object maps well to a trait's semantics, specify it in `devices.objects.trait` as in the example above. If an Object→Trait mapping isn't implemented yet it can be added in the [adapt](adapt) package — each Go file there handles one BACnet object type's mapping into relevant traits.

More complex mappings from multiple objects to a single named trait are configured via the `traits` config property and the [merge](merge) package.

The driver also publishes a non-Smart Core gRPC API described in [bacnet.proto](rpc/bacnet.proto) that provides low-level access to BACnet services like ReadProperty and WriteProperty against configured devices.

## BACnet - Destination Network Addressing

One project that uses this driver had the following setup:

```
controller-1/ DeviceID 1100
├─ fcu-1 DeviceID 1111
├─ fcu-2 DeviceID 1112
controller-2/ DeviceID 1200
├─ fcu-3 DeviceID 1211
├─ fcu-4 DeviceID 1212
```

The controllers were BACnet/IP devices, connected to FCUs and other field devices over [BACnet MS/TP (RS485)](http://www.bacnetwiki.com/wiki/index.php?title=BACnet_MS/TP). The controllers were on the BMS VLAN; this driver ran on a separate SC VLAN with UDP traffic on port `0xBAC0` allowed to the BMS VLAN; broadcast traffic wasn't set up.

YABE on the BMS VLAN discovered all controllers and FCUs and could make requests to specific FCUs. From the SC VLAN, any request to a specific FCU device id sent to a controller IP responded with `unknown-object`. The fix was to specify the destination network and address in the request, which ends up in the [NPDU packet](http://www.bacnetwiki.com/wiki/index.php?title=Network_Layer_Protocol_Data_Unit) (inspectable with Wireshark). In this particular case the info was partially encoded in the device ids: FCU device id `1211` was on destination network `50012`, address `11`.

### TL;DR

For some BACnet requests you may need to set a destination network and address as well as `IP:port` and device id. Configure this via the `Comm#Destination` field.
