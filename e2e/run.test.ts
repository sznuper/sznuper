import { describe, it, expect } from "vitest";
import {
  runSznuper,
  withTempDir,
  writeConfig,
  makeTestHealthcheck,
  minimalConfig,
} from "./helpers.js";

describe("sznuper run", () => {
  it("dry-run shows expected output fields", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "ok_check",
        `echo "--- event"\necho "type=ok"\necho "usage_percent=42.0"`,
      );
      const configPath = await writeConfig(
        dir,
        minimalConfig({
          healthcheckPath: hcPath,
          template: "[{{event.type}}] test",
        }),
      );

      const { stdout, exitCode } = await runSznuper([
        "run",
        "test_alert",
        "--dry-run",
        "--config",
        configPath,
      ]);

      expect(exitCode).toBe(0);
      expect(stdout).toContain("\u2713"); // checkmark
      expect(stdout).toContain("Fields:");
      expect(stdout).toContain("type=ok");
      expect(stdout).toContain("Rendered:");
      expect(stdout).toContain("Would notify:");
      expect(stdout).toContain("logger");
    });
  });

  it("run without --dry-run notifies via logger", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "ok_check",
        `echo "--- event"\necho "type=ok"`,
      );
      const configPath = await writeConfig(
        dir,
        minimalConfig({ healthcheckPath: hcPath }),
      );

      const { stdout, exitCode } = await runSznuper([
        "run",
        "test_alert",
        "--config",
        configPath,
      ]);

      expect(exitCode).toBe(0);
      expect(stdout).toContain("\u2713");
      expect(stdout).toContain("Notified:");
      expect(stdout).toContain("logger");
    });
  });

  it("fails for unknown alert name", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "ok_check",
        `echo "--- event"\necho "type=ok"`,
      );
      const configPath = await writeConfig(
        dir,
        minimalConfig({ healthcheckPath: hcPath }),
      );

      const { exitCode, stderr } = await runSznuper([
        "run",
        "nonexistent_alert",
        "--config",
        configPath,
      ]);

      expect(exitCode).not.toBe(0);
      expect(stderr).toContain("not found");
    });
  });

  it("shows error for unresolvable healthcheck", async () => {
    await withTempDir(async (dir) => {
      const configPath = await writeConfig(
        dir,
        minimalConfig({ healthcheckPath: "/nonexistent/check" }),
      );

      const { stdout, exitCode } = await runSznuper([
        "run",
        "test_alert",
        "--dry-run",
        "--config",
        configPath,
      ]);

      expect(exitCode).not.toBe(0);
      expect(stdout).toContain("\u2717"); // X mark
      expect(stdout).toMatch(/Error/i);
    });
  });

  it("handles multi-event healthcheck", async () => {
    await withTempDir(async (dir) => {
      const hcPath = await makeTestHealthcheck(
        dir,
        "multi_check",
        `echo "--- event"\necho "type=warn"\necho "level=high"\necho "--- event"\necho "type=ok"\necho "level=normal"`,
      );
      const configPath = await writeConfig(
        dir,
        minimalConfig({ healthcheckPath: hcPath }),
      );

      const { stdout, exitCode } = await runSznuper([
        "run",
        "test_alert",
        "--dry-run",
        "--config",
        configPath,
      ]);

      expect(exitCode).toBe(0);
      expect(stdout).toContain("type=warn");
      expect(stdout).toContain("type=ok");
    });
  });
});
