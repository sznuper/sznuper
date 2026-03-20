# Event-Based Healthcheck Protocol Redesign — Implementation Status

Post-implementation notes for [2026-03-14-event-based-redesign.md](2026-03-14-event-based-redesign.md).

---

## Completed

### Go Codebase (sznuper daemon)

All Go code has been migrated to the v2 event-based protocol. No v1 protocol code remains in the daemon.

| File | Changes |
|------|---------|
| `internal/healthcheck/parse.go` | Replaced `Parse`/`ParseMulti` with `ParseEvents`. New `Event` struct (Type, Fields, Arrays). Uses `--- event` delimiter. `type` field required. Empty output = zero events. |
| `internal/config/config.go` | Added `Events` struct (Healthy, OnUnmatched, Override). Added `EventOverride` (Template, Cooldown, Notify). Cooldown simplified from per-status struct to duration string. NotifyTarget rewritten (removed Template field, new YAML format with service name as map key). Removed `Cooldown` struct and `cooldownObj`. |
| `internal/cooldown/cooldown.go` | Replaced per-status (warning/critical) timers with per-event-type `map[string]*timer`. New API: `New(now)`, `Check(eventType, duration)`, `ResetAll()`. |
| `internal/notify/template.go` | Renamed `TemplateData.Healthcheck` → `Event`. Removed `statusEmoji` and `status_emoji`. Updated `Render` to register `event()` accessor. |
| `internal/notify/send.go` | Removed `Template` from `NotifyRef`. Template is now an alert/event concern, not a service concern. |
| `internal/runner/runner.go` | New processing pipeline: ParseEvents → per-event config resolution → state machine → cooldown → template → notify. Added `AlertState` for healthy/unhealthy tracking. Implements on_unmatched drop/default. Recovery resets all cooldowns. |
| `internal/runner/result.go` | `Status` → `EventType`. Added `Dropped` field. Removed `Output` (ordered lines). |
| `internal/healthcheck/builtin.go` | Lifecycle output changed to `--- event\ntype=started\n` (was `status=warning\nevent=started\n`). |
| `internal/scheduler/scheduler.go` | `buildCooldownState` simplified (creates empty per-event-type state). Builds `AlertState` when `events.healthy` is configured. |
| `internal/scheduler/pipe.go` | Updated to pass `RunOpts` instead of separate cooldown param. |
| `internal/scheduler/watch.go` | Updated to pass `RunOpts` instead of separate cooldown param. |
| `cmd/sznuper/run.go` | `Status` → `EventType` in output. `Output` → `Fields`. |
| `cmd/sznuper/start.go` | `status` → `event_type` in log attributes. |

### Default Configs

| File | Status |
|------|--------|
| `internal/initcmd/defaults/base.yml` | Updated. All 4 alerts (lifecycle, disk, memory, cpu) use v2 format. |
| `internal/initcmd/defaults/systemd.yml` | Updated. ssh_journal uses `events.override` + `on_unmatched: drop`. |

### Tests

All test files rewritten for v2. All pass.

| File | Status |
|------|--------|
| `internal/config/config_test.go` | Tests new Events, EventOverride, NotifyTarget format, simple cooldown. |
| `internal/healthcheck/parse_test.go` | Tests `ParseEvents` with `--- event` delimiter, type requirement, empty output, arrays. |
| `internal/cooldown/cooldown_test.go` | Tests per-event-type Check, ResetAll, Infinite, zero-duration. |
| `internal/notify/template_test.go` | Uses `event.*` namespace. No status_emoji tests. |
| `internal/notify/send_test.go` | No per-target template tests. Updated BuildTemplateData calls. |
| `internal/runner/runner_test.go` | Scripts emit `--- event` format. Tests empty output (zero events). |
| `internal/scheduler/scheduler_test.go` | Scripts emit `--- event` format. Watch/pipe tests check Fields instead of Output. |

---

## Remaining Work

### 1. Healthcheck C Binaries (blocking)

All 4 healthcheck binaries still output v1 protocol. They must be updated before the next release.

