# cloudsim

`cloudsim` is a development and testing tool for a cloud-based config and software-update distribution
system for Smart Core BOS. It allows you to test BOS's cloud connection functionality (implemented in
`internal/cloud`) without a real connection to Smart Core Connect.

This is only intended for development and testing purposes. It is not production-ready - it lacks
authentication/authorisation and hardening for operating on the public internet.

It distributes two kinds of payload to BOS nodes:

- **Config versions** — node-scoped binary configuration payloads, tracked via `config_deployments`.
- **Binary artefacts** — site- and platform-scoped BOS software packages (podman save tarballs),
  rolled out via `binary_deployments`.

It provides an HTTP JSON API. The API is implemented in `/internal/cloud/sim`; see the OpenAPI spec
for an endpoint list.

It also provides a basic web application for manual access to the data. Simply load the cloudsim
URL in a browser.

## Storage

Metadata is stored in an SQLite database (the `-data` path, default `cloudsim.db`). See
`/internal/cloud/sim/store` for the schema and access code.

Binary-artefact payloads are large, so they are stored as external files on disk rather than as
database BLOBs. They live in a directory beside the database file, named from its basename
(`foo/bar.db` -> `foo/bar-artefacts/`), with one file per artefact named by its id. The files are
streamed in and out, never buffered in memory in full.

Deleting an artefact (or deleting a site, which cascade-deletes its artefacts) removes only the
database row; the payload file is left on disk. Run with `-cleanup` to sweep the artefacts directory
on startup, deleting any payload file that no longer has a matching database row (plus any leftover
temp files from interrupted uploads).

## Deviations from Smart Core Connect behaviour
- Checksums (which are formated as algorithm-prefixed base64 strings) use MD5 in SCC, and SHA256
  in cloudsim. BOS supports both.
- Deployment IDs have a different format. BOS treats these as opaque strings, so this has no
  consequences.

## Usage notes
### Upgrading schema
Schema is upgraded automatically on boot. Older versions of cloudsim did not generate checksums for
config versions. Use the `-cleanup` flag to scan for and generate these on startup.