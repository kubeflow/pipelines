/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// A bounded native-fixture smoke for real browser versions unsupported by Playwright.
// Start the selected WebDriver separately. This script creates and closes a fresh session.
// It reads fixture data and changes only transient UI selection/theme; no backend mutations.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const driver = new URL(process.env.KFP_WEBDRIVER_URL || 'http://127.0.0.1:4444');
const base = new URL(process.env.KFP_BROWSER_FLOOR_URL || 'http://127.0.0.1:4174/');
for (const url of [driver, base]) {
  assert.ok(
    ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname),
    'use loopback fixtures only',
  );
  assert.equal(url.protocol, 'http:');
}
const browserName = process.env.KFP_WEBDRIVER_BROWSER || 'firefox';
assert.ok(['firefox', 'safari', 'chrome', 'MicrosoftEdge'].includes(browserName));
const expectedVersion = process.env.KFP_BROWSER_FLOOR_VERSION || '128.0';
const out = process.env.KFP_BROWSER_FLOOR_OUTPUT || '/tmp/kfp-browser-floor-evidence';
const elementKey = 'element-6066-11e4-a52e-4f735466cecf';
const runId = 'e0115ac1-0479-4194-a22d-01e65e09a32b';
const nodeSelector = '.react-flow__node[data-id="task.chicago-taxi-trips-dataset"]';
const report = {
  recordedAt: new Date().toISOString(),
  status: 'running',
  expectedVersion,
  browserName,
  checks: [],
  screenshots: [],
  limitations: [
    'Native small fixtures; comparison has two runs but no parameters or scalar metrics.',
    'Desktop viewport checks do not establish actual iOS, assistive-technology, or full workflow parity.',
    'Page readiness and captured errors are checked; errors before initial Runs readiness are not observed, and this is not a complete network or console trace.',
  ],
};
let session;
await mkdir(out, { recursive: true });

async function command(method, path, body) {
  const response = await fetch(new URL(path, driver), {
    method,
    headers: body ? { 'content-type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(40000),
  });
  const data = await response.json();
  if (!response.ok || data.value?.error) {
    throw new Error(
      `${method} ${path}: ${data.value?.error || response.status}: ${data.value?.message || ''}`,
    );
  }
  return data.value;
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
async function click(selector) {
  const element = await find(selector);
  await command('POST', `/session/${session}/element/${element[elementKey]}/click`, {});
}
async function type(selector, text) {
  const element = await find(selector);
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
  const detail = await exercise();
  report.checks.push({ name, status: 'passed', detail: detail ?? null });
  console.log(`PASS ${name}`);
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
  const capabilities = { browserName, acceptInsecureCerts: false };
  if (browserName === 'firefox') {
    capabilities['moz:firefoxOptions'] = {
      ...(process.env.KFP_FIREFOX_BINARY ? { binary: process.env.KFP_FIREFOX_BINARY } : {}),
      args: ['-headless'],
      prefs: { 'app.update.auto': false, 'browser.shell.checkDefaultBrowser': false },
    };
  }
  const created = await command('POST', '/session', {
    capabilities: { alwaysMatch: capabilities },
  });
  session = created.sessionId;
  report.capabilities = created.capabilities;
  assert.equal(
    created.capabilities.browserVersion,
    expectedVersion,
    'must run the requested real version',
  );
  await command('POST', `/session/${session}/timeouts`, {
    implicit: 0,
    pageLoad: 30000,
    script: 10000,
  });
  await command('POST', `/session/${session}/window/rect`, { width: 1440, height: 900 });
  await command('POST', `/session/${session}/url`, { url: new URL('#/runs', base).href });
  await wait(
    () => document.querySelectorAll('[data-testid="run-name-link"]').length === 4,
    'four fixture runs',
  );
  report.environment = await execute(() => ({
    userAgent: navigator.userAgent,
    width: innerWidth,
    height: innerHeight,
    dpr: devicePixelRatio,
  }));
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
  await check('Runs loaded names and native name filter', async () => {
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
    await type('input[placeholder="Filter runs by name"]', 'xgboost');
    await wait(() => {
      const links = document.querySelectorAll('[data-testid="run-name-link"]');
      return links.length === 1 && links[0].textContent === 'v2-xgboost-ilbo';
    }, 'one matching run');
    await screenshot('runs-filtered');
    return { initialNames: names, filteredName: 'v2-xgboost-ilbo' };
  });
  await check('Run Details graph and task inspection', async () => {
    await click(`[data-testid="run-name-link"][data-run-id="${runId}"]`);
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
  await check('Pipelines cards and keyboard selection', async () => {
    await click('a[aria-label="Pipelines"]');
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
    await key(' ');
    await wait(
      (s) => document.querySelector(s)?.getAttribute('aria-checked') === 'false',
      'Space deselects pipeline',
      selector,
    );
    await screenshot('pipeline-cards');
  });
  await check(
    'Switch pointer and Space activation preserve unsubmitted feature state',
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
      await key(' ');
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
    await screenshot('comparison');
    return { selectedRuns: 2, parameters: 'empty fixture', scalarMetrics: 'empty fixture' };
  });
  await check('dark theme and narrow command-dialog keyboard containment', async () => {
    await click('select[aria-label="Theme"] option[value="dark"]');
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
    await command('POST', `/session/${session}/window/rect`, { width: 600, height: 900 });
    await click('button[aria-label="Search"]');
    await wait(
      () =>
        document.activeElement?.getAttribute('aria-label') ===
        'Search pipelines, experiments, and runs',
      'dialog autofocus',
    );
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
    await screenshot('command-dialog-narrow-dark');
    await key('\uE00C');
    await wait(
      () =>
        !document.querySelector('.kfp-page-dialog[role="dialog"]') &&
        document.activeElement?.getAttribute('aria-label') === 'Search',
      'Escape closes and returns focus',
    );
    assert.equal(await execute(() => document.documentElement.scrollWidth <= innerWidth), true);
    return colors;
  });
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
