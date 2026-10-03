/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// Firefox/geckodriver lifecycle for disposable CI runners. Does not install browsers.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { createWriteStream } from 'node:fs';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import { isAbsolute, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

assert.equal(process.env.CI, 'true', 'Native browser qualification runs only in CI');
assert.equal(process.env.GITHUB_ACTIONS, 'true', 'Native qualification requires GitHub Actions');
assert.equal(
  process.env.RUNNER_ENVIRONMENT,
  'github-hosted',
  'Native qualification requires a disposable GitHub-hosted runner',
);
assert.ok(
  process.env.RUNNER_TEMP && isAbsolute(process.env.RUNNER_TEMP),
  'Absolute RUNNER_TEMP required',
);
const frontend = fileURLToPath(new URL('../', import.meta.url));
const output = resolve(process.env.KFP_BROWSER_FLOOR_OUTPUT || 'browser-results/native-firefox');
await mkdir(output, { recursive: true });
// Remove only reports owned by this runner; setup failures must not retain a prior pass.
for (const name of ['result.json', 'lifecycle.json', 'fixture.log', 'webdriver.log', 'suite.log']) {
  await rm(join(output, name), { force: true });
}
const children = [];
const logs = [];
const lifecycle = {
  status: 'running',
  startedAt: new Date().toISOString(),
  sourceRevision: process.env.GITHUB_SHA,
};
const env = { ...process.env, KFP_WEBDRIVER_BROWSER: 'firefox', KFP_BROWSER_FLOOR_OUTPUT: output };
function launch(binary, args, name) {
  const log = createWriteStream(join(output, `${name}.log`));
  logs.push(log);
  const child = spawn(binary, args, { cwd: frontend, env, stdio: ['ignore', 'pipe', 'pipe'] });
  children.push(child);
  child.stdout.pipe(log, { end: false });
  child.stderr.pipe(log, { end: false });
  child.on('error', (error) => {
    lifecycle.error = String(error);
  });
  return child;
}
async function ready(url, child) {
  const deadline = Date.now() + 60000;
  while (Date.now() < deadline) {
    assert.ok(
      child.exitCode === null && child.signalCode === null,
      `Service exited before readiness: ${url}`,
    );
    if (lifecycle.error) throw new Error(lifecycle.error);
    try {
      if ((await fetch(url, { signal: AbortSignal.timeout(2000) })).ok) return;
    } catch {}
    await delay(200);
  }
  throw new Error(`Timed out waiting for ${url}`);
}
try {
  assert.ok(
    env.KFP_FIREFOX_BINARY && env.KFP_GECKODRIVER_BINARY && env.KFP_BROWSER_FLOOR_VERSION,
    'Installer must supply exact Firefox version, binary and geckodriver',
  );
  const fixturePort = env.KFP_BROWSER_FLOOR_PORT || '4174';
  env.KFP_BROWSER_FLOOR_URL = `http://127.0.0.1:${fixturePort}/`;
  env.KFP_WEBDRIVER_URL = 'http://127.0.0.1:4444';
  const fixture = launch(
    process.execPath,
    ['--import', 'tsx', 'scripts/ui-modernization-native-server.ts'],
    'fixture',
  );
  await ready(new URL('/__qualification', env.KFP_BROWSER_FLOOR_URL), fixture);
  const driver = launch(
    env.KFP_GECKODRIVER_BINARY,
    ['--host', '127.0.0.1', '--port', '4444'],
    'webdriver',
  );
  await ready(`${env.KFP_WEBDRIVER_URL}/status`, driver);
  const suite = launch(process.execPath, ['scripts/ui-modernization-browser-floor.mjs'], 'suite');
  const outcome = await new Promise((resolve, reject) => {
    suite.once('error', reject);
    const timeout = setTimeout(
      () => resolve({ code: null, signal: 'SUITE_TIMEOUT_15_MINUTES' }),
      15 * 60 * 1000,
    );
    suite.once('exit', (code, signal) => {
      clearTimeout(timeout);
      resolve({ code, signal });
    });
    suite.once('error', () => clearTimeout(timeout));
  });
  assert.equal(
    outcome.code,
    0,
    `Native suite failed (${outcome.code ?? outcome.signal}); inspect suite.log and result.json`,
  );
  lifecycle.status = 'passed';
} catch (error) {
  lifecycle.status = 'failed';
  lifecycle.error = error.stack || String(error);
  console.error(lifecycle.error);
  process.exitCode = 1;
} finally {
  for (const child of children.reverse()) {
    if (child.exitCode === null && child.signalCode === null) {
      child.kill('SIGTERM');
      for (
        let attempt = 0;
        attempt < 25 && child.exitCode === null && child.signalCode === null;
        attempt++
      )
        await delay(100);
      if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
    }
  }
  await Promise.all(logs.map((log) => new Promise((resolve) => log.end(resolve))));
  lifecycle.finishedAt = new Date().toISOString();
  await writeFile(join(output, 'lifecycle.json'), `${JSON.stringify(lifecycle, null, 2)}\n`);
}
