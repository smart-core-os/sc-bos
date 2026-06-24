# Shared configuration for the supervisor update end-to-end test harness.
#
# Sourced by every script, on BOTH the Mac and inside the rocky-10 machine, so it must contain only
# variable definitions and pure-bash helpers — no calls to tools that exist on only one side.
#
# See README.md for the run order.

# --- topology ---------------------------------------------------------------
# cloudsim runs on the Mac bound to loopback. The macOS Application Firewall blocks the VM from
# reaching host services directly, so we use Apple container's documented host-redirect: a DNS domain
# created with `sudo container system dns create host.container.internal --localhost 203.0.113.113`
# installs a packet-filter rule mapping 203.0.113.113 -> the host's localhost. Processes in the
# machine (and the BOS container, via Network=host) reach cloudsim at that IP. Re-run the dns create
# after a host/machine restart (the pf rule does not survive a reboot).
MACHINE="rocky-10"                 # Apple container machine acting as the Rocky/Podman host
HOST_REDIRECT_IP="203.0.113.113"   # -> host localhost from inside the machine (see note above)
CLOUDSIM_PORT="8080"
CLOUDSIM_LISTEN="0.0.0.0:${CLOUDSIM_PORT}"   # binding includes loopback, which the redirect targets
BOSAPI_ROOT="http://${HOST_REDIRECT_IP}:${CLOUDSIM_PORT}"      # cloud root BOS+supervisor use (in machine)
CLOUDSIM_LOCAL="http://127.0.0.1:${CLOUDSIM_PORT}"            # cloudsim as seen from the Mac
MGMT="${CLOUDSIM_LOCAL}/api/v1/management"                   # management API, driven from the Mac

# --- paths (identical on Mac and machine via the /Users/mattr bind mount) ---
REPO="/Users/mattr/Code/smart-core-os/sc-bos"
SCRIPTS="${REPO}/supervisor/scripts"
CONF="${SCRIPTS}/conf"
BUILD="${SCRIPTS}/.build"          # gitignored scratch shared across the bind mount
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
COMMIT_DEADLINE="60s"        # supervisor rollback deadline (short, for fast tests)
GOARCH="arm64"               # the machine is aarch64

# Demo Ops UI login. Policy enforcement stays ON (production-like); these are deliberately weak
# credentials for a throwaway demo node, not a security posture. 03-install.sh hashes the password
# with cmd/pash and installs it as BOS's local account.
ADMIN_USER="admin"
ADMIN_PASSWORD="admin"

# --- machine-side install locations (the machine's own filesystem) ----------
SUP_INSTALL="/usr/local/bin/sc-bos-supervisor"
SUP_CONF_DIR="/etc/sc-bos-supervisor"
QUADLET_DIR="/etc/containers/systemd"
SYSTEMD_DIR="/etc/systemd/system"
BOS_DATA="/root/data"        # host dir bind-mounted into the BOS container as /data
BOS_USERS_FILE="/etc/sc-bos/users.json"   # host file bind-mounted over /cfg/users.json (demo login)
SUP_STATE_DIR="/var/lib/sc-bos-supervisor"

# --- helpers (Mac side only; harmless to define inside the machine) ---------

# run_in_machine SCRIPT [ARGS...] — execute a script file as root inside the machine.
run_in_machine() { container machine run -n "$MACHINE" -- sudo bash "$@"; }

# machine_sh CMD — execute a one-off shell command as root inside the machine.
machine_sh() { container machine run -n "$MACHINE" -- sudo bash -c "$1"; }
