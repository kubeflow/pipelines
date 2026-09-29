/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// Native browser workflow qualification for browsers unsupported by Playwright.
// Start the selected WebDriver separately. This script creates and closes a fresh session.
// It reads fixture data and changes only transient UI selection/theme; no backend mutations.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';
import { setTimeout as delay } from 'node:timers/promises';
import { requestWebDriver } from './ui-modernization-native-http.mjs';
import {
  clickNativeSafariLink,
  clickNativeSafariRadio,
  inspectMobileTarget,
  planMobileGraphPan,
  panNativeSafariGraph,
  scrollMobileTargetIntoView,
  prepareNativeSafariTap,
} from './ui-modernization-native-safari.mjs';

const driver = new URL(process.env.KFP_WEBDRIVER_URL || 'http://127.0.0.1:4444');
const base = new URL(process.env.KFP_BROWSER_FLOOR_URL || 'http://127.0.0.1:4174/');
for (const url of [driver, base]) {
  assert.ok(
    ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname),
    'use loopback fixtures only',
  );
  assert.equal(url.protocol, 'http:');
}
assert.equal(process.env.CI, 'true', 'Native browser qualification runs only in CI');
assert.equal(process.env.GITHUB_ACTIONS, 'true', 'Native qualification requires GitHub Actions');
assert.equal(
  process.env.RUNNER_ENVIRONMENT,
  'github-hosted',
  'Native qualification requires a disposable GitHub-hosted runner',
);
const mobile = process.env.KFP_BROWSER_FLOOR_MOBILE === '1';
const extraCapabilities = JSON.parse(process.env.KFP_WEBDRIVER_CAPABILITIES || '{}');
const browserName = process.env.KFP_WEBDRIVER_BROWSER || 'firefox';
assert.ok(['firefox', 'safari', 'chrome', 'MicrosoftEdge'].includes(browserName));
const expectedVersion = process.env.KFP_BROWSER_FLOOR_VERSION?.trim();
assert.ok(
  expectedVersion,
  'Set KFP_BROWSER_FLOOR_VERSION to the exact browser version in the release qualification matrix',
);
const out = process.env.KFP_BROWSER_FLOOR_OUTPUT || '/tmp/kfp-browser-floor-evidence';
const elementKey = 'element-6066-11e4-a52e-4f735466cecf';
const runId = 'e0115ac1-0479-4194-a22d-01e65e09a32b';
const nodeSelector = '.react-flow__node[data-id="task.chicago-taxi-trips-dataset"]';
const report = {
  recordedAt: new Date().toISOString(),
  status: 'running',
  expectedVersion,
  browserName,
  mobile,
  sourceRevision:
    process.env.GITHUB_SHA ||
    execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
  checks: [],
  screenshots: [],
  gaps: mobile
    ? [
        'Hardware keyboard Tab/Space/Escape focus containment is not exercised by the touch-only simulator lane.',
        'Simulator Safari does not establish physical-device, VoiceOver, or pinch-zoom behavior.',
        'Mobile text entry uses Appium WebDriver web typing atoms and input events; it does not establish physical keyboard typing.',
      ]
    : [],
  limitations: [
    'Native small fixtures; comparison has two runs but no parameters or scalar metrics.',
    'Read-only fixtures cover navigation and unsubmitted form drafts, not cluster mutations, authorization, upload submission, or full 48-case Playwright suite parity.',
    'Page readiness and captured errors are checked; errors before initial Runs readiness are not observed, and this is not a complete network or console trace.',
  ],
};
let session;
let mobileStartup = mobile;
if (mobile) report.nativeSafariStartupCommandTimeoutMs = 60000;
await mkdir(out, { recursive: true });

async function command(
  method,
  path,
  body,
  timeout = mobile && path === '/session' ? 600000 : mobileStartup ? 60000 : 40000,
) {
  return requestWebDriver(new URL(path, driver), { method, body, timeout });
}

const execute = (fn, ...args) =>
  command('POST', `/session/${session}/execute/sync`, {
    script: `return (${fn.toString()})(...arguments);`,
    args,
  });
