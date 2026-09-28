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

// Run after npm run build. All HTTP is bounded to local deterministic fixtures.
import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import { chromium, firefox, webkit } from 'playwright';

const origin = 'http://kfp.test';
const build = new URL('../build/', import.meta.url);
const targetId = 'artifact/01 space';
const detailPath = `/artifacts/${encodeURIComponent(targetId)}`;
const types = [
  'Dataset',
  'Model',
  'Metric',
  'HTML',
  'ClassificationMetric',
  'SlicedClassificationMetric',
];
const typeNumbers = {
  Model: 2,
  Dataset: 3,
  HTML: 4,
  Metric: 6,
  ClassificationMetric: 7,
  SlicedClassificationMetric: 8,
};
const artifacts = Array.from({ length: 18 }, (_, index) => ({
  artifact_id: index === 0 ? targetId : `artifact-${String(index + 1).padStart(2, '0')}`,
  name: `Artifact ${String(index + 1).padStart(2, '0')}`,
  type: types[index % types.length],
  uri: `s3://artifact-bucket/output-${index + 1}.csv?endpoint=https%3A%2F%2Fstorage.example`,
  namespace: index < 16 ? 'team-a' : 'team-b',
  description: 'Deterministic artifact fixture',
  created_at: new Date(Date.UTC(2026, 8, 26 - index)).toISOString(),
}));
const inputArtifact = {
  artifact_id: 'input',
  name: 'Source dataset',
  type: 'Dataset',
  namespace: 'team-a',
};
const outputArtifact = {
  artifact_id: 'output',
  name: 'Trained model',
  type: 'Model',
  namespace: 'team-a',
};
const legacyArtifact = {
  artifact_id: 'legacy',
  name: 'mlpipeline-ui-metadata',
  type: 'Artifact',
  uri: 's3://artifact-bucket/ui.json',
  namespace: 'team-a',
};
const tensorboardLogdir = 'volume://logs-pvc/run/one';
const tensorboardImage = 'registry.example/tensorboard:custom';
const tensorboardPodTemplate = {
  spec: {
    serviceAccountName: 'viewer-account',
    volumes: [{ name: 'logs', persistentVolumeClaim: { claimName: 'logs-pvc' } }],
  },
};
const richOutputs = [
  {
    type: 'table',
    storage: 'inline',
    format: 'csv',
    header: ['Metric', 'Value'],
    source: Array.from(
      { length: 15 },
      (_, index) => `metric-${String(index).padStart(2, '0')},${index}`,
    ).join('\n'),
  },
  {
    type: 'markdown',
    storage: 'inline',
    source:
      '# Viewer report\n\n**Measured values**\n\n<strong>Literal markup</strong>\n\n[External documentation](https://example.com/report)',
  },
  {
    type: 'web-app',
    storage: 'inline',
    source:
      '<!doctype html><html><body><h1>Isolated report</h1><p>Rendered report content</p></body></html>',
  },
];
const relationships = [
  {
    id: 'produced',
    artifact_id: targetId,
    run_id: 'run/one',
    task_id: 'task/producer',
    type: 'OUTPUT',
    key: 'dataset',
    producer: { task_name: 'Declared producer' },
  },
  {
    id: 'consumed',
    artifact_id: targetId,
    run_id: 'run/one',
    task_id: 'task/consumer',
    type: 'COMPONENT_INPUT',
    key: 'training_data',
  },
  {
    id: 'root-output',
    artifact_id: targetId,
    run_id: 'run/one',
    task_id: 'root',
    type: 'OUTPUT',
    key: 'run_output',
  },
];
const tasks = [
  { task_id: 'task/producer', run_id: 'run/one', display_name: 'Prepare dataset', type: 'RUNTIME' },
  { task_id: 'task/consumer', run_id: 'run/one', display_name: 'Train model', type: 'RUNTIME' },
  { task_id: 'root', run_id: 'run/one', display_name: 'Root', type: 'ROOT' },
];
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
const decodeFilter = (value) =>
  value ? JSON.parse(value.startsWith('%') ? decodeURIComponent(value) : value).predicates : [];

