import { describe, it, expect } from "vitest";
import { runSznuper } from "./helpers.js";

describe("sznuper version", () => {
  it("exits 0 and prints version string", async () => {
    const { stdout, exitCode } = await runSznuper(["version"]);
    expect(exitCode).toBe(0);
    expect(stdout).toMatch(/^sznuper .+ \(commit: .+, built: .+\)$/);
  });
});
