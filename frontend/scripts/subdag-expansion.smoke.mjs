// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
// Start Storybook, then run: node --test scripts/subdag-expansion.smoke.mjs
// Optional: STORYBOOK_URL and SCREENSHOT_DIR override the local server and output directory.

import assert from 'node:assert/strict';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { test } from 'node:test';
import { chromium } from 'playwright';

const baseUrl = process.env.STORYBOOK_URL || 'http://localhost:6006';
const outputDirectory = process.env.SCREENSHOT_DIR || '.visual/subdag-expansion';
const expandedGroups = '[data-testid="subdag-header"] button[aria-expanded="true"]';
const collapsedGroups = '[data-testid="subdag-header"] button[aria-expanded="false"]';

async function captureGraph(page, name, fit = true) {
  if (fit) {
    await page.getByRole('button', { name: 'Fit view', exact: true }).click();
    await page.waitForFunction(() => {
      const canvas = document.querySelector('[data-testid="DagCanvas"]').getBoundingClientRect();
      return [...document.querySelectorAll('.react-flow__node')].every((node) => {
        const rect = node.getBoundingClientRect();
        return (
          rect.width > 0 &&
          rect.height > 0 &&
          rect.left >= canvas.left &&
          rect.right <= canvas.right &&
          rect.top >= canvas.top &&
          rect.bottom <= canvas.bottom
        );
      });
    });
  }
  await page.mouse.move(1, 1);
  await page.getByRole('tooltip').waitFor({ state: 'hidden' });
  await page.evaluate(
    () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
  );
  await page.screenshot({ path: path.join(outputDirectory, name) });
}

async function captureComparison(browser, prefix, height, partialDescription, fullDescription) {
  const partial = await readFile(path.join(outputDirectory, `${prefix}-default.png`));
  const full = await readFile(path.join(outputDirectory, `${prefix}-full.png`));
  const comparison = await browser.newPage({ viewport: { width: 3600, height: height + 110 } });
  await comparison.setContent(`<!doctype html><html><head><style>
    * { box-sizing: border-box; } body { margin: 0; font-family: Arial, sans-serif; color: #24364b; }
    main { display: flex; } section { width: 1800px; } header { height: 110px; padding: 22px 32px; }
    h1 { margin: 0 0 8px; font-size: 28px; } p { margin: 0; font-size: 20px; }
    img { display: block; width: 1800px; height: ${height}px; }
  </style></head><body><main>
    <section><header><h1>Default — partial expansion</h1><p>${partialDescription}</p></header>
      <img src="data:image/png;base64,${partial.toString('base64')}" /></section>
    <section><header><h1>After clicking Expand all — full expansion</h1><p>${fullDescription}</p></header>
      <img src="data:image/png;base64,${full.toString('base64')}" /></section>
  </main></body></html>`);
  await comparison
    .locator('img')
    .evaluateAll((images) => Promise.all(images.map((image) => image.decode())));
  await comparison.screenshot({ path: path.join(outputDirectory, `${prefix}-comparison.png`) });
  await comparison.close();
}

