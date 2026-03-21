# sznuper — Healthchecks

## Healthcheck URI Schemes

### `file://`

Runs a local executable directly.

- `file://disk_usage` — relative to `options.healthchecks_dir`
- `file:///usr/local/bin/my_check.py` — absolute path

Behavior:
- No caching, no downloading. Runs the file directly from disk.
- Daemon verifies the file exists and is executable on startup.
- Fails loud if missing.
- `sha256` is **optional**, defaults to `false`. If set to a hash string, the daemon validates the file's hash before every run and refuses to execute on mismatch.

### `https://`

Fetches a remote script, caches it locally, and runs the cached version.

- `sha256` field is **mandatory** in the config — either a hash string or `false` (explicit opt-out). If missing, the daemon refuses to start and prints:

```
Error: alert "my_check" uses https:// but has no sha256 field.
Add the script's sha256 hash, or set sha256: false to skip verification.
```

#### When `sha256` is a hash string (pinned):

- Cache directory stores files named by their sha256.
- On run: check if hash exists in cache → if yes, run it → if no, download, validate hash, cache, run.
- If download fails and file is cached → run cached version.
- If download fails and not cached → alert errors out.
- Never re-downloads if the hash file already exists in cache.

#### When `sha256: false` (unpinned):

- Script is fetched once per daemon start and cached in memory/temp.
- Re-fetched on `sznuper validate`.
- Pinned scripts survive restarts, unpinned scripts don't.

### `builtin://`

Synthetic healthchecks handled directly by the daemon — no external process is spawned.

- `builtin://lifecycle` — emits a startup/shutdown event with the configured alert count. Used internally by the default `sznuper_lifecycle` alert.
- `builtin://ok` — always emits a single `type=ok` event. Useful for alerts that just need to run on a schedule (cron jobs, periodic tasks) without any actual verification — the healthcheck always succeeds, so the notification always fires. Also handy for testing notification pipelines or validating config.

Behavior:
- No file resolution, downloading, or caching. The daemon generates the output in-process.
- `sha256` is not applicable and should be omitted.
- `args` are passed as params to the builtin handler. `builtin://ok` ignores all params.

### `sha256` Summary

| Scheme    | `sha256` field | Default   | Behavior                                          |
| --------- | -------------- | --------- | ------------------------------------------------- |
| `file://`  | optional       | `false`   | If set, validates hash before every run.          |
| `https://` | required       | —         | Hash string: fetch once, cache forever.           |
| `https://` | required       | —         | `false`: fetch once per daemon start, no persist. |
| `builtin://`| n/a           | —         | No file on disk — output generated in-process.    |

### Healthcheck Lifecycle Flowcharts

#### 1. `file://` without sha256 (default)

```yaml
- name: disk_check
  healthcheck: file://disk_usage
  triggers:
    - interval: 30s
```

```mermaid
flowchart TD
    A[Alert triggered] --> B{File exists at\noptions.healthchecks_dir/disk_usage ?}
    B -->|No| C[❌ Alert errors out\nLog: file not found]
    B -->|Yes| D{File is executable?}
    D -->|No| E[❌ Alert errors out\nLog: not executable]
    D -->|Yes| F[Run healthcheck with\nenv vars + stdin]
    F --> G[Parse events from output]
    G --> H[Process each event through\nconfig resolution → state machine →\ncooldown → template → notify]
```

#### 2. `file://` with sha256 string

```yaml
- name: disk_check
  healthcheck: file://disk_usage
  sha256: b7e4f2c8d1a9...
  triggers:
    - interval: 30s
```

```mermaid
flowchart TD
    A[Alert triggered] --> B{File exists at\noptions.healthchecks_dir/disk_usage ?}
    B -->|No| C[❌ Alert errors out\nLog: file not found]
    B -->|Yes| D{File is executable?}
    D -->|No| E[❌ Alert errors out\nLog: not executable]
    D -->|Yes| F[Compute sha256\nof file on disk]
    F --> G{Hash matches config?}
    G -->|No| H[❌ Alert errors out\nLog: hash mismatch\nfile may have been tampered with]
    G -->|Yes| I[Run healthcheck with\nenv vars + stdin]
    I --> J[Parse events from output]
    J --> K[Process each event through\nconfig resolution → state machine →\ncooldown → template → notify]
```

#### 3. `https://` with sha256 string (pinned)

```yaml
- name: ssl_expiry
  healthcheck: https://github.com/sznuper/healthchecks/releases/download/v1.0.0/ssl_check
  sha256: a1b2c3d4e5f6...
  triggers:
    - interval: 6h
```

```mermaid
flowchart TD
    A[Alert triggered] --> B{File exists in cache as\noptions.cache_dir/a1b2c3d4e5f6... ?}
    B -->|Yes| F[Run cached healthcheck with\nenv vars + stdin]
    B -->|No| C[Attempt HTTPS download]
    C --> D{Download successful?}
    D -->|No| E[❌ Alert errors out\nLog: download failed\nand no cache available]
    D -->|Yes| G[Compute sha256 of\ndownloaded content]
    G --> H{Hash matches config?}
    H -->|No| I[❌ Alert errors out\nLog: hash mismatch\nremote content may have changed]
    H -->|Yes| J[Save to options.cache_dir/a1b2c3d4e5f6...\nMark as executable]
    J --> F
    F --> K[Parse events from output]
    K --> L[Process each event through\nconfig resolution → state machine →\ncooldown → template → notify]
```

