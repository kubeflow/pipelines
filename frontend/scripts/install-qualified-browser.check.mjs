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
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
import {
  assertAppIdentity,
  assertArchiveObject,
  browserUpdatePolicy,
  ownedBrowserMount,
  assertChecksum,
  manifestPath,
  requireHostedRunner,
  selectBrowser,
} from './install-qualified-browser.mjs';

const hosted = {
  CI: 'true',
  GITHUB_ACTIONS: 'true',
  RUNNER_ENVIRONMENT: 'github-hosted',
  RUNNER_TEMP: '/tmp/hosted-runner',
  GITHUB_OUTPUT: '/tmp/hosted-output',
};

test('installer rejects local, self-hosted, unsupported OS and absent runner paths before work', () => {
  for (const env of [
    {},
    { ...hosted, CI: 'false' },
    { ...hosted, GITHUB_ACTIONS: '' },
    { ...hosted, RUNNER_ENVIRONMENT: 'self-hosted' },
    { ...hosted, RUNNER_TEMP: '' },
    { ...hosted, RUNNER_TEMP: 'relative' },
    { ...hosted, GITHUB_OUTPUT: '' },
    { ...hosted, GITHUB_OUTPUT: 'relative' },
  ]) {
    assert.throws(() => requireHostedRunner(env, 'darwin', 'arm64'));
  }
  assert.throws(() => requireHostedRunner(hosted, 'linux', 'arm64'));
  assert.throws(() => requireHostedRunner(hosted, 'darwin', 'x64'));
  assert.doesNotThrow(() => requireHostedRunner(hosted, 'darwin', 'arm64'));
  const result = spawnSync(
    process.execPath,
    [fileURLToPath(new URL('./install-qualified-browser.mjs', import.meta.url)), 'chrome-stable'],
    {
      encoding: 'utf8',
      env: { ...process.env, CI: '', GITHUB_ACTIONS: '', RUNNER_ENVIRONMENT: '', RUNNER_TEMP: '' },
    },
  );
  assert.equal(result.status, 1);
  assert.match(result.stderr, /restricted to GitHub-hosted CI/);
});

test('checksum, exact version and signing team fail closed', () => {
  assert.throws(() => assertChecksum('expected', 'different'), /SHA-256/);
  assert.doesNotThrow(() => assertChecksum('same', 'same'));
  const row = { version: '154.0.8037.58', teamId: 'EQHXZ8M8AV' };
  assert.throws(
    () => assertAppIdentity(row, '154.0.8037.57', 'TeamIdentifier=EQHXZ8M8AV'),
    /version changed/,
  );
  assert.throws(() => assertAppIdentity(row, row.version, 'TeamIdentifier=ANOTHER'), /team/);
  assert.doesNotThrow(() =>
    assertAppIdentity(row, row.version, 'Executable=browser\nTeamIdentifier=EQHXZ8M8AV\n'),
  );
});

test('dated manifest separates installable exact artifacts from unresolved policy slots', async () => {
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  assert.match(manifest.resolvedAt, /^\d{4}-\d{2}-\d{2}$/);
  assert.equal(new Set(manifest.browsers.map((row) => row.id)).size, manifest.browsers.length);
  for (const row of manifest.browsers) {
    assert.equal(selectBrowser(manifest, row.id), row);
    assert.equal(new URL(row.url).protocol, 'https:');
    assert.equal(new URL(row.source).protocol, 'https:');
    assert.ok(['chromium', 'firefox'].includes(row.browser));
    assert.ok(['dmg', 'pkg', 'zip'].includes(row.kind));
    if (row.verification === 'cft-archive') assert.equal(row.teamId, null);
    else assert.match(row.teamId, /^[A-Z0-9]{10}$/);
    assert.equal(row.platform, 'darwin');
    assert.equal(row.architecture, 'arm64');
    if (row.sha256 !== null) assert.match(row.sha256, /^[a-f0-9]{64}$/);
    else assert.ok(row.id.startsWith('chrome-'));
    if (row.id.startsWith('chrome-testing-')) assert.equal(row.qualification, 'supplementary');
  }
  for (const gap of manifest.gaps) {
    assert.ok(gap.reason);
    assert.throws(() => selectBrowser(manifest, gap.id), /unresolved browser slot/);
  }
  assert.match(manifest.geckodriver.sha256, /^[a-f0-9]{64}$/);
  assert.throws(() => selectBrowser(manifest, '../escape'), /unresolved browser slot/);
});

