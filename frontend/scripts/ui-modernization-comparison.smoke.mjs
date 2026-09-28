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

// Run after npm run build. Fixtures exercise real native-run comparison and viewers.
// Optional candidate-only timings: KFP_SCALING_SAMPLES=3 KFP_SCALING_OUTPUT_DIR=/tmp/...
// KFP_SOURCE_COMMIT=<built source SHA> node --test --test-name-pattern="candidate-only scaling" <this file>
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { before, after, test } from 'node:test';
import { chromium, firefox, webkit } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const alphaId = 'run/alpha space';
const betaId = 'run/beta space';
const comparisonHash = `#/compare?runlist=${encodeURIComponent(`${betaId},${alphaId}`)}`;
const runs = [
  {
    run_id: alphaId,
    display_name: 'Alpha run',
    runtime_config: { parameters: { epochs: 0, enabled: false, empty: '' } },
  },
  {
    run_id: betaId,
    display_name: 'Beta run',
    runtime_config: { parameters: { optimizer: 'adam' } },
  },
].map((run) => ({
  ...run,
  state: 'SUCCEEDED',
  storage_state: 'AVAILABLE',
  created_at: '2026-09-26T12:00:00Z',
  finished_at: '2026-09-26T12:01:00Z',
  pipeline_spec: {
    pipelineInfo: { name: 'comparison-fixture' },
    root: { dag: { tasks: {} } },
    schemaVersion: '2.1.0',
  },
}));
const confidenceMetrics = [
  { confidenceThreshold: 1, falsePositiveRate: 0, recall: 0 },
  { confidenceThreshold: 0.5, falsePositiveRate: 0.1, recall: 0.9 },
  { confidenceThreshold: 0, falsePositiveRate: 1, recall: 1 },
];
const confusionMatrix = {
  annotationSpecs: [{ displayName: 'cat' }, { displayName: 'dog' }],
  rows: [{ row: [2, 0] }, { row: [1, 3] }],
};
function tasksFor(runId) {
  const alpha = runId === alphaId;
  const prefix = alpha ? 'alpha' : 'beta';
  return [
    {
      task_id: `${prefix}/evaluate`,
      run_id: runId,
      name: 'evaluate',
      display_name: 'Evaluate',
      type: 'RUNTIME',
      state: 'SUCCEEDED',
      outputs: {
        artifacts: [
          {
            artifact_key: 'metrics',
            artifacts: [
              {
                artifact_id: `${prefix}-metric`,
                name: alpha ? 'accuracy' : 'loss',
                type: 'Metric',
                number_value: alpha ? 0 : 0.25,
              },
            ],
          },
          {
            artifact_key: 'classification',
            artifacts: Array.from({ length: alpha ? 110 : 1 }, (_, index) => ({
              artifact_id: `${prefix}-curve-${index}`,
              name: `Curve ${String(index).padStart(3, '0')}`,
              type: 'ClassificationMetric',
              metadata: { confidenceMetrics, ...(index === 0 ? { confusionMatrix } : {}) },
            })),
          },
          {
            artifact_key: 'report',
            artifacts: [
              {
                artifact_id: `${prefix}-html`,
                name: 'HTML report',
                type: 'HTML',
                namespace: 'team-a',
                uri: `s3://comparison/${prefix}.html`,
              },
            ],
          },
        ],
      },
    },
  ];
}
let browser;
before(async () => {
  const engineName = process.env.KFP_BROWSER || 'chromium';
  const engine = { chromium, firefox, webkit }[engineName];
  assert.ok(engine, `Unsupported KFP_BROWSER: ${engineName}`);
  browser = await engine.launch({
    channel: engineName === 'chromium' ? process.env.PLAYWRIGHT_CHANNEL || undefined : undefined,
  });
});
after(async () => browser?.close());

