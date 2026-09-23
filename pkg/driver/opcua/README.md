# Smart Core OPC UA driver

This package implements integration between OPC UA and Smart Core. 
The driver uses the [gopcua](https://github.com/gopcua/opcua) to communicate with OPC UA servers.

## How it works

Everything in OPC UA is a node, the Nodes we are most interested in are the Variable Nodes.
These variables represent a read/writable value in the OPC UA server.
Each Variable Node has a NodeID, which is a unique identifier for the node in the server.
In the config, the device defines which variables it wants to subscribe to.
When the underlying value of the Variable Node changes, we get an event through the channel and act, 
depending on which traits this device has configured to support.
2 different devices can subscribe to the same nodeID without an issue. 

## Traits

In the config, each device configures which traits it supports. (see config/sample.json for an example)
Each trait has its own configuration, which is used to map the OPC UA Variable Node to the trait.

## Connecting to the server

The `conn` block says where the server is and how to authenticate against it.

| Field | Type | Notes |
|---|---|---|
| `endpoint` | string | **Required.** OPC UA server endpoint, e.g. `opc.tcp://server.example.com:4840`. |
| `subscriptionInterval` | duration | How often the server publishes subscription updates. Defaults to `5s`. |
| `clientId` | number | Client ID, unique within a server. A random one is generated when unset. |
| `auth.username` | string | OPC UA user to authenticate as. Omit the whole `auth` block to connect anonymously. |
| `auth.passwordFile` | string | **Required with `auth`.** Path to a file containing that user's password. A plaintext `password` in the config is rejected. |
| `security.policy` | string | Security policy short name: `None`, `Basic128Rsa15`, `Basic256`, `Basic256Sha256`, `Aes128Sha256RsaOaep` or `Aes256Sha256RsaPss`. Defaults to `Basic256Sha256` when `auth` is set, `None` otherwise. |
| `security.mode` | string | Message security mode: `None`, `Sign` or `SignAndEncrypt`. Defaults to `SignAndEncrypt` when `auth` is set, `None` otherwise. |
| `security.certFile` | string | Client X509 certificate. **Required for `Sign` and `SignAndEncrypt`.** |
| `security.keyFile` | string | RSA private key matching `certFile`. **Required for `Sign` and `SignAndEncrypt`.** |

With neither `auth` nor `security` the driver connects anonymously over an unsecured
channel, which is how it has always behaved, so existing configs keep working unchanged.

Setting `auth` defaults the channel to `Basic256Sha256`/`SignAndEncrypt`: we never pick an
unencrypted channel for a password on your behalf, so `auth` on its own is an error until
you supply `certFile` and `keyFile` too. If you really do want credentials over a plain
channel — a simulator on a bench, say — ask for it explicitly with
`"security": {"policy": "None", "mode": "None"}`. The driver logs a warning at connect when
it sends a password that way.

Two notes on how the connection is made once security is configured:

- The driver first calls `GetEndpoints` on `endpoint` and picks the endpoint matching the
  configured policy and mode, then dials **the URL that endpoint advertises**, because
  strict servers reject a session created against any other URL. When the advertised URL
  differs from the configured one it is logged at info level; a server advertising a
  hostname the controller cannot resolve is the usual cause of a connect failure here.
- That discovery call itself is unsecured, as the OPC UA spec requires of the discovery
  endpoint. A server hardened to secure discovery too will fail at this step.

### Authenticated example

```json
{
  "name": "opcua",
  "type": "opcua",
  "conn": {
    "endpoint": "opc.tcp://server.example.com:4840",
    "subscriptionInterval": "5s",
    "auth": {
      "username": "sc-bos",
      "passwordFile": "/etc/sc-bos/secrets/opcua-password"
    },
    "security": {
      "policy": "Basic256Sha256",
      "mode": "SignAndEncrypt",
      "certFile": "/etc/sc-bos/opcua/client.pem",
      "keyFile": "/etc/sc-bos/opcua/client.key"
    }
  },
  "devices": []
}
```

### Generating a client certificate

`Sign` and `SignAndEncrypt` need a client key pair. The certificate **must carry a URI
subject alternative name**: gopcua reads it and sends it as the client's application URI,
and servers check that against the session, so a certificate without one is rejected.

```bash
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
  -keyout client.key -out client.pem \
  -subj "/CN=sc-bos" \
  -addext "subjectAltName=URI:urn:sc-bos:opcua-client"
```

The key may be PKCS#1 (`openssl genrsa`) or PKCS#8 (`openssl genpkey`, and the default
above), PEM or DER — gopcua sniffs the content rather than trusting the file extension.

Most servers then need the certificate trusting before they will accept a session: copy
`client.pem` into the server's trusted-client store, or connect once and move the
certificate from its rejected list to its trusted list.
