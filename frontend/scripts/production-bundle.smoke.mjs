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
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { PNG } from 'pngjs';
import { resolve } from 'node:path';
import { startupDiagnostics } from './production-bundle-diagnostics.mjs';
import { launchBrowser } from './browser-launch.mjs';

async function routeProductionBundle(page, responses = {}) {
  await page.route('**/*', async (route) => {
    const url = new URL(route.request().url());
    if (url.origin !== 'http://kfp.test') {
      await route.abort();
      return;
    }
    if (url.pathname === '/' || url.pathname.startsWith('/static/')) {
      const name = url.pathname === '/' ? 'index.html' : url.pathname.slice(1);
      const file = new URL(`../build/${name}`, import.meta.url);
      const contentType = name.endsWith('.js')
        ? 'text/javascript'
        : name.endsWith('.css')
          ? 'text/css'
          : name.endsWith('.html')
            ? 'text/html'
            : 'application/octet-stream';
      await route.fulfill({ body: await readFile(file), contentType });
      return;
    }
    await route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(responses[url.pathname] || {}),
    });
  });
}

test(
  'dataset and model previews load only when opened, on both artifact surfaces',
  { timeout: 60000 },
  async () => {
    const browser = await launchBrowser();
    try {
      const kinds = ['dataset', 'model'];
      const previews = Object.fromEntries(
        await Promise.all(
          kinds.map(async (kind) => [
            kind,
            JSON.parse(
              await readFile(
                new URL(`../mock-backend/data/v2/artifact-previews/${kind}.json`, import.meta.url),
                'utf8',
              ),
            ),
          ]),
        ),
      );
      previews.model.hyperparameters['parameter_'.repeat(25)] = 'v'.repeat(256);
      const artifacts = kinds.map((kind) => ({
        artifact_id: `preview-${kind}`,
        name: `preview-${kind}`,
        type: kind === 'dataset' ? 'Dataset' : 'Model',
        uri: `s3://bucket/private-artifacts/kubeflow/${kind}`,
        namespace: 'kubeflow',
        created_at: '2026-01-01T00:00:00Z',
        metadata: { _kfp_preview: { version: 1, path: '_kfp_preview.json' } },
      }));
      const pipelineSpec = {
        pipelineInfo: { name: 'artifact-previews' },
        root: {
          dag: {
            tasks: {
              produce: { taskInfo: { name: 'produce' }, componentRef: { name: 'producer' } },
            },
          },
        },
        components: {
          producer: {
            executorLabel: 'exec-producer',
            outputDefinitions: {
              artifacts: Object.fromEntries(
                artifacts.map((artifact, index) => [
                  kinds[index],
                  {
                    artifactType: {
                      schemaTitle: `system.${artifact.type}`,
                      schemaVersion: '0.0.1',
                    },
                  },
                ]),
              ),
            },
          },
        },
        deploymentSpec: {
          executors: { 'exec-producer': { container: { image: 'unused-test-image' } } },
        },
      };
      const responses = {
        '/apis/v2/runs/preview-run': {
          run_id: 'preview-run',
          display_name: 'Artifact previews',
          namespace: 'kubeflow',
          state: 'SUCCEEDED',
          pipeline_spec: pipelineSpec,
        },
        '/apis/v2/runs/preview-run/tasks': {
          tasks: [
            {
              task_id: 'root',
              name: 'root',
              type: 'ROOT',
              scope_path: 'root',
              state: 'SUCCEEDED',
              run_id: 'preview-run',
              child_tasks: [{ name: 'produce', task_id: 'producer-task' }],
            },
            {
              task_id: 'producer-task',
              name: 'produce',
              display_name: 'Produce preview artifacts',
              type: 'RUNTIME',
              parent_task_id: 'root',
              scope_path: 'root.produce',
              state: 'SUCCEEDED',
              run_id: 'preview-run',
              outputs: {
                artifacts: artifacts.map((artifact, index) => ({
                  artifact_key: kinds[index],
                  type: 'OUTPUT',
                  artifacts: [artifact],
                })),
              },
            },
          ],
        },
        ...Object.fromEntries(
          artifacts.flatMap((artifact, index) => [
            [`/apis/v2/artifacts/${artifact.artifact_id}`, artifact],
            [`/artifacts/${artifact.artifact_id}/preview`, previews[kinds[index]]],
          ]),
        ),
      };
      for (const surface of ['run', 'details']) {
        for (const [index, kind] of kinds.entries()) {
          const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
          const reads = [];
          page.on('request', (request) => {
            const url = new URL(request.url());
            if (url.pathname.startsWith('/artifacts/')) reads.push(url.pathname);
          });
          await routeProductionBundle(page, responses);
          const artifact = artifacts[index];
          if (surface === 'run') {
            await page.goto('http://kfp.test/#/runs/details/preview-run');
            await page
              .getByTestId('DagCanvas')
              .getByRole('button', { name: kind, exact: true })
              .click();
            await page.getByRole('tab', { name: 'Artifact Info', exact: true }).waitFor();
            assert.deepEqual(reads, [], 'Selecting an artifact must not fetch its contents');
            await page.getByRole('tab', { name: 'Visualization', exact: true }).click();
          } else {
            await page.goto(`http://kfp.test/#/artifacts/${artifact.artifact_id}`);
            const open = page.getByRole('button', { name: 'Show artifact visualization' });
            await open.waitFor();
            assert.deepEqual(reads, [], 'Artifact Details must not eagerly fetch row samples');
            await open.click();
          }
          if (kind === 'dataset') {
            const table = page.getByRole('table', { name: 'Dataset sample' });
            await table.waitFor();
            assert.equal(await table.getByRole('row').count(), 11);
            assert.equal(
              await table.getByRole('columnheader').count(),
              previews.dataset.columns.length,
            );
          } else {
            await page.getByRole('region', { name: 'Model preview' }).waitFor();
            await page.getByText('xgboost.XGBClassifier', { exact: true }).waitFor();
            await page.getByRole('heading', { name: 'Recorded hyperparameters' }).waitFor();
            await page.getByRole('heading', { name: 'Structural information' }).waitFor();
            for (const width of [1600, 800, 640]) {
              await page.setViewportSize({ width, height: 1000 });
              for (const [title, entries] of [
                ['Recorded hyperparameters', previews.model.hyperparameters],
                ['Structural information', previews.model.structure],
              ]) {
                const table = page.getByRole('table', { name: title });
                const layout = await table.evaluate((element) => {
                  const headers = Array.from(element.tHead.rows[0].cells).map((cell) =>
                    cell.getBoundingClientRect(),
                  );
                  const rows = Array.from(element.tBodies[0].rows);
                  return {
                    width: element.getBoundingClientRect().width,
                    bodyWidth: element.tBodies[0].getBoundingClientRect().width,
                    containerWidth: element.parentElement.getBoundingClientRect().width,
                    rowCount: rows.length,
                    columnsAligned: rows.every((row) =>
                      Array.from(row.cells).every((cell, index) => {
                        const bounds = cell.getBoundingClientRect();
                        return (
                          Math.abs(bounds.left - headers[index].left) < 1 &&
                          Math.abs(bounds.width - headers[index].width) < 1 &&
                          cell.scrollWidth <= cell.clientWidth + 1
                        );
                      }),
                    ),
                  };
                });
                assert.equal(layout.rowCount, Object.keys(entries).length);
                assert.ok(Math.abs(layout.width - layout.bodyWidth) < 1);
                assert.ok(Math.abs(layout.width - layout.containerWidth) < 1);
                assert.ok(
                  layout.columnsAligned,
                  `${surface} ${title} must align and wrap its cells at viewport width ${width}`,
                );
              }
            }
          }
          assert.deepEqual(
            reads,
            [`/artifacts/${artifact.artifact_id}/preview`],
            'Only the selected preview file is fetched, never the data/model bundle',
          );
          await page.close();
        }
      }
    } finally {
      await browser.close();
    }
  },
);

