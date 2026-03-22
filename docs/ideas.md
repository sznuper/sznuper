# Ideas

## Environment variables and secrets management

sznuper needs secrets (API tokens, chat IDs) to talk to notification services. Currently the systemd service reads from `EnvironmentFile=-/etc/sznuper/.env`, but there's no coherent strategy across install modes. The `.env` file isn't created by anything — users have to know to make it themselves.

This needs a proper design that covers all the deployment shapes:

- **Root + systemd** — system service, config in `/etc/sznuper/`, `.env` next to it. Systemd `EnvironmentFile=` handles loading.
- **Non-root + systemd user service** — config in `~/.config/sznuper/`, `.env` next to it. Needs `loginctl enable-linger` for persistence. We don't set this up yet.
- **Non-root, no systemd** — running `sznuper start` manually or via cron. Who loads the `.env`? Does sznuper itself read it, or does the user `source` it in their shell?
- **Root, no systemd** — e.g. Alpine/containers with OpenRC. Same question about who loads `.env`.

Open questions:
- Whose responsibility is it to create the `.env` file? `install.sh`, `sznuper init`, or both? `sznuper init` knows which services were chosen and could pre-fill variable names. `install.sh` could create the file with correct permissions (`600`).
- Should sznuper itself load `.env` at startup (like Docker Compose does), or rely on the environment? Loading it ourselves would make the non-systemd case simpler — no need to `source` first. But it adds a feature to maintain and a place where "where do env vars come from?" gets confusing.
- Where does `.env` live in each mode? Next to `config.yml`? Fixed path? Configurable in `options`?
- File permissions — `.env` contains secrets and must be `600`. Who enforces this? Should `sznuper start` warn if the file is world-readable?
- Secrets encryption — future goal. Don't design `.env` handling in a way that would conflict with later encryption support (e.g. `sznuper secrets set TELEGRAM_TOKEN` that encrypts at rest and decrypts at startup).
- Should `sznuper validate` check that referenced `${VAR}` values are actually set in the environment / `.env` file, and warn if they're empty?

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

## Config hot-reload

Re-read the config file when it changes on disk, without requiring a full daemon restart. Watch for file changes (e.g. inotify/fsnotify) and apply the new config at runtime — add/remove/update alerts, services, and options.
