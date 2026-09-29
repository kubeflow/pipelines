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
  prepareNativeSafariTap,
  safariKeyboardDoneSelector,
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
  closeCount = 1,
  keyboard = false,
  toolbarDoneCount = 0,
  dismissError,
  restoreError,
} = {}) {
  const calls = [];
  const evidence = [];
  const snapshots = [];
  const element = (id) => ({ 'element-6066-11e4-a52e-4f735466cecf': id });
  const command = async (method, path, body) => {
    calls.push({ method, path, body });
    if (method === 'GET' && path.endsWith('/context')) return 'WEBVIEW_1';
    if (method === 'POST' && path.endsWith('/context')) {
      if (body.name === 'WEBVIEW_1' && restoreError) throw restoreError;
      return null;
    }
    if (path.endsWith('/source')) return '<native-source />';
    if (path.endsWith('/elements') && body.value === safariKeyboardDoneSelector)
      return Array.from({ length: toolbarDoneCount }, (_, i) => element(`done-${i}`));
    if (path.endsWith('/elements') && body.using === 'xpath')
      return Array.from({ length: closeCount }, (_, i) => element(`close-${i}`));
    if (path.endsWith('/elements')) return tip ? [element('tip')] : [];
    if (path.endsWith('/element/done-0/click')) {
      keyboard = false;
      return null;
    }
    if (path.endsWith('/click')) {
      assert.match(path, /element\/close-0\/click$/);
      tip = false;
      return null;
    }
    if (body?.script === 'mobile: isKeyboardShown') return keyboard;
    if (body?.script === 'mobile: hideKeyboard') {
      if (dismissError) throw dismissError;
      keyboard = false;
      return null;
    }
    assert.fail(`Unexpected native preparation command: ${method} ${path}`);
  };
  return {
    calls,
    evidence,
    snapshots,
    run: () =>
      prepareNativeSafariTap(command, 'fixture', evidence, async (source, label) =>
        snapshots.push({ source, label }),
      ),
  };
}

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
  const lookup = fixture.calls.find(({ body }) => body?.using === 'xpath');
  assert.match(lookup.body.value, /ancestor-or-self/);
  assert.equal(fixture.evidence[0].actions.length, 2);
  assert.equal(fixture.snapshots[0].label, 'safari-tip');
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
