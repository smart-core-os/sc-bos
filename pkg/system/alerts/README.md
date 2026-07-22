# Alerts System

The alerts system manages a collection of alerts, these are records that indicate a change to the controller that can be
acknowledged by an operator.

**Example alerts:**

- Toilet paper is low in cubical 2
- Unable to communicate with BMS subsystem
- Power has been restored to floor 6

The alert system as implemented is backed by a database holding the alerts themselves, one row per alert. The alert
system will setup the relevant tables in the database when started.

## Alert metadata

Alerts can contain some metadata, typically to describe where the alert originated. This metadata is fairly unstructured
including the fields `floor`, `zone`, and `source` which are intended to aid with filtering in user interfaces. For
example to show only the alerts on floor 3, or that involve Meeting Room 2.

## Implementations

The primary implementation is backed by a postgres database as the alert system is supposed to run on the building
controller. As we try to allow any controller to run any system, you can setup an area controller to run this alert
backend by configuring the system to point to an existing postgres database:

```json5
// .data/area-controller-01/area-controller.local.json
{
  // other config
  "systems": {
    // other systems
    "alerts": {
      "storage": {
        "type": "postgres",
        "uri": "postgres://username@localhost:5432/smart_core",
        "passwordFile": "/secrets/postgres-password"
      }
    }
  }
}
```

The `storage` block accepts the same connection options as the shared node
store, including the optional `read`/`write`/`admin` sub-blocks for connecting
each privilege level as a distinct, least-privilege database user. See the
[stores README](../../app/stores/README.md) for details and suggested grants.

For development, you can use the same postgres database setup via the [docker-compose.yaml](../../../docker-compose.yml)
file located at the root of the project. See the [dev guide](../../../docs/install/dev.md) for more info.

You'll then need to update the username above to `postgres` and create a secrets file with the password in (
also `postgres`)
and update the file path above (relative to the project root).