async function withFixture(options, exercise) {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'en-US',
    timezoneId: 'UTC',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  const fixture = {
    requests: [],
    errors: [],
    failList: false,
    failArtifact: false,
    failPreview: false,
    failRelationships: false,
    failTask: false,
    failTensorboardStart: false,
    failTensorboardStop: false,
    tensorboardReady: false,
  };
  await context.addInitScript(({ namespace }) => {
    if (location.protocol === 'about:') return;
    localStorage.setItem('kfp.theme', 'light');
    if (namespace)
      window.centraldashboard = {
        CentralDashboardEventHandler: {
          init(callback) {
            const handler = {};
            callback(handler);
            handler.onNamespaceSelected(namespace);
            window.selectFixtureNamespace = (value) => handler.onNamespaceSelected(value);
          },
        },
      };
  }, options);
  page.on('pageerror', (error) => fixture.errors.push(error.message));
  await context.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/pipeline(?=\/)/, '');
    const query = Object.fromEntries(url.searchParams);
    const json = (body, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
    try {
      assert.equal(url.origin, origin, 'no external request is permitted');
      const tensorboardMutation =
        options.tensorboardMutations &&
        ((path === '/apps/tensorboard' && ['POST', 'DELETE'].includes(request.method())) ||
          (path === '/apps/tensorboard/proxy/fixture-token/' && request.method() === 'HEAD'));
      assert.ok(
        request.method() === 'GET' || tensorboardMutation,
        'only the explicit TensorBoard fixture may receive mutations',
      );
      assert.ok(fixture.requests.length < 200, 'fixture traffic must remain bounded');
      fixture.requests.push({ path, pathname: url.pathname, query, method: request.method() });
      if (path === '/embed')
        return route.fulfill({
          contentType: 'text/html',
          body: '<!doctype html><title>Dashboard fixture</title><iframe title="Pipelines" src="/pipeline/#/artifacts" style="width:100%;height:850px;border:0"></iframe>',
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
              `window.KFP_FLAGS.HIDE_SIDENAV=${!!options.embedded};`,
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
      if (path === '/apis/v2beta1/healthz')
        return json({ apiServerTagName: 'fixture', apiServerMultiUser: !!options.namespace });
      if (path === '/system/cluster-name' || path === '/system/project-id')
        return route.fulfill({ body: '' });
      if (path === '/apis/v2beta1/artifacts') {
        assert.equal(request.method(), 'GET');
        assert.ok([10, 20, 50, 100].includes(Number(query.page_size)));
        assert.match(
          query.sort_by,
          /^(name|created_at|artifact_id|type|uri|namespace)( (asc|desc))?$/,
        );
        if (fixture.failList)
          return json({ message: 'Artifact list temporarily unavailable' }, 503);
        const predicates = decodeFilter(query.filter);
        let rows = artifacts.filter(
          (artifact) => !query.namespace || artifact.namespace === query.namespace,
        );
        for (const predicate of predicates) {
          if (predicate.key === 'name') {
            assert.equal(predicate.operation, 'IS_SUBSTRING');
            rows = rows.filter((artifact) =>
              artifact.name.toLowerCase().includes(predicate.string_value.toLowerCase()),
            );
          } else {
            assert.equal(predicate.key, 'type');
            assert.equal(predicate.operation, 'IN');
            assert.ok(
              predicate.int_values.values.every((value) => [2, 3, 4, 6, 7, 8].includes(value)),
            );
            rows = rows.filter((artifact) =>
              predicate.int_values.values.includes(typeNumbers[artifact.type]),
            );
          }
        }
        const [sort, direction = 'asc'] = query.sort_by.split(' ');
        rows.sort(
          (a, b) =>
            String(a[sort]).localeCompare(String(b[sort])) * (direction === 'desc' ? -1 : 1),
        );
        assert.ok(!query.page_token || /^offset-\d+$/.test(query.page_token));
        const offset = Number(query.page_token?.replace('offset-', '') || 0);
        const end = offset + Number(query.page_size);
        return json({
          artifacts: rows.slice(offset, end),
          next_page_token: end < rows.length ? `offset-${end}` : '',
        });
      }
      if (path.startsWith('/apis/v2beta1/artifacts/')) {
        if (fixture.failArtifact) return json({ message: 'Artifact temporarily unavailable' }, 503);
        const id = decodeURIComponent(path.slice('/apis/v2beta1/artifacts/'.length));
        const artifact = [...artifacts, inputArtifact, outputArtifact, legacyArtifact].find(
          (item) => item.artifact_id === id,
        );
        assert.ok(artifact, `unknown artifact ${id}`);
        return json(artifact);
      }
      if (path === '/apis/v2beta1/artifact_tasks') {
        assert.equal(request.method(), 'GET');
        assert.ok([5, 10, 20, 50, 100].includes(Number(query.page_size)));
        if (fixture.failRelationships)
          return json({ message: 'Relationship page temporarily unavailable' }, 503);
        if (query.task_ids)
          return json({
            artifact_tasks:
              query.task_ids === 'task/producer'
                ? [
                    {
                      id: 'input-edge',
                      task_id: 'task/producer',
                      artifact_id: 'input',
                      type: 'COMPONENT_INPUT',
                      key: 'source',
                    },
                  ]
                : query.task_ids === 'task/consumer'
                  ? [
                      {
                        id: 'output-edge',
                        task_id: 'task/consumer',
                        artifact_id: 'output',
                        type: 'OUTPUT',
                        key: 'model',
                      },
                    ]
                  : [],
          });
        if (query.artifact_ids !== targetId) return json({ artifact_tasks: [] });
        assert.ok(!query.page_token || query.page_token === 'related-next');
        return json({
          artifact_tasks: query.page_token ? [relationships[2]] : relationships.slice(0, 2),
          next_page_token: query.page_token ? '' : 'related-next',
        });
      }
      if (path === '/apis/v2beta1/runs/run%2Fone')
        return json({ run_id: 'run/one', display_name: 'Artifact training run' });
      if (path === '/apis/v2beta1/runs/run%2Fone/tasks') return json({ tasks });
      if (path.startsWith('/apis/v2beta1/runs/run%2Fone/tasks/')) {
        if (fixture.failTask) return json({ message: 'Task temporarily unavailable' }, 503);
        const id = decodeURIComponent(path.slice('/apis/v2beta1/runs/run%2Fone/tasks/'.length));
        const task = tasks.find((item) => item.task_id === id);
        assert.ok(task, `unknown task ${id}`);
        return json(task);
      }
      if (path === '/artifacts/get') {
        assert.equal(query.namespace, 'team-a');
        assert.equal(query.source, 's3');
        assert.equal(query.bucket, 'artifact-bucket');
        if (query.key === 'ui.json')
          return json({
            outputs: options.richViewers
              ? richOutputs
              : [
                  {
                    type: 'tensorboard',
                    source: options.tensorboardMutations
                      ? tensorboardLogdir
                      : 's3://artifact-bucket/tensorboard',
                    ...(options.tensorboardMutations
                      ? { image: tensorboardImage, pod_template_spec: tensorboardPodTemplate }
                      : {}),
                  },
                ],
          });
        assert.equal(query.key, 'output-1.csv');
        assert.equal(query.artifactUriQuery, 'endpoint=https%3A%2F%2Fstorage.example');
        assert.equal(query.peek, '256');
        if (fixture.failPreview)
          return route.fulfill({ status: 503, body: 'Preview temporarily unavailable' });
        return route.fulfill({ contentType: 'text/plain', body: 'sample,value\nrow,42\n' });
      }
      if (path === '/apps/tensorboard/proxy/fixture-token/') {
        assert.ok(options.tensorboardMutations);
        if (request.method() === 'HEAD')
          return route.fulfill({ status: fixture.tensorboardReady ? 200 : 503 });
        return route.fulfill({
          contentType: 'text/html',
          body: '<!doctype html><title>TensorBoard proxy fixture</title><h1>TensorBoard fixture</h1>',
        });
      }
      if (path === '/apps/tensorboard') {
        assert.equal(query.namespace, 'team-a');
        assert.equal(
          query.logdir,
          options.tensorboardMutations ? tensorboardLogdir : 's3://artifact-bucket/tensorboard',
        );
        if (request.method() === 'GET') return json({ proxyPath: '', image: '' });
        assert.ok(options.tensorboardMutations);
        if (request.method() === 'POST') {
          assert.equal(query.image, tensorboardImage);
          assert.deepEqual(JSON.parse(query.podtemplatespec), tensorboardPodTemplate);
          assert.equal(request.headers()['content-type'], 'application/json');
          return route.fulfill({
            status: fixture.failTensorboardStart ? 403 : 200,
            body: fixture.failTensorboardStart
              ? 'Start forbidden by fixture'
              : 'apps/tensorboard/proxy/fixture-token/',
          });
        }
        assert.equal(request.method(), 'DELETE');
        return route.fulfill({
          status: fixture.failTensorboardStop ? 403 : 200,
          body: fixture.failTensorboardStop ? 'Stop forbidden by fixture' : '',
        });
      }
      throw new Error(`Unexpected request: ${request.method()} ${path}`);
    } catch (error) {
      fixture.errors.push(error.message);
      await json({ message: error.message }, 500);
    }
  });
  try {
    await exercise(page, fixture);
    assert.deepEqual(fixture.errors, [], 'application and HTTP contracts must remain valid');
  } catch (error) {
    if (fixture.errors.length) console.error('Fixture errors:', fixture.errors);
    throw error;
  } finally {
    await context.close();
  }
}

