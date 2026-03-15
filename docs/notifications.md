# sznuper — Notifications

## Notification Templates

Alert templates define the message body sent to services. Templates are Go [`text/template`](https://pkg.go.dev/text/template) strings with [Sprig](https://masterminds.github.io/sprig/) functions, resolved at notification time when the healthcheck has run.

```yaml
template: "[{{event.type | upper}}] {{globals.hostname}}: Disk {{args.mount}} at {{event.usage_percent}}%"
```

**Available template variables:**

| Variable | Source |
|---|---|
| `{{event.type}}` | The event type name (required field) |
| `{{event.*}}` | All key-value pairs from the event payload |
| `{{globals.hostname}}` | Global `hostname` or system hostname |
| `{{alert.name}}` | Alert's `name` field |
| `{{args.*}}` | Args from alert config |

All event field values are strings. Use `atoi` or `float64` for numeric operations.

**Sprig functions are available for advanced formatting:**

```yaml
# String manipulation
template: "[{{event.type | upper}}] on {{globals.hostname}}"                   # "[HIGH_USAGE] on vps-01"

# Conditionals
template: >
  {{globals.hostname}}: Disk at {{event.usage_percent}}%

# Math
template: "{{event.available_bytes | float64 | div 1073741824 | printf \"%.1f\"}}GB remaining"

# Default values
template: "{{args.mount | default \"/\"}}"

# Date/time
template: "{{now | date \"15:04\"}} [{{event.type | upper}}] {{globals.hostname}}"
```

See [Sprig documentation](https://masterminds.github.io/sprig/) for the full list of available functions.

`template` is a required field. Each healthcheck defines its own output keys, so a meaningful template must be written per alert.

### Template Inheritance

The alert-level `template` is the default for all event types. Per-event-type templates can be specified in `events.override.<type>.template`. Templates are not a per-service concern — all notify targets for an event receive the same rendered message.

```yaml
alerts:
  - name: ssh_journal
    healthcheck: file://ssh_journal
    trigger:
      pipe: journalctl -f --since=now -u ssh -u sshd --output=json --output-fields=MESSAGE,__REALTIME_TIMESTAMP --no-pager
    template: "SSH {{event.type}} from {{event.host}} as {{event.user}}"
    cooldown: 5m
    notify:
      - telegram
      - logger
    events:
      on_unmatched: drop
      override:
        failure:
          cooldown: 1m
          notify:
            - logger
            - telegram:
                params:
                  notification: "false"
        login:
          template: "SSH login by {{event.user}} from {{event.host}}"
          notify:
            - telegram
```

`{{...}}` variables work inside Shoutrrr params values too (e.g. `notification`).

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
    template: "[{{event.type | upper}}] {{globals.hostname}}: Disk {{args.mount}} at {{event.usage_percent}}%"
```

---

## Service Options

Service options map directly to Shoutrrr query params for each service type. Options can be overridden per alert.

**Resolution order (later overrides earlier):**

```
service.params              ← base Shoutrrr params
alert.notify[].params       ← override for this specific alert
```

**Simple notify (service defaults):**

```yaml
notify: [telegram, logfile]
```

**Per-alert service override:**

```yaml
notify:
  - logfile
  - telegram:
      params:
        notification: "true"   # override telegram's default for this alert
```

Options values support `{{...}}` template variables for dynamic behavior based on healthcheck output.

---

## Notification Delivery

Built on top of [Shoutrrr](https://shoutrrr.nickfedor.com/v0.14.0/services/overview/). Any Shoutrrr-supported service works as a notification destination.

Supported services include: Telegram, Discord, Slack, Email (SMTP), Gotify, Google Chat, IFTTT, Mattermost, Matrix, Ntfy, OpsGenie, Pushbullet, Pushover, Rocketchat, Teams, Zulip, generic webhooks, and the built-in logger.

The daemon's responsibility is:
- **Routing:** which services to notify, based on alert config.
- **Option merging:** resolve service base options → alert-level overrides into a final set of Shoutrrr params.
- **Spam prevention:** cooldown logic per alert and event type.
- **Templating:** resolve `{{...}}` variables into the final message body and option values.

Shoutrrr handles the actual delivery. The daemon does not interpret service options — it passes the merged key-value pairs directly to Shoutrrr.

---

## Language

- **Daemon:** Go. Key dependencies: Shoutrrr (notification delivery), fsnotify (file watching), robfig/cron (cron scheduling), Sprig (template functions), envsubst (env var interpolation).
- **Official healthchecks:** C compiled with Cosmopolitan Libc. Produces single portable binaries that run on any Linux architecture. Direct syscalls for system metrics, zero external dependencies.
- **User healthchecks:** Any language. The daemon executes any file with the executable bit set.
