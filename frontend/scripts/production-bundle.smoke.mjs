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
        '/apis/v2beta1/runs/timeline-layout-test': {
          run_id: 'timeline-layout-test',
          state: 'SUCCEEDED',
          created_at: at(0),
          finished_at: at(400),
          pipeline_spec: pipelineSpec,
        },
        '/apis/v2beta1/runs/timeline-layout-test/tasks': { tasks },
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
