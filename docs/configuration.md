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
  healthchecks/                            # file:// healthchecks
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
  healthchecks/
~/.cache/sznuper/                         # https:// cached scripts
~/.local/state/sznuper/logs/              # daemon log
```

`sznuper init` places files according to whether it's running as root or not.

---

## Config Structure

```yaml
# Directories — have sensible defaults, user can override
dirs:
  healthchecks: /etc/sznuper/healthchecks  # file:// resolves relative to this
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
    healthcheck: file://disk_usage
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
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: Disk {{args.mount}} at {{healthcheck.usage}}% ({{healthcheck.available}} remaining)"
    notify: [telegram, logfile]

  - name: ssl_expiry
    healthcheck: https://raw.githubusercontent.com/sznuper/healthchecks/v1.0.0/ssl_check
    sha256: a1b2c3d4e5f6...              # required for https
    trigger:
      interval: 6h
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: Certificate for {{healthcheck.domain}} expires in {{healthcheck.days_left}} days"
    notify: [telegram]

  - name: experimental_check
    healthcheck: https://example.com/beta_check.sh
    sha256: false                         # explicit opt-out, re-fetched on daemon start
    trigger:
      interval: 1h
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: {{healthcheck.message}}"
    notify: [logfile]

  - name: ssh_login
    healthcheck: file://ssh_login
    trigger:
      watch: /var/log/auth.log
    timeout: 30s
    args:
      watch: all
      exclude_users: deploy
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: SSH login by {{healthcheck.user}} from {{healthcheck.ip}}"
    notify: [telegram]

  # Per-alert service override with template conditionals
  - name: postgres_down
    healthcheck: file://systemd_unit
    trigger:
      interval: 15s
    args:
      units: postgresql
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: Unit {{healthcheck.unit}} is {{healthcheck.state}}"
    notify:
      - logfile
      - ops-slack
      - service: telegram
        options:
          notification: "{{if eq healthcheck.status \"warning\"}}false{{else}}true{{end}}"
```
