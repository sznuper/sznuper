# sznuper — Overview

A lightweight server monitor that runs healthchecks and sends notifications — Discord, Slack, Telegram, Teams, and more.

It runs a set of healthchecks — some bundled (CPU, memory, disk, SSH logins via journald), some user-defined (any executable that emits events as key-value output). When a healthcheck triggers, it routes notifications through one or more services according to the config. It handles cooldowns, per-event-type overrides, and templating so raw scripts don't have to.

---

## Glossary

| Term            | Definition                                                                                                                                                |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Healthcheck** | A script/binary that inspects something and emits events (`--- event` blocks with `type` + key-value payload). Reusable. Lives on disk or is fetched via HTTPS. |
| **Alert**       | A configured instance of a healthcheck. Ties together a healthcheck, its arguments, trigger, cooldown, template, and where to send notifications. |
| **Notification**| A message sent to a service when an alert triggers.                                                                                                       |
| **Service**     | A configured notification destination. Defined by a Shoutrrr URL with options. Examples: Telegram, Slack, webhook, logger.                                |

**Flow:** A healthcheck runs → emits events (`--- event` blocks with `type` + key-value payload) → each event is resolved against the alert's event config → state machine checks healthy/unhealthy transition → cooldown is evaluated → a notification is rendered from the template → sent to services with merged params.