test('only Edge receives the vendor-documented disabled-update policy', async () => {
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  for (const browser of manifest.browsers) {
    const policy = browserUpdatePolicy(browser);
    if (browser.id.startsWith('edge-')) {
      assert.equal(policy.updateDefault, 3);
      assert.equal(policy.path, '/Library/Managed Preferences/com.microsoft.EdgeUpdater.plist');
      assert.match(policy.plist, /<key>UpdateDefault<\/key><integer>3<\/integer>/);
      assert.match(policy.source, /^https:\/\/learn\.microsoft\.com\//);
    } else assert.equal(policy, null);
  }
});

test('CfT archive checks reject changed metadata, scope expansion and signing identity', async () => {
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  const browser = selectBrowser(manifest, 'chrome-testing-current');
  const metadata = browser.archiveObject;
  assert.doesNotThrow(() => assertArchiveObject(metadata, metadata));
  for (const key of ['bucket', 'name', 'generation', 'size', 'md5Hash']) {
    assert.throws(
      () => assertArchiveObject(metadata, { ...metadata, [key]: 'changed' }),
      /reviewed pin/,
    );
  }
  for (const overrides of [
    { qualification: 'policy' },
    { sha256: null },
    { url: 'https://example.test/archive.zip' },
    { teamId: 'EQHXZ8M8AV' },
    { archiveObject: { ...metadata, name: 'different.zip' } },
  ]) {
    assert.throws(
      () => selectBrowser({ browsers: [{ ...browser, ...overrides }] }, browser.id),
      /exact supplementary Google artifact pins/,
    );
  }
  const signature = 'Signature=adhoc\nTeamIdentifier=not set\nSealed Resources=none\n';
  assert.doesNotThrow(() => assertAppIdentity(browser, browser.version, signature));
  assert.throws(() => assertAppIdentity(browser, '0.0', signature), /version changed/);
  assert.throws(
    () => assertAppIdentity(browser, browser.version, 'TeamIdentifier=EQHXZ8M8AV'),
    /packaging changed/,
  );
  for (const actualBrowser of manifest.browsers.filter((row) => row.qualification === 'policy')) {
    assert.throws(
      () => assertAppIdentity(actualBrowser, actualBrowser.version, signature),
      /code signature team/,
    );
  }
});

test('Edge finalization accepts only its exact owned mount and skips incomplete setup', () => {
  const directory = '/tmp/runner/kfp-qualified-browsers/edge-previous';
  const browser = { appName: 'Microsoft Edge.app' };
  const record = {
    mounted: true,
    path: `${directory}/read-only-volume`,
    image: `${directory}/browser-read-only.dmg`,
  };
  assert.equal(ownedBrowserMount(directory, browser, undefined), null);
  assert.equal(ownedBrowserMount(directory, browser, { mounted: false }), null);
  assert.equal(
    ownedBrowserMount(directory, browser, record).app,
    `${record.path}/Microsoft Edge.app`,
  );
  for (const mutation of [{ path: '/Volumes/another-app' }, { image: '/tmp/another.dmg' }]) {
    assert.throws(
      () => ownedBrowserMount(directory, browser, { ...record, ...mutation }),
      /not owned/,
    );
  }
  assert.throws(
    () => ownedBrowserMount(directory, { appName: 'Firefox.app' }, record),
    /not owned/,
  );
  const result = spawnSync(
    process.execPath,
    [
      fileURLToPath(new URL('./install-qualified-browser.mjs', import.meta.url)),
      '--finalize',
      'edge-previous',
    ],
    {
      encoding: 'utf8',
      env: { ...process.env, CI: '', GITHUB_ACTIONS: '', RUNNER_ENVIRONMENT: '', RUNNER_TEMP: '' },
    },
  );
  assert.equal(result.status, 1);
  assert.match(result.stderr, /restricted to GitHub-hosted CI/);
});
