/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Run after npm run build: node --test scripts/ui-modernization-run-details.smoke.mjs
// Local HTTP fixtures only. This guards presentation migration, not live cluster authorization.
import assert from 'node:assert/strict';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import { chromium, firefox, webkit } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const runId = 'browser-run';
const createdAt = '2026-09-26T12:00:00.000Z';
const finishedAt = '2026-09-26T12:01:00.000Z';
const baseSpec = JSON.parse(
  await readFile(
    new URL(
      '../mock-backend/data/v2/pipeline/lightweight_python_functions_v2_pipeline.json',
      import.meta.url,
    ),
    'utf8',
  ),
);
const pipelineSpec = {
  ...baseSpec,
  pipelineInfo: { name: 'browser-nested-pipeline' },
  components: { ...baseSpec.components, 'comp-group': baseSpec.root },
  root: {
    ...baseSpec.root,
    dag: {
      tasks: {
        prepare: { ...baseSpec.root.dag.tasks.preprocess, taskInfo: { name: 'prepare' } },
        group: {
          taskInfo: { name: 'group' },
          componentRef: { name: 'comp-group' },
          dependentTasks: ['prepare'],
        },
      },
    },
  },
};
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

function task(name, parent, type = 'RUNTIME', state = 'SUCCEEDED') {
  return {
    task_id: `task-${name}`,
    run_id: runId,
    name,
    display_name: name,
    parent_task_id: parent && `task-${parent}`,
    type,
    state,
    create_time: createdAt,
    end_time: state === 'RUNNING' ? undefined : finishedAt,
    state_history: [{ state, update_time: finishedAt }],
  };
}
function gate() {
  let release;
  const promise = new Promise((resolve) => {
    release = resolve;
  });
  return { promise, release };
}

