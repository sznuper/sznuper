# Ideas

## `builtin://ok`

A built-in healthcheck that always emits a single `type=ok` event. Useful for testing notification pipelines, validating config, or as a minimal smoke test.

```
--- event
type=ok
```

For now, keep it explicit — require `healthcheck: builtin://ok` in the config rather than making it a fallback when `healthcheck` is omitted.

## Multiple triggers per alert

Currently each alert has a single `trigger` field (one of `interval`, `cron`, `watch`, `pipe`, or `lifecycle`). Change this to a list so an alert can have multiple triggers — e.g. multiple crons, or an interval combined with a watch.

```yaml
# before
trigger:
  interval: 30s

# after
triggers:
  - interval: 30s
  - cron: "0 */6 * * *"
  - watch: /etc/nginx/nginx.conf
```

Each trigger fires the same healthcheck independently.

## Side effects

Allow alerts to define a list of executables that run after each event, in addition to notifications. Side effects resolve and execute the same way healthchecks do (`file://`, `https://`), keeping the interface uniform.

The raw `--- event` block from the healthcheck's stdout is piped as stdin to each side effect. No JSON, no new format — the side effect gets exactly what the healthcheck emitted and is responsible for parsing it (or ignoring it). Parsing `KEY=VALUE` lines is trivial in any language, so the cost of each side effect independently parsing is negligible compared to the daemon needing to serialize into a different format.

This keeps the daemon dumb — it just pipes bytes through — and stays consistent with the existing I/O contract.

```yaml
- name: disk_check
  healthcheck: file://disk_usage
  side_effects:
    - file://log_to_sqlite
    - file://update_dashboard
  trigger:
    interval: 30s
  template: "..."
  notify:
    - telegram
```

Open questions:
- Should side effects also receive globals (hostname, etc.) as env vars, same as healthchecks get args?
- Should side effect failures be logged silently, or can they optionally trigger notifications too?
- Should side effects run in parallel or sequentially?

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

Each entry: timestamp, alert name, event type, target service, and the error.
