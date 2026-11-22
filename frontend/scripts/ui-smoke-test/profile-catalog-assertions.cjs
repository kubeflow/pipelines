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

function hasRecord(message) {
  // The by-ID responses contain length-delimited records in field 1. Artifact
  // responses may also include types in field 2, which are not artifact records.
  let offset = 0;
  let found = false;
  while (offset < message.length) {
    const tag = message[offset++];
    assert.ok(tag === 0x0a || tag === 0x12, 'Unexpected MLMD response field');
    let length = 0;
    for (let shift = 0; ; shift += 7) {
      assert.ok(offset < message.length && shift <= 28, 'Invalid MLMD response length');
      const byte = message[offset++];
      length += (byte & 0x7f) * (2 ** shift);
      if (!(byte & 0x80)) break;
    }
    offset += length;
    assert.ok(offset <= message.length, 'Truncated MLMD response record');
    found ||= tag === 0x0a;
  }
  return found;
}

async function lookupReturnedRecord(response) {
  if ([403, 404].includes(response.status())) return false;
  assert.equal(response.status(), 200, 'Unexpected HTTP status from the by-ID lookup');
  let status = response.headers()['grpc-status'];
  const body = await response.body();
  let dataFrame = false;
  let record = false;
  for (let offset = 0; offset < body.length;) {
    assert.ok(offset + 5 <= body.length, 'Truncated gRPC-Web frame header');
    const type = body[offset];
    const end = offset + 5 + body.readUInt32BE(offset + 1);
    assert.ok(end <= body.length, 'Truncated gRPC-Web frame');
    const payload = body.subarray(offset + 5, end);
    if (type === 0x00) {
      dataFrame = true;
      record = hasRecord(payload) || record;
    } else {
      assert.equal(type, 0x80, 'Unexpected gRPC-Web frame type');
      status = /^grpc-status:\s*(\d+)\s*$/im.exec(payload.toString())?.[1] ?? status;
    }
    offset = end;
  }
  if (record) return true;
  if (status === '5' || status === '7') return false;
  assert.ok(status === '0' && dataFrame, 'Unexpected gRPC status from the by-ID lookup: ' + status);
  return false;
}

module.exports = { lookupReturnedRecord, scanCatalog };