**`healthchecks/src/disk_usage.c`**
- Currently: outputs `status=ok|warning|critical` (single record, no delimiter)
- Needed: output `--- event\n` delimiter, `type=ok` / `type=high_usage` / `type=critical_usage` based on thresholds, remove `status=` line

**`healthchecks/src/cpu_usage.c`**
- Same pattern as disk_usage

**`healthchecks/src/memory_usage.c`**
- Same pattern as disk_usage

**`healthchecks/src/ssh_journal.c`**
- Currently: outputs `--- records` / `--- record` delimiters, `status=warning|critical`, `event=failure|login`
- Needed: replace `--- records` / `--- record` with `--- event`, replace `status=` with nothing, rename `event=` to `type=`, remove status determination logic, always emit `--- event` per-event (even for the zero-events case — just emit nothing)

**`healthchecks/src/sznuper.h`**
- No changes needed (status-agnostic utility code)

### 2. Documentation (blocking)

Every doc file in `sznuper/docs/` still describes the v1 protocol. All need updating.

**`docs/overview.md`** — v1 references throughout:
- Line 4: "returns a status and key-value output"
- Line 13: Glossary entry says healthcheck "returns `status` + key-value output"
- Line 18: Flow description says "outputs `status` (ok/warning/critical)"

**`docs/healthchecks.md`** — extensive v1 protocol documentation:
- Lines 69-73, 95-98, 123-127, 160-163: Mermaid flowcharts check `status` in output and branch on ok/warning/critical
- Lines 200-206: Example config uses `cooldown: {warning: 10m, critical: 1m, recovery: true}` and `{{healthcheck.*}}` template syntax
- Lines 248-258: "Reserved keys" table documents `status` as required, ok/warning/critical semantics
- Lines 268-333: Entire "Multi-Record Output" section documents `--- records` / `--- record` delimiters, global props section, per-record status — all v1
- Lines 362-378: "Documenting Healthcheck Interfaces" example uses status-based output and logic

**`docs/configuration.md`** — v1 config examples:
- Lines 85-90: Per-status cooldown format (`warning: 10m, critical: 1m, recovery: true`)
- Lines 89, 97, 105, 116, 126: `{{healthcheck.*}}` template syntax
- Lines 130-132: Per-service notify with `service:` key and per-target `template` field
- Line 132: Template conditional using `healthcheck.status`

**`docs/notifications.md`** — v1 template system:
- Lines 8-19: Template variables table uses `{{healthcheck.*}}` and documents `status_emoji`
- Lines 23, 29, 33-34: Example templates use `healthcheck.status`, `healthcheck.status_emoji`
- Lines 50-84: "Per-Service Template Overrides" section documents per-target template feature (removed in v2)
- Lines 64-79: Examples use per-status cooldown, `{{healthcheck.*}}` syntax, per-target templates
- Lines 136-141: Per-alert service override uses `service:` key format

**`docs/cooldown.md`** — entirely v1:
- Entire file describes per-status (warning/critical) cooldown with recovery flag
- Lines 17-22: Per-status config format
- Lines 34-73: State machine diagram with warning/critical/recovery regions
- Lines 79-86: Behavior description references `warning`/`critical`/`ok` status values
- Lines 92-124: Example timelines use `status →` format

**`docs/triggers.md`** — minor v1 references:
- Lines 85-87, 103-104: Template examples use `healthcheck.event`, `healthcheck.host` (should be `event.*`)
- Line 138: References `--- records` / `--- record` output format

**`docs/cli.md`** — v1 output examples:
- Lines 47-53, 60-68: `sznuper run` output shows `status=warning` and `Output:` section (v2 shows `type=` and `Fields:`)

### 3. CLAUDE.md (non-blocking)

`sznuper/CLAUDE.md` line 13 says:
> User healthchecks: Any executable that outputs `KEY=VALUE` lines to stdout with a required `status` key

Should reference `--- event` blocks with required `type` field instead.

---

## Migration Order

1. **Healthcheck binaries** — update C source, rebuild, compute new sha256 hashes
2. **Default config sha256 values** — update `base.yml` and `systemd.yml` with new hashes after rebuilding
3. **Documentation** — rewrite all doc files for v2 protocol
4. **CLAUDE.md** — update project description
