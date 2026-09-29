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
import {
  appleQualification,
  mobileSafariAppPath,
  observeAppleCommand,
  stopOwnedAppleChild,
  requireHostedAppleRunner,
  selectSimulator,
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
