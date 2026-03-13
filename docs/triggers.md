# sznuper — Triggers

## Trigger Types

Each alert must have exactly one trigger.

### Interval

Runs the healthcheck periodically on a fixed schedule.

```yaml
trigger:
  interval: 30s
```

First run is immediate on daemon start, then repeats on the configured interval.

### Cron

Runs the healthcheck on a cron schedule. Uses [robfig/cron](https://github.com/robfig/cron) internally — no system cron involved. Supports standard 5-field and extended 6-field (with seconds) expressions.

```yaml
# Every 5 minutes
trigger:
  cron: "*/5 * * * *"

# Every day at 3am
trigger:
  cron: "0 3 * * *"

# Every Monday at noon
trigger:
  cron: "0 12 * * 1"

# With seconds (6-field): every hour at :30
trigger:
  cron: "0 30 * * * *"
```

`interval` is better for frequent healthchecks ("every 30 seconds"). `cron` is better for scheduled healthchecks ("every day at 3am").

### File Watch

Watches a file for changes using inotify. When new lines are appended, they are piped to the healthcheck via stdin.

```yaml
trigger:
  watch: /var/log/auth.log
```

Behavior:
- On daemon start, seeks to the end of the file. Only lines appearing after startup are processed.
- No state is persisted to disk. If the daemon restarts, anything that happened while it was down is missed.
- On normal append: reads new lines from stored offset, pipes to healthcheck via stdin, updates offset.
- On log rotation (inode change / `MOVE_SELF`): re-opens the path, resets offset to 0.
- On truncation (file size < stored offset): resets offset to 0, reads from start.
- Multiple new lines are batched into a single healthcheck invocation. The healthcheck receives all new lines on stdin at once.

### Pipe

Runs an arbitrary command and feeds its stdout to the healthcheck via stdin. Designed for real-time event streams — primarily `journalctl -f` — where inotify has no analog.

```yaml
trigger:
  pipe: "journalctl -f --since=now -u ssh -u sshd --output=json --output-fields=MESSAGE,__REALTIME_TIMESTAMP --no-pager"
```

Behavior:
- The command is run via `/bin/sh -c`. It is expected to run indefinitely (streaming output).
- Stdout chunks are buffered and flushed to the healthcheck as stdin. Multiple chunks arriving while a healthcheck is running are batched into the next invocation.
- If the command exits (non-zero or EOF), the pipe trigger restarts it after a 5-second backoff. This handles transient failures and system journal restarts.
- If the daemon context is cancelled, the subprocess is killed and the loop exits cleanly.

Example — real-time SSH failure detection via the systemd journal (works on any distro, including Debian 13+ without `auth.log`):

```yaml
- name: ssh_failures
  healthcheck: file://ssh_journal
  trigger:
    pipe: "journalctl -f --since=now -u ssh -u sshd --output=json --output-fields=MESSAGE,__REALTIME_TIMESTAMP --no-pager"
  args:
    alert_on: failure
    threshold_warn_count: 1
    threshold_crit_count: 20
  template: |
    SSH {{ healthcheck.event }}: {{ healthcheck.user }} from {{ healthcheck.host }} at {{ healthcheck.timestamp }}
  notify:
    - telegram
```

Advanced mode — pass additional journal fields through to the template:

```yaml
- name: ssh_failures
  healthcheck: file://ssh_journal
  trigger:
    pipe: "journalctl -f --since=now -u ssh -u sshd --output=json --output-fields=MESSAGE,__REALTIME_TIMESTAMP,_HOSTNAME --no-pager"
  args:
    alert_on: failure
    threshold_warn_count: 1
    threshold_crit_count: 20
    advanced: true
  template: |
    SSH {{ healthcheck.event }}: {{ healthcheck.user }} from {{ healthcheck.host }} at {{ healthcheck.timestamp }} ({{ healthcheck._HOSTNAME }})
  notify:
    - telegram
```

---

## Timeout and Concurrent Execution

### Timeout

An optional `timeout` field can be set per alert. If a healthcheck exceeds the timeout, the process is killed.

```yaml
- name: disk_check
  trigger:
    interval: 30s
  timeout: 10s              # optional
```

If not set, no timeout — the healthcheck runs as long as it wants.

### Concurrent Execution

Concurrency is tracked **per alert name**, not per healthcheck script. Two alerts using the same healthcheck with different args are independent.

| Trigger | Behavior when previous healthcheck still running |
|---|---|
| `interval` / `cron` | Blocks — waits for the current invocation to finish before scheduling the next tick. [TODO: kill previous and start new] |
| `watch` | Buffers new bytes; runs next invocation after current completes with all accumulated data |
| `pipe` | Buffers new stdout chunks; runs next invocation after current completes with all accumulated data |

For `watch` and `pipe` triggers, there is no queue — just a single byte buffer. New data keeps accumulating while a healthcheck is running. When the current invocation finishes, the entire buffer is flushed into the next invocation as a single stdin payload.

For multi-record healthchecks (using `--- records` / `--- record` output), the buffer gate waits until all records from a batch are fully processed (channel closed) before firing the next invocation. Buffered data accumulated during that time is flushed as one batch.
