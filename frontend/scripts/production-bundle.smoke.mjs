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
import { chromium } from 'playwright';
import protobuf from 'protobufjs';

// Minimal wire fixtures for the existing MLMD RPCs (field numbers from metadata_store.proto).
const mlmd = protobuf.parse(`
  syntax = "proto3";
  message Value { int64 int_value = 1; string string_value = 3; }
  message Execution {
    int64 id = 1; string type = 7; int32 last_known_state = 3;
    map<string, Value> custom_properties = 5;
    int64 create_time_since_epoch = 8; int64 last_update_time_since_epoch = 9;
  }
  message Context { int64 id = 1; }
  message ContextResponse { Context context = 1; }
  message ExecutionsResponse { repeated Execution executions = 1; }
`).root;

function grpcResponse(type, data) {
  const payload = type ? mlmd.lookupType(type).encode(data).finish() : new Uint8Array();
  const frame = Buffer.alloc(5 + payload.length);
  frame.writeUInt32BE(payload.length, 1);
  frame.set(payload, 5);
  const trailers = Buffer.from('grpc-status: 0\r\n');
  const trailerFrame = Buffer.alloc(5 + trailers.length);
  trailerFrame[0] = 0x80;
  trailerFrame.writeUInt32BE(trailers.length, 1);
  trailerFrame.set(trailers, 5);
  return Buffer.concat([frame, trailerFrame]);
}

async function routeProductionBundle(page, responses = {}, executions = []) {
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
    if (url.pathname.startsWith('/ml_metadata.')) {
      const method = url.pathname.split('/').at(-1);
      const type =
        method === 'GetContextByTypeAndName'
          ? 'ContextResponse'
          : method === 'GetExecutionsByContext'
            ? 'ExecutionsResponse'
            : undefined;
      const data = type === 'ContextResponse' ? { context: { id: 1 } } : { executions };
      await route.fulfill({
        contentType: 'application/grpc-web+proto',
        body: grpcResponse(type, data),
      });
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
test('production bundle renders the pipeline upload control', { timeout: 30000 }, async () => {
  const browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
  });
  try {
    const page = await browser.newPage();
    const errors = [];
    page.on('pageerror', (error) => errors.push(error.message));
    // Startup only: empty API responses, without a backend or external network.
    await routeProductionBundle(page);
    await page.goto('http://kfp.test/');
    try {
      await page.locator('#createPipelineVersionBtn').waitFor({ state: 'visible', timeout: 10000 });
    } catch (error) {
      assert.deepEqual(errors, [], 'production bundle must initialize without uncaught errors');
      throw error;
    }
    assert.deepEqual(errors, [], 'production bundle must initialize without uncaught errors');
  } finally {
    await browser.close();
  }
});

test(
  'expanded navigation footer remains accessible in short windows',
  { timeout: 30000 },
  async () => {
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    try {
      const page = await browser.newPage({ viewport: { width: 1600, height: 500 } });
      await routeProductionBundle(page);
      await page.goto('http://kfp.test/');
      await page.locator('#createPipelineVersionBtn').waitFor();
      const navigation = page.getByTestId('sideNav');
      const reportIssue = navigation.getByRole('link', { name: 'Report an Issue', exact: true });
      await reportIssue.scrollIntoViewIfNeeded();
      await reportIssue.click({ trial: true, timeout: 2000 });
      const version = navigation.getByText('Version:', { exact: true });
      const bounds = await version.boundingBox();
      assert.ok(bounds && bounds.height > 0 && bounds.y >= 0 && bounds.y + bounds.height <= 500);
      await navigation.getByTestId('chevron-toggle').click({ trial: true, timeout: 2000 });
      await reportIssue.click({ trial: true, timeout: 2000 });
      await navigation.locator('#pipelinesBtn').click({ trial: true, timeout: 2000 });
    } finally {
      await browser.close();
    }
  },
);

test(
  'Timeline keeps task details visible when scrolling long charts',
  { timeout: 60000 },
  async () => {
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
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
      const executions = [
        {
          id: 1,
          type: 'system.DAGExecution',
          customProperties: { task_name: { stringValue: '' } },
        },
        ...tasks.map((task, index) => ({
          id: index + 2,
          type: 'system.ContainerExecution',
          lastKnownState: 3,
          createTimeSinceEpoch: Date.parse(task.create_time),
          lastUpdateTimeSinceEpoch: Date.parse(task.end_time),
          customProperties: {
            task_name: { stringValue: task.name },
            display_name: { stringValue: task.display_name },
            parent_dag_id: { intValue: 1 },
          },
        })),
      ];
      await routeProductionBundle(
        page,
        {
          '/apis/v2beta1/runs/timeline-layout-test': {
            run_id: 'timeline-layout-test',
            state: 'SUCCEEDED',
            created_at: at(0),
            finished_at: at(400),
            pipeline_spec: pipelineSpec,
          },
        },
        executions,
      );
      await page.goto('http://kfp.test/#/runs/details/timeline-layout-test?tab=timeline');
      const timeline = page.getByRole('region', { name: 'Run timeline' });
      const chart = page.getByRole('table', { name: 'Component timeline timings' });
      await chart.waitFor();
      const inspector = page.getByRole('complementary', { name: 'Selected task' });
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
        if (height === 500) {
          assert.ok(
            await inspector.evaluate((element) => element.scrollHeight > element.clientHeight),
            'Details scroll internally in a short window',
          );
        }
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
          .getByTestId('sideNav')
          .getByRole('link', { name: 'Report an Issue', exact: true })
          .click({ trial: true, timeout: 2000 });
      }
      await page.setViewportSize({ width: 1000, height: 900 });
      const chartBounds = await chart.boundingBox();
      const detailsBounds = await inspector.boundingBox();
      assert.equal(
        await inspector.evaluate((element) => getComputedStyle(element).position),
        'static',
      );
      assert.ok(
        detailsBounds.y >= chartBounds.y + chartBounds.height,
        'Narrow layouts stack details below the chart without an overlay',
      );
      await page.setViewportSize({ width: 1600, height: 900 });
      await inspector.getByRole('button', { name: 'Open task in graph' }).click();
      await page.getByTestId('DagCanvas').waitFor();
      await page.getByRole('button', { name: 'Task Details', exact: true }).click();
      const taskNameRow = page.getByText('Task name', { exact: true }).locator('..');
      assert.ok((await taskNameRow.textContent()).includes('Component 33'));
      assert.ok(!page.url().includes('tab=timeline'), 'Graph navigation clears the Timeline URL');
      assert.deepEqual(errors, []);
    } finally {
      await browser.close();
    }
  },
);