async function withFixture(options, exercise) {
  const state = options.state || 'SUCCEEDED';
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  await page.clock.install({ time: new Date('2026-09-26T12:02:00.000Z') });
  const fixture = {
    run: {
      run_id: runId,
      display_name: 'Nested browser run',
      description: 'Run Details browser fixture',
      experiment_id: 'experiment-a',
      pipeline_spec: pipelineSpec,
      storage_state: 'AVAILABLE',
      state,
      created_at: createdAt,
      scheduled_at: createdAt,
      finished_at: state === 'RUNNING' ? undefined : finishedAt,
      state_history: [
        { state: 'RUNNING', update_time: createdAt },
        { state, update_time: finishedAt },
      ],
      runtime_config: { parameters: { message: 'fixture input' } },
    },
    tasks: [
      task('root', undefined, 'ROOT', state),
      task('prepare', 'root'),
      task('group', 'root', 'DAG', state),
      task('preprocess', 'group'),
      task('train', 'group', 'RUNTIME', state),
    ],
    logs: new Map([
      ['train-pod', 'executor attempt one\ntraining complete'],
      ['prepare-driver', 'driver initialization output'],
    ]),
    requests: [],
    errors: [],
    failRunReads: !!options.failRunReads,
    retryCount: 0,
    runGate: undefined,
    logGates: new Map(),
  };
  fixture.tasks[1].pods = [{ name: 'prepare-driver', type: 'DRIVER' }];
  fixture.tasks[3].pods = [{ name: 'preprocess-pod', type: 'EXECUTOR' }];
  fixture.tasks[3].outputs = {
    artifacts: [
      {
        artifact_key: 'executor-logs',
        artifacts: [
          {
            artifact_id: 'logs-artifact',
            name: 'executor-logs',
            type: 'Artifact',
            uri: 's3://fixture-bucket/preprocess.log',
          },
        ],
      },
    ],
  };
  fixture.tasks[4].pods = [{ name: 'train-pod', type: 'EXECUTOR' }];
  if (state === 'FAILED')
    fixture.tasks[4].state_history[0].error = { code: 13, message: 'Training fixture failed' };
  page.on('pageerror', (error) => fixture.errors.push(error.message));
  await context.addInitScript(({ namespace }) => {
    localStorage.setItem('kfp.theme', 'light');
    if (namespace)
      window.centraldashboard = {
        CentralDashboardEventHandler: {
          init(callback) {
            const handler = {};
            callback(handler);
            handler.onNamespaceSelected(namespace);
          },
        },
      };
  }, options);
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/pipeline(?=\/)/, '');
    const json = (value, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(value) });
    try {
      assert.equal(url.origin, origin, 'fixture must not contact external services');
      assert.ok(fixture.requests.length < 250, 'requests must remain bounded');
      fixture.requests.push({
        path,
        pathname: url.pathname,
        method: request.method(),
        query: Object.fromEntries(url.searchParams),
      });
      if (path === '/embed')
        return route.fulfill({
          contentType: 'text/html',
          body: '<!doctype html><title>Dashboard fixture</title><iframe title="Pipelines" src="/pipeline/#/runs/details/browser-run?task=task-train" style="width:100%;height:850px;border:0"></iframe>',
        });
      if (path === '/' || path.startsWith('/static/')) {
        const name = path === '/' ? 'index.html' : path.slice(1);
        let body = await readFile(new URL(name, build));
        if (path === '/')
          body = body
            .toString()
            .replace(
              /window\.KFP_FLAGS\.DEPLOYMENT\s*=\s*null;?/,
              `window.KFP_FLAGS.DEPLOYMENT=${JSON.stringify(options.namespace ? 'KUBEFLOW' : null)};`,
            )
            .replace(
              /window\.KFP_FLAGS\.HIDE_SIDENAV\s*=\s*null;?/,
              `window.KFP_FLAGS.HIDE_SIDENAV=${!!options.namespace};`,
            );
        return route.fulfill({
          body,
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
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      if (path === `/apis/v2beta1/runs/${runId}`) {
        assert.equal(request.method(), 'GET');
        if (fixture.failRunReads)
          return json({ code: 14, message: 'Fixture run temporarily unavailable' }, 503);
        const snapshot = structuredClone(fixture.run);
        if (fixture.runGate) {
          const pending = fixture.runGate;
          await pending.promise;
          fixture.runGate = undefined;
        }
        return json(snapshot);
      }
      if (path === `/apis/v2beta1/runs/${runId}/tasks`) {
        assert.equal(request.method(), 'GET');
        assert.equal(url.searchParams.get('page_size'), '200');
        assert.equal(url.searchParams.get('order_by'), 'create_time asc');
        return json({ tasks: fixture.tasks });
      }
      if (path === `/apis/v2beta1/runs/${runId}:retry`) {
        assert.equal(request.method(), 'POST');
        assert.equal(request.postData(), null);
        assert.equal(++fixture.retryCount, 1, 'retry must happen exactly once');
        fixture.run.state = 'RUNNING';
        fixture.run.finished_at = undefined;
        fixture.run.state_history.push({
          state: 'RUNNING',
          update_time: '2026-09-26T12:02:00.000Z',
        });
        fixture.tasks = fixture.tasks.map((task) =>
          task.name === 'train'
            ? {
                ...task,
                state: 'RUNNING',
                end_time: undefined,
                pods: [...task.pods, { name: 'retry-pod', type: 'EXECUTOR' }],
              }
            : task,
        );
        return json({});
      }
      if (path === '/apis/v2beta1/experiments/experiment-a')
        return json({
          experiment_id: 'experiment-a',
          display_name: 'Browser experiment',
          namespace: options.namespace || 'team-a',
        });
      if (path === '/k8s/pod/logs') {
        assert.equal(request.method(), 'GET');
        assert.equal(url.searchParams.get('runid'), runId);
        assert.equal(url.searchParams.get('podnamespace'), options.namespace || 'team-a');
        assert.equal(url.searchParams.get('createdat'), '2026-09-26');
        const pod = url.searchParams.get('podname');
        if (pod === 'preprocess-pod')
          return route.fulfill({ status: 404, body: 'Fixture pod was collected' });
        assert.ok(fixture.logs.has(pod), `unexpected pod ${pod}`);
        const text = fixture.logs.get(pod);
        if (fixture.logGates.has(pod)) await fixture.logGates.get(pod).promise;
        return route.fulfill({ contentType: 'text/plain', body: text });
      }
      if (path === '/artifacts/get') {
        assert.equal(request.method(), 'GET');
        assert.equal(url.searchParams.get('source'), 's3');
        assert.equal(url.searchParams.get('bucket'), 'fixture-bucket');
        assert.equal(url.searchParams.get('key'), 'preprocess.log');
        assert.equal(url.searchParams.get('namespace'), options.namespace || 'team-a');
        return route.fulfill({
          contentType: 'text/plain',
          body: 'artifact executor output\npreprocessing complete',
        });
      }
      throw new Error(`Unexpected fixture request: ${request.method()} ${path}`);
    } catch (error) {
      fixture.errors.push(error.message);
      await json({ message: error.message }, 500);
    }
  });
  try {
    await exercise(page, fixture);
    assert.deepEqual(
      fixture.errors,
      [],
      'bundle and request contracts must have no unexpected errors',
    );
  } catch (error) {
    if (fixture.errors.length) console.error('Fixture errors:', fixture.errors);
    // Keep timeout evidence small and failure-only; never alter graph timing or
    // suppress browser errors to make a visibility assertion pass.
    const graphDiagnostic = await page
      .evaluate(() => {
        const describe = (element) => {
          const style = getComputedStyle(element);
          const rect = element.getBoundingClientRect();
          return {
            id: element.getAttribute('data-id'),
            className: String(element.className),
            inlineStyle: element.getAttribute('style'),
            visibility: style.visibility,
            display: style.display,
            width: style.width,
            height: style.height,
            offsetWidth: element.offsetWidth,
            offsetHeight: element.offsetHeight,
            bounds: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
          };
        };
        const nodes = [...document.querySelectorAll('.react-flow__node')];
        return {
          url: location.href,
          nodeCount: nodes.length,
          edgeCount: document.querySelectorAll('.react-flow__edge').length,
          nodes: nodes.slice(0, 20).map(describe),
          containers: [
            ...document.querySelectorAll(
              '.kfp-graph-workspace,.kfp-graph-canvas,.react-flow__renderer,.react-flow__viewport',
            ),
          ]
            .slice(0, 8)
            .map(describe),
        };
      })
      .catch((diagnosticError) => ({ unavailable: diagnosticError.message }));
    console.error('Run Details failure diagnostics:', JSON.stringify(graphDiagnostic));
    throw error;
  } finally {
    fixture.runGate?.release();
    for (const pending of fixture.logGates.values()) pending.release();
    await context.close();
  }
}

