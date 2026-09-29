/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// Browser-free contract checks. These do not launch or install browsers.
import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:net';
import { createServer as createHttpServer } from 'node:http';
import { requestWebDriver } from './ui-modernization-native-http.mjs';
import {
  clickNativeSafariLink,
  clickNativeSafariRadio,
  nativeSafariRadioSelector,
  inspectMobileTarget,
  planMobileGraphPan,
  panNativeSafariGraph,
  nativeSafariGraphSelector,
  scrollMobileTargetIntoView,
  nativeSafariLinkSelector,
  prepareNativeSafariTap,
  safariKeyboardDoneSelector,
  safariActiveAddressSelector,
  safariStartPageCloseSelector,
} from './ui-modernization-native-safari.mjs';
import { JSDOM } from 'jsdom';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';
import { test } from 'node:test';

const frontend = fileURLToPath(new URL('../', import.meta.url));
for (const script of ['run-native-qualification.mjs', 'ui-modernization-browser-floor.mjs']) {
  test(`${script} rejects a developer workstation before browser launch`, () => {
    const result = spawnSync(process.execPath, [`scripts/${script}`], {
      cwd: frontend,
      env: { ...process.env, CI: 'true', GITHUB_ACTIONS: '', RUNNER_ENVIRONMENT: '' },
      encoding: 'utf8',
      timeout: 10000,
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /requires GitHub Actions/);
  });
}

test('missing installer inputs replace stale success with failed lifecycle evidence', async () => {
  const output = await mkdtemp(join(tmpdir(), 'kfp-native-contract-'));
  try {
    await writeFile(join(output, 'result.json'), '{"status":"passed"}');
    const result = spawnSync(process.execPath, ['scripts/run-native-qualification.mjs'], {
      cwd: frontend,
      env: {
        ...process.env,
        CI: 'true',
        GITHUB_ACTIONS: 'true',
        RUNNER_ENVIRONMENT: 'github-hosted',
        RUNNER_TEMP: output,
        KFP_BROWSER_FLOOR_OUTPUT: output,
        KFP_FIREFOX_BINARY: '',
        KFP_GECKODRIVER_BINARY: '',
        KFP_BROWSER_FLOOR_VERSION: '',
      },
      encoding: 'utf8',
      timeout: 10000,
    });
    assert.notEqual(result.status, 0);
    const evidence = JSON.parse(await readFile(join(output, 'lifecycle.json'), 'utf8'));
    assert.equal(evidence.status, 'failed');
    assert.match(evidence.error, /Installer must supply/);
    await assert.rejects(readFile(join(output, 'result.json')), { code: 'ENOENT' });
  } finally {
    await rm(output, { recursive: true, force: true });
  }
});

test(
  'native fixture serves real fixed data and rejects/reports mutations and missing assets',
  { timeout: 30000 },
  async () => {
    const reservation = createServer();
    await new Promise((resolve) => reservation.listen(0, '127.0.0.1', resolve));
    const port = reservation.address().port;
    await new Promise((resolve) => reservation.close(resolve));
    const origin = `http://127.0.0.1:${port}`;
    let logs = '';
    const child = spawn(
      process.execPath,
      ['--import', 'tsx', 'scripts/ui-modernization-native-server.ts'],
      {
        cwd: frontend,
        env: { ...process.env, CI: 'true', KFP_BROWSER_FLOOR_PORT: String(port) },
        stdio: ['ignore', 'pipe', 'pipe'],
      },
    );
    child.stdout.on('data', (data) => {
      logs += data;
    });
    child.stderr.on('data', (data) => {
      logs += data;
    });
    try {
      let ready = false;
      for (let attempt = 0; attempt < 100; attempt++) {
        assert.equal(child.exitCode, null, logs);
        try {
          ready = (await fetch(`${origin}/__qualification`)).ok;
        } catch {}
        if (ready) break;
        await delay(100);
      }
      assert.ok(ready, logs);
      const runs = await (await fetch(`${origin}/apis/v2beta1/runs`)).json();
      assert.equal(runs.runs.length, 4);
      assert.ok(runs.runs.some((run) => run.run_id === 'e0115ac1-0479-4194-a22d-01e65e09a32b'));
      const taskPath = `${origin}/apis/v2beta1/runs/e0115ac1-0479-4194-a22d-01e65e09a32b/tasks`;
      const taskList = await (await fetch(taskPath)).json();
      for (const task of taskList.tasks) {
        const detail = await fetch(`${taskPath}/${task.task_id}`);
        assert.equal(detail.status, 200);
        assert.deepEqual(await detail.json(), task, 'detail and list must share fixture identity');
      }
      assert.equal((await fetch(`${taskPath}/missing-task`)).status, 404);
      assert.equal(
        (await fetch(`${origin}/apis/v2beta1/runs/missing-run/tasks/mock-task-producer`)).status,
        404,
      );
      const artifacts = await (await fetch(`${origin}/apis/v2beta1/artifacts`)).json();
      assert.equal(artifacts.artifacts[0].artifact_id, 'mock-artifact-1');
      assert.equal(
        (await fetch(`${origin}/apis/v2beta1/runs`, { method: 'POST', body: '{}' })).status,
        405,
      );
      assert.equal((await fetch(`${origin}/static/missing-native-contract.js`)).status, 404);
      const evidence = await (await fetch(`${origin}/__qualification`)).json();
      assert.deepEqual(evidence.mutations, [{ method: 'POST', path: '/apis/v2beta1/runs' }]);
      assert.deepEqual(evidence.missingAssets, ['/static/missing-native-contract.js']);
    } finally {
      child.kill('SIGTERM');
      for (
        let attempt = 0;
        attempt < 30 && child.exitCode === null && child.signalCode === null;
        attempt++
      )
        await delay(100);
      if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
    }
  },
);

test('WebDriver transport honors command deadlines for delayed headers and bodies', async () => {
  const server = createHttpServer(async (request, response) => {
    if (request.url === '/delayed-headers') {
      await delay(80);
      response.end(JSON.stringify({ value: { sessionId: 'fixture-session' } }));
    } else if (request.url === '/wda-status') {
      response.end(JSON.stringify({ sessionId: 'native-session', value: { ready: true } }));
    } else if (request.url === '/delayed-body') {
      response.writeHead(200, { 'content-type': 'application/json' });
      response.write('{"value":');
      await delay(150);
      response.end('null}');
    } else {
      response.writeHead(500, { 'content-type': 'application/json' });
      response.end(
        JSON.stringify({ value: { error: 'session not created', message: 'fixture WDA failure' } }),
      );
    }
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  try {
    assert.deepEqual(
      await requestWebDriver(`${base}/delayed-headers`, {
        method: 'POST',
        body: {},
        timeout: 1000,
      }),
      { sessionId: 'fixture-session' },
    );
    assert.deepEqual(
      await requestWebDriver(`${base}/wda-status`, {
        method: 'GET',
        timeout: 1000,
        envelope: true,
      }),
      { sessionId: 'native-session', value: { ready: true } },
    );
    for (const path of ['/delayed-headers', '/delayed-body']) {
      await assert.rejects(
        requestWebDriver(`${base}${path}`, { method: 'POST', timeout: 20 }),
        (error) => {
          assert.match(error.message, /POST \/delayed-(headers|body)/);
          assert.match(error.message, /ABORT_ERR/);
          assert.match(error.message, /TimeoutError/);
          assert.ok(error.cause);
          return true;
        },
      );
    }
    await assert.rejects(
      requestWebDriver(`${base}/webdriver-error`, { method: 'POST', timeout: 1000 }),
      /session not created: fixture WDA failure/,
    );
  } finally {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  }
});

function safariPreparationFixture({
  tip = false,
  startPageTip = false,
  closeCount = 1,
  keyboard = false,
  toolbarDoneCount = 0,
  keyboardDismissesStartPage = false,
  dismissError,
  restoreError,
  activeAddressCount = 0,
  currentUrl = 'http://127.0.0.1:4174/#/runs/details/current?tab=graph',
  actualUrl,
  addressError,
} = {}) {
  const calls = [];
  const evidence = [];
  const snapshots = [];
  let addressCompleted = false;
  const element = (id) => ({ 'element-6066-11e4-a52e-4f735466cecf': id });
  const command = async (method, path, body) => {
    calls.push({ method, path, body });
    if (method === 'GET' && path.endsWith('/url'))
      return addressCompleted ? actualUrl || currentUrl : currentUrl;
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_1';
    if (method === 'POST' && path.endsWith('/context')) {
      if (body.name === 'WEBVIEW_1' && restoreError) throw restoreError;
      return null;
    }
    if (path.endsWith('/source')) return '<native-source />';
    if (path.endsWith('/elements') && body.value === safariActiveAddressSelector)
      return Array.from({ length: activeAddressCount }, (_, i) => element(`address-${i}`));
    if (path.endsWith('/element/address-0/clear')) return null;
    if (path.endsWith('/element/address-0/value')) {
      if (addressError) throw addressError;
      activeAddressCount = 0;
      addressCompleted = true;
      return null;
    }
    if (path.endsWith('/elements') && body.value === safariKeyboardDoneSelector)
      return Array.from({ length: toolbarDoneCount }, (_, i) => element(`done-${i}`));
    if (path.endsWith('/elements') && body.using === 'xpath')
      return Array.from({ length: closeCount }, (_, i) => element(`close-${i}`));
    if (path.endsWith('/elements')) {
      const visible = body.value.includes('onboardingButton-CustomizeStartPage')
        ? startPageTip
        : tip;
      return visible ? [element('tip')] : [];
    }
    if (path.endsWith('/element/done-0/click')) {
      keyboard = false;
      return null;
    }
    if (path.endsWith('/click')) {
      assert.match(path, /element\/close-0\/click$/);
      if (tip) tip = false;
      else startPageTip = false;
      return null;
    }
    if (body?.script === 'mobile: isKeyboardShown') return keyboard;
    if (body?.script === 'mobile: hideKeyboard') {
      if (dismissError) throw dismissError;
      keyboard = false;
      if (keyboardDismissesStartPage) startPageTip = false;
      return null;
    }
    assert.fail(`Unexpected native preparation command: ${method} ${path}`);
  };
  return {
    calls,
    evidence,
    snapshots,
    run: () =>
      prepareNativeSafariTap(
        command,
        'fixture',
        evidence,
        async (source, label) => snapshots.push({ source, label }),
        'http://127.0.0.1:4174',
      ),
  };
}

test('Safari active Address selector matches captured iPad editor and excludes nested/page-owned fields', () => {
  // Browser-owned address subtree from the eleventh hosted iPad failure.
  const address = `<XCUIElementTypeTextField value="Search or enter website" name="SearchFieldItemView?isActive=true&amp;UUID=34D7ACB3-49A3-4D89-96E6-B85EDF67DE51&amp;isPinned=false&amp;isDistractionControlOverlayUp=false" label="Address" enabled="true" visible="true" accessible="true" x="230" y="32" width="360" height="44" placeholderValue="Search or enter website">
  <XCUIElementTypeTextField value="‎127.0.0.1" name="TabBarItemTitleContainer" label="Address" enabled="true" visible="true" x="230" y="32" width="360" height="44" />
  </XCUIElementTypeTextField>`;
  const dom = new JSDOM(
    `<AppiumAUT><XCUIElementTypeApplication name="Safari">
    ${address}<XCUIElementTypeWebView>${address}</XCUIElementTypeWebView>
    ${address.replace('isActive=true', 'isActive=false')}
    ${address.replace('isActive=true', 'isActive=trueOther')}
    ${address.replace('visible="true"', 'visible="false"')}
    </XCUIElementTypeApplication></AppiumAUT>`,
    { contentType: 'text/xml' },
  );
  try {
    const { document, XPathResult } = dom.window;
    const matches = document.evaluate(
      safariActiveAddressSelector,
      document,
      null,
      XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
      null,
    );
    assert.equal(matches.snapshotLength, 1);
    assert.equal(matches.snapshotItem(0).getAttribute('x'), '230');
    assert.equal(matches.snapshotItem(0).getAttribute('y'), '32');
  } finally {
    dom.window.close();
  }
});

test('Safari completes only its active browser Address with the exact current fixture route and native Return', async () => {
  const currentUrl = 'http://127.0.0.1:4174/prefix/?namespace=team#/runs/details/current';
  const fixture = safariPreparationFixture({ activeAddressCount: 1, currentUrl });
  await fixture.run();
  const writes = fixture.calls.filter(({ path }) =>
    /\/element\/address-0\/(clear|value)$/.test(path),
  );
  assert.equal(writes.length, 2);
  assert.ok(writes[0].path.endsWith('/clear'));
  assert.deepEqual(writes[1].body, { text: `${currentUrl}\n` });
  assert.deepEqual(
    fixture.snapshots.map(({ label }) => label),
    ['safari-address-edit', 'safari-address-completed'],
  );
  assert.equal(fixture.evidence[0].addressCompletion.actualUrl, currentUrl);
  assert.equal(fixture.evidence[0].status, 'passed');
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('Safari refuses ambiguous or non-fixture address completion before clearing or typing', async () => {
  for (const options of [
    { activeAddressCount: 2 },
    { currentUrl: 'https://example.org/#/runs' },
    { currentUrl: 'http://127.0.0.1:9999/#/runs' },
    { currentUrl: 'https://127.0.0.1:4174/#/runs' },
    { currentUrl: 'http://user:secret@127.0.0.1:4174/#/runs' },
  ]) {
    const fixture = safariPreparationFixture({ activeAddressCount: 1, ...options });
    await assert.rejects(fixture.run(), /ambiguous|loopback|origin|HTTP|credentials/);
    assert.equal(
      fixture.calls.some(({ path }) => /\/(clear|value)$/.test(path)),
      false,
    );
    assert.equal(fixture.snapshots.at(-1).label, 'safari-preparation-failed');
    assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
  }
});

test('Safari address completion preserves native failures and rejects an unexpected resulting route', async () => {
  const error = new Error('native address typing failed');
  const broken = safariPreparationFixture({
    activeAddressCount: 1,
    addressError: error,
    restoreError: new Error('context restore failed'),
  });
  await assert.rejects(broken.run(), (observed) => observed === error);
  assert.match(broken.evidence[0].restoreError, /context restore failed/);
  const redirected = safariPreparationFixture({
    activeAddressCount: 1,
    actualUrl: 'http://127.0.0.1:4174/#/wrong',
  });
  await assert.rejects(redirected.run(), /changed the current fixture route/);
  assert.equal(redirected.evidence[0].status, 'failed');
  assert.deepEqual(redirected.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('native Safari preparation never clicks Close when the known onboarding tip is absent', async () => {
  const fixture = safariPreparationFixture();
  await fixture.run();
  assert.equal(
    fixture.calls.some(({ path }) => path.endsWith('/click')),
    false,
  );
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
  assert.equal(fixture.evidence[0].status, 'passed');
});

test('native Safari preparation scopes dismissal to the known tip and hides the keyboard', async () => {
  const fixture = safariPreparationFixture({ tip: true, keyboard: true });
  await fixture.run();
  const lookup = fixture.calls.find(
    ({ body }) => body?.using === 'xpath' && body.value.includes('View Bookmarks'),
  );
  assert.match(lookup.body.value, /ancestor-or-self/);
  assert.equal(fixture.evidence[0].actions.length, 2);
  assert.equal(fixture.snapshots[0].label, 'safari-bookmarks-tip');
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('native Safari preparation rejects ambiguous close controls and retains native diagnostics', async () => {
  const fixture = safariPreparationFixture({ tip: true, closeCount: 2 });
  await assert.rejects(fixture.run(), /one identifiable close control/);
  assert.equal(
    fixture.calls.some(({ path }) => path.endsWith('/click')),
    false,
  );
  assert.equal(fixture.snapshots.at(-1).label, 'safari-preparation-failed');
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('native Safari keyboard failure restores web context and preserves the original failure', async () => {
  const error = new Error('keyboard could not be hidden');
  const fixture = safariPreparationFixture({
    keyboard: true,
    dismissError: error,
    restoreError: new Error('restore failed'),
  });
  await assert.rejects(fixture.run(), (observed) => observed === error);
  assert.equal(fixture.evidence[0].status, 'failed');
  assert.match(fixture.evidence[0].restoreError, /restore failed/);
  assert.equal(fixture.snapshots.at(-1).label, 'safari-preparation-failed');
});

test('native Safari uses its form-toolbar Done without calling keyboard-only dismissal', async () => {
  const fixture = safariPreparationFixture({ keyboard: true, toolbarDoneCount: 1 });
  await fixture.run();
  assert.equal(
    fixture.calls.some(({ path }) => path.endsWith('/element/done-0/click')),
    true,
  );
  assert.equal(
    fixture.calls.some(({ body }) => body?.script === 'mobile: hideKeyboard'),
    false,
  );
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('native Safari refuses ambiguous form-toolbar Done controls', async () => {
  const fixture = safariPreparationFixture({ keyboard: true, toolbarDoneCount: 2 });
  await assert.rejects(fixture.run(), /ambiguous Done controls/);
  assert.equal(
    fixture.calls.some(({ path }) => path.endsWith('/click')),
    false,
  );
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('Safari Done selector matches captured native hierarchy and excludes page-owned and unrelated Done buttons', () => {
  // Native form accessory subtree captured by hosted iPhone qualification.
  const toolbar = `<XCUIElementTypeToolbar type="XCUIElementTypeToolbar" name="Toolbar" label="Toolbar" enabled="true" visible="true" accessible="false" x="0" y="508" width="402" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="0" y="508" width="402" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="16" y="508" width="370" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="16" y="508" width="370" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="false" accessible="false" x="16" y="508" width="0" height="0" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="16" y="508" width="370" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="16" y="508" width="51" height="48" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="21" y="513" width="41" height="38" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="21" y="513" width="41" height="38" index="0" traits="">
<XCUIElementTypeButton type="XCUIElementTypeButton" name="Previous" label="Previous" enabled="true" visible="true" accessible="true" x="21" y="513" width="41" height="38" index="0" traits="Button" />
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="70" y="508" width="51" height="48" index="1" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="75" y="513" width="41" height="38" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="75" y="513" width="41" height="38" index="0" traits="">
<XCUIElementTypeButton type="XCUIElementTypeButton" name="Next" label="Next" enabled="true" visible="true" accessible="true" x="75" y="513" width="41" height="38" index="0" traits="Button" />
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="336" y="508" width="50" height="48" index="2" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="341" y="513" width="40" height="38" index="0" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="341" y="513" width="40" height="38" index="0" traits="">
<XCUIElementTypeButton type="XCUIElementTypeButton" name="Done" label="Done" enabled="true" visible="true" accessible="true" x="341" y="513" width="40" height="38" index="0" traits="Button" />
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeOther>
</XCUIElementTypeToolbar>`;
  const dom = new JSDOM(
    `<AppiumAUT><XCUIElementTypeApplication name="Safari">
    ${toolbar}
    <XCUIElementTypeWebView>${toolbar}</XCUIElementTypeWebView>
    <XCUIElementTypeToolbar visible="true"><XCUIElementTypeButton name="Done" visible="true" enabled="true" /></XCUIElementTypeToolbar>
    <XCUIElementTypeButton name="Done" visible="true" enabled="true" />
  </XCUIElementTypeApplication></AppiumAUT>`,
    { contentType: 'text/xml' },
  );
  try {
    const { document, XPathResult } = dom.window;
    const matches = document.evaluate(
      safariKeyboardDoneSelector,
      document,
      null,
      XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
      null,
    );
    assert.equal(matches.snapshotLength, 1);
    assert.equal(matches.snapshotItem(0).getAttribute('name'), 'Done');
    assert.equal(matches.snapshotItem(0).getAttribute('x'), '341');
    assert.equal(matches.snapshotItem(0).getAttribute('y'), '513');
  } finally {
    dom.window.close();
  }
});

test('native Safari dismisses the identified Start Page onboarding without tapping Customize', async () => {
  const fixture = safariPreparationFixture({ startPageTip: true });
  await fixture.run();
  assert.equal(
    fixture.calls.some(({ body }) => body?.value === safariStartPageCloseSelector),
    true,
  );
  assert.deepEqual(fixture.evidence[0].actions, [
    'dismissed Safari Start Page tip through native accessibility',
  ]);
  assert.deepEqual(fixture.calls.at(-1).body, { name: 'WEBVIEW_1' });
});

test('Start Page close selector matches captured native card and excludes unrelated close buttons', () => {
  // Native onboarding card captured by hosted iPad qualification.
  const card = `<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="56" y="120" width="707" height="181" index="1" traits="">
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="85" y="145" width="301" height="131" index="0" traits="">
<XCUIElementTypeImage type="XCUIElementTypeImage" name="SafariOnboardingCustomizeStartPage" enabled="true" visible="true" accessible="false" x="85" y="145" width="301" height="131" index="0" traits="Image" />
</XCUIElementTypeOther>
<XCUIElementTypeOther type="XCUIElementTypeOther" enabled="true" visible="true" accessible="false" x="397" y="140" width="301" height="141" index="1" traits="">
<XCUIElementTypeStaticText type="XCUIElementTypeStaticText" value="Start Page" name="Start Page" label="Start Page" enabled="true" visible="true" accessible="true" x="397" y="140" width="84" height="21" index="0" traits="StaticText" />
<XCUIElementTypeStaticText type="XCUIElementTypeStaticText" value="Customize your wallpaper and sections that appear when creating new tabs." name="Customize your wallpaper and sections that appear when creating new tabs." label="Customize your wallpaper and sections that appear when creating new tabs." enabled="true" visible="true" accessible="true" x="397" y="184" width="301" height="39" index="1" traits="StaticText" />
<XCUIElementTypeButton type="XCUIElementTypeButton" name="onboardingButton-CustomizeStartPage" label="Customize Start Page" enabled="true" visible="true" accessible="true" x="397" y="246" width="197" height="35" index="2" traits="Button">
<XCUIElementTypeStaticText type="XCUIElementTypeStaticText" value="Customize Start Page" name="Customize Start Page" label="Customize Start Page" enabled="true" visible="true" accessible="false" x="409" y="253" width="173" height="21" index="0" traits="StaticText" />
</XCUIElementTypeButton>
</XCUIElementTypeOther>
<XCUIElementTypeButton type="XCUIElementTypeButton" name="close" label="close" enabled="true" visible="true" accessible="true" x="723" y="136" width="24" height="23" index="2" traits="Button" />
</XCUIElementTypeOther>`;
  const dom = new JSDOM(
    `<AppiumAUT><XCUIElementTypeApplication name="Safari">
    ${card}
    <XCUIElementTypeWebView>${card}</XCUIElementTypeWebView>
    <XCUIElementTypeButton name="close" label="close" enabled="true" visible="true" />
  </XCUIElementTypeApplication></AppiumAUT>`,
    { contentType: 'text/xml' },
  );
  try {
    const { document, XPathResult } = dom.window;
    const matches = document.evaluate(
      safariStartPageCloseSelector,
      document,
      null,
      XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
      null,
    );
    assert.equal(matches.snapshotLength, 1);
    assert.equal(matches.snapshotItem(0).getAttribute('x'), '723');
    assert.equal(matches.snapshotItem(0).getAttribute('y'), '136');
  } finally {
    dom.window.close();
  }
});

test('native Safari retains separate evidence for multiple onboarding tips', async () => {
  const fixture = safariPreparationFixture({ tip: true, startPageTip: true });
  await fixture.run();
  assert.deepEqual(
    fixture.snapshots.map(({ label }) => label),
    ['safari-bookmarks-tip', 'safari-start-page-tip'],
  );
  assert.equal(fixture.evidence[0].actions.length, 2);
});

test('native link selector resolves captured Link without its identical StaticText or browser chrome', () => {
  // Actual iPhone Safari subtree after the hosted Appium coordinate mis-tap.
  const link = `<XCUIElementTypeLink type="XCUIElementTypeLink" name="v2-xgboost-ilbo" label="v2-xgboost-ilbo" enabled="true" visible="true" accessible="false" x="106" y="424" width="120" height="22" index="0" traits="Link">
    <XCUIElementTypeStaticText type="XCUIElementTypeStaticText" value="v2-xgboost-ilbo" name="v2-xgboost-ilbo" label="v2-xgboost-ilbo" enabled="true" visible="true" accessible="true" x="106" y="424" width="120" height="22" index="0" traits="Link, StaticText" />
  </XCUIElementTypeLink>`;
  const dom = new JSDOM(
    `<AppiumAUT><XCUIElementTypeApplication>${link}<XCUIElementTypeWebView>${link}
    <XCUIElementTypeLink name="v2-xgboost-ilbo" visible="false" enabled="true" />
    <XCUIElementTypeLink name="v2-xgboost-ilbo" visible="true" enabled="false" />
    <XCUIElementTypeLink name="A &quot;quoted&quot; link's name" visible="true" enabled="true" />
    </XCUIElementTypeWebView></XCUIElementTypeApplication></AppiumAUT>`,
    { contentType: 'text/xml' },
  );
  try {
    const { document, XPathResult } = dom.window;
    for (const name of ['v2-xgboost-ilbo', 'A "quoted" link\'s name']) {
      const matches = document.evaluate(
        nativeSafariLinkSelector(name),
        document,
        null,
        XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null,
      );
      assert.equal(matches.snapshotLength, 1);
      assert.equal(matches.snapshotItem(0).tagName, 'XCUIElementTypeLink');
      assert.equal(matches.snapshotItem(0).getAttribute('name'), name);
    }
  } finally {
    dom.window.close();
  }
});

for (const count of [0, 1, 2]) {
  test(`native link click requires one typed match and restores context (${count} matches)`, async () => {
    const calls = [];
    const evidence = [];
    const command = async (method, path, body) => {
      calls.push({ method, path, body });
      if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_7';
      if (path.endsWith('/elements'))
        return Array.from({ length: count }, (_, i) => ({
          'element-6066-11e4-a52e-4f735466cecf': `native-link-${i}`,
        }));
      return null;
    };
    const exercise = clickNativeSafariLink(command, 'fixture', 'v2-xgboost-ilbo', evidence);
    if (count === 1) await exercise;
    else await assert.rejects(exercise, /exactly one visible matching Link/);
    assert.deepEqual(
      calls.filter(({ path }) => path.endsWith('/click')).map(({ path }) => path),
      count === 1 ? ['/session/fixture/element/native-link-0/click'] : [],
    );
    assert.deepEqual(calls.at(-1).body, { name: 'WEBVIEW_7' });
    assert.equal(evidence[0].status, count === 1 ? 'passed' : 'failed');
    assert.ok(!calls.some(({ path }) => path.endsWith('/execute/sync')));
  });
}

test('native link failure retains the original error when web context restoration also fails', async () => {
  const error = new Error('native touch rejected');
  const evidence = [];
  const command = async (method, path, body) => {
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_7';
    if (path.endsWith('/elements'))
      return [{ 'element-6066-11e4-a52e-4f735466cecf': 'native-link' }];
    if (path.endsWith('/click')) throw error;
    if (body?.name === 'WEBVIEW_7') throw new Error('context unavailable');
    return null;
  };
  await assert.rejects(
    clickNativeSafariLink(command, 'fixture', 'v2-xgboost-ilbo', evidence),
    (observed) => observed === error,
  );
  assert.equal(evidence[0].status, 'failed');
  assert.match(evidence[0].restoreError, /context unavailable/);
});

test('native radio selector resolves captured run-type controls without StaticText or unrelated toggles', () => {
  // iPad AX controls captured when Appium's label calibration tapped off-screen.
  const recurring =
    '<XCUIElementTypeOther type="XCUIElementTypeOther" value="0" name="Recurring" label="Recurring" enabled="true" visible="true" accessible="true" x="206" y="622" width="88" height="21" index="0" traits="ToggleButton" />';
  const oneOff =
    '<XCUIElementTypeOther type="XCUIElementTypeOther" value="1" name="One-off" label="One-off" enabled="true" visible="true" accessible="true" x="121" y="622" width="73" height="21" index="0" traits="ToggleButton" />';
  const dom = new JSDOM(
    `<AppiumAUT><XCUIElementTypeApplication>${recurring}
    <XCUIElementTypeWebView>${recurring}${oneOff}
      <XCUIElementTypeStaticText name="Recurring" label="Recurring" enabled="true" visible="true" traits="StaticText" />
      ${recurring.replace('visible="true"', 'visible="false"')}
      ${recurring.replace('enabled="true"', 'enabled="false"')}
      ${recurring.replace('traits="ToggleButton"', 'traits="Button"')}
      ${recurring.replaceAll('Recurring', 'Recurring runs')}
    </XCUIElementTypeWebView></XCUIElementTypeApplication></AppiumAUT>`,
    { contentType: 'text/xml' },
  );
  try {
    const { document, XPathResult } = dom.window;
    for (const [name, expectedX, expectedValue] of [
      ['Recurring', '206', '0'],
      ['One-off', '121', '1'],
    ]) {
      const matches = document.evaluate(
        nativeSafariRadioSelector(name),
        document,
        null,
        XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null,
      );
      assert.equal(matches.snapshotLength, 1);
      assert.equal(matches.snapshotItem(0).tagName, 'XCUIElementTypeOther');
      assert.equal(matches.snapshotItem(0).getAttribute('x'), expectedX);
      assert.equal(matches.snapshotItem(0).getAttribute('value'), expectedValue);
    }
  } finally {
    dom.window.close();
  }
});

for (const count of [0, 1, 2]) {
  test(`native radio clicks only one typed control and restores web context (${count} matches)`, async () => {
    const calls = [];
    const evidence = [];
    const command = async (method, path, body) => {
      calls.push({ method, path, body });
      if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_radio';
      if (path.endsWith('/elements'))
        return Array.from({ length: count }, (_, i) => ({
          'element-6066-11e4-a52e-4f735466cecf': `radio-${i}`,
        }));
      return null;
    };
    const exercise = clickNativeSafariRadio(command, 'fixture', 'Recurring', evidence);
    if (count === 1) await exercise;
    else await assert.rejects(exercise, /exactly one visible matching Radio/);
    assert.deepEqual(calls.find(({ path }) => path.endsWith('/elements')).body, {
      using: 'xpath',
      value: nativeSafariRadioSelector('Recurring'),
    });
    assert.deepEqual(
      calls.filter(({ path }) => path.endsWith('/click')).map(({ path }) => path),
      count === 1 ? ['/session/fixture/element/radio-0/click'] : [],
    );
    assert.deepEqual(calls.at(-1).body, { name: 'WEBVIEW_radio' });
    assert.equal(evidence[0].method, 'native Radio element click');
    assert.equal(evidence[0].status, count === 1 ? 'passed' : 'failed');
    assert.ok(
      !calls.some(({ path }) => path.endsWith('/execute/sync') || path.endsWith('/actions')),
    );
  });
}

test('native radio preserves action failures when restoring context also fails', async () => {
  const error = new Error('native radio touch failed');
  const evidence = [];
  const command = async (method, path, body) => {
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_radio';
    if (path.endsWith('/elements')) return [{ 'element-6066-11e4-a52e-4f735466cecf': 'radio' }];
    if (path.endsWith('/click')) throw error;
    if (body?.name === 'WEBVIEW_radio') throw new Error('restore unavailable');
    return null;
  };
  await assert.rejects(
    clickNativeSafariRadio(command, 'fixture', 'Recurring', evidence),
    (observed) => observed === error,
  );
  assert.equal(evidence[0].status, 'failed');
  assert.match(evidence[0].restoreError, /restore unavailable/);
});

test('native radio reports restore failure after a successful native action', async () => {
  const error = new Error('web context no longer available');
  const evidence = [];
  const command = async (method, path, body) => {
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_radio';
    if (path.endsWith('/elements')) return [{ 'element-6066-11e4-a52e-4f735466cecf': 'radio' }];
    if (body?.name === 'WEBVIEW_radio') throw error;
    return null;
  };
  await assert.rejects(
    clickNativeSafariRadio(command, 'fixture', 'One-off', evidence),
    (observed) => observed === error,
  );
  assert.equal(evidence[0].status, 'failed');
  assert.match(evidence[0].restoreError, /web context no longer available/);
});

test('native radio refuses a missing web context before switching or touching controls', async () => {
  const calls = [];
  const command = async (method, path) => {
    calls.push({ method, path });
    return 'NATIVE_APP';
  };
  await assert.rejects(
    clickNativeSafariRadio(command, 'fixture', 'Recurring', []),
    /requires a selected web context/,
  );
  assert.deepEqual(calls, [{ method: 'GET', path: '/session/fixture/context' }]);
});

test('native preparation dismisses address keyboard before looking for remaining Start Page chrome', async () => {
  const fixture = safariPreparationFixture({
    keyboard: true,
    startPageTip: true,
    keyboardDismissesStartPage: true,
  });
  await fixture.run();
  const dismissal = fixture.calls.findIndex(({ body }) => body?.script === 'mobile: hideKeyboard');
  const tipLookup = fixture.calls.findIndex(({ body }) =>
    body?.value?.includes('onboardingButton-CustomizeStartPage'),
  );
  assert.ok(dismissal >= 0 && tipLookup > dismissal);
  assert.equal(
    fixture.calls.some(({ path }) => path.endsWith('/click')),
    false,
  );
  assert.equal(fixture.evidence[0].status, 'passed');
});

test('mobile target readiness rejects offscreen and occluded targets, including Safari zoom offsets', () => {
  const dom = new JSDOM(
    '<button style="visibility:visible" aria-label="Task"><span>Task</span></button><div aria-label="Overlay"></div>',
  );
  try {
    const { document } = dom.window;
    const target = document.querySelector('button');
    let rect = { x: 32, y: 746, width: 125, height: 53 };
    target.getBoundingClientRect = () => rect;
    target.getClientRects = () => [rect];
    Object.defineProperty(dom.window, 'visualViewport', {
      value: { offsetLeft: 0, offsetTop: 0, width: 402, height: 714, scale: 1 },
      configurable: true,
    });
    document.elementFromPoint = () => target.querySelector('span');
    assert.equal(
      inspectMobileTarget(target).ready,
      false,
      'captured graph button is below viewport',
    );
    rect = { ...rect, y: 300 };
    assert.equal(inspectMobileTarget(target).ready, true, 'child hit belongs to the target');
    document.elementFromPoint = () => document.querySelector('div');
    assert.equal(inspectMobileTarget(target).ready, false, 'overlay intercepts target');
    document.elementFromPoint = () => target;
    Object.assign(dom.window.visualViewport, {
      offsetLeft: 57,
      offsetTop: 60,
      width: 326,
      height: 580,
      scale: 1.23134,
    });
    rect = { x: 0, y: 100, width: 50, height: 30 };
    assert.equal(
      inspectMobileTarget(target).ready,
      false,
      'center falls outside offset visual viewport',
    );
    rect = { x: 100, y: 100, width: 50, height: 30 };
    assert.equal(inspectMobileTarget(target).ready, true);
    assert.equal(inspectMobileTarget(target).viewport.scale, 1.23134);
  } finally {
    dom.window.close();
  }
});

test('mobile wrapped-link readiness hits the first real fragment instead of the union-box gap', () => {
  const dom = new JSDOM(
    '<table><tbody><tr><td><a href="#/artifacts/mock" style="visibility:visible">mock-dataset</a></td></tr></tbody></table><div aria-label="Overlay"></div>',
  );
  try {
    const { document } = dom.window;
    const target = document.querySelector('a');
    const bounding = { x: 97, y: 222.656, width: 45.5, height: 35 };
    const first = { ...bounding, width: 32, height: 15 };
    const second = { ...bounding, y: 242.656, height: 15 };
    const empty = { ...bounding, width: 0, height: 0 };
    let fragments = [empty, first, second];
    target.getBoundingClientRect = () => bounding;
    target.getClientRects = () => fragments;
    Object.defineProperty(dom.window, 'visualViewport', {
      value: { offsetLeft: 0, offsetTop: 0, width: 402, height: 714, scale: 1 },
    });
    const hitFragment = (x, y) =>
      [first, second].some(
        (rect) => x > rect.x && x < rect.x + rect.width && y > rect.y && y < rect.y + rect.height,
      )
        ? target
        : document.querySelector('td');
    document.elementFromPoint = hitFragment;
    assert.equal(
      hitFragment(bounding.x + bounding.width / 2, bounding.y + bounding.height / 2).tagName,
      'TD',
      'the hosted wrapped-link union center falls in its interline gap',
    );
    const geometry = inspectMobileTarget(target);
    assert.equal(geometry.ready, true);
    assert.deepEqual(geometry.rect, first);
    assert.deepEqual(geometry.boundingRect, bounding);
    assert.deepEqual(geometry.center, { x: 113, y: 230.156 });
    assert.equal(geometry.hitTag, 'A');
    assert.deepEqual(geometry.viewport, {
      left: 0,
      top: 0,
      width: 402,
      height: 714,
      scale: 1,
    });
    document.elementFromPoint = () => document.querySelector('div');
    assert.equal(inspectMobileTarget(target).ready, false, 'a real overlay still blocks the link');
    document.elementFromPoint = hitFragment;
    fragments = [];
    assert.equal(inspectMobileTarget(target).ready, false, 'a union box alone is not a fragment');
    fragments = [empty];
    assert.equal(inspectMobileTarget(target).ready, false, 'empty fragments are not clickable');
    fragments = [{ ...first, y: -100 }, second];
    assert.equal(
      inspectMobileTarget(target).ready,
      false,
      'an offscreen first fragment cannot be replaced by a more convenient later fragment',
    );
  } finally {
    dom.window.close();
  }
});

test('mobile graph placement scrolls the canvas, preserving hidden viewport and pane offsets', () => {
  const dom = new JSDOM(
    '<main><div class="react-flow" style="overflow:hidden"><div class="react-flow__viewport" style="overflow:hidden"><button>Task</button></div></div><input></main>',
  );
  try {
    const { document } = dom.window;
    const canvas = document.querySelector('.react-flow');
    const pane = document.querySelector('.react-flow__viewport');
    const node = document.querySelector('button');
    const input = document.querySelector('input');
    canvas.scrollTop = 7;
    canvas.scrollLeft = 9;
    pane.scrollTop = 13;
    pane.scrollLeft = 17;
    let anchor;
    canvas.scrollIntoView = () => {
      anchor = canvas;
    };
    input.scrollIntoView = () => {
      anchor = input;
    };
    node.scrollIntoView = () => {
      canvas.scrollTop = 100;
      pane.scrollLeft = 200;
      assert.fail('Node scrolling changes hidden graph ancestors');
    };
    const result = scrollMobileTargetIntoView(node);
    assert.equal(anchor, canvas);
    assert.deepEqual(result.canvasScrollBefore, { top: 7, left: 9 });
    assert.deepEqual(result.canvasScrollAfter, result.canvasScrollBefore);
    assert.equal(pane.scrollTop, 13);
    assert.equal(pane.scrollLeft, 17);
    assert.equal(scrollMobileTargetIntoView(input).anchor, 'target');
    assert.equal(anchor, input);
  } finally {
    dom.window.close();
  }
});

test('graph pan planning uses empty pane and bounded visible coordinates for a clipped node', () => {
  const dom = new JSDOM(
    '<div class="react-flow"><div class="react-flow__pane"><div class="react-flow__node"><button>Task</button></div></div></div>',
  );
  try {
    const { document } = dom.window;
    const canvas = document.querySelector('.react-flow');
    const pane = document.querySelector('.react-flow__pane');
    const target = document.querySelector('button');
    canvas.getBoundingClientRect = () => ({
      x: 80,
      y: 200,
      width: 306,
      height: 336,
      right: 386,
      bottom: 536,
    });
    const geometry = {
      center: { x: 63, y: 230 },
      viewport: { left: 70, top: 45, width: 326, height: 580 },
    };
    document.elementFromPoint = () => pane;
    const plan = planMobileGraphPan(target, geometry);
    assert.ok(plan, 'observed left-clipped node requires a native pan');
    assert.ok(plan.to.x > plan.from.x);
    for (const point of [plan.from, plan.to]) {
      assert.ok(point.x >= plan.bounds.left && point.x <= plan.bounds.right);
      assert.ok(point.y >= plan.bounds.top && point.y <= plan.bounds.bottom);
    }
    document.elementFromPoint = () => target;
    assert.equal(planMobileGraphPan(target, geometry), null, 'never initiate a node drag');
    document.elementFromPoint = () => pane;
    assert.equal(
      planMobileGraphPan(target, { ...geometry, center: { x: 230, y: 360 } }),
      null,
      'do not pan a target already inside the canvas when another overlay occludes it',
    );
    assert.equal(planMobileGraphPan(canvas, geometry), null, 'pan only graph-node targets');
  } finally {
    dom.window.close();
  }
});

test('native graph pan maps measured canvas rectangles, sends native drag, and restores context', async () => {
  const calls = [];
  const evidence = [];
  const command = async (method, path, body) => {
    calls.push({ method, path, body });
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_7';
    if (path.endsWith('/elements')) return [{ 'element-6066-11e4-a52e-4f735466cecf': 'canvas' }];
    if (path.endsWith('/rect')) return { x: 99, y: 464, width: 376, height: 415 };
    return null;
  };
  const plan = {
    canvas: { x: 80, y: 200, width: 306, height: 336 },
    from: { x: 150, y: 280 },
    to: { x: 250, y: 330 },
  };
  await panNativeSafariGraph(command, 'fixture', plan, evidence);
  const drag = calls.find(({ body }) => body?.script === 'mobile: dragFromToForDuration');
  assert.ok(drag);
  assert.equal(drag.body.args[0].duration, 0.5);
  assert.equal(drag.body.args[0].fromX, 99 + 70 * (376 / 306));
  assert.equal(drag.body.args[0].toY, 464 + 130 * (415 / 336));
  assert.equal(
    calls.find(({ path }) => path.endsWith('/elements')).body.value,
    nativeSafariGraphSelector,
  );
  assert.deepEqual(calls.at(-1).body, { name: 'WEBVIEW_7' });
  assert.equal(evidence[0].status, 'passed');
});

for (const failureMode of ['ambiguous', 'distorted', 'outside', 'driver']) {
  test(`native graph pan fails closed and restores context for ${failureMode}`, async () => {
    const calls = [];
    const evidence = [];
    const command = async (method, path, body) => {
      calls.push({ method, path, body });
      if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_7';
      if (path.endsWith('/elements'))
        return Array.from({ length: failureMode === 'ambiguous' ? 2 : 1 }, () => ({
          'element-6066-11e4-a52e-4f735466cecf': 'canvas',
        }));
      if (path.endsWith('/rect'))
        return { x: 99, y: 100, width: 300, height: failureMode === 'distorted' ? 100 : 300 };
      if (body?.script === 'mobile: dragFromToForDuration') throw new Error('native drag failed');
      return null;
    };
    const plan = {
      canvas: { x: 0, y: 0, width: 300, height: 300 },
      from: { x: 50, y: 50 },
      to: { x: failureMode === 'outside' ? 500 : 100, y: 100 },
    };
    await assert.rejects(panNativeSafariGraph(command, 'fixture', plan, evidence));
    assert.deepEqual(calls.at(-1).body, { name: 'WEBVIEW_7' });
    assert.equal(evidence[0].status, 'failed');
    assert.equal(
      calls.some(({ body }) => body?.script === 'mobile: dragFromToForDuration'),
      failureMode === 'driver',
    );
  });
}
