# bacnet-comm-test

A diagnostic tool for testing connectivity to a BACnet/IP device. It connects to a device, reads its basic properties, lists all objects with their present values, and optionally writes a value to a point.

## Usage

```
bacnet-comm-test <nic[:port]> <server[:port]> [device] [-write Type:instance=value] [-priority 1-16]
```

| Argument | Description |
|---|---|
| `nic[:port]` | Local network interface name and optional UDP port to bind (default port: 47808) |
| `server[:port]` | BACnet device IP address and optional port (default port: 47808) |
| `device` | BACnet device instance number (default: 4194303, the wildcard) |
| `-write Type:instance=value` | Write a value to an object's present-value after reading (optional) |
| `-priority 1-16` | BACnet write priority (default: 8 — Manual Operator) |

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

Write a float value to an analog object:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219 -write AnalogValue:1=21.5
```

Write a boolean value to a binary object:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219 -write BinaryValue:0=true
```

Write an integer value to a multi-state object:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219 -write MultiStateValue:2=3
```

Override the write priority:

```
bacnet-comm-test "WiFi:47809" 10.104.20.204:56923 393219 -write AnalogValue:1=21.5 -priority 16
```

### Write value types

The value type is inferred from the string:

| Value format | Go type | BACnet encoding | Typical use |
|---|---|---|---|
| `true` / `false` | `bool` | Boolean | Binary objects |
| Integer (no decimal) | `uint32` | Unsigned | Multi-state objects |
| Decimal (e.g. `21.5`, `18.0`) | `float32` | Real | Analog objects |

> **Important:** Analog objects (AnalogInput, AnalogOutput, AnalogValue) require a decimal value.
> Writing `18` will fail — use `18.0` instead.

### Write priority

BACnet priority 1 (highest) to 16 (lowest). The default is **8 (Manual Operator)**, which overrides most automated setpoints. Use a lower number (higher priority) if the write is being silently overridden by automation.

## Output

### Read-only

```
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

### With `-write`

After the object table, the write result and read-back value are printed:

```
Writing 22.0 to AnalogValue:1 (priority 8)...
Write succeeded
Read back AnalogValue:1 present-value = 22
```

Objects that do not support `Present-Value` (such as the Device object itself) are shown with `-`.
