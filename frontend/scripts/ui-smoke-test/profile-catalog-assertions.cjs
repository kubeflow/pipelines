// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');

async function scanCatalog(page, visit) {
  await page.goto('/#/artifacts');
  const rows = page.getByTestId('table-row');
  const next = page.getByTestId('next-page-btn');
  const error = page.getByRole('link', { name: 'Troubleshooting guide' });
  const empty = page.getByText('No artifacts found.', { exact: true });
  const visited = new Set();

  for (;;) {
    await rows.first().or(error).or(empty).first().waitFor({ timeout: 60000 });
    await page.getByRole('progressbar').waitFor({ state: 'hidden' });
    if (await error.isVisible()) {
      throw new Error('Artifact catalog error: ' + await error.locator('../..').innerText());
    }
    if (!(await rows.count())) return false;

    const first = await rows.first().getAttribute('data-row-id');
    assert.ok(first && !visited.has(first), 'Artifact catalog repeated a page');
    visited.add(first);
    if (await visit()) return true;
    if (await next.isDisabled()) return false;

    const previous = page.locator('[data-row-id="' + first + '"]');
    await next.click();
    await previous.waitFor({ state: 'detached' });
  }
}

module.exports = { scanCatalog };
