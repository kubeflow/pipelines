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

import assert from 'node:assert/strict';
import { spawn, execFileSync } from 'node:child_process';
import { once } from 'node:events';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { cpus, release, totalmem } from 'node:os';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { gzipSync } from 'node:zlib';
import { dump } from 'js-yaml';
import {
  cls,
  compareTiming,
  compareEditorTiming,
  inventory,
  sha256,
  verifyBuild,
  validateSample,
  validateStaticTransfers,
  cleanupWithEvidence,
  hostedReadinessProtocol,
  isExpectedLegacyWorkerError,
} from './performance-evidence.mjs';

assert.equal(process.env.CI, 'true');
assert.equal(process.env.GITHUB_ACTIONS, 'true');
assert.equal(
  process.env.RUNNER_ENVIRONMENT,
  'github-hosted',
  'Performance browsers run only on disposable hosted CI',
);
const { chromium } = await import('playwright');
const frontend = fileURLToPath(new URL('../', import.meta.url));
const builds = resolve(process.env.KFP_PERFORMANCE_BUILDS);
const output = resolve(process.env.KFP_PERFORMANCE_OUTPUT || 'performance-results');
const budgets = JSON.parse(await readFile(new URL('./performance-budgets.json', import.meta.url)));
const protocolPath = new URL(
  './fixtures/performance-protocol.json',
  import.meta.url,
);
const protocolBytes = await readFile(protocolPath);
const protocol = hostedReadinessProtocol(JSON.parse(protocolBytes));
const hostedProtocolBytes = `${JSON.stringify(protocol, null, 2)}\n`;
// Compile trusted, versioned repository instrumentation as functions: Playwright treats
// string evaluate arguments as expressions, not callable functions with arguments.
const instrumentation = Object.fromEntries(
  Object.entries(protocol.scripts)
    .filter(([name]) => name !== 'preNavigation')
    .map(([name, source]) => [name, Function(`return (${source})`)()]),
);
const sources = {
  legacy: '02cbc725ac9ddcd950f4400d8355dd78bfcd6c57',
  checkpoint: 'e79f8d423e6b118e5df94815ee2f36f68570a9a9',
  previous: 'dc1b3cfbaba17444408c9a59eb8ab9354b2dfd79',
  candidate: process.env.GITHUB_SHA,
};
const expectedEditor = dump(
  JSON.parse(
    await readFile(
      new URL(
        '../mock-backend/data/v2/pipeline/lightweight_python_functions_v2_pipeline.json',
        import.meta.url,
      ),
    ),
  ),
);
const runId = 'e0115ac1-0479-4194-a22d-01e65e09a32b';
const nodeSelector =
  '[data-testid="DagCanvas"] .react-flow__node[data-id="task.chicago-taxi-trips-dataset"]';
