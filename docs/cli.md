# barker — CLI Commands

## `barker init`

Generates default config if it doesn't exist, then runs `barker validate`.

- Detects root vs non-root and places files accordingly (`/etc/barker/` vs `~/.config/barker/`).
- Downloads official checks from the official repository and pre-populates the cache.
- Does nothing if config already exists (will not overwrite).

## `barker start`

Runs `barker validate` first, then starts the daemon in the foreground. Reads config, starts all alert intervals and file watchers, runs until interrupted.

**Signal handling:**
- **SIGINT** (Ctrl+C) — graceful shutdown. Finishes any currently running checks, then exits.
- **SIGTERM** (systemd stop) — graceful shutdown. Same behavior as SIGINT.
- **SIGHUP** — triggers `validate` logic: re-reads config, re-fetches unpinned `https://` checks. If validation fails, the reload is rejected and the daemon continues with the previous config.

## `barker validate`

Validates and syncs the current config:

- Validates YAML syntax.
- Verifies all services have valid Shoutrrr URLs.
- Verifies all `file://` checks exist and are executable.
- Verifies all `sha256` hashes match.
- Fetches/re-fetches all `https://` checks (pinned: only if not cached; unpinned: always).
- Reports errors and exits non-zero if anything fails.

Used automatically by `barker init` and `barker start`. Can be run standalone for CI/deploy pipelines.

## `barker run <alert_name>`

Manually triggers a specific alert once. Runs the check, renders the template, and sends notifications to configured services. Ignores the trigger type — just executes the check immediately.

For checks that expect stdin (file watch checks), pipe input manually:

```
$ echo "some log line" | barker run ssh_login
```

```
$ barker run disk_check
✓ Check: file://disk_usage
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
$ barker run disk_check --dry-run
✓ Check: file://disk_usage
  Output:
    status=warning
    usage=84
    total=50G
    available=8G
  Rendered: "WARNING vps-01: Disk / at 84% (8G remaining)"
  Would notify: telegram, logfile
```

This allows barker to be used as a standalone one-shot tool (e.g. from cron) without running the daemon.

## `barker hash <file>`

Prints the sha256 hash of a file. Convenience for users adding pinned checks to their config.

```
$ barker hash checks/disk_usage
a1b2c3d4e5f6789...
```
