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
import { gzipSync } from 'node:zlib';
import requests from 'supertest';
import {
  getArtifactsAuthMiddleware,
  getArtifactsProxyHandler,
  getArtifactReceiverNamespaceMiddleware,
  getArtifactStorageReceiverMiddleware,
  ARTIFACT_STORAGE_PROXY_PREFIX,
} from '../handlers/artifacts.js';
import { decideArtifactOwnership, validateArtifactNamespace } from '../helpers/mlmd-validator.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import { loadConfigs, type ArtifactAuthorizationMode } from '../configs.js';
import { getArtifactCredentialBinding } from '../helpers/artifact-authorization.js';
import { getK8sSecret } from '../k8s-helper.js';
vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

vi.mock('../helpers/mlmd-validator.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../helpers/mlmd-validator.js')>()),
  validateArtifactNamespace: vi.fn(),
}));

function content(response: { text?: string; body: unknown }): string {
  return Buffer.isBuffer(response.body) ? response.body.toString('utf8') : response.text || '';
}

describe('storage-authorized artifact routing', () => {
  let downstream: Server;
  let endpoint: string;
  let forwarded: string[];
  let receivingNamespace: string;
  let receiverUID: string;
  let receiverSecret: string;
  let receiverSSL: boolean;
  let legacyReceiver: boolean;
  let legacyFraming: 'length' | 'chunked' | 'gzip';
  let receiverStatus: number;
  let receiverStream: boolean;
  let finishReceiverStream: () => void;
  function receiverOptions() {
    return loadConfigs(['node', 'server.js', '/tmp'], {
      FRONTEND_SERVER_NAMESPACE: receivingNamespace,
      KFP_ARTIFACT_NAMESPACE_UID: receiverUID,
      MINIO_ACCESS_KEY: 'team-a-key',
      MINIO_SECRET_KEY: receiverSecret,
      MINIO_SSL: String(receiverSSL),
    });
  }
  function approval(state = 'ready') {
    return JSON.stringify({
      version: 1,
      namespace: 'team-a',
      namespaceUID: 'uid-a',
      state,
      bucket: 'mlpipeline',
      credentialBinding: getArtifactCredentialBinding(
        'team-a',
        'uid-a',
        receiverOptions().artifacts.minio,
      ),
    });
  }
  const authorize = vi.fn<AuthorizeFn>();
  const localFetch = vi.fn<RequestHandler>();
  const serviceGetter = vi.fn<(namespace: string) => string>();

  beforeEach(async () => {
    vi.clearAllMocks();
    authorize.mockResolvedValue(undefined);
    // Storage authorization must work when no artifact records are available.
    vi.mocked(validateArtifactNamespace).mockResolvedValue({
      valid: false,
      reason: 'mlmd-unavailable',
    });
    forwarded = [];
    receivingNamespace = 'team-a';
    receiverUID = 'uid-a';
    receiverSecret = 'team-a-secret';
    receiverSSL = false;
    legacyReceiver = false;
    legacyFraming = 'length';
    receiverStatus = 200;
    receiverStream = false;
    finishReceiverStream = () => undefined;
    vi.mocked(getK8sSecret).mockRejectedValue({ code: 404 });
    const service = express();
    service.get(`${ARTIFACT_STORAGE_PROXY_PREFIX}/*`, (req, res, next) => {
      if (legacyReceiver) {
        const html = '<html><script>old receiver SPA</script></html>';
        res.status(200).set({
          'Content-Type': 'text/html',
          'Content-Disposition': 'inline',
          ETag: 'old-receiver-body',
          'Set-Cookie': 'tenant-controlled=value',
          'Content-Range': 'bytes 0-44/45',
        });
        if (legacyFraming === 'gzip') {
          res.set('x-kfp-artifact-proxy-version', '2');
          res.set('Content-Encoding', 'gzip').send(gzipSync(html));
        } else if (legacyFraming === 'chunked') {
          res.write(html.slice(0, 10));
          res.end(html.slice(10));
        } else res.send(html);
        return;
      }
      getArtifactStorageReceiverMiddleware(receiverOptions())(req, res, next);
    });
    service.use((req, res, next) =>
      getArtifactReceiverNamespaceMiddleware(receivingNamespace)(req, res, next),
    );
    service.use((req, res) => {
      forwarded.push(req.url);
      if (receiverStream) {
        finishReceiverStream = () => res.end(Buffer.alloc(128 * 1024, 'b'));
        res.write(Buffer.alloc(128 * 1024, 'a'));
        return;
      }
      res
        .status(req.url.includes('denied') ? 403 : receiverStatus)
        .send('namespace storage response');
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

  it.each(['auto', 'storage'] as const)(
    'rejects a percent-encoded legacy root in %s mode before storage access',
    async (mode) => {
      vi.mocked(getK8sSecret).mockResolvedValue(approval());
      const response = await requests(app({ mode }))
        .get('/artifacts/minio/mlpipeline/%61rtifacts/victim/model.txt?namespace=team-a')
        .set('kubeflow-userid', 'alice@example.com');
      expect([400, 403]).toContain(response.status);
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('keeps a similarly named isolated key on the storage-authorized path', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue(approval());
    await requests(app({ mode: 'auto' }))
      .get(
        `/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline', key: 'artifacts-team-a/model.txt' })}`,
      )
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(validateArtifactNamespace).not.toHaveBeenCalled();
    expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
    expect(localFetch).not.toHaveBeenCalled();
  });

  describe.each(['auto', 'storage'] as const)(
    'ownership audit under %s authorization',
    (authorizationMode) => {
      describe.each(['ready', 'missing'])('legacy artifacts with %s managed record', (record) => {
        it.each([
          ['enforce', 'team-a', 403],
          ['audit', 'team-a', 200],
          ['audit', 'team-b', 403],
          ['audit', '', 403],
        ] as const)(
          'keeps %s ownership semantics for metadata namespace %s',
          async (mode, owner, status) => {
            if (record === 'ready') vi.mocked(getK8sSecret).mockResolvedValue(approval());
            vi.mocked(validateArtifactNamespace).mockImplementation(
              async (_address, uri, namespace) =>
                decideArtifactOwnership(
                  uri,
                  namespace,
                  [
                    {
                      artifactId: 1,
                      contexts: owner
                        ? [{ contextType: 'system.PipelineRun', namespace: owner }]
                        : [],
                    },
                  ],
                  mode,
                ),
            );
            await requests(app({ mode: authorizationMode }))
              .get(
                `/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline', key: 'artifacts/history/model.txt' })}`,
              )
              .set('kubeflow-userid', 'alice@example.com')
              .expect(status);
            expect(validateArtifactNamespace).toHaveBeenCalledOnce();
            expect(serviceGetter).toHaveBeenCalledTimes(status === 200 ? 1 : 0);
            expect(localFetch).not.toHaveBeenCalled();
          },
        );
      });
    },
  );

  describe.each(['auto', 'storage'] as const)('legacy artifacts in %s mode', (mode) => {
    it.each([
      ['root', '/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts&namespace=team-a'],
      [
        'encoded query',
        '/artifacts/get?source=minio&bucket=mlpipeline&key=%61rtifacts%2Fvictim%2Fmodel.txt&namespace=team-a',
      ],
      [
        'preview',
        '/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/victim/model.txt&namespace=team-a',
      ],
      ['download', '/artifacts/minio/mlpipeline/artifacts/victim/model.txt?namespace=team-a'],
      [
        'directory preview',
        '/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/victim/&peek=256&namespace=team-a',
      ],
      [
        'directory download',
        '/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/victim/&download=true&namespace=team-a',
      ],
    ])('retains ownership validation for shared legacy artifact %s', async (_kind, route) => {
      vi.mocked(getK8sSecret).mockResolvedValue(approval());
      vi.mocked(validateArtifactNamespace).mockResolvedValue({
        valid: false,
        reason: 'custom-root-requires-namespace-prefix',
      });
      await requests(app({ mode }))
        .get(route)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(403);
      expect(validateArtifactNamespace).toHaveBeenCalledOnce();
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    });
  });

  it.each([
    ['minio', 'custom-bucket'],
    ['s3', 'mlpipeline'],
    ['s3', 'research-data'],
    ['gcs', 'mlpipeline'],
  ])(
    'keeps artifacts/ in custom %s://%s storage under its identity grant',
    async (source, bucket) => {
      await requests(app({ mode: 'storage' }))
        .get(`/artifacts/get?${artifactQuery({ source, bucket, key: 'artifacts/model.txt' })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(200);
      expect(validateArtifactNamespace).not.toHaveBeenCalled();
      expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('explains where to investigate pending managed storage readiness', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue(approval('pending'));
    const result = await requests(app({ mode: 'auto' }))
      .get(`/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(503);
    expect(result.headers['retry-after']).toBe('5');
    expect(content(result)).toContain('profile controller logs');
    expect(content(result)).toContain('namespace artifact service rollout');
    expect(content(result)).not.toContain('Retry shortly');
    expect(forwarded).toHaveLength(0);
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('keeps unrelated stores under prefix authorization while managed storage is pending', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue(approval('pending'));
    vi.mocked(validateArtifactNamespace).mockResolvedValue({ valid: true });
    await requests(app({ mode: 'auto' }))
      .get(`/artifacts/get?${artifactQuery({ key: 'private-artifacts/team-a/model' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(200);
    expect(validateArtifactNamespace).toHaveBeenCalled();
    expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
    expect(localFetch).not.toHaveBeenCalled();
  });

  it.each(['', '   '])(
    'classifies an empty record payload as configuration failure',
    async (payload) => {
      vi.mocked(getK8sSecret).mockResolvedValue(payload);
      const result = await requests(app({ mode: 'auto' }))
        .get(`/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline' })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(503);
      expect(content(result)).toContain('repair the controller-managed authorization record');
      expect(result.headers['retry-after']).toBeUndefined();
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it.each(['length', 'chunked', 'gzip'] as const)(
    'replaces an incompatible receiver %s body with an actionable plain error',
    async (framing) => {
      legacyReceiver = true;
      legacyFraming = framing;
      const result = await requests(app())
        .get(`/artifacts/get?${artifactQuery()}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(503);
      const body = content(result);
      expect(body).toBe(
        'Namespace artifact service upgrade required. Wait for its rollout to complete; if this persists, check the profile controller and namespace artifact service deployment. No broader credentials were used.',
      );
      expect(result.headers['x-kfp-artifact-proxy-error']).toBe('receiver-upgrade-required');
      expect(result.headers['content-type']).toMatch(/^text\/plain/);
      expect(result.headers['content-disposition']).toBe('attachment');
      expect(result.headers['x-content-type-options']).toBe('nosniff');
      expect(result.headers['content-length']).toBe(String(Buffer.byteLength(body)));
      expect(result.headers['retry-after']).toBe('5');
      for (const header of ['content-encoding', 'content-range', 'transfer-encoding', 'set-cookie'])
        expect(result.headers[header]).toBeUndefined();
      expect(result.headers.etag).not.toBe('old-receiver-body');
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it.each(['unauthorized', 'forbidden', 'malformed'])(
    'reports actionable %s record errors without downgrading authorization',
    async (failure) => {
      if (failure !== 'malformed')
        vi.mocked(getK8sSecret).mockRejectedValue({
          code: failure === 'forbidden' ? 403 : 401,
          body: 'sensitive-secret',
        });
      else vi.mocked(getK8sSecret).mockResolvedValue('{}');
      const result = await requests(app({ mode: 'auto' }))
        .get(`/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline' })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .expect(503);
      expect(result.headers['retry-after']).toBeUndefined();
      expect(content(result)).toContain(
        failure === 'malformed'
          ? 'repair the controller-managed authorization record'
          : 'restore UI service-account authentication and Secret-read access',
      );
      expect(content(result)).not.toContain('Retry shortly');
      expect(content(result)).not.toContain('sensitive-secret');
      expect(validateArtifactNamespace).not.toHaveBeenCalled();
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('streams a compatible download before the namespace receiver finishes', async () => {
    receiverStream = true;
    let receivedFirstChunk = false;
    const result = await requests(app())
      .get('/artifacts/s3/research-data/historical/model.txt?namespace=team-a')
      .set('kubeflow-userid', 'alice@example.com')
      .buffer(true)
      .parse((res, callback) => {
        const chunks: Buffer[] = [];
        res.on('data', (chunk: Buffer) => {
          chunks.push(chunk);
          if (!receivedFirstChunk) {
            receivedFirstChunk = true;
            finishReceiverStream();
          }
        });
        res.on('end', () => callback(null, Buffer.concat(chunks)));
        res.on('error', callback);
      })
      .timeout(3000)
      .expect(200);
    expect(receivedFirstChunk).toBe(true);
    expect(result.body).toEqual(
      Buffer.concat([Buffer.alloc(128 * 1024, 'a'), Buffer.alloc(128 * 1024, 'b')]),
    );
    expect(result.headers['x-kfp-artifact-proxy-error']).toBeUndefined();
    expect(result.headers['content-disposition']).toBe('attachment');
    expect(localFetch).not.toHaveBeenCalled();
  });

  it.each([200, 403, 503])(
    'keeps missing-record reads and receiver status %s inside the namespace proxy',
    async (status) => {
      vi.mocked(validateArtifactNamespace).mockResolvedValue({ valid: true });
      receiverStatus = status;
      const result = await requests(app({ mode: 'auto' }))
        .get(
          `/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline', key: 'private-artifacts/team-a/model' })}`,
        )
        .set('kubeflow-userid', 'alice@example.com')
        .expect(status);
      expect(content(result)).toBe('namespace storage response');
      expect(validateArtifactNamespace).toHaveBeenCalled();
      expect(serviceGetter).toHaveBeenCalledExactlyOnceWith('team-a');
      expect(forwarded).toHaveLength(1);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it('rejects a foreign artifact when the automatic authorization record is absent', async () => {
    await requests(app({ mode: 'auto' }))
      .get(
        `/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline', key: 'private-artifacts/team-b/model' })}`,
      )
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(validateArtifactNamespace).toHaveBeenCalled();
    expect(serviceGetter).not.toHaveBeenCalled();
    expect(localFetch).not.toHaveBeenCalled();
  });

  it('rejects changed coordinates on a missing-record request before namespace proxying', async () => {
    vi.mocked(validateArtifactNamespace).mockResolvedValue({ valid: true });
    await requests(
      app({
        mode: 'auto',
        mutate: (req, _res, next) => {
          req.url = req.url.replace('team-a', 'team-b');
          next();
        },
      }),
    )
      .get(`/artifacts/get?${artifactQuery({ source: 'minio', bucket: 'mlpipeline' })}`)
      .set('kubeflow-userid', 'alice@example.com')
      .expect(403);
    expect(forwarded).toHaveLength(0);
    expect(localFetch).not.toHaveBeenCalled();
  });

  it.each(['storage', 'key-prefix'] as const)(
    'rejects a tenant Service alias targeting another namespace in %s mode',
    async (mode) => {
      // A tenant-a Service alias resolves to tenant-b, which must not use its credentials.
      vi.mocked(validateArtifactNamespace).mockResolvedValue({ valid: true });
      receivingNamespace = 'team-b';
      await requests(app({ mode }))
        .get(`/artifacts/get?${artifactQuery({ key: 'private-artifacts/team-a/model.txt' })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .set('x-kfp-artifact-namespace', 'team-b')
        .expect(403);
      expect(forwarded).toHaveLength(0);
      expect(localFetch).not.toHaveBeenCalled();
    },
  );

  it.each(['storage', 'key-prefix'] as const)(
    'replaces spoofed client bindings with the authorized namespace in %s mode',
    async (mode) => {
      vi.mocked(validateArtifactNamespace).mockResolvedValue({ valid: true });
      await requests(app({ mode }))
        .get(`/artifacts/get?${artifactQuery({ key: 'private-artifacts/team-a/model.txt' })}`)
        .set('kubeflow-userid', 'alice@example.com')
        .set('x-kfp-artifact-namespace', 'team-b')
        .expect(200);
      expect(forwarded).toHaveLength(1);
    },
  );

  it('rejects requests without a binding at the receiving service', async () => {
    await requests(endpoint).get('/artifacts/get').expect(403);
    expect(forwarded).toHaveLength(0);
  });

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

  it('pins the namespace and object before choosing the proxy target', async () => {
    let incoming: Request | undefined;
    serviceGetter.mockImplementation(() => {
      if (!incoming) throw new Error('Expected authorized request');
      incoming.url = `/artifacts/get?${artifactQuery({ namespace: 'team-b', key: 'victim/object' })}`;
      return endpoint;
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

  it('rejects malformed percent-encoding in download paths', async () => {
    await requests(app())
      .get('/artifacts/s3/research-data/historical%ZZmodel?namespace=team-a')
      .set('kubeflow-userid', 'alice@example.com')
      .expect(400);
    expect(forwarded).toHaveLength(0);
  });

  it('preserves explicit key-prefix authorization', async () => {
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
