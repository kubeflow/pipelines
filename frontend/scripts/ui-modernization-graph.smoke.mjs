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

// npm run build && node --test scripts/ui-modernization-graph.smoke.mjs
// KFP_GRAPH_CAPTURE_DIR retains geometry, screenshots and timing samples for same-machine review.
// KFP_GRAPH_REFERENCE_DIR compares identities/connectivity to an earlier capture, not intentional sizing.
// Optional candidate-only timings: KFP_SCALING_SAMPLES=3 KFP_SCALING_OUTPUT_DIR=/tmp/...
// KFP_SOURCE_COMMIT=<built source SHA> node --test --test-name-pattern="candidate-only scaling" <this file>
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import { chromium, firefox, webkit } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const runId = 'graph-fixture';
const createdAt = '2026-09-26T12:00:00.000Z';
const finishedAt = '2026-09-26T12:01:00.000Z';
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

function runtimeTask(id, name, parent, type = 'RUNTIME', extra = {}) {
  return {
    task_id: id,
    run_id: runId,
    name,
    display_name: name,
    parent_task_id: parent,
    type,
    state: 'SUCCEEDED',
    create_time: createdAt,
    end_time: finishedAt,
    state_history: [{ state: 'SUCCEEDED', update_time: finishedAt }],
    ...extra,
  };
}
function leaf(name, component = 'leaf') {
  return { taskInfo: { name }, componentRef: { name: component } };
}
function largeFixture() {
  const graphTasks = {};
  const tasks = [runtimeTask('root-task', 'root', undefined, 'ROOT')];
  const expectedEdges = [];
  const name = (i) => `step-${String(i).padStart(3, '0')}`;
  for (let i = 0; i < 200; i++) {
    const dependencies =
      i < 10 ? [] : [...new Set([name(i - 10), name(Math.floor(i / 10) * 10 - 10)])];
    graphTasks[name(i)] = {
      ...leaf(name(i), i === 0 ? 'producer' : 'leaf'),
      dependentTasks: dependencies,
    };
    expectedEdges.push(
      ...dependencies.map((dependency) => ({
        id: `paramedge.${dependency}.${name(i)}`,
        connection: `Edge from task.${dependency} to task.${name(i)}`,
      })),
    );
    tasks.push(runtimeTask(`runtime-${i}`, name(i), 'root-task'));
  }
  graphTasks[name(10)].inputs = {
    artifacts: {
      dataset: { taskOutputArtifact: { producerTask: name(0), outputArtifactKey: 'dataset' } },
    },
  };
  expectedEdges.push(
    {
      id: `outedge.${name(0)}.dataset`,
      connection: `Edge from task.${name(0)} to artifact.${name(0)}.dataset`,
    },
    {
      id: `inedge.dataset.${name(10)}`,
      connection: `Edge from artifact.${name(0)}.dataset to task.${name(10)}`,
    },
  );
  return {
    spec: {
      pipelineInfo: { name: 'large-200-task-browser-fixture' },
      root: { dag: { tasks: graphTasks } },
      components: {
        leaf: { executorLabel: 'executor' },
        producer: {
          executorLabel: 'executor',
          outputDefinitions: {
            artifacts: {
              dataset: { artifactType: { schemaTitle: 'system.Dataset', schemaVersion: '0.0.1' } },
            },
          },
        },
      },
      deploymentSpec: {
        executors: {
          executor: { container: { image: 'fixture.invalid/image', command: ['true'] } },
        },
      },
    },
    tasks,
    expectedNodes: [
      ...Object.keys(graphTasks).map((key) => `task.${key}`),
      'artifact.step-000.dataset',
    ].sort(),
    expectedEdges: expectedEdges.sort((a, b) => a.id.localeCompare(b.id)),
  };
}
function nestedFixture() {
  const tasks = [
    runtimeTask('root-task', 'root', undefined, 'ROOT'),
    runtimeTask('group-task', 'group', 'root-task', 'DAG', { scope_path: 'root.group' }),
    runtimeTask('loop-task', 'loop', 'group-task', 'LOOP', {
      scope_path: 'root.group.loop',
      type_attributes: { iteration_count: '2' },
    }),
  ];
  for (const iteration of [0, 1]) {
    tasks.push(
      runtimeTask(`body-${iteration}`, 'body', 'loop-task', 'DAG', {
        scope_path: 'root.group.loop.body',
        type_attributes: { iteration_index: String(iteration) },
      }),
    );
    for (const name of ['train', 'validate'])
      tasks.push(
        runtimeTask(`${name}-${iteration}`, name, `body-${iteration}`, 'RUNTIME', {
          scope_path: `root.group.loop.body.${name}`,
          type_attributes: { iteration_index: String(iteration) },
        }),
      );
  }
  return {
    spec: {
      pipelineInfo: { name: 'nested-loop-browser-fixture' },
      root: { dag: { tasks: { group: leaf('group', 'group') } } },
      components: {
        group: { dag: { tasks: { loop: leaf('loop', 'loop') } } },
        loop: { dag: { tasks: { body: leaf('body', 'body') } } },
        body: {
          dag: {
            tasks: {
              train: leaf('train'),
              validate: { ...leaf('validate'), dependentTasks: ['train'] },
            },
          },
        },
        leaf: { executorLabel: 'executor' },
      },
      deploymentSpec: {
        executors: {
          executor: { container: { image: 'fixture.invalid/image', command: ['true'] } },
        },
      },
    },
    tasks,
  };
}

