/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
import {
  requireLinuxRunner,
  selectLinuxBrowser,
  assertLinuxIdentity,
} from './install-qualified-firefox-linux.mjs';
import { manifestPath } from './install-qualified-browser.mjs';
const env = {
  CI: 'true',
  GITHUB_ACTIONS: 'true',
  RUNNER_ENVIRONMENT: 'github-hosted',
  RUNNER_TEMP: '/tmp/runner',
  GITHUB_OUTPUT: '/tmp/output',
};
test('Linux installer rejects local and wrong-platform execution before files or downloads', () => {
  requireLinuxRunner(env, 'linux', 'x64');
  for (const overrides of [
    { CI: '' },
    { GITHUB_ACTIONS: '' },
    { RUNNER_ENVIRONMENT: 'self-hosted' },
    { RUNNER_TEMP: 'relative' },
    { GITHUB_OUTPUT: '' },
  ])
    assert.throws(() => requireLinuxRunner({ ...env, ...overrides }, 'linux', 'x64'));
  assert.throws(() => requireLinuxRunner(env, 'darwin', 'arm64'));
  const result = spawnSync(
    process.execPath,
    [
      fileURLToPath(new URL('./install-qualified-firefox-linux.mjs', import.meta.url)),
      'firefox-stable-linux',
    ],
    { env: { ...process.env, CI: '', GITHUB_ACTIONS: '' }, encoding: 'utf8' },
  );
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /requires disposable GitHub-hosted CI/);
});
test('Linux Firefox rows pin official packages, checksums and exact vendor versions', async () => {
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  assert.equal(manifest.linuxBrowsers.length, 2);
  for (const row of manifest.linuxBrowsers) {
    assert.equal(selectLinuxBrowser(manifest, row.id), row);
    assertLinuxIdentity(row, `Mozilla Firefox ${row.archiveVersion}`);
    assert.throws(() => assertLinuxIdentity(row, 'Mozilla Firefox 1.0'));
    for (const override of [
      { url: 'https://example.test/archive' },
      { sha256: null },
      { platform: 'darwin' },
      { version: '1.0' },
    ])
      assert.throws(() => selectLinuxBrowser({ linuxBrowsers: [{ ...row, ...override }] }, row.id));
  }
  assert.match(manifest.linuxGeckodriver.sha256, /^[a-f0-9]{64}$/);
});