async function withFixture(exercise, { checkHeaderLayout = false, checkEmptyLayout = false } = {}) {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  await context.addInitScript(() => {
    // Sandboxed report frames intentionally have no storage access.
    if (window === window.top) localStorage.setItem('kfp.theme', 'light');
  });
  let releaseRuns;
  const runsReady =
    checkHeaderLayout || checkEmptyLayout
      ? new Promise((resolve) => {
          releaseRuns = resolve;
        })
      : Promise.resolve();
  if (checkHeaderLayout) {
    await context.addInitScript(() => {
      window.__kfpComparisonContentY = [];
      new MutationObserver(() => {
        const content = document.querySelector('.kfp-modern-page-content');
        if (!content) return;
        const top = content.getBoundingClientRect().top;
        const tops = window.__kfpComparisonContentY;
        if (tops.length < 100 && tops[tops.length - 1] !== top) tops.push(top);
      }).observe(document, { childList: true, subtree: true });
    });
  }
  const fixture = {
    requests: [],
    errors: [],
    runs: structuredClone(runs).map((run) =>
      checkEmptyLayout ? { ...run, runtime_config: { parameters: {} } } : run,
    ),
    runGate: undefined,
  };
  page.on('pageerror', (error) => fixture.errors.push(error.message));
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const query = Object.fromEntries(url.searchParams);
    const json = (value, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(value) });
    try {
      assert.equal(url.origin, origin, 'comparison must not fetch outside its local fixture');
      assert.ok(fixture.requests.length < 100, 'comparison traffic must remain bounded');
      assert.equal(request.method(), 'GET');
      fixture.requests.push({ path, query });
      if (path === '/' || path.startsWith('/static/')) {
        const name = path === '/' ? 'index.html' : path.slice(1);
        return route.fulfill({
          body: await readFile(new URL(name, build)),
          contentType: name.endsWith('.js')
            ? 'text/javascript'
            : name.endsWith('.css')
              ? 'text/css'
              : name.endsWith('.html')
                ? 'text/html'
                : 'application/octet-stream',
        });
      }
      if (path === '/apis/v2beta1/healthz') return json({ apiServerTagName: 'fixture' });
      if (path === '/apis/v2beta1/experiments') return json({ experiments: [] });
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      const runPath = /^\/apis\/v2beta1\/runs\/([^/]+)(\/tasks)?$/.exec(path);
      if (runPath) {
        const id = decodeURIComponent(runPath[1]);
        assert.ok([alphaId, betaId].includes(id), `unexpected run id ${id}`);
        if (runPath[2]) {
          assert.equal(query.page_size, '200');
          assert.equal(query.order_by, 'create_time asc');
          assert.equal(query.page_token, undefined);
          return json({ tasks: checkEmptyLayout ? [] : tasksFor(id) });
        }
        const snapshot = structuredClone(fixture.runs.find((run) => run.run_id === id));
        await runsReady;
        if (fixture.runGate) await fixture.runGate.promise;
        return json(snapshot);
      }
      if (path === '/artifacts/get') {
        assert.equal(query.source, 's3');
        assert.equal(query.namespace, 'team-a');
        assert.equal(query.bucket, 'comparison');
        assert.ok(['alpha.html', 'beta.html'].includes(query.key));
        return route.fulfill({
          contentType: 'text/html',
          body: `<h1>${query.key === 'alpha.html' ? 'Alpha' : 'Beta'} report</h1><p>Deterministic report content.</p>`,
        });
      }
      throw new Error(`Unexpected comparison request: ${path}`);
    } catch (error) {
      fixture.errors.push(error.message);
      await json({ message: error.message }, 500);
    }
  });
  try {
    fixture.navigationStartedAt = performance.now();
    await page.goto(`${origin}/${comparisonHash}`);
    let loadingContentTop;
    let loadingGeometry;
    if (checkEmptyLayout) {
      await page.getByText('Loading parameters…', { exact: true }).waitFor();
      await page.getByText('Loading scalar metrics artifacts…', { exact: true }).waitFor();
      await page.getByRole('table', { name: 'Runs', exact: true }).waitFor();
      await page.evaluate(() => document.fonts.ready);
      loadingGeometry = await comparisonGeometry(page);
      releaseRuns();
    }
    if (checkHeaderLayout) {
      await page.getByRole('heading', { name: 'Compare 2 runs', exact: true }).waitFor();
      loadingContentTop = await page
        .locator('.kfp-modern-page-content')
        .evaluate((content) => content.getBoundingClientRect().top);
      assert.equal(
        loadingContentTop,
        108,
        'comparison reserves the desktop breadcrumb header before run data resolves',
      );
      assert.deepEqual(await page.locator('.kfp-page-breadcrumbs a').allTextContents(), ['Runs']);
      releaseRuns();
    }
    if (checkEmptyLayout) {
      await emptyComparisonReady(page);
      sameComparisonGeometry(loadingGeometry, await comparisonGeometry(page), 'initial empty data');
    } else {
      await page.getByRole('table', { name: 'parameters comparison', exact: true }).waitFor();
    }
    if (checkHeaderLayout) {
      const readyTop = await page
        .locator('.kfp-modern-page-content')
        .evaluate((content) => content.getBoundingClientRect().top);
      assert.equal(
        readyTop,
        loadingContentTop,
        'comparison data arrival must not shift the content wrapper',
      );
      const observedTops = await page.evaluate(() => window.__kfpComparisonContentY);
      assert.ok(observedTops.length > 0);
      assert.ok(
        observedTops.every((top) => top === 108),
        `comparison mount must reserve the header from its first DOM commit: ${observedTops}`,
      );
    }
    await exercise(page, fixture);
    assert.deepEqual(fixture.errors, [], 'comparison bundle and HTTP contracts must remain valid');
  } catch (error) {
    if (fixture.errors.length) console.error('Fixture errors:', fixture.errors);
    throw error;
  } finally {
    releaseRuns?.();
    fixture.runGate?.release();
    await context.close();
  }
}
async function emptyComparisonReady(page) {
  await page
    .getByText('There are no parameters available on the selected runs.', { exact: true })
    .waitFor();
  await page
    .getByText('There are no scalar metrics artifacts available on the selected runs.', {
      exact: true,
    })
    .waitFor();
  await page.waitForFunction(() => {
    const table = document.querySelector('table[aria-label="Runs"]');
    return (
      table?.getAttribute('aria-busy') === 'false' &&
      table.querySelectorAll('[data-testid="run-name-link"]').length === 2
    );
  });
}
async function comparisonGeometry(page) {
  return page.evaluate(() => {
    const bounds = (element) => {
      const { x, y, width, height } = element.getBoundingClientRect();
      return { x, y, width, height };
    };
    const selectors = {
      overview: '.kfp-comparison-overview',
      footer: '.kfp-comparison-overview .kfp-runs-table-footer',
      metricTabs: '[role="tablist"][aria-label="Metric types"]',
    };
    const geometry = Object.fromEntries(
      Object.entries(selectors).map(([name, selector]) => [
        name,
        bounds(document.querySelector(selector)),
      ]),
    );
    for (const [index, element] of [
      ...document.querySelectorAll('.kfp-comparison-section-heading'),
    ].entries())
      geometry[`heading${index}`] = bounds(element);
    for (const [index, element] of [...document.querySelectorAll('.kfp-comparison-card')].entries())
      geometry[`card${index}`] = bounds(element);
    return geometry;
  });
}
function sameComparisonGeometry(before, after, phase) {
  assert.deepEqual(Object.keys(after), Object.keys(before));
  for (const name of Object.keys(before)) {
    for (const dimension of ['x', 'y', 'width', 'height']) {
      assert.ok(
        Math.abs(after[name][dimension] - before[name][dimension]) < 1,
        `${phase}: ${name}.${dimension} shifted from ${before[name][dimension]} to ${after[name][dimension]}`,
      );
    }
  }
}

