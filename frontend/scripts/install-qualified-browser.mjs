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

import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { createReadStream } from 'node:fs';
import { appendFile, mkdir, readFile, readdir, realpath, writeFile } from 'node:fs/promises';
import { isAbsolute, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const manifestPath = fileURLToPath(new URL('./qualified-browsers.json', import.meta.url));

export function requireHostedRunner(env, platform, architecture) {
  if (
    env.CI !== 'true' ||
    env.GITHUB_ACTIONS !== 'true' ||
    env.RUNNER_ENVIRONMENT !== 'github-hosted'
  ) {
    throw new Error('Browser installation is restricted to GitHub-hosted CI; do not run locally.');
  }
  if (
    !env.RUNNER_TEMP ||
    !isAbsolute(env.RUNNER_TEMP) ||
    !env.GITHUB_OUTPUT ||
    !isAbsolute(env.GITHUB_OUTPUT)
  ) {
    throw new Error('Hosted CI must provide absolute RUNNER_TEMP and GITHUB_OUTPUT paths.');
  }
  if (platform !== 'darwin' || architecture !== 'arm64') {
    throw new Error('This browser manifest requires a macOS ARM64 hosted runner.');
  }
}

export function selectBrowser(manifest, id) {
  const browser = manifest.browsers.find((row) => row.id === id);
  if (!browser) throw new Error(`Unknown or unresolved browser slot: ${id}`);
  if (!/^[a-z0-9-]+$/.test(browser.id) || !/^\d+(\.\d+)+$/.test(browser.version)) {
    throw new Error('Browser manifest contains an invalid identity or version.');
  }
  if (browser.verification === 'cft-archive') {
    const object = browser.archiveObject;
    const name = `${browser.version}/mac-arm64/chrome-mac-arm64.zip`;
    if (
      browser.qualification !== 'supplementary' ||
      browser.kind !== 'zip' ||
      browser.appName !== 'Google Chrome for Testing.app' ||
      browser.teamId !== null ||
      browser.url !== `https://storage.googleapis.com/chrome-for-testing-public/${name}` ||
      !/^[a-f0-9]{64}$/.test(browser.sha256 || '') ||
      object?.bucket !== 'chrome-for-testing-public' ||
      object.name !== name ||
      !/^\d+$/.test(object.generation) ||
      !/^\d+$/.test(object.size) ||
      !/^[A-Za-z0-9+/]{22}==$/.test(object.md5Hash) ||
      object.metadataSource !==
        `https://storage.googleapis.com/storage/v1/b/${object.bucket}/o/${encodeURIComponent(name)}?generation=${object.generation}`
    ) {
      throw new Error(
        'Chrome for Testing archive verification requires exact supplementary Google artifact pins.',
      );
    }
  }
  return browser;
}

function command(executable, args, timeout = 120_000) {
  const result = spawnSync(executable, args, {
    encoding: 'utf8',
    timeout,
    maxBuffer: 4 * 1024 * 1024,
  });
  if (result.error || result.status !== 0) {
    throw new Error(
      `${executable} failed: ${result.error?.message || result.stderr || result.stdout}`,
    );
  }
  return `${result.stdout || ''}${result.stderr || ''}`;
}

async function archiveDigests(path) {
  const sha256 = createHash('sha256');
  const md5 = createHash('md5');
  let size = 0;
  for await (const chunk of createReadStream(path)) {
    sha256.update(chunk);
    md5.update(chunk);
    size += chunk.length;
  }
  return { sha256: sha256.digest('hex'), md5Hash: md5.digest('base64'), size: String(size) };
}

export function assertArchiveObject(expected, actual) {
  for (const key of ['bucket', 'name', 'generation', 'size', 'md5Hash']) {
    if (expected[key] !== actual[key])
      throw new Error(`Google archive object ${key} differs from its reviewed pin.`);
  }
}

export function assertChecksum(expected, actual) {
  if (expected && expected !== actual)
    throw new Error('Downloaded artifact SHA-256 does not match the vendor pin.');
}

async function download(url, destination, expectedHash, archiveObject) {
  if (new URL(url).protocol !== 'https:') throw new Error('Browser downloads require HTTPS.');
  command(
    'curl',
    [
      '--fail',
      '--location',
      '--retry',
      '3',
      '--proto',
      '=https',
      '--proto-redir',
      '=https',
      '--max-time',
      '300',
      '--output',
      destination,
      url,
    ],
    360_000,
  );
  const digests = await archiveDigests(destination);
  assertChecksum(expectedHash, digests.sha256);
  if (archiveObject) {
    assertArchiveObject(archiveObject, { ...archiveObject, ...digests });
  }
  return {
    url,
    ...digests,
    vendorChecksumVerified: Boolean(expectedHash),
    vendorChecksumAlgorithm: archiveObject ? 'MD5' : expectedHash ? 'SHA-256' : null,
    sha256Source: archiveObject
      ? 'reviewed-hosted-downloads'
      : expectedHash
        ? 'vendor'
        : 'observed',
  };
}

async function findApps(directory, name) {
  const found = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const child = join(directory, entry.name);
    if (entry.name === name) found.push(child);
    else found.push(...(await findApps(child, name)));
  }
  return found;
}