const node = (page, name) => page.locator(`.react-flow__node[data-id="task.${name}"]`);
const tab = (page, name) =>
  page.getByRole('button', { name, exact: true }).or(page.getByRole('tab', { name, exact: true }));
async function selected(page, name) {
  await page.locator(`.react-flow__node[data-id="task.${name}"].selected`).waitFor();
}
async function openTask(page, taskName = 'train', extra = '') {
  await page.goto(`${origin}/#/runs/details/${runId}?task=task-${taskName}${extra}`);
  await selected(page, taskName);
}
async function changeTask(page, taskName) {
  await page.evaluate((name) => {
    const url = new URL(location.href);
    const query = new URLSearchParams(url.hash.split('?')[1]);
    query.set('task', `task-${name}`);
    location.hash = `${url.hash.split('?')[0]}?${query}`;
  }, taskName);
  await selected(page, taskName);
}
async function logs(page, text) {
  await tab(page, 'Logs').click();
  await page.getByTestId('logs-view-window').getByText(text, { exact: true }).waitFor();
}
async function screenshot(page, name) {
  if (!process.env.KFP_RUN_DETAILS_SCREENSHOT_DIR) return;
  await mkdir(process.env.KFP_RUN_DETAILS_SCREENSHOT_DIR, { recursive: true });
  await page.screenshot({
    path: join(process.env.KFP_RUN_DETAILS_SCREENSHOT_DIR, `${name}.png`),
    animations: 'disabled',
  });
}
async function graph(page) {
  await page.getByTestId('DagCanvas').waitFor();
  return page.evaluate(async () => {
    await document.fonts.ready;
    function measure() {
      return {
        nodes: [...document.querySelectorAll('.react-flow__node')]
          .map((element) => {
            const matrix = new DOMMatrixReadOnly(element.style.transform);
            return {
              id: element.dataset.id,
              x: matrix.m41,
              y: matrix.m42,
              width: element.offsetWidth,
              height: element.offsetHeight,
              visible: getComputedStyle(element).visibility === 'visible',
            };
          })
          .sort((a, b) => a.id.localeCompare(b.id)),
        edges: [...document.querySelectorAll('.react-flow__edge')]
          .map((element) => ({
            id: element.dataset.id,
            path: element.querySelector('.react-flow__edge-path')?.getAttribute('d') || '',
            visible: getComputedStyle(element).visibility === 'visible',
          }))
          .sort((a, b) => a.id.localeCompare(b.id)),
      };
    }
    let previous,
      stable = 0;
    for (let frame = 0; frame < 90; frame++) {
      await new Promise(requestAnimationFrame);
      const current = measure();
      const key = JSON.stringify(current);
      stable = key === previous ? stable + 1 : 0;
      previous = key;
      if (
        current.nodes.length >= 2 &&
        current.nodes.every((node) => node.visible && node.width > 0 && node.height > 0) &&
        current.edges.length > 0 &&
        current.edges.every((edge) => edge.visible && edge.path) &&
        stable >= 3
      )
        return current;
    }
    throw new Error('graph node/edge geometry did not settle');
  });
}
function sameGraph(before, after) {
  assert.deepEqual(
    after.nodes.map((node) => node.id),
    before.nodes.map((node) => node.id),
  );
  assert.deepEqual(
    after.edges.map((edge) => edge.id),
    before.edges.map((edge) => edge.id),
  );
  for (let i = 0; i < before.nodes.length; i++)
    for (const field of ['x', 'y', 'width', 'height'])
      assert.ok(
        Math.abs(before.nodes[i][field] - after.nodes[i][field]) < 0.05,
        `${before.nodes[i].id} ${field} changed`,
      );
  for (let i = 0; i < before.edges.length; i++) {
    const numbers = (value) => value.match(/-?\d+(?:\.\d+)?(?:e[-+]?\d+)?/gi).map(Number);
    const a = numbers(before.edges[i].path),
      b = numbers(after.edges[i].path);
    assert.equal(a.length, b.length);
    assert.ok(
      a.every((value, index) => Math.abs(value - b[index]) < 0.05),
      `edge ${before.edges[i].id} changed`,
    );
  }
}