async function screenshot(page, name) {
  if (!process.env.KFP_COMPARISON_SCREENSHOT_DIR) return;
  await mkdir(process.env.KFP_COMPARISON_SCREENSHOT_DIR, { recursive: true });
  await page.screenshot({
    path: join(process.env.KFP_COMPARISON_SCREENSHOT_DIR, `${name}.png`),
    animations: 'disabled',
  });
}
async function waitFocused(page, locator) {
  const handle = await locator.elementHandle();
  try {
    await page.waitForFunction((element) => document.activeElement === element, handle);
  } finally {
    await handle.dispose();
  }
}

async function renderedCurves(page, count) {
  await page.waitForFunction((expected) => {
    const paths = [...document.querySelectorAll('.kfp-roc-section .recharts-line-curve')];
    return paths.length === expected && paths.every((path) => !!path.getAttribute('d'));
  }, count);
}
async function matrixContent(container) {
  await container.locator('.kfp-confusion-cell').first().waitFor();
  assert.deepEqual(await container.locator('.kfp-confusion-cell').allTextContents(), [
    '0',
    '3',
    '2',
    '1',
  ]);
  assert.deepEqual(await container.locator('.kfp-confusion-xlabel').allTextContents(), [
    'cat',
    'dog',
  ]);
  assert.deepEqual(await container.locator('.kfp-confusion-ylabel').allTextContents(), [
    'dog',
    'cat',
  ]);
}

