# sznuper — CLI Commands

## `sznuper init`

Generates default config if it doesn't exist, then runs `sznuper validate`.

- Detects root vs non-root and places files accordingly (`/etc/sznuper/` vs `~/.config/sznuper/`).
- Downloads official healthchecks from the official repository and pre-populates the cache.
- Does nothing if config already exists (will not overwrite).

## `sznuper start`

Runs `sznuper validate` first, then starts the daemon in the foreground. Reads config, starts all alert intervals and file watchers, runs until interrupted.

**Signal handling:**
- **SIGINT** (Ctrl+C) — graceful shutdown. Finishes any currently running healthchecks, then exits.
- **SIGTERM** (systemd stop) — graceful shutdown. Same behavior as SIGINT.
- **SIGHUP** — triggers `validate` logic: re-reads config, re-fetches unpinned `https://` healthchecks. If validation fails, the reload is rejected and the daemon continues with the previous config.

## `sznuper validate`

Validates and syncs the current config:

- Validates YAML syntax.
- Verifies all services have valid Shoutrrr URLs.
- Verifies all `file://` healthchecks exist and are executable.
- Verifies all `sha256` hashes match.
- Fetches/re-fetches all `https://` healthchecks (pinned: only if not cached; unpinned: always).
- Reports errors and exits non-zero if anything fails.

Used automatically by `sznuper init` and `sznuper start`. Can be run standalone for CI/deploy pipelines.

## `sznuper run <alert_name>`

Manually triggers a specific alert once. Runs the healthcheck, renders the template, and sends notifications to configured services. Ignores the trigger type — just executes the healthcheck immediately.

For healthchecks that expect stdin (file watch healthchecks), pipe input manually:

```
$ echo "some log line" | sznuper run ssh_login
```

```
$ sznuper run disk_check
✓ Healthcheck: file://disk_usage
  Output:
    status=warning
    usage=84
    total=50G
    available=8G
  Rendered: "WARNING vps-01: Disk / at 84% (8G remaining)"
  Notified: telegram, logfile
```

Use `--dry-run` to skip sending notifications:

```
$ sznuper run disk_check --dry-run
✓ Healthcheck: file://disk_usage
  Output:
    status=warning
    usage=84
    total=50G
    available=8G
  Rendered: "WARNING vps-01: Disk / at 84% (8G remaining)"
  Would notify: telegram, logfile
```

This allows sznuper to be used as a standalone one-shot tool (e.g. from cron) without running the daemon.

## `sznuper hash <file>`

Prints the sha256 hash of a file. Convenience for users adding pinned healthchecks to their config.

```
$ sznuper hash healthchecks/disk_usage
a1b2c3d4e5f6789...
```
