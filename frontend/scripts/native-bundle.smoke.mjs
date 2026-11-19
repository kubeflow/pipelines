// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { chromium } from 'playwright';

test(
  'production entrypoints use only native APIs and keep credentials out of storage',
  { timeout: 60000 },
  async () => {
    const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
    try {
      for (const entry of ['index.html', 'argo.html']) {
        const page = await browser.newPage();
        const errors = [],
          requests = [];
        page.on('pageerror', (error) => errors.push(error.message));
        await page.route('**/*', async (route) => {
          const request = route.request();
          const url = new URL(request.url());
          assert.equal(url.origin, 'http://native.test');
          if (url.pathname.endsWith('.html') || url.pathname.startsWith('/static/')) {
            const file = url.pathname.slice(1);
            const contentType = file.endsWith('.js')
              ? 'text/javascript'
              : file.endsWith('.css')
                ? 'text/css'
                : 'text/html';
            await route.fulfill({
              body: await readFile(new URL(`../build/${file}`, import.meta.url)),
              contentType,
            });
            return;
          }
          requests.push(url.pathname);
          assert.ok(
            url.pathname.startsWith('/argo/') || url.pathname.startsWith('/kubernetes/'),
            url.pathname,
          );
          assert.equal(request.headers().authorization, 'Bearer namespace-test-token');
          await route.fulfill({
            contentType: 'application/json',
            body: JSON.stringify({ items: [] }),
          });
        });
        await page.goto(`http://native.test/${entry}?namespace=team`);
        await page.getByLabel('Kubernetes bearer token').fill('namespace-test-token');
        await page.getByRole('button', { name: 'Connect', exact: true }).click();
        await page.getByText('No labeled workflows found.', { exact: true }).waitFor();
        for (const name of ['Pipelines and versions', 'Experiments', 'Recurring runs']) {
          await page.getByRole('tab', { name, exact: true }).click();
        }
        assert.ok(
          requests.some((path) => path.startsWith('/kubernetes/api/v1/namespaces/team/configmaps')),
        );
        assert.deepEqual(
          await page.evaluate(() => ({
            local: { ...localStorage },
            session: { ...sessionStorage },
          })),
          { local: {}, session: {} },
        );
        await page.reload();
        await page.getByLabel('Kubernetes bearer token').waitFor();
        assert.equal(await page.getByLabel('Kubernetes bearer token').inputValue(), '');
        assert.deepEqual(errors, []);
        await page.close();
      }
    } finally {
      await browser.close();
    }
  },
);