const routes = {
  runs: '#/runs',
  'run-details': `#/runs/details/${runId}`,
  compare: `#/compare?runlist=mock-run-0,${runId}`,
};
const settings = {
  viewport: { width: 1280, height: 720 },
  deviceScaleFactor: 1,
  colorScheme: 'light',
  locale: 'en-US',
  timezoneId: 'UTC',
  cpuThrottlingRate: 4,
  network: {
    offline: false,
    latency: 150,
    downloadThroughput: 1600000 / 8,
    uploadThroughput: 750000 / 8,
  },
  observationSettleMs: 500,
};
const report = {
  status: 'running',
  acceptance:
    budgets.status === 'accepted' ? 'accepted-not-yet-evaluated' : 'pending-maintainer-acceptance',
  startedAt: new Date().toISOString(),
  sources,
  run: {
    id: process.env.GITHUB_RUN_ID,
    attempt: process.env.GITHUB_RUN_ATTEMPT,
    testedSource: process.env.GITHUB_SHA,
    pullRequestHead: process.env.KFP_PERFORMANCE_PR_HEAD || null,
  },
  budgets,
  settings,
  historicalProtocolSha256: sha256(protocolBytes),
  protocolSha256: sha256(hostedProtocolBytes),
  harnessSha256: sha256(await readFile(new URL(import.meta.url))),
  runtime: {
    node: process.version,
    platform: process.platform,
    arch: process.arch,
    os: release(),
    cpu: cpus()[0]?.model,
    memoryBytes: totalmem(),
  },
  builds: {},
  comparisonOrders: [
    ['legacy', 'previous', 'candidate'],
    ['candidate', 'previous', 'legacy'],
    ['previous', 'candidate', 'legacy'],
    ['legacy', 'candidate', 'previous'],
    ['candidate', 'legacy', 'previous'],
    ['previous', 'legacy', 'candidate'],
  ],
  samples: [],
  comparisons: {},
  scaling: {},
  limitations: [
    'Laboratory fixture measurements, not deployed backend performance or field percentiles.',
    'Fresh contexts; browser process and runner file/OS caches remain warm.',
    'Public static JS/CSS use the production gzip-sidecar handler for every build; immutable baselines without sidecars retain identity transfer. Actual encoding, length and no-store headers are checked against build inventories.',
    'Numeric network profile defines a new hosted protocol, not an exact repeat of historical Fast 4G.',
    'First editor open uses separate blocking engineering gates: 4 s median display and 5 s median worker readiness. This accepts a cold lazy-load tradeoff, not parity with the eager legacy editor. Both builds measure the complete read-only model, fonts and two frames; candidate worker initialization is separately required and measured.',
    'The immutable legacy build omits worker-yaml.js. Its actual HTTP404 is retained; there is no equivalent fully worker-ready legacy timing.',
  ],
};
await mkdir(output, { recursive: true });
await writeFile(resolve(output, 'hosted-protocol.json'), hostedProtocolBytes);
const save = () =>
  writeFile(resolve(output, 'performance.json'), `${JSON.stringify(report, null, 2)}\n`);
