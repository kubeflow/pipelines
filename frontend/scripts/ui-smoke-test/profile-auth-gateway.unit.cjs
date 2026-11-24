// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

const assert = require('node:assert/strict');
const { test } = require('node:test');
const { chromium } = require('playwright');
const { startFixtureGateway } = require('./profile-auth-gateway.js');

test('a Kubernetes username can sign in through the fixture form', async t => {
  const identity = 'system:serviceaccount:team-a:default-editor';
  const gateway = await startFixtureGateway('http://127.0.0.1:1', [identity, 'other@example.com']);
  t.after(() => gateway.close());
  const browser = await chromium.launch(process.env.PROFILE_TEST_CHROME_CHANNEL ?
    { channel: process.env.PROFILE_TEST_CHROME_CHANNEL } : {});
  t.after(() => browser.close());
  const page = await browser.newPage();
  page.setDefaultTimeout(5000);

  const session = gateway.url + '/__profile_fixture/session';
  assert.equal((await page.request.get(session)).status(), 401);
  await gateway.signIn(page, identity);
  assert.equal(await page.locator('body').innerText(), 'Signed in');
  assert.equal((await page.request.get(session)).status(), 200);
});
