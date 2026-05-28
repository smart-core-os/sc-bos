SC-BOS Command
===============

This is the main Smart Core Building Operating System (SC-BOS) executable. SC-BOS instances can be configured to run in
different roles depending on their location and purpose:

- **Area Controller** - physically located near or in an area of a building, connects to local devices and provides
  area-specific features (e.g., connecting to a local lighting controller)
- **Building Controller** - provides building-wide functionality like global automation, account management, and
  certificate authority services
- **Edge Gateway** - exposes external APIs and aggregates data from multiple controllers, typically without direct device
  integrations

The same `bos` binary is used for all roles - the behavior is determined by the configuration files.

## Configuration

The controller is configured via command line arguments and configuration files. There are two levels of configuration:

- **System config** (`system.conf.json`) - runtime settings like which ports to serve on, database connections, enabled
  systems
- **App config** (`app.conf.json`) - application-level settings like drivers, automations, zones

See [pkg/app/sysconf](../../pkg/app/sysconf) for system config documentation,
and [pkg/app/appconf](../../pkg/app/appconf) for app config documentation.

The controller looks for config files in the config directory (e.g. `.conf`), and stores local data in the data
directory (usually `.data`), which includes local caches and generated certificates. The data directory is created on
first run if it doesn't exist.

## Config Directory

- `system.json`, `system.conf.json` - System config for the controller: ports, features, database connections, etc.
- `app.conf.json` - App config including drivers, automations, zones, etc.
- `tenants.json` - a JSON list of tenants and their hashed client secrets. Used by
  the [authn system](../../pkg/system/authn) as one option for validating credentials (client_id, client_secret)
- `users.json` - a JSON list of users and their hashed passwords. Used by the [authn system](../../pkg/system/authn) as
  one option for validating user credentials (username, password)

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

SC-BOS can be built, run, and tested using standard `go build`, `go run`, or `go test` commands:

```shell
# Run with default settings
go run github.com/smart-core-os/sc-bos/cmd/bos

# Run with custom configuration
go run ./cmd/bos --appconf example/config/vanti-ugs/app.conf.json --sysconf example/config/vanti-ugs/system.conf.json --data .data/vanti-ugs

# Build a binary
go build -o sc-bos ./cmd/bos
```

See the [example configurations](../../example/config/) for working examples of different controller setups.