async function extractApp(browser, archive, directory) {
  if (browser.kind === 'dmg') {
    const mount = join(directory, 'mount');
    const app = join(directory, browser.appName);
    await mkdir(mount);
    command('hdiutil', ['attach', '-readonly', '-nobrowse', '-mountpoint', mount, archive]);
    try {
      command('ditto', [join(mount, browser.appName), app]);
    } finally {
      command('hdiutil', ['detach', mount]);
    }
    return app;
  }
  const expanded = join(directory, 'expanded');
  if (browser.kind === 'pkg') {
    command('pkgutil', ['--check-signature', archive]);
    command('pkgutil', ['--expand-full', archive, expanded]);
  } else if (browser.kind === 'zip') {
    await mkdir(expanded);
    // Match the archive extraction used by Google's recommended Puppeteer installer.
    command('unzip', ['-q', archive, '-d', expanded]);
  } else throw new Error(`Unsupported archive format: ${browser.kind}`);
  const apps = await findApps(expanded, browser.appName);
  if (apps.length !== 1) throw new Error(`Expected one ${browser.appName}, found ${apps.length}.`);
  return apps[0];
}

export function assertAppIdentity(browser, actualVersion, signature) {
  if (actualVersion !== browser.version) {
    throw new Error(
      `Browser version changed: expected ${browser.version}, found ${actualVersion}. Review the dated manifest.`,
    );
  }
  if (browser.verification === 'cft-archive') {
    for (const expected of ['Signature=adhoc', 'TeamIdentifier=not set', 'Sealed Resources=none']) {
      if (!signature.split('\n').includes(expected))
        throw new Error('Chrome for Testing packaging changed; review its archive identity.');
    }
    return;
  }
  if (!signature.split('\n').includes(`TeamIdentifier=${browser.teamId}`)) {
    throw new Error('Browser code signature team does not match the expected vendor.');
  }
}