async function ready(page, count) {
  await page.getByRole('table', { name: 'Artifacts', exact: true }).waitFor();
  await page.waitForFunction((expected) => {
    const table = document.querySelector('table[aria-label="Artifacts"]');
    return (
      table?.getAttribute('aria-busy') === 'false' &&
      table.querySelectorAll('[data-row-id]').length === expected
    );
  }, count);
}
async function screenshot(page, name) {
  if (!process.env.KFP_ARTIFACTS_SCREENSHOT_DIR) return;
  await mkdir(process.env.KFP_ARTIFACTS_SCREENSHOT_DIR, { recursive: true });
  await page.screenshot({
    path: join(process.env.KFP_ARTIFACTS_SCREENSHOT_DIR, `${name}.png`),
    animations: 'disabled',
  });
}

test('Artifacts combines scoped type/name filters with token paging, sorting, and refresh recovery', async () => {
  await withFixture({ namespace: 'team-a' }, async (page, fixture) => {
    fixture.failList = true;
    await page.goto(`${origin}/#/artifacts`);
    await page.getByRole('alert').waitFor();
    fixture.failList = false;
    await page.getByRole('button', { name: 'Refresh', exact: true }).click();
    await ready(page, 10);
    await page.getByRole('alert').waitFor({ state: 'hidden' });
    assert.equal(
      fixture.requests.filter((item) => item.path.startsWith('/apis/v2beta1/artifacts/')).length,
      0,
      'list rows must not fetch per-artifact details',
    );
    await screenshot(page, 'artifacts-light');
    await page.getByRole('button', { name: 'Next page', exact: true }).click();
    await ready(page, 6);
    await page.getByRole('searchbox', { name: 'Filter artifacts by name' }).fill('Artifact 1');
    await ready(page, 7);
    assert.ok(
      !fixture.requests.filter((item) => item.path === '/apis/v2beta1/artifacts').at(-1).query
        .page_token,
    );
    await page.getByRole('button', { name: 'Name', exact: true }).click();
    await page.getByRole('columnheader', { name: 'Name', exact: true }).waitFor();
    await page.getByRole('button', { name: 'Metrics', exact: true }).click();
    await ready(page, 3);
    const query = fixture.requests
      .filter((item) => item.path === '/apis/v2beta1/artifacts')
      .at(-1).query;
    assert.equal(query.namespace, 'team-a');
    assert.equal(query.sort_by, 'name');
    assert.deepEqual(
      decodeFilter(query.filter).find((predicate) => predicate.key === 'type').int_values.values,
      [6, 7, 8],
    );
    await page.getByRole('button', { name: 'All', exact: true }).click();
    await ready(page, 7);
    await page.getByRole('button', { name: 'Clear filter', exact: true }).click();
    await ready(page, 10);
    await page.getByRole('combobox', { name: 'Rows per page' }).selectOption('20');
    await ready(page, 16);
    await page.evaluate(() => window.selectFixtureNamespace('team-b'));
    await ready(page, 2);
    assert.equal(
      fixture.requests.filter((item) => item.path === '/apis/v2beta1/artifacts').at(-1).query
        .namespace,
      'team-b',
    );
  });
});

