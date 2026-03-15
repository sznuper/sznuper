#!/bin/sh
set -eu

REPO="sznuper/sznuper"
BINARY_NAME="sznuper"

# --- helpers ---

info() {
    printf '\033[1;34m::\033[0m %s\n' "$1"
}

ok() {
    printf '\033[1;32m::\033[0m %s\n' "$1"
}

warn() {
    printf '\033[1;33m::\033[0m %s\n' "$1"
}

err() {
    printf '\033[1;31m::\033[0m %s\n' "$1" >&2
    exit 1
}

need() {
    command -v "$1" >/dev/null 2>&1 || err "Required command not found: $1"
}

# --- steps ---

detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64)  ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        *)       err "Unsupported architecture: $arch" ;;
    esac
}

detect_install_paths() {
    if [ "$(id -u)" -eq 0 ]; then
        BIN_DIR="/usr/local/bin"
        CONFIG_DIR="/etc/sznuper"
        CONFIG_PATH="$CONFIG_DIR/config.yml"
        IS_ROOT=1
    else
        BIN_DIR="$HOME/.local/bin"
        CONFIG_DIR="$HOME/.config/sznuper"
        CONFIG_PATH="$CONFIG_DIR/config.yml"
        IS_ROOT=0
    fi
}

download_binary() {
    need curl

    if [ -n "${VERSION:-}" ]; then
        TAG="$VERSION"
    else
        info "Fetching latest release..."
        TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
            | grep '"tag_name"' | head -1 | cut -d'"' -f4)

        if [ -z "$TAG" ]; then
            err "Could not determine latest release"
        fi
    fi

    ASSET="${BINARY_NAME}-linux-${ARCH}"
    URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"

    info "Downloading $BINARY_NAME $TAG ($ARCH)..."
    mkdir -p "$BIN_DIR"
    TMP_BIN=$(mktemp "$BIN_DIR/$BINARY_NAME.XXXXXX")
    trap 'rm -f "$TMP_BIN"' EXIT
    curl -fsSL -o "$TMP_BIN" "$URL"
    chmod +x "$TMP_BIN"
    mv -f "$TMP_BIN" "$BIN_DIR/$BINARY_NAME"
    trap - EXIT

    ok "Installed $BIN_DIR/$BINARY_NAME ($TAG)"
}

init_config() {
    if [ -f "$CONFIG_PATH" ]; then
        warn "Config already exists at $CONFIG_PATH — skipping init"
        return
    fi

    info "Initializing config..."

    # When run via curl|sh, stdin is the pipe — try /dev/tty for interactive init
    if (exec < /dev/tty) 2>/dev/null && "$BIN_DIR/$BINARY_NAME" init < /dev/tty; then
        ok "Config created at $CONFIG_PATH"
    else
        warn "Skipped config init (no TTY) — run 'sznuper init' manually"
    fi
}

setup_systemd() {
    if [ "$IS_ROOT" -ne 1 ]; then
        return
    fi

    if ! command -v systemctl >/dev/null 2>&1; then
        warn "systemctl not found — skipping service setup"
        return
    fi

    SERVICE_FILE="/etc/systemd/system/sznuper.service"

    info "Installing systemd service..."
    cat > "$SERVICE_FILE" <<'EOF'
[Unit]
Description=sznuper server monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sznuper start
EnvironmentFile=-/etc/sznuper/.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload

    if [ -f "$CONFIG_PATH" ]; then
        systemctl enable --now sznuper
        ok "Systemd service installed and started"
    else
        systemctl enable sznuper
        ok "Systemd service installed (will start after config is created)"
    fi
}

print_summary() {
    printf '\n'
    ok "sznuper installation complete"
    printf '\n'
    printf '  Binary:  %s\n' "$BIN_DIR/$BINARY_NAME"
    printf '  Config:  %s\n' "$CONFIG_PATH"

    if [ "$IS_ROOT" -eq 1 ] && command -v systemctl >/dev/null 2>&1; then
        printf '  Service: systemctl status sznuper\n'
    else
        printf '\n'
        warn "No systemd service was set up."
        printf '  To run manually:  %s start\n' "$BIN_DIR/$BINARY_NAME"
        printf '  To run on boot:   set up a user systemd service or cron job\n'

        case ":${PATH}:" in
            *":$BIN_DIR:"*) ;;
            *) printf '\n'
               warn "$BIN_DIR is not in your PATH — add it to your shell profile:"
               # shellcheck disable=SC2016
               printf '  export PATH="%s:$PATH"\n' "$BIN_DIR" ;;
        esac
    fi

    printf '\n'
}

# --- main ---

main() {
    detect_arch
    detect_install_paths
    download_binary
    init_config
    setup_systemd
    print_summary
}

main
