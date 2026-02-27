# sznuper — Notifications

## Notification Templates

Alert templates define the message body sent to services. Templates are Go [`text/template`](https://pkg.go.dev/text/template) strings with [Sprig](https://masterminds.github.io/sprig/) functions, resolved at notification time when the healthcheck has run.

```yaml
template: "{{healthcheck.status | upper}} {{globals.hostname}}: Disk {{args.mount}} at {{healthcheck.usage}}%"
```

**Available template variables:**

| Variable | Source |
|---|---|
| `{{globals.hostname}}` | Global `hostname` or system hostname |
| `{{alert.name}}` | Alert's `name` field |
| `{{healthcheck.*}}` | All key-value pairs from healthcheck stdout (including `{{healthcheck.status}}`) |
| `{{healthcheck.status_emoji}}` | Derived from `healthcheck.status`: 🔴 critical, 🟡 warning, 🟢 ok |
| `{{args.*}}` | Args from alert config |

All healthcheck output values are strings. Use `atoi` or `float64` for numeric operations.

`healthcheck.status_emoji` is a **derived variable** — computed by the daemon from `healthcheck.status`, not from healthcheck output. A healthcheck cannot override it.

**Sprig functions are available for advanced formatting:**

```yaml
# String manipulation
template: "{{healthcheck.status | upper}} on {{globals.hostname}}"                   # "WARNING on vps-01"

# Conditionals
template: >
  {{healthcheck.status_emoji}}
  {{globals.hostname}}: Disk at {{healthcheck.usage}}%

# Math
template: "{{healthcheck.available_bytes | float64 | div 1073741824 | printf \"%.1f\"}}GB remaining"

# Default values
template: "{{args.mount | default \"/\"}}"

# Date/time
template: "{{now | date \"15:04\"}} {{healthcheck.status | upper}} {{globals.hostname}}"
```

See [Sprig documentation](https://masterminds.github.io/sprig/) for the full list of available functions.

`template` is a required field. Each healthcheck defines its own output keys, so a meaningful template must be written per alert.

### Per-Service Template Overrides

The top-level `template` is the default message body for all services. Individual services can override it in the `notify` list, along with any Shoutrrr options:

```yaml
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
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: Disk {{args.mount}} at {{healthcheck.usage}}%"
    notify:
      - logfile
      - ops-slack
      - service: telegram
        template: "*{{healthcheck.status | upper}}* `{{globals.hostname}}`: Disk {{args.mount}} at {{healthcheck.usage}}%"
        options:
          parsemode: MarkdownV2
          notification: "{{if eq healthcheck.status \"warning\"}}false{{else}}true{{end}}"
      - service: email
        template: "Disk {{args.mount}} is at {{healthcheck.usage}}%\n\nHost: {{globals.hostname}}\nAvailable: {{healthcheck.available}}"
        options:
          subject: "[{{healthcheck.status | upper}}] {{globals.hostname}}: {{alert.name}}"
```

**Template resolution:** alert.notify[].template → alert.template

`{{...}}` variables work inside Shoutrrr options values too (e.g. `subject`, `notification`).

---

## Variable Interpolation

The config uses two distinct variable syntaxes resolved at different times:

**`${...}` — environment variables, resolved at config parse time.**

Uses [envsubst](https://github.com/a8m/envsubst) to substitute system environment variables when the YAML is loaded. Used for secrets and deployment-specific values.

```yaml
services:
  telegram:
    url: telegram://${TELEGRAM_TOKEN}@telegram
    options:
      chats: ${TELEGRAM_CHAT_ID}
```

**`{{...}}` — template variables, resolved at notification time.**

Populated from globals, alert config, and healthcheck output when a notification is sent.

```yaml
alerts:
  - name: disk_check
    template: "{{healthcheck.status | upper}} {{globals.hostname}}: Disk {{args.mount}} at {{healthcheck.usage}}%"
```

---

## Service Options

Service options map directly to Shoutrrr query params for each service type. Options can be overridden per alert.

**Resolution order (later overrides earlier):**

```
service.options              ← base Shoutrrr params
alert.notify[].options       ← override for this specific alert
```

**Simple notify (service defaults):**

```yaml
notify: [telegram, logfile]
```

**Per-alert service override:**

```yaml
notify:
  - logfile
  - service: telegram
    options:
      notification: true     # override telegram's default for this alert
```

Options values support `{{...}}` template variables for dynamic behavior based on healthcheck output.

---

## Notification Delivery

Built on top of [Shoutrrr](https://containrrr.dev/shoutrrr/). Any Shoutrrr-supported service works as a notification destination.

Supported services include: Telegram, Discord, Slack, Email (SMTP), Gotify, Google Chat, IFTTT, Mattermost, Matrix, Ntfy, OpsGenie, Pushbullet, Pushover, Rocketchat, Teams, Zulip, generic webhooks, and the built-in logger.

The daemon's responsibility is:
- **Routing:** which services to notify, based on alert config.
- **Option merging:** resolve service base options → alert-level overrides into a final set of Shoutrrr params.
- **Spam prevention:** cooldown logic per alert and status.
- **Templating:** resolve `{{...}}` variables into the final message body and option values.

Shoutrrr handles the actual delivery. The daemon does not interpret service options — it passes the merged key-value pairs directly to Shoutrrr.

---

## Language

- **Daemon:** Go. Key dependencies: Shoutrrr (notification delivery), fsnotify (file watching), robfig/cron (cron scheduling), Sprig (template functions), envsubst (env var interpolation).
- **Official healthchecks:** C compiled with Cosmopolitan Libc. Produces single portable binaries that run on any Linux architecture. Direct syscalls for system metrics, zero external dependencies.
- **User healthchecks:** Any language. The daemon executes any file with the executable bit set.
