/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  appleQualification,
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