test('Artifact details preserves encoded identity, bounded preview consent, provider-aware download, and read recovery', async () => {
  await withFixture({}, async (page, fixture) => {
    fixture.failArtifact = true;
    await page.goto(`${origin}/#${detailPath}`);
    await page.getByRole('alert').waitFor();
    fixture.failArtifact = false;
    await page.getByRole('button', { name: 'Refresh', exact: true }).click();
    await page.getByRole('region', { name: 'Artifact details', exact: true }).waitFor();
    await page.getByRole('alert').waitFor({ state: 'hidden' });
    assert.equal(fixture.requests.filter((item) => item.path === '/artifacts/get').length, 0);
    const download = page.locator('a[download]');
    const url = new URL(await download.getAttribute('href'), origin);
    assert.equal(url.searchParams.get('download'), 'true');
    assert.equal(url.searchParams.get('namespace'), 'team-a');
    assert.equal(
      url.searchParams.get('artifactUriQuery'),
      'endpoint=https%3A%2F%2Fstorage.example',
    );
    await page.getByText('Full URI', { exact: true }).click();
    await page.getByText(artifacts[0].uri, { exact: true }).waitFor();
    fixture.failPreview = true;
    await page.getByRole('button', { name: 'Preview file contents', exact: true }).click();
    await page.getByText('Error in retrieving artifact preview.', { exact: true }).waitFor();
    fixture.failPreview = false;
    await page.getByRole('button', { name: 'Retry preview', exact: true }).click();
    await page.getByText('sample,value\nrow,42', { exact: true }).waitFor();
    assert.equal(fixture.requests.filter((item) => item.path === '/artifacts/get').length, 2);
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await screenshot(page, 'artifact-details-dark');
  });
});

