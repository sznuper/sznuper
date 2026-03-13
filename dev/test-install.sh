#!/usr/bin/env bash
# Test the install script on a fresh Debian 13 VPS.
#
# Usage:
#   ./dev/test-install.sh [--keep]
#
#   --keep  Leave the VPS running after tests (prints SERVER_ID/SERVER_IP)
#
# Requires: VERSION env var (e.g. v1.2.0)
# Requires: dev/.env with HETZNER_API_TOKEN and SSH_KEY (or env vars set directly)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

KEEP=0
for arg in "$@"; do
    case "$arg" in
        --keep)  KEEP=1 ;;
        *) echo "Unknown argument: $arg" >&2; exit 1 ;;
    esac
done

# shellcheck source=.env
[[ -f "$SCRIPT_DIR/.env" ]] && source "$SCRIPT_DIR/.env"

: "${HETZNER_API_TOKEN:?HETZNER_API_TOKEN must be set}"
: "${SSH_KEY:?SSH_KEY must be set}"
: "${VERSION:?VERSION must be set (e.g. v1.2.0)}"

log() { echo "==> $*"; }

SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR)

# ── Create VPS ───────────────────────────────────────────────────────────────

SERVER_ID=""
SERVER_IP=""

cleanup() {
    echo ""
    if [[ "$KEEP" -eq 0 ]]; then
        if [[ -n "$SERVER_ID" ]]; then
            log "Deleting server $SERVER_ID..."
            "$SCRIPT_DIR/delete-server.sh" "$SERVER_ID"
        fi
    else
        log "Server kept alive."
        echo "  SERVER_ID=$SERVER_ID"
        echo "  SERVER_IP=$SERVER_IP"
        echo ""
        echo "  To delete later: ./dev/delete-server.sh $SERVER_ID"
    fi
}
trap cleanup EXIT INT TERM

log "Creating VPS..."
# shellcheck source=create-server.sh
source <("$SCRIPT_DIR/create-server.sh")

SERVER_ID="${SERVER_ID:?create-server.sh did not set SERVER_ID}"
SERVER_IP="${SERVER_IP:?create-server.sh did not set SERVER_IP}"

ssh_run() { ssh "${SSH_OPTS[@]}" "root@$SERVER_IP" "$@"; }

# ── Run install script ───────────────────────────────────────────────────────

log "Uploading and running install.sh with VERSION=$VERSION..."
scp -q -O "${SSH_OPTS[@]}" "$REPO_DIR/install.sh" "root@$SERVER_IP:/root/install.sh"
ssh_run "VERSION=$VERSION sh /root/install.sh"

# init is skipped (no TTY over SSH) — create a minimal config so the daemon can start
log "Creating minimal config for testing..."
ssh_run "mkdir -p /etc/sznuper && cat > /etc/sznuper/config.yml" <<'CONF'
options:
  healthchecks_dir: /var/lib/sznuper/healthchecks
  cache_dir: /var/cache/sznuper

globals:
  hostname: e2e-test
CONF

log "Starting systemd service..."
ssh_run "systemctl start sznuper"
sleep 2

# ── Verify ───────────────────────────────────────────────────────────────────

log "Verifying installation..."
echo ""

PASS=0
FAIL=0

check() {
    local desc="$1"
    shift
    printf "  %-30s" "$desc"
    if ssh_run "$@" >/dev/null 2>&1; then
        echo "OK"
        PASS=$((PASS + 1))
    else
        echo "FAIL"
        FAIL=$((FAIL + 1))
    fi
}

check "binary exists"           "test -x /usr/local/bin/sznuper"
check "version matches"         "sznuper version | grep -qF '$VERSION'"
check "config exists"           "test -f /etc/sznuper/config.yml"
check "systemd unit installed"  "test -f /etc/systemd/system/sznuper.service"
check "systemd enabled"         "systemctl is-enabled sznuper"
check "systemd ran ok"          "systemctl show -p Result --value sznuper | grep -qF success"

if [[ $FAIL -gt 0 ]]; then
    echo ""
    log "Debug: systemctl status sznuper"
    ssh_run "systemctl status sznuper 2>&1" || true
    echo ""
    log "Debug: journalctl -u sznuper"
    ssh_run "journalctl -u sznuper --no-pager -n 20 2>&1" || true
fi

echo ""
log "Results: $PASS passed, $FAIL failed"
[[ $FAIL -eq 0 ]]
