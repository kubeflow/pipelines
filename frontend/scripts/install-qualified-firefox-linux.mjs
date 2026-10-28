/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { appendFile, mkdir, readFile, realpath, writeFile } from 'node:fs/promises';
import { isAbsolute, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { manifestPath } from './install-qualified-browser.mjs';

export function requireLinuxRunner(env, platform, arch) {
  assert.ok(
    env.CI === 'true' &&
      env.GITHUB_ACTIONS === 'true' &&
      env.RUNNER_ENVIRONMENT === 'github-hosted',
    'Firefox installation requires disposable GitHub-hosted CI',
  );
  assert.equal(platform, 'linux');
  assert.equal(arch, 'x64');
  assert.ok(env.RUNNER_TEMP && isAbsolute(env.RUNNER_TEMP));
  assert.ok(env.GITHUB_OUTPUT && isAbsolute(env.GITHUB_OUTPUT));
}
export function selectLinuxBrowser(manifest, id) {
  const browser = manifest.linuxBrowsers.find((row) => row.id === id);
  assert.ok(browser, 'Unknown Linux browser');
  assert.match(browser.id, /^firefox-(stable|esr)-linux$/);
  assert.equal(browser.platform, 'linux');
  assert.equal(browser.architecture, 'x64');
  assert.match(browser.sha256, /^[a-f0-9]{64}$/);
  assert.match(browser.archiveVersion, /^\d+\.\d+(?:\.\d+)?(?:esr)?$/);
  assert.equal(browser.version, browser.archiveVersion.replace(/esr$/, ''));
  assert.equal(
    browser.url,
    `https://archive.mozilla.org/pub/firefox/releases/${browser.archiveVersion}/linux-x86_64/en-US/firefox-${browser.archiveVersion}.tar.xz`,
  );
  return browser;
}
const hash = (bytes) => createHash('sha256').update(bytes).digest('hex');
function command(binary, args) {
  const result = spawnSync(binary, args, {
    encoding: 'utf8',
    timeout: 300000,
    maxBuffer: 4 * 1024 * 1024,
  });
  assert.ok(
    !result.error && result.status === 0,
    `${binary} failed: ${result.error || result.stderr}`,
  );
  return result.stdout.trim();
}
async function download(pin, path) {
  command('curl', [
    '--fail',
    '--location',
    '--retry',
    '3',
    '--proto',
    '=https',
    '--proto-redir',
    '=https',
    '--max-time',
    '240',
    '--output',
    path,
    pin.url,
  ]);
  const digest = hash(await readFile(path));
  assert.equal(digest, pin.sha256, 'Archive does not match official pinned checksum');
  return { url: pin.url, sha256: digest, checksumSource: pin.checksumSource || pin.source };
}
export function assertLinuxIdentity(browser, text) {
  assert.equal(
    text.trim(),
    `Mozilla Firefox ${browser.archiveVersion}`,
    'Firefox exact vendor version changed',
  );
}
export async function runLinuxInstaller(id, finalize = false, env = process.env) {
  requireLinuxRunner(env, process.platform, process.arch);
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  const browser = selectLinuxBrowser(manifest, id);
  const directory = join(await realpath(env.RUNNER_TEMP), 'kfp-qualified-browsers', id);
  const binary = join(directory, 'firefox/firefox');
  const driver = join(directory, 'geckodriver');
  const provenancePath = join(directory, 'provenance.json');
  if (finalize) {
    let provenance;
    try {
      provenance = JSON.parse(await readFile(provenancePath, 'utf8'));
    } catch (error) {
      if (error.code === 'ENOENT') return;
      throw error;
    }
    if (!provenance.binarySha256) return;
    try {
      const version = command(binary, ['--version']);
      assertLinuxIdentity(browser, version);
      assert.equal(hash(await readFile(binary)), provenance.binarySha256);
      assert.equal(hash(await readFile(driver)), provenance.driver.binarySha256);
      provenance.postSuiteIdentity = {
        status: 'passed',
        actualVersion: version,
        binarySha256: provenance.binarySha256,
        checkedAt: new Date().toISOString(),
      };
    } catch (error) {
      provenance.status = 'failed';
      provenance.postSuiteIdentity = { status: 'failed', error: String(error) };
      throw error;
    } finally {
      await writeFile(provenancePath, `${JSON.stringify(provenance, null, 2)}\n`);
    }
    return;
  }
  await mkdir(join(directory, '..'), { recursive: true });
  await mkdir(directory);
  const provenance = {
    sourceSha: env.GITHUB_SHA,
    resolvedAt: browser.resolvedAt,
    browser,
    platform: 'linux',
    architecture: 'x64',
    runnerImage: env.ImageOS,
    runnerImageVersion: env.ImageVersion,
    status: 'installing',
  };
  try {
    const archive = join(directory, 'firefox.tar.xz');
    provenance.download = await download(browser, archive);
    command('tar', ['-xJf', archive, '-C', directory]);
    const version = command(binary, ['--version']);
    assertLinuxIdentity(browser, version);
    provenance.actualVersion = browser.version;
    provenance.vendorVersion = version;
    provenance.binarySha256 = hash(await readFile(binary));
    await mkdir(join(directory, 'firefox/distribution'), { recursive: true });
    await writeFile(
      join(directory, 'firefox/distribution/policies.json'),
      JSON.stringify({ policies: { DisableAppUpdate: true } }),
    );
    provenance.updatePolicy = { DisableAppUpdate: true };
    const driverArchive = join(directory, 'geckodriver.tar.gz');
    provenance.driver = {
      ...manifest.linuxGeckodriver,
      download: await download(manifest.linuxGeckodriver, driverArchive),
    };
    command('tar', ['-xzf', driverArchive, '-C', directory, 'geckodriver']);
    provenance.driver.actualVersion = command(driver, ['--version']).split('\n')[0];
    assert.ok(
      provenance.driver.actualVersion.startsWith(
        `geckodriver ${manifest.linuxGeckodriver.version} `,
      ),
    );
    provenance.driver.binarySha256 = hash(await readFile(driver));
    for (const [key, value] of Object.entries({ binary, driver, version: browser.version })) {
      assert.ok(!/[\r\n]/.test(value));
      await appendFile(env.GITHUB_OUTPUT, `${key}=${value}\n`);
    }
    provenance.status = 'passed';
  } catch (error) {
    provenance.status = 'failed';
    provenance.error = String(error);
    throw error;
  } finally {
    await writeFile(provenancePath, `${JSON.stringify(provenance, null, 2)}\n`);
  }
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await runLinuxInstaller(
    process.argv[process.argv[2] === '--finalize' ? 3 : 2],
    process.argv[2] === '--finalize',
  );
}
