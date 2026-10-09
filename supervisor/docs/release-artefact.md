Release artefact format
=======================

In this document: the release artefact for the Supervisor's `freebsdrc` installer, which updates BOS on FreeBSD.
It's unrelated to the container image that the `podman` installer loads on Linux. See `supervisor/internal/release`
for the code that reads and validates it.

A release artefact is a gzipped tarball holding a `manifest.json`, the BOS binary and the Ops UI. The cloud's
`sha256` checksum covers the tarball as a whole.

Tarball layout
--------------

```
manifest.json
bin/sc-bos
ui/ops/index.html
ui/ops/...            # the built ui/ops dist folder
```

- Compression is gzip. The archive holds directories and regular files only: no symlinks, hard links or device
  files.
- Entry names are relative, with no leading `/` and no `..` segments. A leading `./` is allowed.
- Every directory has its own entry, listed before its contents. Missing parent directories are not created.
- `bin/sc-bos` is the only entrypoint. Its name is fixed, whatever main package it was built from, so service
  scripts are the same across products. Its tar header must have the owner execute bit set.
- `ui/ops/` is required in schema version 1.
- The artefact carries no site configuration: no `system.conf.json`, `app.conf.json` or `ui-config.json`.

Manifest
--------

`manifest.json` is UTF-8 JSON at the root of the tarball:

```json
{
  "schemaVersion": 1,
  "version": "v2026.10.1",
  "os": "freebsd",
  "arch": "amd64",
  "entrypoint": "bin/sc-bos",
  "ui": { "ops": "ui/ops" }
}
```

| Field           | Rule                                                                                                                    |
|-----------------|-------------------------------------------------------------------------------------------------------------------------|
| `schemaVersion` | Must be `1`. Readers reject any other value.                                                                            |
| `version`       | Must be a valid OCI tag (`internal/util/ocitag`). Must equal the version stamped into the binary and the version the cloud offers. |
| `os`, `arch`    | The Go `GOOS` and `GOARCH` the binary was built for. Must not be empty.                                                 |
| `entrypoint`    | Must be `bin/sc-bos` in schema version 1.                                                                               |
| `ui.ops`        | Must be `ui/ops` in schema version 1.                                                                                   |

Readers reject unknown fields, so a typo fails loudly. The manifest still records the fixed values (`entrypoint`
and `ui.ops`), so a later schema version can relax them without changing the field set.

Before installing, the Supervisor checks that `version` matches the version it was asked to install, and that
`os` and `arch` match the host.
