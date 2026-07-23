## Paxton Net2 access control driver

Integrates a [Paxton Net2](https://www.paxton-access.com/) access control system with
Smart Core. Doors and cardholders are announced as devices carrying the `Access` trait
(last access attempt), and access events can optionally be exposed through the
`SecurityEvent` trait.

## How it talks to Net2

The driver uses two channels against the Net2 server (`baseUrl`):

- **REST API** (`api/v1/...`) — OAuth2 password grant for auth, then polls `users`, `doors`
  and `events`. Enabled by default; disable with `disablePolling`.
- **SignalR** (ASP.NET SignalR 2 over WebSocket) — live event streaming. Disabled by
  default; enable with `enableSignalR`.

At least one event source must be active when `enableSecurityEvents` is true (i.e. you
cannot set `disablePolling: true` and `enableSignalR: false` together).

## Set up

- The Net2 server must have its API enabled and an OAuth2 client registered (`clientId`).
- Create an operator account for the driver and place its password in a file readable by
  the building controller — referenced by `auth.passwordFile` (the password is never read
  from the config JSON).

## Configuration

| Field | Type | Notes |
|---|---|---|
| `baseUrl` | string | **Required.** Base URL of the Net2 server, e.g. `https://paxton.example.com`. |
| `auth.username` | string | Net2 operator username. |
| `auth.passwordFile` | string | **Required.** Path to a file containing the operator password. |
| `auth.grantType` | string | OAuth2 grant type. Defaults to `password`. |
| `auth.clientId` | string | OAuth2 client ID registered with the Net2 server. |
| `auth.scope` | string | OAuth2 scope. Defaults to `offline_access`. |
| `deviceNamePrefix` | string | Prefix for announced door names (`<prefix>/doors/<id>`). |
| `cardHolderPrefix` | string | Prefix for announced cardholder names (`<prefix>/cardholder/<id>`). |
| `doorsInterval` | duration | How often the door list is refreshed. Defaults to `5m`. |
| `eventsInterval` | duration | How often events are polled (when polling). Defaults to `5s`. |
| `cardsInterval` | duration | How often the cardholder list is refreshed. Defaults to `5m`. |
| `enableSecurityEvents` | bool | Announce the `SecurityEvent` trait. Off by default. |
| `securityEventsName` | string | Smart Core node name for security events. **Required when `enableSecurityEvents` is true.** |
| `disablePolling` | bool | Disable REST event polling. Polling is on by default. |
| `enableSignalR` | bool | Enable SignalR live event streaming. Off by default. |
| `seenEventsCleanupInterval` | duration | Dedup-cache sweep interval. Defaults to `1m`. |
| `seenEventsMaxAge` | duration | How long event IDs are retained for dedup. Defaults to `5m`. |
| `insecureSkipVerify` | bool | Skip TLS certificate verification. Development use only. |

### Example

```json
{
  "name": "paxton",
  "type": "paxton",
  "baseUrl": "https://paxton.example.com",
  "auth": {
    "username": "sc-bos",
    "passwordFile": "/etc/sc-bos/secrets/paxton-password",
    "clientId": "smart-core"
  },
  "deviceNamePrefix": "building/paxton",
  "cardHolderPrefix": "building/paxton",
  "enableSecurityEvents": true,
  "securityEventsName": "building/paxton/security-events"
}
```
