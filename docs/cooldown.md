# sznuper — Cooldown

Cooldown suppresses repeated notifications for the same status. Each status (`warning`, `critical`) has its own independent timer. Healthchecks always run regardless of cooldown state — cooldown only affects whether a notification is sent.

## Config

```yaml
# Simple — same cooldown for all statuses
cooldown: 5m

# Per-status — independent timers
cooldown:
  warning: 10m
  critical: 1m
  recovery: true       # default: false
```

When `cooldown: 5m` (simple form), it's shorthand for the same value applied to each status independently.

## Behavior

- Each status (`warning`, `critical`) has its own independent cooldown timer.
- When a healthcheck returns `warning`/`critical` and that status's cooldown is not active: send notification, start cooldown timer.
- When a healthcheck returns `warning`/`critical` and that status's cooldown is active: suppress notification.
- When a healthcheck returns `ok` after any previous `warning`/`critical` and `recovery: true`: send recovery notification, reset all cooldown timers.
- When a healthcheck returns `ok` after any previous `warning`/`critical` and `recovery: false`: no notification, reset all cooldown timers.
- When a healthcheck returns `ok` and previous result was also `ok`: nothing.
- If `status` is missing (broken healthcheck): logged as error, does not trigger cooldown.
- If the healthcheck outputs a status that has no matching cooldown key, it falls back to the simple `cooldown` value. If neither exists, no cooldown.

## Example Timeline

```
cooldown:
  warning: 10m
  critical: 1m
  recovery: true

t=0:00  healthcheck → ok       → nothing (no previous alert)
t=0:30  healthcheck → warning  → notify, start cooldown(warning, 10m)
t=1:00  healthcheck → warning  → suppress (warning cooldown active)
t=1:30  healthcheck → critical → notify (critical has its own timer), start cooldown(critical, 1m)
t=2:00  healthcheck → critical → suppress (critical cooldown active)
t=2:30  healthcheck → critical → notify (critical cooldown expired), restart cooldown
t=3:00  healthcheck → ok       → notify recovery, reset all cooldowns
t=3:30  healthcheck → ok       → nothing (already recovered)
t=4:00  healthcheck → warning  → notify (cooldowns were reset, fresh incident)
```
