// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
// Run against `npm run storybook -- --ci --no-open`.
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'playwright';

const baseUrl = process.env.STORYBOOK_URL || 'http://localhost:6006';
const outDir = process.argv[2] || '.visual/grouped-dags';
await mkdir(outDir, { recursive: true });
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 }, deviceScaleFactor: 1 });
const errors = [];
page.on('pageerror', error => errors.push(error.message));

async function settle() {
  await page.locator('.react-flow__edge-path').first().waitFor({ state: 'attached' });
  // Let ResizeObserver and React Flow finish measuring nodes and routing handles.
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
}

async function screenshot(name) {
  await settle();
  await page.screenshot({ path: path.join(outDir, `${name}.png`) });
  console.log(`Captured ${name}`);
}

try {
  for (const story of ['nested-artifacts', 'conditions', 'parallel-iterations', 'nested-loops', 'exit-handler']) {
    await page.goto(`${baseUrl}/iframe.html?id=v2-groupeddag--${story}&viewMode=story`);
    await page.locator('.react-flow__node-subDagGroup').first().waitFor();
    await settle();
    assert.ok(await page.locator('.react-flow__node-subDagGroup').count() >= 2);
    assert.equal(await page.getByText('Unable to display sub-DAG.', { exact: false }).count(), 0);
    const clippedNodes = await page.locator('.react-flow__node').evaluateAll(nodes => {
      const canvas = document.querySelector('[data-testid="DagCanvas"]').getBoundingClientRect();
      return nodes.filter(node => {
        const rect = node.getBoundingClientRect();
        return rect.left < canvas.left - 1 || rect.right > canvas.right + 1 ||
          rect.top < canvas.top - 1 || rect.bottom > canvas.bottom + 1;
      }).map(node => node.getAttribute('data-id'));
    });
    assert.deepEqual(clippedNodes, [], `${story} must fit the complete expanded topology`);
    await screenshot(story);
    if (story === 'nested-artifacts') {
      const before = await page.locator('.react-flow__node').count();
      const button = page.getByRole('button', { name: 'Collapse Training and evaluation', exact: true });
      await button.focus();
      await page.keyboard.press('Enter');
      await page.getByRole('button', { name: 'Expand Training and evaluation', exact: true }).waitFor();
      assert.ok(await page.locator('.react-flow__node').count() < before);
      assert.equal(await page.getByText('Train model', { exact: true }).count(), 0);
      assert.equal(await page.getByText('Prepare data', { exact: true }).count(), 1);
      await page.locator('.react-flow__controls-fitview').click();
      await screenshot('collapsed');
      await page.getByRole('button', { name: 'Expand Training and evaluation', exact: true }).click();
      await page.getByText('Train model', { exact: true }).waitFor();
      assert.equal(await page.locator('.react-flow__node').count(), before);
      await page.getByText('Train model', { exact: true }).click();
      assert.ok((await page.locator('body').innerText()).includes('root / workflow / fit / Train model'));
    }
    if (story === 'parallel-iterations') {
      assert.equal(await page.getByRole('button', { name: /^Collapse sweep\./ }).count(), 2);
      await page.getByRole('button', { name: 'Collapse sweep.0', exact: true }).click();
      assert.equal(await page.getByRole('button', { name: 'Collapse sweep.1', exact: true }).count(), 1);
      assert.equal(await page.locator('.react-flow__node-EXECUTION').count(), 3);
    }
  }
  assert.deepEqual(errors, []);
  console.log('Grouped DAG browser smoke checks passed.');
} finally {
  await browser.close();
}
