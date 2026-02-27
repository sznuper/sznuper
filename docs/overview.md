# sznuper — Overview

sznuper is a single-binary daemon that sits on a Linux host, watches for system events, and sends notifications to the places you care about. No dashboard, no database, no web UI — just a YAML config file and a process.

It runs a set of healthchecks — some bundled (CPU, memory, disk, SSH logins, systemd unit failures), some user-defined (any executable that returns a status and key-value output). When a healthcheck triggers, it routes a notification through one or more services according to the config. It handles cooldowns, per-alert overrides, and templating so raw scripts don't have to.

**One-liner pitch:** "Monit meets ntfy — a single binary that replaces your scattered cron alert scripts with a config-driven daemon and modern notification providers."

---

## Glossary

| Term            | Definition                                                                                                                                                |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Healthcheck** | A script/binary that inspects something and returns `status` + key-value output. Reusable. Lives on disk or is fetched via HTTPS.                 |
| **Alert**       | A configured instance of a healthcheck. Ties together a healthcheck, its arguments, trigger, cooldown, template, and where to send notifications. |
| **Notification**| A message sent to a service when an alert triggers.                                                                                                       |
| **Service**     | A configured notification destination. Defined by a Shoutrrr URL with options. Examples: Telegram, Slack, webhook, logger.                                |

**Flow:** A healthcheck runs → outputs `status` (ok/warning/critical) + key-value data → if not `ok`, the alert triggers → a notification is rendered from the template → sent to services with merged options.