// Vitest transforms imports differently from the production bundler. Load the
// emitted bundle in a browser to catch startup failures such as Ace import order.
test('production bundle renders the pipeline upload control', { timeout: 30000 }, async (t) => {
  const directory = resolve(process.env.KFP_BROWSER_REPORT_DIR || 'browser-results');
  const diagnostic = startupDiagnostics(directory, {
    sourceSha: process.env.GITHUB_SHA || null,
    timeoutMs: 30000,
  });
  let page;
  const capture = async () => {
    if (page && !page.isClosed()) {
      try {
        await page.screenshot({
          path: resolve(directory, 'production-startup-failure.png'),
          timeout: 2000,
        });
      } catch (error) {
        diagnostic.report.screenshotError = String(error);
        diagnostic.fail(diagnostic.report.failure?.error || error);
      }
    }
  };
  const aborted = () => {
    diagnostic.fail(t.signal.reason);
    void capture();
  };
  t.signal.addEventListener('abort', aborted, { once: true });
  diagnostic.stage('launch');
  let browser;
  try {
    browser = await launchBrowser();
    diagnostic.stage('version');
    if (process.env.KFP_EXPECTED_BROWSER_VERSION) {
      assert.equal(
        browser.version(),
        process.env.KFP_EXPECTED_BROWSER_VERSION,
        'Browser version must match KFP_EXPECTED_BROWSER_VERSION; select the intended browser binary',
      );
    }
    diagnostic.stage('new-page');
    page = await browser.newPage();
    const errors = [];
    page.on('pageerror', (error) => {
      errors.push(error.message);
      diagnostic.pageError(error.message);
    });
    await routeProductionBundle(page);
    diagnostic.stage('navigation');
    await page.goto('http://kfp.test/');
    diagnostic.stage('upload-control-ready');
    try {
      await page.locator('#createPipelineVersionBtn').waitFor({ state: 'visible', timeout: 10000 });
    } catch (error) {
      assert.deepEqual(errors, [], 'production bundle must initialize without uncaught errors');
      throw error;
    }
    assert.deepEqual(errors, [], 'production bundle must initialize without uncaught errors');
  } catch (error) {
    diagnostic.fail(error);
    await capture();
    throw error;
  } finally {
    diagnostic.stage('browser-close');
    try {
      await browser?.close();
    } catch (error) {
      diagnostic.fail(error);
      throw error;
    }
  }
  t.signal.removeEventListener('abort', aborted);
  diagnostic.pass();
});

