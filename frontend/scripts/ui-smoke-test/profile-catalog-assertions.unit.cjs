// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { scanCatalog } = require('./profile-catalog-assertions.cjs');

async function openCatalog(t, { repeat = false } = {}) {
  const browser = await chromium.launch(process.env.PROFILE_TEST_CHROME_CHANNEL ?
    { channel: process.env.PROFILE_TEST_CHROME_CHANNEL } : {});
  t.after(() => browser.close());
  const page = await browser.newPage({ baseURL: 'http://catalog.test' });
  page.setDefaultTimeout(1500);
  await page.route('http://catalog.test/apis/v2/artifacts?*', route => route.fulfill({ json: {} }));
  await page.route('http://catalog.test/', route => route.fulfill({
    contentType: 'text/html',
    body: `<main data-testid="table-row" data-row-id="1">unrelated</main>
      <button data-testid="next-page-btn">Next page</button>
      <div role="progressbar" hidden></div>
      <script>
        const labels = ['unrelated', 'owner', 'foreign'];
        let index = 0;
        const button = document.querySelector('button');
        const spinner = document.querySelector('[role="progressbar"]');
        button.onclick = async () => {
          spinner.hidden = false;
          await fetch('/apis/v2/artifacts?page_token=' + (index + 1)).then(response => response.json());
          if (!${repeat}) ++index;
          const row = document.querySelector('main');
          row.dataset.rowId = index + 1;
          row.textContent = labels[index];
          button.disabled = index === labels.length - 1;
          spinner.hidden = true;
        };
      </script>`,
  }));
  return page;
}

test('catalog finds the owner on page two and detects a foreign row on page three', async t => {
  const page = await openCatalog(t);
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

test('catalog reports a repeated page when the same rows stay visible', async t => {
  const page = await openCatalog(t, { repeat: true });
  await assert.rejects(scanCatalog(page, () => false), /Artifact catalog repeated a page/);
});
