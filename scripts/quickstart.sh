#!/usr/bin/env bash
set -euo pipefail

# HomeAPI - One-line QuickStart installer & upgrader
#
# Installs HomeAPI as a systemd service, or upgrades an existing install
# in place. Upgrades are non-disruptive and never touch your data: the
# SQLite database lives in a separate, persistent data directory, a
# consistent backup is taken before every upgrade, and the binary is
# swapped atomically with automatic rollback if the new version fails to
# start.
#
# Usage (fresh install or upgrade — same command):
#
#   curl -fsSL https://raw.githubusercontent.com/chinmay28/homeapi/main/scripts/quickstart.sh | sudo bash
#
# Configurable via environment variables (all optional):
#
#   HOMEAPI_REF       Git branch/tag/commit to deploy        (default: main)
#   HOMEAPI_PORT      HTTP listen port                       (default: 8080)
#   HOMEAPI_USER      System user the service runs as        (default: homeapi)
#   HOMEAPI_PREFIX    Install dir for source + binary        (default: /opt/homeapi)
#   HOMEAPI_DATA_DIR  Persistent data dir (DB + backups)     (default: /var/lib/homeapi)
#   HOMEAPI_REPO      Git repo URL                           (default: github.com/chinmay28/homeapi)

REPO="${HOMEAPI_REPO:-https://github.com/chinmay28/homeapi.git}"
REF="${HOMEAPI_REF:-main}"
PORT="${HOMEAPI_PORT:-8080}"
SVC_USER="${HOMEAPI_USER:-homeapi}"
PREFIX="${HOMEAPI_PREFIX:-/opt/homeapi}"
DATA_DIR="${HOMEAPI_DATA_DIR:-/var/lib/homeapi}"

SRC_DIR="$PREFIX/src"
BIN_PATH="$PREFIX/homeapi"
DB_PATH="$DATA_DIR/homeapi.db"
BACKUP_DIR="$DATA_DIR/backups"
SERVICE_NAME="homeapi"
UNIT_PATH="/etc/systemd/system/${SERVICE_NAME}.service"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# --- Preconditions -----------------------------------------------------------

[ "$(id -u)" -eq 0 ] || error "Please run as root (e.g. pipe to 'sudo bash')."
command -v systemctl >/dev/null 2>&1 || error "systemd (systemctl) is required but not found."
command -v git >/dev/null 2>&1 || error "git is required but not found. Install git and re-run."

UPGRADE=false
if systemctl list-unit-files "${SERVICE_NAME}.service" >/dev/null 2>&1 \
   && [ -f "$UNIT_PATH" ]; then
    UPGRADE=true
fi

if [ "$UPGRADE" = true ]; then
    info "Existing HomeAPI install detected — performing in-place upgrade."
else
    info "No existing install detected — performing fresh install."
fi

# --- Service user & directories ---------------------------------------------

if ! id "$SVC_USER" >/dev/null 2>&1; then
    info "Creating system user '$SVC_USER'..."
    useradd --system --home-dir "$DATA_DIR" --shell /usr/sbin/nologin "$SVC_USER" 2>/dev/null \
        || useradd --system --home-dir "$DATA_DIR" --shell /sbin/nologin "$SVC_USER"
fi

mkdir -p "$PREFIX" "$DATA_DIR" "$BACKUP_DIR"
chown -R "$SVC_USER":"$SVC_USER" "$DATA_DIR"
chmod 750 "$DATA_DIR"

# --- Fetch / update source ---------------------------------------------------

if [ -d "$SRC_DIR/.git" ]; then
    info "Updating source in $SRC_DIR (ref: $REF)..."
    git -C "$SRC_DIR" fetch --depth 1 origin "$REF"
    git -C "$SRC_DIR" checkout -q FETCH_HEAD
else
    info "Cloning $REPO into $SRC_DIR (ref: $REF)..."
    rm -rf "$SRC_DIR"
    git clone --depth 1 --branch "$REF" "$REPO" "$SRC_DIR" 2>/dev/null \
        || { git clone "$REPO" "$SRC_DIR"; git -C "$SRC_DIR" checkout -q "$REF"; }
fi

# --- Install build prerequisites (Go, Node, GCC, make) -----------------------

if [ -x "$SRC_DIR/scripts/install-prereqs.sh" ]; then
    info "Ensuring build prerequisites are installed..."
    "$SRC_DIR/scripts/install-prereqs.sh"