test(
  'expanded navigation footer remains accessible in short windows',
  { timeout: 30000 },
  async (t) => {
    const diagnostic = startupDiagnostics(
      resolve(process.env.KFP_BROWSER_REPORT_DIR || 'browser-results', 'production-footer'),
      { sourceSha: process.env.GITHUB_SHA || null, timeoutMs: 30000 },
    );
    const aborted = () => diagnostic.fail(t.signal.reason);
    t.signal.addEventListener('abort', aborted, { once: true });
    let browser;
    try {
      diagnostic.stage('launch');
      browser = await launchBrowser();
      diagnostic.stage('new-page');
      const page = await browser.newPage({ viewport: { width: 1600, height: 500 } });
      page.on('pageerror', (error) => diagnostic.pageError(error.message));
      await routeProductionBundle(page);
      diagnostic.stage('navigation');
      await page.goto('http://kfp.test/');
      diagnostic.stage('upload-control-ready');
      await page.locator('#createPipelineVersionBtn').waitFor();
      const navigation = page.getByRole('complementary', { name: 'Pipelines sidebar' });
      const reportIssue = navigation.getByRole('link', { name: 'Report an issue', exact: true });
      diagnostic.stage('footer-scroll');
      await reportIssue.scrollIntoViewIfNeeded();
      diagnostic.stage('footer-issue-actionable');
      await reportIssue.click({ trial: true, timeout: 2000 });
      diagnostic.stage('footer-bounds');
      const version = navigation.locator('.kfp-shell-footer');
      const bounds = await version.boundingBox();
      assert.ok(bounds && bounds.height > 0 && bounds.y >= 0 && bounds.y + bounds.height <= 500);
      diagnostic.stage('footer-collapse-actionable');
      await navigation
        .getByRole('button', { name: 'Collapse navigation' })
        .click({ trial: true, timeout: 2000 });
      await reportIssue.click({ trial: true, timeout: 2000 });
      diagnostic.stage('pipelines-actionable');
      await navigation.locator('#pipelinesBtn').click({ trial: true, timeout: 2000 });
    } catch (error) {
      diagnostic.fail(error);
      throw error;
    } finally {
      diagnostic.stage('browser-close');
      try {
        await browser?.close();
      } catch (error) {
        diagnostic.fail(error);
        throw error;
      } finally {
        t.signal.removeEventListener('abort', aborted);
      }
    }
    diagnostic.pass();
  },
);

