import { describe, it, expect } from "vitest";
import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import { runSznuper, withTempDir } from "./helpers.js";

describe("sznuper init", () => {
  it("creates config with --add-service in non-interactive mode", async () => {
    await withTempDir(async (dir) => {
      const outPath = path.join(dir, "config.yml");
      const { exitCode, stderr } = await runSznuper([
        "init",
        "--add-service",
        "logger:logger://",
        "--output",
        outPath,
        "--force",
      ]);

      expect(exitCode).toBe(0);
      expect(stderr).toContain("Config written to");

      const content = await readFile(outPath, "utf-8");
      expect(content).toContain("services:");
      expect(content).toContain("logger:");
      expect(content).toContain("url: logger://");
      expect(content).toContain("alerts:");
      expect(content).toContain("- logger");
    });
  });

  it("refuses to overwrite existing config without --force", async () => {
    await withTempDir(async (dir) => {
      const outPath = path.join(dir, "config.yml");

      // Create the config first
      await runSznuper([
        "init",
        "--add-service",
        "logger:logger://",
        "--output",
        outPath,
        "--force",
      ]);

      // Try again without --force
      const { exitCode, stderr } = await runSznuper([
        "init",
        "--add-service",
        "logger:logger://",
        "--output",
        outPath,
      ]);

      expect(exitCode).not.toBe(0);
      expect(stderr).toMatch(/already exists/i);
    });
  });

  it("creates config from base file with --from", async () => {
    await withTempDir(async (dir) => {
      // First create a base config
      const basePath = path.join(dir, "base.yml");
      await runSznuper([
        "init",
        "--add-service",
        "logger:logger://",
        "--output",
        basePath,
        "--force",
      ]);

      // Now init from that base
      const outPath = path.join(dir, "derived.yml");
      const { exitCode } = await runSznuper([
        "init",
        "--from",
        basePath,
        "--add-service",
        "test:logger://",
        "--output",
        outPath,
        "--force",
      ]);

      expect(exitCode).toBe(0);
      const content = await readFile(outPath, "utf-8");
      expect(content).toContain("- test");
    });
  });

  it("fails without TTY and without --add-service", async () => {
    await withTempDir(async (dir) => {
      const outPath = path.join(dir, "config.yml");
      const { exitCode, stderr } = await runSznuper([
        "init",
        "--output",
        outPath,
      ]);

      expect(exitCode).not.toBe(0);
      expect(stderr).toMatch(/no TTY detected/i);
    });
  });
});
