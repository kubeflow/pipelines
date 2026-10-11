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
  }: {
    mutate?: RequestHandler;
    proxy?: boolean;
    auth?: boolean;
    base?: string;
  } = {}) {
    const options = loadConfigs(['node', 'server', '/tmp', '3000'], {
      HTTP_BASE_URL: base,
      MINIO_HOST: 'synthetic-store',
      MINIO_NAMESPACE: 'kubeflow',
    });
    options.auth.enabled = auth;
    const app = express();
    app.get(
      '/artifacts/*',
      getArtifactsAuthMiddleware(async () => undefined, auth, 'user', 'synthetic-mlmd'),
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

  it('keeps authenticated HTTP serving central even when the proxy is enabled', async () => {
    syntheticFetch.mockResolvedValue(new Response('central data'));
    await get(buildApp({ proxy: true }), { source: 'https', bucket: 'files.example' }).expect(200);
    expect(serviceGetter).not.toHaveBeenCalled();
    expect(syntheticFetch).toHaveBeenCalledWith(`https://files.example/${key}`, expect.anything());
  });

  it.each([
    ['same tenant', 'https://files.example/private-artifacts/team-a/final.txt', 200],
    ['other tenant', 'https://files.example/private-artifacts/team-b/final.txt', 403],
    ['encoded other tenant', 'https://files.example/private-artifacts/%74eam-b/final.txt', 403],
    [
      'encoded separator',
      'https://files.example/private-artifacts/team-a%2F..%2Fteam-b/final.txt',
      400,
    ],
    ['other origin', 'https://other.example/private-artifacts/team-a/final.txt', 400],
    ['signed query', 'https://files.example/private-artifacts/team-a/final.txt?signature=x', 403],
    ['fragment', 'https://files.example/private-artifacts/team-a/final.txt#other', 403],
  ])('checks %s redirects before fetching their destination', async (_name, target, status) => {
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, { status: 302, headers: { location: target } }),
    );
    syntheticFetch.mockResolvedValueOnce(new Response('final'));
    await get(buildApp(), { source: 'https', bucket: 'files.example' }).expect(status as number);
    expect(syntheticFetch).toHaveBeenCalledTimes(status === 200 ? 2 : 1);
  });

  it.each(['artifact-not-found', 'namespace-mismatch'] as const)(
    'preserves target ownership denial (%s) after a same-tenant redirect',
    async (reason) => {
      vi.mocked(ownership.validateArtifactNamespace)
        .mockResolvedValueOnce({ valid: true })
        .mockResolvedValueOnce({ valid: false, reason });
      syntheticFetch.mockResolvedValueOnce(
        new Response(null, {
          status: 302,
          headers: {
            location: 'https://files.example/private-artifacts/team-a/final.txt',
          },
        }),
      );
      await get(buildApp(), { source: 'https', bucket: 'files.example' }).expect(403);
      expect(ownership.validateArtifactNamespace).toHaveBeenNthCalledWith(
        2,
        'synthetic-mlmd',
        'https://files.example/private-artifacts/team-a/final.txt',
        namespace,
      );
      expect(syntheticFetch).toHaveBeenCalledTimes(1);
    },
  );

  it('allows same-tenant gateway redirects but rejects a different logical bucket', async () => {
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: {
          location: 'https://gateway.example/artifacts/data/private-artifacts/team-a/final.txt',
        },
      }),
    );
    syntheticFetch.mockResolvedValueOnce(new Response('final'));
    await get(buildApp({ base: 'gateway.example/artifacts/' }), { source: 'https' }).expect(200);
    syntheticFetch.mockReset();
    syntheticFetch.mockImplementation(async () => new Response('synthetic data'));
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: {
          location: 'https://gateway.example/artifacts/other/private-artifacts/team-a/final.txt',
        },
      }),
    );
    await get(buildApp({ base: 'gateway.example/artifacts/' }), { source: 'https' }).expect(403);
    expect(syntheticFetch).toHaveBeenCalledTimes(1);
  });

  it('preserves audit custom-root initial reads without granting redirect authority', async () => {
    vi.mocked(ownership.validateArtifactNamespace).mockResolvedValue({
      valid: true,
      reason: 'audit-custom-root',
    });
    syntheticFetch.mockResolvedValueOnce(new Response('legacy custom root'));
    await get(buildApp({ base: 'https://files.example/' }), {
      source: 'https',
      bucket: 'files.example',
      key: 'legacy/result.txt',
    }).expect(200);
    syntheticFetch.mockReset();
    syntheticFetch.mockImplementation(async () => new Response('synthetic data'));
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: {
          location: `https://files.example/${key}`,
        },
      }),
    );
    await get(buildApp({ base: 'https://files.example/' }), {
      source: 'https',
      bucket: 'files.example',
      key: 'legacy/result.txt',
    }).expect(403);
    expect(syntheticFetch).toHaveBeenCalledTimes(1);
  });

  it('retains standalone HTTP redirect behavior within the approved base', async () => {
    syntheticFetch.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: {
          location: 'https://files.example/private-artifacts/team-b/final.txt',
        },
      }),
    );
    syntheticFetch.mockResolvedValueOnce(new Response('standalone'));
    await get(buildApp({ auth: false }), { source: 'https', bucket: 'files.example' }).expect(200);
    expect(ownership.validateArtifactNamespace).not.toHaveBeenCalled();
  });
});
