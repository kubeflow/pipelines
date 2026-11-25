// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { openFixturePage, startFixtureGateway } = require('./profile-auth-gateway');

test('each profile can compare the metrics recorded before upgrade', async t => {
  const profiles = [
    {
      identity: process.env.PROFILE_OWNER,
      namespace: process.env.PROFILE_OWNER_NAMESPACE,
      runs: [process.env.PROFILE_RUN_ID, process.env.PROFILE_COMPARISON_RUN_ID],
      names: [process.env.PROFILE_RUN_NAME, process.env.PROFILE_COMPARISON_RUN_NAME],
      scores: ['1717', '1212'],
    },
    {
      identity: process.env.PROFILE_FOREIGN,
      namespace: process.env.PROFILE_FOREIGN_NAMESPACE,
      runs: [process.env.PROFILE_FOREIGN_RUN_ID, process.env.PROFILE_FOREIGN_COMPARISON_RUN_ID],
      names: [process.env.PROFILE_FOREIGN_RUN_NAME, process.env.PROFILE_FOREIGN_COMPARISON_RUN_NAME],
      scores: ['2727', '2323'],
    },
  ];

  for (const profile of profiles) {
    assert.ok([profile.identity, profile.namespace, ...profile.runs, ...profile.names].every(Boolean),
      'Missing pre-upgrade comparison fixture');
    assert.notEqual(profile.runs[0], profile.runs[1]);
  }

  const gateway = await startFixtureGateway(
    process.env.PROFILE_UI_URL || 'http://127.0.0.1:3000',
    profiles.map(profile => profile.identity),
  );
  t.after(() => gateway.close());
  const browser = await chromium.launch();
  t.after(() => browser.close());
  for (const profile of profiles) {
    await t.test(profile.namespace, async t => {
      const page = await openFixturePage(browser, gateway, t, profile);
      await page.goto('/#/runs');
      for (const runId of profile.runs) {
        const row = page.locator('[data-row-id="' + runId + '"]');
        await row.getByRole('checkbox').check();
      }
      const taskPaths = new Set(profile.runs.map(id => '/apis/v2/runs/' + encodeURIComponent(id) + '/tasks'));
      const loadedTaskPaths = new Set();
      page.on('response', response => {
        const path = new URL(response.url()).pathname;
        if (response.ok() && response.request().method() === 'GET' && taskPaths.has(path)) {
          loadedTaskPaths.add(path);
        }
      });
      await page.getByRole('button', { name: 'Compare runs', exact: true }).click();
      await page.waitForURL(url => url.hash.startsWith('#/compare?'));
      const metric = page.getByRole('row').filter({
        has: page.getByRole('cell', {
          name: /^.+ \/ profile_upgrade_score$/,
          exact: true,
        }),
      });
      await metric.waitFor({ timeout: 60000 });
      const comparison = page.getByRole('table').filter({ has: metric });
      await comparison.getByRole('cell', { name: profile.names[0], exact: true }).waitFor();
      const headings = await comparison.getByRole('row').first().getByRole('cell').allTextContents();
      assert.equal(headings[0], '', 'Expected a metric label column');
      assert.deepEqual(headings.slice(1).sort(), [...profile.names].sort(),
        'Compare must show exactly the selected runs');
      const values = await metric.getByRole('cell').allTextContents();
      for (const [i, name] of profile.names.entries()) {
        const column = headings.indexOf(name);
        assert.ok(column > 0, 'Missing historical run column ' + name);
        assert.equal(values[column], profile.scores[i]);
      }
      assert.deepEqual(loadedTaskPaths, taskPaths, 'Compare must load both runs through native task APIs');
      const other = profiles.find(entry => entry !== profile);
      for (const score of other.scores) {
        assert.equal(await metric.getByRole('cell', { name: score, exact: true }).count(), 0);
      }
    });
  }
});
