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
import express from 'express';
import type { Server } from 'node:http';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import * as artifacts from '../handlers/artifacts.js';
import { getConfigMap } from '../k8s-helper.js';
import { asText, commonSetup } from './test-helper.js';

vi.mock('../k8s-helper.js', () => ({
  getArgoWorkflow: vi.fn(),
  getConfigMap: vi.fn(),
  getK8sSecret: vi.fn(),
  getPod: vi.fn(),
  getPodLogs: vi.fn(),
  getServerNamespace: vi.fn(),
}));

// Real HTTP transports and production validator/serving handlers. The API fixture supplies
// identity and metadata responses; it is not a Kubernetes authorizer or object-access authority.
describe('authenticated HTTP artifact routing with a namespace receiver', () => {
  const { argv } = commonSetup();
  const listeners: Server[] = [];
  const uis: UIServer[] = [];
  let storeOrigin: string;
  let apiOrigin: string;
  let receiverOrigin: string;
  let redirect: string | undefined;
  let storageRequests: { path: string; authorization?: string; user?: string }[];
  let receiverRequests: string[];
  let apiRequests: { path: string; namespace: unknown; user?: string }[];

  async function listen(app: express.Express, onRequest?: (path: string) => void): Promise<string> {
    const server = await new Promise<Server>((resolve) => {
      const running = app.listen(0, '127.0.0.1', () => resolve(running));
    });
    if (onRequest) server.on('request', (request) => onRequest(request.url || ''));
    listeners.push(server);
    const address = server.address();
    if (!address || typeof address === 'string') throw new Error('Expected TCP listener');
    return `http://127.0.0.1:${address.port}`;
  }

  beforeEach(async () => {
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    redirect = undefined;
    storageRequests = [];
    apiRequests = [];
    receiverRequests = [];
    const store = express();
    store.use((req, res) => {
      storageRequests.push({
        path: req.url,
        authorization: req.get('authorization'),
        user: req.get('kubeflow-userid'),
      });
      if (redirect && req.path.endsWith('/history.txt')) {
        res.redirect(302, redirect);
      } else {
        res.type('text/plain').send('historical artifact bytes');
      }
    });
    storeOrigin = await listen(store);

    const api = express();
    api.use((req, res) => {
      apiRequests.push({
        path: req.path,
        namespace: req.query.namespace,
        user: req.get('kubeflow-userid'),
      });
      if (req.query.namespace !== 'team-a' || req.get('kubeflow-userid') !== 'a@example.com') {
        res.status(403).json({ error: 'fixture namespace access denied' });
      } else if (req.path === '/apis/v2/artifacts') {
        res.json({ artifacts: [{ artifact_id: 'fixture-metadata-row' }] });
      } else if (req.path === '/apis/v2/auth') {
        res.json({});
      } else {
        res.sendStatus(404);
      }
    });
    apiOrigin = await listen(api);

    const receiver = new UIServer(
      loadConfigs(argv, {
        ENABLE_AUTHZ: 'false',
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'false',
        HTTP_BASE_URL: `${storeOrigin}/reports/`,
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_AUTHORIZATION_DEFAULT_VALUE: 'receiver-only-token',
        ALLOWED_ARTIFACT_DOMAIN_REGEX: '^(127\\.0\\.0\\.1|localhost)$',
      }),
    );
    uis.push(receiver);
    receiverOrigin = await listen(receiver.app, (path) => receiverRequests.push(path));
  });

  afterEach(async () => {
    await Promise.all(
      listeners.splice(0).map(
        (server) =>
          new Promise<void>((resolve) => {
            server.close(() => resolve());
            server.closeAllConnections();
          }),
      ),
    );
    await Promise.all(uis.splice(0).map((ui) => ui.close()));
  });

  function central(proxy: boolean): UIServer {
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(() => receiverOrigin);
    const api = new URL(apiOrigin);
    const ui = new UIServer(
      loadConfigs(argv, {
        ENABLE_AUTHZ: 'true',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
        ML_PIPELINE_SERVICE_HOST: api.hostname,
        ML_PIPELINE_SERVICE_PORT: api.port,
        ARTIFACTS_SERVICE_PROXY_ENABLED: String(proxy),
        // A proxied deployment deliberately has no central HTTP base or storage credential.
        HTTP_BASE_URL: proxy ? '' : `${storeOrigin}/reports/`,
        HTTP_AUTHORIZATION_KEY: proxy ? '' : 'Authorization',
        HTTP_AUTHORIZATION_DEFAULT_VALUE: proxy ? '' : 'direct-only-token',
        ALLOWED_ARTIFACT_DOMAIN_REGEX: '^(127\\.0\\.0\\.1|localhost)$',
      }),
    );
    uis.push(ui);
    return ui;
  }

  function query(key = 'reports/customer-training/history.txt') {
    return { source: 'http', bucket: new URL(storeOrigin).host, key, namespace: 'team-a' };
  }

  it.each([false, true])(
    'serves a custom root and same-base signed redirect (proxy=%s)',
    async (proxy) => {
      redirect = '/reports/signed/download.txt?signature=fixture';
      const ui = central(proxy);
      const response = await asText(
        requests(ui.app)
          .get('/artifacts/get')
          .query(query())
          .set('kubeflow-userid', 'a@example.com'),
      ).expect(200);
      expect(response.body).toBe('historical artifact bytes');
      expect(response.headers['content-disposition']).toContain('attachment');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(storageRequests).toEqual([
        {
          path: '/reports/customer-training/history.txt',
          authorization: proxy ? 'receiver-only-token' : 'direct-only-token',
          user: undefined,
        },
        {
          path: redirect,
          authorization: proxy ? 'receiver-only-token' : 'direct-only-token',
          user: undefined,
        },
      ]);
      expect(apiRequests.map((request) => request.path)).toEqual([
        '/apis/v2/auth',
        '/apis/v2/artifacts',
      ]);
      expect(
        apiRequests.every(
          (request) => request.user === 'a@example.com' && request.namespace === 'team-a',
        ),
      ).toBe(true);
    },
  );

  it('preserves a raw HTTP download through the namespace receiver', async () => {
    const ui = central(true);
    const response = await asText(
      requests(ui.app)
        .get('/artifacts/get')
        .query({ ...query(), download: true })
        .set('kubeflow-userid', 'a@example.com'),
    ).expect(200);
    expect(response.body).toBe('historical artifact bytes');
    expect(storageRequests).toEqual([
      {
        path: '/reports/customer-training/history.txt',
        authorization: 'receiver-only-token',
        user: undefined,
      },
    ]);
  });

  it.each([false, true])(
    'rejects a known foreign namespace prefix before serving (proxy=%s)',
    async (proxy) => {
      const ui = central(proxy);
      await requests(ui.app)
        .get('/artifacts/get')
        .query(query('private-artifacts/team-b/secret'))
        .set('kubeflow-userid', 'a@example.com')
        .expect(403);
      expect(storageRequests).toEqual([]);
      expect(apiRequests.map((request) => request.path)).toEqual(['/apis/v2/auth']);
    },
  );

  it.each([false, true])(
    'retains HTTP base and malformed redirect boundaries (proxy=%s)',
    async (proxy) => {
      const ui = central(proxy);
      for (const location of [
        '/outside/secret',
        '/reports/..%2Fsecret',
        '/reports/file%5C..%5Csecret',
        storeOrigin.replace('127.0.0.1', 'localhost') + '/reports/secret',
      ]) {
        redirect = location;
        storageRequests = [];
        await requests(ui.app)
          .get('/artifacts/get')
          .query(query())
          .set('kubeflow-userid', 'a@example.com')
          .expect(400);
        expect(storageRequests).toHaveLength(1);
      }
    },
  );

  it.each([false, true])('rejects unsafe request keys before storage (proxy=%s)', async (proxy) => {
    const ui = central(proxy);
    for (const key of ['reports/../secret', 'reports/folder\\secret']) {
      await requests(ui.app)
        .get('/artifacts/get')
        .query(query(key))
        .set('kubeflow-userid', 'a@example.com')
        .expect(403);
    }
    expect(storageRequests).toEqual([]);
  });

  it('retains authentication and namespace authorization before proxying', async () => {
    const ui = central(true);
    await requests(ui.app).get('/artifacts/get').query(query()).expect(401);
    await requests(ui.app)
      .get('/artifacts/get')
      .query({ ...query(), namespace: 'team-b' })
      .set('kubeflow-userid', 'a@example.com')
      .expect(403);
    expect(storageRequests).toEqual([]);
    expect(apiRequests.map((request) => request.path)).toEqual(['/apis/v2/auth']);
  });

  it.each([
    ['http', false],
    ['http', true],
    ['s3', false],
    ['s3', true],
  ] as const)(
    'rejects unsafe %s keys without metadata configuration (proxy=%s)',
    async (source, proxy) => {
      const options = loadConfigs(argv, {
        ENABLE_AUTHZ: 'true',
        HTTP_BASE_URL: `${storeOrigin}/reports/`,
      });
      const ui = express();
      ui.get(
        '/artifacts/get',
        artifacts.getArtifactsAuthMiddleware(async () => undefined, true, 'kubeflow-userid'),
        artifacts.getArtifactsProxyHandler({
          enabled: proxy,
          allowedDomain: '.*',
          namespacedServiceGetter: () => receiverOrigin,
        }),
        artifacts.getArtifactsHandler({
          artifactsConfigs: options.artifacts,
          useParameter: false,
          tryExtract: false,
          options,
        }),
      );
      for (const key of ['reports/../secret', 'reports/./secret']) {
        await requests(ui)
          .get('/artifacts/get')
          .query({ source, bucket: 'storage-bucket', key, namespace: 'team-a' })
          .set('kubeflow-userid', 'a@example.com')
          .expect(403);
      }
      expect(receiverRequests).toEqual([]);
      expect(storageRequests).toEqual([]);
      expect(apiRequests).toEqual([]);
    },
  );
});