test('Run Details preserves nested task links, history, copied URLs, and graph geometry', async () => {
  await withFixture({}, async (page, fixture) => {
    const pendingRun = gate();
    fixture.runGate = pendingRun;
    const requested = page.waitForRequest(
      (request) => new URL(request.url()).pathname === `/apis/v2beta1/runs/${runId}`,
    );
    await page.goto(`${origin}/#/runs/details/${runId}?task=task-train&view=graph`);
    await requested;
    const content = page.locator('.kfp-modern-page-content');
    await content.waitFor();
    const pendingY = await content.evaluate((element) => element.getBoundingClientRect().y);
    pendingRun.release();
    await selected(page, 'train');
    await page.getByTestId('page-title').getByText('Nested browser run', { exact: true }).waitFor();
    const loadedY = await content.evaluate((element) => element.getBoundingClientRect().y);
    assert.ok(
      Math.abs(loadedY - pendingY) < 1,
      `Run Details content shifted from ${pendingY} to ${loadedY}`,
    );
    await tab(page, 'Task Details').click();
    await page.getByText('task-train', { exact: true }).waitFor();
    const original = await graph(page);
    await screenshot(page, 'run-details-task');
    if (process.env.KFP_RUN_DETAILS_SCREENSHOT_DIR) {
      await writeFile(
        join(process.env.KFP_RUN_DETAILS_SCREENSHOT_DIR, 'run-details-geometry.json'),
        `${JSON.stringify(original, null, 2)}\n`,
      );
    }
    await changeTask(page, 'preprocess');
    await tab(page, 'Task Details').click();
    await page.getByText('task-preprocess', { exact: true }).waitFor();
    sameGraph(original, await graph(page));
    await page.goBack();
    await selected(page, 'train');
    await page.goForward();
    await selected(page, 'preprocess');
    await page.reload();
    await selected(page, 'preprocess');
    sameGraph(original, await graph(page));
    await page.getByRole('button', { name: 'close', exact: true }).click();
    await page.waitForURL((url) => url.hash === `#/runs/details/${runId}?view=graph`);
    assert.equal(await page.locator('.react-flow__node.selected').count(), 0);
    sameGraph(original, await graph(page));
    // Closing the inspector replaces controlled node objects. Actual measured
    // dimensions and edge handles must survive each selection/close rerender.
    for (const taskName of ['train', 'preprocess']) {
      await page.getByRole('button', { name: 'Fit View', exact: true }).click();
      await node(page, taskName).getByRole('button').click();
      await selected(page, taskName);
      sameGraph(original, await graph(page));
      const inspector = page.getByRole('dialog');
      await inspector.getByRole('button', { name: 'close', exact: true }).click();
      await inspector.waitFor({ state: 'hidden' });
      await page.waitForURL((url) => url.hash === `#/runs/details/${runId}?view=graph`);
      assert.equal(await page.locator('.react-flow__node.selected').count(), 0);
      sameGraph(original, await graph(page));
    }
    await screenshot(page, 'run-details-graph');
  });
});