for (const kind of ['pipeline', 'run']) {
  test(
    `${kind} graph is flush with tabs and retains expansion choices across reloads`,
    { timeout: 30000 },
    async () => {
      const browser = await launchBrowser();
      try {
        const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
        const pipelineSpec = {
          pipelineInfo: { name: 'graph-spacing' },
          root: {
            dag: {
              tasks: {
                ...Object.fromEntries(
                  Array.from({ length: 25 }, (_, index) => [
                    `group-${index}`,
                    {
                      taskInfo: { name: `Group ${index}` },
                      componentRef: { name: 'group' },
                      dependentTasks: index ? [`group-${index - 1}`] : [],
                    },
                  ]),
                ),
              },
            },
          },
          components: {
            step: { executorLabel: 'exec-step' },
            group: {
              dag: {
                tasks: {
                  step: {
                    taskInfo: { name: 'Example task' },
                    componentRef: { name: 'step' },
                  },
                },
              },
            },
          },
          deploymentSpec: {
            executors: { 'exec-step': { container: { image: 'unused-test-image' } } },
          },
        };
        const version = {
          pipeline_id: 'graph-spacing',
          pipeline_version_id: 'version',
          display_name: 'v1',
          pipeline_spec: pipelineSpec,
        };
        await routeProductionBundle(page, {
          '/apis/v2/pipelines/graph-spacing': {
            pipeline_id: 'graph-spacing',
            display_name: 'Graph spacing',
          },
          '/apis/v2/pipelines/graph-spacing/versions/version': version,
          '/apis/v2/pipelines/graph-spacing/versions': { pipeline_versions: [version] },
          '/apis/v2/runs/graph-spacing': {
            run_id: 'graph-spacing',
            display_name: 'Graph spacing',
            state: 'SUCCEEDED',
            pipeline_spec: pipelineSpec,
          },
          '/apis/v2/runs/graph-spacing/tasks': { tasks: [] },
        });
        const route =
          kind === 'pipeline'
            ? '/pipelines/details/graph-spacing/version/version'
            : '/runs/details/graph-spacing';
        await page.goto(`http://kfp.test/#${route}`);
        await page.getByText('Example task', { exact: true }).first().waitFor();
        assert.equal(await page.getByText('Example task', { exact: true }).count(), 10);
        for (const height of [900, 500]) {
          await page.setViewportSize({ width: 1440, height });
          const canvas = await page.getByTestId('DagCanvas').boundingBox();
          const tabsBottom = await page
            .getByRole('tab', { name: 'Graph', exact: true })
            .evaluate((button) => button.parentElement.getBoundingClientRect().bottom);
          assert.equal(canvas.y - tabsBottom, 0, 'Graph starts directly below the tab divider');
          const canvasTop = PNG.sync.read(
            await page.screenshot({
              clip: {
                x: canvas.x + 1,
                y: tabsBottom,
                width: Math.floor(canvas.width) - 2,
                height: 8,
              },
            }),
          );
          assert.ok(
            canvasTop.data.some((channel) => channel !== 255),
            'The canvas starts without an opaque white strip',
          );
          assert.ok(canvas.height > 0, 'Canvas retains a usable height');
          assert.ok(
            await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
            'Graph does not introduce horizontal page overflow',
          );
          // The modern run summary intentionally scrolls on short viewports.
          // The graph and its controls must remain reachable within that layout.
          const collapseControl = page.getByRole('button', { name: 'Collapse all', exact: true });
          await collapseControl.scrollIntoViewIfNeeded();
          const controlBox = await collapseControl.boundingBox();
          assert.ok(
            controlBox && controlBox.y >= 0 && controlBox.y + controlBox.height <= height,
            'Graph controls remain reachable by scrolling',
          );
          await collapseControl.click();
          await page.getByRole('button', { name: 'Expand Group 0', exact: true }).waitFor();
          await page.getByRole('button', { name: 'Expand all', exact: true }).click();
          assert.equal(await page.getByText('Example task', { exact: true }).count(), 25);
          await page.getByRole('tab', { name: 'Graph', exact: true }).scrollIntoViewIfNeeded();
        }
        await page.getByRole('button', { name: 'Collapse all', exact: true }).click();
        await page.reload();
        await page.getByRole('button', { name: 'Expand Group 0', exact: true }).waitFor();
        assert.equal(await page.getByText('Example task', { exact: true }).count(), 0);
        await page.getByRole('button', { name: 'Expand all', exact: true }).click();
        await page.reload();
        await page.getByRole('button', { name: 'Collapse Group 24', exact: true }).waitFor();
        assert.equal(await page.getByText('Example task', { exact: true }).count(), 25);
        await page.getByRole('button', { name: 'Collapse Group 24', exact: true }).click();
        await page.reload();
        await page.getByRole('button', { name: 'Expand Group 24', exact: true }).waitFor();
        assert.equal(await page.getByText('Example task', { exact: true }).count(), 24);
      } finally {
        await browser.close();
      }
    },
  );
}

