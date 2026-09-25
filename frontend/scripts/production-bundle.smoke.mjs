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

// Vitest transforms imports differently from the production bundler. Load the
// emitted bundle in a browser to catch startup failures such as Ace import order.
test('production bundle renders the pipeline upload control', { timeout: 30000 }, async () => {
  const browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
  });
  try {
    const page = await browser.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.route('**/*', async route => {
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
      // Startup only: empty API responses, without a backend or external network.
      await route.fulfill({ contentType: 'application/json', body: '{}' });
    });
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