const servers = [];
let browser;
async function startServer(variant, port) {
  const child = spawn(
    process.execPath,
    ['--import', 'tsx', 'scripts/ui-modernization-native-server.ts'],
    {
      cwd: frontend,
      env: {
        ...process.env,
        KFP_BROWSER_FLOOR_PORT: String(port),
        KFP_BROWSER_BUILD_DIR: resolve(builds, variant, 'build'),
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    },
  );
  servers.push(child);
  let log = '';
  child.stdout.on('data', (data) => {
    log += data;
  });
  child.stderr.on('data', (data) => {
    log += data;
  });
  child.on('close', () =>
    writeFile(resolve(output, `${variant}-fixture.log`), log).catch(console.error),
  );
  const origin = `http://127.0.0.1:${port}`;
  const deadline = Date.now() + 30000;
  while (Date.now() < deadline) {
    assert.equal(child.exitCode, null, `Fixture exited: ${log}`);
    try {
      if ((await fetch(`${origin}/__qualification`, { signal: AbortSignal.timeout(1000) })).ok)
        return origin;
    } catch {}
    await new Promise((done) => setTimeout(done, 100));
  }
  throw new Error(`Fixture startup failed: ${log}`);
}
async function stopTrace(session, path) {
  const complete = new Promise((done) => session.once('Tracing.tracingComplete', done));
  await session.send('Tracing.end');
  const { stream } = await Promise.race([
    complete,
    new Promise((_, reject) =>
      setTimeout(() => reject(new Error('Trace completion timed out')), 30000).unref(),
    ),
  ]);
  const chunks = [];
  try {
    for (;;) {
      const data = await session.send('IO.read', { handle: stream });
      chunks.push(Buffer.from(data.data, data.base64Encoded ? 'base64' : 'utf8'));
      if (data.eof) break;
    }
  } finally {
    await session.send('IO.close', { handle: stream });
  }
  const bytes = gzipSync(Buffer.concat(chunks));
  await writeFile(resolve(output, path), bytes);
  return { path, bytes: bytes.length, sha256: sha256(bytes) };
}
async function geometry(page) {
  return page.evaluate(() => {
    const rect = (element) => {
      if (!element) throw new Error('Missing stable control');
      const { x, y, width, height } = element.getBoundingClientRect();
      return { x, y, width, height };
    };
    return {
      footer: rect(document.querySelector('.kfp-runs-table-footer')),
      filter: rect(document.querySelector('input[type="search"]')),
      sidebar: rect(document.querySelector('.kfp-shell-footer')),
      columns: [...document.querySelectorAll('table[aria-label="Runs"] th')].map(rect),
    };
  });
}
function stableControls(before, after) {
  assert.equal(before.columns.length, after.columns.length);
  for (const [name, first, last] of [
    ...['footer', 'filter', 'sidebar'].map((key) => [key, before[key], after[key]]),
    ...before.columns.map((value, i) => [`column-${i}`, value, after.columns[i]]),
  ])
    for (const dimension of ['x', 'y', 'width', 'height'])
      assert.ok(
        Math.abs(first[dimension] - last[dimension]) <= budgets.controlMovementMaximumPx,
        `${name}.${dimension} moved`,
      );
}
async function sample(variant, origin, kind, trial) {
  const record = { variant, kind, trial, status: 'running' };
  report.samples.push(record);
  const context = await browser.newContext({
    viewport: settings.viewport,
    deviceScaleFactor: 1,
    colorScheme: 'light',
    locale: 'en-US',
    timezoneId: 'UTC',
  });
  await context.addInitScript({ content: protocol.scripts.preNavigation });
  await context.addInitScript(() => {
    window.__kfpRowShifts = [];
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries())
        window.__kfpRowShifts.push({
          time: entry.startTime,
          value: entry.value,
          recentInput: entry.hadRecentInput,
          sources: entry.sources.map(({ node, previousRect, currentRect }) => ({
            rowId: node
              ?.closest?.('tr')
              ?.querySelector('[data-testid="run-name-link"]')
              ?.getAttribute('data-run-id'),
            text: node?.textContent?.slice(0, 180),
            previous: previousRect.toJSON(),
            current: currentRect.toJSON(),
          })),
        });
    }).observe({ type: 'layout-shift', buffered: true });
  });
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', (error) => errors.push(error.message));
  record.staticTransfers = [];
  page.on('response', (response) => {
    const path = new URL(response.url()).pathname;
    if (!/^\/static\/[\w.-]+\.(?:js|css)$/.test(path)) return;
    const headers = response.headers();
    record.staticTransfers.push({
      path,
      status: response.status(),
      encoding: headers['content-encoding'] || 'identity',
      bytes: Number(headers['content-length']),
      vary: headers.vary || '',
      cacheControl: headers['cache-control'] || '',
    });
  });
  page.on('request', (request) => {
    if (!request.url().startsWith(origin + '/') && !request.url().startsWith('blob:'))
      errors.push(`Unexpected external request: ${request.url()}`);
  });
  const session = await context.newCDPSession(page);
  let tracing = false;
  try {
    await session.send('Emulation.setCPUThrottlingRate', { rate: settings.cpuThrottlingRate });
    await session.send('Network.enable');
    await session.send('Network.emulateNetworkConditions', settings.network);
    await session.send('Tracing.start', {
      categories:
        'devtools.timeline,blink.user_timing,loading,disabled-by-default-devtools.timeline',
      transferMode: 'ReturnAsStream',
    });
    tracing = true;
    if (kind === 'editor') {
      const id = '8fbe3bd6-a01f-11e8-98d0-529269fb1460';
      await page.goto(`${origin}/#/pipelines/details/${id}/version/${id}`);
      await page.locator('[data-testid="DagCanvas"] .react-flow__node').first().waitFor();
      // The retained legacy MD2Tabs used buttons; the migrated component exposes tabs.
      const tab = page.getByRole(variant === 'legacy' ? 'button' : 'tab', {
        name: 'Pipeline Spec',
        exact: true,
      });
      await tab.evaluate((element) =>
        element.addEventListener('click', () => performance.mark('editor-start'), {
          once: true,
          capture: true,
        }),
      );
      const workerResponse = page
        .waitForResponse((response) =>
          /\/worker-yaml[^/]*\.js$/.test(new URL(response.url()).pathname),
        )
        .then(
          (response) => ({ response }),
          (error) => ({ error }),
        );
      await tab.click();
      await page.waitForFunction((expected) => {
        const editor = document.querySelector('[data-testid="spec-ir"] .ace_editor')?.env?.editor;
        return editor?.getReadOnly() && editor.getValue() === expected;
      }, expectedEditor);
      record.editor = await page.evaluate(async (expected) => {
        const editor = document.querySelector('[data-testid="spec-ir"] .ace_editor').env.editor;
        await document.fonts.ready;
        await new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done)));
        if (!editor.getReadOnly() || editor.getValue() !== expected)
          throw new Error('Editor model changed before confirming paint');
        performance.measure('editor-model-ready', 'editor-start');
        return {
          duration: performance.getEntriesByName('editor-model-ready')[0].duration,
          endpoint: 'complete-read-only-model-fonts-two-frames',
          model: editor.getValue(),
          readOnly: editor.getReadOnly(),
        };
      }, expectedEditor);
      record.editor.expectedModelSha256 = sha256(expectedEditor);
      record.editor.modelSha256 = sha256(record.editor.model);
      delete record.editor.model;
      const workerResult = await workerResponse;
      if (workerResult.error) throw workerResult.error;
      const response = workerResult.response;
      record.editor.workerResponseStatus = response.status();
      record.editor.workerPath = new URL(response.url()).pathname;
      assert.equal(await response.finished(), null);
      if (variant === 'legacy') {
        // The immutable baseline omitted Ace's default worker asset. Record its real
        // failure; only the common model endpoint is comparable to this baseline.
        assert.equal(record.editor.workerPath, '/worker-yaml.js');
        assert.equal(response.status(), 404);
        assert.ok(!report.builds.legacy.assets.some((asset) => asset.path === 'worker-yaml.js'));
        record.editor.workerStatus = 'unavailable-missing-baseline-asset';
      } else {
        assert.equal(response.status(), 200);
        await page.waitForFunction(
          () =>
            !!document.querySelector('[data-testid="spec-ir"] .ace_editor')?.env?.editor?.session
              .$worker,
        );
        const worker = await page.evaluate(async (expected) => {
          const editor = document.querySelector('[data-testid="spec-ir"] .ace_editor').env.editor;
          const workerValue = await new Promise((resolve, reject) => {
            const timeout = setTimeout(
              () => reject(new Error('YAML worker readiness timed out')),
              30000,
            );
            editor.session.$worker.call('getValue', [], (value) => {
              clearTimeout(timeout);
              resolve(value);
            });
          });
          if (workerValue !== expected || editor.getValue() !== expected || !editor.getReadOnly())
            throw new Error('YAML worker or editor model differs from fixture');
          performance.measure('editor-worker-ready', 'editor-start');
          return {
            workerValue,
            workerReadyMs: performance.getEntriesByName('editor-worker-ready')[0].duration,
          };
        }, expectedEditor);
        record.editor.workerModelSha256 = sha256(worker.workerValue);
        record.editor.workerReadyMs = worker.workerReadyMs;
        record.editor.workerStatus = 'ready';
      }
    } else {
      await page.goto(origin + '/' + (routes[kind] || routes.runs));
      record.readiness = await page.evaluate(
        instrumentation.perfReadinessFunction,
        routes[kind] ? kind : 'runs',
      );
      if (kind === 'filter') {
        const matching = page.locator(`a[data-testid="run-name-link"][href*="${runId}"]`);
        const handle = await matching.elementHandle();
        record.before = variant === 'candidate' ? await geometry(page) : null;
        await page.evaluate(instrumentation.perfFilterInstrument);
        await page
          .locator('input[placeholder="Filter runs by name"], input#tableFilterBox')
          .fill('xgboost');
        record.filter = await page.evaluate(instrumentation.perfFilterRead);
        if (variant === 'candidate')
          assert.equal(
            await handle.evaluate(
              (element) =>
                element.isConnected &&
                document.querySelector('a[data-testid="run-name-link"]') === element,
            ),
            true,
            'Filtering must retain the matching row identity',
          );
        record.filterStart = await page.evaluate(
          () => performance.getEntriesByName('filter-start')[0].startTime,
        );
        if (variant === 'candidate') {
          record.after = await geometry(page);
          stableControls(record.before, record.after);
        }
      } else if (kind === 'navigation') {
        await page.evaluate(instrumentation.perfNavigationInstrument, 'run-open');
        await page.locator(`a[data-testid="run-name-link"][href*="${runId}"]`).click();
        record.runOpen = await page.evaluate(instrumentation.perfNavigationRead, 'run-open');
        await page.evaluate(instrumentation.perfNavigationInstrument, 'task-open');
        await page.locator(nodeSelector).click();
        record.taskOpen = await page.evaluate(instrumentation.perfNavigationRead, 'task-open');
      }
    }
    await page.waitForTimeout(settings.observationSettleMs);
    record.observations = await page.evaluate(() => ({
      observedAtMs: performance.now(),
      readiness: window.__kfpContentReady,
      paint: window.__kfpPaint,
      rowShifts: window.__kfpRowShifts,
      resources: performance.getEntriesByType('resource').map((entry) => entry.toJSON()),
    }));
    validateSample(record);
    validateStaticTransfers(record.staticTransfers, report.builds[variant].assets);
    record.cls = cls(record.observations.paint.shifts, kind === 'filter' ? record.filterStart : 0);
    if (variant === 'candidate' && kind === 'filter') {
      const shifts = record.observations.rowShifts.filter(
        (entry) => !entry.recentInput && entry.time >= record.filterStart,
      );
      assert.ok(
        shifts.every(
          (entry) =>
            entry.sources.length &&
            entry.sources.every(
              (source) =>
                source.rowId === runId &&
                source.previous.x === source.current.x &&
                source.previous.width === source.current.width &&
                source.previous.height === source.current.height &&
                source.current.y <= source.previous.y,
            ),
        ),
        'Only matching-row upward compaction may contribute filter CLS',
      );
    }
    record.errors = errors;
    const expectedWorkerError = (error) => isExpectedLegacyWorkerError(record, error, origin);
    record.expectedBaselineWorkerErrors = errors.filter(expectedWorkerError);
    assert.deepEqual(
      errors.filter((error) => !expectedWorkerError(error)),
      [],
    );
    record.status = 'measured';
  } catch (error) {
    record.status = 'failed';
    record.error = error.stack || String(error);
    record.failedReadiness = await page
      .evaluate(() => ({
        readiness: window.__kfpContentReady,
        paint: window.__kfpPaint,
        capturedAtMs: performance.now(),
      }))
      .catch((failure) => ({ captureError: String(failure) }));
    await page
      .screenshot({ path: resolve(output, `${variant}-${kind}-${trial}-failure.png`) })
      .catch(() => {});
    throw error;
  } finally {
    const failures = await cleanupWithEvidence(
      [
        [
          'trace',
          async () => {
            if (tracing)
              record.trace = await stopTrace(session, `${variant}-${kind}-${trial}.trace.json.gz`);
          },
        ],
        ['context', () => context.close()],
      ],
      (failure) => {
        record.status = 'failed';
        (record.cleanupErrors ??= []).push(failure);
      },
      save,
      40000,
    );
    if (failures.length)
      throw new Error('Sample cleanup failed; retained failures are in performance.json');
  }
}
try {
  for (const [variant, source] of Object.entries(sources)) {
    const provenance = JSON.parse(await readFile(resolve(builds, variant, 'provenance.json')));
    report.builds[variant] = await verifyBuild(
      resolve(builds, variant, 'build'),
      provenance,
      source,
    );
  }
  const fixtureFiles = [
    'scripts/ui-modernization-native-server.ts',
    'server/static-assets.ts',
    'scripts/ui-modernization-native-transactions.ts',
    'mock-backend/mock-api-app.ts',
    'mock-backend/mock-api-middleware.ts',
    'mock-backend/fixed-data.ts',
  ];
  report.fixtureFiles = await Promise.all(
    fixtureFiles.map(async (path) => ({
      path,
      sha256: sha256(await readFile(resolve(frontend, path))),
    })),
  );
  report.fixtureData = await inventory(resolve(frontend, 'mock-backend/data'));
  const origins = {
    legacy: await startServer('legacy', 4181),
    candidate: await startServer('candidate', 4182),
    previous: await startServer('previous', 4183),
  };
  browser = await chromium.launch();
  report.browserVersion = browser.version();
  report.playwrightVersion = JSON.parse(
    await readFile(resolve(frontend, 'node_modules/playwright/package.json')),
  ).version;
  for (let trial = 1; trial <= budgets.samples; trial++)
    for (const kind of ['runs', 'run-details', 'compare', 'filter', 'navigation', 'editor'])
      for (const variant of report.comparisonOrders[(trial - 1) % report.comparisonOrders.length])
        await sample(variant, origins[variant], kind, trial);
  const closeFailures = await cleanupWithEvidence(
    [['matched-browser', () => browser.close()]],
    (failure) => {
      (report.cleanupErrors ??= []).push(failure);
    },
    save,
  );
  if (closeFailures.length)
    throw new Error('Matched browser did not close before scaling; see retained evidence');
  browser = null;
  for (const [variant, origin] of Object.entries(origins)) {
    const fixture = await (await fetch(`${origin}/__qualification`)).json();
    assert.deepEqual(fixture.mutations, []);
    assert.deepEqual(
      fixture.missingAssets,
      variant === 'legacy' ? Array(budgets.samples).fill('/worker-yaml.js') : [],
      'Only the independently observed missing legacy YAML worker is expected',
    );
    report[`${variant}Fixture`] = fixture;
  }
  const extract = (variant, kind, read) =>
    report.samples.filter((sample) => sample.variant === variant && sample.kind === kind).map(read);
  for (const kind of ['runs', 'run-details', 'compare'])
    report.comparisons[kind] = compareTiming(
      extract('legacy', kind, (sample) => sample.observations.readiness.contentReadyMs),
      extract('candidate', kind, (sample) => sample.observations.readiness.contentReadyMs),
      budgets,
    );
  for (const [name, kind, read] of [
    [
      'filter',
      'filter',
      (sample) => sample.filter.measures.find((entry) => entry.name === 'filter-results').duration,
    ],
    ['run-open', 'navigation', (sample) => sample.runOpen.measure[0].duration],
    ['task-open', 'navigation', (sample) => sample.taskOpen.measure[0].duration],
  ])
    report.comparisons[name] = compareTiming(
      extract('legacy', kind, read),
      extract('candidate', kind, read),
      budgets,
    );
  for (const kind of ['runs', 'run-details', 'compare'])
    report.comparisons[`previous-${kind}`] = compareTiming(
      extract('previous', kind, (sample) => sample.observations.readiness.contentReadyMs),
      extract('candidate', kind, (sample) => sample.observations.readiness.contentReadyMs),
      budgets,
    );
  for (const [name, kind, read] of [
    [
      'filter',
      'filter',
      (sample) => sample.filter.measures.find((entry) => entry.name === 'filter-results').duration,
    ],
    ['run-open', 'navigation', (sample) => sample.runOpen.measure[0].duration],
    ['task-open', 'navigation', (sample) => sample.taskOpen.measure[0].duration],
    ['editor-display', 'editor', (sample) => sample.editor.duration],
    ['editor-worker-ready', 'editor', (sample) => sample.editor.workerReadyMs],
  ])
    report.comparisons[`previous-${name}`] = compareTiming(
      extract('previous', kind, read),
      extract('candidate', kind, read),
      budgets,
    );
  report.previousEditorWorkerReadyMs = extract(
    'previous',
    'editor',
    (sample) => sample.editor.workerReadyMs,
  );
  report.comparisons.previousEntryGzip = {
    previousBytes: report.builds.previous.entryGzipBytes,
    candidateBytes: report.builds.candidate.entryGzipBytes,
    passesProposedBudget:
      report.builds.candidate.entryGzipBytes <= report.builds.previous.entryGzipBytes,
  };
  report.editorSamples = Object.fromEntries(
    ['legacy', 'previous', 'candidate'].map((variant) => [
      variant,
      extract(variant, 'editor', (sample) => sample.editor.duration),
    ]),
  );
  report.editorEndpoint = 'complete-read-only-model-fonts-two-frames';
  report.candidateEditorWorkerReadyMs = extract(
    'candidate',
    'editor',
    (sample) => sample.editor.workerReadyMs,
  );
  Object.assign(
    report.comparisons,
    compareEditorTiming(
      report.editorSamples.candidate,
      report.candidateEditorWorkerReadyMs,
      budgets,
    ),
  );
  report.comparisons.entryGzip = {
    ratio: report.builds.candidate.entryGzipBytes / report.builds.legacy.entryGzipBytes,
    limit: budgets.entryGzipRatio,
    passesProposedBudget:
      report.builds.candidate.entryGzipBytes <=
      report.builds.legacy.entryGzipBytes * budgets.entryGzipRatio,
  };
  for (const kind of ['runs', 'run-details', 'compare', 'filter']) {
    const values = extract('candidate', kind, (sample) => sample.cls);
    const limit = kind === 'filter' ? budgets.filterClsMaximum : budgets.loadClsMaximum;
    report.comparisons[`${kind}-cls`] = {
      values,
      limit,
      passesProposedBudget: values.every((value) => value <= limit),
    };
  }
  for (const workload of ['graph', 'comparison']) {
    const variants =
      workload === 'graph' ? ['checkpoint', 'candidate'] : ['candidate', 'checkpoint'];
    for (const variant of variants) {
      const directory = resolve(output, 'scaling', variant);
      await mkdir(directory, { recursive: true });
      const script = `scripts/ui-modernization-${workload}.smoke.mjs`;
      try {
        const log = execFileSync(
          process.execPath,
          ['--test', '--test-name-pattern=candidate-only scaling', script],
          {
            cwd: frontend,
            encoding: 'utf8',
            timeout: 300000,
            maxBuffer: 8 * 1024 * 1024,
            env: {
              ...process.env,
              KFP_PERFORMANCE_BUILD_DIR: resolve(builds, variant, 'build'),
              KFP_SOURCE_COMMIT: sources[variant],
              KFP_EXPECTED_BROWSER_VERSION: report.browserVersion,
              KFP_SCALING_SAMPLES: String(budgets.samples),
              KFP_SCALING_OUTPUT_DIR: directory,
            },
          },
        );
        await writeFile(resolve(directory, `${workload}.tap`), log);
      } catch (error) {
        await writeFile(
          resolve(directory, `${workload}.tap`),
          `${error.stdout || ''}\n${error.stderr || ''}`,
        );
        throw error;
      }
      report.scaling[`${workload}-${variant}`] = JSON.parse(
        await readFile(resolve(directory, `${workload}-chromium.json`)),
      );
    }
    const previous = report.scaling[`${workload}-checkpoint`];
    const candidate = report.scaling[`${workload}-candidate`];
    for (const [variant, result] of [
      ['checkpoint', previous],
      ['candidate', candidate],
    ]) {
      assert.equal(result.applicationSource, sources[variant]);
      assert.equal(result.browser.version, report.browserVersion);
      for (const asset of result.assets) {
        const expected = report.builds[variant].assets.find(
          (entry) => entry.path === asset.path.replace(/^\//, ''),
        );
        assert.ok(expected, 'Scaling requested an unknown build asset');
        assert.equal(asset.sha256, expected.sha256);
      }
      assert.equal(
        result.indexSha256,
        report.builds[variant].assets.find((asset) => asset.path === 'index.html').sha256,
      );
    }
    assert.equal(previous.fixtureSha256, candidate.fixtureSha256);
    assert.equal(previous.harnessSha256, candidate.harnessSha256);
    for (const metric of Object.keys(previous.samples[0]).filter((key) => key.endsWith('Ms')))
      report.comparisons[`${workload}-${metric}`] = compareTiming(
        previous.samples.map((sample) => sample[metric]),
        candidate.samples.map((sample) => sample[metric]),
        budgets,
      );
  }
  report.proposedBudgetsPass = Object.values(report.comparisons).every(
    (result) => result.passesProposedBudget,
  );
  report.status = 'measured';
  if (budgets.status === 'accepted') {
    report.acceptance = report.proposedBudgetsPass
      ? 'accepted-budgets-pass'
      : 'accepted-budgets-fail';
    assert.equal(report.proposedBudgetsPass, true, 'Accepted performance budgets exceeded');
  }
} catch (error) {
  report.status = 'failed';
  report.error = error.stack || String(error);
  process.exitCode = 1;
} finally {
  await cleanupWithEvidence(
    [
      [
        'browser',
        async () => {
          if (browser) await browser.close();
        },
      ],
      ...servers.map((server, index) => [
        `fixture-${index}`,
        async () => {
          if (server.exitCode !== null || server.signalCode !== null) return;
          const exited = once(server, 'exit');
          server.kill('SIGTERM');
          try {
            await Promise.race([exited, new Promise((done) => setTimeout(done, 5000).unref())]);
          } finally {
            if (server.exitCode === null && server.signalCode === null) {
              server.kill('SIGKILL');
              await exited;
            }
          }
        },
      ]),
    ],
    (failure) => {
      report.status = 'failed';
      process.exitCode = 1;
      (report.cleanupErrors ??= []).push(failure);
    },
    async () => {
      report.finishedAt = new Date().toISOString();
      await save();
    },
  );
}
