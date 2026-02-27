# Sznuper

Sznuper is a single-binary monitoring daemon for Linux servers. It runs checks on your system and sends notifications when something needs attention — disk filling up, a service going down, a suspicious SSH login.

No dashboard. No database. No UI. Just a YAML config, a binary, and alerts delivered to Telegram, Slack, email, or any service supported by [Shoutrrr](https://containrrr.dev/shoutrrr/).

## Repositories

- **[sznuper](https://github.com/sznuper/sznuper)** — the daemon, written in Go
- **[checks](https://github.com/sznuper/checks)** — official checks, written in C with Cosmopolitan Libc for cross-architecture portability