test('Comparison preserves selected URL order, encoded links, and absent versus zero, false, and empty values', async () => {
  await withFixture(
    async (page, fixture) => {
      const parameters = page.getByRole('table', { name: 'parameters comparison', exact: true });
      assert.deepEqual(await parameters.locator('thead a').allTextContents(), [
        'Beta run',
        'Alpha run',
      ]);
      assert.deepEqual(
        await parameters
          .locator('thead a')
          .evaluateAll((links) => links.map((link) => link.getAttribute('href'))),
        [
          `#/runs/details/${encodeURIComponent(betaId)}`,
          `#/runs/details/${encodeURIComponent(alphaId)}`,
        ],
      );
      const epochs = parameters.getByRole('row').filter({
        has: page.getByRole('rowheader', { name: 'epochs (values differ)', exact: true }),
      });
      assert.deepEqual(await epochs.getByRole('cell').allTextContents(), ['—', '0']);
      assert.equal(await epochs.getByLabel('Not provided', { exact: true }).count(), 1);
      assert.deepEqual(
        await parameters
          .getByRole('row')
          .filter({ hasText: 'enabled' })
          .getByRole('cell')
          .allTextContents(),
        ['—', 'false'],
      );
      assert.equal(await parameters.getByLabel('Empty string', { exact: true }).count(), 1);
      const metrics = page.getByRole('table', {
        name: 'scalar metrics artifacts comparison',
        exact: true,
      });
      await metrics.waitFor();
      assert.deepEqual(
        await metrics
          .getByRole('row')
          .filter({ hasText: 'Evaluate / accuracy' })
          .getByRole('cell')
          .allTextContents(),
        ['—', '0'],
      );
      assert.ok(fixture.requests.filter((item) => item.path.endsWith('/tasks')).length === 2);
      await screenshot(page, 'comparison-light');
      const alphaCheckbox = page.getByRole('checkbox', {
        name: 'Select run Alpha run',
        exact: true,
      });
      await alphaCheckbox.uncheck();
      await page.waitForFunction(
        () =>
          document.querySelectorAll('table[aria-label="parameters comparison"] thead a').length ===
          1,
      );
      assert.deepEqual(await parameters.locator('thead a').allTextContents(), ['Beta run']);
      await alphaCheckbox.check();
      await page.waitForFunction(
        () =>
          document.querySelectorAll('table[aria-label="parameters comparison"] thead a').length ===
          2,
      );
      assert.deepEqual(await parameters.locator('thead a').allTextContents(), [
        'Beta run',
        'Alpha run',
      ]);
      assert.equal(new URL(page.url()).hash, comparisonHash);
    },
    { checkHeaderLayout: true },
  );
});

test('Comparison keeps empty section geometry stable during slow initial data and refresh', async () => {
  await withFixture(
    async (page, fixture) => {
      const before = await comparisonGeometry(page);
      let release;
      fixture.runGate = {
        promise: new Promise((resolve) => {
          release = resolve;
        }),
        release: () => release(),
      };
      fixture.runs = fixture.runs.map((run) => ({
        ...run,
        display_name: `${run.display_name} updated`,
      }));
      const requested = page.waitForRequest((request) =>
        /\/apis\/v2beta1\/runs\/[^/]+$/.test(new URL(request.url()).pathname),
      );
      await page.getByRole('button', { name: 'Refresh', exact: true }).click();
      await requested;
      await page.getByText('Refreshing runs…', { exact: true }).waitFor();
      assert.equal(
        await page.getByRole('table', { name: 'Runs', exact: true }).getAttribute('aria-busy'),
        'true',
      );
      assert.equal(await page.getByText('Loading parameters…', { exact: true }).count(), 0);
      assert.equal(
        await page.getByText('Loading scalar metrics artifacts…', { exact: true }).count(),
        0,
      );
      assert.ok(
        await page
          .getByText('There are no parameters available on the selected runs.', { exact: true })
          .isVisible(),
      );
      assert.ok(
        await page
          .getByText('There are no scalar metrics artifacts available on the selected runs.', {
            exact: true,
          })
          .isVisible(),
      );
      sameComparisonGeometry(before, await comparisonGeometry(page), 'pending refresh');
      fixture.runGate.release();
      await emptyComparisonReady(page);
      const table = page.getByRole('table', { name: 'Runs', exact: true });
      await table.getByRole('link', { name: 'Alpha run updated', exact: true }).waitFor();
      await table.getByRole('link', { name: 'Beta run updated', exact: true }).waitFor();
      assert.equal(await table.getByRole('link', { name: 'Alpha run', exact: true }).count(), 0);
      assert.equal(await table.getByRole('link', { name: 'Beta run', exact: true }).count(), 0);
      assert.deepEqual(
        await table
          .locator('[data-testid="run-name-link"]')
          .evaluateAll((links) => links.map((link) => link.dataset.runId)),
        [betaId, alphaId],
      );
      assert.equal(await table.locator('[data-row-id][data-selected="true"]').count(), 2);
      sameComparisonGeometry(before, await comparisonGeometry(page), 'completed refresh');
      assert.equal(fixture.requests.filter((request) => request.path.endsWith('/tasks')).length, 4);
      assert.equal(new URL(page.url()).hash, comparisonHash);
    },
    { checkEmptyLayout: true },
  );
});