test('Related artifact tasks preserve producer/root semantics, encoded task destinations, and pagination', async () => {
  await withFixture({}, async (page, fixture) => {
    await page.goto(`${origin}/#${detailPath}/lineage`);
    const table = page.getByRole('table', { name: 'Related tasks', exact: true });
    const produced = table.getByRole('link', {
      name: 'Artifact training run · Prepare dataset',
      exact: true,
    });
    await produced.waitFor();
    assert.equal(
      await produced.getAttribute('href'),
      '#/runs/details/run%2Fone?task=task%2Fproducer',
    );
    await table
      .getByRole('link', { name: 'Artifact training run · Train model', exact: true })
      .waitFor();
    assert.equal(
      fixture.requests.filter((item) => item.path === '/apis/v2beta1/runs/run%2Fone/tasks').length,
      1,
      'related rows share task-name reads',
    );
    await page.getByRole('button', { name: 'Next page', exact: true }).click();
    await table
      .getByRole('link', { name: 'Artifact training run · Run output (root)', exact: true })
      .waitFor();
    assert.equal(
      await page.getByRole('button', { name: 'Next page', exact: true }).isEnabled(),
      false,
    );
    await page.getByRole('tab', { name: 'Overview', exact: true }).click();
    await page.waitForURL((url) => url.hash === `#${detailPath}`);
    await page.goBack();
    await table.waitFor();
  });
});

