#!/usr/bin/env bash
# build-release.sh - build a FreeBSD release artefact: a version-stamped BOS binary and the matching Ops UI,
# packed as a tarball with a manifest. See supervisor/docs/release-artefact.md for the format.
#
# Works from any Go module that builds a BOS main package, including a project module that imports sc-bos:
#
#   bash path/to/sc-bos/scripts/build-release.sh --main ./cmd/my-product
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: build-release.sh [options]

  --main PKG         Go main package to build (default ./cmd/bos)
  --module-dir DIR   Go module to build in (default: current directory)
  --version V        Release version (default: git describe --tags --match 'v[0-9]*' --always --dirty
                     in the module)
  --arch ARCH        Target GOARCH (default amd64)
  --ui-dist DIR      Use a prebuilt Ops UI dist folder instead of building one
  --out DIR          Output folder (default ./.build/release)
  --allow-dirty      Allow a build from uncommitted changes to tracked files
  -h, --help         Show this help

Building the Ops UI needs yarn.
EOF
}

die() {
  echo "build-release.sh: $*" >&2
  exit 1
}

main_pkg="./cmd/bos"
module_dir="."
version=""
goarch="amd64"
ui_dist=""
out="./.build/release"
allow_dirty=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --main|--module-dir|--version|--arch|--ui-dist|--out)
      [[ $# -ge 2 ]] || die "$1 needs a value"
      case "$1" in
        --main) main_pkg="$2" ;;
        --module-dir) module_dir="$2" ;;
        --version) version="$2" ;;
        --arch) goarch="$2" ;;
        --ui-dist) ui_dist="$2" ;;
        --out) out="$2" ;;
      esac
      shift 2
      ;;
    --allow-dirty) allow_dirty=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; die "unknown argument: $1" ;;
  esac
done

# --- 1. resolve and validate options -------------------------------------------------------------
# Paths are made absolute, because later steps run in other folders.
module_dir="$(cd "$module_dir" && pwd)" || die "module dir not found"
if [[ -n "$ui_dist" ]]; then
  ui_dist="$(cd "$ui_dist" && pwd)" || die "UI dist folder not found"
  [[ -f "$ui_dist/index.html" ]] || die "$ui_dist has no index.html"
else
  command -v yarn >/dev/null || die "yarn is required to build the Ops UI; install it or pass --ui-dist"
fi
# The default ignores tags of the UI packages and Go submodules, such as ui/ops/v1.2.3.
if [[ -z "$version" ]]; then
  version="$(git -C "$module_dir" describe --tags --match 'v[0-9]*' --always --dirty)" || die "can't get a version from git; pass --version"
fi
# The OCI tag pattern from internal/util/ocitag.
[[ "$version" =~ ^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$ ]] || die "version $version is not a valid OCI tag"
[[ "$goarch" =~ ^[a-z0-9]+$ ]] || die "arch $goarch is not a GOARCH value"
if [[ "$allow_dirty" != true ]]; then
  if [[ "$version" == *-dirty ]]; then
    die "version $version has uncommitted changes; commit them or pass --allow-dirty"
  fi
  if git -C "$module_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1 &&
    [[ -n "$(git -C "$module_dir" status --porcelain --untracked-files=no)" ]]; then
    die "$module_dir has uncommitted changes; commit them or pass --allow-dirty"
  fi
fi

mkdir -p "$out"
out="$(cd "$out" && pwd)"

# Archive timestamps: SOURCE_DATE_EPOCH if set, otherwise the module's last commit time.
epoch="${SOURCE_DATE_EPOCH:-$(git -C "$module_dir" log -1 --format=%ct 2>/dev/null || true)}"

name="sc-bos-$version-freebsd-$goarch.tar.gz"
echo "==> building $name from $main_pkg in $module_dir"

# --- 2. staging folder ---------------------------------------------------------------------------
work="$(mktemp -d "${TMPDIR:-/tmp}/build-release.XXXXXX")"
trap 'rm -rf "$work"' EXIT
staging="$work/staging"
mkdir -p "$staging/bin" "$staging/ui"

