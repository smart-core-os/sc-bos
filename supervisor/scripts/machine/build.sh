#!/usr/bin/env bash
# machine/build.sh (runs as root INSIDE rocky-10) — build the images and save two tarballs.
# Invoked by 01-build.sh; not run directly. Arg 1 is FORCE (1 = rebuild even if outputs exist).
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../config.sh"
FORCE="${1:-0}"

# --- base image: the repo's main Dockerfile, UI included -----------------------------------------
# pre:  the repo (Dockerfile + sources) is visible at $REPO via the bind mount.
# post: $IMAGE_REPO:base is the production-like image with the Ops UI baked in.
#       --network=host: the nested podman bridge (netavark/nftables) is unavailable, so RUN steps
#         (yarn/go) use the machine's working network instead.
#       --ignorefile: augments the repo .dockerignore with this env's bulky non-input dirs so the
#         build context stays small (see conf/dockerignore).
if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:base"; then
  echo "==> podman build $IMAGE_REPO:base (main Dockerfile, UI included)"
  sudo podman build --network=host --ignorefile "$CONF/dockerignore" \
    -t "$IMAGE_REPO:base" -f "$REPO/Dockerfile" "$REPO"
else
  echo "==> $IMAGE_REPO:base exists, skipping (FORCE=1 to rebuild)"
fi
sudo podman image exists "$IMAGE_REPO:base"

# --- good images: the base, two version tags -----------------------------------------------------
# pre:  $IMAGE_REPO:base exists; 01-build.sh staged the extension context (Containerfile + the
#       adapted vanti-ugs config) at $BOS_CTX.
# post: :v1 and :v2 extend the base, differing ONLY in the baked BOS_VERSION_OVERRIDE — so they get
#       distinct image ids, which is what lets the Supervisor tell a successful update from a rollback.
#       --no-cache is required: buildah keys the `ENV ...=$BOS_VERSION` layer without the build-arg
#       value, so a cached build would silently give both tags the first version's id.
test -f "$BOS_CTX/Containerfile"
for v in "$V1" "$V2"; do
  if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:$v"; then
    echo "==> podman build $IMAGE_REPO:$v"
    sudo podman build --no-cache --build-arg "BOS_VERSION=$v" \
      -t "$IMAGE_REPO:$v" "$BOS_CTX"
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

# --- bad image: inert, never commits -------------------------------------------------------------
# post: :v2bad exists; loading + running it will never call Commit, forcing rollback under test.
if [[ "$FORCE" == 1 ]] || ! sudo podman image exists "$IMAGE_REPO:$V2BAD"; then
  echo "==> podman build $IMAGE_REPO:$V2BAD"
  sudo podman build -t "$IMAGE_REPO:$V2BAD" -f "$CONF/Containerfile.bad" "$CONF"
else
  echo "==> $IMAGE_REPO:$V2BAD exists, skipping"
fi
sudo podman image exists "$IMAGE_REPO:$V2BAD"

# --- save the update artefacts -------------------------------------------------------------------
# pre:  the v2/v2bad images carry the tag the Supervisor will swap to :current after `podman load`.
# post: $BUILD/v2.tar and v2bad.tar are podman-save tarballs (tags + metadata intact) readable by the
#       Mac (mode 0644) for cloudsim to upload. `podman save` of a tagged image preserves the tag, so
#       `podman load` on the host restores localhost/smartcore/bos:<version> exactly.
# v1 is saved too so it can be deployed as an update target to reverse a v2 update (v1 is also the
# image 03-install.sh seeds as the initial :current).
for pair in "$V1:$BUILD/v1.tar" "$V2:$BUILD/v2.tar" "$V2BAD:$BUILD/v2bad.tar"; do
  tag="${pair%%:*}"; out="${pair#*:}"
  if [[ "$FORCE" == 1 || ! -s "$out" ]]; then
    echo "==> podman save $IMAGE_REPO:$tag -> $out"
    sudo rm -f "$out"   # podman save refuses to modify an existing docker-archive tar
    sudo podman save -o "$out" "$IMAGE_REPO:$tag"
    sudo chmod 0644 "$out"   # ensure the Mac (cloudsim) can read it regardless of mount-owner mapping
  else
    echo "==> $out exists, skipping"
  fi
  test -s "$out"
done

echo "==> machine/build.sh done"
