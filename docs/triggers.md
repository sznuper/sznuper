# barker — Triggers

## Trigger Types

Each alert must have exactly one trigger. Config validation rejects alerts with multiple triggers or no trigger.

### Interval

Runs the check periodically on a fixed schedule.

```yaml
trigger:
  interval: 30s
```

### Cron

Runs the check on a cron schedule. Uses [robfig/cron](https://github.com/robfig/cron) internally — no system cron involved. Supports standard 5-field and extended 6-field (with seconds) expressions.

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

`interval` is better for frequent checks ("every 30 seconds"). `cron` is better for scheduled checks ("every day at 3am").

### File Watch

Watches a file for changes using inotify. When new lines are appended, they are piped to the check via stdin.

```yaml
trigger:
  watch: /var/log/auth.log
```

Behavior:
- On daemon start, seeks to the end of the file. Only lines appearing after startup are processed.
- No state is persisted to disk. If the daemon restarts, anything that happened while it was down is missed.
- On normal append: reads new lines from stored offset, pipes to check via stdin, updates offset.
- On log rotation (inode change / `MOVE_SELF`): re-opens the path, resets offset to 0.
- On truncation (file size < stored offset): resets offset to 0, reads from start.
- Multiple new lines are batched into a single check invocation. The check receives all new lines on stdin at once.

---

## Timeout and Concurrent Execution

### Timeout

An optional `timeout` field can be set per alert. If a check exceeds the timeout, the process is killed.

```yaml
- name: disk_check
  trigger:
    interval: 30s
  timeout: 10s              # optional
```

If not set, no timeout — the check runs as long as it wants.

### Concurrent Execution

Concurrency is tracked **per alert name**, not per check script. Two alerts using the same check with different args are independent.

| Trigger | Behavior when previous check still running |
|---|---|
| `interval` / `cron` | Kill previous process, start new invocation |
| `watch` | Buffer new lines, run after previous completes. If previous exceeds `timeout`, kill it and flush buffer into new invocation |

For watch triggers, there is no queue — just a single line buffer. New lines keep appending to the buffer while a check is running. When the current check finishes (or is killed by timeout), all buffered lines are flushed into the next invocation.
