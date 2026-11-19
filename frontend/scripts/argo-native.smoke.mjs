// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

import assert from 'node:assert/strict';
import { test } from 'node:test';
import { chromium } from 'playwright';

// Run after samples/argo-native/verify.py with the Argo and Vite port-forwards up.
test(
  'native Argo runs, graph, cached artifacts, comparison and logs',
  { timeout: 60000 },
  async () => {
    const first = process.env.ARGO_FIRST_RUN;
    const second = process.env.ARGO_SECOND_RUN;
    assert.ok(first && second, 'Set ARGO_FIRST_RUN and ARGO_SECOND_RUN to the verified demo runs.');
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
    const requests = [];
    const errors = [];
    page.on('request', (request) => requests.push(new URL(request.url()).pathname));
    page.on('pageerror', (error) => errors.push(error.message));
    try {
      await page.goto('http://127.0.0.1:3002/argo.html?namespace=argo');
      const runList = page.getByRole('complementary', { name: 'Runs' });
      await runList.getByRole('button', { name: second, exact: true }).click();
      await page.getByRole('button', { name: 'prepare', exact: true }).click();
      await page.getByText('Reused from Argo memoization.', { exact: false }).waitFor();
      await page.getByRole('link', { name: 'dataset', exact: true }).waitFor();
      const producer = await page.getByText('Producer:', { exact: false }).textContent();
      assert.ok(producer.includes(first), producer);
      for (const name of [first, second]) {
        const row = runList.getByRole('button', { name, exact: true }).locator('..');
        await row.getByRole('checkbox', { name: 'Compare' }).check();
      }
      await page.getByRole('cell', { name: 'evaluate.score: 1', exact: true }).waitFor();
      await page
        .getByRole('cell', { name: 'evaluate.score: 0.6666666666666666', exact: true })
        .waitFor();
      await page.getByRole('button', { name: 'Show run logs' }).click();
      await page
        .getByText(/Looking for component `train`/)
        .first()
        .waitFor();
      await page.screenshot({
        path: process.env.ARGO_SCREENSHOT || '/tmp/kfp-argo-ui.png',
        fullPage: true,
      });
      assert.deepEqual(errors, []);
      assert.ok(requests.some((path) => path.startsWith('/argo/api/v1/workflows/')));
      assert.ok(
        !requests.some((path) => /^\/(apis|api|system|k8s)\//.test(path)),
        'A KFP backend request was made.',
      );
    } finally {
      await browser.close();
    }
  },
);
