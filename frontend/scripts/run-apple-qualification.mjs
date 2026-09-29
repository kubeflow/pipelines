/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// Apple automation is deliberately restricted to disposable GitHub-hosted runners.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { createWriteStream } from 'node:fs';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { dirname, isAbsolute, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..');
export const appleQualification = {
  desktop: { browserVersion: '26.6.2' },
  iphone: { browserVersion: '26.5', platformVersion: '26.5', deviceType: 'iPhone 17' },
  ipad: { browserVersion: '26.5', platformVersion: '26.5', deviceType: 'iPad (A16)' },
};
const appiumVersion = '3.8.0';
const xcuitestVersion = '12.13.3';

export function requireHostedAppleRunner(env = process.env, platform = process.platform) {
  assert.equal(platform, 'darwin', 'Apple qualification requires macOS');
  assert.equal(env.GITHUB_ACTIONS, 'true', 'Apple qualification runs only in GitHub Actions');
  assert.equal(env.CI, 'true', 'Apple qualification runs only in CI');
  assert.equal(
    env.RUNNER_ENVIRONMENT,
    'github-hosted',
    'Apple qualification requires a disposable GitHub-hosted runner',
  );
  assert.ok(env.RUNNER_TEMP && isAbsolute(env.RUNNER_TEMP), 'RUNNER_TEMP must be an absolute path');
}

export function selectSimulator(inventory, config) {
  const runtime = inventory.runtimes.find(
    (item) =>
      item.isAvailable &&
      item.version === config.platformVersion &&
      item.identifier.startsWith('com.apple.CoreSimulator.SimRuntime.iOS-'),
  );
  assert.ok(runtime, `Required iOS ${config.platformVersion} runtime is unavailable`);
  assert.ok(runtime.buildversion, 'Simulator runtime build identity is unavailable');
  const deviceType = inventory.devicetypes.find((item) => item.name === config.deviceType);
  assert.ok(deviceType, `Required simulator device type ${config.deviceType} is unavailable`);
  return { runtime, deviceType };
}