test('Comparison ROC pagination preserves off-page selection, provenance, search, and explicit deselection', async () => {
  await withFixture(async (page) => {
    await page.getByRole('tab', { name: 'Classification Metrics', exact: true }).click();
    const trigger = page.getByRole('combobox', { name: 'ROC curves', exact: true });
    await trigger.waitFor();
    assert.match(await trigger.textContent(), /3 curves selected/);
    const provenance = page.getByRole('list', {
      name: 'Selected ROC curve provenance',
      exact: true,
    });
    const original = await provenance.getByRole('listitem').allTextContents();
    assert.equal(original.length, 3);
    await renderedCurves(page, 3);
    await trigger.click();
    const options = page.getByRole('listbox', { name: 'ROC curves', exact: true });
    await options.waitFor();
    assert.equal(await options.getByRole('option').count(), 100);
    await page.keyboard.press('Escape');
    await waitFocused(page, trigger);
    await page.getByRole('button', { name: 'Next ROC curves', exact: true }).click();
    assert.match(await trigger.textContent(), /3 curves selected/);
    await trigger.click();
    await options.waitFor();
    assert.equal(await options.getByRole('option').count(), 11);
    const last = options.getByRole('option').last();
    const lastName = (await last.textContent()).trim();
    await last.click();
    assert.match(await trigger.textContent(), /4 curves selected/);
    await renderedCurves(page, 4);
    await page.keyboard.press('Escape');
    const selected = await provenance.getByRole('listitem').allTextContents();
    assert.ok(
      original.every((name) => selected.includes(name)),
      'first-page selection must survive adding a second-page curve',
    );
    assert.ok(selected.some((name) => name.trim() === lastName));
    const search = page.getByRole('textbox', { name: 'Search ROC curves', exact: true });
    await search.fill(lastName);
    await trigger.click();
    const selectedOption = options.getByRole('option', { name: lastName, exact: true });
    assert.equal(await selectedOption.getAttribute('aria-selected'), 'true');
    await selectedOption.click();
    await page.keyboard.press('Escape');
    assert.match(await trigger.textContent(), /3 curves selected/);
    assert.deepEqual(await provenance.getByRole('listitem').allTextContents(), original);
    await renderedCurves(page, 3);
    await search.fill('no matching curve');
    await page.getByText('No ROC curves match this search.', { exact: true }).waitFor();
    assert.match(await trigger.textContent(), /3 curves selected/);
    await search.fill('');
    await page.getByRole('button', { name: 'Expand ROC chart', exact: true }).click();
    await page.getByRole('button', { name: 'Compact ROC chart', exact: true }).waitFor();
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await screenshot(page, 'comparison-roc-dark');
  });
});

