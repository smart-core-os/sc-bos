#!/usr/bin/env bash
# 01-build.sh — build everything the test needs (run on the Rocky host).
#
# Builds the Supervisor binary (it runs as a host systemd service), then builds the production-like
# BOS base image from the repo's main Dockerfile (UI included), the v1/v2/v2bad images, and the
# v1/v2/v2bad artefact tarballs.
#
# Idempotent: existing outputs are reused unless FORCE=1 is set.
#   ./01-build.sh          # build only what's missing
#   FORCE=1 ./01-build.sh  # rebuild everything (incl. the full base image)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
source ./config.sh

FORCE="${FORCE:-0}"
mkdir -p "$BUILD"

# --- 1. build the Supervisor binary (host binary, runs as a systemd service) ----------------------
# pre:  Go toolchain on the host; the supervisor package compiles.
# post: $SUP_BIN is a static linux executable installed at /usr/local/bin by 03-install.sh.
if [[ "$FORCE" == 1 || ! -x "$SUP_BIN" ]]; then
  echo "==> building supervisor -> $SUP_BIN"
  ( cd "$REPO" && CGO_ENABLED=0 GOOS=linux go build -o "$SUP_BIN" ./supervisor )
else
  echo "==> supervisor binary present, skipping (FORCE=1 to rebuild)"
fi
# assert: the binary exists and is non-empty.
test -s "$SUP_BIN"

# --- 2. stage the demo node config (vanti-ugs, adapted to run DB-less) ----------------------------
# The deployed images ship the vanti-ugs example (mock devices + ops/cohort Ops UI), adapted so the
# node needs no Postgres: history automations write to sqlite, and the Ops UI uses local-auth login
# (required because policy enforcement stays on). Staged here rather than committed into conf/ to avoid
# duplicating the large example. post: $BOS_CTX is a self-contained build context for the extension.
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
echo "==> staging demo config from $VANTI_UGS -> $BOS_CTX"
rm -rf "$BOS_CTX"; mkdir -p "$BOS_CTX/ui-config"
# app.conf: vanti-ugs, with every history automation's postgres storage switched to sqlite (the only
# non-postgres storage the history automation supports besides bolt/memory; sqlite auto-opens at
# /data/history.sqlite3). The only "postgres" storage refs in app.conf are these history automations.
sed 's/"type": "postgres"/"type": "sqlite"/g' "$VANTI_UGS/app.conf.json" > "$BOS_CTX/app.conf.json"
# assert: no postgres storage remains.
! grep -q '"type": "postgres"' "$BOS_CTX/app.conf.json"
# system.conf: our DB-less overlay (supervisor + cloud + staticHosting + fileAccounts; no postgres
# systems). post: copied verbatim into the context.
cp "$CONF/system.conf.json" "$BOS_CTX/system.conf.json"
cp "$CONF/Containerfile.bos" "$BOS_CTX/Containerfile"
# ui-config: vanti-ugs (hub:true -> Cohort page, ops:true), but enable local-auth login so the UI
# authenticates under policy enforcement (vanti-ugs ships auth disabled for a keycloak/dev setup).
jq '.config.auth.disabled = false | .config.auth.providers = ["localAuth"]' \
  "$VANTI_UGS/ui-config.json" > "$BOS_CTX/ui-config/ui-config.json"
# assets referenced by ui-config as ./assets/... resolve relative to /__/scos/, so co-locate them.
cp -R "$VANTI_UGS/assets" "$BOS_CTX/ui-config/assets"
# assert: the context is self-contained.
test -f "$BOS_CTX/app.conf.json" && test -f "$BOS_CTX/ui-config/ui-config.json"

