// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import express, { type RequestHandler } from 'express';
import requests from 'supertest';
import { Readable } from 'node:stream';
import type { Server } from 'node:http';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { loadConfigs } from '../configs.js';
import * as ownership from '../helpers/mlmd-validator.js';
import * as minioHelper from '../minio-helper.js';
import * as serverInfo from '../helpers/server-info.js';
import {
  getArtifactsAuthMiddleware,
  getArtifactsHandler,
  getArtifactsProxyHandler,
} from '../handlers/artifacts.js';

vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn(), getPod: vi.fn() }));

describe('artifact authorization continuity through production handlers', () => {
  const namespace = 'team-a';
  const key = 'private-artifacts/team-a/result.txt';
  const syntheticFetch = vi.fn();
  const serviceGetter = vi.fn();
  let tenantServer: Server | undefined;

  beforeEach(() => {
    vi.spyOn(ownership, 'validateArtifactNamespace').mockResolvedValue({ valid: true });
    vi.spyOn(minioHelper, 'createMinioClient').mockResolvedValue({} as never);
    vi.spyOn(minioHelper, 'getObjectStream').mockImplementation(async () =>
      Readable.from(['data']),
    );
    vi.stubGlobal('fetch', syntheticFetch);
    syntheticFetch.mockReset();
    syntheticFetch.mockImplementation(async () => new Response('synthetic data'));
    serviceGetter.mockReset();
  });

  afterEach(async () => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    if (tenantServer) {
      await new Promise<void>((resolve) => tenantServer!.close(() => resolve()));
      tenantServer = undefined;
    }
  });

  function buildApp({
    mutate,
    proxy = false,
    auth = true,
    base = 'https://files.example/private-artifacts/',
    mlmd = true,
    httpToken = '',
    allowedDomain = '.*',
  }: {
    mutate?: RequestHandler;
    proxy?: boolean;
    auth?: boolean;
    base?: string;
    mlmd?: boolean;
    httpToken?: string;
    allowedDomain?: string;
  } = {}) {
    const options = loadConfigs(['node', 'server', '/tmp', '3000'], {
      HTTP_BASE_URL: base,
      ALLOWED_ARTIFACT_DOMAIN_REGEX: allowedDomain,
      HTTP_AUTHORIZATION_KEY: httpToken ? 'Authorization' : '',
      HTTP_AUTHORIZATION_DEFAULT_VALUE: httpToken,
      MINIO_HOST: 'synthetic-store',
      MINIO_NAMESPACE: 'kubeflow',
    });
    options.auth.enabled = auth;
    const app = express();
    app.get(
      '/artifacts/*',
      getArtifactsAuthMiddleware(
        async () => undefined,
        auth,
        'user',
        mlmd ? 'synthetic-mlmd' : undefined,
      ),
      mutate ?? ((_req, _res, next) => next()),
      getArtifactsProxyHandler({
        enabled: proxy,
        allowedDomain: '.*',
        namespacedServiceGetter: serviceGetter,
      }),
    );
    for (const [route, useParameter] of [
      ['/artifacts/get', false],
      ['/artifacts/:source/:bucket/*', true],
    ] as const) {
      app.get(
        route,
        getArtifactsHandler({
          options,
          artifactsConfigs: options.artifacts,
          useParameter,
          tryExtract: false,
        }),
      );
    }
    return app;
  }

  function get(app: ReturnType<typeof buildApp>, query: Record<string, string> = {}) {
    return requests(app)
      .get('/artifacts/get')
      .query({
        source: 'minio',
        bucket: 'data',
        key,
        namespace,
        ...query,
      })
      .set('user', 'reader');
  }

  it.each(['source', 'bucket', 'key', 'namespace'])(
    'rejects a parsed %s change before storage access',
    async (field) => {
      const app = buildApp({
        mutate: (req, _res, next) => {
          Object.defineProperty(req, 'query', {
            value: { ...req.query, [field]: field === 'source' ? 's3' : 'changed' },
          });
          next();
        },
      });
      const response = await get(app);
      expect(response.status).toBe(403);
      expect(minioHelper.getObjectStream).not.toHaveBeenCalled();
    },
  );

  it('denies missing identity before storage or proxy access', async () => {
    await requests(buildApp({ proxy: true }))
      .get('/artifacts/get')
      .query({
        source: 'minio',
        bucket: 'data',
        key,
        namespace,
      })
      .expect(401);
    expect(minioHelper.getObjectStream).not.toHaveBeenCalled();
    expect(serviceGetter).not.toHaveBeenCalled();
  });

  it('keeps encoded download-path coordinates unchanged through serving', async () => {
    const encodedKey = 'private-artifacts/team-a/file%2Fname.txt';
    await requests(buildApp())
      .get(`/artifacts/minio/data/${encodeURIComponent(encodedKey)}`)
      .query({ namespace })
      .set('user', 'reader')
      .expect(200);
    expect(ownership.validateArtifactNamespace).toHaveBeenCalledWith(
      'synthetic-mlmd',
      `minio://data/${encodedKey}`,
      namespace,
    );
    expect(minioHelper.getObjectStream).toHaveBeenCalledWith(
      expect.objectContaining({ key: encodedKey }),
    );
  });

  it('compares coordinates without ambiguous URI concatenation', async () => {
    const app = buildApp({
      mutate: (req, _res, next) => {
        Object.defineProperty(req, 'query', {
          value: { ...req.query, bucket: 'data/private-artifacts', key: 'team-a/result.txt' },
        });
        next();
      },
    });
    await get(app).expect(403);
    expect(minioHelper.getObjectStream).not.toHaveBeenCalled();
  });

  it('preserves harmless dot segments in authenticated volume reads', async () => {
    const directory = await mkdtemp(join(tmpdir(), 'artifact-continuity-'));
    try {
      await writeFile(join(directory, 'result.txt'), 'volume result');
      vi.spyOn(serverInfo, 'getHostPod').mockResolvedValue([
        {
          spec: {
            containers: [
              { name: 'ml-pipeline-ui', volumeMounts: [{ name: 'data', mountPath: directory }] },
            ],
            volumes: [{ name: 'data', persistentVolumeClaim: { claimName: 'data' } }],
          },
        },
        undefined,
      ]);
      await get(buildApp(), { source: 'volume', key: './result.txt' }).expect(200, 'volume result');
      expect(ownership.validateArtifactNamespace).not.toHaveBeenCalled();
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });

  it.each(['source', 'bucket', 'key', 'namespace'])(
    'rejects changed raw proxy %s before contacting a tenant service',
    async (field) => {
      const app = buildApp({
        proxy: true,
        mutate: (req, _res, next) => {
          const url = new URL(req.url, 'http://local');
          url.searchParams.set(field, 'changed');
          req.url = url.pathname + url.search;
          next();
        },
      });
      await get(app).expect(403);
      expect(serviceGetter).not.toHaveBeenCalled();
    },
  );

  it.each(['/artifacts/get', '/ARTIFACTS/GET/'])(
    'preserves query downloads from %s when forwarding to an older tenant service',
    async (route) => {
      const received: string[] = [];
      const tenant = express();
      tenant.get('/artifacts/:source/:bucket/*', (req, res) => {
        received.push(req.params[0]);
        res.send('legacy download');
      });
      tenantServer = await new Promise<Server>((resolve) => {
        const server = tenant.listen(0, '127.0.0.1', () => resolve(server));
      });
      const address = tenantServer.address();
      if (!address || typeof address === 'string') throw new Error('Missing test server port');
      serviceGetter.mockReturnValue(`http://127.0.0.1:${address.port}`);
      const encodedKey = 'private-artifacts/team-a/file%2Fname.txt';
      await requests(buildApp({ proxy: true }))
        .get(route)
        .query({ source: 'minio', bucket: 'data', namespace, key: encodedKey, download: 'true' })
        .set('user', 'reader')
        .expect(200);
      expect(received).toEqual([encodedKey]);
    },
  );

  it('authorizes case-insensitive download routes from their path, not decoy queries', async () => {
    const received: string[] = [];
    const tenant = express();
    tenant.get('/artifacts/:source/:bucket/*', (req, res) => {
      received.push(req.params[0]);
      res.send('must not serve');
    });
    tenantServer = await new Promise<Server>((resolve) => {
      const server = tenant.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = tenantServer.address();
    if (!address || typeof address === 'string') throw new Error('Missing test server port');
    serviceGetter.mockReturnValue(`http://127.0.0.1:${address.port}`);
    vi.mocked(ownership.validateArtifactNamespace).mockImplementation(async (_envoy, uri, ns) =>
      ownership.decideFromPrefixFallback(uri, ns, 'mlmd-then-prefix'),
    );
    await requests(buildApp({ proxy: true }))
      .get('/ARTIFACTS/minio/data/private-artifacts/team-b/result.txt')
      .query({ source: 'minio', bucket: 'data', key, namespace })
      .set('user', 'reader')
      .expect(403);
    expect(received).toEqual([]);
    expect(serviceGetter).not.toHaveBeenCalled();
    await requests(buildApp({ proxy: true }))
      .get('/ARTIFACTS/minio/data/private-artifacts/team-a/result.txt')
      .query({ namespace })
      .set('user', 'reader')
      .expect(200);
    expect(received).toEqual([key]);
  });

  async function startTenant(tenant: ReturnType<typeof express>) {
    tenantServer = await new Promise<Server>((resolve) => {
      const server = tenant.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = tenantServer.address();
    if (!address || typeof address === 'string') throw new Error('Missing test server port');
    serviceGetter.mockReturnValue(`http://127.0.0.1:${address.port}`);
  }

  it.each([false, true])(
    'preserves custom-root reads without an MLMD endpoint (proxy=%s)',
    async (proxy) => {
      if (proxy) {
        await startTenant(buildApp({ auth: false }));
      }
      await get(buildApp({ proxy, mlmd: false }), { key: 'historical/model/output.txt' }).expect(
        200,
      );
      expect(ownership.validateArtifactNamespace).not.toHaveBeenCalled();
      expect(minioHelper.getObjectStream).toHaveBeenCalledWith(
        expect.objectContaining({ key: 'historical/model/output.txt' }),
      );
    },
  );

  describe.each([false, true])('key safety without an MLMD endpoint (proxy=%s)', (proxy) => {
    it.each([
      'historical/../team-b/model',
      'historical/%2e%2e/team-b/model',
      'historical/model%2F..%2Fteam-b/model',
      'historical/%00model',
      'historical/%ZZmodel',
      'historical//model',
    ])('rejects %s before storage or proxy access', async (unsafeKey) => {
      await get(buildApp({ proxy, mlmd: false }), { key: unsafeKey }).expect(403);
      expect(ownership.validateArtifactNamespace).not.toHaveBeenCalled();
      expect(minioHelper.getObjectStream).not.toHaveBeenCalled();
      expect(serviceGetter).not.toHaveBeenCalled();
    });
  });

  it.each([false, true])(
    'forwards HTTP %s downloads to the older namespace receiver during rollout',
    async (download) => {
      const tenant = express();
      const received: Array<{ key: unknown; namespace: unknown }> = [];
      // Model the two existing receiver routes, without requiring a new protocol.
      tenant.get('/artifacts/get', (req, res) => {
        received.push({ key: req.query.key, namespace: req.query.namespace });
        res.send('legacy namespace response');
      });
      tenant.get('/artifacts/:source/:bucket/*', (req, res) => {
        received.push({ key: req.params[0], namespace: req.query.namespace });
        res.send('legacy namespace response');
      });
      await startTenant(tenant);
      await get(buildApp({ proxy: true, base: '' }), {
        source: 'https',
        bucket: 'files.example',
        key: 'reports/model%2Foutput.txt',
        download: String(download),
      }).expect(200);
      expect(received).toEqual([{ key: 'reports/model%2Foutput.txt', namespace: undefined }]);
      expect(syntheticFetch).not.toHaveBeenCalled();
    },
  );

  it.each([false, true])(
    'uses only the updated namespace receiver HTTP credentials (download=%s)',
    async (download) => {
      await startTenant(
        buildApp({
          auth: false,
          base: 'https://files.example/reports/',
          httpToken: 'synthetic-namespace-token',
        }),
      );
      syntheticFetch.mockResolvedValueOnce(
        new Response(null, {
          status: 302,
          headers: { location: 'https://files.example/reports/final.txt' },
        }),
      );
      syntheticFetch.mockResolvedValueOnce(new Response('namespace artifact'));
      await get(buildApp({ proxy: true, base: '', httpToken: 'wrong-central-token' }), {
        source: 'https',
        bucket: 'files.example',
        key: 'reports/result.txt',
        download: String(download),
      }).expect(200);
      expect(syntheticFetch).toHaveBeenCalledTimes(2);
      for (const call of syntheticFetch.mock.calls) {
        expect(call[1]).toEqual({
          headers: { Authorization: 'synthetic-namespace-token' },
          redirect: 'manual',
        });
      }
      // Ownership evidence applies to the requested artifact, not a new
      // object selected by a redirect within the operator's approved boundary.
      expect(ownership.validateArtifactNamespace).toHaveBeenCalledTimes(1);
    },
  );

  it.each([
    ['custom root', 'https://files.example/reports/final.txt', 200],
    ['encoded filename', 'https://files.example/reports/final%20report.txt', 200],
    ['outside base', 'https://files.example/reports-other/final.txt', 400],
    ['other origin', 'https://other.example/reports/final.txt', 400],
    ['signed foreign origin', 'https://other.example/reports/final.txt?signature=x', 400],
    ['signed path escape', 'https://files.example/private/final.txt?signature=x', 400],
    ['embedded credentials', 'https://user:pass@files.example/reports/final.txt', 400],
    ['encoded traversal', 'https://files.example/reports/model%2F..%2F..%2Fprivate/file', 400],
    ['signed query', 'https://files.example/reports/final.txt?signature=x', 200],
    ['fragment', 'https://files.example/reports/final.txt#other', 200],
  ])(
    'checks %s redirects in updated receivers before fetching their destination',
    async (_name, target, status) => {
      await startTenant(
        buildApp({
          auth: false,
          base: 'https://files.example/reports/',
          httpToken: 'synthetic-namespace-token',
        }),
      );
      syntheticFetch.mockResolvedValueOnce(
        new Response(null, { status: 302, headers: { location: target } }),
      );
      syntheticFetch.mockResolvedValueOnce(new Response('final'));
      await get(buildApp({ proxy: true, base: '' }), {
        source: 'https',
        bucket: 'files.example',
        key: 'reports/result.txt',
      }).expect(status as number);
      expect(syntheticFetch).toHaveBeenCalledTimes(status === 200 ? 2 : 1);
    },
  );

  it.each([
    ['https://gateway.example/downloads/final.txt?signature=x', 'synthetic-namespace-token'],
    ['https://cdn.example/downloads/final.txt?signature=x', undefined],
  ])(
    'preserves approved gateway redirects through the namespace receiver: %s',
    async (target, expectedToken) => {
      await startTenant(
        buildApp({
          auth: false,
          base: 'gateway.example/artifacts/',
          httpToken: 'synthetic-namespace-token',
          allowedDomain: '^(gateway|cdn)\\.example$',
        }),
      );
      syntheticFetch.mockResolvedValueOnce(
        new Response(null, { status: 302, headers: { location: target! } }),
      );
      syntheticFetch.mockResolvedValueOnce(new Response('final'));
      await get(buildApp({ proxy: true, base: '' }), {
        source: 'https',
        key: 'legacy/start.txt',
      }).expect(200);
      expect(syntheticFetch).toHaveBeenNthCalledWith(2, target, {
        headers: expectedToken ? { Authorization: expectedToken } : {},
        redirect: 'manual',
      });
    },
  );

  it('rejects a gateway redirect outside the receiver allowlist without forwarding credentials', async () => {
    await startTenant(
      buildApp({
        auth: false,
        base: 'gateway.example/artifacts/',
        httpToken: 'synthetic-namespace-token',
        allowedDomain: '^gateway\\.example$',
      }),
    );
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: { location: 'http://169.254.169.254/private?signature=x' },
      }),
    );
    await get(buildApp({ proxy: true, base: '' }), {
      source: 'https',
      key: 'legacy/start.txt',
    }).expect(500);
    expect(syntheticFetch).toHaveBeenCalledTimes(1);
  });

  it.each([false, true])(
    'retains custom-root HTTP reads and redirects within the approved base (auth=%s)',
    async (auth) => {
      syntheticFetch.mockResolvedValueOnce(
        new Response(null, {
          status: 302,
          headers: { location: 'https://files.example/reports/final.txt' },
        }),
      );
      syntheticFetch.mockResolvedValueOnce(new Response('custom root'));
      await get(buildApp({ auth, base: 'https://files.example/reports/' }), {
        source: 'https',
        bucket: 'files.example',
        key: 'reports/start.txt',
      }).expect(200);
      expect(ownership.validateArtifactNamespace).toHaveBeenCalledTimes(auth ? 1 : 0);
    },
  );
});
