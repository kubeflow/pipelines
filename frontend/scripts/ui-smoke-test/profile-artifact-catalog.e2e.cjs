// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

// Run after upgrading the disposable cluster with the two retained fixture IDs.
// check-profile-upgrade-history.sh capture/verify checks that those runs predate the upgrade.
const assert = require('node:assert/strict');
const { readFile } = require('node:fs/promises');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { openFixturePage, startFixtureGateway } = require('./profile-auth-gateway');

const baseURL = process.env.PROFILE_UI_URL || 'http://127.0.0.1:3000';

function getProfiles() {
  const profiles = [
    { identity: process.env.PROFILE_OWNER, namespace: process.env.PROFILE_OWNER_NAMESPACE, runId: process.env.PROFILE_RUN_ID },
    { identity: process.env.PROFILE_FOREIGN, namespace: process.env.PROFILE_FOREIGN_NAMESPACE, runId: process.env.PROFILE_FOREIGN_RUN_ID },
  ];
  for (const key of ['identity', 'namespace', 'runId']) {
    for (const profile of profiles) assert.ok(profile[key], 'Missing fixture ' + key);
    assert.notEqual(profiles[0][key], profiles[1][key]);
  }
  return profiles;
}

async function readCatalog(page, namespace) {
  const artifacts = [];
  const tokens = new Set();
  let token = '';
  do {
    const params = { namespace, page_size: '100', page_token: token };
    const response = await page.request.get('/apis/v2/artifacts', { params });
    assert.equal(response.status(), 200, 'List artifacts for ' + namespace);
    const body = await response.json();
    for (const artifact of body.artifacts || []) {
      assert.equal(artifact.namespace, namespace, 'Catalog returned another profile');
      artifacts.push(artifact);
    }
    token = body.next_page_token || '';
    if (token) {
      assert.ok(!tokens.has(token), 'Catalog repeated a page token');
      tokens.add(token);
    }
  } while (token);
  return artifacts;
}

async function openDataset(page, dataset) {
  await page.goto('/#/artifacts');
  const row = page.getByTestId('table-row').filter({
    has: page.getByRole('link', { name: 'Artifact ID ' + dataset.artifact_id, exact: true }),
  });
  const rows = page.getByTestId('table-row');
  await rows.first().waitFor();
  while (!(await row.count())) {
    const first = await rows.first().getAttribute('data-row-id');
    const previous = rows.filter({ has: page.getByRole('link', { name: 'Artifact ID ' + first, exact: true }) });
    const next = page.getByRole('button', { name: 'Next page', exact: true });
    assert.ok(await next.isEnabled(), 'Dataset missing from the visible artifact catalog');
    await next.click();
    await previous.waitFor({ state: 'detached' });
    await rows.first().waitFor();
  }
  await row.getByText(dataset.uri, { exact: true }).waitFor();
  await row.getByRole('link', { name: 'Artifact ID ' + dataset.artifact_id, exact: true }).click();
  await page.getByText('Artifact details', { exact: true }).waitFor();
}

async function assertDenied(page, path) {
  const response = await page.request.get(path);
  assert.ok([403, 404].includes(response.status()), 'Expected denial for ' + path + ', got ' + response.status());
}

test('native artifact catalog preserves each profile’s historical dataset and lineage', async t => {
  const profiles = getProfiles();
  const gateway = await startFixtureGateway(baseURL, profiles.map(profile => profile.identity));
  t.after(() => gateway.close());
  const browser = await chromium.launch();
  t.after(() => browser.close());

  for (const profile of profiles) {
    await t.test(profile.namespace, async t => {
      const other = profiles.find(candidate => candidate !== profile);
      const page = await openFixturePage(browser, gateway, t, profile);
      const runPath = '/apis/v2/runs/' + encodeURIComponent(profile.runId);
      const response = await page.request.get(runPath);
      assert.equal(response.status(), 200);
      const run = await response.json();
      assert.equal(run.namespace, profile.namespace);
      assert.equal(run.run_id, profile.runId);
      assert.equal(run.state, 'SUCCEEDED');
      assert.ok(run.display_name);

      const artifacts = await readCatalog(page, profile.namespace);
      const matches = artifacts.filter(artifact => {
        if (!artifact.uri) return false;
        const parts = artifact.uri.split('/');
        return parts.includes(profile.runId) && parts.at(-1) === 'output_gcs';
      });
      assert.equal(matches.length, 1, 'Expected exactly one original output dataset');
      const dataset = matches[0];
      assert.ok(dataset.artifact_id);
      assert.ok(!artifacts.some(artifact => artifact.uri?.includes(other.runId)));

      await page.goto('/#/runs');
      const runLink = page.getByTestId('run-name-link').filter({ hasText: run.display_name });
      await runLink.click();
      await page.getByTitle('output_gcs', { exact: true }).getByTestId('artifact-icon-live').waitFor();
      await openDataset(page, dataset);
      await page.getByRole('button', { name: 'Show artifact visualization', exact: true }).click();
      const preview = page.getByRole('region', { name: 'Dataset preview' });
      await preview.getByRole('columnheader', { name: /^value\b/ }).waitFor();
      await preview.getByRole('cell', { name: run.display_name, exact: true }).waitFor();

      const downloadLink = page.locator('a[download]').and(page.getByTitle(dataset.uri, { exact: true }));
      const url = new URL(await downloadLink.getAttribute('href'), page.url());
      assert.equal(url.origin, gateway.url);
      assert.equal(url.searchParams.get('namespace'), profile.namespace);
      const [download] = await Promise.all([page.waitForEvent('download'), downloadLink.click()]);
      assert.equal(await download.failure(), null);
      assert.deepEqual(await readFile(await download.path()), Buffer.from('value\n' + run.display_name + '\n'));

      await page.getByRole('button', { name: 'Related tasks', exact: true }).click();
      for (const name of ['component1', 'component2']) {
        await page.getByRole('link', { name: run.display_name + ' · ' + name, exact: true }).waitFor();
      }

      await page.getByRole('button', { name: 'Lineage Explorer', exact: true }).click();
      const taskPaths = [];
      const taskLinks = [];
      for (const [direction, name] of [['Producer', 'component1'], ['Consumer', 'component2']]) {
        const region = page.getByRole('region', { name: direction + ' ' + name, exact: true });
        const link = region.getByRole('link', { name, exact: true });
        const target = new URL(await link.getAttribute('href'), page.url());
        const taskLink = new URL(target.hash.slice(1), gateway.url);
        assert.equal(taskLink.pathname, '/runs/details/' + profile.runId);
        const taskId = taskLink.searchParams.get('task');
        assert.ok(taskId, 'Lineage must link to a concrete task');
        const path = runPath + '/tasks/' + encodeURIComponent(taskId);
        const taskResponse = await page.request.get(path);
        assert.equal(taskResponse.status(), 200);
        assert.equal((await taskResponse.json()).task_id, taskId);
        taskPaths.push(path);
        taskLinks.push({ link, target });
      }
      assert.notEqual(taskPaths[0], taskPaths[1]);
      await taskLinks[0].link.click();
      await page.waitForURL(taskLinks[0].target.href);

      const foreign = await openFixturePage(browser, gateway, t, other);
      await assertDenied(foreign, runPath);
      await assertDenied(foreign, url.pathname + url.search);
      await assertDenied(foreign, '/apis/v2/artifacts/' + encodeURIComponent(dataset.artifact_id));
      for (const path of taskPaths) await assertDenied(foreign, path);
    });
  }
});
