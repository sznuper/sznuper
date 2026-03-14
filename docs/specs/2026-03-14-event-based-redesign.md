# Event-Based Healthcheck Protocol Redesign — Spec

## Context & Motivation

Sznuper's current healthcheck protocol requires healthchecks to emit `status=ok|warning|critical`, coupling two responsibilities into the healthcheck binary: (1) detecting system events and (2) deciding their severity. This means users cannot control notification behavior — which events trigger notifications, with what parameters, to which services — without modifying and rebuilding the healthcheck source code.

**Real-world problem:** With the ssh_journal healthcheck on a VPS, every SSH brute-force attempt sends a Telegram notification with sound. The user wants failures sent silently and logins sent with sound — but the only way to achieve this is modifying C source code and rebuilding the binary.

**Goal:** Redesign the protocol so healthchecks are pure event emitters (timeless, reusable sensors) and all notification routing, severity classification, and delivery behavior is controlled from sznuper's config file.

**Non-goals:** Manifest files for healthcheck metadata (future work), aggregation/batching logic, backwards compatibility with v1 protocol.

---

## Healthcheck Output Protocol

### Format

Healthcheck output is a list of events. Each event starts with a `--- event` delimiter, followed by `key=value` pairs (one per line).

```
--- event
type=failure
user=root
host=1.2.3.4
timestamp=2026-03-14T01:06:00Z
--- event
type=login
user=niar
host=10.0.0.1
timestamp=2026-03-14T01:08:00Z
```

### Rules

- Every event block starts with `--- event` on its own line
- The `type` field is **required** in every event — it names the event type
- All other fields are arbitrary `key=value` pairs (the event payload)
- Empty output (no `--- event` markers) is valid — it's a list of zero events, processed normally (loop over zero elements)
- Non-`key=value` lines within an event block are ignored
- Array values are supported: `hosts=[1.2.3.4, 5.6.7.8]` (parsed as typed arrays — same as current protocol)
- Lines before the first `--- event` are ignored

### Removed

- No `status` field — healthchecks no longer determine severity
- No `--- records` / `--- record` delimiters — replaced by unified `--- event`
- No global fields section — all data lives inside event blocks

### Input (unchanged)

- `HEALTHCHECK_TRIGGER` env var — trigger type (interval, cron, watch, pipe, lifecycle)
- `HEALTHCHECK_ARG_*` env vars — per-argument from config `args`
- `stdin` — piped data for watch/pipe triggers

---

## Configuration

### Full Config Structure

```yaml
options:
  healthchecks_dir: /var/lib/sznuper/healthchecks
  cache_dir: /var/cache/sznuper
  logs_dir: /var/log/sznuper

globals:
  hostname: my-server

services:
  telegram:
    url: telegram://${TELEGRAM_TOKEN}@telegram
    params:
      chats: ${TELEGRAM_CHAT_ID}
      parsemode: MarkdownV2
  logger:
    url: logger://

alerts:
  - name: <string>             # required
    healthcheck: <uri>         # required (file://, https://, builtin://)
    sha256: <hash> | false     # optional
    trigger:                   # required (exactly one)
      interval: <duration>
      cron: <expression>
      watch: <file path>
      pipe: <shell command>
      lifecycle: <bool>
    timeout: <duration>        # optional
    args:                      # optional — passed as HEALTHCHECK_ARG_* env vars
      key: value
    template: <string>         # required — default template for all events
    cooldown: <duration>       # optional — default cooldown for all events
    notify:                    # required — default notify targets for all events
      - <service_name>
      - <service_name>:
          params:
            key: value
    events:                    # optional — event handling configuration
      healthy: [<type>, ...]   # event types that represent healthy state
      on_unmatched: default | drop   # default: "default"
      override:
        <type>:
          template: <string>     # overrides alert-level template
          cooldown: <duration>   # overrides alert-level cooldown
          notify:                # overrides alert-level notify (replaces entirely)
            - <service_name>
            - <service_name>:
                params:
                  key: value
```

### Alert-Level Defaults & Event-Level Overrides

Three fields are **inheritable**: `template`, `cooldown`, and `notify`. They follow one consistent rule:

- **Alert-level** = default for all event types
- **Event-level** (`events.override.<type>`) = override for that specific event type

