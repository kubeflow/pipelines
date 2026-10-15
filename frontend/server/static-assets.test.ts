// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import * as fs from 'node:fs/promises';
import { createServer, request, type IncomingHttpHeaders, type Server } from 'node:http';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { gzipSync } from 'node:zlib';
import express from 'express';
import { createPrecompressedStaticApp } from './static-assets.js';

vi.mock('node:fs/promises', async (importOriginal) => {
  const actual = await importOriginal<typeof import('node:fs/promises')>();
  return { ...actual, stat: vi.fn(actual.stat) };
});

const source = Buffer.from('window.editor = "' + 'editor contents '.repeat(100) + '";');
const compressed = gzipSync(source);
let directory: string;
let server: Server;
let port: number;
let serverError: unknown;

function get(url: string, headers: Record<string, string> = {}, method = 'GET') {
  return new Promise<{ status: number; headers: IncomingHttpHeaders; body: Buffer }>(
    (resolve, reject) => {
      const req = request({ hostname: '127.0.0.1', port, path: url, method, headers }, (res) => {
        const chunks: Buffer[] = [];
        res.on('data', (chunk: Buffer) => chunks.push(chunk));
        res.on('error', reject);
        res.on('end', () =>
          resolve({ status: res.statusCode!, headers: res.headers, body: Buffer.concat(chunks) }),
        );
      });
      req.on('error', reject);
      req.end();
    },
  );
}

beforeAll(async () => {
  directory = await mkdtemp(path.join(tmpdir(), 'kfp-static-assets-'));
  await mkdir(path.join(directory, 'static'));
  await Promise.all([
    writeFile(path.join(directory, 'static/Editor-hash.js'), source),
    writeFile(path.join(directory, 'static/Editor-hash.js.gz'), compressed),
    writeFile(path.join(directory, 'static/Editor-hash.css'), source),
    writeFile(path.join(directory, 'static/Editor-hash.css.gz'), compressed),
    writeFile(path.join(directory, 'static/Editor.min-hash.js'), source),
    writeFile(path.join(directory, 'static/Editor.min-hash.js.gz'), compressed),
    writeFile(path.join(directory, 'static/identity-only.js'), source),
    writeFile(path.join(directory, 'static/orphan.js.gz'), compressed),
  ]);
  const app = express();
  const assets = createPrecompressedStaticApp(directory);
  app.use('/pipeline', assets);
  app.use('/fixture', createPrecompressedStaticApp(directory, { cacheControl: 'no-store' }));
  app.use(assets);
  app.use((_req, res) => {
    res.status(404).send('fallback');
  });
  app.use(
    (error: unknown, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
      serverError = error;
      res.status(500).send('lookup error');
    },
  );
  server = createServer(app);
  await new Promise<void>((resolve) => {
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('Missing test listener');
  port = address.port;
});

afterAll(async () => {
  await new Promise<void>((resolve, reject) =>
    server.close((error) => (error ? reject(error) : resolve())),
  );
  await rm(directory, { recursive: true, force: true });
});

it.each([
  '/static/Editor-hash.js',
  '/pipeline/static/Editor-hash.js',
  '/static/Editor.min-hash.js',
])('serves exact gzip bytes with original MIME at %s', async (url) => {
  const res = await get(url, { 'Accept-Encoding': 'gzip' });
  expect(res.status).toBe(200);
  expect(res.body).toEqual(compressed);
  expect(res.headers['content-encoding']).toBe('gzip');
  expect(res.headers['content-type']).toMatch(/javascript/);
  expect(res.headers['content-length']).toBe(String(compressed.length));
  expect(res.headers.vary).toBe('Accept-Encoding');
  expect(res.headers['cache-control']).toBe('public, max-age=0');
});

it('preserves CSS MIME and query strings', async () => {
  const res = await get('/static/Editor-hash.css?v=1', { 'Accept-Encoding': 'gzip' });
  expect(res.headers['content-type']).toMatch(/^text\/css/);
  expect(res.body).toEqual(compressed);
});

it.each([
  [undefined, undefined, 200],
  ['gzip;q=0, identity;q=1', undefined, 200],
  ['gzip;q=0.5, identity;q=1', undefined, 200],
  ['gzip;q=1, identity;q=0.5', 'gzip', 200],
  ['br, gzip;q=0.5', 'gzip', 200],
  ['*;q=1, identity;q=0', 'gzip', 200],
  ['gzip;q=bogus, identity;q=1', undefined, 200],
  ['gzip;q=0, identity;q=0', undefined, 406],
  ['*;q=0', undefined, 406],
])('negotiates %s through Express', async (accept, encoding, status) => {
  const res = await get('/static/Editor-hash.js', accept ? { 'Accept-Encoding': accept } : {});
  expect(res.status).toBe(status);
  expect(res.headers['content-encoding']).toBe(encoding);
  expect(res.headers.vary).toBe('Accept-Encoding');
  if (status === 200) expect(res.body).toEqual(encoding ? compressed : source);
});

it('supports missing sidecars without fabricating unavailable encodings', async () => {
  const fallback = await get('/static/identity-only.js', { 'Accept-Encoding': 'gzip' });
  expect(fallback.status).toBe(200);
  expect(fallback.body).toEqual(source);
  expect(fallback.headers.vary).toBe('Accept-Encoding');
  const rejected = await get('/static/identity-only.js', {
    'Accept-Encoding': 'gzip, identity;q=0',
  });
  expect(rejected.status).toBe(406);
});

it('keeps validators specific to each representation and varies 304 responses', async () => {
  const gzip = await get('/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' });
  const identity = await get('/static/Editor-hash.js', { 'Accept-Encoding': 'identity' });
  expect(gzip.headers.etag).not.toBe(identity.headers.etag);
  for (const [encoding, etag] of [
    ['gzip', gzip.headers.etag!],
    ['identity', identity.headers.etag!],
  ]) {
    const fresh = await get('/static/Editor-hash.js', {
      'Accept-Encoding': encoding,
      'If-None-Match': etag,
    });
    expect(fresh.status).toBe(304);
    expect(fresh.headers.vary).toBe('Accept-Encoding');
    expect(fresh.body.length).toBe(0);
  }
  const different = await get('/static/Editor-hash.js', {
    'Accept-Encoding': 'identity',
    'If-None-Match': gzip.headers.etag!,
  });
  expect(different.status).toBe(200);
  expect(different.body).toEqual(source);
});

it('serves HEAD without a body and ignores gzip ranges while preserving identity ranges', async () => {
  const head = await get('/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' }, 'HEAD');
  expect(head.status).toBe(200);
  expect(head.body.length).toBe(0);
  expect(head.headers['content-length']).toBe(String(compressed.length));
  const gzip = await get('/static/Editor-hash.js', {
    'Accept-Encoding': 'gzip',
    Range: 'bytes=0-3',
  });
  expect(gzip.status).toBe(200);
  expect(gzip.body).toEqual(compressed);
  expect(gzip.headers['accept-ranges']).toBeUndefined();
  const identity = await get('/static/Editor-hash.js', {
    'Accept-Encoding': 'identity',
    Range: 'bytes=0-3',
  });
  expect(identity.status).toBe(206);
  expect(identity.body).toEqual(source.subarray(0, 4));
});