test(
  'Timeline keeps task details visible when scrolling long charts',
  { timeout: 60000 },
  async () => {
    const browser = await launchBrowser();
    try {
      const page = await browser.newPage({ viewport: { width: 1600, height: 900 } });
      const errors = [];
      page.on('pageerror', (error) => errors.push(error.message));
      const at = (seconds) => new Date(Date.UTC(2026, 0, 1) + seconds * 1000).toISOString();
      const tasks = Array.from({ length: 35 }, (_, index) => ({
        task_id: `task-${index}`,
        name: `component-${index}`,
        display_name: `Component ${index}`,
        type: 'RUNTIME',
        state: 'SUCCEEDED',
        create_time: at(index * 10),
        end_time: at(index * 10 + index + 1),
        state_history:
          index === 34
            ? Array.from({ length: 30 }, (_, attempt) => ({
                state: attempt === 29 ? 'SUCCEEDED' : attempt % 2 === 0 ? 'RUNNING' : 'FAILED',
                update_time: at(index * 10 + attempt),
              }))
            : [],
      }));
      const pipelineSpec = {
        pipelineInfo: { name: 'timeline-layout-test' },
        root: {
          dag: {
            tasks: Object.fromEntries(
              tasks.map((task) => [
                task.name,
                {
                  taskInfo: { name: task.name },
                  componentRef: { name: `comp-${task.name}` },
                },
              ]),
            ),
          },
        },
        components: Object.fromEntries(
          tasks.map((task) => [`comp-${task.name}`, { executorLabel: `exec-${task.name}` }]),
        ),
        deploymentSpec: {
          executors: Object.fromEntries(
            tasks.map((task) => [
              `exec-${task.name}`,
              { container: { image: 'unused-test-image' } },
            ]),
          ),
        },
      };
      await routeProductionBundle(page, {
        '/apis/v2/runs/timeline-layout-test': {
          run_id: 'timeline-layout-test',
          state: 'SUCCEEDED',
          created_at: at(0),
          finished_at: at(400),
          pipeline_spec: pipelineSpec,
        },
        '/apis/v2/runs/timeline-layout-test/tasks': { tasks },
      });
      await page.goto('http://kfp.test/#/runs/details/timeline-layout-test?tab=timeline');
      const timeline = page.getByRole('region', { name: 'Run timeline' });
      const chart = page.getByRole('table', { name: 'Component timeline timings' });
      await chart.waitFor();
      const inspector = page.getByRole('complementary', { name: 'Selected task' });
      await chart.getByRole('button', { name: 'Component 0', exact: true }).click();
      // Verify the real production CSS follows theme changes, including inline bar colors.
      for (const colorScheme of ['dark', 'light']) {
        await page.emulateMedia({ colorScheme });
        await page.waitForFunction(
          (dark) =>
            document
              .querySelector('.run-timeline')
              ?.closest('.kfp-theme')
              ?.classList.contains('dark') === dark,
          colorScheme === 'dark',
        );
        const colors = await timeline.evaluate((element) => {
          const matchesToken = (target, property, token) => {
            const probe = document.createElement('span');
            probe.style.color = `var(${token})`;
            element.append(probe);
            const expected = getComputedStyle(probe).color;
            probe.remove();
            return getComputedStyle(target)[property] === expected;
          };
          return {
            background: matchesToken(element, 'backgroundColor', '--background'),
            text: matchesToken(element, 'color', '--foreground'),
            panel: matchesToken(element.querySelector('.rt-panel'), 'backgroundColor', '--card'),
            selected: matchesToken(
              element.querySelector('.rt-selected'),
              'backgroundColor',
              '--primary-soft',
            ),
            status: matchesToken(
              element.querySelector('.rt-status'),
              'color',
              '--status-succeeded',
            ),
            bar: matchesToken(
              element.querySelector('.rt-bar'),
              'backgroundColor',
              '--status-succeeded',
            ),
          };
        });
        assert.ok(
          Object.values(colors).every(Boolean),
          `${colorScheme} Timeline uses theme tokens: ${JSON.stringify(colors)}`,
        );
      }

      for (const height of [900, 500]) {
        await page.setViewportSize({ width: 1600, height });
        await timeline.evaluate((element) => {
          element.scrollTop = 0;
        });
        await chart.getByRole('button', { name: 'Component 0', exact: true }).click();
        const initialTop = (await inspector.boundingBox()).y;
        const last = chart.getByRole('button', { name: 'Component 34', exact: true });
        await last.scrollIntoViewIfNeeded();
        await last.click();
        await inspector.getByRole('heading', { name: 'Component 34', exact: true }).waitFor();
        const bounds = await inspector.boundingBox();
        assert.ok(
          await timeline.evaluate((element) => element.scrollTop > 0),
          'Timeline owns the chart scroll',
        );
        assert.ok(
          Math.abs(bounds.y - initialTop) < 1,
          `Details stay pinned while scrolling (viewport=${height}, before=${initialTop}, after=${bounds.y})`,
        );
        assert.ok(
          bounds.y >= 0 && bounds.y + bounds.height <= height,
          'Details fit in the viewport',
        );
        assert.ok(
          await inspector.evaluate((element) => element.scrollHeight > element.clientHeight),
          'Long details scroll internally',
        );
        await inspector.evaluate((element) => {
          element.scrollTop = element.scrollHeight;
        });
        const footer = await inspector
          .getByRole('button', { name: 'Open task in graph' })
          .boundingBox();
        assert.ok(
          footer.y >= bounds.y && footer.y + footer.height <= bounds.y + bounds.height,
          'The footer remains reachable',
        );
        await chart.getByRole('button', { name: 'Component 33', exact: true }).click();
        assert.equal(
          await inspector.evaluate((element) => element.scrollTop),
          0,
          'A new task starts at the top of its details',
        );
        const nextBounds = await inspector.boundingBox();
        assert.ok(nextBounds.y >= 0 && nextBounds.y + nextBounds.height <= height);
        await page
          .getByRole('complementary', { name: 'Pipelines sidebar' })
          .getByRole('link', { name: 'Report an issue', exact: true })
          .click({ trial: true, timeout: 2000 });
      }
      await page.setViewportSize({ width: 1000, height: 900 });
      // Resizing also changes the scroll anchor. Read both rectangles together
      // after the responsive layout settles, rather than across separate frames.
      await page.waitForFunction(
        () => {
          const chart = document.querySelector(
            '[role="table"][aria-label="Component timeline timings"]',
          );
          const details = document.querySelector('[aria-label="Selected task"]');
          if (!chart || !details) return false;
          const chartBounds = chart.getBoundingClientRect();
          const detailsBounds = details.getBoundingClientRect();
          return (
            getComputedStyle(details).position === 'static' &&
            detailsBounds.y >= chartBounds.y + chartBounds.height
          );
        },
        undefined,
        { timeout: 5000 },
      );
      assert.deepEqual(errors, []);
    } finally {
      await browser.close();
    }
  },
);