# --- 3. binary -----------------------------------------------------------------------------------
echo "==> go build $main_pkg for freebsd/$goarch"
(
  cd "$module_dir"
  CGO_ENABLED=0 GOOS=freebsd GOARCH="$goarch" go build -trimpath \
    -ldflags "-X github.com/smart-core-os/sc-bos/pkg/app.buildVersion=$version" \
    -o "$staging/bin/sc-bos" "$main_pkg"
)

# --- 4. Ops UI -----------------------------------------------------------------------------------
if [[ -n "$ui_dist" ]]; then
  echo "==> copying Ops UI from $ui_dist"
  cp -R "$ui_dist" "$staging/ui/ops"
else
  sc_bos_dir="$(cd "$module_dir" && go list -m -f '{{.Dir}}' github.com/smart-core-os/sc-bos)"
  [[ -n "$sc_bos_dir" ]] || die "sc-bos isn't in the module cache; run go mod download github.com/smart-core-os/sc-bos"
  [[ -d "$sc_bos_dir/ui/ops" ]] || die "no Ops UI source in $sc_bos_dir; pass --ui-dist"
  echo "==> building Ops UI from $sc_bos_dir/ui"
  # Build in a writable copy, because the module cache is read-only. From a git work tree, copy only tracked
  # files, so local files such as ui/ops/.env.local don't change the release.
  mkdir -p "$work/ui"
  if git -C "$sc_bos_dir" rev-parse --is-inside-work-tree >/dev/null 2>&1 &&
    [[ -z "$(git -C "$sc_bos_dir" rev-parse --show-prefix)" ]]; then
    git -C "$sc_bos_dir" ls-files -z -- ui | (cd "$sc_bos_dir" && tar -cf - --null -T -) | (cd "$work" && tar -xf -)
  else
    (cd "$sc_bos_dir/ui" && tar -cf - --exclude node_modules .) | (cd "$work/ui" && tar -xf -)
  fi
  chmod -R u+w "$work/ui"
  rm -rf "$work/ui/ops/dist"
  (cd "$work/ui" && yarn install --frozen-lockfile)
  # The UI reports the same version as the binary.
  (cd "$work/ui/ops" && GIT_VERSION="$version" yarn build)
  cp -R "$work/ui/ops/dist" "$staging/ui/ops"
fi

# --- 5. manifest and tarball ---------------------------------------------------------------------
echo "==> packing $out/$name"
cat > "$staging/manifest.json" <<EOF
{
  "schemaVersion": 1,
  "version": "$version",
  "os": "freebsd",
  "arch": "$goarch",
  "entrypoint": "bin/sc-bos",
  "ui": {"ops": "ui/ops"}
}
EOF
[[ -z "$(find "$staging" -type l)" ]] || die "the staging folder holds symlinks, which the installer rejects"
find "$staging" -type d -exec chmod 755 {} +
find "$staging" -type f -exec chmod 644 {} +
chmod 755 "$staging/bin/sc-bos"
if [[ -n "$epoch" ]]; then
  stamp="$(date -u -r "$epoch" +%Y%m%d%H%M.%S 2>/dev/null || date -u -d "@$epoch" +%Y%m%d%H%M.%S)"
  find "$staging" -exec env TZ=UTC touch -t "$stamp" {} +
fi
# List entries in name order, so each directory comes before its contents, as the installer requires.
# COPYFILE_DISABLE stops macOS tar adding ._ metadata files.
(cd "$staging" && find manifest.json bin ui | LC_ALL=C sort |
  COPYFILE_DISABLE=1 tar -c --no-recursion --no-xattrs -T - -f -) | gzip -n > "$out/$name"

# --- 6. checksum ---------------------------------------------------------------------------------
if command -v sha256sum >/dev/null; then
  sha256=(sha256sum)
else
  sha256=(shasum -a 256)
fi
(cd "$out" && "${sha256[@]}" "$name") > "$out/$name.sha256"
checksum="$(cut -d ' ' -f 1 "$out/$name.sha256")"
size="$(wc -c < "$out/$name" | tr -d ' ')"

echo "==> done"
echo "artefact: $out/$name"
echo "size:     $size bytes"
echo "sha256:   $checksum"