async function wait(fn, label, ...args) {
  const deadline = Date.now() + 20000;
  while (Date.now() < deadline) {
    const value = await execute(fn, ...args);
    if (value) return value;
    await delay(100);
  }
  throw new Error(`Timed out: ${label}`);
}
async function find(selector) {
  return command('POST', `/session/${session}/element`, { using: 'css selector', value: selector });
}
async function prepareMobileTap() {
  if (!mobile) return;
  report.nativeSafariPreparation ??= [];
  await prepareNativeSafariTap(
    command,
    session,
    report.nativeSafariPreparation,
    async (source, label) => {
      const path = `${label}-${report.nativeSafariPreparation.length}.xml`;
      await writeFile(join(out, path), source);
      report.nativeSafariSources ??= [];
      report.nativeSafariSources.push({ path });
    },
    base.origin,
  );
}
async function readyMobileTarget(element, label) {
  if (!mobile) return;
  report.nativeSafariTargets ??= [];
  const entry = { label, startedAt: new Date().toISOString(), status: 'pending' };
  report.nativeSafariTargets.push(entry);
  try {
    entry.before = await execute(inspectMobileTarget, element);
    // Viewport placement is setup; activation still uses a native WebDriver tap.
    entry.placement = await execute(scrollMobileTargetIntoView, element);
    assert.deepEqual(
      entry.placement.canvasScrollAfter,
      entry.placement.canvasScrollBefore,
      'Viewport placement must preserve React Flow internal scroll offsets',
    );
    entry.after = await execute(inspectMobileTarget, element);
    for (let attempt = 0; attempt < 2 && !entry.after.ready; attempt++) {
      const plan = await execute(planMobileGraphPan, element, entry.after);
      if (!plan) break;
      entry.pans ??= [];
      const beforePan = entry.after;
      await panNativeSafariGraph(command, session, plan, entry.pans);
      entry.after = await execute(inspectMobileTarget, element);
      entry.pans.at(-1).targetAfter = entry.after;
      const actualX = entry.after.center.x - beforePan.center.x;
      const actualY = entry.after.center.y - beforePan.center.y;
      const plannedX = plan.to.x - plan.from.x;
      const plannedY = plan.to.y - plan.from.y;
      assert.ok(
        actualX * plannedX + actualY * plannedY > 1,
        'Native graph pan must move the node in the requested direction',
      );
    }
    const deadline = Date.now() + 20000;
    do {
      entry.after = await execute(inspectMobileTarget, element);
      if (entry.after.ready) break;
      await delay(100);
    } while (Date.now() < deadline);
    assert.ok(entry.after.ready, `Mobile target is outside the viewport or occluded: ${label}`);
    entry.status = 'passed';
  } catch (error) {
    entry.status = 'failed';
    entry.error = String(error);
    throw error;
  }
}
async function click(selector) {
  await prepareMobileTap();
  const element = await find(selector);
  await readyMobileTarget(element, selector);
  await command('POST', `/session/${session}/element/${element[elementKey]}/click`, {});
}
async function type(selector, text) {
  await prepareMobileTap();
  const element = await find(selector);
  await readyMobileTarget(element, selector);
  await command('POST', `/session/${session}/element/${element[elementKey]}/value`, { text });
}
async function key(value, shift = false) {
  const actions = [];
  if (shift) actions.push({ type: 'keyDown', value: '\uE008' });
  actions.push({ type: 'keyDown', value }, { type: 'keyUp', value });
  if (shift) actions.push({ type: 'keyUp', value: '\uE008' });
  await command('POST', `/session/${session}/actions`, {
    actions: [{ type: 'key', id: 'keyboard', actions }],
  });
}
async function screenshot(name) {
  const data = Buffer.from(await command('GET', `/session/${session}/screenshot`), 'base64');
  await writeFile(join(out, `${name}.png`), data);
  report.screenshots.push({
    path: `${name}.png`,
    sha256: createHash('sha256').update(data).digest('hex'),
  });
}
async function check(name, exercise) {
  const startedAt = new Date().toISOString();
  try {
    const detail = await exercise();
    report.checks.push({ name, status: 'passed', startedAt, detail: detail ?? null });
    console.log(`PASS ${name}`);
  } catch (error) {
    report.checks.push({ name, status: 'failed', startedAt, error: String(error) });
    throw error;
  }
}
async function navigate(hash) {
  await execute((next) => {
    location.hash = next;
  }, hash);
}
async function textElement(selector, text) {
  return wait(
    (selector, text) =>
      Array.from(document.querySelectorAll(selector)).find(
        (element) =>
          element.textContent.trim() === text && element.getBoundingClientRect().width > 0,
      ),
    `${selector}: ${text}`,
    selector,
    text,
  );
}
async function clickText(selector, text) {
  await prepareMobileTap();
  const element = await textElement(selector, text);
  await readyMobileTarget(element, `${selector}: ${text}`);
  if (mobile && selector === 'label' && ['Recurring', 'One-off'].includes(text)) {
    assert.deepEqual(
      await execute((label) => ({ type: label.control?.type, name: label.control?.name }), element),
      { type: 'radio', name: 'runType' },
      'Native run-type action requires the associated radio control',
    );
    report.nativeSafariRadioClicks ??= [];
    await clickNativeSafariRadio(command, session, text, report.nativeSafariRadioClicks);
    await wait(
      (name) =>
        Array.from(document.querySelectorAll('label')).find(
          (label) => label.textContent.trim() === name,
        )?.control?.checked === true,
      'native run-type radio selected',
      text,
    );
  } else if (mobile && selector === 'a') {
    // Safari exposes both an inline link and its text child with the same label.
    // Activate the unique native Link, preserving its actual accessible action.
    const href = await execute((target) => target.getAttribute('href'), element);
    assert.ok(href?.startsWith('#/'), 'Native fixture link must retain its hash route');
    report.nativeSafariLinkClicks ??= [];
    await clickNativeSafariLink(command, session, text, report.nativeSafariLinkClicks);
    await wait((expected) => location.hash === expected, 'native link route', href);
  } else {
    await command('POST', `/session/${session}/element/${element[elementKey]}/click`, {});
  }
}
async function field(label) {
  return wait(
    (label) =>
      Array.from(document.querySelectorAll('label')).find(
        (element) => element.textContent.replace(/\s*\*\s*$/, '').trim() === label,
      )?.control,
    `field ${label}`,
    label,
  );
}
async function fillLabel(label, text) {
  await prepareMobileTap();
  const element = await field(label);
  await readyMobileTarget(element, label);
  if (browserName === 'safari' && !mobile) {
    // Safari's Element Clear can empty the DOM without sending the input event that
    // updates React-controlled state. Exercise an actual user edit instead.
    await command('POST', `/session/${session}/element/${element[elementKey]}/click`, {});
    await command('POST', `/session/${session}/actions`, {
      actions: [
        {
          type: 'key',
          id: 'keyboard',
          actions: [
            { type: 'keyDown', value: '\uE03D' },
            { type: 'keyDown', value: 'a' },
            { type: 'keyUp', value: 'a' },
            { type: 'keyUp', value: '\uE03D' },
            { type: 'keyDown', value: '\uE003' },
            { type: 'keyUp', value: '\uE003' },
          ],
        },
      ],
    });
  } else {
    await command('POST', `/session/${session}/element/${element[elementKey]}/clear`, {});
  }
  if (text && browserName === 'safari' && !mobile) {
    // Pace individual native keyboard events; exact-value assertions below still
    // verify the complete field content without correcting it.
    await command('POST', `/session/${session}/actions`, {
      actions: [
        {
          type: 'key',
          id: 'keyboard',
          actions: Array.from(text).flatMap((value) => [
            { type: 'keyDown', value },
            { type: 'keyUp', value },
            { type: 'pause', duration: 50 },
          ]),
        },
      ],
    });
  } else if (text) {
    await command('POST', `/session/${session}/element/${element[elementKey]}/value`, { text });
  }
}
async function valueLabel(label) {
  const element = await field(label);
  return command('GET', `/session/${session}/element/${element[elementKey]}/property/value`);
}
async function theme(value) {
  // Safari's native select popup is outside the web context on iOS. Use its change event
  // for theme setup; interaction assertions continue through WebDriver click/sendkeys.
  await execute((value) => {
    const select = document.querySelector('select[aria-label="Theme"]');
    select.value = value;
    select.dispatchEvent(new Event('change', { bubbles: true }));
  }, value);
  await wait(
    (dark) => document.querySelector('.kfp-theme')?.classList.contains('dark') === dark,
    `${value} theme`,
    value === 'dark',
  );
}
const visibleNode = (selector) => {
  const node = document.querySelector(selector);
  return (
    !!node &&
    getComputedStyle(node).visibility === 'visible' &&
    node.getBoundingClientRect().width > 0
  );
};

