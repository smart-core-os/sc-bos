# Shared configuration for the supervisor update demo.
#
# Sourced by every script. Run directly on a Rocky Linux 10 host (see README.md for the run order and
# prerequisites); contains only variable definitions and pure-bash helpers.

# --- topology ---------------------------------------------------------------
# Everything runs on one box: cloudsim, the Supervisor (a host systemd service), and the BOS
# container. cloudsim binds all interfaces so it is reachable both on the host loopback (the host
# Supervisor and the management curls) and on the Podman bridge gateway (the BOS container).
#
# The Supervisor performs the artefact download, and cloudsim stamps the payload URL from the
# check-in request's Host header, so the in-container BOS and the host Supervisor must reach cloudsim
# at the SAME address. We use the Podman bridge gateway: it is the host's own address on the bridge,
# so the host reaches it directly and every container reaches it too. 03-install.sh discovers the
# gateway and builds the device-facing bosapi_root from it.
NODE_HOSTNAME="rocky-10"           # label for the cloudsim node (cosmetic)
CLOUDSIM_PORT="8080"
CLOUDSIM_LISTEN="0.0.0.0:${CLOUDSIM_PORT}"   # reachable on loopback and the bridge gateway
CLOUDSIM_LOCAL="http://127.0.0.1:${CLOUDSIM_PORT}"           # cloudsim as seen from the host
MGMT="${CLOUDSIM_LOCAL}/api/v1/management"                  # management API, driven from the host

# --- paths ------------------------------------------------------------------
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"   # repo root, from this file's location
DEMO="${REPO}/supervisor/demo"
CONF="${DEMO}/conf"
BUILD="${DEMO}/.build"          # gitignored scratch dir
STATE_ENV="${BUILD}/state.env"     # runtime ids/secrets captured by 02-cloudsim.sh
USERS_JSON="${BUILD}/users.json"   # generated local accounts (fixed demo password) for BOS
BOS_CTX="${BUILD}/bos-ctx"         # staged build context for the v1/v2 extension images
VANTI_UGS="${REPO}/example/config/vanti-ugs"   # base app/ui config the demo node ships
SUP_BIN="${BUILD}/sc-bos-supervisor"
CLOUDSIM_BIN="${BUILD}/cloudsim"
CLOUDSIM_DB="${BUILD}/cloudsim.db"
CLOUDSIM_PID="${BUILD}/cloudsim.pid"
CLOUDSIM_LOG="${BUILD}/cloudsim.log"

# --- image tags / versions --------------------------------------------------
IMAGE_REPO="localhost/smartcore/bos"
V1="v1"          # initial good version
V2="v2"          # update target (good) -> happy path COMPLETED
V2BAD="v2bad"    # update target that never commits -> auto-rollback to v2, FAILED

# --- knobs ------------------------------------------------------------------
PLATFORM="podman"            # cloudsim node + artefact platform (DB constraint: podman|freebsd)
COMMIT_DEADLINE="60s"        # supervisor rollback deadline (short, for a fast demo cycle)

# Demo Ops UI login. Policy enforcement stays ON (production-like); these are deliberately weak
# credentials for a throwaway demo node, not a security posture. 03-install.sh hashes the password
# with cmd/pash and installs it as BOS's local account.
ADMIN_USER="admin"
ADMIN_PASSWORD="admin"

# --- install locations (this host's filesystem) -----------------------------
SUP_INSTALL="/usr/local/bin/sc-bos-supervisor"
SUP_CONF_DIR="/etc/sc-bos-supervisor"
QUADLET_DIR="/etc/containers/systemd"
SYSTEMD_DIR="/etc/systemd/system"
BOS_DATA_VOLUME="sc-bos-data"   # named Podman volume mounted into the BOS container as /data
BOS_USERS_FILE="/etc/sc-bos/users.json"   # host file bind-mounted over /cfg/users.json (demo login)
SUP_STATE_DIR="/var/lib/sc-bos-supervisor"