test(
  '25 sub-DAGs: default expands 10, Expand all expands 25, Collapse all collapses 25',
  {
    timeout: 90000,
  },
  async () => {
    await mkdir(outputDirectory, { recursive: true });
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    try {
      const page = await browser.newPage({ viewport: { width: 1800, height: 1400 } });
      const errors = [];
      page.on('pageerror', (error) => errors.push(error.message));
      await page.goto(
        `${baseUrl}/iframe.html?id=v2-groupeddag--twenty-five-sub-dags&viewMode=story`,
      );
      await page.getByRole('button', { name: 'Expand Group 24', exact: true }).waitFor();

      async function assertCounts(expanded, collapsed) {
        await page.waitForFunction(
          ({ selector, expected }) => document.querySelectorAll(selector).length === expected,
          { selector: expandedGroups, expected: expanded },
        );
        assert.equal(await page.locator(expandedGroups).count(), expanded);
        assert.equal(await page.locator(collapsedGroups).count(), collapsed);
        assert.equal(await page.getByText('Step', { exact: true }).count(), expanded);
      }

      const capture = (name) => captureGraph(page, name);

      await assertCounts(10, 15);
      for (let index = 0; index < 25; index++) {
        const button = page.getByRole('button', {
          name: `${index < 10 ? 'Collapse' : 'Expand'} Group ${index}`,
          exact: true,
        });
        assert.equal(await button.getAttribute('aria-expanded'), String(index < 10));
      }
      const labels = await page
        .getByRole('group', { name: 'Graph controls' })
        .getByRole('button')
        .evaluateAll((buttons) => buttons.map((button) => button.getAttribute('aria-label')));
      assert.deepEqual(labels, [
        'Expand all',
        'Collapse all',
        'Zoom in',
        'Zoom out',
        'Fit view',
        'Lock graph',
      ]);
      assert.equal(await page.getByText('Layers', { exact: true }).count(), 0);
      assert.equal(await page.getByRole('button', { name: 'Open Sub-DAG' }).count(), 0);
      await capture('twenty-five-subdags-default.png');

      await page.getByRole('button', { name: 'Expand all', exact: true }).click();
      await assertCounts(25, 0);
      await capture('twenty-five-subdags-full.png');

      await page.getByRole('button', { name: 'Collapse all', exact: true }).click();
      await assertCounts(0, 25);
      await page.getByRole('button', { name: 'Expand Group 24', exact: true }).click();
      await assertCounts(1, 24);
      await page.reload();
      await assertCounts(10, 15);
      assert.deepEqual(errors, [], 'graph controls must not cause uncaught browser errors');

      await captureComparison(
        browser,
        'twenty-five-subdags',
        1400,
        '10 expanded · 15 collapsed (Groups 10–24, bottom three rows)',
        '25 expanded · 0 collapsed',
      );
      await writeFile(
        path.join(outputDirectory, 'twenty-five-subdags-validation.json'),
        JSON.stringify(
          {
            story: 'v2-groupeddag--twenty-five-sub-dags',
            default: { expanded: 10, collapsed: 15 },
            expandAll: { expanded: 25, collapsed: 0 },
            collapseAll: { expanded: 0, collapsed: 25 },
            individualExpansion: { expanded: 1, collapsed: 24 },
            reloadRestoresDefault: true,
            browserErrors: errors,
          },
          null,
          2,
        ) + '\n',
      );
    } finally {
      await browser.close();
    }
  },
);