try {
  const capabilities = { ...extraCapabilities, browserName, acceptInsecureCerts: false };
  if (browserName === 'firefox') {
    capabilities['moz:firefoxOptions'] = {
      ...(process.env.KFP_FIREFOX_BINARY ? { binary: process.env.KFP_FIREFOX_BINARY } : {}),
      args: ['-headless'],
      prefs: { 'app.update.auto': false, 'browser.shell.checkDefaultBrowser': false },
    };
  }
  if (mobile) {
    assert.equal(browserName, 'safari');
    assert.equal(capabilities.platformName?.toLowerCase(), 'ios');
    assert.ok(process.env.KFP_EXPECTED_PLATFORM_VERSION, 'Exact iOS runtime version required');
    assert.equal(capabilities['appium:platformVersion'], process.env.KFP_EXPECTED_PLATFORM_VERSION);
  }
  const created = await command('POST', '/session', {
    capabilities: { alwaysMatch: capabilities },
  });
  session = created.sessionId;
  report.capabilities = created.capabilities;
  if (mobile) {
    assert.equal(
      created.capabilities['appium:platformVersion'] || created.capabilities.platformVersion,
      process.env.KFP_EXPECTED_PLATFORM_VERSION,
      'must run requested iOS runtime',
    );
    const expectedIdleTimeout = capabilities['appium:waitForIdleTimeout'];
    assert.ok(
      typeof expectedIdleTimeout === 'number' && expectedIdleTimeout > 0,
      'Native idle waits must remain enabled',
    );
    const wda = new URL(process.env.KFP_WDA_URL);
    assert.equal(wda.origin, `http://127.0.0.1:${capabilities['appium:wdaLocalPort']}`);
    // Appium's settings route reads its own cache; query the owned WDA directly.
    report.nativeDriverStatus = await requestWebDriver(new URL('/status', wda), {
      method: 'GET',
      timeout: 40000,
      envelope: true,
    });
    const wdaSession = report.nativeDriverStatus.sessionId;
    assert.ok(
      typeof wdaSession === 'string' && wdaSession.length > 0,
      'Active WDA session required',
    );
    report.nativeDriverSettings = await requestWebDriver(
      new URL(`/session/${encodeURIComponent(wdaSession)}/appium/settings`, wda),
      { method: 'GET', timeout: 40000 },
    );
    assert.equal(
      report.nativeDriverSettings.waitForIdleTimeout,
      expectedIdleTimeout,
      'WDA must apply the requested positive idle wait',
    );
  } else {
    assert.equal(
      created.capabilities.browserVersion,
      expectedVersion,
      'must run the requested real version',
    );
  }
  await command('POST', `/session/${session}/timeouts`, {
    implicit: 0,
    pageLoad: 30000,
    script: 10000,
  });
  if (!mobile)
    await command('POST', `/session/${session}/window/rect`, { width: 1440, height: 900 });
  // Browser-owned onboarding and address editing can obstruct the first page query.
  await prepareMobileTap();
  await command('POST', `/session/${session}/url`, { url: new URL('#/runs', base).href });
  await wait(
    () => document.querySelectorAll('[data-testid="run-name-link"]').length === 4,
    'four fixture runs',
  );
  // Appium can finish JavaScript while its first native alert probe still runs.
  // Confine the cold-start allowance to first-page readiness; checks retain 40s.
  mobileStartup = false;
  if (mobile) report.nativeSafariStartupCompletedAt = new Date().toISOString();
  report.environment = await execute(() => ({
    userAgent: navigator.userAgent,
    width: innerWidth,
    height: innerHeight,
    dpr: devicePixelRatio,
    maxTouchPoints: navigator.maxTouchPoints,
  }));
  if (mobile) {
    assert.equal(
      report.environment.userAgent.match(/Version\/([\d.]+)/)?.[1],
      expectedVersion,
      'must run requested Safari version',
    );
    assert.ok(report.environment.maxTouchPoints > 0, 'mobile lane must expose a touch device');
  }
  await execute(() => {
    window.floorErrors = [];
    addEventListener('error', (event) => window.floorErrors.push(event.message));
    addEventListener('unhandledrejection', (event) =>
      window.floorErrors.push(String(event.reason)),
    );
  });
  const assetUrls = await execute(() => [
    ...Array.from(document.querySelectorAll('script[type="module"][src]'), (el) => el.src),
    ...Array.from(document.querySelectorAll('link[rel="stylesheet"]'), (el) => el.href),
  ]);
  report.assets = [];
  for (const assetUrl of assetUrls) {
    assert.equal(new URL(assetUrl).origin, base.origin);
    const response = await fetch(assetUrl);
    assert.equal(response.status, 200);
    const data = Buffer.from(await response.arrayBuffer());
    report.assets.push({
      path: new URL(assetUrl).pathname,
      bytes: data.length,
      sha256: createHash('sha256').update(data).digest('hex'),
    });
  }
  await check('required platform features', async () => {
    const features = await execute(() => ({
      colorMix: CSS.supports('color', 'color-mix(in srgb, red, blue)'),
      dynamicViewport: CSS.supports('height', '100dvh'),
      focusVisible: CSS.supports('selector(:focus-visible)'),
      registeredProperties: typeof CSS.registerProperty === 'function',
      resizeObserver: typeof ResizeObserver === 'function',
      weakRef: typeof WeakRef === 'function',
      structuredClone: typeof structuredClone === 'function',
      objectHasOwn: typeof Object.hasOwn === 'function',
      arrayAt: typeof Array.prototype.at === 'function',
    }));
    assert.ok(Object.values(features).every(Boolean), JSON.stringify(features));
    const nativeInputActivation = await execute(() => {
      const activated = (Constructor) => {
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.dispatchEvent(
          new Constructor('click', { bubbles: true, cancelable: true, composed: true }),
        );
        return input.checked;
      };
      return { pointerEvent: activated(PointerEvent), mouseEvent: activated(MouseEvent) };
    });
    assert.equal(nativeInputActivation.mouseEvent, true);
    return { features, nativeInputActivation };
  });
  await check('Runs loaded names and WebDriver name filter', async () => {
    const names = await execute(() =>
      Array.from(
        document.querySelectorAll('[data-testid="run-name-link"]'),
        (el) => el.textContent,
      ),
    );
    assert.deepEqual(
      [...names].sort(),
      ['Python two steps', 'Loops and conditions', 'v2-xgboost-ilbo', 'Various IO types'].sort(),
    );
    const beforeInputScale = mobile ? await execute(() => window.visualViewport?.scale ?? 1) : null;
    await type('input[placeholder="Filter runs by name"]', 'xgboost');
    await wait(() => {
      const links = document.querySelectorAll('[data-testid="run-name-link"]');
      return links.length === 1 && links[0].textContent === 'v2-xgboost-ilbo';
    }, 'one matching run');
    let afterInputScale = null;
    if (mobile) {
      await prepareMobileTap();
      afterInputScale = await execute(() => window.visualViewport?.scale ?? 1);
      assert.ok(
        Math.abs(afterInputScale - beforeInputScale) < 0.01,
        'Filtering must not automatically zoom the page and hide subsequent dialog controls',
      );
    }
    await screenshot('runs-filtered');
    return {
      initialNames: names,
      filteredName: 'v2-xgboost-ilbo',
      beforeInputScale,
      afterInputScale,
    };
  });
  await check('Run Details graph and task inspection', async () => {
    const runSelector = `[data-testid="run-name-link"][data-run-id="${runId}"]`;
    if (mobile) {
      await prepareMobileTap();
      assert.equal(
        await execute((selector) => document.querySelector(selector)?.textContent, runSelector),
        'v2-xgboost-ilbo',
      );
      await readyMobileTarget(await find(runSelector), runSelector);
      report.nativeSafariLinkClicks ??= [];
      await clickNativeSafariLink(
        command,
        session,
        'v2-xgboost-ilbo',
        report.nativeSafariLinkClicks,
      );
    } else {
      await click(runSelector);
    }
    await wait(visibleNode, 'visible native task node', nodeSelector);
    const geometry = await execute((selector) => {
      const box = document.querySelector(selector).getBoundingClientRect();
      return {
        width: box.width,
        height: box.height,
        nodes: document.querySelectorAll('.react-flow__node').length,
      };
    }, nodeSelector);
    await click(`${nodeSelector} button.kfp-graph-node`);
    await wait(() => !!document.querySelector('.kfp-inspector-panel'), 'task inspector');
    await screenshot('run-details-inspector');
    await click('.kfp-inspector-panel button[aria-label="close"]');
    await wait(() => !document.querySelector('.kfp-inspector-panel'), 'inspector closes');
    await wait(visibleNode, 'graph remains measured after close', nodeSelector);
    return geometry;
  });
  await check(
    mobile ? 'Pipelines cards and touch selection' : 'Pipelines cards and keyboard selection',
    async () => {
      if (mobile) await navigate('#/pipelines');
      else await click('a[aria-label="Pipelines"]');
      await wait(
        () => document.querySelectorAll('[data-testid="pipeline-card"]').length === 4,
        'four pipeline cards',
      );
      await type('input[placeholder="Filter pipelines"]', 'XGBoost');
      await wait(
        () => document.querySelectorAll('[data-testid="pipeline-card"]').length === 1,
        'one pipeline card',
      );
      const selector = '[role="checkbox"][aria-label="Select pipeline XGBoost"]';
      await wait(
        (s) => {
          const checkbox = document.querySelector(s);
          return (
            checkbox &&
            checkbox.getAttribute('aria-disabled') !== 'true' &&
            document.querySelector('ul[aria-label="Pipelines"]')?.getAttribute('aria-busy') ===
              'false'
          );
        },
        'filtered pipeline selection enabled',
        selector,
      );
      await click(selector);
      await wait(
        (s) => document.querySelector(s)?.getAttribute('aria-checked') === 'true',
        'pipeline selected',
        selector,
      );
      if (mobile) await click(selector);
      else await key(' ');
      await wait(
        (s) => document.querySelector(s)?.getAttribute('aria-checked') === 'false',
        'Space deselects pipeline',
        selector,
      );
      await screenshot('pipeline-cards');
    },
  );
  await check(
    'Pipeline details load production YAML editor and retain read-only content',
    async () => {
      const id = '8fbe3bd6-a01f-11e8-98d0-529269fb1460';
      await navigate(`#/pipelines/details/${id}/version/${id}`);
      await wait(visibleNode, 'pipeline graph', '[data-testid="DagCanvas"]');
      await clickText('[role="tab"]', 'Pipeline Spec');
      await wait(
        () =>
          !!document.querySelector('.ace_editor .ace_content') &&
          document
            .querySelector('[data-testid="spec-ir"]')
            ?.textContent.includes('comp-preprocess'),
        'rendered pipeline spec',
      );
      // Ace virtualizes rendered lines, so focus can change visible text without
      // changing the document. Verify the complete existing editor model instead.
      const readEditor = () => {
        const editor = document.querySelector('[data-testid="spec-ir"] .ace_editor')?.env?.editor;
        if (!editor) throw new Error('The rendered pipeline spec has no initialized Ace editor');
        return { value: editor.getValue(), readOnly: editor.getReadOnly() };
      };
      const before = await execute(readEditor);
      assert.equal(before.readOnly, true);
      assert.ok(before.value.includes('comp-preprocess'));
      if (!mobile) {
        // Ace's read-only textarea rejects Element Send Keys in some native drivers.
        // Focus it, then exercise the same keyboard action a user would send.
        assert.equal(
          await execute(() => {
            const input = document.querySelector('[data-testid="spec-ir"] .ace_text-input');
            input.focus();
            return document.activeElement === input;
          }),
          true,
        );
        await key('x');
        assert.deepEqual(await execute(readEditor), before);
      }
      await screenshot('pipeline-spec-light');
      await theme('dark');
      await screenshot('pipeline-spec-dark');
      await theme('light');
      assert.deepEqual(await execute(readEditor), before);
      return {
        modelCharacters: before.value.length,
        readOnly: before.readOnly,
        readOnlyTypingChecked: !mobile,
      };
    },
  );
  await check('Pipeline import draft validates required fields and local-file choice', async () => {
    await navigate('#/pipeline_versions/new');
    await fillLabel('Pipeline Name', 'native-browser-draft');
    await fillLabel('Package Url', 'https://example.test/fixture.yaml');
    assert.equal(await valueLabel('Pipeline Name'), 'native-browser-draft');
    assert.equal(await valueLabel('Package Url'), 'https://example.test/fixture.yaml');
    // Import modes retain their drafts. Clear the URL before asserting that no package exists.
    await fillLabel('Package Url', '');
    await wait(
      () =>
        document.querySelector('#createNewPipelineOrVersionBtn')?.disabled &&
        document
          .querySelector('.kfp-pipeline-form-error')
          ?.textContent.includes('Must specify either package url'),
      'empty URL and file disable creation',
    );
    await clickText('label', 'Upload a file');
    await wait(
      () =>
        !!document.querySelector('input[type="file"]') &&
        !document.querySelector('input[type="file"]').disabled &&
        document.querySelector('input[type="file"]').files.length === 0 &&
        document.querySelector('#createNewPipelineOrVersionBtn')?.disabled,
      'empty file upload enabled while creation remains disabled',
    );
    const create = await textElement('button', 'Create');
    assert.equal(
      await execute((element) => element.disabled, create),
      true,
      'upload draft without a file cannot submit',
    );
    await screenshot('pipeline-upload-draft');
    return { submitted: false, fileChooser: 'enabled', missingPackageRejected: true };
  });
  await check('New experiment draft preserves name and description', async () => {
    await navigate('#/experiments/new');
    // Observe native event delivery and controlled-field commits without changing
    // values or input behavior. This distinguishes driver/OS edits from UI resets.
    await execute(() => {
      const ids = new Set(['experimentName', 'experimentDescription']);
      const trace = { events: [], dropped: 0 };
      window.floorInputTrace = trace;
      const record = (event, phase) => {
        const target = event.target;
        if (!ids.has(target?.id)) return;
        if (trace.events.length >= 800) {
          trace.dropped++;
          return;
        }
        trace.events.push({
          time: performance.now(),
          phase,
          type: event.type,
          id: target.id,
          key: event.key ?? null,
          data: event.data ?? null,
          inputType: event.inputType ?? null,
          composing: event.isComposing ?? null,
          trusted: event.isTrusted,
          prevented: event.defaultPrevented,
          modifiers: {
            meta: event.metaKey ?? false,
            control: event.ctrlKey ?? false,
            alt: event.altKey ?? false,
            shift: event.shiftKey ?? false,
          },
          active: document.activeElement?.id || null,
          value: target.value,
          selectionStart: target.selectionStart,
          selectionEnd: target.selectionEnd,
        });
      };
      for (const type of [
        'keydown',
        'keyup',
        'beforeinput',
        'input',
        'change',
        'compositionstart',
        'compositionupdate',
        'compositionend',
        'focus',
        'blur',
      ]) {
        document.addEventListener(
          type,
          (event) => {
            record(event, 'capture');
            if (event.type === 'input') {
              queueMicrotask(() => record(event, 'after event'));
              requestAnimationFrame(() => record(event, 'next animation frame'));
            }
          },
          true,
        );
      }
    });
    try {
      await fillLabel('Experiment name', 'Native browser experiment');
      report.experimentDraftValues = { nameBeforeBlur: await valueLabel('Experiment name') };
      await fillLabel('Description', 'Unsubmitted automated qualification draft');
      Object.assign(report.experimentDraftValues, {
        nameAfterBlur: await valueLabel('Experiment name'),
        descriptionBeforeBlur: await valueLabel('Description'),
      });
      assert.equal(report.experimentDraftValues.nameAfterBlur, 'Native browser experiment');
      assert.equal(
        report.experimentDraftValues.descriptionBeforeBlur,
        'Unsubmitted automated qualification draft',
      );
      await screenshot('experiment-draft');
      return { submitted: false };
    } finally {
      try {
        report.inputDiagnostics = await execute(() => window.floorInputTrace);
      } catch (error) {
        report.inputDiagnosticError = String(error);
      }
    }
  });
  await check(
    'One-off and recurring creation retain pipeline and editable form state',
    async () => {
      const id = '8fbe3bd6-a01f-11e8-98d0-529269fb1460';
      await navigate(
        `#/runs/new?pipelineId=${id}&pipelineVersionId=${id}&experimentId=275ea11d-ac63-4ce3-bc33-ec81981ed56b`,
      );
      await field('Run name');
      await wait(
        () =>
          Array.from(document.querySelectorAll('input')).some(
            (input) => input.value === 'Python two steps',
          ),
        'selected pipeline loaded',
      );
      await fillLabel('Run name', 'Native workflow draft');
      await fillLabel('Description', 'Retained between run types');
      await clickText('label', 'Recurring');
      assert.equal(await valueLabel('Recurring run config name'), 'Native workflow draft');
      await fillLabel('Maximum concurrent runs', '3');
      assert.equal(await valueLabel('Maximum concurrent runs'), '3');
      await screenshot('recurring-run-draft');
      await clickText('label', 'One-off');
      assert.equal(await valueLabel('Run name'), 'Native workflow draft');
      assert.equal(await valueLabel('Description'), 'Retained between run types');
      await screenshot('one-off-run-draft');
      return { submitted: false, pipeline: 'Python two steps', concurrency: 3 };
    },
  );
  await check('Artifact details, related tasks and directed lineage load', async () => {
    await navigate('#/artifacts');
    await clickText('a', 'mock-dataset');
    await wait(() => document.body.innerText.includes('mock-artifact-1'), 'artifact metadata');
    await clickText('[role="tab"]', 'Related tasks');
    await wait(
      () => document.querySelector('table[aria-label="Related tasks"] a'),
      'related task navigation',
    );
    const links = await execute(() =>
      Array.from(document.querySelectorAll('table[aria-label="Related tasks"] a'), (element) => ({
        text: element.textContent,
        href: element.getAttribute('href'),
      })),
    );
    assert.ok(
      links.some((link) => link.href.includes('/runs/details/') && link.href.includes('task=')),
    );
    await clickText('[role="tab"]', 'Lineage Explorer');
    await wait(
      () =>
        !!document.querySelector('[aria-label="Lineage history"]') &&
        !!document.querySelector('[aria-label="Producer Chicago taxi trips dataset"] a') &&
        !!document.querySelector('[aria-label="Consumer Convert CSV to Apache Parquet"] a'),
      'resolved producer and consumer lineage',
    );
    await screenshot('artifact-lineage-light');
    await theme('dark');
    await screenshot('artifact-lineage-dark');
    await theme('light');
    return { relatedTasks: links };
  });
  await check(
    mobile
      ? 'Switch touch activation preserves unsubmitted feature state'
      : 'Switch pointer and Space activation preserve unsubmitted feature state',
    async () => {
      await execute(() => {
        location.hash = '#/frontend_features';
      });
      const selector = '[role="switch"][aria-label="Enable functional_component"]';
      await wait((s) => !!document.querySelector(s), 'feature draft switch', selector);
      const before = await execute(
        (s) => ({
          checked: document.querySelector(s).getAttribute('aria-checked'),
          stored: localStorage.getItem('flags'),
          runtime: window.__FEATURE_FLAGS__ || null,
        }),
        selector,
      );
      await click(selector);
      await wait(
        (s, original) => document.querySelector(s)?.getAttribute('aria-checked') !== original,
        'pointer toggles feature draft',
        selector,
        before.checked,
      );
      if (mobile) await click(selector);
      else await key(' ');
      await wait(
        (s, original) => document.querySelector(s)?.getAttribute('aria-checked') === original,
        'Space restores feature draft',
        selector,
        before.checked,
      );
      assert.deepEqual(
        await execute(() => ({
          stored: localStorage.getItem('flags'),
          runtime: window.__FEATURE_FLAGS__ || null,
        })),
        { stored: before.stored, runtime: before.runtime },
      );
      await screenshot('feature-switch');
      return { persistedOrActivated: false };
    },
  );
  await check('comparison retains both native runs and honest empty data states', async () => {
    // Hash navigation retains the error observers installed in this fresh document.
    await execute((hash) => {
      location.hash = hash;
    }, `#/compare?runlist=mock-run-0,${runId}`);
    await wait(() => {
      const names = Array.from(
        document.querySelectorAll('[data-testid="run-name-link"]'),
        (el) => el.textContent,
      );
      return (
        names.length === 2 &&
        names.includes('Python two steps') &&
        names.includes('v2-xgboost-ilbo') &&
        document.body.innerText.includes(
          'There are no parameters available on the selected runs.',
        ) &&
        document.body.innerText.includes(
          'There are no scalar metrics artifacts available on the selected runs.',
        )
      );
    }, 'two selected runs and loaded comparison');
    const checkbox = '[role="checkbox"][aria-label="Select run Python two steps"]';
    await click(checkbox);
    await wait(
      (selector) => document.querySelector(selector)?.getAttribute('aria-checked') === 'false',
      'comparison run deselected',
      checkbox,
    );
    await click(checkbox);
    await wait(
      (selector) => document.querySelector(selector)?.getAttribute('aria-checked') === 'true',
      'comparison run selected again',
      checkbox,
    );
    await screenshot('comparison');
    return { selectedRuns: 2, parameters: 'empty fixture', scalarMetrics: 'empty fixture' };
  });
  await check(
    mobile
      ? 'dark theme and mobile command-dialog dismissal'
      : 'dark theme and narrow command-dialog keyboard containment',
    async () => {
      await theme('dark');
      await wait(
        () => document.querySelector('.kfp-theme')?.classList.contains('dark'),
        'dark theme',
      );
      const colors = await execute(() => {
        const el = document.querySelector('.kfp-theme');
        const style = getComputedStyle(el);
        return { foreground: style.color, background: style.backgroundColor };
      });
      assert.notEqual(colors.foreground, colors.background);
      if (!mobile)
        await command('POST', `/session/${session}/window/rect`, { width: 600, height: 900 });
      await click('button[aria-label="Search"]');
      await wait(
        () =>
          document.activeElement?.getAttribute('aria-label') ===
          'Search pipelines, experiments, and runs',
        'dialog autofocus',
      );
      if (!mobile) {
        await key('\uE004', true);
        await wait(() => {
          const dialog = document.querySelector('.kfp-page-dialog[role="dialog"]');
          const close = dialog?.querySelector('.kfp-command-footer button');
          return close?.textContent === 'Close' && document.activeElement === close;
        }, 'Shift+Tab wraps to final Close button');
        await key('\uE004');
        await wait(
          () =>
            document.activeElement?.getAttribute('aria-label') ===
            'Search pipelines, experiments, and runs',
          'Tab wraps back to input',
        );
      }
      await screenshot('command-dialog-narrow-dark');
      if (mobile) await click('.kfp-command-footer button');
      else await key('\uE00C');
      await wait(
        (isMobile) =>
          !document.querySelector('.kfp-page-dialog[role="dialog"]') &&
          (isMobile || document.activeElement?.getAttribute('aria-label') === 'Search'),
        'dialog closes and desktop focus returns',
        mobile,
      );
      assert.equal(await execute(() => document.documentElement.scrollWidth <= innerWidth), true);
      return colors;
    },
  );
  const fixture = await (await fetch(new URL('/__qualification', base))).json();
  report.fixture = fixture;
  assert.deepEqual(fixture.mutations, [], 'drafts must not submit backend mutations');
  assert.deepEqual(fixture.missingAssets, [], 'production assets must load without missing files');
  report.errors = await execute(() => window.floorErrors);
  assert.deepEqual(report.errors, []);
  report.status = 'passed';
} catch (error) {
  report.status = 'failed';
  report.error = error.stack || String(error);
  if (session) {
    try {
      report.failureState = await execute(() => ({
        url: location.href,
        text: document.body.innerText.slice(0, 5000),
        errors: window.floorErrors || [],
        pipelinesBusy: document
          .querySelector('ul[aria-label="Pipelines"]')
          ?.getAttribute('aria-busy'),
        pipelineCheckboxes: Array.from(document.querySelectorAll('[role="checkbox"]'), (el) => ({
          label: el.getAttribute('aria-label'),
          checked: el.getAttribute('aria-checked'),
          disabled: el.getAttribute('aria-disabled'),
        })),
      }));
      await screenshot('failure');
    } catch (captureError) {
      report.captureError = String(captureError);
    }
    if (mobile) {
      try {
        await command('POST', `/session/${session}/context`, { name: 'NATIVE_APP' });
        const source = await command('GET', `/session/${session}/source`);
        await writeFile(join(out, 'failure-native-source.xml'), source);
        report.nativeFailureSource = 'failure-native-source.xml';
      } catch (captureError) {
        report.nativeSourceCaptureError = String(captureError);
      }
    }
  }
  console.error(report.error);
  process.exitCode = 1;
} finally {
  if (session) {
    try {
      await command('DELETE', `/session/${session}`);
    } catch (error) {
      report.cleanupError = String(error);
      report.status = 'failed';
      process.exitCode = 1;
    }
  }
  await writeFile(join(out, 'result.json'), `${JSON.stringify(report, null, 2)}\n`);
  console.log(`Evidence: ${join(out, 'result.json')}`);
}
