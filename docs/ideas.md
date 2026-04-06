# Ideas

## Environment variables and secrets management

Spec: [2026-03-22-env-file-support.md](specs/2026-03-22-env-file-support.md)

## Notification retry + failed delivery log

### Retry

When notification delivery fails, retry automatically. Retries go through the same pipeline as normal events — they're subject to the alert's existing cooldown, not a separate retry-specific mechanism. If cooldown suppresses the retry, it's dropped. This avoids stale/duplicate notifications and keeps the logic simple (no new pipe).

Enabled by default, but users can disable it per alert or globally.

Open questions:
- How many retries? Fixed count (e.g., 3) or time-bounded (e.g., retry for up to 5 minutes)?
- Backoff strategy — exponential, fixed interval, or just piggyback on the next trigger firing?

### Failed delivery log

When a notification permanently fails (all retries exhausted or retries disabled), log it to a local append-only file — the one thing that doesn't depend on an external service being up.

Location: `~/.local/share/sznuper/failed.log` (or `/var/log/sznuper/failed.log` for root).

Each entry: timestamp, alert name, event type, target channel, and the error.

## Consecutive failure threshold

Only send a notification after a healthcheck reports unhealthy N times in a row. Useful for alerts where a single spike is acceptable but sustained failure needs attention (e.g. CPU briefly hitting 100% during a deploy vs being stuck there).

Config would be a per-alert setting, defaulting to `1` (current behavior — notify on first unhealthy event).

```yaml
alerts:
  - name: cpu
    healthcheck: file://cpu_usage
    trigger:
      interval: 30s
    threshold: 3  # only notify after 3 consecutive unhealthy events
```

The daemon would track a per-alert counter that increments on each unhealthy event and resets to zero on a healthy one. Notification fires when the counter reaches the threshold.

Open questions:
- Naming: `threshold`, `consecutive_failures`, `unhealthy_count`, `confirm_count`?
- Should it apply to pipe triggers that emit multiple events per invocation, or only interval/cron triggers where each run produces one result?
- Interaction with cooldown — does the counter reset after cooldown expires, or does it persist?

## ~~Config hot-reload~~ Done

Implemented SIGHUP-based config reload. Sending SIGHUP (or `systemctl reload sznuper`) validates the new config, cancels the current scheduler, and starts a fresh one. Invalid configs are rejected and the daemon keeps running. New lifecycle events: `reload_success` and `reload_failure`.

## Logging overhaul (Caddy-inspired)

Rethink how the daemon does logging, taking inspiration from Caddy's logging architecture. Caddy uses structured JSON logging with configurable outputs (stdout, file, network), multiple named loggers, per-logger filtering/levels, and log sampling. Worth studying how Caddy separates access logs from application logs and lets users route different log streams to different sinks.

Open questions:
- What log sinks do we need? stdout + file at minimum, network (syslog, remote) later?
- Should logging be configurable in the sznuper config file, or only via CLI flags/env vars?
- Structured JSON by default, or human-readable with a JSON option?
- Per-alert log routing — e.g., send healthcheck output to a separate file from daemon logs?

## Interactive healthcheck selection in `init` TUI

During `sznuper init`, present users with a selectable list of healthchecks to include. Auto-detected healthchecks that pass detection would be shown pre-selected (but togglable), and ones that fail detection would be hidden entirely. This gives users more control over what ends up in the generated config instead of silently including everything that passes.

Adding arbitrary `https://` healthchecks in the TUI is probably not worth it -- a single alert has too many options to configure interactively. Users already have `--from` for that.

## ~~Rename "services" to "channels"~~ Done

Renamed the notification `services` concept to `channels` throughout the codebase and config. "Channel" is more intuitive and aligns with the terminology used by tools like Oncall/OpenClaw and similar incident management tools that have become popular.

## Debug builtin healthcheck

A `builtin://debug` healthcheck that emits verbose daemon-internal events -- config reload success/failure, validation errors, scheduler restarts, etc. Opt-in: only runs if explicitly added to an alert in the config, not present by default.

The regular `builtin://lifecycle` should stay lightweight (started/stopped/reload_success/reload_failure). The debug check is for users who want deeper observability into what the daemon is doing internally, delivered through the same alert/notification pipeline as everything else.

```yaml
alerts:
  - name: sznuper-debug
    healthcheck: builtin://debug
    trigger:
      lifecycle: true
    template: "[sznuper] {{ .Event.type }}: {{ .Event.detail }}"
    notify:
      - channel: telegram
```

Open questions:
- What events should it emit? Candidates: `config_validated`, `healthcheck_timeout`, `notification_failed`, `scheduler_restarted` (reload events are already covered by `builtin://lifecycle`)
- Should it overlap with lifecycle at all, or be strictly additive?
- Naming: `builtin://debug`, `builtin://daemon`, `builtin://internal`?

## Rename lifecycle event types

The lifecycle events `started` and `stopped` use past tense, which is inconsistent with the healthcheck event naming convention. Healthchecks use snake_case nouns/adjective states: `ok`, `high_usage`, `critical_usage`, `failure`, `login`, `logout`. The lifecycle events should follow the same pattern -- e.g. `start` and `stop` instead of `started` and `stopped`.

This is a breaking change for anyone using `events.override` with `started`/`stopped` keys in their config.

## ~~Goreleaser~~ Done

Integrated GoReleaser to automate building and releasing sznuper binaries. Handles cross-compilation (linux/amd64, linux/arm64), GitHub releases with tar.gz archives, checksums, and structured changelogs from `.goreleaser.yml`. Replaced the custom matrix build pipeline and Makefile.
