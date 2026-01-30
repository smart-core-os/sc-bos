# Smart Core BACnet/IP driver

This package implements integration between BACnet/IP and Smart Core. The driver uses a
Vanti [fork of gobacnet](https://github.com/smart-core-os/gobacnet/tree/write) maintained on the `write` branch.

The driver interprets [config](config/root.go) and sets up connections to BACnet devices and associates BACnet objects
with Smart Core traits and properties.

There's a definitive sample config file in [config/testdata](config/testdata/sample.json5). The driver only supports
JSON (not json5), but the json5 helps with documenting the properties available.

**Sample BACnet driver config**

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

## BACnet - Smart Core Mapping

The driver does not make any assumptions about which objects implement which traits. If an object does map well to the
semantics of a trait then you can specify this in the `devices.objects.trait` property as shown in the example above. If
an Object->Trait mapping isn't implemented it can be added into the [adapt](adapt) package where each go file contains
mappings from that BACnet object type into relevant traits.

More complicated mappings from multiple objects to a single named trait are configured via the `traits` config property
and the [merge](merge) package.

The driver also publishes a non-Smart Core gRPC API described in [bacnet.proto](rpc/bacnet.proto) that provides low
level access to BACnet services like ReadProperty and WriteProperty against configured devices.

## BACnet - Destination Network Addressing

One project worked on, that uses this driver had the following setup:

```
controller-1/ DeviceID 1100
├─ fcu-1 DeviceID 1111
├─ fcu-2 DeviceID 1112
controller-2/ DeviceID 1200
├─ fcu-3 DeviceID 1211
├─ fcu-4 DeviceID 1212
```

The controllers were BACnet/IP devices, that connected to FCUs and other field devices over [BACnet MS/TP (RS485)](http://www.bacnetwiki.com/wiki/index.php?title=BACnet_MS/TP). The controllers were all on the BMS VLAN, and this driver was running on a separate SC VLAN with UDP traffic on port `0xBAC0` allowed to the BMS VLAN; broadcast traffic wasn't supported/set-up.

It was noticed that YABE on the BMS VLAN would discover all controllers and all FCUs, and be able to make requests to specific FCUs. However, from the SC VLAN, any request to a specific FCU device ID, sent to a controller IP address, would respond with an error `unknown-object`.

It was discovered that a destination network and address must also be specified in the request. This ends up in the [NPDU packet](http://www.bacnetwiki.com/wiki/index.php?title=Network_Layer_Protocol_Data_Unit), and can be inspected with Wireshark. In this particular case the information we need was (partially) encoded into the device IDs: FCU device ID `1211` was on destination network `50012`, and had address `11`; this could only be found out because we had YABE working on VLAN, else we'd have had to ask for extra information. Also, because they had their software auto-assign IDs, which made the pattern I suppose.

### TL;DR

For some BACnet requests you may need to set a destination network and address, as well as `IP:port`, and device ID. This is configured by the `Comm#Destination` field.