If an event type is not listed in `events.override`, it uses alert-level defaults (unless `on_unmatched: drop`, in which case it's discarded).

### Notify Targets

Notify is always an **array**. Each element is either a service name string or a service name with params:

```yaml
notify:
  - telegram
  - logger

# or with per-target params:
notify:
  - telegram:
      params:
        notification: "false"
  - logger
```

Notify targets do **not** have a `template` field. Templates are an alert/event concern, not a service concern.

When notify is specified at event level, it **replaces** the alert-level notify entirely (not merged).

### Events Configuration

```yaml
events:
  healthy: [ok]           # which event types mean "all clear"
  on_unmatched: drop      # what to do with unlisted event types
  override:
    failure:              # config for type=failure events
      cooldown: 1m
      notify:
        - telegram:
            params:
              notification: "false"
        - logger
    login:                # config for type=login events
      template: "Login by {{event.user}} from {{event.host}}"
      notify:
        - telegram
```

- **`healthy`**: Array of event type names that represent healthy state. Enables the healthy/unhealthy state machine. If omitted, no state machine — all events notify directly (subject to cooldown).
- **`on_unmatched`**: What to do when a healthcheck emits an event type not listed in `override`.
  - `default` (implicit if omitted): use alert-level template/notify/cooldown.
  - `drop`: silently discard the event (no notification). Note: healthy events still trigger state machine transitions even when dropped — they just don't send a notification.
- **`override`**: Per-event-type configuration. Keys are event type names (matching the `type` field in healthcheck output).

---

## State Machine

### Healthy / Unhealthy States

When `events.healthy` is defined, sznuper maintains a binary state per alert:

```
                    ┌──────────────────────────────┐
                    ▼                              │
               ┌─────────┐                   ┌─────────┐
          ┌───>│ Healthy  │───unhealthy──────>│Unhealthy│──┐
          │    └─────────┘    event           └─────────┘  │
          │         │       (notify per            │       │
          │         │        cooldown)             │       │
          │    healthy event                  unhealthy    │
          │    (no notification)              event        │
          │         │                        (notify per   │
          │         ▼                         cooldown)    │
          │    ┌─────────┐                        │       │
          │    │ Healthy  │                        │       │
          │    └─────────┘                        ▼       │
          │                                  ┌─────────┐  │
          └──────────healthy─────────────────│Unhealthy│  │
                     event                   └─────────┘  │
              (recovery notify per                │       │
               cooldown + reset all               └───────┘
               cooldowns)
```

**Transitions:**

| From | Event | To | Action |
|------|-------|----|--------|
| Healthy | healthy event | Healthy | No notification |
| Healthy | unhealthy event | Unhealthy | Send notification (subject to cooldown) |
| Unhealthy | unhealthy event | Unhealthy | Send notification (subject to cooldown) |
| Unhealthy | healthy event | Healthy | Send recovery notification (subject to cooldown) + reset all cooldowns |

All notifications go through the same cooldown logic — there are no special cases. The recovery transition resets all cooldown timers, so the next unhealthy event after recovery will always fire (since cooldown was just cleared).

**Rules:**
- An event type is "healthy" if listed in `events.healthy`. Everything else is "unhealthy".
- Initial state is **Healthy**.
- Recovery notifications use the healthy event type's config (from `events.override` or alert-level defaults per `on_unmatched`).
- If `on_unmatched: drop` and the healthy event type is not in `override`, the state transitions but no recovery notification is sent.
- Events within a single healthcheck run are processed in order.

### Without State Machine

When `events.healthy` is not defined:
- No state tracking. No recovery concept.
- Every event is processed and notified independently (subject to cooldown).
- This is the simpler mode, suitable for pipe/watch triggers where events are inherently noteworthy.

---

## Cooldown

Cooldown is **per-event-type** (not per-status as in v1).

```yaml
# Alert-level default
cooldown: 5m

# Per-event-type override
events:
  override:
    failure:
      cooldown: 1m
    login:
      cooldown: 10m
```

**Behavior:**
- Each event type has its own independent cooldown timer
- When an event fires and cooldown is active for that type, the notification is suppressed
- When the state machine transitions unhealthy→healthy (recovery), all cooldown timers are reset
- `inf` duration is supported — suppresses until recovery
- If cooldown is not specified (at any level), no cooldown — every event notifies

---

## Template System

### Namespace Changes

| v1 | v2 | Description |
|----|-----|-------------|
| `{{healthcheck.*}}` | `{{event.*}}` | Event payload fields |
| `{{healthcheck.status}}` | removed | No status in events |
| `{{healthcheck.status_emoji}}` | removed | No status in events |
| `{{globals.*}}` | `{{globals.*}}` | Unchanged |
| `{{alert.*}}` | `{{alert.*}}` | Unchanged |
| `{{args.*}}` | `{{args.*}}` | Unchanged |

### Available Variables

- `{{event.type}}` — the event type name (required field)
- `{{event.<key>}}` — any key=value field from the event payload
- `{{event.<array_key>}}` — array values, usable with array functions
- `{{globals.<key>}}` — from config `globals` section
- `{{alert.name}}` — alert name
- `{{args.<key>}}` — healthcheck args from config

### Functions

All Sprig functions remain available. Custom array functions unchanged:
- `arrayJoin`, `arrayMax`, `arrayMin`, `arraySum`, `arrayFirst`, `arrayLast`, `arrayContains`

### Template Inheritance

Templates follow the same inheritance as notify and cooldown:
- Alert-level `template` is the default
- `events.override.<type>.template` overrides for that event type
- No per-service template overrides (templates are not a service concern)

---

## Processing Pipeline

For each healthcheck execution:

```
1. EXEC healthcheck binary (unchanged)
2. PARSE output → list of events (each with type + payload)
3. For each event in list:
   a. RESOLVE config: find matching events.override.<type> or apply on_unmatched rule
   b. STATE MACHINE: if events.healthy defined, check transition
      - healthy→healthy: skip notification, continue
      - unhealthy→healthy: mark as recovery, reset cooldowns
      - healthy→unhealthy: proceed to notify
      - unhealthy→unhealthy: proceed to notify
   c. COOLDOWN: check if event type is in cooldown period
      - if suppressed: skip notification, continue
      - if not: start cooldown timer for this event type
   d. TEMPLATE: render template with event payload + globals + alert + args
   e. NOTIFY: send to each target service with resolved params
```

---

## Examples

### Minimal Config (disk_usage)

```yaml
services:
  telegram:
    url: telegram://${TELEGRAM_TOKEN}@telegram
    params:
      chats: ${TELEGRAM_CHAT_ID}

alerts:
  - name: disk_usage
    healthcheck: https://github.com/sznuper/healthchecks/.../disk_usage
    sha256: abc123...
    trigger:
      interval: 5m
    args:
      mount: /
      threshold_warn_percent: 80
      threshold_crit_percent: 95
    template: "Disk {{event.mount}} at {{event.usage_percent}}%"
    cooldown: 10m
    notify:
      - telegram
    events:
      healthy: [ok]
```

Healthcheck output when disk is fine:
```
--- event
type=ok
mount=/
usage_percent=45.2
```

Healthcheck output when disk is high:
```
--- event
type=high_usage
mount=/
usage_percent=87.3
```

Behavior: `ok` events keep state healthy (no notification). `high_usage` transitions to unhealthy and notifies. When disk drops back, `ok` triggers recovery notification and resets cooldown.

### Advanced Config (ssh_journal)

```yaml
alerts:
  - name: ssh_journal
    healthcheck: https://github.com/sznuper/healthchecks/.../ssh_journal
    sha256: def456...
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

Healthcheck output:
```
--- event
type=failure
user=root
host=103.129.109.51
timestamp=2026-03-14T01:06:00Z
--- event
type=login
user=niar
host=10.0.0.1
timestamp=2026-03-14T01:08:00Z
```

Behavior:
- `failure` → sent silently to telegram + logger, 1m cooldown
- `login` → sent with sound to telegram only (custom template), 5m cooldown (inherited)
- No `events.healthy` → no state machine, no recovery (pure event stream)

### Lifecycle (builtin)

```yaml
alerts:
  - name: lifecycle
    healthcheck: builtin://lifecycle
    trigger:
      lifecycle: true
    template: "sznuper {{event.type}} ({{event.alerts}} alerts configured)"
    notify:
      - telegram
      - logger
```

Healthcheck output:
```
--- event
type=started
alerts=5
```

---

## Healthcheck Migration (v1 → v2)

### Output Protocol Changes

| v1 | v2 |
|----|-----|
| `status=ok\|warning\|critical` (required) | Removed |
| `--- records` (first separator) | `--- event` (uniform) |
| `--- record` (subsequent separators) | `--- event` (uniform) |
| `event=failure` | `type=failure` |
| Global fields before `--- records` | Removed (all data inside event blocks) |

### Healthcheck Binary Changes

Each healthcheck needs to:
1. Remove all status determination logic (threshold comparisons for status)
2. Replace `status=` output with `type=` output
3. Use `--- event` delimiter instead of `--- records` / `--- record`
4. Keep threshold args — they now control which event types to emit (e.g., emit `type=high_usage` when above warn threshold, `type=critical_usage` when above crit threshold, `type=ok` otherwise)
5. Always emit at least one `--- event` block per run (for interval/cron healthchecks)

### Affected Healthcheck Binaries

- `ssh_journal.c` — remove status logic, emit `type=failure` / `type=login`, use `--- event` delimiter
- `disk_usage.c` — emit `type=ok` / `type=high_usage` / `type=critical_usage` based on thresholds
- `cpu_usage.c` — same pattern as disk_usage
- `memory_usage.c` — same pattern as disk_usage
- `sznuper.h` — remove status-related helpers if any, update documentation

### Affected Sznuper Go Files

- `internal/config/config.go` — update Alert struct (add Events config, remove status-related fields)
- `internal/config/config_test.go` — update tests
- `internal/healthcheck/parse.go` — new event-based parser (replace record parser)
- `internal/healthcheck/parse_test.go` — update tests
- `internal/notify/template.go` — rename `Healthcheck` → event namespace in TemplateData, remove status_emoji
- `internal/notify/template_test.go` — update tests
- `internal/notify/send.go` — update ResolveTargets to use event config
- `internal/runner/runner.go` — new processing pipeline with state machine + per-event-type config resolution
- `internal/cooldown/cooldown.go` — refactor to per-event-type (remove per-status)
- `internal/cooldown/cooldown_test.go` — update tests
- `internal/initcmd/defaults/base.yml` — update default configs
- `internal/initcmd/defaults/systemd.yml` — update default configs
- `docs/` — update all documentation