test(
  '25 nested sub-DAGs with two components per level collapse deeper levels by default',
  {
    timeout: 90000,
  },
  async () => {
    await mkdir(outputDirectory, { recursive: true });
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    try {
      const page = await browser.newPage({ viewport: { width: 1800, height: 1600 } });
      const errors = [];
      page.on('pageerror', (error) => errors.push(error.message));
      await page.goto(
        baseUrl + '/iframe.html?id=v2-groupeddag--twenty-five-nested-sub-dags&viewMode=story',
      );
      await page.getByRole('button', { name: 'Expand Level 11', exact: true }).waitFor();
      const counts = async () => ({
        components: await page.getByText(/^Component \d{3}$/, { exact: true }).count(),
        expanded: await page.locator(expandedGroups).count(),
        collapsed: await page.locator(collapsedGroups).count(),
      });
      const partial = { components: 20, expanded: 10, collapsed: 1 };
      const full = { components: 50, expanded: 25, collapsed: 0 };
      assert.deepEqual(await counts(), partial);
      assert.equal(await page.locator('.react-flow__node').count(), 31);
      assert.equal(await page.getByText('Component 050', { exact: true }).count(), 0);
      await captureGraph(page, 'twenty-five-levels-default.png');
      const boundary = await page
        .getByRole('button', { name: 'Level 11', exact: true })
        .boundingBox();
      await page.mouse.move(boundary.x + boundary.width / 2, boundary.y + boundary.height / 2);
      await page.mouse.wheel(0, -800);
      await page.waitForFunction(
        () =>
          document
            .querySelector('button[aria-label="Expand Level 11"]')
            .closest('.react-flow__node')
            .getBoundingClientRect().width >= 260,
      );
      await captureGraph(page, 'twenty-five-levels-detail-default.png', false);
      await page.getByRole('button', { name: 'Expand all', exact: true }).click();
      assert.deepEqual(await counts(), full);
      assert.equal(await page.locator('.react-flow__node').count(), 75);
      await captureGraph(page, 'twenty-five-levels-full.png');
      await captureComparison(
        browser,
        'twenty-five-levels',
        1600,
        'Levels 1–10 expanded · Level 11 collapsed · 20 components visible',
        'All 25 levels expanded · 50 components visible',
      );

      await page.getByRole('button', { name: 'Collapse Level 10', exact: true }).click();
      assert.deepEqual(await counts(), { components: 18, expanded: 9, collapsed: 1 });
      await page.getByRole('button', { name: 'Expand all', exact: true }).click();
      assert.deepEqual(await counts(), full);
      await page.getByRole('button', { name: 'Collapse all', exact: true }).click();
      assert.deepEqual(await counts(), { components: 0, expanded: 0, collapsed: 1 });
      await page.getByRole('button', { name: 'Expand Level 01', exact: true }).click();
      assert.deepEqual(await counts(), { components: 2, expanded: 1, collapsed: 1 });
      await page.getByRole('button', { name: 'Expand all', exact: true }).click();
      assert.deepEqual(await counts(), full);
      await page.getByRole('button', { name: 'Fit view', exact: true }).click();
      const target = page.getByText('Component 050', { exact: true });
      const bounds = await target.boundingBox();
      await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2);
      await page.mouse.wheel(0, -800);
      await page.waitForFunction(() => {
        const node = [...document.querySelectorAll('.react-flow__node-EXECUTION')].find(
          (node) => node.textContent === 'Component 050',
        );
        return node && node.getBoundingClientRect().width >= 170;
      });
      await target.click();
      const deepestPath = [
        'root',
        ...Array.from({ length: 25 }, (_, index) => 'level-' + (index + 1)),
      ];
      await page
        .getByText('Selected: ' + deepestPath.join(' / ') + ' / Component 050', { exact: false })
        .waitFor();
      await captureGraph(page, 'twenty-five-levels-detail-full.png', false);
      await captureComparison(
        browser,
        'twenty-five-levels-detail',
        1600,
        'Level 11 stays collapsed; its components and deeper levels are hidden',
        'All levels, including 21–25, and their component tasks are revealed',
      );
      await page.reload();
      await page.getByRole('button', { name: 'Expand Level 11', exact: true }).waitFor();
      assert.deepEqual(await counts(), partial);
      assert.deepEqual(errors, []);
      await writeFile(
        path.join(outputDirectory, 'twenty-five-levels-validation.json'),
        JSON.stringify(
          {
            story: 'v2-groupeddag--twenty-five-nested-sub-dags',
            nestingDepth: 25,
            componentsPerLevel: 2,
            totalGraphNodes: 75,
            default: partial,
            expandAll: full,
            defaultHiddenLevels: Array.from({ length: 14 }, (_, index) => index + 12),
            collapseLevel10: { components: 18, expanded: 9, collapsed: 1 },
            collapseAll: { components: 0, expanded: 0, collapsed: 1 },
            reopenLevel1: { components: 2, expanded: 1, collapsed: 1 },
            deepestSelectionPath: deepestPath,
            reloadRestoresDefault: true,
            browserErrors: errors,
          },
          null,
          2,
        ) + '\n',
      );
    } finally {
      await browser.close();
    }
  },
);