#### 4. `https://` with sha256: false (unpinned)

```yaml
- name: experimental_check
  healthcheck: https://example.com/beta_check.sh
  sha256: false
  triggers:
    - interval: 1h
```

**Phase 1: Daemon start / validate**

```mermaid
flowchart TD
    A[Daemon starts or\nsznuper validate] --> B[Attempt HTTPS download]
    B --> C{Download successful?}
    C -->|Yes| D[Cache healthcheck in\nmemory/temp for\nthis session]
    C -->|No| E[⚠️ Log warning:\ndownload failed]
    E --> F{Previous session\ncache exists in temp?}
    F -->|No| G[⚠️ Alert disabled\nfor this session\nLog: no healthcheck available]
    F -->|Yes| H[⚠️ Use stale cache\nLog warning]
```

**Phase 2: Alert triggered**

```mermaid
flowchart TD
    A[Alert triggered] --> B{Healthcheck available\nin session cache?}
    B -->|No| C[❌ Alert errors out\nLog: no healthcheck available]
    B -->|Yes| D[Run cached healthcheck with\nenv vars + stdin]
    D --> E[Parse events from output]
    E --> F[Process each event through\nconfig resolution → state machine →\ncooldown → template → notify]
```

**Phase 3: Daemon stops**

```mermaid
flowchart TD
    A[Daemon stops] --> B[Session cache discarded]
    B --> C[Unpinned healthchecks will be\nre-fetched on next start]
```

### Bundled Scripts and `sznuper init` [TODO]

> **[TODO]** `sznuper init` is not yet implemented. The bundled healthcheck distribution described below is the planned behaviour.

Official healthcheck scripts are written in C and compiled with Cosmopolitan Libc into single portable binaries. On `sznuper init`:

1. Official healthchecks are downloaded from the official repository and placed into `options.healthchecks_dir` as local files.
2. Cached versions are also placed in `options.cache_dir` with their sha256 filenames.
3. Default config is generated referencing the official repo HTTPS URLs with matching sha256 values.

Result: works offline immediately after init. The config references canonical HTTPS URLs but the cache is pre-populated. Since official healthchecks are Cosmopolitan portable binaries, the same URL and sha256 work on any architecture — configs are fully portable across machines.

Official scripts are not a special case. They are distributed via the same `https://` mechanism as any community healthcheck. They just happen to live in the official repository (e.g. `github.com/sznuper/healthchecks`) and are pre-cached on init as a convenience.

Example of what `sznuper init` generates:

```yaml
alerts:
  - name: disk_check
    healthcheck: https://raw.githubusercontent.com/sznuper/healthchecks/v1.0.0/disk_usage
    sha256: a1b2c3d4e5f6...
    triggers:
      - interval: 30s
    args:
      threshold_warn_percent: 80
      threshold_crit_percent: 95
      mount: /
    cooldown: 10m
    template: "[{{event.type | upper}}] {{globals.hostname}}: Disk {{args.mount}} at {{event.usage_percent}}%"
    notify:
      - telegram
    events:
      healthy: [ok]
```

The user can also change the URI to `file://disk_usage` to use the local copy directly. Both work.

---

## Healthcheck Interface

### Input

**Environment variables:**

Daemon metadata (set by the daemon):

| Variable | Description | Set for |
|---|---|---|
| `HEALTHCHECK_TRIGGER` | `"interval"`, `"cron"`, `"watch"`, or `"pipe"` | always |
| `HEALTHCHECK_ALERT_NAME` | Name of the alert being executed | always |
| `HEALTHCHECK_FILE` | Watched file path | watch only [TODO] |
| `HEALTHCHECK_LINE_COUNT` | Number of new lines | watch only [TODO] |

User args (from config `args`, prefixed with `HEALTHCHECK_ARG_`):

| Config | Environment variable |
|---|---|
| `threshold_warn_percent: 80` | `HEALTHCHECK_ARG_THRESHOLD_WARN_PERCENT=80` |
| `mount: /` | `HEALTHCHECK_ARG_MOUNT=/` |

Arg keys are uppercased as-is when mapped to environment variables (e.g., `threshold_warn_percent` → `HEALTHCHECK_ARG_THRESHOLD_WARN_PERCENT`, `mount` → `HEALTHCHECK_ARG_MOUNT`).

**Stdin:**

- For `watch` triggers: new bytes appended to the watched file since the last invocation.
- For `pipe` triggers: bytes accumulated from the pipe command's stdout since the last invocation.
- For `interval`/`cron` triggers: empty.

### Output

**Format:** A list of events. Each event starts with a `--- event` delimiter on its own line, followed by `KEY=VALUE` pairs (one per line). Split on first `=` only. Lines without `=` within an event block are ignored. Lines before the first `--- event` are ignored.

**Required field:**

