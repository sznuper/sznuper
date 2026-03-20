# Ideas

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
  triggers:
    - interval: 30s
  template: "..."
  notify:
    - telegram
```

Open questions:
- Should side effects also receive globals (hostname, etc.) as env vars, same as healthchecks get args?
- Should side effect failures be logged silently, or can they optionally trigger notifications too?
- Should side effects run in parallel or sequentially?

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
