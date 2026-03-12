# Sznuper

A lightweight server monitor that runs healthchecks and sends notifications — Discord, Slack, Telegram, Teams, and more.

No dashboard. No database. No UI. Just a YAML config, a binary, and alerts delivered to Telegram, Slack, email, or any service supported by [Shoutrrr](https://containrrr.dev/shoutrrr/).

## Repositories

- **[sznuper](https://github.com/sznuper/sznuper)** — the daemon, written in Go
- **[healthchecks](https://github.com/sznuper/healthchecks)** — official healthchecks, written in C with Cosmopolitan Libc for cross-architecture portability