test('Artifact lineage bounds neighborhoods, preserves directed identity and history, and recovers reads', async () => {
  await withFixture({}, async (page, fixture) => {
    await page.goto(`${origin}/#${detailPath}/explorer`);
    const graph = page.getByRole('region', { name: 'Lineage graph' });
    const input = graph.getByRole('button', { name: 'Source dataset', exact: true });
    await input.waitFor();
    await graph.getByRole('button', { name: 'Trained model', exact: true }).waitFor();
    await page.waitForFunction(
      () => document.querySelectorAll('svg path[data-from][data-to]').length === 4,
    );
    assert.deepEqual(
      await graph
        .locator('svg path[data-from][data-to]')
        .evaluateAll((edges) => edges.map((edge) => [edge.dataset.from, edge.dataset.to]).sort()),
      [
        ['task:produced', `target:${targetId}`],
        ['task:produced:artifact:input-edge', 'task:produced'],
        [`target:${targetId}`, 'task:consumed'],
        ['task:consumed', 'task:consumed:artifact:output-edge'],
      ].sort(),
      'rendered arrows retain producer, consumer, and adjacent artifact identities',
    );
    assert.ok(
      fixture.requests
        .filter((item) => item.path === '/apis/v2beta1/artifact_tasks')
        .every((item) => item.query.page_size === '5' && !item.query.page_token),
    );
    assert.equal(
      await graph.getByRole('link', { name: 'Prepare dataset', exact: true }).getAttribute('href'),
      '#/runs/details/run%2Fone?task=task%2Fproducer',
    );
    await input.click();
    await page.getByText('Neighborhood 2', { exact: true }).waitFor({ state: 'attached' });
    await graph.getByText('No recorded relationships.', { exact: true }).waitFor();
    await page
      .getByRole('navigation', { name: 'Lineage history', exact: true })
      .getByRole('button', { name: 'Back', exact: true })
      .click();
    await input.waitFor();
    fixture.failRelationships = true;
    await page.getByRole('button', { name: 'Load more relationships', exact: true }).click();
    await page.getByText('Some relationships could not be loaded.', { exact: false }).waitFor();
    assert.equal(await input.isVisible(), true, 'loaded neighborhood survives a next-page failure');
    fixture.failRelationships = false;
    await page.getByRole('button', { name: 'Retry relationships', exact: true }).click();
    await page
      .getByText('Some relationships could not be loaded.', { exact: false })
      .waitFor({ state: 'hidden' });
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await page.waitForFunction(() => {
      const canvas = document.querySelector('.kfp-lineage-canvas');
      const artifact = document.querySelector('.kfp-lineage-artifact');
      const brand = document.querySelector('.kfp-shell-brand-title');
      return (
        getComputedStyle(canvas).backgroundColor === 'rgb(14, 16, 21)' &&
        getComputedStyle(artifact).backgroundColor === 'rgb(22, 25, 33)' &&
        getComputedStyle(brand).color === 'rgb(236, 238, 244)'
      );
    });
    await screenshot(page, 'artifact-lineage-dark');
    await page.setViewportSize({ width: 600, height: 900 });
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
      true,
    );
    assert.equal(
      await graph.evaluate((element) => element.scrollWidth > element.clientWidth),
      true,
    );
    await screenshot(page, 'artifact-lineage-narrow');
  });
});

test('Legacy artifact metadata retains its namespace-scoped TensorBoard viewer', async () => {
  await withFixture({}, async (page, fixture) => {
    await page.goto(`${origin}/#/artifacts/legacy`);
    await page.getByRole('button', { name: 'Start Tensorboard', exact: true }).waitFor();
    assert.equal(fixture.requests.filter((item) => item.path === '/apps/tensorboard').length, 1);
    assert.equal(
      fixture.requests.filter((item) => item.path === '/apps/tensorboard')[0].query.namespace,
      'team-a',
    );
  });
});

test('Embedded artifacts retain their prefix and active namespace', async () => {
  await withFixture({ namespace: 'team-a', embedded: true }, async (page, fixture) => {
    await page.goto(`${origin}/embed`);
    const app = page.frameLocator('iframe');
    await app.getByRole('link', { name: 'Artifact 01', exact: true }).waitFor();
    assert.equal(await app.getByRole('complementary').count(), 0);
    await app.getByRole('link', { name: 'Artifact 01', exact: true }).click();
    await app.getByRole('region', { name: 'Artifact details', exact: true }).waitFor();
    await app.getByRole('button', { name: 'Preview file contents', exact: true }).click();
    await app.getByText('sample,value\nrow,42', { exact: true }).waitFor();
    const preview = fixture.requests.find((item) => item.path === '/artifacts/get');
    assert.equal(preview.pathname, '/pipeline/artifacts/get');
    assert.equal(preview.query.namespace, 'team-a');
  });
});

