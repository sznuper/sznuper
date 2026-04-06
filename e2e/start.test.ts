import { describe, it, expect } from "vitest";
import { execa } from "execa";
import { setTimeout as sleep } from "node:timers/promises";
import {
  SZNUPER_BIN,
  withTempDir,
  writeConfig,
  makeTestHealthcheck,
} from "./helpers.js";

describe("sznuper start", () => {
  it("starts and stops gracefully on SIGINT", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "ok_check",
        `echo "--- event"\necho "type=ok"`,
      );
      const configPath = await writeConfig(
        dir,
        `\
options:
  healthchecks_dir: /tmp
  cache_dir: /tmp/sznuper-e2e-cache
globals:
  hostname: test-host
channels:
  logger:
    url: logger://
alerts:
  - name: test_alert
    healthcheck: "file://${hcPath}"
    triggers:
      - interval: 30s
    template: "{{event.type}}"
    notify:
      - logger
    events:
      healthy: [ok]
`,
      );

      const proc = execa(SZNUPER_BIN, ["start", "--config", configPath], {
        reject: false,
        timeout: 15_000,
      });

      await sleep(2000);
      proc.kill("SIGINT");

      const result = await proc;

      expect(result.stderr).toContain("sznuper daemon starting");
      expect(result.stderr).toContain("sznuper daemon stopped");
    });
  });

  it("starts with --dry-run and processes alerts", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "ok_check",
        `echo "--- event"\necho "type=ok"`,
      );
      const configPath = await writeConfig(
        dir,
        `\
options:
  healthchecks_dir: /tmp
  cache_dir: /tmp/sznuper-e2e-cache
globals:
  hostname: test-host
channels:
  logger:
    url: logger://
alerts:
  - name: test_alert
    healthcheck: "file://${hcPath}"
    triggers:
      - interval: 1s
    template: "{{event.type}}"
    notify:
      - logger
    events:
      healthy: [ok]
`,
      );

      const proc = execa(
        SZNUPER_BIN,
        ["start", "--dry-run", "--config", configPath],
        { reject: false, timeout: 15_000 },
      );

      await sleep(3000);
      proc.kill("SIGINT");

      const result = await proc;

      expect(result.stderr).toContain("sznuper daemon starting");
      expect(result.stderr).toContain("alert completed");
    });
  });

  it("fails with nonexistent config", async () => {
    const result = await execa(
      SZNUPER_BIN,
      ["start", "--config", "/nonexistent/config.yml"],
      { reject: false },
    );

    expect(result.exitCode).not.toBe(0);
  });
});
