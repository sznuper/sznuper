import { describe, it, expect } from "vitest";
import { createHash } from "node:crypto";
import { writeFile } from "node:fs/promises";
import path from "node:path";
import { runSznuper, withTempDir } from "./helpers.js";

describe("sznuper hash", () => {
  it("prints correct SHA256 for a file", async () => {
    await withTempDir(async (dir) => {
      const filePath = path.join(dir, "testfile");
      const content = "hello sznuper\n";
      await writeFile(filePath, content);

      const expected = createHash("sha256").update(content).digest("hex");
      const { stdout, exitCode } = await runSznuper(["hash", filePath]);

      expect(exitCode).toBe(0);
      expect(stdout.trim()).toBe(expected);
    });
  });

  it("fails for nonexistent file", async () => {
    const { exitCode, stderr } = await runSznuper(["hash", "/nonexistent"]);
    expect(exitCode).not.toBe(0);
    expect(stderr).toMatch(/no such file|not found/i);
  });

  it("fails with no arguments", async () => {
    const { exitCode, stderr } = await runSznuper(["hash"]);
    expect(exitCode).not.toBe(0);
    expect(stderr).toMatch(/requires|accepts|argument/i);
  });
});