async function withFixture(data, exercise) {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(15000);
  const requests = [],
    errors = [];
  await context.addInitScript(() => localStorage.setItem('kfp.theme', 'light'));
  page.on('pageerror', (error) => errors.push(error.message));
  await context.route('**/*', async (route) => {
    const request = route.request(),
      url = new URL(request.url()),
      path = url.pathname;
    const json = (value, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(value) });
    try {
      assert.equal(url.origin, origin, 'graph fixture must not contact external services');
      assert.ok(requests.length < 200, 'graph fixture requests remain bounded');
      requests.push({
        path,
        query: Object.fromEntries(url.searchParams),
        method: request.method(),
      });
      assert.equal(request.method(), 'GET', 'graph inspection must not mutate data');
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
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      if (path === `/apis/v2beta1/runs/${runId}`)
        return json({
          run_id: runId,
          display_name: data.spec.pipelineInfo.name,
          pipeline_spec: data.spec,
          state: 'SUCCEEDED',
          storage_state: 'AVAILABLE',
          created_at: createdAt,
          finished_at: finishedAt,
        });
      if (path === `/apis/v2beta1/runs/${runId}/tasks`) {
        assert.equal(url.searchParams.get('page_size'), '200');
        assert.equal(url.searchParams.get('order_by'), 'create_time asc');
        const token = url.searchParams.get('page_token') || '';
        assert.ok(['', 'remaining-tasks'].includes(token), 'unexpected runtime task page token');
        return json(
          token
            ? { tasks: data.tasks.slice(200) }
            : {
                tasks: data.tasks.slice(0, 200),
                ...(data.tasks.length > 200 ? { next_page_token: 'remaining-tasks' } : {}),
              },
        );
      }
      throw new Error(`Unexpected graph request ${path}`);
    } catch (error) {
      errors.push(error.message);
      await json({ message: error.message }, 500);
    }
  });
  try {
    await exercise(page, requests);
    assert.deepEqual(
      errors,
      [],
      'production graph and fixture contracts have no unexpected errors',
    );
  } catch (error) {
    if (errors.length) console.error('Graph fixture errors:', errors);
    throw error;
  } finally {
    await context.close();
  }
}

