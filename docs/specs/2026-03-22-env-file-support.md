# Environment Variable File Support — Spec

## Context & Motivation

Sznuper needs secrets (API tokens, chat IDs) to talk to notification services. Config already supports `${VAR}` interpolation via `a8m/envsubst`, which means configs can be shared (e.g. via `sznuper init --from <url>`) with secrets kept per-machine. But there's no standard for where those secrets live.

Currently the systemd service has `EnvironmentFile=-/etc/sznuper/.env`, but nothing creates that file. Non-systemd users must `source .env` or export variables manually. The result is a different setup story for every deployment mode, and no tooling to help.

**Goal:** Provide an explicit, portable mechanism for loading environment variables from a file across all deployment modes.

**Non-goals:** Auto-discovery of `.env` files, secrets encryption (future work), `_FILE` suffix convention.

---

## Research Summary

We surveyed 11 comparable tools to understand how self-hosted daemons handle secrets:

| Tool | Loads .env? | ${VAR} in config? | Approach |
|---|---|---|---|
| **Gatus** | No | Yes (`os.ExpandEnv`) | Env var interpolation in YAML, relies on environment |
| **Caddy** | Via `--envfile` flag | Yes (`{$VAR}`) | Explicit flag, recommended via systemd `EnvironmentFile` |
| **Grafana** | No | Yes (`$__env{}`, `GF_*`) | Env var overrides + `__FILE` suffix for secrets |
| **Alertmanager** | No | No | Plaintext in YAML, `_file` suffix for some fields |
| **Vaultwarden** | Yes (auto from CWD) | N/A (pure env vars) | Auto-loads `.env`, `_FILE` suffix |
| **Gotify** | No | No | Plaintext in YAML or `GOTIFY_*` env |
| **ntfy** | No | No | Plaintext in YAML or `NTFY_*` env |
| **Healthchecks.io** | Yes (`docker/.env`) | N/A (pure env vars) | Env vars + `_FILE` variants |
| **Traefik** | No | Dynamic config only | `TRAEFIK_*` env vars for static config |
| **Watchtower** | No | N/A (no config file) | Env vars + file references for secrets |
| **Uptime Kuma** | Yes (v1.20+) | N/A (web UI/SQLite) | `.env` in root directory |

**Key finding:** No Go daemon auto-loads `.env` files by convention. Tools that support env file loading do so via explicit flags (Caddy) or as pure-env-var apps (Vaultwarden, Healthchecks.io). The dominant pattern for YAML-config daemons is env var interpolation with the deployment environment providing the variables.

**Closest match to sznuper: Gatus** — same pattern (YAML config + `${VAR}` interpolation), relies on the environment to provide variables.

**Chosen model: Caddy** — explicit `--env-file` flag passed to the binary. No auto-discovery, no magic. The systemd unit file and install tooling handle wiring it up.

---

## Decision

Two complementary mechanisms, matching Caddy's approach:

### 1. Systemd: `EnvironmentFile=` in the unit file

The systemd service file includes `EnvironmentFile=-/etc/sznuper/.env` (already present today). Systemd loads the variables before sznuper starts — sznuper doesn't need to know about `.env` at all. The `-` prefix makes the file optional (service starts fine without it).

Users who need to customize the `.env` path or add more environment variables can use `systemctl edit sznuper` to create a drop-in override, keeping the base service file untouched.

### 2. Non-systemd: `--env-file` flag

Add an `--env-file` flag to all commands that load config (`sznuper start`, `sznuper run`, and `sznuper validate`) for users running without systemd (manual execution, cron, containers, OpenRC).

- Accepts a path to a `.env` file (`KEY=VALUE` format, one per line, `#` comments)
- Loaded before config parsing (before `envsubst` runs)
- **Precedence:** process environment variables override `.env` file values (same as Docker Compose)
- If the flag is passed and the file does not exist, exit with an error
- If the flag is not passed, no `.env` loading occurs — current behavior preserved

### Why Not Auto-Load?

- **Explicitness:** the flag makes it obvious where variables come from — no "which `.env` did it read?" debugging
- **Daemon convention:** Go daemons don't auto-load `.env` files; the deployment environment provides env vars
- **Compatibility:** works identically across systemd, Docker, cron, manual execution — the flag is the same everywhere
- **Simplicity:** no search paths, no precedence between multiple `.env` locations

### Why Not a Config Option?

Chicken-and-egg: config contains `${VAR}` references that need `.env` to be loaded first. A two-pass parse (read config for the path, load `.env`, re-parse with envsubst) is fragile — the path itself could contain variables. The flag avoids this by resolving before any config parsing.

---

## Future Work

These items depend on this spec but are out of scope for the initial implementation:

- **`install.sh` (dist repo) creates `.env`** — with `600` permissions next to the config (systemd loads it via `EnvironmentFile=` in the unit file). Can scaffold placeholder variable names based on which services were chosen during init.
- **Permission warning** — `sznuper start` warns if the `--env-file` file is world-readable
- **Validation** — `sznuper validate` checks that `${VAR}` references in config resolve to non-empty values
- **Secrets encryption** — `sznuper secrets set KEY` that encrypts at rest and decrypts at startup (must not conflict with `.env` loading)
