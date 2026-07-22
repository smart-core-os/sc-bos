# Stores

The `stores` package provides shared storage connections (databases) for the
systems and automations running on a controller. It is configured via the
top-level `stores` block in the controller config:

```json5
{
  // other config
  "stores": {
    "postgres": {
      "uri": "postgres://username@localhost:5432/smart_core",
      "passwordFile": "/secrets/postgres-password"
    }
  }
}
```

Systems that use postgres (`alerts`, `hub`, `tenants`, `publications`,
`history`) and the `history` automation borrow this shared pool by default. A
system can instead open its own dedicated connection by setting a `storage`
block with its own connection config; see that system's README.

## Read / write / admin roles (least privilege)

Each postgres query runs over a connection scoped to the minimum privilege it
needs:

- **read** — `SELECT` only
- **write** — `INSERT` / `UPDATE` / `DELETE` (and any transaction that both reads
  and writes)
- **admin** — schema setup (`CREATE` / `ALTER`) and maintenance (`VACUUM`)

By default all three roles use the single `uri`/`passwordFile` above, so a
single connection pool is opened (no behaviour change). To harden a deployment,
connect each role as a distinct database user by adding `read`, `write`, and/or
`admin` sub-blocks. Any role you omit falls back to the top-level config.

```json5
{
  "stores": {
    "postgres": {
      // admin defaults to the top-level config below unless overridden
      "uri": "postgres://sc_admin@localhost:5432/smart_core",
      "passwordFile": "/secrets/sc_admin-password",
      "read": {
        "uri": "postgres://sc_read@localhost:5432/smart_core",
        "passwordFile": "/secrets/sc_read-password"
      },
      "write": {
        "uri": "postgres://sc_write@localhost:5432/smart_core",
        "passwordFile": "/secrets/sc_write-password"
      }
    }
  }
}
```

The three roles are expected to point at the **same database** and differ only
in the database user (and therefore its privileges).

### Suggested grants

The `admin` role owns the schema (it runs `CREATE TABLE`/`ALTER` on startup and
`VACUUM`). Once the tables exist, grant the read/write roles the minimum they
need. For example, per table used by the enabled systems:

```sql
-- read role: SELECT only
GRANT SELECT ON <table> TO sc_read;

-- write role: DML (writers often need to read within a transaction too)
GRANT SELECT, INSERT, UPDATE, DELETE ON <table> TO sc_write;

-- admin role: owns the table (DDL) and can run VACUUM (needs ownership or the
-- MAINTAIN privilege on PostgreSQL 16+)
ALTER TABLE <table> OWNER TO sc_admin;
```

Tables with a serial/identity column (currently `history`, whose `id` is
`BIGSERIAL`) also need the write role to hold `USAGE` on the backing sequence —
`INSERT ... RETURNING id` calls `nextval()`, which fails with `permission denied
for sequence` otherwise. Grant it once for the schema so sequences created by
later migrations are covered too:

```sql
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sc_write;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO sc_write;
```

(UUID-keyed tables need nothing extra — `uuid_generate_v4()` is
`PUBLIC`-executable.)

### Connection sizing

Each role with a distinct config opens its own `pgxpool` pool, and each pool
defaults to a max of `max(4, numCPU)` connections. Splitting one shared pool
into `read`/`write`/`admin` therefore multiplies the peak connection count to
the database by up to three. Size each pool explicitly with the
`pool_max_conns` query parameter on its `uri` (e.g.
`postgres://sc_read@host/smart_core?pool_max_conns=4`) so the combined total
stays within the database's `max_connections`.