test('Rich artifact viewers preserve table paging/sort, Markdown content, and isolated HTML', async () => {
  await withFixture({ richViewers: true }, async (page, fixture) => {
    await page.goto(`${origin}/#/artifacts/legacy`);
    const table = page.getByRole('table', { name: 'Table output', exact: true });
    await table.getByRole('cell', { name: 'metric-00', exact: true }).waitFor();
    assert.equal(await table.locator('tbody tr.kfp-viewer-table-data').count(), 10);
    await table.getByRole('button', { name: 'Metric', exact: true }).click();
    assert.equal(
      await table.locator('tbody tr').first().locator('td').first().textContent(),
      'metric-14',
    );
    assert.equal(
      await table
        .getByRole('columnheader', { name: 'Metric', exact: true })
        .getAttribute('aria-sort'),
      'descending',
    );
    await page.getByRole('button', { name: 'Go to next page', exact: true }).click();
    await page.getByText('11–15 of 15', { exact: true }).waitFor();
    await table.getByRole('cell', { name: 'metric-04', exact: true }).waitFor();
    await page.getByRole('combobox', { name: 'Rows per page', exact: true }).selectOption('25');
    await page.getByText('1–15 of 15', { exact: true }).waitFor();
    assert.equal(await table.locator('tbody tr.kfp-viewer-table-data').count(), 15);
    assert.equal(
      await page.getByRole('button', { name: 'Go to previous page', exact: true }).isEnabled(),
      false,
    );
    const markdown = page.locator('.kfp-markdown-viewer');
    await markdown.getByRole('heading', { name: 'Viewer report', exact: true }).waitFor();
    assert.equal(
      await markdown
        .locator('strong')
        .allTextContents()
        .then((text) => text.join()),
      'Measured values',
    );
    await markdown.getByText('<strong>Literal markup</strong>', { exact: true }).waitFor();
    const externalLink = markdown.getByRole('link', { name: 'External documentation' });
    assert.equal(await externalLink.getAttribute('href'), 'https://example.com/report');
    assert.equal(await externalLink.getAttribute('target'), '_blank');
    assert.match(await externalLink.getAttribute('rel'), /noopener/);
    const report = page.locator('iframe[title="HTML report"]');
    await report.contentFrame().getByRole('heading', { name: 'Isolated report' }).waitFor();
    assert.equal(await report.getAttribute('sandbox'), 'allow-scripts');
    assert.equal(await report.getAttribute('src'), 'about:blank');
    assert.equal(
      fixture.requests.filter((item) => item.path === '/artifacts/get').length,
      1,
      'inline metadata content needs no extra source fetch',
    );
    const lightText = await table.evaluate((element) => getComputedStyle(element).color);
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    await page.waitForFunction((lightColor) => {
      const table = document.querySelector('.kfp-theme.dark table[aria-label="Table output"]');
      const theme = table?.closest('.kfp-theme');
      const cell = table?.querySelector('tbody td');
      return (
        cell &&
        theme &&
        getComputedStyle(cell).color !== lightColor &&
        getComputedStyle(cell).color === getComputedStyle(theme).color
      );
    }, lightText);
    const border = await table
      .locator('tbody tr')
      .first()
      .locator('td')
      .first()
      .evaluate((cell) => {
        const style = getComputedStyle(cell);
        return {
          left: style.borderLeftWidth,
          right: style.borderRightWidth,
          color: style.color,
          background: getComputedStyle(cell.closest('.kfp-theme')).backgroundColor,
          themeText: getComputedStyle(cell.closest('.kfp-theme')).color,
        };
      });
    assert.deepEqual([border.left, border.right], ['1px', '1px']);
    assert.notEqual(border.color, lightText, 'table foreground follows the theme change');
    assert.equal(border.color, border.themeText, 'table inherits the active theme foreground');
    assert.notEqual(border.color, border.background, 'table text remains distinct in dark mode');
    await table.scrollIntoViewIfNeeded();
    await screenshot(page, 'artifact-rich-viewers-dark');
    await markdown.scrollIntoViewIfNeeded();
    await screenshot(page, 'artifact-markdown-dark');
    await report.scrollIntoViewIfNeeded();
    await screenshot(page, 'artifact-html-dark');
    await page.setViewportSize({ width: 600, height: 900 });
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
      true,
    );
  });
});

