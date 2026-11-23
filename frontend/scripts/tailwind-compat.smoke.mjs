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

test('Tailwind retains graph colors, selection rings, and utility precedence', async () => {
  const css = await readFile(new URL('../src/build/tailwind.output.css', import.meta.url), 'utf8');
  const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || undefined });
  try {
    const page = await browser.newPage();
    // MUI injectFirst places component rules before the application stylesheet.
    await page.setContent(
      '<style>.MuiButton-root { color: red; width: 10px; }</style>' +
        '<button class="MuiButton-root text-mui-grey-600 w-60 ring shrink-0">node</button>',
    );
    await page.addStyleTag({ content: css });
    const actual = await page.locator('button').evaluate((node) => {
      const style = getComputedStyle(node);
      return {
        color: style.color,
        width: style.width,
        shadow: style.boxShadow,
        shrink: style.flexShrink,
      };
    });
    assert.equal(actual.color, 'rgb(117, 117, 117)');
    assert.equal(actual.width, '240px');
    assert.equal(actual.shrink, '0');
    assert.match(actual.shadow, /rgba\(59, 130, 246, 0\.5\) 0px 0px 0px 3px/);
  } finally {
    await browser.close();
  }
});