async function geometry(page, expectedCount) {
  await page.getByTestId('DagCanvas').waitFor();
  return page.evaluate(async (count) => {
    await document.fonts.ready;
    const measure = () => ({
      nodes: [...document.querySelectorAll('.react-flow__node')]
        .map((element) => {
          const matrix = new DOMMatrixReadOnly(element.style.transform);
          return {
            id: element.dataset.id,
            type: [...element.classList].find((name) =>
              /^react-flow__node-(EXECUTION|ARTIFACT|SUB_DAG)$/.test(name),
            ),
            x: matrix.m41,
            y: matrix.m42,
            width: element.offsetWidth,
            height: element.offsetHeight,
          };
        })
        .sort((a, b) => a.id.localeCompare(b.id)),
      edges: [...document.querySelectorAll('.react-flow__edge')]
        .map((element) => ({
          id: element.dataset.id,
          connection: element.getAttribute('aria-label'),
          path: element.querySelector('.react-flow__edge-path')?.getAttribute('d') || '',
        }))
        .sort((a, b) => a.id.localeCompare(b.id)),
    });
    let previous,
      stable = 0;
    for (let frame = 0; frame < 180; frame++) {
      await new Promise(requestAnimationFrame);
      const current = measure(),
        key = JSON.stringify(current);
      stable = key === previous ? stable + 1 : 0;
      previous = key;
      if (
        current.nodes.length === count &&
        current.nodes.every((node) => node.width > 0 && node.height > 0) &&
        current.edges.every((edge) => edge.path) &&
        stable >= 3
      )
        return current;
    }
    throw new Error(`Graph did not settle at ${count} nodes`);
  }, expectedCount);
}
function sameGeometry(a, b) {
  assert.deepEqual(
    a.nodes.map(({ id, type }) => ({ id, type })),
    b.nodes.map(({ id, type }) => ({ id, type })),
  );
  assert.deepEqual(
    a.edges.map(({ id, connection }) => ({ id, connection })),
    b.edges.map(({ id, connection }) => ({ id, connection })),
  );
  for (let i = 0; i < a.nodes.length; i++)
    for (const field of ['x', 'y', 'width', 'height'])
      assert.ok(
        Math.abs(a.nodes[i][field] - b.nodes[i][field]) < 0.05,
        `${a.nodes[i].id} ${field} changed`,
      );
  for (let i = 0; i < a.edges.length; i++) {
    const numbers = (path) => path.match(/-?\d+(?:\.\d+)?(?:e[-+]?\d+)?/gi).map(Number);
    const before = numbers(a.edges[i].path),
      after = numbers(b.edges[i].path);
    assert.equal(before.length, after.length);
    assert.ok(
      before.every((n, j) => Math.abs(n - after[j]) < 0.05),
      `${a.edges[i].id} geometry changed`,
    );
  }
}
async function capture(page, name, graph, timings) {
  if (process.env.KFP_GRAPH_REFERENCE_DIR) {
    const reference = JSON.parse(
      await readFile(join(process.env.KFP_GRAPH_REFERENCE_DIR, `${name}.json`), 'utf8'),
    );
    assert.deepEqual(
      graph.nodes.map(({ id, type }) => ({ id, type })),
      reference.graph.nodes.map(({ id, type }) => ({ id, type })),
      'redesign preserves node identities and types',
    );
    assert.deepEqual(
      graph.edges.map(({ id, connection }) => ({ id, connection })),
      reference.graph.edges.map(({ id, connection }) => ({ id, connection })),
      'redesign preserves graph connectivity',
    );
  }
  if (!process.env.KFP_GRAPH_CAPTURE_DIR) return;
  const index = await readFile(new URL('index.html', build), 'utf8');
  const script = index.match(/src="([^"]+\.js)"/)?.[1];
  assert.ok(script, 'production entry script must be identifiable');
  const entry = await readFile(new URL(script.replace(/^\//, ''), build));
  await mkdir(process.env.KFP_GRAPH_CAPTURE_DIR, { recursive: true });
  await writeFile(
    join(process.env.KFP_GRAPH_CAPTURE_DIR, `${name}.json`),
    `${JSON.stringify({ browser: browser.version(), viewport: page.viewportSize(), entryScript: script, entrySha256: createHash('sha256').update(entry).digest('hex'), timings, graph }, null, 2)}\n`,
  );
  await page.screenshot({
    path: join(process.env.KFP_GRAPH_CAPTURE_DIR, `${name}.png`),
    animations: 'disabled',
  });
}
async function selectTask(page, taskId, nodeId) {
  await page.evaluate((id) => {
    const url = new URL(location.href),
      query = new URLSearchParams(url.hash.split('?')[1]);
    query.set('task', id);
    location.hash = `${url.hash.split('?')[0]}?${query}`;
  }, taskId);
  await page.locator(`.react-flow__node[data-id="${nodeId}"].selected`).waitFor();
}
async function viewport(page) {
  return page.locator('.react-flow__viewport').evaluate((element) => {
    const matrix = new DOMMatrixReadOnly(element.style.transform);
    return { x: matrix.m41, y: matrix.m42, zoom: matrix.a };
  });
}
async function changedViewport(page, previous) {
  await page.waitForFunction((before) => {
    const matrix = new DOMMatrixReadOnly(
      document.querySelector('.react-flow__viewport').style.transform,
    );
    return (
      Math.abs(matrix.m41 - before.x) > 1 ||
      Math.abs(matrix.m42 - before.y) > 1 ||
      Math.abs(matrix.a - before.zoom) > 0.01
    );
  }, previous);
}

async function settledViewport(page) {
  await page.evaluate(async () => {
    let previous,
      stable = 0;
    for (let frame = 0; frame < 90; frame++) {
      await new Promise(requestAnimationFrame);
      const transform = document.querySelector('.react-flow__viewport').style.transform;
      stable = transform === previous ? stable + 1 : 0;
      previous = transform;
      if (stable >= 3) return;
    }
    throw new Error('Viewport animation did not settle');
  });
}

test('200-task graph preserves full paginated data, geometry and zoom/pan/selection', async () => {
  const data = largeFixture();
  await withFixture(data, async (page, requests) => {
    const started = performance.now();
    await page.goto(`${origin}/#/runs/details/${runId}`);
    const original = await geometry(page, 201);
    // These automation timings end at the first observed change, not animation completion.
    const timings = {
      graphReadyMs: performance.now() - started,
      selectMs: [],
      zoomMs: [],
      panMs: [],
    };
    assert.deepEqual(
      original.nodes.map(({ id }) => id),
      data.expectedNodes,
    );
    assert.ok(
      original.nodes.every(({ width, height }) => width === 200 && height === 56),
      'leaf and artifact boxes must match the explicit200×56 layout dimensions',
    );
    assert.deepEqual(
      original.edges.map(({ id, connection }) => ({ id, connection })),
      data.expectedEdges,
    );
    assert.deepEqual(
      requests
        .filter(({ path }) => path.endsWith('/tasks'))
        .map(({ query }) => query.page_token || ''),
      ['', 'remaining-tasks'],
    );
    sameGeometry(original, await geometry(page, 201));
    await capture(page, 'large-graph-initial', original, timings);
    for (const i of [199, 101, 0]) {
      const start = performance.now();
      await selectTask(page, `runtime-${i}`, `task.step-${String(i).padStart(3, '0')}`);
      timings.selectMs.push(performance.now() - start);
      sameGeometry(original, await geometry(page, 201));
    }
    await page.getByRole('button', { name: 'close', exact: true }).click();
    const inspect = page.getByRole('button', { name: 'step-000', exact: true });
    await inspect.focus();
    await inspect.press('Enter');
    await page.locator('.react-flow__node[data-id="task.step-000"].selected').waitFor();
    await page.getByRole('tab', { name: 'Task Details', exact: true }).click();
    await page.getByText('runtime-0', { exact: true }).waitFor();
    assert.equal(
      new URLSearchParams(new URL(page.url()).hash.split('?')[1]).get('task'),
      null,
      'direct node inspection clears the previous linked-task query',
    );
    await page.getByRole('button', { name: 'close', exact: true }).click();
    for (let i = 0; i < 3; i++) {
      const previous = await viewport(page),
        start = performance.now();
      await page.getByRole('button', { name: 'Zoom Out', exact: true }).click();
      await changedViewport(page, previous);
      timings.zoomMs.push(performance.now() - start);
    }
    for (let i = 0; i < 3; i++) {
      const point = await page.locator('.react-flow__pane').evaluate((pane) => {
        const box = pane.getBoundingClientRect();
        for (const vertical of [0.15, 0.85, 0.5]) {
          for (const horizontal of [0.25, 0.5, 0.75]) {
            const x = box.x + box.width * horizontal;
            const y = box.y + box.height * vertical;
            if (
              document.elementFromPoint(x, y) === pane &&
              document.elementFromPoint(x + 80, y + 30) === pane
            )
              return { x, y };
          }
        }
        return null;
      });
      assert.ok(point, 'pan starts on exposed canvas, away from graph nodes and overlays');
      const previous = await viewport(page),
        start = performance.now();
      await page.mouse.move(point.x, point.y);
      await page.mouse.down();
      await page.mouse.move(point.x + 80, point.y + 30, { steps: 6 });
      await page.mouse.up();
      await changedViewport(page, previous);
      timings.panMs.push(performance.now() - start);
    }
    sameGeometry(original, await geometry(page, 201));
    await page.getByRole('button', { name: 'Fit View', exact: true }).click();
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await page.waitForFunction(() => {
      const shell = document.querySelector('.kfp-shell');
      const brand = document.querySelector('.kfp-shell-brand-title');
      const node = document.querySelector('.kfp-graph-node');
      return (
        getComputedStyle(shell).backgroundColor === 'rgb(14, 16, 21)' &&
        getComputedStyle(brand).color === 'rgb(236, 238, 244)' &&
        getComputedStyle(node).backgroundColor === 'rgb(22, 25, 33)'
      );
    });
    await settledViewport(page);
    assert.equal(
      await page
        .locator('.kfp-graph-node')
        .first()
        .evaluate((node) => getComputedStyle(node).backgroundColor),
      'rgb(22, 25, 33)',
    );
    sameGeometry(original, await geometry(page, 201));
    await capture(page, 'large-graph-dark', original, timings);
    await page.reload();
    sameGeometry(original, await geometry(page, 201));
    assert.deepEqual(
      requests
        .filter(({ path }) => path.endsWith('/tasks'))
        .map(({ query }) => query.page_token || ''),
      ['', 'remaining-tasks', '', 'remaining-tasks'],
    );
  });
});

test('nested loop task links preserve iteration identities and hierarchy navigation', async () => {
  await withFixture(nestedFixture(), async (page, requests) => {
    await page.goto(`${origin}/#/runs/details/${runId}?task=train-1&view=graph`);
    await page.locator('.react-flow__node[data-id="task.train"].selected').waitFor();
    await page.getByRole('tab', { name: 'Task Details', exact: true }).click();
    await page.getByText('train-1', { exact: true }).waitFor();
    const original = await geometry(page, 2);
    assert.deepEqual(
      original.nodes.map(({ id }) => id),
      ['task.train', 'task.validate'],
    );
    await capture(page, 'nested-loop-body', original, {});
    await selectTask(page, 'train-0', 'task.train');
    await page.getByText('train-0', { exact: true }).waitFor();
    sameGeometry(original, await geometry(page, 2));
    await page.goBack();
    await page.getByText('train-1', { exact: true }).waitFor();
    await page.getByRole('button', { name: 'loop', exact: true }).click();
    const iterations = await geometry(page, 2);
    assert.deepEqual(
      iterations.nodes.map(({ id }) => id),
      ['task.loop.0', 'task.loop.1'],
    );
    await capture(page, 'nested-loop-iterations', iterations, {});
    for (const nodeId of ['task.loop.1', 'task.body']) {
      const expand = page
        .locator(`.react-flow__node[data-id="${nodeId}"]`)
        .getByTestId('expand-button');
      // ReactFlow briefly hides new nodes while measuring them; keyboard focus
      // must wait until the native button is visible and can accept it.
      await expand.waitFor({ state: 'visible' });
      await expand.focus();
      assert.equal(await expand.evaluate((element) => element === document.activeElement), true);
      await expand.press('Enter');
    }
    sameGeometry(original, await geometry(page, 2));
    await page.getByRole('button', { name: 'root', exact: true }).click();
    assert.deepEqual(
      (await geometry(page, 1)).nodes.map(({ id }) => id),
      ['task.group'],
    );
    assert.equal(requests.filter(({ path }) => path.endsWith('/tasks')).length, 1);
  });
});

// These repeated laboratory samples are automation readiness timings, not field INP or a baseline comparison.
if (process.env.KFP_SCALING_SAMPLES) {
  test('candidate-only scaling: 200 tasks and 363 edges in fresh contexts', async () => {
    const count = Number(process.env.KFP_SCALING_SAMPLES);
    assert.ok(Number.isInteger(count) && count >= 3 && count <= 20, 'use 3–20 scaling samples');
    assert.match(
      process.env.KFP_SOURCE_COMMIT || '',
      /^[a-f0-9]{40}$/,
      'identify the built source',
    );
    assert.ok(process.env.KFP_SCALING_OUTPUT_DIR, 'provide a directory for retained raw samples');
    const data = largeFixture();
    assert.equal(data.tasks.length, 201, '200 executable tasks plus the structural root');
    assert.equal(data.expectedNodes.length, 201, '200 task nodes plus one artifact');
    assert.equal(data.expectedEdges.length, 363);
    const samples = [];
    const assetPaths = new Set();
    for (let sample = 1; sample <= count; sample++) {
      await withFixture(data, async (page, requests) => {
        const started = performance.now();
        await page.goto(`${origin}/#/runs/details/${runId}`);
        const initial = await geometry(page, 201);
        assert.deepEqual(
          initial.nodes.map(({ id }) => id),
          data.expectedNodes,
        );
        assert.deepEqual(
          initial.edges.map(({ id, connection }) => ({ id, connection })),
          data.expectedEdges,
        );
        assert.ok(initial.nodes.every(({ width, height }) => width === 200 && height === 56));
        const graphReadyMs = performance.now() - started;
        assert.deepEqual(
          requests
            .filter(({ path }) => path.endsWith('/tasks'))
            .map(({ query }) => query.page_token || ''),
          ['', 'remaining-tasks'],
        );

        // Native keyboard activation exercises the actual rendered task button and inspector.
        const task = page.getByRole('button', { name: 'step-000', exact: true });
        await task.focus();
        const selectStarted = performance.now();
        await task.press('Enter');
        await page.locator('.react-flow__node[data-id="task.step-000"].selected').waitFor();
        await page.getByRole('dialog', { name: 'step-000', exact: true }).waitFor();
        await page.getByRole('tab', { name: 'Task Details', exact: true }).waitFor();
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        const selectInspectorReadyMs = performance.now() - selectStarted;
        await page.getByRole('button', { name: 'close', exact: true }).click();
        await page.getByRole('dialog').waitFor({ state: 'hidden' });
        sameGeometry(initial, await geometry(page, 201));

        // Change zoom first so Fit View must perform a real viewport update.
        const beforeZoom = await viewport(page);
        await page.getByRole('button', { name: 'Zoom Out', exact: true }).click();
        await changedViewport(page, beforeZoom);
        await settledViewport(page);
        const beforeFit = await viewport(page);
        const fitStarted = performance.now();
        await page.getByRole('button', { name: 'Fit View', exact: true }).click();
        await changedViewport(page, beforeFit);
        await settledViewport(page);
        const fitSettledMs = performance.now() - fitStarted;
        sameGeometry(initial, await geometry(page, 201));
        samples.push({ sample, graphReadyMs, selectInspectorReadyMs, fitSettledMs });
        for (const { path } of requests) if (path.startsWith('/static/')) assetPaths.add(path);
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
        viewport: { width: 1440, height: 900 },
        deviceScaleFactor: 1,
        locale: 'en-US',
        timezoneId: 'UTC',
        theme: 'light',
        reducedMotion: 'reduce',
        cpuThrottle: 'none',
        networkThrottle: 'none',
      },
      method:
        'Fresh context per sample in one warm browser process; local route fixtures and file/OS caches stay warm. Automation timings include Playwright overhead. Graph readiness includes complete paginated data and stable geometry/fonts; selection ends at the real inspector controls plus two frames; Fit View ends at a changed viewport stable for three frames. No field INP, backend load, or baseline speedup claim.',
      fixture: {
        executableTasks: 200,
        runtimeRecords: 201,
        renderedNodes: 201,
        edges: 363,
        nodeWidth: 200,
        nodeHeight: 56,
      },
      assets,
      samples,
    };
    await mkdir(process.env.KFP_SCALING_OUTPUT_DIR, { recursive: true });
    await writeFile(
      join(
        process.env.KFP_SCALING_OUTPUT_DIR,
        `graph-${process.env.KFP_BROWSER || 'chromium'}.json`,
      ),
      `${JSON.stringify(report, null, 2)}\n`,
    );
  });
}
