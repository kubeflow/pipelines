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
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

test('browser preflight failures exit nonzero and retain machine-readable evidence', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'kfp-browser-evidence-'));
  try {
    for (const name of [
      'results.tap',
      'results.xml',
      'production-startup.json',
      'production-startup-failure.png',
    ]) {
      await writeFile(join(directory, name), 'stale passing report');
    }
    await writeFile(join(directory, 'workflow.json'), 'separate workflow evidence');
    const result = spawnSync(
      process.execPath,
      [fileURLToPath(new URL('./run-browser-suite.mjs', import.meta.url))],
      {
        env: {
          ...process.env,
          KFP_BROWSER: 'unsupported-engine',
          KFP_BROWSER_REPORT_DIR: directory,
        },
        encoding: 'utf8',
      },
    );
    assert.equal(result.status, 1, result.stderr);
    const report = JSON.parse(await readFile(join(directory, 'environment.json'), 'utf8'));
    assert.equal(report.status, 'failed');
    assert.equal(report.engine, 'unsupported-engine');
    assert.match(report.error, /Unsupported KFP_BROWSER/);
    assert.equal(report.browserVersion, undefined);
    assert.ok(report.finishedAt);
    for (const name of [
      'results.tap',
      'results.xml',
      'production-startup.json',
      'production-startup-failure.png',
    ]) {
      await assert.rejects(readFile(join(directory, name)), { code: 'ENOENT' });
    }
    assert.equal(
      await readFile(join(directory, 'workflow.json'), 'utf8'),
      'separate workflow evidence',
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
});
