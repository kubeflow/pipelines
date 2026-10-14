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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import express, { type Request, type RequestHandler } from 'express';
import type { Server } from 'http';
import requests from 'supertest';
import { getArtifactsAuthMiddleware, getArtifactsProxyHandler } from '../handlers/artifacts.js';
import { validateArtifactNamespace } from '../helpers/artifact-validator.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import type { ArtifactAuthorizationMode } from '../configs.js';
import { getConfigMap } from '../k8s-helper.js';
import { TEST_ONLY as launcherConfigTestOnly } from '../helpers/launcher-config.js';

vi.mock('../k8s-helper.js', () => ({
  getConfigMap: vi.fn().mockResolvedValue([undefined, { message: 'not found' }]),
}));
vi.mock('../helpers/artifact-validator.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../helpers/artifact-validator.js')>()),
  validateArtifactNamespace: vi.fn(),
}));

describe('storage-authorized artifact routing', () => {
  let downstream: Server;
  let endpoint: string;
  let forwarded: string[];
  const authorize = vi.fn<AuthorizeFn>();
  const localFetch = vi.fn<RequestHandler>();
  const serviceGetter = vi.fn<(namespace: string) => string>();

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    launcherConfigTestOnly.clearLauncherConfigurationCache();
    authorize.mockResolvedValue(undefined);
    // Storage authorization must work when no artifact records are available.
    vi.mocked(validateArtifactNamespace).mockResolvedValue({
      valid: false,
      reason: 'artifact-api-unavailable',
    });
    forwarded = [];
    const service = express();
    service.use((req, res) => {
      forwarded.push(req.url);
      res.status(req.url.includes('denied') ? 403 : 200).send('namespace storage response');
    });
    downstream = await new Promise<Server>((resolve) => {
      const server = service.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = downstream.address();
    if (!address || typeof address === 'string') throw new Error('Expected TCP listener');
    endpoint = `http://127.0.0.1:${address.port}`;
    serviceGetter.mockImplementation(() => endpoint);
    localFetch.mockImplementation((_req, res) => {
      res.send('shared credential read');
    });
  });

  afterEach(async () => {
    await new Promise<void>((resolve) => downstream.close(() => resolve()));
  });

  function app({
    mutate,
    proxyEnabled = true,
    mode = 'storage',
    authEnabled = true,
    middlewareProxyEnabled = true,
  }: {
    mutate?: RequestHandler;
    proxyEnabled?: boolean;
    mode?: ArtifactAuthorizationMode;
    authEnabled?: boolean;
    middlewareProxyEnabled?: boolean;
  } = {}) {
    const server = express();
    server.use(
      getArtifactsAuthMiddleware(
        authorize,
        authEnabled,
        'kubeflow-userid',
        'http://api.invalid',
        middlewareProxyEnabled,
        mode,
      ),
    );
    if (mutate) server.use(mutate);
    server.use(
      getArtifactsProxyHandler({
        enabled: proxyEnabled,
        allowedDomain: '^127\\.0\\.0\\.1$',
        namespacedServiceGetter: serviceGetter,
      }),
    );
    server.use(localFetch);
    return server;
  }

  function artifactQuery(extra: Record<string, string> = {}) {
    return new URLSearchParams({
      source: 's3',
      bucket: 'research-data',
      key: 'historical/customer-17/model.txt',
      namespace: 'team-a',
      ...extra,
    }).toString();
  }

  it.each(['s3', 'minio', 'gcs'])(
    'accepts existing %s roots without artifact records',
    async (source) => {
      await requests(app())
        .get(`/artifacts/get?${artifactQuery({ source })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(200);
      expect(authorize).toHaveBeenCalledWith(
        expect.objectContaining({ namespace: 'team-a' }),
        expect.anything(),
      );
      expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
      expect(validateArtifactNamespace).not.toHaveBeenCalled();
      expect(forwarded).toHaveLength(1);
      expect(forwarded[0]).not.toContain('namespace=');
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('preserves a single trailing slash for legacy directory previews', async () => {
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ key: 'historical/customer-17/' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(new URL(forwarded[0], endpoint).searchParams.get('key')).toBe('historical/customer-17/');
    expect(validateArtifactNamespace).not.toHaveBeenCalled();
  });

  it('still authorizes the caller before delegating to storage', async () => {
    authorize.mockResolvedValue({ message: 'User cannot access team-b' });
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ namespace: 'team-b' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(serviceGetter).not.toHaveBeenCalled();
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('rejects unauthenticated requests', async () => {
    await requests(app()).get(`/artifacts/get?${artifactQuery()}`).expect(401);
    expect(serviceGetter).not.toHaveBeenCalled();
  });

  it.each([
    ['namespace', 'team-b'],
    ['namespace', ''],
    ['source', 'minio'],
    ['bucket', 'other-data'],
    ['key', 'victim/model.txt'],
    ['artifactUriQuery', 'endpoint=evil.example'],
  ])('rejects changed %s between authorization and proxying', async (parameter, value) => {
    const mutate: RequestHandler = (req, _res, next) => {
      const url = new URL(req.url, 'http://ui.invalid');
      url.searchParams.set(parameter, value);
      req.url = url.pathname + url.search;
      next();
    };
    await requests(app({ mutate }))
      .get(`/artifacts/get?${artifactQuery()}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(forwarded).toHaveLength(0);
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('pins the namespace and object through asynchronous provider resolution', async () => {
    let incoming: Request | undefined;
    vi.mocked(getConfigMap).mockImplementation(async () => {
      if (!incoming) throw new Error('Expected authorized request');
      incoming.url = `/artifacts/get?${artifactQuery({ namespace: 'team-b', key: 'victim/object' })}`;
      return [undefined, { message: 'not found' }];
    });
    await requests(
      app({
        mutate: (req, _res, next) => {
          incoming = req;
          next();
        },
      }),
    )
      .get(`/artifacts/get?${artifactQuery()}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
    expect(forwarded[0]).toContain('historical');
    expect(forwarded[0]).not.toContain('victim');
    expect(localFetch).not.toHaveBeenCalled();
  });

  it.each([
    'historical/../victim',
    'historical/./model',
    'historical//model',
    'historical/model//',
  ])('rejects malformed storage key %s', async (key) => {
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ key })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(400);
    expect(forwarded).toHaveLength(0);
  });

  it('rejects noncanonical percent-encoding before proxy access', async () => {
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ key: 'historical%2Fmodel', keyEncoding: 'uri' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(400);
    expect(forwarded).toHaveLength(0);
  });

  it('preserves the key-prefix default', async () => {
    await requests(app({ mode: 'key-prefix' }))
      .get(`/artifacts/get?${artifactQuery()}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(validateArtifactNamespace).toHaveBeenCalled();
    expect(forwarded).toHaveLength(0);
  });

  it.each(['http', 'https'])(
    'does not exempt %s from existing ownership validation',
    async (source) => {
      await requests(app())
        .get(`/artifacts/get?${artifactQuery({ source })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(403);
      expect(validateArtifactNamespace).toHaveBeenCalled();
      expect(forwarded).toHaveLength(0);
    },
  );

  it('rejects unsupported storage sources', async () => {
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ source: 'ftp' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(forwarded).toHaveLength(0);
  });

  it.each([{ proxyEnabled: false }, { middlewareProxyEnabled: false }, { authEnabled: false }])(
    'fails closed when namespace proxy prerequisites are absent: %j',
    async (options) => {
      await requests(app(options))
        .get(`/artifacts/get?${artifactQuery()}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(503);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('preserves storage denial without a shared-credential fallback', async () => {
    await requests(app())
      .get(`/artifacts/get?${artifactQuery({ key: 'denied/object' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('fails visibly when the namespace service is unavailable', async () => {
    await new Promise<void>((resolve) => downstream.close(() => resolve()));
    const response = await requests(app())
      .get(`/artifacts/get?${artifactQuery()}`)
      .set('kubeflow-userid', 'alice@example.com');
    expect(response.status).toBeGreaterThanOrEqual(500);
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('uses only launcher provider credentials from the authorized namespace', async () => {
    vi.mocked(getConfigMap).mockResolvedValue([
      {
        data: {
          providers: `
s3:
  default:
    credentials:
      fromEnv: true
  Overrides:
    - bucketName: research-data
      keyPrefix: historical/customer-17
      credentials:
        fromEnv: false
        secretRef:
          secretName: team-a-store
          accessKeyKey: access-key
          secretKeyKey: secret-key
`,
        },
      },
      undefined,
    ]);
    await requests(app())
      .get(
        `/artifacts/get?${artifactQuery({
          providerInfo: JSON.stringify({
            Provider: 's3',
            Params: { fromEnv: 'false', secretName: 'shared-admin' },
          }),
        })}`,
      )
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(getConfigMap).toHaveBeenCalledExactlyOnceWith('kfp-launcher', 'team-a');
    const forwardedProvider = new URL(forwarded[0], endpoint).searchParams.get('providerInfo');
    expect(JSON.parse(forwardedProvider || '')).toMatchObject({
      Provider: 's3',
      Params: { fromEnv: 'false', secretName: 'team-a-store' },
    });
    expect(forwarded[0]).not.toContain('shared-admin');
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('keeps failed launcher discovery inside the namespace storage boundary', async () => {
    vi.mocked(getConfigMap).mockResolvedValue([
      undefined,
      { message: 'Forbidden', additionalInfo: { code: 403 } },
    ]);
    await requests(app())
      .get(`/artifacts/get?${artifactQuery()}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(getConfigMap).toHaveBeenCalledExactlyOnceWith('kfp-launcher', 'team-a');
    expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
    expect(forwarded[0]).not.toContain('providerInfo');
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('removes browser-selected provider credentials before delegation', async () => {
    await requests(app())
      .get(
        `/artifacts/get?${artifactQuery({
          providerInfo: JSON.stringify({
            Provider: 's3',
            Params: { fromEnv: 'false', secretName: 'shared-admin' },
          }),
        })}`,
      )
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(forwarded[0]).not.toContain('providerInfo');
    expect(forwarded[0]).not.toContain('shared-admin');
  });
});
