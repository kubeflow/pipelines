// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

// Run after upgrading the disposable cluster with the two retained fixture IDs.
// check-profile-upgrade-history.sh capture/verify checks that those runs predate the upgrade.
const assert = require('node:assert/strict');
const { readFile } = require('node:fs/promises');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { startFixtureGateway } = require('./profile-auth-gateway');
const { assertProducerInCatalog, lookupReturnedRecord, scanCatalog } = require('./profile-catalog-assertions.cjs');

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

async function openPage(browser, gateway, t, { identity, namespace }) {
  const context = await browser.newContext({ baseURL: gateway.url });
  t.after(() => context.close());
  // The port-forwarded UI runs without its enclosing dashboard on the disposable cluster.
  await context.addInitScript(selected => {
    window.centraldashboard = {
      CentralDashboardEventHandler: {
        init(register) {
          const handler = {};
          register(handler);
          handler.onNamespaceSelected(selected);
        },
      },
    };
  }, namespace);
  await context.route('**/dashboard_lib.bundle.js', route =>
    route.fulfill({ status: 200, contentType: 'application/javascript', body: '' }),
  );
  const page = await context.newPage();
  await gateway.signIn(page, identity);
  return page;
}

async function openDataset(page, profile) {
  const row = page.getByTestId('table-row').filter({ hasText: profile.runId }).filter({
    has: page.getByRole('link', { name: 'output_gcs', exact: true }),
  });
  assert.ok(await scanCatalog(page, 'artifacts', () => row.count()), profile.namespace + ' catalog is missing ' + profile.runId);

  const uri = await row.getByText(/^(?:s3|minio|gs):\/\//).innerText();
  const { pathname } = new URL(uri);
  assert.ok(pathname.split('/').includes(profile.runId), 'Unexpected fixture URI: ' + uri);
  assert.ok(pathname.endsWith('/output_gcs'), 'Unexpected fixture output: ' + uri);
  const artifactId = await row.getAttribute('data-row-id');
  assert.match(artifactId || '', /^[1-9]\d*$/);
  const link = row.getByRole('link', { name: 'output_gcs', exact: true });
  assert.match(await link.getAttribute('href'), new RegExp('/artifacts/' + artifactId + '$'));
  await link.click();
  await page.locator('dd').filter({ hasText: uri }).waitFor();
  return { artifactId, uri };
}

async function checkLineage(page, { artifactId, uri }) {
  await page.getByRole('button', { name: 'Lineage Explorer', exact: true }).click();
  const executions = page.locator('.LineageExplorer > .execution');
  const producer = executions.nth(0).getByRole('link', { name: 'component1', exact: true });
  const consumer = executions.nth(1).getByRole('link', { name: 'component2', exact: true });
  await producer.waitFor();
  await consumer.waitFor();
  const producerHref = await producer.getAttribute('href');
  const consumerHref = await consumer.getAttribute('href');
  assert.match(producerHref, /\/executions\/[1-9]\d*$/);
  assert.match(consumerHref, /\/executions\/[1-9]\d*$/);
  assert.notEqual(producerHref, consumerHref);
  await producer.click();
  const outputs = page.locator('section').filter({
    has: page.getByRole('heading', { name: 'Outputs', exact: true }),
  });
  const dataset = outputs.getByRole('row').filter({ hasText: uri });
  await dataset.getByRole('cell', { name: artifactId, exact: true }).waitFor();
  assert.match(await dataset.getByRole('link', { name: 'output_gcs', exact: true }).getAttribute('href'),
    new RegExp('/artifacts/' + artifactId + '$'));
  return producerHref.split('/').pop();
}

function checkIsolation(t, exposed, description) {
  if (exposed && process.env.PROFILE_ISOLATION_MODE === 'report') {
    t.todo('Known isolation gap after #15844: ' + description);
    console.log('::warning::Known profile isolation gap: ' + description);
    return;
  }
  assert.equal(exposed, false, description);
}

async function detailExposesRecord(page, kind, id) {
  const method = kind === 'artifact' ? 'GetArtifactsByID' : 'GetExecutionsByID';
  const [response] = await Promise.all([
    page.waitForResponse(response => response.request().method() === 'POST' &&
      new URL(response.url()).pathname === '/ml_metadata.MetadataStoreService/' + method),
    page.goto('/#/' + kind + 's/' + id),
  ]);
  return lookupReturnedRecord(response);
}

test('each profile can use its retained dataset after upgrade', async t => {
  const profiles = getProfiles();
  const gateway = await startFixtureGateway(baseURL, profiles.map(profile => profile.identity));
  t.after(() => gateway.close());
  const browser = await chromium.launch();
  t.after(() => browser.close());
  for (const profile of profiles) {
    const other = profiles.find(candidate => candidate !== profile);
    await t.test(profile.namespace, async t => {
      const page = await openPage(browser, gateway, t, profile);
      const run = await page.request.get('/apis/v2beta1/runs/' + encodeURIComponent(profile.runId));
      assert.equal(run.status(), 200);
      const record = await run.json();
      assert.equal(record.namespace, profile.namespace);
      assert.equal(record.run_id, profile.runId);
      assert.equal(record.state, 'SUCCEEDED');
      assert.ok(record.display_name);

      await t.test('read the exact original bytes from the run download link', async t => {
        await page.goto('/#/runs');
        const runLink = page.getByTestId('run-name-link').filter({ hasText: record.display_name });
        const runPath = '/runs/details/' + encodeURIComponent(profile.runId);
        assert.ok((await runLink.getAttribute('href')).endsWith(runPath), 'Historical run link changed');
        await runLink.click();
        await page.getByTitle('output_gcs', { exact: true }).getByTestId('artifact-icon-live').waitFor();
        await page.getByTitle('component1', { exact: true }).getByRole('button').click();
        await page.getByRole('button', { name: 'Input/Output', exact: true }).click();
        // The run graph can render before its namespace is available. Only
        // click the download link once it represents the selected profile.
        const namespace = encodeURIComponent(profile.namespace);
        const link = page.locator('a[download][href*="namespace=' + namespace + '&"]')
          .filter({ hasText: profile.runId }).filter({ hasText: /\/output_gcs$/ });
        const url = new URL(await link.getAttribute('href'), page.url());
        assert.equal(url.origin, new URL(gateway.url).origin);
        const startDownload = async () => {
          const [download] = await Promise.all([page.waitForEvent('download'), link.click()]);
          const target = new URL(download.url());
          assert.equal(target.origin, url.origin);
          assert.equal(target.pathname, url.pathname);
          assert.equal(target.searchParams.get('namespace'), profile.namespace);
          assert.equal(target.searchParams.get('key'), url.searchParams.get('key'));
          return download;
        };
        let download = await startDownload();
        let failure = await download.failure();
        if (failure === 'canceled') {
          t.diagnostic('Chromium canceled the download. Retrying the same link once.');
          download = await startDownload();
          failure = await download.failure();
        }
        assert.equal(failure, null, 'Browser download did not complete');
        assert.deepEqual(await readFile(await download.path()), Buffer.from('value\n' + record.display_name + '\n'));
      });

      const dataset = await openDataset(page, profile);
      await t.test('render the producer-recorded dataset sample', async () => {
        await page.getByRole('button', { name: 'Show artifact visualization', exact: true }).click();
        const preview = page.getByRole('region', { name: 'Dataset preview' });
        await preview.getByRole('columnheader', { name: /^value\b/ }).waitFor();
        await preview.getByRole('cell', { name: record.display_name, exact: true }).waitFor();
      });
      let producerId;
      await t.test('follow both lineage edges to the original execution', async () => {
        producerId = await checkLineage(page, dataset);
        await assertProducerInCatalog(page, producerId);
      });
      const otherPage = await openPage(browser, gateway, t, other);
      await t.test('exclude the first profile from the second profile catalog', async t => {
        const rows = otherPage.getByTestId('table-row');
        let ownVisible = false;
        let foreignVisible = false;
        await scanCatalog(otherPage, 'artifacts', async () => {
          ownVisible = ownVisible || (await rows.filter({ hasText: other.runId }).count()) > 0;
          foreignVisible = foreignVisible || (await rows.filter({ hasText: profile.runId }).count()) > 0;
          return false;
        });
        assert.ok(ownVisible, 'Expected the second profile to see its own dataset');
        checkIsolation(t, foreignVisible, 'catalog includes a dataset from another profile');
      });
      await t.test('deny artifact details to the second profile', async t => {
        checkIsolation(t, await detailExposesRecord(otherPage, 'artifact', dataset.artifactId),
          'artifact lookup returns a record to another profile');
      });
      if (producerId) {
        await t.test('deny execution details to the second profile', async t => {
          checkIsolation(t, await detailExposesRecord(otherPage, 'execution', producerId),
            'execution lookup returns a record to another profile');
        });
      }
    });
  }
});
