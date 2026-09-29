/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// Apple automation is deliberately restricted to disposable GitHub-hosted runners.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { createWriteStream } from 'node:fs';
import { mkdir, mkdtemp, readFile, readdir, rm, stat, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
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
// First-boot OS migration on a hosted simulator has a separate bounded budget.
const simulatorBootTimeout = 300_000;
const mobileIdleTimeoutSeconds = 2;
const wdaLocalPort = 8100;

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

// These are native macOS text-input settings, not mutations of tested DOM fields.
// Apple documents the settings; nix-darwin documents their NSGlobalDomain keys:
// https://support.apple.com/guide/mac-help/mchlp2299/mac
// https://github.com/nix-darwin/nix-darwin/blob/4cff07de74b50e64bdd68cd4e722ab5b6b35ee48/modules/system/defaults/NSGlobalDomain.nix
export async function configureDesktopTextInput(
  run,
  env = process.env,
  platform = process.platform,
) {
  requireHostedAppleRunner(env, platform);
  const configuration = {
    domain: 'NSGlobalDomain',
    preferences: {},
    excludedCoverage: [
      'Native inline predictive text and automatic spelling correction are disabled.',
      'IME composition and autocorrection compatibility remain unqualified.',
    ],
  };
  for (const key of [
    'NSAutomaticInlinePredictionEnabled',
    'NSAutomaticSpellingCorrectionEnabled',
  ]) {
    await run('/usr/bin/defaults', ['write', '-g', key, '-bool', 'false'], `disable-${key}`);
    const readback = await run('/usr/bin/defaults', ['read', '-g', key], `read-${key}`);
    assert.equal(readback, '0', `Native text-input preference ${key} was not disabled`);
    configuration.preferences[key] = { configured: false, readback };
  }
  return configuration;
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

export function mobileSafariAppPath(runtime) {
  assert.ok(
    runtime.runtimeRoot && isAbsolute(runtime.runtimeRoot),
    'Selected simulator runtime root is unavailable',
  );
  return join(runtime.runtimeRoot, 'Applications', 'MobileSafari.app');
}

// Resolve the driver's own WDA dependency, including npm's nested/hoisted layouts.
export async function resolveWdaPackage(xcuitestPackagePath) {
  const packagePath = createRequire(xcuitestPackagePath).resolve(
    'appium-webdriveragent/package.json',
  );
  const packageBytes = await readFile(packagePath);
  const metadata = JSON.parse(packageBytes);
  assert.equal(metadata.name, 'appium-webdriveragent');
  assert.ok(typeof metadata.version === 'string' && metadata.version, 'WDA version is unavailable');
  const root = dirname(packagePath);
  const projectPath = join(root, 'WebDriverAgent.xcodeproj');
  assert.ok((await stat(projectPath)).isDirectory(), 'WDA Xcode project is unavailable');
  return {
    version: metadata.version,
    root,
    packagePath,
    packageSha256: createHash('sha256').update(packageBytes).digest('hex'),
    projectPath,
  };
}

export async function inspectWdaBuild(derivedDataPath) {
  const products = join(derivedDataPath, 'Build', 'Products');
  const testRuns = (await readdir(products, { withFileTypes: true }))
    .filter((entry) => entry.isFile() && entry.name.endsWith('.xctestrun'))
    .map((entry) => join(products, entry.name))
    .sort();
  assert.ok(testRuns.length, 'WDA build produced no .xctestrun file');
  const runnerApp = join(products, 'Debug-iphonesimulator', 'WebDriverAgentRunner-Runner.app');
  assert.ok((await stat(runnerApp)).isDirectory(), 'WDA runner app is unavailable');
  const files = [];
  for (const path of [
    ...testRuns,
    join(runnerApp, 'Info.plist'),
    join(runnerApp, 'WebDriverAgentRunner-Runner'),
    join(runnerApp, 'PlugIns', 'WebDriverAgentRunner.xctest', 'WebDriverAgentRunner'),
  ]) {
    const bytes = await readFile(path);
    assert.ok(bytes.length, `WDA build output is empty: ${path}`);
    files.push({ path, sha256: createHash('sha256').update(bytes).digest('hex') });
  }
  return { runnerApp, testRuns, files };
}

// A child may exit while an Apple service still holds its inherited stdout/stderr.
// Observe process exit separately from stream close and bound the final log drain.
export function observeAppleCommand(
  child,
  log,
  entry,
  { drainTimeout = 1000, onExit = () => {} } = {},
) {
  let stdout = '';
  let finished = false;
  let drainTimer;
  let resolveCompletion;
  const completion = new Promise((resolveResult) => {
    resolveCompletion = resolveResult;
  });
  log.on('error', (error) => {
    entry.error = `Could not write command log: ${error.message}`;
    finish();
  });
  child.stdout?.on('data', (chunk) => {
    log.write(chunk);
    stdout = (stdout + chunk.toString()).slice(-2_000_000);
  });
  child.stderr?.on('data', (chunk) => log.write(chunk));
  function finish() {
    if (finished) return;
    finished = true;
    clearTimeout(drainTimer);
    child.stdout?.destroy();
    child.stderr?.destroy();
    entry.finishedAt = new Date().toISOString();
    const resolveResult = () =>
      resolveCompletion({
        code: entry.code ?? null,
        signal: entry.signal ?? null,
        stdout,
        error: entry.error,
      });
    if (log.destroyed) resolveResult();
    else {
      log.once('error', resolveResult);
      log.end(resolveResult);
    }
  }
  child.once('error', (error) => {
    entry.error = error.message;
    log.write(`${error.stack}\n`);
    finish();
  });
  child.once('exit', (code, signal) => {
    Object.assign(entry, { code, signal, exitedAt: new Date().toISOString() });
    onExit();
    if (finished) return;
    drainTimer = setTimeout(() => {
      entry.stdioDrainTimedOut = true;
      finish();
    }, drainTimeout);
  });
  child.once('close', (code, signal) => {
    Object.assign(entry, { code, signal });
    onExit();
    finish();
  });
  return {
    completion,
    abandon() {
      entry.abandoned = true;
      onExit();
      child.unref();
      finish();
    },
  };
}

export function stopOwnedAppleChild(child, signal = 'SIGTERM', kill = process.kill) {
  if (!Number.isInteger(child.pid) || child.exitCode !== null || child.signalCode !== null) return;
  try {
    // The child owns its group. Never target Apple services outside that group.
    kill(-child.pid, signal);
  } catch (error) {
    if (error.code !== 'ESRCH') return `${signal} for owned process ${child.pid}: ${error.message}`;
  }
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
  const controls = new WeakMap();
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
    const error = stopOwnedAppleChild(child, signal);
    if (error) {
      report.processErrors ??= [];
      report.processErrors.push(error);
      report.status = 'failed';
      process.exitCode = 1;
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
    const observed = observeAppleCommand(child, log, entry, {
      onExit: () => children.delete(child),
    });
    controls.set(child, observed);
    return { child, ...observed, entry };
  }

  async function run(command, args, label, timeout = 60_000) {
    const launched = start(command, args, label);
    let forceTimer;
    const timer = setTimeout(() => {
      // Process exit wins over a timeout while its final output is still draining.
      if (launched.child.exitCode !== null || launched.child.signalCode !== null) return;
      launched.entry.timedOut = true;
      stop(launched.child);
      forceTimer = setTimeout(() => {
        stop(launched.child, 'SIGKILL');
        launched.abandon();
      }, 10_000);
    }, timeout);
    try {
      const result = await launched.completion;
      assert.ok(!result.error, `${label} failed: ${result.error}; see ${launched.entry.log}`);
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
      report.textInput = await configureDesktopTextInput(run);
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
        simulatorBootTimeout,
      );
      // System Safari belongs to the exact runtime used to create this simulator.
      // Reading its immutable bundle avoids a booted LaunchServices lookup, which
      // can stall even after simctl bootstatus has reported successful startup.
      const mobileSafariApp = mobileSafariAppPath(selected.runtime);
      report.mobileSafari = {
        app: mobileSafariApp,
        identitySource: 'selected simulator runtime bundle',
      };
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
      // Compile before session startup: cold builds must not consume WDA's
      // readiness polling budget. Appium then runs only test-without-building.
      // https://appium.github.io/appium-xcuitest-driver/latest/guides/run-prebuilt-wda/
      const derivedDataPath = join(work, 'wda-derived-data');
      report.wdaBuild = {
        ...(await resolveWdaPackage(
          join(env.APPIUM_HOME, 'node_modules/appium-xcuitest-driver/package.json'),
        )),
        derivedDataPath,
        simulator,
        timeoutMs: 480_000,
        status: 'building',
      };
      report.installedTools.webdriveragent = report.wdaBuild.version;
      try {
        await run(
          '/usr/bin/xcodebuild',
          [
            'build-for-testing',
            '-project',
            report.wdaBuild.projectPath,
            '-scheme',
            'WebDriverAgentRunner',
            '-derivedDataPath',
            derivedDataPath,
            '-destination',
            `id=${simulator}`,
            `IPHONEOS_DEPLOYMENT_TARGET=${config.platformVersion}`,
            'GCC_TREAT_WARNINGS_AS_ERRORS=0',
            'COMPILER_INDEX_STORE_ENABLE=NO',
            'CODE_SIGNING_ALLOWED=NO',
          ],
          'build-wda',
          report.wdaBuild.timeoutMs,
        );
        report.wdaBuild.outputs = await inspectWdaBuild(derivedDataPath);
        report.wdaBuild.status = 'passed';
      } catch (error) {
        report.wdaBuild.status = 'failed';
        throw error;
      }
      const driver = start(
        process.execPath,
        [appium, '--address', '127.0.0.1', '--port', '4444', '--log-timestamp'],
        'appium',
      );
      // Loading XCUITest on a cold hosted runner can exceed the other services' startup budget.
      await ready('http://127.0.0.1:4444/status', driver, 120_000);
      capabilities = {
        platformName: 'iOS',
        'appium:automationName': 'XCUITest',
        'appium:udid': simulator,
        'appium:platformVersion': config.platformVersion,
        'appium:deviceName': config.deviceType,
        'appium:derivedDataPath': derivedDataPath,
        'appium:usePrebuiltWDA': true,
        // Reuse the booted simulator without Appium restarting it to show its UI.
        'appium:isHeadless': true,
        'appium:showXcodeLog': true,
        'appium:simulatorStartupTimeout': simulatorBootTimeout,
        'appium:wdaLaunchTimeout': 180000,
        'appium:wdaStartupRetries': 1,
        'appium:wdaLocalPort': wdaLocalPort,
        // Native Safari input can require three calibrated taps. Bound each XCTest idle
        // wait while retaining quiescence checks and the harness's explicit DOM readiness.
        'appium:waitForIdleTimeout': mobileIdleTimeoutSeconds,
        // Cold hosted simulators can publish Safari's inspector application after
        // the driver's 5-second default discovery deadline.
        'appium:webviewConnectTimeout': 60000,
        'appium:safariLogAllCommunication': true,
        'appium:newCommandTimeout': 120,
        'appium:nativeWebTap': true,
        'appium:screenshotQuality': 0,
        'appium:safariInitialUrl': 'http://127.0.0.1:4174/',
      };
      report.nativeInteraction = {
        waitForIdleTimeoutSeconds: mobileIdleTimeoutSeconds,
        quiescence: 'Enabled by the pinned XCUITest driver',
        wdaUrl: `http://127.0.0.1:${wdaLocalPort}`,
        source:
          'https://github.com/appium/appium-xcuitest-driver/blob/v12.13.3/lib/commands/wda/startup.ts#L389-L394',
      };
    }
    Object.assign(env, {
      KFP_WEBDRIVER_URL: 'http://127.0.0.1:4444',
      KFP_WDA_URL: mode === 'desktop' ? '' : `http://127.0.0.1:${wdaLocalPort}`,
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
    report.status = report.processErrors?.length ? 'failed' : 'passed';
  } catch (error) {
    report.status = 'failed';
    report.error = error.stack;
    process.exitCode = 1;
    console.error(error);
    if (simulator && !interrupted) {
      try {
        const path = 'failure-simulator.png';
        await run(
          '/usr/bin/xcrun',
          ['simctl', 'io', simulator, 'screenshot', '--type=png', join(out, path)],
          'failure-simulator-screenshot',
          15_000,
        );
        const png = await readFile(join(out, path));
        assert.equal(png.subarray(0, 8).toString('hex'), '89504e470d0a1a0a');
        report.failureScreenshot = {
          path,
          sha256: createHash('sha256').update(png).digest('hex'),
        };
      } catch (diagnosticError) {
        // Diagnostics must never replace the qualification failure or prevent cleanup.
        report.diagnosticErrors = [diagnosticError.message];
      }
    }
  } finally {
    process.off('SIGTERM', interrupt);
    process.off('SIGINT', interrupt);
    for (const child of children) stop(child);
    // Allow graceful driver/session shutdown, then kill only owned process groups.
    await delay(1_000);
    for (const child of children) {
      stop(child, 'SIGKILL');
      controls.get(child)?.abandon();
    }
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
