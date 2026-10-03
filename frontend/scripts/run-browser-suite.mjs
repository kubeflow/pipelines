/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import { arch, platform, release } from 'node:os';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, firefox, webkit } from 'playwright';

const frontend = fileURLToPath(new URL('../', import.meta.url));
const directory = resolve(frontend, process.env.KFP_BROWSER_REPORT_DIR || 'browser-results');
const files = [
  'production-bundle.smoke.mjs',
  'ui-modernization-runs.smoke.mjs',
  'ui-modernization-run-details.smoke.mjs',
  'ui-modernization-pipelines.smoke.mjs',
  'ui-modernization-workflows.smoke.mjs',
  'ui-modernization-artifacts.smoke.mjs',
  'ui-modernization-comparison.smoke.mjs',
  'ui-modernization-graph.smoke.mjs',
].map((name) => `scripts/${name}`);
const engineName = process.env.KFP_BROWSER || 'chromium';
const report = {
  sourceSha: process.env.GITHUB_SHA || null,
  startedAt: new Date().toISOString(),
  platform: platform(),
  release: release(),
  arch: arch(),
  node: process.version,
  engine: engineName,
  channel:
    engineName === 'chromium' && !process.env.KFP_BROWSER_EXECUTABLE_PATH
      ? process.env.PLAYWRIGHT_CHANNEL || null
      : null,
  executablePath: process.env.KFP_BROWSER_EXECUTABLE_PATH || null,
  files,
  status: 'running',
};
await mkdir(directory, { recursive: true });
const save = () =>
  writeFile(resolve(directory, 'environment.json'), `${JSON.stringify(report, null, 2)}\n`);
await save();
try {
  // A failed preflight must never retain a previous run's successful test reports.
  for (const name of [
    'results.tap',
    'results.xml',
    'production-startup.json',
    'production-startup-failure.png',
  ]) {
    await rm(resolve(directory, name), { force: true });
  }
  const engine = { chromium, firefox, webkit }[engineName];
  assert.ok(engine, `Unsupported KFP_BROWSER: ${engineName}`);
  const browser = await engine.launch({
    ...(process.env.KFP_BROWSER_EXECUTABLE_PATH
      ? { executablePath: process.env.KFP_BROWSER_EXECUTABLE_PATH }
      : {
          channel:
            engineName === 'chromium' ? process.env.PLAYWRIGHT_CHANNEL || undefined : undefined,
        }),
  });
  try {
    report.browserVersion = browser.version();
    if (process.env.KFP_EXPECTED_BROWSER_VERSION) {
      assert.equal(report.browserVersion, process.env.KFP_EXPECTED_BROWSER_VERSION);
    }
  } finally {
    await browser.close();
  }
  report.playwrightVersion = JSON.parse(
    await readFile(new URL('../node_modules/playwright/package.json', import.meta.url), 'utf8'),
  ).version;
  // These hashes identify the exact shared build, regardless of the runner OS.
  const build = resolve(frontend, 'build');
  const entries = (await readdir(build, { recursive: true, withFileTypes: true }))
    .filter((entry) => entry.isFile())
    .map((entry) => resolve(entry.parentPath, entry.name))
    .sort();
  assert.ok(entries.length, 'Production bundle must not be empty');
  report.assets = await Promise.all(
    entries.map(async (path) => ({
      path: path.slice(build.length + 1).replaceAll('\\', '/'),
      sha256: createHash('sha256')
        .update(await readFile(path))
        .digest('hex'),
    })),
  );
  await save();
  console.log(`Running ${files.length} suites in ${engineName} ${report.browserVersion}`);
  const result = spawnSync(
    process.execPath,
    [
      '--test',
      '--test-concurrency=1',
      '--test-reporter=tap',
      `--test-reporter-destination=${resolve(directory, 'results.tap')}`,
      '--test-reporter=junit',
      `--test-reporter-destination=${resolve(directory, 'results.xml')}`,
      ...files,
    ],
    {
      cwd: frontend,
      env: { ...process.env, KFP_EXPECTED_BROWSER_VERSION: report.browserVersion },
      stdio: 'inherit',
    },
  );
  if (result.error) throw result.error;
  process.stdout.write(await readFile(resolve(directory, 'results.tap'), 'utf8'));
  report.exitCode = result.status;
  report.signal = result.signal;
  report.status = result.status === 0 ? 'passed' : 'failed';
  process.exitCode = result.status === 0 ? 0 : 1;
} catch (error) {
  report.status = 'failed';
  report.error = error.stack || String(error);
  console.error(error);
  process.exitCode = 1;
} finally {
  report.finishedAt = new Date().toISOString();
  await save();
}
