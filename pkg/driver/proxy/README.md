# Smart Core Proxy Driver

This driver maps a local Smart Core Trait to a remote Smart Core Trait. You configure it by specifying some remote host
and port `10.100.1.100:23557`, the driver will attempt to inspect the remote node by querying the DeviceApi. 
Each device will be announced with the root node for the controller.

Config for this driver looks similar to this:

```json
{
  "type": "proxy", "name": "area-controller-01",
  "nodes": [
    {
      "host": ":23557",
      "tls": {
        "insecureSkipVerify": true,
        "insecureNoClientCert": true
      }
    }
  ]
}
```

The proxy server uses the controllers standard method of connecting to other Smart Core nodes. This means that if this
controller is enrolled with a manager then connections to other enrolled nodes will be trusted and those nodes will
trust this controller. When running with self-signed certificates this has the effect of never being able to connect
to another node as that node will not trust our self signed certificates.

You may configure the driver to not send a client cert - disabling mTLS - by setting the
`nodes.tls.insecureNoClientCert` property to `true`. Disabling the verification of server certificates is accomplished
by setting `nodes.tls.insecureSkipVerify` to `true`. These properties should not be used in production environments!

## OAuth 2 support (tenant proxying)

The proxy driver supports using access tokens to authenticate to the remote node.
Each node can be configured using an `oauth2` section to set the token endpoint
and credentials.
This is designed to be used with Smart Core's built-in OAuth 2 server, using the
tenant token system. The client credentials grant type is used.
When enabled, access tokens will automatically be fetched where necessary and refreshed
when expired.
The same TLS client configuration is used for connecting to the token server specified
in `tokenEndpoint` as the proxied gRPC server specified in `host`.

### Example Node Configuration

```json
{
  "host": "1.2.3.4:23557",
  "oauth2": {
    "tokenEndpoint": "https://1.2.3.4:8443/oauth2/token",
    "clientId": "foobarclientid",
    "clientSecretFile": "/run/secrets/client-secret"
  }
}
```