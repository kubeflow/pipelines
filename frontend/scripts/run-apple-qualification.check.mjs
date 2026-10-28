/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

import assert from 'node:assert/strict';
import { test } from 'node:test';
import { EventEmitter } from 'node:events';
import { PassThrough } from 'node:stream';
import { mkdir, mkdtemp, realpath, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { createHash } from 'node:crypto';
import {
  appleQualification,
  selectAppleQualification,
  verifyDesktopSafariIdentity,
  configureDesktopTextInput,
  mobileSafariAppPath,
  resolveWdaPackage,
  inspectWdaBuild,
  observeAppleCommand,
  stopOwnedAppleChild,
  requireHostedAppleRunner,
  selectSimulator,
  readSimulatorInventory,
} from './run-apple-qualification.mjs';

const hosted = {
  CI: 'true',
  GITHUB_ACTIONS: 'true',
  RUNNER_ENVIRONMENT: 'github-hosted',
  RUNNER_TEMP: '/tmp/disposable-runner',
};

test('Apple tooling rejects local and self-hosted execution before setup', () => {
  assert.throws(() => requireHostedAppleRunner({}, 'darwin'), /only in GitHub Actions/);
  assert.throws(
    () => requireHostedAppleRunner({ ...hosted, RUNNER_ENVIRONMENT: 'self-hosted' }, 'darwin'),
    /disposable GitHub-hosted/,
  );
  assert.throws(() => requireHostedAppleRunner(hosted, 'linux'), /requires macOS/);
  for (const RUNNER_TEMP of [undefined, '', 'relative-runner']) {
    assert.throws(
      () => requireHostedAppleRunner({ ...hosted, RUNNER_TEMP }, 'darwin'),
      /RUNNER_TEMP must be an absolute path/,
    );
  }
  assert.doesNotThrow(() => requireHostedAppleRunner(hosted, 'darwin'));
});

test('simulator selection requires exact available iOS runtime and device family', () => {
  const inventory = {
    runtimes: [
      {
        identifier: 'com.apple.CoreSimulator.SimRuntime.tvOS-26-5',
        version: '26.5',
        buildversion: 'test-build-26.5',
        isAvailable: true,
      },
      {
        identifier: 'com.apple.CoreSimulator.SimRuntime.iOS-26-4',
        version: '26.4',
        isAvailable: true,
      },
      {
        identifier: 'com.apple.CoreSimulator.SimRuntime.iOS-26-5',
        version: '26.5',
        buildversion: 'test-build-26.5',
        isAvailable: true,
      },
    ],
    devicetypes: [
      { name: 'iPhone 17', identifier: 'phone' },
      { name: 'iPad (A16)', identifier: 'tablet' },
    ],
  };
  const phone = selectSimulator(inventory, appleQualification.iphone);
  assert.equal(phone.runtime.identifier, 'com.apple.CoreSimulator.SimRuntime.iOS-26-5');
  assert.equal(phone.deviceType.identifier, 'phone');
  assert.equal(phone.runtime.buildversion, 'test-build-26.5');
  assert.equal(selectSimulator(inventory, appleQualification.ipad).deviceType.identifier, 'tablet');
  assert.throws(
    () => selectSimulator(inventory, { ...appleQualification.iphone, platformVersion: '27.0' }),
    /Required iOS 27.0 runtime is unavailable/,
  );
  const buildversion = inventory.runtimes[2].buildversion;
  delete inventory.runtimes[2].buildversion;
  assert.throws(
    () => selectSimulator(inventory, appleQualification.iphone),
    /Simulator runtime build identity is unavailable/,
  );
  inventory.runtimes[2].buildversion = buildversion;
  inventory.runtimes[2].isAvailable = false;
  assert.throws(
    () => selectSimulator(inventory, appleQualification.iphone),
    /runtime is unavailable/,
  );
  inventory.runtimes[2].isAvailable = true;
  assert.throws(
    () => selectSimulator({ ...inventory, devicetypes: [] }, appleQualification.ipad),
    /device type iPad/,
  );
});

test('annual Apple pins never fall back to an available runtime from another annual family', () => {
  const profile = selectAppleQualification('iphone', '27');
  assert.equal(profile.config.platformVersion, '27.0');
  assert.equal(profile.config.browserVersion, '27.0');
  assert.equal(profile.developerDirectory, '/Applications/Xcode_27.app/Contents/Developer');
  const inventory = {
    runtimes: [
      {
        identifier: 'com.apple.CoreSimulator.SimRuntime.iOS-26-5',
        version: '26.5',
        buildversion: 'older-build',
        isAvailable: true,
      },
    ],
    devicetypes: [{ name: 'iPhone 17', identifier: 'phone' }],
  };
  assert.throws(() => selectSimulator(inventory, profile.config), /Required iOS 27.0/);
  inventory.runtimes.push({
    identifier: 'com.apple.CoreSimulator.SimRuntime.iOS-27-0',
    version: '27.0',
    buildversion: 'current-build',
    isAvailable: true,
  });
  assert.equal(selectSimulator(inventory, profile.config).runtime.buildversion, 'current-build');
  assert.equal(selectAppleQualification('desktop', '27').config.browserVersion, '27.0.1');
  assert.deepEqual(selectAppleQualification('ipad').config, appleQualification.ipad);
  assert.throws(() => selectAppleQualification('iphone', '28'), /annual version/);
  assert.throws(() => selectAppleQualification('watch', '27'), /desktop, iphone, or ipad/);
});

test('Mobile Safari identity comes from the selected runtime without a booted app lookup', () => {
  const runtimeRoot =
    '/Library/Developer/CoreSimulator/Volumes/iOS_23F77/Library/Developer/CoreSimulator/Profiles/Runtimes/iOS 26.5.simruntime/Contents/Resources/RuntimeRoot';
  assert.equal(
    mobileSafariAppPath({ runtimeRoot }),
    `${runtimeRoot}/Applications/MobileSafari.app`,
  );
  for (const invalid of [undefined, '', 'relative/runtime']) {
    assert.throws(
      () => mobileSafariAppPath({ runtimeRoot: invalid }),
      /Selected simulator runtime root is unavailable/,
    );
  }
});

function fakeChild() {
  return Object.assign(new EventEmitter(), {
    pid: 123456,
    exitCode: null,
    signalCode: null,
    stdout: new PassThrough(),
    stderr: new PassThrough(),
    unref() {},
  });
}

test('an exited Apple command completes when an inherited pipe never closes', async () => {
  const child = fakeChild();
  const entry = {};
  const observed = observeAppleCommand(child, new PassThrough(), entry, { drainTimeout: 5 });
  child.stdout.write('Finished boot status\n');
  child.exitCode = 0;
  child.emit('exit', 0, null);
  const result = await observed.completion;
  assert.equal(result.code, 0);
  assert.equal(result.stdout, 'Finished boot status\n');
  assert.equal(entry.stdioDrainTimedOut, true);
  assert.equal(child.stdout.destroyed, true);
  assert.equal(child.stderr.destroyed, true);
  stopOwnedAppleChild(child, 'SIGTERM', () => assert.fail('must not signal an exited command'));
});

test('denied process-group signals remain diagnostic and timeout completion stays bounded', async () => {
  const child = fakeChild();
  const entry = { timedOut: true };
  const observed = observeAppleCommand(child, new PassThrough(), entry);
  const signalError = stopOwnedAppleChild(child, 'SIGKILL', () => {
    throw Object.assign(new Error('kill EPERM'), { code: 'EPERM' });
  });
  assert.match(signalError, /SIGKILL for owned process 123456: kill EPERM/);
  observed.abandon();
  const result = await observed.completion;
  assert.equal(result.code, null);
  assert.equal(entry.abandoned, true);
  assert.equal(entry.timedOut, true);
});

test('log failure keeps a live child owned until cleanup explicitly abandons it', async () => {
  const child = fakeChild();
  const owned = new Set([child]);
  const log = new PassThrough();
  const observed = observeAppleCommand(child, log, {}, { onExit: () => owned.delete(child) });
  log.destroy(new Error('test log failure'));
  const result = await observed.completion;
  assert.match(result.error, /Could not write command log: test log failure/);
  assert.equal(owned.has(child), true);
  let signalled = false;
  stopOwnedAppleChild(child, 'SIGTERM', () => {
    signalled = true;
  });
  assert.equal(signalled, true);
  observed.abandon();
  assert.equal(owned.has(child), false);
});

test('native text-input configuration rejects local mutation and checks every preference readback', async () => {
  const calls = [];
  const run = async (command, args) => {
    calls.push({ command, args });
    return args[0] === 'read' ? '0' : '';
  };
  await assert.rejects(configureDesktopTextInput(run, {}, 'darwin'), /only in GitHub Actions/);
  assert.equal(calls.length, 0, 'local execution must not write preferences');
  const configuration = await configureDesktopTextInput(run, hosted, 'darwin');
  assert.equal(configuration.domain, 'NSGlobalDomain');
  assert.equal(calls.length, 4);
  for (let index = 0; index < calls.length; index += 2) {
    const write = calls[index];
    const read = calls[index + 1];
    assert.equal(write.command, '/usr/bin/defaults');
    assert.deepEqual(write.args.slice(0, 2), ['write', '-g']);
    assert.deepEqual(write.args.slice(3), ['-bool', 'false']);
    assert.deepEqual(read.args, ['read', '-g', write.args[2]]);
    assert.deepEqual(configuration.preferences[write.args[2]], {
      configured: false,
      readback: '0',
    });
  }
  await assert.rejects(
    configureDesktopTextInput(async () => '1', hosted, 'darwin'),
    /preference .* was not disabled/,
  );
});

test('WDA resolution follows the installed driver dependency rather than an unrelated package', async () => {
  const root = await realpath(await mkdtemp(join(tmpdir(), 'kfp-wda-package-')));
  try {
    const driver = join(root, 'node_modules', 'appium-xcuitest-driver');
    const nested = join(driver, 'node_modules', 'appium-webdriveragent');
    const hoisted = join(root, 'node_modules', 'appium-webdriveragent');
    for (const [folder, version] of [
      [nested, '16.12.11'],
      [hoisted, '99.0.0'],
    ]) {
      await mkdir(join(folder, 'WebDriverAgent.xcodeproj'), { recursive: true });
      await writeFile(
        join(folder, 'package.json'),
        JSON.stringify({
          name: 'appium-webdriveragent',
          version,
          exports: { './package.json': './package.json' },
        }),
      );
    }
    await writeFile(join(driver, 'package.json'), '{"name":"appium-xcuitest-driver"}');
    const resolved = await resolveWdaPackage(join(driver, 'package.json'));
    assert.equal(resolved.root, nested);
    assert.equal(resolved.version, '16.12.11');
    assert.equal(resolved.projectPath, join(nested, 'WebDriverAgent.xcodeproj'));
    assert.match(resolved.packageSha256, /^[a-f0-9]{64}$/);
    const otherDriver = join(root, 'node_modules', 'other-driver');
    await mkdir(otherDriver);
    await writeFile(join(otherDriver, 'package.json'), '{}');
    await assert.rejects(
      resolveWdaPackage(join(otherDriver, 'package.json')),
      /Unreviewed WebDriverAgent version/,
    );
    await writeFile(
      join(nested, 'package.json'),
      JSON.stringify({ name: 'appium-webdriveragent', version: '16.12.12' }),
    );
    await assert.rejects(
      resolveWdaPackage(join(driver, 'package.json')),
      /Unreviewed WebDriverAgent version/,
    );
    await writeFile(
      join(nested, 'package.json'),
      JSON.stringify({ name: 'appium-webdriveragent', version: '16.12.11' }),
    );
    await rm(join(nested, 'WebDriverAgent.xcodeproj'), { recursive: true });
    await assert.rejects(resolveWdaPackage(join(driver, 'package.json')), /ENOENT/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('WDA output inspection rejects incomplete builds and records completed build identities', async () => {
  const root = await mkdtemp(join(tmpdir(), 'kfp-wda-products-'));
  try {
    const products = join(root, 'Build', 'Products');
    await mkdir(products, { recursive: true });
    await assert.rejects(inspectWdaBuild(root), /no .xctestrun file/);
    const testRun = join(products, 'WebDriverAgentRunner_iphonesimulator26.5-x86_64.xctestrun');
    await writeFile(testRun, 'fixture test manifest');
    await assert.rejects(inspectWdaBuild(root), /ENOENT/);
    const runnerApp = join(products, 'Debug-iphonesimulator', 'WebDriverAgentRunner-Runner.app');
    await mkdir(runnerApp, { recursive: true });
    await writeFile(join(runnerApp, 'Info.plist'), 'fixture bundle identity');
    await assert.rejects(inspectWdaBuild(root), /ENOENT/);
    await writeFile(join(runnerApp, 'WebDriverAgentRunner-Runner'), 'fixture executable');
    await assert.rejects(inspectWdaBuild(root), /ENOENT/);
    const plugin = join(runnerApp, 'PlugIns', 'WebDriverAgentRunner.xctest');
    await mkdir(plugin, { recursive: true });
    await writeFile(join(plugin, 'WebDriverAgentRunner'), 'fixture WDA test executable');
    const result = await inspectWdaBuild(root);
    assert.deepEqual(result.testRuns, [testRun]);
    assert.equal(result.runnerApp, runnerApp);
    assert.equal(result.files.length, 4);
    assert.equal(
      result.files[0].sha256,
      createHash('sha256').update('fixture test manifest').digest('hex'),
    );
    await writeFile(join(runnerApp, 'WebDriverAgentRunner-Runner'), '');
    await assert.rejects(inspectWdaBuild(root), /build output is empty/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('cold simulator inventory gets one bounded startup call without changing runtime identity', async () => {
  const expected = {
    runtimes: [
      {
        identifier: 'com.apple.CoreSimulator.SimRuntime.iOS-26-5',
        version: '26.5',
        buildversion: '23F77',
        isAvailable: true,
      },
    ],
    devicetypes: [
      { name: 'iPhone 17', identifier: 'com.apple.CoreSimulator.SimDeviceType.iPhone-17' },
    ],
  };
  const calls = [];
  const inventory = await readSimulatorInventory(async (...args) => {
    calls.push(args);
    return JSON.stringify(expected);
  });
  assert.deepEqual(calls, [['/usr/bin/xcrun', ['simctl', 'list', '-j'], 'simulators', 300000]]);
  assert.deepEqual(inventory, expected);
  const selected = selectSimulator(inventory, appleQualification.iphone);
  assert.deepEqual(selected.runtime, expected.runtimes[0]);
  assert.deepEqual(selected.deviceType, expected.devicetypes[0]);
});

test('cold inventory allowance never retries command failures or accepts malformed inventory', async () => {
  const failure = new Error('simctl service startup failed');
  let calls = 0;
  await assert.rejects(
    readSimulatorInventory(async () => {
      calls++;
      throw failure;
    }),
    (error) => error === failure,
  );
  assert.equal(calls, 1);
  for (const output of ['', '{bad', 'null', '{}', '{"runtimes":[],"devicetypes":{}}']) {
    let attempts = 0;
    await assert.rejects(
      readSimulatorInventory(async () => {
        attempts++;
        return output;
      }),
    );
    assert.equal(attempts, 1);
  }
});

test('Safari rolling-image pins require the exact documented image and patch pair', () => {
  for (const [image, version] of [
    ['20260928.0222.1', '27.0'],
    ['20261006.0244.1', '27.0.1'],
  ]) {
    assert.equal(verifyDesktopSafariIdentity('27', version, image, '27.0.1'), version);
    assert.throws(
      () =>
        verifyDesktopSafariIdentity('27', version === '27.0' ? '27.0.1' : '27.0', image, '27.0.1'),
      /Pinned Safari version changed/,
    );
  }
  for (const image of [undefined, '20261006.0244.2', 'unknown']) {
    assert.throws(
      () => verifyDesktopSafariIdentity('27', '27.0.1', image, '27.0.1'),
      /Unqualified Safari runner image/,
    );
  }
  assert.throws(
    () => verifyDesktopSafariIdentity('26', '26.6.3', undefined, '26.6.2'),
    /Pinned Safari version changed/,
  );
});
