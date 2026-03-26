import { execa } from "execa";
import { mkdtemp, rm, writeFile, chmod } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

export const SZNUPER_BIN =
  process.env.SZNUPER_BIN || path.resolve(import.meta.dirname, "../sznuper");

export async function runSznuper(
  args: string[],
  opts?: { env?: Record<string, string>; input?: string; cwd?: string },
) {
  const result = await execa(SZNUPER_BIN, args, {
    reject: false,
    env: opts?.env,
    input: opts?.input,
    cwd: opts?.cwd,
    timeout: 25_000,
  });
  return {
    stdout: result.stdout,
    stderr: result.stderr,
    exitCode: result.exitCode,
  };
}

export async function withTempDir(
  fn: (dir: string) => Promise<void>,
): Promise<void> {
  const dir = await mkdtemp(path.join(tmpdir(), "sznuper-e2e-"));
  try {
    await fn(dir);
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

export async function writeConfig(dir: string, yaml: string): Promise<string> {
  const configPath = path.join(dir, "config.yml");
  await writeFile(configPath, yaml, "utf-8");
  return configPath;
}

/**
 * Creates a shell script healthcheck that emits the given output lines.
 * Each line becomes an `echo` statement. Returns the absolute path.
 */
export async function makeTestHealthcheck(
  dir: string,
  name: string,
  script: string,
): Promise<string> {
  const scriptPath = path.join(dir, name);
  await writeFile(scriptPath, `#!/bin/sh\n${script}\n`, "utf-8");
  await chmod(scriptPath, 0o755);
  return scriptPath;
}

/**
 * Creates a minimal valid config YAML using a file:// healthcheck.
 */
export function minimalConfig(opts: {
  healthcheckPath: string;
  alertName?: string;
  template?: string;
  cooldown?: string;
}): string {
  const alertName = opts.alertName || "test_alert";
  const template = opts.template || "{{event.type}}";
  return `\
options:
  healthchecks_dir: /tmp
  cache_dir: /tmp/sznuper-e2e-cache
globals:
  hostname: test-host
services:
  logger:
    url: logger://
alerts:
  - name: ${alertName}
    healthcheck: "file://${opts.healthcheckPath}"
    triggers:
      - interval: 30s
    template: "${template}"
${opts.cooldown ? `    cooldown: ${opts.cooldown}\n` : ""}\
    notify:
      - logger
    events:
      healthy: [ok]
`;
}
