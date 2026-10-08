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
const examples = [
  { story: 'nested-artifacts', collapse: ['Training and evaluation'] },
  { story: 'conditions', collapse: ['If accuracy is sufficient', 'Otherwise retrain'] },
  { story: 'parallel-iterations', collapse: ['sweep.0', 'sweep.1'] },
  { story: 'nested-loops', collapse: ['for-loop-2', 'for-loop-14'] },
  { story: 'exit-handler', collapse: ['my-pipeline', 'condition-1'] },
];

async function settle() {
  await page.locator('.react-flow__edge-path').first().waitFor({ state: 'attached' });
  // Let ResizeObserver and React Flow finish measuring nodes and routing handles.
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
}

async function screenshot(name) {
  await settle();
  const clippedNodes = await page.locator('.react-flow__node').evaluateAll(nodes => {
    const canvas = document.querySelector('[data-testid="DagCanvas"]').getBoundingClientRect();
    return nodes.filter(node => {
      const rect = node.getBoundingClientRect();
      return rect.left < canvas.left - 1 || rect.right > canvas.right + 1 ||
        rect.top < canvas.top - 1 || rect.bottom > canvas.bottom + 1;
    }).map(node => node.getAttribute('data-id'));
  });
  assert.deepEqual(clippedNodes, [], `${name} must fit the complete visible topology`);
  for (const expanded of [true, false]) {
    const buttons = page.locator(`button[aria-expanded="${expanded}"]`);
    assert.equal(await buttons.getByTestId(expanded ? 'CloseFullscreenIcon' : 'OpenInFullIcon').count(), await buttons.count());
  }
  await page.screenshot({ path: path.join(outDir, `${name}.png`) });
  console.log(`Captured ${name}`);
}

try {
  for (const { story, collapse } of examples) {
    await page.goto(`${baseUrl}/iframe.html?id=v2-groupeddag--${story}&viewMode=story`);
    await page.locator('.react-flow__node-subDagGroup').first().waitFor();
    await settle();
    assert.ok(await page.locator('.react-flow__node-subDagGroup').count() >= 2);
    assert.equal(await page.getByText('Unable to display sub-DAG.', { exact: false }).count(), 0);
    await screenshot(`${story}-expanded`);
    const before = await page.locator('.react-flow__node').count();
    for (const [index, label] of collapse.entries()) {
      const button = page.getByRole('button', { name: `Collapse ${label}`, exact: true });
      // Exercise keyboard activation as well as pointer activation.
      if (story === 'nested-artifacts') {
        await button.focus();
        await page.keyboard.press('Enter');
      } else {
        await button.click();
      }
      await page.getByRole('button', { name: `Expand ${label}`, exact: true }).waitFor();
      if (story === 'parallel-iterations' && index === 0) {
        assert.equal(await page.getByRole('button', { name: 'Collapse sweep.1', exact: true }).count(), 1);
        assert.equal(await page.locator('.react-flow__node-EXECUTION').count(), 3);
      }
    }
    assert.ok(await page.locator('.react-flow__node').count() < before);
    if (story === 'nested-artifacts') {
      assert.equal(await page.getByText('Train model', { exact: true }).count(), 0);
      assert.equal(await page.getByText('Prepare data', { exact: true }).count(), 1);
    }
    await settle();
    await page.locator('.react-flow__controls-fitview').click();
    await screenshot(`${story}-collapsed`);
    for (const label of collapse) {
      await page.getByRole('button', { name: `Expand ${label}`, exact: true }).click();
    }
    await settle();
    assert.equal(await page.locator('.react-flow__node').count(), before);
    if (story === 'nested-artifacts') {
      await page.getByText('Train model', { exact: true }).click();
      assert.ok((await page.locator('body').innerText()).includes('root / workflow / fit / Train model'));
    }
  }
  assert.deepEqual(errors, []);
  console.log('Grouped DAG browser smoke checks passed (five expanded/collapsed pairs).');
} finally {
  await browser.close();
}