async function main() {
  // Refuse before creating files, running Apple tools, or installing automation dependencies.
  requireHostedAppleRunner();
  const mode = process.argv[2];
  assert.ok(Object.hasOwn(appleQualification, mode), 'Use desktop, iphone, or ipad');
  const config = appleQualification[mode];
  const out = join(frontend, 'browser-results', `apple-${mode}`);
  // A setup failure must never upload successful checks from an earlier invocation.
  await rm(out, { recursive: true, force: true });
  await mkdir(out, { recursive: true });
  const work = await mkdtemp(join(process.env.RUNNER_TEMP, `kfp-apple-${mode}-`));
  const env = {
    ...process.env,
    APPIUM_HOME: join(work, 'appium-home'),
    npm_config_cache: join(work, 'npm-cache'),
    TMPDIR: work,
  };
  const report = {
    startedAt: new Date().toISOString(),
    status: 'running',
    sourceSha: process.env.GITHUB_SHA,
    mode,
    requested: config,
    runner: {
      os: process.env.RUNNER_OS,
      arch: process.env.RUNNER_ARCH,
      imageOS: process.env.ImageOS,
      imageVersion: process.env.ImageVersion,
    },
    tooling: { appiumVersion, xcuitestVersion },
    commands: [],
    limitations: [
      'Pinned runner-available Apple versions do not establish coverage of newer Safari/iOS releases or the full supported version policy.',
      'Simulator checks do not establish physical-device, assistive-technology, or complete accessibility conformance.',
    ],
  };
  const children = new Set();
  let simulator;
  let commandIndex = 0;
  let interrupted = false;
  const interrupt = () => {
    interrupted = true;
    for (const child of children) stop(child);
  };
  process.on('SIGTERM', interrupt);
  process.on('SIGINT', interrupt);

  function stop(child, signal = 'SIGTERM') {
    if (!Number.isInteger(child.pid)) return;
    try {
      // Every child has its own group; never terminate unrelated runner processes.
      process.kill(-child.pid, signal);
    } catch (error) {
      if (error.code !== 'ESRCH') throw error;
    }
  }

  function start(command, args, label) {
    assert.ok(!interrupted, 'Qualification was interrupted');
    const logName = `${String(++commandIndex).padStart(2, '0')}-${label}.log`;
    const log = createWriteStream(join(out, logName));
    const entry = { command, args, log: logName, startedAt: new Date().toISOString() };
    report.commands.push(entry);
    const child = spawn(command, args, { cwd: frontend, env, detached: true });
    children.add(child);
    let stdout = '';
    child.stdout.on('data', (chunk) => {
      log.write(chunk);
      stdout = (stdout + chunk.toString()).slice(-2_000_000);
    });
    child.stderr.on('data', (chunk) => log.write(chunk));
    const completion = new Promise((resolveResult) => {
      child.once('error', (error) => {
        entry.error = error.message;
        log.write(`${error.stack}\n`);
      });
      child.once('close', (code, signal) => {
        children.delete(child);
        Object.assign(entry, { code, signal, finishedAt: new Date().toISOString() });
        log.end(() => resolveResult({ code, signal, stdout, error: entry.error }));
      });
    });
    return { child, completion, entry };
  }

  async function run(command, args, label, timeout = 60_000) {
    const launched = start(command, args, label);
    let forceTimer;
    const timer = setTimeout(() => {
      launched.entry.timedOut = true;
      stop(launched.child);
      forceTimer = setTimeout(() => stop(launched.child, 'SIGKILL'), 10_000);
    }, timeout);
    try {
      const result = await launched.completion;
      assert.equal(result.code, 0, `${label} failed; see ${launched.entry.log}`);
      assert.ok(!launched.entry.timedOut, `${label} exceeded ${timeout}ms`);
      return result.stdout.trim();
    } finally {
      clearTimeout(timer);
      clearTimeout(forceTimer);
    }
  }

  async function ready(url, launched, timeout = 60_000) {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline && !interrupted) {
      assert.ok(
        !launched.entry.error &&
          launched.child.exitCode === null &&
          launched.child.signalCode === null,
        `Service exited before readiness; see ${launched.entry.log}`,
      );
      try {
        const response = await fetch(url, { signal: AbortSignal.timeout(2_000) });
        if (response.ok) return;
      } catch {
        // Retry only during bounded service startup.
      }
      await delay(500);
    }
    throw new Error(`Timed out waiting for ${url}; see ${launched.entry.log}`);
  }

  try {
    report.macos = await run('/usr/bin/sw_vers', [], 'macos');
    report.xcode = await run('/usr/bin/xcodebuild', ['-version'], 'xcode');
    report.safariVersion = await run(
      '/usr/libexec/PlistBuddy',
      ['-c', 'Print :CFBundleShortVersionString', '/Applications/Safari.app/Contents/Info.plist'],
      'safari-version',
    );
    report.safariBuildVersion = await run(
      '/usr/libexec/PlistBuddy',
      ['-c', 'Print :CFBundleVersion', '/Applications/Safari.app/Contents/Info.plist'],
      'safari-build-version',
    );
    assert.ok(report.safariBuildVersion, 'Safari build identity is unavailable');
    const fixture = start(
      process.execPath,
      ['--import', 'tsx', 'scripts/ui-modernization-native-server.ts'],
      'fixture',
    );
    await ready('http://127.0.0.1:4174/__qualification', fixture);
    let capabilities = {};
    if (mode === 'desktop') {
      assert.equal(report.safariVersion, config.browserVersion, 'Pinned Safari version changed');
      await run('/usr/bin/sudo', ['-n', '/usr/bin/safaridriver', '--enable'], 'enable-safari');
      const driver = start('/usr/bin/safaridriver', ['--port', '4444'], 'safaridriver');
      await ready('http://127.0.0.1:4444/status', driver);
    } else {
      const inventory = JSON.parse(
        await run('/usr/bin/xcrun', ['simctl', 'list', '-j'], 'simulators'),
      );
      await writeFile(
        join(out, 'simulator-inventory.json'),
        `${JSON.stringify(inventory, null, 2)}\n`,
      );
      const selected = selectSimulator(inventory, config);
      report.simulator = selected;
      simulator = await run(
        '/usr/bin/xcrun',
        [
          'simctl',
          'create',
          `KFP-${mode}-${process.env.GITHUB_RUN_ID}`,
          selected.deviceType.identifier,
          selected.runtime.identifier,
        ],
        'create-simulator',
      );
      report.simulator.udid = simulator;
      await run('/usr/bin/xcrun', ['simctl', 'boot', simulator], 'boot-simulator');
      await run(
        '/usr/bin/xcrun',
        ['simctl', 'bootstatus', simulator, '-b'],
        'simulator-ready',
        180_000,
      );
      const mobileSafariApp = await run(
        '/usr/bin/xcrun',
        ['simctl', 'get_app_container', simulator, 'com.apple.mobilesafari', 'app'],
        'mobile-safari-app',
      );
      assert.ok(isAbsolute(mobileSafariApp), 'Mobile Safari application path is unavailable');
      report.mobileSafari = { app: mobileSafariApp };
      for (const [field, key] of [
        ['version', 'CFBundleShortVersionString'],
        ['buildVersion', 'CFBundleVersion'],
      ]) {
        report.mobileSafari[field] = await run(
          '/usr/libexec/PlistBuddy',
          ['-c', `Print :${key}`, join(mobileSafariApp, 'Info.plist')],
          `mobile-safari-${field}`,
        );
        assert.ok(report.mobileSafari[field], `Mobile Safari ${field} identity is unavailable`);
      }
      const tools = join(work, 'tools');
      await run(
        'npm',
        ['install', '--prefix', tools, '--no-audit', '--no-fund', `appium@${appiumVersion}`],
        'install-appium',
        300_000,
      );
      const appium = join(tools, 'node_modules', 'appium', 'index.js');
      await run(
        process.execPath,
        [appium, 'driver', 'install', `xcuitest@${xcuitestVersion}`],
        'install-xcuitest',
        300_000,
      );
      report.installedTools = {
        appium: JSON.parse(await readFile(join(tools, 'node_modules/appium/package.json'), 'utf8'))
          .version,
        xcuitest: JSON.parse(
          await readFile(
            join(env.APPIUM_HOME, 'node_modules/appium-xcuitest-driver/package.json'),
            'utf8',
          ),
        ).version,
      };
      assert.equal(report.installedTools.appium, appiumVersion);
      assert.equal(report.installedTools.xcuitest, xcuitestVersion);
      report.appiumDrivers = await run(
        process.execPath,
        [appium, 'driver', 'list', '--installed', '--json'],
        'appium-drivers',
      );
      const driver = start(
        process.execPath,
        [appium, '--address', '127.0.0.1', '--port', '4444', '--log-timestamp'],
        'appium',
      );
      await ready('http://127.0.0.1:4444/status', driver);
      capabilities = {
        platformName: 'iOS',
        'appium:automationName': 'XCUITest',
        'appium:udid': simulator,
        'appium:platformVersion': config.platformVersion,
        'appium:deviceName': config.deviceType,
        'appium:derivedDataPath': join(work, 'wda-derived-data'),
        'appium:simulatorStartupTimeout': 180000,
        'appium:wdaLaunchTimeout': 180000,
        'appium:newCommandTimeout': 120,
        'appium:nativeWebTap': true,
        'appium:screenshotQuality': 0,
        'appium:safariInitialUrl': 'http://127.0.0.1:4174/',
      };
    }
    Object.assign(env, {
      KFP_WEBDRIVER_URL: 'http://127.0.0.1:4444',
      KFP_WEBDRIVER_BROWSER: 'safari',
      KFP_WEBDRIVER_CAPABILITIES: JSON.stringify(capabilities),
      KFP_BROWSER_FLOOR_MOBILE: mode === 'desktop' ? '0' : '1',
      KFP_BROWSER_FLOOR_VERSION: config.browserVersion,
      KFP_EXPECTED_PLATFORM_VERSION: config.platformVersion || '',
      KFP_BROWSER_FLOOR_URL: 'http://127.0.0.1:4174/',
      KFP_BROWSER_FLOOR_OUTPUT: join(out, 'checks'),
    });
    await run(
      process.execPath,
      ['scripts/ui-modernization-browser-floor.mjs'],
      'native-checks',
      900_000,
    );
    report.status = 'passed';
  } catch (error) {
    report.status = 'failed';
    report.error = error.stack;
    process.exitCode = 1;
    console.error(error);
  } finally {
    process.off('SIGTERM', interrupt);
    process.off('SIGINT', interrupt);
    for (const child of children) stop(child);
    // Allow graceful driver/session shutdown, then kill only owned process groups.
    await delay(1_000);
    for (const child of children) stop(child, 'SIGKILL');
    if (simulator) {
      interrupted = false;
      for (const action of ['shutdown', 'delete']) {
        try {
          await run('/usr/bin/xcrun', ['simctl', action, simulator], `${action}-simulator`);
        } catch (error) {
          report.cleanupErrors ??= [];
          report.cleanupErrors.push(error.message);
          report.status = 'failed';
          process.exitCode = 1;
        }
      }
    }
    report.finishedAt = new Date().toISOString();
    await writeFile(join(out, 'environment.json'), `${JSON.stringify(report, null, 2)}\n`);
    console.log(`Apple ${mode}: ${report.status}. Evidence: ${out}`);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  await main();
}
