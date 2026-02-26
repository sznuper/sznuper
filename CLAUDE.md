# Barker

Single-binary monitoring daemon for Linux. Runs checks, sends notifications via Shoutrrr. No database, no UI — just YAML config and a process.

## Design Philosophy

Barker is intentionally dumb. The daemon is a generic executor — it runs any executable, reads `KEY=VALUE` stdout, and routes notifications. There are no special code paths for "official" vs "user" checks. Official checks are just pre-built conveniences distributed through the same `https://` + sha256 mechanism available to anyone. A user who writes their own checks from scratch uses the exact same interface and gets the same capabilities. Everything in the "official" architecture is user-extensible by design.

## Architecture

- **Daemon:** Go — generic check executor, doesn't interpret check logic
- **Official checks:** C compiled with Cosmopolitan Libc (portable single binaries, direct syscalls, zero dependencies) — a convenience, not a requirement
- **User checks:** Any executable that outputs `KEY=VALUE` lines to stdout with a required `status` key — first-class, identical interface to official checks

## Docs

Specification lives in `docs/`:

- `overview.md` — project pitch, glossary, flow
- `configuration.md` — config file location, layout, full config structure
- `checks.md` — URI schemes (file://, https://), sha256 verification, check I/O interface, lifecycle flowcharts
- `triggers.md` — interval, cron, file watch triggers, timeout, concurrency
- `cooldown.md` — cooldown config, behavior, timeline example
- `cli.md` — init, start, validate, run, hash commands
- `notifications.md` — templates, Sprig functions, variable interpolation, service options, Shoutrrr delivery
