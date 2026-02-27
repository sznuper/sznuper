# sznuper — Configuration

## Config File Location

The daemon looks for config in this order:
1. Explicit flag: `sznuper --config /path/to/config.yaml`
2. User: `~/.config/sznuper/config.yaml`
3. System: `/etc/sznuper/config.yaml`

## File Layout

**System-wide (root):**

```
/usr/bin/sznuper                          # binary
/etc/sznuper/
  config.yaml                             # main config
  checks/                                 # file:// checks
    disk_usage                            # bundled, Cosmopolitan portable binary
    cpu_usage                             # bundled, Cosmopolitan portable binary
    memory_usage                          # bundled, Cosmopolitan portable binary
    ssh_login                             # bundled, Cosmopolitan portable binary
    systemd_unit                          # bundled, Cosmopolitan portable binary
/var/cache/sznuper/                       # https:// cached scripts
    a1b2c3d4e5f6...                       # named by sha256
    f6e5d4c3b2a1...
/var/log/sznuper/
    sznuper.log                           # daemon log
```

**User-local (non-root):**

```
~/.local/bin/sznuper                      # binary
~/.config/sznuper/
  config.yaml
  checks/
~/.cache/sznuper/                         # https:// cached scripts
~/.local/state/sznuper/logs/              # daemon log
```

`sznuper init` places files according to whether it's running as root or not.

---

## Config Structure

```yaml
# Directories — have sensible defaults, user can override
dirs:
  checks: /etc/sznuper/checks         # file:// resolves relative to this
  cache: /var/cache/sznuper            # https:// cached scripts
  logs: /var/log/sznuper               # daemon logs

# Global options
hostname: vps-01                       # optional, defaults to system hostname

# Notification services (Shoutrrr URLs with aliases)
services:
  telegram:
    url: telegram://${TELEGRAM_TOKEN}@telegram
    options:
      chats: ${TELEGRAM_CHAT_ID}
      notification: true
      parsemode: MarkdownV2
      preview: false

  ops-slack:
    url: slack://${SLACK_TOKENS}

  logfile:
    url: logger://

# Alerts
alerts:
  - name: disk_check
    check: file://disk_usage
    trigger:
      interval: 30s
    args:
      threshold_warn: 0.80
      threshold_crit: 0.95
      mount: /
    cooldown:
      warning: 10m
      critical: 1m
      recovery: true
    template: "{{check.status | upper}} {{globals.hostname}}: Disk {{args.mount}} at {{check.usage}}% ({{check.available}} remaining)"
    notify: [telegram, logfile]

  - name: ssl_expiry
    check: https://raw.githubusercontent.com/sznuper/checks/v1.0.0/ssl_check
    sha256: a1b2c3d4e5f6...              # required for https
    trigger:
      interval: 6h
    template: "{{check.status | upper}} {{globals.hostname}}: Certificate for {{check.domain}} expires in {{check.days_left}} days"
    notify: [telegram]

  - name: experimental_check
    check: https://example.com/beta_check.sh
    sha256: false                         # explicit opt-out, re-fetched on daemon start
    trigger:
      interval: 1h
    template: "{{check.status | upper}} {{globals.hostname}}: {{check.message}}"
    notify: [logfile]

  - name: ssh_login
    check: file://ssh_login
    trigger:
      watch: /var/log/auth.log
    timeout: 30s
    args:
      watch: all
      exclude_users: deploy
    template: "{{check.status | upper}} {{globals.hostname}}: SSH login by {{check.user}} from {{check.ip}}"
    notify: [telegram]

  # Per-alert service override with template conditionals
  - name: postgres_down
    check: file://systemd_unit
    trigger:
      interval: 15s
    args:
      units: postgresql
    template: "{{check.status | upper}} {{globals.hostname}}: Unit {{check.unit}} is {{check.state}}"
    notify:
      - logfile
      - ops-slack
      - service: telegram
        options:
          notification: "{{if eq check.status \"warning\"}}false{{else}}true{{end}}"
```
