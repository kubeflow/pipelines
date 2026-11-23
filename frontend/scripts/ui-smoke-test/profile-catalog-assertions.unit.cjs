// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { scanCatalog } = require('./profile-catalog-assertions.cjs');

test('catalog finds the owner on page two and detects a foreign row on page three', async t => {
  const browser = await chromium.launch(process.env.PROFILE_TEST_CHROME_CHANNEL ?
    { channel: process.env.PROFILE_TEST_CHROME_CHANNEL } : {});
  t.after(() => browser.close());
  const page = await browser.newPage({ baseURL: 'http://catalog.test' });
  await page.route('http://catalog.test/', route => route.fulfill({
    contentType: 'text/html',
    body: `<main data-testid="table-row" data-row-id="1">unrelated</main>
      <button data-testid="next-page-btn">Next page</button>
      <script>
        const labels = ['unrelated', 'owner', 'foreign'];
        let index = 0;
        const button = document.querySelector('button');
        button.onclick = () => {
          ++index;
          const row = document.querySelector('main');
          row.dataset.rowId = index + 1;
          row.textContent = labels[index];
          button.disabled = index === labels.length - 1;
        };
      </script>`,
  }));
  const rows = page.getByTestId('table-row');
  assert.equal(await scanCatalog(page, () => rows.filter({ hasText: 'owner' }).count()), true);
  await page.goto('about:blank');

  const seen = [];
  await scanCatalog(page, async () => {
    seen.push(await rows.first().innerText());
    return false;
  });
  assert.deepEqual(seen, ['unrelated', 'owner', 'foreign']);
});
