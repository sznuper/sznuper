# sznuper — Configuration

## Config File Location

The daemon looks for config in this order:
1. Explicit flag: `sznuper --config /path/to/config.yml`
2. User: `~/.config/sznuper/config.yml`
3. System: `/etc/sznuper/config.yml`

## File Layout

**System-wide (root):**

```
/usr/bin/sznuper                          # binary
/etc/sznuper/
  config.yml                             # main config
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
  config.yml
  healthchecks/
~/.cache/sznuper/                         # https:// cached scripts
~/.local/state/sznuper/logs/              # daemon log
```

`sznuper init` places files according to whether it's running as root or not. **[TODO]**

---

## Config Structure

```yaml
# Options — have sensible defaults, user can override
options:
  healthchecks_dir: /etc/sznuper/healthchecks  # file:// resolves relative to this
  cache_dir: /var/cache/sznuper                # https:// cached scripts
  logs_dir: /var/log/sznuper                   # daemon logs

# Globals — free-form key-value pairs available in all templates as {{globals.*}}
globals:
  hostname: vps-01                       # optional, defaults to system hostname

# Notification services (Shoutrrr URLs with params)
services:
  telegram:
    url: telegram://${TELEGRAM_TOKEN}@telegram
    params:
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
    triggers:
      - interval: 30s
    args:
      threshold_warn_percent: 80
      threshold_crit_percent: 95
      mount: /
    cooldown: 10m
    template: "[{{event.type | upper}}] {{globals.hostname}}: Disk {{args.mount}} at {{event.usage_percent}}% ({{event.available}} remaining)"
    notify:
      - telegram
      - logfile
    events:
      healthy: [ok]

  - name: ssl_expiry
    healthcheck: https://raw.githubusercontent.com/sznuper/healthchecks/v1.0.0/ssl_check
    sha256: a1b2c3d4e5f6...              # required for https
    triggers:
      - interval: 6h
    template: "[{{event.type | upper}}] {{globals.hostname}}: Certificate for {{event.domain}} expires in {{event.days_left}} days"
    notify:
      - telegram

  - name: experimental_check
    healthcheck: https://example.com/beta_check.sh
    sha256: false                         # explicit opt-out, re-fetched on daemon start
    triggers:
      - interval: 1h
    template: "[{{event.type | upper}}] {{globals.hostname}}: {{event.message}}"
    notify:
      - logfile

  - name: ssh_journal
    healthcheck: file://ssh_journal
    triggers:
      - pipe: journalctl -f --since=now SYSLOG_FACILITY=10 SYSLOG_FACILITY=4 --output=json --output-fields=MESSAGE,__REALTIME_TIMESTAMP --no-pager
    cooldown: 5m
    template: "SSH {{event.type}} from {{event.host}} as {{event.user}}"
    notify:
      - telegram
    events:
      on_unmatched: drop
      override:
        login: {}
        logout: {}

  # Per-alert service override with per-event-type params
  - name: postgres_down
    healthcheck: file://systemd_unit
    triggers:
      - interval: 15s
    args:
      units: postgresql
    template: "[{{event.type | upper}}] {{globals.hostname}}: Unit {{event.unit}} is {{event.state}}"
    notify:
      - logfile
      - ops-slack
      - telegram
```