test('Run Details distinguishes executor, artifact, and driver log sources', async () => {
  await withFixture({}, async (page, fixture) => {
    await openTask(page);
    await logs(page, 'executor attempt one');
    await changeTask(page, 'preprocess');
    await logs(page, 'artifact executor output');
    assert.equal(
      await page
        .getByTestId('logs-view-window')
        .getByText('executor attempt one', { exact: true })
        .count(),
      0,
    );
    await changeTask(page, 'prepare');
    await logs(page, 'driver initialization output');
    await page
      .getByText(
        'Showing driver initialization logs. These are not component executor output logs.',
        { exact: true },
      )
      .waitFor();
    assert.deepEqual(
      fixture.requests
        .filter((request) => request.path === '/k8s/pod/logs')
        .map((request) => request.query.podname),
      ['train-pod', 'preprocess-pod', 'prepare-driver'],
    );
    await screenshot(page, 'run-details-driver-logs');
  });
});

async function tick(page, path) {
  const response = page.waitForResponse((response) =>
    new URL(response.url()).pathname.endsWith(path),
  );
  await page.clock.fastForward(10000);
  await response;
}

test('Run Details recovers a failed initial read and preserves its graph during refresh failures', async () => {
  await withFixture({ failRunReads: true, state: 'RUNNING' }, async (page, fixture) => {
    await page.goto(`${origin}/#/runs/details/${runId}?task=task-train`);
    await page
      .getByRole('alert')
      .filter({ hasText: 'Unable to load run details. Refresh this page to retry.' })
      .waitFor({ timeout: 15000 });
    fixture.failRunReads = false;
    await page.reload();
    await selected(page, 'train');
    assert.equal(
      await page
        .getByText('Unable to load run details. Refresh this page to retry.', { exact: true })
        .count(),
      0,
    );
    const original = await graph(page);
    fixture.failRunReads = true;
    await tick(page, `/apis/v2beta1/runs/${runId}`);
    const warning = page.getByText(
      'Unable to refresh this run. The last known run state is still shown. Refresh the page to try again.',
      { exact: true },
    );
    await warning.waitFor();
    await selected(page, 'train');
    sameGraph(original, await graph(page));
    fixture.failRunReads = false;
    await tick(page, `/apis/v2beta1/runs/${runId}`);
    await warning.waitFor({ state: 'hidden' });
    sameGraph(original, await graph(page));
  });
});

