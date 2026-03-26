import { describe, it, expect } from "vitest";
import {
  runSznuper,
  withTempDir,
  writeConfig,
  makeTestHealthcheck,
  minimalConfig,
} from "./helpers.js";

describe("exit codes", () => {
  it("sznuper (no command) exits 0", async () => {
    const { exitCode } = await runSznuper([]);
    expect(exitCode).toBe(0);
  });

  it("sznuper --help exits 0", async () => {
    const { exitCode, stdout } = await runSznuper(["--help"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("sznuper");
  });

  it("sznuper unknown-command exits non-zero", async () => {
    const { exitCode } = await runSznuper(["unknown-command"]);
    expect(exitCode).not.toBe(0);
  });

  it("sznuper validate --config <bad> exits non-zero", async () => {
    await withTempDir(async (dir) => {
      const configPath = await writeConfig(dir, "invalid: {{yaml");
      const { exitCode } = await runSznuper([
        "validate",
        "--config",
        configPath,
      ]);
      expect(exitCode).not.toBe(0);
    });
  });

  it("sznuper run bad_name exits non-zero", async () => {
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
      const { exitCode } = await runSznuper([
        "run",
        "nonexistent",
        "--config",
        configPath,
      ]);
      expect(exitCode).not.toBe(0);
    });
  });

  it("sznuper start --config <nonexistent> exits non-zero", async () => {
    const { exitCode } = await runSznuper([
      "start",
      "--config",
      "/nonexistent/config.yml",
    ]);
    expect(exitCode).not.toBe(0);
  });
});
