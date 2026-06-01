# bacnet-comm-test

A diagnostic tool for testing connectivity to a BACnet/IP device. It connects to a device, reads its basic properties, lists all objects, and prints their present values.

## Usage

```
bacnet-comm-test <nic[:port]> <server[:port]> [device]
```

| Argument | Description |
|---|---|
| `nic[:port]` | Local network interface name and optional UDP port to bind (default port: 47808) |
| `server[:port]` | BACnet device IP address and optional port (default port: 47808) |
| `device` | BACnet device instance number (default: 4194303, the wildcard) |

### Choosing a local port

The default local port is 47808 (the standard BACnet port). If another process on your machine is already using that port — for example a BACnet simulator — you will receive no responses. Use a different local port to avoid the conflict:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219
```

### Finding the right interface

On Windows, use the adapter's friendly name as shown in `ipconfig`. To list all interfaces and their addresses, run:

```powershell
Get-NetAdapter | Select-Object Name, InterfaceDescription
```

On Linux, use the interface name from `ip addr` (e.g. `eth0`).

## Examples

Connect to a device on the standard BACnet port, using the wildcard device ID:

```
bacnet-comm-test eth0 192.168.1.100
```

Connect to a device on a non-standard port with a known device instance:

```
bacnet-comm-test eth0 192.168.1.100:56923 393219
```

Connect using a named Windows interface and a non-default local port:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219
```

## Output

```
2026/06/01 09:56:00 Auto-detected interface: WiFi
2026/06/01 09:56:00 Connecting to {0 0 6 [10 104 20 204 222 91] []}
2026/06/01 09:56:00 Device 393219  vendor=61440  maxApdu=1476
2026/06/01 09:56:00 Fetching object list...
2026/06/01 09:56:00 Fetching present values for 12 objects...

Type                           Instance   Name                                     Present Value
----------------------------------------------------------------------------------------------------
AnalogInput                    0          Zone Temperature                         21.5
AnalogInput                    1          Zone Humidity                            45.2
BinaryInput                    0          Occupancy                                true
Device                         393219     Room Simulator                           -
...

Total: 12 objects
```

Objects that do not support `Present-Value` (such as the Device object itself) are shown with `-`.