test('Run Details retries once, preserves prior logs while loading, and rejects stale task output', async () => {
  await withFixture({ state: 'FAILED' }, async (page, fixture) => {
    await openTask(page);
    await tab(page, 'Task Details').click();
    await page.getByText('Training fixture failed', { exact: true }).first().waitFor();
    await logs(page, 'executor attempt one');
    await screenshot(page, 'run-details-failed-logs');
    const runPending = gate(),
      logsPending = gate();
    fixture.runGate = runPending;
    fixture.logGates.set('retry-pod', logsPending);
    fixture.logs.set('retry-pod', 'retry attempt output');
    const retryRead = page.waitForRequest(
      (request) => new URL(request.url()).pathname === `/apis/v2beta1/runs/${runId}`,
    );
    await page.getByRole('button', { name: 'Retry', exact: true }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Retry', exact: true }).click();
    await retryRead;
    assert.equal(fixture.retryCount, 1);
    await page
      .getByTestId('logs-view-window')
      .getByText('executor attempt one', { exact: true })
      .waitFor();
    const retryLogs = page.waitForRequest(
      (request) => new URL(request.url()).searchParams.get('podname') === 'retry-pod',
    );
    runPending.release();
    await retryLogs;
    await page
      .getByTestId('logs-view-window')
      .getByText('executor attempt one', { exact: true })
      .waitFor();
    await changeTask(page, 'preprocess');
    await logs(page, 'artifact executor output');
    const completed = page.waitForResponse(
      (response) => new URL(response.url()).searchParams.get('podname') === 'retry-pod',
    );
    logsPending.release();
    await completed;
    await page.evaluate(
      () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
    );
    assert.equal(
      await page
        .getByTestId('logs-view-window')
        .getByText('retry attempt output', { exact: true })
        .count(),
      0,
    );
    await page
      .getByTestId('logs-view-window')
      .getByText('artifact executor output', { exact: true })
      .waitFor();
    await changeTask(page, 'train');
    await logs(page, 'retry attempt output');
    assert.equal(await page.getByRole('button', { name: 'Retry', exact: true }).isDisabled(), true);
    assert.equal(fixture.retryCount, 1);
  });
});

test('Run Details log output scrolls horizontally and pauses then resumes following new lines', async () => {
  await withFixture({ state: 'RUNNING' }, async (page, fixture) => {
    const lines = Array.from(
      { length: 650 },
      (_, index) => `entry-${String(index).padStart(4, '0')} ${'x'.repeat(140)}`,
    );
    fixture.logs.set('train-pod', lines.join('\n'));
    await openTask(page);
    await logs(page, lines.at(-1));
    const viewer = page.locator('#logViewer');
    await page.waitForFunction(() => {
      const viewer = document.querySelector('#logViewer');
      return viewer.scrollHeight - viewer.scrollTop - viewer.clientHeight <= 20;
    });
    assert.equal(await viewer.evaluate((node) => node.scrollWidth > node.clientWidth), true);
    await viewer.evaluate(
      (node) =>
        new Promise((resolve) => {
          node.addEventListener('scroll', () => requestAnimationFrame(resolve), { once: true });
          node.scrollTop = 0;
        }),
    );
    lines.push('appended while reading earlier output');
    fixture.logs.set('train-pod', lines.join('\n'));
    await tick(page, '/k8s/pod/logs');
    await page.waitForFunction((count) => {
      const viewer = document.querySelector('#logViewer');
      return viewer.scrollHeight >= count * 15 && viewer.scrollTop <= 20;
    }, lines.length);
    await viewer.evaluate(
      (node) =>
        new Promise((resolve) => {
          node.addEventListener('scroll', () => requestAnimationFrame(resolve), { once: true });
          node.scrollTop = node.scrollHeight;
        }),
    );
    lines.push('appended while following the tail');
    fixture.logs.set('train-pod', lines.join('\n'));
    await tick(page, '/k8s/pod/logs');
    await page.waitForFunction((count) => {
      const viewer = document.querySelector('#logViewer');
      return (
        viewer.scrollHeight >= count * 15 &&
        viewer.scrollHeight - viewer.scrollTop - viewer.clientHeight <= 20
      );
    }, lines.length);
    await screenshot(page, 'run-details-long-logs');
  });
});

test('Embedded Run Details preserves its prefix and namespace for a deep-linked task log', async () => {
  await withFixture({ namespace: 'team-b' }, async (page, fixture) => {
    await page.goto(`${origin}/embed`);
    const app = page.frames().find((frame) => frame.url().includes('/pipeline/'));
    assert.ok(app);
    await selected(app, 'train');
    await logs(app, 'executor attempt one');
    const request = fixture.requests.find((request) => request.path === '/k8s/pod/logs');
    assert.equal(request.pathname, '/pipeline/k8s/pod/logs');
    assert.equal(request.query.podnamespace, 'team-b');
    await tab(app, 'Task Details').click();
    await app.getByText('task-train', { exact: true }).waitFor();
    await page.reload();
    const reloaded = page.frames().find((frame) => frame.url().includes('/pipeline/'));
    assert.ok(reloaded);
    await selected(reloaded, 'train');
  });
});

// The failed-task entry point is new presentation behavior, beyond the six captured baseline cases.
test('Run Details View logs opens the real failed runtime task from the run summary', async () => {
  await withFixture({ state: 'FAILED' }, async (page, fixture) => {
    await page.goto(`${origin}/#/runs/details/${runId}`);
    await node(page, 'group').waitFor();
    await page.getByRole('button', { name: 'View logs', exact: true }).click();
    await selected(page, 'train');
    assert.equal(
      new URLSearchParams(new URL(page.url()).hash.split('?')[1]).get('task'),
      'task-train',
    );
    await page
      .getByTestId('logs-view-window')
      .getByText('executor attempt one', { exact: true })
      .waitFor();
    assert.equal(fixture.retryCount, 0);
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    assert.equal(
      await page
        .getByRole('dialog')
        .evaluate((element) => element.closest('.kfp-theme')?.classList.contains('dark')),
      true,
    );
    await screenshot(page, 'run-details-dark-failed-logs');
  });
});

// Focus behavior is part of the new responsive inspector, beyond the legacy baseline.
test('Run Details inspector restores desktop focus and traps keyboard focus on narrow screens', async () => {
  await withFixture({}, async (page) => {
    await page.goto(`${origin}/#/runs/details/${runId}`);
    const group = node(page, 'group');
    await group.waitFor();
    const trigger = group.locator('button').first();
    await page.evaluate(() => {
      document.addEventListener('focusin', (event) => {
        if (event.target.closest?.('.react-flow__node[data-id="task.group"]'))
          window.kfpFixtureGraphFocusTarget = event.target;
      });
      // Safari pointer activation focuses the nearest focusable ancestor. Record
      // that real target before React opens the inspector; keyboard activation
      // still records and returns to the native node button below.
      document.addEventListener(
        'click',
        (event) => {
          if (event.target.closest?.('.react-flow__node[data-id="task.group"]'))
            window.kfpFixturePointerFocusTarget = document.activeElement;
        },
        true,
      );
    });
    await trigger.focus();
    await trigger.click();
    const inspector = page.getByRole('dialog');
    await inspector.waitFor();
    const desktopReturnTarget = await page.evaluateHandle(
      () => window.kfpFixturePointerFocusTarget,
    );
    assert.equal(await inspector.getAttribute('aria-modal'), null);
    const close = inspector.getByRole('button', { name: 'close', exact: true });
    const resize = inspector.getByRole('separator', { name: 'Resize node details', exact: true });
    const assertWidth = (expected) =>
      page.waitForFunction((width) => {
        const panel = document.querySelector('.kfp-inspector-panel');
        const handle = document.querySelector(
          '[role="separator"][aria-label="Resize node details"]',
        );
        return (
          panel &&
          handle &&
          Number(handle.getAttribute('aria-valuenow')) === width &&
          Math.abs(panel.getBoundingClientRect().width - width) < 1
        );
      }, expected);
    await assertWidth(380);
    await resize.focus();
    await page.keyboard.press('ArrowLeft');
    await assertWidth(400);
    await page.keyboard.press('ArrowRight');
    await assertWidth(380);
    await page.keyboard.press('End');
    await assertWidth(Number(await resize.getAttribute('aria-valuemax')));
    await page.keyboard.press('Home');
    await assertWidth(Number(await resize.getAttribute('aria-valuemin')));
    await page.keyboard.press('ArrowLeft');
    await assertWidth(320);
    const handleBox = await resize.boundingBox();
    assert.ok(handleBox);
    const handleX = handleBox.x + handleBox.width / 2;
    const handleY = handleBox.y + handleBox.height / 2;
    await page.mouse.move(handleX, handleY);
    await page.mouse.down();
    await page.mouse.move(handleX - 40, handleY, { steps: 4 });
    await page.mouse.up();
    await assertWidth(360);
    await tab(page, 'Task Details').click();
    await assertWidth(360);
    await resize.focus();
    await page.setViewportSize({ width: 600, height: 900 });
    await page.waitForFunction(
      () =>
        !document.querySelector('[role="separator"][aria-label="Resize node details"]') &&
        document.activeElement?.getAttribute('aria-label') === 'close',
    );
    assert.equal(await inspector.getAttribute('aria-modal'), 'true');
    assert.equal(
      await page
        .locator('.kfp-inspector-panel')
        .evaluate((element) => Math.abs(element.getBoundingClientRect().width - innerWidth) < 1),
      true,
    );
    await page.setViewportSize({ width: 1440, height: 900 });
    await assertWidth(360);
    await page.waitForFunction(
      () => document.querySelector('[role="dialog"]')?.getAttribute('aria-modal') !== 'true',
    );
    await close.focus();
    await page.keyboard.press('Escape');
    await inspector.waitFor({ state: 'hidden' });
    await page.waitForFunction((target) => document.activeElement === target, desktopReturnTarget);
    await desktopReturnTarget.dispose();
    await page.setViewportSize({ width: 600, height: 900 });
    // Bring the task into the resized canvas before testing keyboard activation.
    await page.getByRole('button', { name: 'Fit View', exact: true }).click();
    await trigger.focus();
    await page.keyboard.press('Enter');
    await inspector.waitFor();
    const narrowReturnTarget = await page.evaluateHandle(() => window.kfpFixtureGraphFocusTarget);
    assert.equal(await inspector.getAttribute('aria-modal'), 'true');
    await close.focus();
    await page.keyboard.press('Shift+Tab');
    await page.waitForFunction(() => {
      const panel = document.querySelector('[role="dialog"]');
      const controls = [
        ...panel.querySelectorAll('a[href],button,input,select,textarea,[tabindex]'),
      ].filter(
        (element) =>
          element.tabIndex >= 0 &&
          !element.disabled &&
          element.getBoundingClientRect().width > 0 &&
          getComputedStyle(element).visibility !== 'hidden',
      );
      return document.activeElement === controls.at(-1);
    });
    await page.keyboard.press('Tab');
    await page.waitForFunction(
      () => document.activeElement?.getAttribute('aria-label') === 'close',
    );
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
      true,
    );
    await screenshot(page, 'run-details-narrow-inspector');
    await page.keyboard.press('Escape');
    await inspector.waitFor({ state: 'hidden' });
    await page.waitForFunction((target) => document.activeElement === target, narrowReturnTarget);
    await narrowReturnTarget.dispose();
  });
});

