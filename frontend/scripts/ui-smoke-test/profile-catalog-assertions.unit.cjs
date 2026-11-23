// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { assertProducerInCatalog, lookupReturnedRecord, scanCatalog } = require('./profile-catalog-assertions.cjs');

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
  assert.equal(await scanCatalog(page, 'artifacts', () => rows.filter({ hasText: 'owner' }).count()), true);
  await page.goto('about:blank');

  const seen = [];
  await scanCatalog(page, 'artifacts', async () => {
    seen.push(await rows.first().innerText());
    return false;
  });
  assert.deepEqual(seen, ['unrelated', 'owner', 'foreign']);
});

test('execution catalog finds the linked producer on page two', async t => {
  const browser = await chromium.launch(process.env.PROFILE_TEST_CHROME_CHANNEL ?
    { channel: process.env.PROFILE_TEST_CHROME_CHANNEL } : {});
  t.after(() => browser.close());
  const page = await browser.newPage({ baseURL: 'http://catalog.test' });
  page.setDefaultTimeout(3000);
  const row = id => `<div data-testid="table-row" data-row-id="${id}">
    <a href="/#/executions/${id}">component1</a></div>`;
  await page.route('http://catalog.test/', route => route.fulfill({
    contentType: 'text/html',
    body: `<main id="rows">${Array.from({ length: 10 }, (_, index) => row(index + 1)).join('')}</main>
      <button data-testid="next-page-btn">Next page</button>
      <template id="second">${row(11)}${row(12)}</template>
      <script>
        document.querySelector('button').onclick = event => {
          document.querySelector('#rows').replaceChildren(
            document.querySelector('#second').content.cloneNode(true));
          event.currentTarget.disabled = true;
        };
      </script>`,
  }));
  await assertProducerInCatalog(page, '12');
  assert.equal(new URL(page.url()).hash, '#/executions');
  assert.ok(await page.getByTestId('next-page-btn').isDisabled());
});

function frame(type, payload) {
  const header = Buffer.alloc(5);
  header[0] = type;
  header.writeUInt32BE(payload.length, 1);
  return Buffer.concat([header, payload]);
}

function response({ http = 200, grpc = 0, fields = [] } = {}) {
  return {
    status: () => http,
    headers: () => ({}),
    body: async () => Buffer.concat([
      frame(0x00, Buffer.from(fields)),
      frame(0x80, Buffer.from('grpc-status: ' + grpc + '\r\n')),
    ]),
  };
}

test('by-ID isolation uses the record response, including gRPC errors and empty results', async () => {
  assert.equal(await lookupReturnedRecord(response({ http: 403 })), false);
  assert.equal(await lookupReturnedRecord(response({ grpc: 5 })), false);
  assert.equal(await lookupReturnedRecord(response({ grpc: 7 })), false);
  assert.equal(await lookupReturnedRecord(response()), false);
  assert.equal(await lookupReturnedRecord(response({ fields: [0x12, 0x02, 0x08, 0x01] })), false);
  assert.equal(await lookupReturnedRecord(response({ fields: [0x0a, 0x02, 0x08, 0x01] })), true);
  await assert.rejects(lookupReturnedRecord(response({ grpc: 13 })), /Unexpected gRPC status/);
  await assert.rejects(lookupReturnedRecord(response({ fields: [0x0a, 0x02, 0x08] })), /Truncated/);
});