it('allows the qualification fixture to preserve no-store caching', async () => {
  const res = await get('/fixture/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' });
  expect(res.status).toBe(200);
  expect(res.headers['cache-control']).toBe('no-store');
});

it.each([
  '/static/orphan.js',
  '/static/missing.js',
  '/index.html',
  '/apis/v2beta1/runs',
  '/artifacts/get',
  '/static/../Editor-hash.js',
  '/static/%2e%2e/Editor-hash.js',
  '/static/nested/Editor-hash.js',
])('leaves non-assets or missing originals to existing routes: %s', async (url) => {
  const res = await get(url, { 'Accept-Encoding': 'gzip' });
  expect(res.status).toBe(404);
  expect(res.body.toString()).toBe('fallback');
  expect(res.headers['content-encoding']).toBeUndefined();
});

it('does not handle mutations', async () => {
  expect((await get('/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' }, 'POST')).status).toBe(
    404,
  );
});

it('leaves an overlong asset filename to existing routes', async () => {
  const res = await get(`/static/${'x'.repeat(300)}.js`, { 'Accept-Encoding': 'gzip' });
  expect(res.status).toBe(404);
  expect(res.body.toString()).toBe('fallback');
  expect(res.headers['content-encoding']).toBeUndefined();
});

it.each(['ENOENT', 'ENOTDIR', 'ENAMETOOLONG'])(
  'preserves static-route fallthrough for %s lookup failures',
  async (code) => {
    const lookup = vi
      .mocked(fs.stat)
      .mockRejectedValueOnce(Object.assign(new Error(code), { code }));
    try {
      const res = await get('/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' });
      expect(res.status).toBe(404);
      expect(res.body.toString()).toBe('fallback');
      expect(res.headers['content-encoding']).toBeUndefined();
    } finally {
      lookup.mockReset();
    }
  },
);

it.each(['EACCES', 'EIO'])('preserves unexpected %s lookup errors', async (code) => {
  const failure = Object.assign(new Error(code), { code });
  const lookup = vi.mocked(fs.stat).mockRejectedValueOnce(failure);
  try {
    const res = await get('/static/Editor-hash.js', { 'Accept-Encoding': 'gzip' });
    expect(res.status).toBe(500);
    expect(serverError).toBe(failure);
  } finally {
    lookup.mockReset();
  }
});