| Key | Required | Description |
|---|---|---|
| `type` | **yes** | The event type name (e.g., `ok`, `high_usage`, `failure`, `login`). Drives config resolution, cooldown, and notification routing. |

Empty output (no `--- event` markers) is valid — it's a list of zero events.

**All other fields are arbitrary key-value pairs (the event payload):**

```
--- event
type=high_usage
mount=/
usage_percent=84.3
available=8G
```

**Multiple events** from a single invocation (e.g., pipe triggers processing a batch):

```
--- event
type=failure
user=root
host=14.18.190.138
timestamp=2026-03-14T01:06:00Z
--- event
type=login
user=niar
host=10.0.0.1
timestamp=2026-03-14T01:08:00Z
```

Each event is processed independently through the pipeline: config resolution → state machine → cooldown → template → notify.

Values are always stored as plain strings in event fields.

#### Template access

Event fields are available in templates as `{{event.*}}`:

```yaml
template: "SSH {{event.type}} from {{event.host}} as {{event.user}}"
```

---

### Healthcheck Types

A healthcheck is any executable file. The daemon does not care about the language or runtime — it executes the file and reads stdout.

**Official bundled healthchecks** are written in C and compiled with [Cosmopolitan Libc](https://github.com/jart/cosmopolitan) into Actually Portable Executables. Each healthcheck is a single binary that runs on any architecture (x86_64, aarch64) and any Linux distribution. They use direct syscalls (`statvfs()`, `/proc/stat`, `/proc/meminfo`, etc.) with zero external dependencies — no `df`, no `awk`, no runtime needed. This means one binary per healthcheck, one URL, one sha256, and the same config works across machines regardless of architecture.

**Community and user healthchecks** can be anything: Go, Rust, Python, Node.js, Bash, C — whatever the user has on their system. Shell scripts are supported but not recommended for shared healthchecks due to portability concerns (reliance on `awk`, `df`, `bc`, etc. varying across distributions). Cosmopolitan C is recommended for healthchecks intended to be distributed.

The daemon treats all healthchecks identically. The distinction between bundled and user healthchecks only matters at distribution time.

### Technology Stack

| Layer                  | Language              | Why                                            |
| ---------------------- | --------------------- | ---------------------------------------------- |
| Daemon (`sznuper`)     | Go                    | Shoutrrr, Sprig, envsubst, fsnotify, robfig/cron |
| Official healthchecks   | C (Cosmopolitan Libc) | Portable single binary, direct syscalls        |
| User/community healthchecks | Anything          | User's choice and responsibility               |

### Documenting Healthcheck Interfaces

Each healthcheck (official or community) should document its arguments, outputs, and event type logic. Example for the official `disk_usage` healthcheck:

```
disk_usage

Arguments (config → env):
  threshold_warn_percent  → HEALTHCHECK_ARG_THRESHOLD_WARN_PERCENT  - warning threshold (0-100)
  threshold_crit_percent  → HEALTHCHECK_ARG_THRESHOLD_CRIT_PERCENT  - critical threshold (0-100)
  mount                   → HEALTHCHECK_ARG_MOUNT                   - mount point to check

Event types:
  ok             - usage below warning threshold
  high_usage     - usage at or above warning threshold
  critical_usage - usage at or above critical threshold

Event fields:
  usage_percent  - percentage as float (0-100)
  available      - remaining space

Event type logic:
  usage >= HEALTHCHECK_ARG_THRESHOLD_CRIT → type=critical_usage
  usage >= HEALTHCHECK_ARG_THRESHOLD_WARN → type=high_usage
  otherwise                               → type=ok
```

This tells the user exactly what to put in `args`, what `{{event.*}}` variables are available for templates, and what event types to expect for `events.override` configuration.

### Example: bundled healthchecks (Cosmopolitan C, single portable binary each)

```
healthchecks/
  disk_usage          # runs on any arch
  cpu_usage           # runs on any arch
  memory_usage        # runs on any arch
  ssh_login           # runs on any arch, reads stdin lines
  systemd_unit        # runs on any arch
```

### Example: user healthchecks (any executable)

```
healthchecks/
  my_custom_check.sh      # bash
  ssl_verify.py           # python
  api_health              # compiled Go/Rust/C
```

### Example: interval healthcheck invocation

```
HEALTHCHECK_TRIGGER=interval HEALTHCHECK_ALERT_NAME=disk_check HEALTHCHECK_ARG_THRESHOLD_WARN_PERCENT=80 HEALTHCHECK_ARG_MOUNT=/ /etc/sznuper/healthchecks/disk_usage
```

### Example: watch healthcheck invocation

> **Note:** The watch trigger works, but environment variables `HEALTHCHECK_FILE` and `HEALTHCHECK_LINE_COUNT` are not yet set by the daemon.

```
HEALTHCHECK_TRIGGER=watch HEALTHCHECK_FILE=/var/log/auth.log HEALTHCHECK_LINE_COUNT=3 HEALTHCHECK_ARG_WATCH=all HEALTHCHECK_ARG_EXCLUDE_USERS=deploy /etc/sznuper/healthchecks/ssh_login <<< "line1\nline2\nline3"
```