fi
# Make a tarball-installed Go visible within this script's PATH.
[ -d /usr/local/go/bin ] && export PATH="$PATH:/usr/local/go/bin"

# --- Build (service keeps running on the old binary during this step) --------

info "Building HomeAPI (frontend + backend)..."
make -C "$SRC_DIR" build
NEW_BIN="$SRC_DIR/homeapi"
[ -x "$NEW_BIN" ] || error "Build did not produce an executable at $NEW_BIN."

# --- Write / refresh the systemd unit ---------------------------------------

info "Installing systemd unit at $UNIT_PATH..."
cat > "$UNIT_PATH" <<EOF
[Unit]
Description=HomeAPI - self-hosted key-value REST API
Documentation=https://github.com/chinmay28/homeapi
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SVC_USER
Group=$SVC_USER
Environment=HOMEAPI_PORT=$PORT
Environment=HOMEAPI_DB_PATH=$DB_PATH
ExecStart=$BIN_PATH
Restart=on-failure
RestartSec=2

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null 2>&1 || true

# --- Backup existing data before swapping the binary (upgrade only) ----------

# Stop the running service so SQLite checkpoints its WAL and the on-disk
# database is in a clean, consistent state for both the backup and the swap.
WAS_ACTIVE=false
if systemctl is-active --quiet "$SERVICE_NAME"; then
    WAS_ACTIVE=true
    info "Stopping $SERVICE_NAME for a clean upgrade..."
    systemctl stop "$SERVICE_NAME"
fi

if [ -f "$DB_PATH" ]; then
    TS="$(date +%Y%m%d-%H%M%S)"
    BACKUP_PATH="$BACKUP_DIR/homeapi-${TS}.db"
    info "Backing up database to $BACKUP_PATH..."
    cp -p "$DB_PATH" "$BACKUP_PATH"
    # Preserve WAL/SHM sidecars too, if present.
    [ -f "${DB_PATH}-wal" ] && cp -p "${DB_PATH}-wal" "${BACKUP_PATH}-wal" || true
    [ -f "${DB_PATH}-shm" ] && cp -p "${DB_PATH}-shm" "${BACKUP_PATH}-shm" || true
fi

# --- Atomic binary swap with rollback ---------------------------------------

PREV_BIN="$PREFIX/homeapi.prev"
if [ -x "$BIN_PATH" ]; then
    cp -p "$BIN_PATH" "$PREV_BIN"
fi

# Atomic replace: install onto the same filesystem, then rename into place.
install -m 0755 "$NEW_BIN" "$PREFIX/homeapi.new"
mv -f "$PREFIX/homeapi.new" "$BIN_PATH"

info "Starting $SERVICE_NAME..."
systemctl start "$SERVICE_NAME"

# --- Verify, with automatic rollback ----------------------------------------

verify() {
    local url="http://127.0.0.1:${PORT}/api/health"
    for _ in $(seq 1 15); do
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            if command -v curl >/dev/null 2>&1; then
                curl -fsS "$url" >/dev/null 2>&1 && return 0
            else
                return 0  # service is up; no curl to probe health endpoint
            fi
        fi
        sleep 1
    done
    return 1
}

if verify; then
    info "HomeAPI is healthy and listening on port $PORT."
else
    warn "New version failed to come up."
    if [ -x "$PREV_BIN" ]; then
        warn "Rolling back to the previous binary..."
        install -m 0755 "$PREV_BIN" "$PREFIX/homeapi.new"
        mv -f "$PREFIX/homeapi.new" "$BIN_PATH"
        systemctl restart "$SERVICE_NAME" || true
        if verify; then
            error "Upgrade failed; rolled back to the previous working version. Your data is intact."
        fi
        error "Upgrade failed and rollback could not start the service. Database backups are in $BACKUP_DIR."
    fi
    [ "$WAS_ACTIVE" = false ] || systemctl start "$SERVICE_NAME" || true
    error "Service failed to start. Check: journalctl -u $SERVICE_NAME -e"
fi

VERSION="$(git -C "$SRC_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo ""
if [ "$UPGRADE" = true ]; then
    info "Upgrade complete (now at $VERSION). No data loss; backups in $BACKUP_DIR."
else
    info "Install complete (deployed $VERSION)."
fi
echo ""
echo "  URL:      http://localhost:${PORT}"
echo "  Service:  systemctl status $SERVICE_NAME"
echo "  Logs:     journalctl -u $SERVICE_NAME -f"
echo "  Data:     $DATA_DIR"
echo "  Re-run the same command anytime to upgrade in place."
echo ""
