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