// Covers the detail selectors and modal dismissal used by the live Selenium smoke tests.
test('Run Details exposes semantic values and restores navigation after closing narrow logs', async () => {
  await withFixture({}, async (page) => {
    await page.setViewportSize({ width: 780, height: 437 });
    await page.goto(`${origin}/#/runs/details/${runId}`);
    await tab(page, 'Detail').click();
    const value = (label) =>
      page.getByText(label, { exact: true }).and(page.locator('dt')).locator('..').locator('dd');
    await value('Status').getByText('Succeeded', { exact: true }).waitFor();
    assert.equal(await value('Description').innerText(), 'Run Details browser fixture');
    assert.equal(await value('message').innerText(), 'fixture input');
    assert.equal(
      await value('Created at').evaluate((element) => Date.parse(element.textContent)),
      Date.parse(createdAt),
    );
    await tab(page, 'Graph').click();
    await openTask(page, 'train');
    const inspector = page.getByRole('dialog');
    const narrowReturnTarget = await page.evaluateHandle(() => window.kfpFixtureGraphFocusTarget);
    assert.equal(await inspector.getAttribute('aria-modal'), 'true');
    await tab(page, 'Logs').click();
    await page
      .getByTestId('logs-view-window')
      .getByText('executor attempt one', { exact: true })
      .waitFor();
    await inspector.getByRole('button', { name: 'close', exact: true }).click();
    await inspector.waitFor({ state: 'hidden' });
    await page.getByRole('link', { name: 'Runs', exact: true }).click({ trial: true });
  });
});
