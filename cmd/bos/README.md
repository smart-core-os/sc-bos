Area Controller Command
=======================

This command runs a sample Area Controller instance. Area controllers are usually physically located near or in an area
of a building and provide features and functionality related to that area. For example they might connect to a local
lighting controller and provide out-of-hours power control for those lights.

The area controller is configured via command line arguments or via configuration files. There's two levels of
configuration, one to configure the features of the controller and another to configure the runtime. Think "which port
should https be served on" vs "configure a BACnet device on 1.2.3.4:47808".
See [pkg/app/sysconf](../../pkg/app/sysconf) for system config, and [pkg/app/appconf](../../pkg/app/appconf) for runtime
config.

The area controller looks for config files in the config dir (e.g. `.conf`), and stores local data in the data directory 
(usually `.data`), which includes local caches of data and any generated certificates. The data directory will be 
created on first run.

## Config Directory

- `system.json`, `system.conf.json` - System config for the controller, ports, features, db connections, etc
- `app.conf.json` - App config including drivers, automations, zones, etc.
- `tenants.json` - a json list of tenants and their hashed client secrets. Used by
  the [authn system](../../pkg/system/authn) as one option for how to validate credentials (client_id, client_secret)
- `users.json` - a json list of users and their hashed passwords. Used by the [authn system](../../pkg/system/authn) as
  one option for how to validate users credentials (username, password)

## Data Directory

### Secret files

- `foo-pass` and other secrets - Files containing passwords and other secrets used by the controller. For
  example `postgres-pass`, these files contain a single secret and should be provided by the environment the controller
  runs in - e.g. Docker Secrets.

### Certificates and TLS

All certificates and keys are encoded using PEM, keys are written using PKCS#8. Paths to any non-self-signed
certificates can be customised in the system config.

- `grpc.key.pem`, `grpc.cert.pem`, `grpc.roots.pem` - keypair used for grpc server and client connections. Incoming
  client connections are checked against `grpc.roots.pem`. The key and cert are also used during enrollment.
- `grpc-self-signed.cert.pem` - a self signed version of `grpc.cert.pem` created and used when the latter is not
  available. HTTPS also uses this certificate if no other option is available. `grpc.roots.pem` is ignored if using self
  signed certificates.
- `https.key.pem`, `https.cert.pem` - keypair used for https (including grpc-web) server connections.
- `hub-ca.key.pem`, `hub-ca.cert.pem`, `hub.roots.pem` - keypair and trust roots used by the enrollment manager to sign
  node certificates amd configure trust between other controllers. The certificate should be configured as a CA (have
  the CA flags). These files are used by the [hub system](../../pkg/system/hub).
    - `hub-self-signed-ca.key.pem`, `hub-self-signed-ca.cert.pem`, `hub-self-signed-ca.roots.pem` - self signed versions
      of the above, used when the hub system is enabled but no certificates are configured.

### Local data and caches

- `db.bolt` - local/non-critical data storage typically used by automations and drivers for persistent information
  like "last seen state"
- `enrollment/`
    - `enrollment.json` - data file generated upon enrollment
    - `root-ca.cert.pem` - Root CA for the Smart Core installation
    - `enrollment.cert.crt` - PEM encoded X.509 certificate for `grpc.key.pem` signed by the Root CA
- `cache/`
    - `publications/` - cache of management server publications, including configuration

## Building and Running

The area controller can be built, run, and tested using standard `go build`, `go run`, or `go test` commands

```shell
go run github.com/smart-core-os/sc-bos/cmd/bos
```
