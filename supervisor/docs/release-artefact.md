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

Building an artefact
--------------------

`scripts/build-release.sh` builds an artefact from any BOS main package. It builds the binary for FreeBSD with
the version stamped in, builds the Ops UI from the same sc-bos version, writes the manifest, and packs them with
`tar`. It writes `sc-bos-<version>-freebsd-<arch>.tar.gz` and a matching `.sha256` file in `sha256sum` format.

In this repo, build `./cmd/bos` for FreeBSD on amd64:

```bash
bash scripts/build-release.sh --version v2026.10.1
```

In a product module that imports sc-bos, run the script from the sc-bos module source, so the script matches
the module's sc-bos version. Download the module first, because `go list` prints no folder for a module that
isn't in the module cache yet:

```bash
go mod download github.com/smart-core-os/sc-bos
bash "$(go list -m -f '{{.Dir}}' github.com/smart-core-os/sc-bos)/scripts/build-release.sh" --main ./cmd/my-product
```

Module cache files aren't executable, so run the script with `bash`.

| Option             | Meaning                                                                                       |
|--------------------|-----------------------------------------------------------------------------------------------|
| `--main PKG`       | Go main package to build. Default `./cmd/bos`.                                                |
| `--module-dir DIR` | Go module to build in. Default: the current folder.                                           |
| `--version V`      | Release version. Default: `git describe --tags --match 'v[0-9]*' --always --dirty` in the module. |
| `--arch ARCH`      | Target `GOARCH`. Default `amd64`.                                                             |
| `--ui-dist DIR`    | Use a prebuilt Ops UI dist folder instead of building one.                                    |
| `--out DIR`        | Output folder. Default `./.build/release`.                                                    |
| `--allow-dirty`    | Allow a build from uncommitted changes to tracked files.                                      |

The version must be a valid OCI tag. Without `--allow-dirty`, the script refuses a version ending in `-dirty`,
and refuses to build when the module's git work tree has uncommitted changes to tracked files.

The script stamps the version into both the binary and the Ops UI. When it builds the UI from a git work tree,
it uses only tracked files, so local files such as `ui/ops/.env.local` don't change the release.

Building the Ops UI needs `yarn`. Without it, build the UI elsewhere and pass `--ui-dist`.

The archive's timestamps come from `SOURCE_DATE_EPOCH` if it's set, otherwise from the module's last commit time.