test('TensorBoard preserves PVC/image/pod-template contracts, proxy readiness, cancellation, and mutation recovery', async () => {
  await withFixture({ namespace: 'team-a', tensorboardMutations: true }, async (page, fixture) => {
    fixture.failTensorboardStart = true;
    await page.goto(`${origin}/#/artifacts/legacy`);
    const image = page.getByRole('combobox', { name: 'TF Image' });
    await image.waitFor();
    assert.equal(await image.inputValue(), tensorboardImage);
    await page.getByRole('button', { name: 'Start Tensorboard', exact: true }).click();
    await page.getByRole('alert').filter({ hasText: 'Start forbidden by fixture' }).waitFor();
    fixture.failTensorboardStart = false;
    const notReady = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === '/apps/tensorboard/proxy/fixture-token/' &&
        response.request().method() === 'HEAD' &&
        response.status() === 503,
    );
    await page.getByRole('button', { name: 'Start Tensorboard', exact: true }).click();
    await notReady;
    const open = page.getByRole('link', { name: 'Open Tensorboard', exact: true });
    await open.waitFor();
    await page
      .getByRole('alert')
      .filter({ hasText: 'Start forbidden by fixture' })
      .waitFor({ state: 'hidden' });
    assert.equal(await open.getAttribute('href'), 'apps/tensorboard/proxy/fixture-token/');
    assert.equal(await open.getAttribute('target'), '_blank');
    await page
      .getByText('Tensorboard is starting, and you may need to wait for a few minutes.', {
        exact: true,
      })
      .waitFor();
    const ready = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === '/apps/tensorboard/proxy/fixture-token/' &&
        response.request().method() === 'HEAD' &&
        response.status() === 200,
    );
    fixture.tensorboardReady = true;
    await ready;
    await page
      .getByText('Tensorboard is starting, and you may need to wait for a few minutes.', {
        exact: true,
      })
      .waitFor({ state: 'hidden' });
    const popupPromise = page.waitForEvent('popup');
    await open.click();
    const popup = await popupPromise;
    await popup.getByRole('heading', { name: 'TensorBoard fixture' }).waitFor();
    assert.equal(new URL(popup.url()).pathname, '/apps/tensorboard/proxy/fixture-token/');
    await popup.close();
    await page.getByRole('combobox', { name: 'Theme', exact: true }).selectOption('dark');
    const stop = page.getByRole('button', { name: 'Stop Tensorboard', exact: true });
    await stop.focus();
    await stop.press('Enter');
    const dialog = page.getByRole('dialog', { name: 'Stop Tensorboard?' });
    const cancel = dialog.getByRole('button', { name: 'Cancel', exact: true });
    await cancel.waitFor();
    await page.waitForFunction(() => document.activeElement?.textContent === 'Cancel');
    assert.equal(await dialog.evaluate((element) => !!element.closest('.kfp-theme.dark')), true);
    await screenshot(page, 'artifact-tensorboard-stop-dark');
    await cancel.press('Shift+Tab');
    await page.waitForFunction(() => document.activeElement?.textContent === 'Stop');
    await page.keyboard.press('Tab');
    await page.waitForFunction(() => document.activeElement?.textContent === 'Cancel');
    await cancel.click();
    await dialog.waitFor({ state: 'hidden' });
    await page.waitForFunction(() => document.activeElement?.textContent === 'Stop Tensorboard');
    assert.equal(fixture.requests.filter((item) => item.method === 'DELETE').length, 0);
    fixture.failTensorboardStop = true;
    await stop.click();
    await dialog.getByRole('button', { name: 'Stop', exact: true }).click();
    await dialog.getByRole('alert').filter({ hasText: 'Stop forbidden by fixture' }).waitFor();
    fixture.failTensorboardStop = false;
    await dialog.getByRole('button', { name: 'Stop', exact: true }).click();
    await dialog.waitFor({ state: 'hidden' });
    await page.getByRole('button', { name: 'Start Tensorboard', exact: true }).waitFor();
    assert.equal(await image.inputValue(), tensorboardImage);
    assert.equal(await page.getByText('Stop forbidden by fixture', { exact: true }).count(), 0);
    assert.equal(fixture.requests.filter((item) => item.method === 'POST').length, 2);
    assert.equal(fixture.requests.filter((item) => item.method === 'DELETE').length, 2);
    assert.ok(fixture.requests.some((item) => item.method === 'HEAD'));
  });
});
