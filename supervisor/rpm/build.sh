#!/usr/bin/env bash
# Build the sc-bos-supervisor RPM.
#
# Builds the Supervisor binary with its version baked in via -ldflags, then packages the prebuilt
# binary (plus the systemd unit and default config from supervisor/deploy/) with rpmbuild. Building
# the binary outside the rpmbuild sandbox avoids vendoring the whole monorepo into the build.
#
#   ./build.sh                  # version from `git describe`
#   VERSION=v2 ./build.sh       # explicit version (the demo uses this, mirroring the image tags)
#   OUT=/tmp/rpms ./build.sh    # where to copy the finished .rpm (default: supervisor/rpm/.out)
#
# Requires: go, rpmbuild (rpmdevtools / "RPM Development Tools").
set -euo pipefail

SPEC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SPEC_DIR/../.." && pwd)"

# VERSION is the version baked into the binary and reported to BOS (so a self-update can tell an
# upgrade from a rollback). RPM_VERSION is the same, sanitised to a valid RPM version field (no
# leading 'v', no '-').
VERSION="${VERSION:-$(cd "$REPO_ROOT" && git describe --tags --always --dirty)}"
RPM_VERSION="$(printf '%s' "$VERSION" | sed 's/^v//; s/-/./g')"
OUT="${OUT:-$SPEC_DIR/.out}"

TOP="$(mktemp -d)"
trap 'rm -rf "$TOP"' EXIT
mkdir -p "$TOP/SOURCES"

# BINARY, if set, packages that prebuilt file as the Supervisor binary instead of compiling. The demo
# uses it to build a deliberately-broken RPM (a binary that never serves the socket) to exercise the
# applier's auto-rollback.
if [[ -n "${BINARY:-}" ]]; then
  echo "==> packaging prebuilt binary $BINARY as sc-bos-supervisor (version $VERSION)"
  install -m 0755 "$BINARY" "$TOP/SOURCES/sc-bos-supervisor"
else
  echo "==> building sc-bos-supervisor binary (version $VERSION)"
  ( cd "$REPO_ROOT" && CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-X github.com/smart-core-os/sc-bos/supervisor/internal/version.Version=$VERSION" \
      -o "$TOP/SOURCES/sc-bos-supervisor" ./supervisor )
fi
cp "$REPO_ROOT/supervisor/deploy/sc-bos-supervisor.service" "$TOP/SOURCES/"
cp "$REPO_ROOT/supervisor/deploy/config.json" "$TOP/SOURCES/"

echo "==> rpmbuild (version $RPM_VERSION)"
rpmbuild --define "_topdir $TOP" --define "version $RPM_VERSION" \
  -bb "$SPEC_DIR/sc-bos-supervisor.spec"

mkdir -p "$OUT"
cp "$TOP"/RPMS/*/*.rpm "$OUT"/
echo "==> wrote:"
ls -1 "$OUT"/*.rpm
