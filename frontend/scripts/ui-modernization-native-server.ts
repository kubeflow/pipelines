/*
 * Copyright 2026 The Kubeflow Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at https://www.apache.org/licenses/LICENSE-2.0
 */

// A CI-only same-origin server for the native browser qualification harness.
// Uses the repository's fixed native API fixtures and never forwards to a cluster.
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { extname, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createMockApiApp } from '../mock-backend/mock-api-app';

assert.equal(process.env.CI, 'true', 'Native browser qualification runs only in CI');
const build = fileURLToPath(new URL('../build/', import.meta.url));
const port = Number(process.env.KFP_BROWSER_FLOOR_PORT || 4174);
assert.ok(Number.isInteger(port) && port > 0 && port < 65536);
const api = createMockApiApp();
const mutations: { method: string; path: string }[] = [];
const missingAssets: string[] = [];
const mime: Record<string, string> = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.css': 'text/css',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.ico': 'image/x-icon',
  '.woff2': 'font/woff2',
  '.woff': 'font/woff',
  '.ttf': 'font/ttf',
  '.map': 'application/json',
};
const server = createServer(async (request, response) => {
  try {
    const pathname = new URL(request.url || '/', 'http://127.0.0.1').pathname;
    response.setHeader('cache-control', 'no-store');
    if (!['GET', 'HEAD'].includes(request.method || '')) {
      mutations.push({ method: request.method || '', path: pathname });
      response.writeHead(405, { 'content-type': 'application/json' });
      response.end(JSON.stringify({ error: 'Native qualification fixtures are read-only.' }));
      return;
    }
    if (pathname === '/__qualification') {
      response.writeHead(200, { 'content-type': 'application/json' });
      response.end(JSON.stringify({ fixture: 'native-fixed-data', mutations, missingAssets }));
      return;
    }
    // The development fixture exposes task lists. Derive detail responses from that same
    // source for lineage; missing run/task identities remain 404 rather than fabricated data.
    const taskDetail = pathname.match(/^\/apis\/v2beta1\/runs\/([^/]+)\/tasks\/([^/]+)$/);
    if (taskDetail) {
      const list = await fetch(
        `http://127.0.0.1:${port}/apis/v2beta1/runs/${taskDetail[1]}/tasks`,
        {
          signal: AbortSignal.timeout(5000),
        },
      );
      assert.ok(list.ok, 'Native task-list fixture must be available');
      const data = await list.json();
      const task = data.tasks.find(
        (candidate: { task_id: string }) => candidate.task_id === decodeURIComponent(taskDetail[2]),
      );
      response.writeHead(task ? 200 : 404, { 'content-type': 'application/json' });
      response.end(JSON.stringify(task || { error: 'Native task not found' }));
      return;
    }
    if (/^\/(api|apis|apps|artifacts|hub|k8s|system)(?:\/|$)/.test(pathname)) {
      api(request, response);
      return;
    }
    const target = resolve(
      build,
      `.${decodeURIComponent(pathname === '/' ? '/index.html' : pathname)}`,
    );
    assert.ok(
      target.startsWith(resolve(build) + sep),
      'Assets must remain within production build',
    );
    try {
      const bytes = await readFile(target);
      response.writeHead(200, {
        'content-type': mime[extname(target)] || 'application/octet-stream',
      });
      response.end(request.method === 'HEAD' ? undefined : bytes);
    } catch {
      missingAssets.push(pathname);
      response.writeHead(404);
      response.end('Missing production asset');
    }
  } catch (error) {
    response.writeHead(400);
    response.end(String(error));
  }
});
server.listen(port, '127.0.0.1', () => console.log(`Native fixture: http://127.0.0.1:${port}`));
for (const signal of ['SIGINT', 'SIGTERM'] as const) {
  process.on(signal, () => server.close(() => process.exit(0)));
}