test('production creation and transfer routes load on demand', { timeout: 60000 }, async () => {
  const browser = await launchBrowser();
  try {
    const page = await browser.newPage();
    const requests = [];
    const errors = [];
    page.on('request', (request) => requests.push(new URL(request.url()).pathname));
    page.on('pageerror', (error) => errors.push(error.message));
    await routeProductionBundle(page);
    await page.goto('http://kfp.test/');
    await page.locator('#createPipelineVersionBtn').waitFor();
    const deferred =
      /\/(?:CreateRunPage|CreateRunForm|CreateExperimentPage|UploadPipelinePage|MetadataTransfer)-/;
    assert.equal(
      requests.some((path) => deferred.test(path)),
      false,
    );
    for (const [route, selector] of [
      ['/experiments/new', '#experimentName'],
      ['/runs/new', 'input[required]'],
      ['/pipeline_versions/new', '#newPipelineName'],
      ['/export-import', 'text=Export / Import'],
    ]) {
      await page.goto(`http://kfp.test/#${route}`);
      await page.locator(selector).first().waitFor();
    }
    assert.ok(requests.some((path) => /\/MetadataTransfer-/.test(path)));
    assert.ok(requests.some((path) => /\/CreateRunForm-/.test(path)));
    assert.ok(requests.some((path) => /\/UploadPipelinePage-/.test(path)));
    assert.deepEqual(errors, []);
  } finally {
    await browser.close();
  }
});
