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

test(
  '25 sub-DAGs: default expands 20, Expand all expands 25, Collapse all collapses 25',
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

      async function capture(name) {
        await page.getByRole('button', { name: 'Fit view', exact: true }).click();
        await page.mouse.move(1, 1);
        await page.getByRole('tooltip').waitFor({ state: 'hidden' });
        await page.waitForFunction(() => {
          const canvas = document
            .querySelector('[data-testid="DagCanvas"]')
            .getBoundingClientRect();
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
        await page.evaluate(
          () =>
            new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
        );
        await page.screenshot({ path: path.join(outputDirectory, name) });
      }

      await assertCounts(20, 5);
      for (let index = 0; index < 25; index++) {
        const button = page.getByRole('button', {
          name: `${index < 20 ? 'Collapse' : 'Expand'} Group ${index}`,
          exact: true,
        });
        assert.equal(await button.getAttribute('aria-expanded'), String(index < 20));
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
      await assertCounts(20, 5);
      assert.deepEqual(errors, [], 'graph controls must not cause uncaught browser errors');

      const partial = await readFile(path.join(outputDirectory, 'twenty-five-subdags-default.png'));
      const full = await readFile(path.join(outputDirectory, 'twenty-five-subdags-full.png'));
      const comparison = await browser.newPage({ viewport: { width: 3600, height: 1510 } });
      await comparison.setContent(`<!doctype html><html><head><style>
      * { box-sizing: border-box; } body { margin: 0; font-family: Arial, sans-serif; color: #24364b; }
      main { display: flex; } section { width: 1800px; } header { height: 110px; padding: 22px 32px; }
      h1 { margin: 0 0 8px; font-size: 28px; } p { margin: 0; font-size: 20px; }
      img { display: block; width: 1800px; height: 1400px; }
    </style></head><body><main>
      <section><header><h1>Default — partial expansion</h1><p>20 expanded · 5 collapsed (Groups 20–24, bottom row)</p></header>
        <img src="data:image/png;base64,${partial.toString('base64')}" /></section>
      <section><header><h1>After clicking Expand all — full expansion</h1><p>25 expanded · 0 collapsed</p></header>
        <img src="data:image/png;base64,${full.toString('base64')}" /></section>
    </main></body></html>`);
      await comparison
        .locator('img')
        .evaluateAll((images) => Promise.all(images.map((image) => image.decode())));
      await comparison.screenshot({
        path: path.join(outputDirectory, 'twenty-five-subdags-comparison.png'),
      });
      await writeFile(
        path.join(outputDirectory, 'twenty-five-subdags-validation.json'),
        JSON.stringify(
          {
            story: 'v2-groupeddag--twenty-five-sub-dags',
            default: { expanded: 20, collapsed: 5 },
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
