## Paxton Net2 access control driver

Integrates a [Paxton Net2](https://www.paxton-access.com/) access control system with
Smart Core. Doors and cardholders are announced as devices carrying the `Access` trait
(last access attempt), and access events can optionally be exposed through the
`SecurityEvent` trait. Cardholders can optionally carry the `AccessCredential` trait, for
managing their cards, fobs and other tokens.

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
| `enableCredentialManagement` | bool | Announce the `AccessCredential` trait on every cardholder. Off by default. See [Managing credentials over gRPC](#managing-credentials-over-grpc). |
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

## Managing credentials over gRPC

With `enableCredentialManagement: true`, every cardholder (`<cardHolderPrefix>/cardholder/<id>`)
also serves the generic `smartcore.bos.AccessCredential` trait:
`smartcore.bos.accesscredential.v1.AccessCredentialApi` (Get, List, Create, Update and Delete
credentials) and `AccessCredentialInfo` (DescribeCredential). Each credential is one Net2
user token, and its `id` is the Net2 token ID.

- `DescribeCredential` lists every Net2 `TokenType` except `Unspecified` as a credential
  `type`, with a generic `kind` (card, fob, vehicle plate, phone number). Values are always
  supplied by the caller.
- Only `type`, `value` and `state` can be written. Net2 has nowhere to store validity times,
  issue levels, invitations or `more`, so setting any of them is `InvalidArgument`.
- `state` maps onto `IsLost`: `LOST` sets it, `ACTIVE` (or unset) clears it. Other states are
  `InvalidArgument`.
- Net2 only supports replacing a whole token, so `UpdateCredential` reads the token, applies
  `update_mask` to it, and writes it all back.
- Net2 errors map to gRPC codes: 404 is `NotFound`, 400 is `InvalidArgument`, and anything
  else is `Unavailable`. A value that's already issued is `AlreadyExists`, unless this
  cardholder already holds the same type and value (for example after a retried POST), in
  which case the existing credential is returned. Error messages never include the value or
  who holds it, and logs only include the token ID and type.
- A user created in Net2 only gets a device, and so this API, after the next cardholder
  refresh (`cardsInterval`, 5 minutes by default).
- Card numbers can be cloned, so the default policy
  (`pkg/auth/policy/default/smartcore.bos.accesscredential.v1.rego`) denies this trait to
  viewers and to tenant `trait:read`/`trait:write` permissions. Admins, commissioners,
  operators and valid certificates are allowed.

## Managing user tokens (Go)

Net2 calls a user's cards, fobs and other credentials *tokens*. Code that imports this
package can manage them through `paxton.Client`, which the gRPC API above wraps.

Build a client with the same HTTP and TLS settings the driver uses. The password can be
set directly; no password file is needed:

```go
cfg := config.Root{
    BaseUrl: "https://paxton.example.com",
    Auth:    config.Auth{Username: "sc-bos", Password: pw, ClientId: "smart-core"},
}
client := paxton.NewClientFromConfig(cfg, logger, nil) // nil: no system check
```

Then, for the Net2 user `userID`:

```go
tokens, err := client.GetUserTokens(ctx, userID)
token, err := client.GetUserToken(ctx, userID, tokenID)
created, err := client.AddUserToken(ctx, userID, paxton.UserToken{TokenType: paxton.TokenTypeProxCard, TokenValue: "12345678"})
err = client.UpdateUserToken(ctx, userID, created.ID, paxton.UserToken{TokenType: created.TokenType, TokenValue: created.TokenValue, IsLost: true})
err = client.DeleteUserToken(ctx, userID, created.ID)
```

- The caller picks the `TokenType`. Every Net2 type has a `TokenType...` constant, and
  nothing is assumed for any particular kind of credential.
- `IsLost: true` disables the token but keeps its record in Net2. `DeleteUserToken`
  removes it entirely.
- An empty `TokenValue` or unknown `TokenType` is rejected before anything is sent.
- A non-2xx response from Net2 is a `*paxton.StatusError`; use `errors.As` to check
  `StatusCode`, for example 404 for an unknown user or token. A 400 or 404 doesn't mark
  the system check failed, since it means the request was wrong rather than Net2 being
  unhealthy.
- Net2 rejects a token value that's already issued with a 400
  (`Card ... has already been issued`). This is also what a caller sees if a POST is
  retried after Net2 created the token but replied with a 5xx, so on a 400 from
  `AddUserToken` it's worth checking `GetUserTokens` before treating the add as failed.
- Token values are credentials, so keep them out of logs. Net2 can echo the value in an
  error body, which ends up in `StatusError.Body`, so take care logging those errors too.
