import { describe, it, expect } from "vitest";
import {
  runSznuper,
  withTempDir,
  writeConfig,
  makeTestHealthcheck,
  minimalConfig,
} from "./helpers.js";

describe("sznuper validate", () => {
  it("passes for valid config with file:// healthcheck", async () => {
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
        "validate",
        "--config",
        configPath,
      ]);

      expect(exitCode).toBe(0);
      expect(stdout).toContain("\u2713"); // checkmark
      expect(stdout).toContain("test_alert");
    });
  });

  it("fails for config with missing healthcheck file", async () => {
    await withTempDir(async (dir) => {
      const configPath = await writeConfig(
        dir,
        minimalConfig({ healthcheckPath: "/nonexistent/check" }),
      );

      const { stdout, exitCode } = await runSznuper([
        "validate",
        "--config",
        configPath,
      ]);

      expect(exitCode).not.toBe(0);
      expect(stdout).toContain("\u2717"); // X mark
    });
  });

  it("fails for invalid YAML", async () => {
    await withTempDir(async (dir) => {
      const configPath = await writeConfig(dir, "{{{{invalid yaml");

      const { exitCode } = await runSznuper([
        "validate",
        "--config",
        configPath,
      ]);

      expect(exitCode).not.toBe(0);
    });
  });

  it("fails for nonexistent config file", async () => {
    const { exitCode, stderr } = await runSznuper([
      "validate",
      "--config",
      "/nonexistent/config.yml",
    ]);

    expect(exitCode).not.toBe(0);
    expect(stderr).toMatch(/no such file|not found|does not exist/i);
  });
});