export function browserUpdatePolicy(browser) {
  if (browser.appName !== 'Microsoft Edge.app') return null;
  return {
    path: '/Library/Managed Preferences/com.microsoft.EdgeUpdater.plist',
    source: 'https://learn.microsoft.com/en-us/deployedge/edge-learnmore-edgeupdater-for-macos',
    updateDefault: 3,
    plist: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>updatePolicies</key><dict><key>global</key><dict>
<key>UpdateDefault</key><integer>3</integer></dict></dict></dict></plist>
`,
  };
}

export function requiresReadOnlyBrowserImage(browser) {
  return ['Microsoft Edge.app', 'Google Chrome.app'].includes(browser.appName);
}

export function ownedBrowserMount(directory, browser, record) {
  if (!record?.mounted) return null;
  const mount = join(directory, 'read-only-volume');
  const image = join(directory, 'browser-read-only.dmg');
  if (!requiresReadOnlyBrowserImage(browser) || record.path !== mount || record.image !== image) {
    throw new Error(
      'Refusing to inspect or detach a mount not owned by this browser installation.',
    );
  }
  return { mount, image, app: join(mount, browser.appName) };
}

export async function finalizeBrowser(id, env = process.env) {
  requireHostedRunner(env, process.platform, process.arch);
  const browser = selectBrowser(JSON.parse(await readFile(manifestPath, 'utf8')), id);
  const directory = join(await realpath(env.RUNNER_TEMP), 'kfp-qualified-browsers', browser.id);
  const provenancePath = join(directory, 'provenance.json');
  let provenance;
  try {
    provenance = JSON.parse(await readFile(provenancePath, 'utf8'));
  } catch (error) {
    if (error.code === 'ENOENT') return;
    throw error;
  }
  const owned = ownedBrowserMount(directory, browser, provenance.readOnlyMount);
  if (!owned) return;
  let failure;
  try {
    if (!provenance.readOnlyMount.binarySha256) {
      provenance.postSuiteIdentity = {
        status: 'not-run',
        reason: 'Installation did not finish initial mounted-app verification.',
      };
    } else {
      const version = command('/usr/libexec/PlistBuddy', [
        '-c',
        'Print :CFBundleShortVersionString',
        join(owned.app, 'Contents', 'Info.plist'),
      ]).trim();
      const signature = command('codesign', ['--display', '--verbose=4', owned.app]);
      command('codesign', ['--verify', '--deep', '--strict', owned.app]);
      assertAppIdentity(browser, version, signature);
      const binary = join(owned.app, 'Contents', 'MacOS', browser.executable);
      const digest = (await archiveDigests(binary)).sha256;
      assertChecksum(provenance.readOnlyMount.binarySha256, digest);
      provenance.postSuiteIdentity = {
        status: 'passed',
        version,
        binarySha256: digest,
        signature,
        checkedAt: new Date().toISOString(),
      };
    }
  } catch (error) {
    failure = error;
    provenance.status = 'failed';
    provenance.postSuiteIdentity = {
      status: 'failed',
      error: error.message,
      checkedAt: new Date().toISOString(),
    };
  } finally {
    try {
      try {
        command('hdiutil', ['detach', owned.mount], 30_000);
      } catch {
        // A vendor updater may retain an open file; force-detach only our exact owned volume.
        command('hdiutil', ['detach', '-force', owned.mount], 30_000);
        provenance.readOnlyMount.forcedDetach = true;
      }
      provenance.readOnlyMount.mounted = false;
      provenance.readOnlyMount.detachedAt = new Date().toISOString();
    } catch (error) {
      failure ??= error;
      provenance.status = 'failed';
      provenance.readOnlyMount.cleanupError = error.message;
    }
    await writeFile(provenancePath, `${JSON.stringify(provenance, null, 2)}\n`);
  }
  if (failure) throw failure;
}

export async function installBrowser(id, env = process.env) {
  requireHostedRunner(env, process.platform, process.arch);
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  const browser = selectBrowser(manifest, id);
  const runnerTemp = await realpath(env.RUNNER_TEMP);
  // mkdir without recursive on the leaf fails closed if a prior installation exists.
  const parent = join(runnerTemp, 'kfp-qualified-browsers');
  await mkdir(parent, { recursive: true });
  const directory = join(parent, browser.id);
  await mkdir(directory);
  const provenancePath = join(directory, 'provenance.json');
  const provenance = {
    resolvedAt: manifest.resolvedAt,
    sourceSha: env.GITHUB_SHA,
    runnerImage: env.ImageOS,
    runnerImageVersion: env.ImageVersion,
    platform: process.platform,
    architecture: process.arch,
    browser,
    status: 'installing',
    startedAt: new Date().toISOString(),
  };
  try {
    const updatePolicy = browserUpdatePolicy(browser);
    if (updatePolicy) {
      // Edge registers its updater even from an extracted app. Freeze only this disposable
      // CI runner before any launch; otherwise later suites can silently receive a new major.
      const policyFile = join(directory, 'edge-updater-policy.plist');
      await writeFile(policyFile, updatePolicy.plist);
      command('/usr/bin/plutil', ['-lint', policyFile]);
      command('/usr/bin/sudo', ['-n', '/bin/mkdir', '-p', '/Library/Managed Preferences']);
      command('/usr/bin/sudo', [
        '-n',
        '/usr/bin/install',
        '-o',
        'root',
        '-g',
        'wheel',
        '-m',
        '644',
        policyFile,
        updatePolicy.path,
      ]);
      const configured = command('/usr/bin/plutil', [
        '-extract',
        'updatePolicies.global.UpdateDefault',
        'raw',
        '-o',
        '-',
        updatePolicy.path,
      ]).trim();
      if (configured !== String(updatePolicy.updateDefault)) {
        throw new Error('Edge updater policy did not retain the disabled-update setting.');
      }
      provenance.updatePolicy = {
        source: updatePolicy.source,
        path: updatePolicy.path,
        updateDefault: updatePolicy.updateDefault,
        verified: true,
      };
    }
    const archive = join(directory, `browser.${browser.kind}`);
    let downloadUrl = browser.url;
    if (browser.verification === 'cft-archive') {
      const response = await fetch(browser.archiveObject.metadataSource, {
        signal: AbortSignal.timeout(30_000),
      });
      if (!response.ok)
        throw new Error(`Google archive metadata request failed: ${response.status}`);
      const metadata = await response.json();
      assertArchiveObject(browser.archiveObject, metadata);
      provenance.archiveObject = metadata;
      downloadUrl += `?generation=${browser.archiveObject.generation}`;
    }
    provenance.download = await download(
      downloadUrl,
      archive,
      browser.sha256,
      browser.archiveObject,
    );
    if (browser.kind === 'zip') {
      provenance.archiveSignatureEntries = command('unzip', ['-Z', '-1', archive])
        .split('\n')
        .filter((entry) => /(?:^|\/)(_CodeSignature|CodeResources)(?:\/|$)/.test(entry));
    }
    const app = await extractApp(browser, archive, directory);
    const signature = command('codesign', ['--display', '--verbose=4', app]);
    provenance.signature = signature;
    const version = command('/usr/libexec/PlistBuddy', [
      '-c',
      'Print :CFBundleShortVersionString',
      join(app, 'Contents', 'Info.plist'),
    ]).trim();
    provenance.actualVersion = version;
    // Keep signature diagnostics even when resource-seal verification fails.
    if (browser.verification !== 'cft-archive') {
      command('codesign', ['--verify', '--deep', '--strict', app]);
    }
    assertAppIdentity(browser, version, signature);
    provenance.signatureVerification =
      browser.verification === 'cft-archive'
        ? 'Ad-hoc linker signature observed; archive verified using reviewed SHA-256 and pinned Google Storage generation, size and MD5. No vendor resource seal exists.'
        : 'Strict resource seal and vendor signing team verified';
    let binary = join(app, 'Contents', 'MacOS', browser.executable);
    if (requiresReadOnlyBrowserImage(browser)) {
      // Vendor updaters can replace even a runner-local app during the suite. A read-only
      // image freezes Chrome and Edge regardless of updater state without modifying their seals.
      const source = join(directory, 'image-source');
      await mkdir(source);
      command('ditto', [app, join(source, browser.appName)]);
      const image = join(directory, 'browser-read-only.dmg');
      command(
        'hdiutil',
        [
          'create',
          '-quiet',
          '-srcfolder',
          source,
          '-format',
          'UDRO',
          '-fs',
          'HFS+',
          '-volname',
          `KFP-${browser.id}`,
          image,
        ],
        300_000,
      );
      const mount = join(directory, 'read-only-volume');
      await mkdir(mount);
      command('hdiutil', ['attach', '-readonly', '-nobrowse', '-mountpoint', mount, image]);
      provenance.readOnlyMount = {
        path: mount,
        image,
        mounted: true,
        format: 'UDRO',
        readOnly: true,
      };
      const mountedApp = join(mount, browser.appName);
      command('codesign', ['--verify', '--deep', '--strict', mountedApp]);
      const mountedVersion = command('/usr/libexec/PlistBuddy', [
        '-c',
        'Print :CFBundleShortVersionString',
        join(mountedApp, 'Contents', 'Info.plist'),
      ]).trim();
      assertAppIdentity(
        browser,
        mountedVersion,
        command('codesign', ['--display', '--verbose=4', mountedApp]),
      );
      binary = join(mountedApp, 'Contents', 'MacOS', browser.executable);
      provenance.readOnlyMount.binarySha256 = (await archiveDigests(binary)).sha256;
    }
    let driver = '';
    if (browser.browser === 'firefox') {
      const driverArchive = join(directory, 'geckodriver.tar.gz');
      provenance.driver = {
        ...manifest.geckodriver,
        download: await download(
          manifest.geckodriver.url,
          driverArchive,
          manifest.geckodriver.sha256,
        ),
      };
      command('tar', ['-xzf', driverArchive, '-C', directory, 'geckodriver']);
      driver = join(directory, 'geckodriver');
      const driverVersion = command(driver, ['--version']).split('\n')[0];
      if (!driverVersion.startsWith(`geckodriver ${manifest.geckodriver.version} `))
        throw new Error('Geckodriver version differs from the manifest.');
      provenance.driver.actualVersion = driverVersion;
    }
    const outputs = {
      browser: browser.browser,
      version,
      binary,
      driver,
      provenance: provenancePath,
    };
    for (const value of Object.values(outputs))
      if (/[\r\n]/.test(value)) throw new Error('Invalid multiline GitHub output.');
    await appendFile(
      env.GITHUB_OUTPUT,
      Object.entries(outputs)
        .map(([key, value]) => `${key}=${value}\n`)
        .join(''),
    );
    provenance.status = 'passed';
    console.log(JSON.stringify({ id, ...outputs }));
  } catch (error) {
    provenance.status = 'failed';
    provenance.error = error.message;
    throw error;
  } finally {
    provenance.finishedAt = new Date().toISOString();
    await writeFile(provenancePath, `${JSON.stringify(provenance, null, 2)}\n`);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (process.argv[2] === '--finalize') await finalizeBrowser(process.argv[3]);
  else await installBrowser(process.argv[2]);
}