# --- 3. base image: the repo's main Dockerfile, UI included ---------------------------------------
# post: $IMAGE_REPO:base is the production-like image with the Ops UI baked in.
#       --ignorefile augments the repo .dockerignore with this harness's bulky non-input dirs so the
#       build context stays small (see conf/dockerignore).
if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:base"; then
  echo "==> podman build $IMAGE_REPO:base (main Dockerfile, UI included)"
  sudo podman build --ignorefile "$CONF/dockerignore" \
    -t "$IMAGE_REPO:base" -f "$REPO/Dockerfile" "$REPO"
else
  echo "==> $IMAGE_REPO:base exists, skipping (FORCE=1 to rebuild)"
fi
sudo podman image exists "$IMAGE_REPO:base"

# --- 4. good images: the base, two version tags --------------------------------------------------
# post: :v1 and :v2 extend the base, differing ONLY in the baked BOS_VERSION_OVERRIDE — so they get
#       distinct image ids, which is what lets the Supervisor tell a successful update from a rollback.
#       --no-cache is required: buildah keys the `ENV ...=$BOS_VERSION` layer without the build-arg
#       value, so a cached build would silently give both tags the first version's id.
test -f "$BOS_CTX/Containerfile"
for v in "$V1" "$V2"; do
  if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:$v"; then
    echo "==> podman build $IMAGE_REPO:$v"
    sudo podman build --no-cache --build-arg "BOS_VERSION=$v" -t "$IMAGE_REPO:$v" "$BOS_CTX"
  else
    echo "==> $IMAGE_REPO:$v exists, skipping"
  fi
done
# assert: both good images resolve AND bake distinct versions (guards against the cache trap above).
sudo podman image exists "$IMAGE_REPO:$V1"
sudo podman image exists "$IMAGE_REPO:$V2"
v1env=$(sudo podman image inspect --format '{{.Config.Env}}' "$IMAGE_REPO:$V1")
v2env=$(sudo podman image inspect --format '{{.Config.Env}}' "$IMAGE_REPO:$V2")
[[ "$v1env" == *"BOS_VERSION_OVERRIDE=$V1"* ]] || { echo "v1 image baked wrong version: $v1env" >&2; exit 1; }
[[ "$v2env" == *"BOS_VERSION_OVERRIDE=$V2"* ]] || { echo "v2 image baked wrong version: $v2env" >&2; exit 1; }

# --- 5. bad image: inert, never commits ----------------------------------------------------------
# post: :v2bad exists; loading + running it will never call Commit, forcing rollback under test.
if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:$V2BAD"; then
  echo "==> podman build $IMAGE_REPO:$V2BAD"
  sudo podman build -t "$IMAGE_REPO:$V2BAD" -f "$CONF/Containerfile.bad" "$CONF"
else
  echo "==> $IMAGE_REPO:$V2BAD exists, skipping"
fi
sudo podman image exists "$IMAGE_REPO:$V2BAD"

# --- 6. save the update artefacts ----------------------------------------------------------------
# post: $BUILD/{v1,v2,v2bad}.tar are podman-save tarballs (tags + metadata intact). `podman save` of a
#       tagged image preserves the tag, so the Supervisor's `podman load` restores
#       localhost/smartcore/bos:<version> exactly. Mode 0644 so cloudsim — which may run unprivileged —
#       can read the root-owned podman output.
# v1 is saved too so it can be deployed as an update target to reverse a v2 update (v1 is also the
# image 03-install.sh seeds as the initial :current).
for pair in "$V1:$BUILD/v1.tar" "$V2:$BUILD/v2.tar" "$V2BAD:$BUILD/v2bad.tar"; do
  tag="${pair%%:*}"; out="${pair#*:}"
  if [[ "$FORCE" == 1 || ! -s "$out" ]]; then
    echo "==> podman save $IMAGE_REPO:$tag -> $out"
    sudo rm -f "$out"   # podman save refuses to modify an existing docker-archive tar
    sudo podman save -o "$out" "$IMAGE_REPO:$tag"
    sudo chmod 0644 "$out"
  else
    echo "==> $out exists, skipping"
  fi
  test -s "$out"
done

echo "==> 01-build.sh done"