test('Comparison selected viewers preserve provenance and fullscreen keyboard focus and theme', async () => {
  await withFixture(async (page, fixture) => {
    await page.getByRole('tab', { name: 'Classification Metrics', exact: true }).click();
    await page.getByRole('tab', { name: 'Confusion matrix', exact: true }).click();
    const alphaLabel = 'Alpha run / Evaluate / Curve 000';
    const betaLabel = 'Beta run / Evaluate / Curve 000';
    await page
      .getByRole('combobox', { name: 'First comparison artifact', exact: true })
      .selectOption({ label: alphaLabel });
    await page
      .getByRole('combobox', { name: 'Second comparison artifact', exact: true })
      .selectOption({ label: betaLabel });
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    const expand = page.getByRole('button', { name: `Expand ${alphaLabel}`, exact: true });
    await matrixContent(page.locator('.kfp-plot-card').filter({ has: expand }));
    await expand.click();
    const dialog = page.getByRole('dialog');
    await dialog.waitFor();
    assert.equal(
      await dialog.evaluate((element) => element.closest('.kfp-theme')?.classList.contains('dark')),
      true,
    );
    await matrixContent(dialog);
    const close = dialog.getByRole('button', { name: 'Close', exact: true });
    await waitFocused(page, close);
    await page.keyboard.press('Tab');
    await waitFocused(page, close);
    await page.keyboard.press('Shift+Tab');
    await waitFocused(page, close);
    await screenshot(page, 'comparison-fullscreen-dark');
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'hidden' });
    await waitFocused(page, expand);
    await page.getByRole('tab', { name: 'HTML', exact: true }).click();
    assert.equal(fixture.requests.filter((item) => item.path === '/artifacts/get').length, 0);
    await page
      .getByRole('combobox', { name: 'First comparison artifact', exact: true })
      .selectOption({ label: 'Alpha run / Evaluate / HTML report' });
    await page
      .frameLocator('iframe')
      .getByRole('heading', { name: 'Alpha report', exact: true })
      .waitFor();
    assert.equal(fixture.requests.filter((item) => item.path === '/artifacts/get').length, 1);
    assert.equal(
      fixture.requests.filter((item) => item.path === '/artifacts/get')[0].query.key,
      'alpha.html',
    );
  });
});

