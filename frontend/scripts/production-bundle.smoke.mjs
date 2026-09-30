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
import { chromium, firefox, webkit } from 'playwright';

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
  const engineName = process.env.KFP_BROWSER || 'chromium';
  const engine = { chromium, firefox, webkit }[engineName];
  assert.ok(engine, `Unsupported KFP_BROWSER: ${engineName}`);
  diagnostic.stage('launch');
  let browser;
  try {
    browser = await engine.launch({
      ...(process.env.KFP_BROWSER_EXECUTABLE_PATH
        ? { executablePath: process.env.KFP_BROWSER_EXECUTABLE_PATH }
        : {
            channel:
              engineName === 'chromium' ? process.env.PLAYWRIGHT_CHANNEL || undefined : undefined,
          }),
    });
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
      // Startup only: empty API responses, without a backend or external network.
      await route.fulfill({ contentType: 'application/json', body: '{}' });
    });
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
