# cloudsim

`cloudsim` is a development and testing tool for a cloud-based config and software-update distribution
system for Smart Core BOS.

This is only intended for development and testing purposes. It is not production-ready — it lacks
authentication/authorisation and hardening for operating on the public internet.

It distributes two kinds of payload to BOS nodes:

- **Config versions** — node-scoped binary configuration payloads, tracked via `config_deployments`.
- **Update artefacts** — site- and platform-scoped BOS software packages (podman save tarballs),
  rolled out via `update_deployments`.

It provides an HTTP JSON API. The API is implemented in `/internal/cloud/sim`; see the OpenAPI spec
for an endpoint list.

## Storage

Metadata is stored in an SQLite database (the `-data` path, default `cloudsim.db`). See
`/internal/cloud/sim/store` for the schema and access code.

Update-artefact payloads are large, so they are stored as external files on disk rather than as
database BLOBs. They live in a directory beside the database file, named from its basename
(`foo/bar.db` → `foo/bar-artefacts/`), with one file per artefact named by its id. The files are
streamed in and out, never buffered in memory in full.

Deleting an artefact (or deleting a site, which cascade-deletes its artefacts) removes only the
database row; the payload file is left on disk. Run with `-cleanup` to sweep the artefacts directory
on startup, deleting any payload file that no longer has a matching database row (plus any leftover
temp files from interrupted uploads).