// These repeated laboratory samples are automation readiness timings, not field INP or a baseline comparison.
if (process.env.KFP_SCALING_SAMPLES) {
  test('candidate-only scaling: populated two-run comparison with 111 classifications', async () => {
    const count = Number(process.env.KFP_SCALING_SAMPLES);
    assert.ok(Number.isInteger(count) && count >= 3 && count <= 20, 'use 3–20 scaling samples');
    assert.match(
      process.env.KFP_SOURCE_COMMIT || '',
      /^[a-f0-9]{40}$/,
      'identify the built source',
    );
    assert.ok(process.env.KFP_SCALING_OUTPUT_DIR, 'provide a directory for retained raw samples');
    const data = { runs, tasks: runs.map(({ run_id }) => tasksFor(run_id)) };
    const classificationCount = data.tasks
      .flat()
      .flatMap((task) => task.outputs.artifacts)
      .filter((output) => output.artifact_key === 'classification')
      .reduce((count, output) => count + output.artifacts.length, 0);
    assert.equal(classificationCount, 111);
    const samples = [];
    const assetPaths = new Set();
    for (let sample = 1; sample <= count; sample++) {
      await withFixture(async (page, fixture) => {
        const parameters = page.getByRole('table', { name: 'parameters comparison', exact: true });
        const metrics = page.getByRole('table', {
          name: 'scalar metrics artifacts comparison',
          exact: true,
        });
        await metrics.waitFor();
        assert.deepEqual(await parameters.locator('thead a').allTextContents(), [
          'Beta run',
          'Alpha run',
        ]);
        assert.deepEqual(
          await metrics
            .getByRole('row')
            .filter({ hasText: 'Evaluate / accuracy' })
            .getByRole('cell')
            .allTextContents(),
          ['—', '0'],
        );
        assert.equal(fixture.requests.filter(({ path }) => path.endsWith('/tasks')).length, 2);
        await page.evaluate(() =>
          document.fonts.ready.then(
            () =>
              new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
          ),
        );
        const contentReadyMs = performance.now() - fixture.navigationStartedAt;

        const chartStarted = performance.now();
        await page.getByRole('tab', { name: 'Classification Metrics', exact: true }).click();
        const trigger = page.getByRole('combobox', { name: 'ROC curves', exact: true });
        await renderedCurves(page, 3);
        const provenance = page.getByRole('list', {
          name: 'Selected ROC curve provenance',
          exact: true,
        });
        const original = await provenance.getByRole('listitem').allTextContents();
        assert.equal(original.length, 3);
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        const chartReadyMs = performance.now() - chartStarted;
        await trigger.click();
        const options = page.getByRole('listbox', { name: 'ROC curves', exact: true });
        await options.waitFor();
        assert.equal(await options.getByRole('option').count(), 100);
        await page.keyboard.press('Escape');
        await page.getByRole('button', { name: 'Next ROC curves', exact: true }).click();
        await trigger.click();
        await options.waitFor();
        assert.equal(await options.getByRole('option').count(), 11);
        const last = options.getByRole('option').last();
        const lastName = (await last.textContent()).trim();
        const selectStarted = performance.now();
        await last.click();
        await renderedCurves(page, 4);
        const selected = await provenance.getByRole('listitem').allTextContents();
        assert.ok(original.every((name) => selected.includes(name)));
        assert.ok(selected.some((name) => name.trim() === lastName));
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        const addCurveReadyMs = performance.now() - selectStarted;
        await page.keyboard.press('Escape');
        const chart = page.locator('.kfp-roc-section .recharts-wrapper');
        const compactHeight = await chart.evaluate(
          (element) => element.getBoundingClientRect().height,
        );
        const expandStarted = performance.now();
        await page.getByRole('button', { name: 'Expand ROC chart', exact: true }).click();
        await page.getByRole('button', { name: 'Compact ROC chart', exact: true }).waitFor();
        await page.waitForFunction(
          (before) =>
            document.querySelector('.kfp-roc-section .recharts-wrapper').getBoundingClientRect()
              .height > before,
          compactHeight,
        );
        await renderedCurves(page, 4);
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        const expandChartReadyMs = performance.now() - expandStarted;
        samples.push({ sample, contentReadyMs, chartReadyMs, addCurveReadyMs, expandChartReadyMs });
        for (const { path } of fixture.requests)
          if (path.startsWith('/static/')) assetPaths.add(path);
      });
    }
    const assets = await Promise.all(
      [...assetPaths].sort().map(async (path) => {
        const bytes = await readFile(new URL(path.slice(1), build));
        return {
          path,
          bytes: bytes.length,
          sha256: createHash('sha256').update(bytes).digest('hex'),
        };
      }),
    );
    const report = {
      applicationSource: process.env.KFP_SOURCE_COMMIT,
      harnessSha256: createHash('sha256')
        .update(await readFile(new URL(import.meta.url)))
        .digest('hex'),
      indexSha256: createHash('sha256')
        .update(await readFile(new URL('index.html', build)))
        .digest('hex'),
      fixtureSha256: createHash('sha256').update(JSON.stringify(data)).digest('hex'),
      browser: {
        engine: process.env.KFP_BROWSER || 'chromium',
        version: browser.version(),
        channel: process.env.PLAYWRIGHT_CHANNEL || null,
      },
      runtime: { node: process.version, platform: process.platform, arch: process.arch },
      settings: {
        viewport: { width: 1440, height: 1000 },
        deviceScaleFactor: 1,
        locale: 'en-US',
        timezoneId: 'UTC',
        theme: 'light',
        reducedMotion: 'reduce',
        cpuThrottle: 'none',
        networkThrottle: 'none',
      },
      method:
        'Fresh context per sample in one warm browser process; local route fixtures and file/OS caches stay warm. Automation timings include Playwright overhead. Initial readiness requires populated parameter/scalar tables and fonts; chart and selection readiness require actual curve paths/provenance; expansion requires increased rendered chart height. Each endpoint also waits two frames. No field INP, backend load, or baseline speedup claim.',
      fixture: {
        runs: 2,
        classificationArtifacts: 111,
        initialVisibleCurves: 3,
        selectedVisibleCurves: 4,
        pageSizes: [100, 11],
      },
      assets,
      samples,
    };
    await mkdir(process.env.KFP_SCALING_OUTPUT_DIR, { recursive: true });
    await writeFile(
      join(
        process.env.KFP_SCALING_OUTPUT_DIR,
        `comparison-${process.env.KFP_BROWSER || 'chromium'}.json`,
      ),
      `${JSON.stringify(report, null, 2)}\n`,
    );
  });
}
