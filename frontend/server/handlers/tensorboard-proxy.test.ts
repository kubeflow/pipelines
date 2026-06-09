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

import express from 'express';
import requests from 'supertest';
import { describe, expect, it, vi } from 'vitest';
import { ErrorDetails } from '../utils.js';
import registerTensorboardProxy, {
  buildTensorboardProxyTarget,
  buildTensorboardProxyUpstreamPath,
  createTensorboardProxyPath,
  getTensorboardProxyBasePath,
  parseTensorboardProxyPayload,
  parseTensorboardProxyRequest,
} from './tensorboard-proxy.js';

const TENSORBOARD_PROXY_PREFIX = '/apps/tensorboard/proxy/';
const TENSORBOARD_PROXY_SIGNING_SECRET = 'tensorboard-proxy-test-secret';

describe('tensorboard-proxy', () => {
  it('creates scoped proxy paths that round-trip through the parser', () => {
    const proxyPath = createTensorboardProxyPath(
      'test-ns',
      'viewer-abcdefg',
      TENSORBOARD_PROXY_SIGNING_SECRET,
    );
    expect(proxyPath.startsWith('apps/tensorboard/proxy/')).toBe(true);

    const parsedRequest = parseTensorboardProxyRequest(TENSORBOARD_PROXY_PREFIX, `/${proxyPath}`);
    expect(parsedRequest).toBeDefined();
    expect(parsedRequest?.proxyPath).toBe('/');

    const parsedPayload = parseTensorboardProxyPayload(
      parsedRequest!.token,
      TENSORBOARD_PROXY_SIGNING_SECRET,
    );
    expect(parsedPayload).toEqual({
      namespace: 'test-ns',
      viewerName: 'viewer-abcdefg',
    });
  });

  it('normalizes custom cluster domains when building proxy targets', () => {
    expect(buildTensorboardProxyTarget('test-ns', 'viewer-abcdefg', 'cluster.test')).toBe(
      'http://viewer-abcdefg-service.test-ns.cluster.test:80',
    );
    expect(buildTensorboardProxyTarget('test-ns', 'viewer-abcdefg', '.svc.cluster.local')).toBe(
      'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80',
    );
  });

  it('rewrites upstream TensorBoard paths without exposing the service host', () => {
    expect(buildTensorboardProxyUpstreamPath('viewer-abcdefg', '/', '')).toBe(
      '/tensorboard/viewer-abcdefg/',
    );
    expect(
      buildTensorboardProxyUpstreamPath('viewer-abcdefg', '/data/logdir', 'experiment=1'),
    ).toBe('/tensorboard/viewer-abcdefg/data/logdir?experiment=1');
  });

  it('extracts the scoped proxy base path from a referer', () => {
    const proxyPath = createTensorboardProxyPath(
      'test-ns',
      'viewer-abcdefg',
      TENSORBOARD_PROXY_SIGNING_SECRET,
    );
    expect(
      getTensorboardProxyBasePath(
        TENSORBOARD_PROXY_PREFIX,
        `http://kfp.example/pipeline/${proxyPath}data/logdir`,
      ),
    ).toBe(`/pipeline/${proxyPath.slice(0, -1)}`);
  });

  it('rejects invalid proxy tokens before proxying upstream traffic', async () => {
    const app = express();
    registerTensorboardProxy(
      app,
      '/pipeline',
      {
        clusterDomain: '.svc.cluster.local',
        proxySigningSecret: TENSORBOARD_PROXY_SIGNING_SECRET,
        tfImageName: 'tensorflow/tensorflow',
      },
      vi.fn(async () => undefined),
    );

    await requests(app).get('/apps/tensorboard/proxy/not-a-valid-token/').expect(403);
  });

  it('rejects proxy path traversal before authorizing the request', async () => {
    const app = express();
    const authorizeFn = vi.fn(async () => undefined);
    const proxyPath = createTensorboardProxyPath(
      'test-ns',
      'viewer-abcdefg',
      TENSORBOARD_PROXY_SIGNING_SECRET,
    );

    registerTensorboardProxy(
      app,
      '/pipeline',
      {
        clusterDomain: '.svc.cluster.local',
        proxySigningSecret: TENSORBOARD_PROXY_SIGNING_SECRET,
        tfImageName: 'tensorflow/tensorflow',
      },
      authorizeFn,
    );

    await requests(app)
      .get(`/${proxyPath}../data/plugin/scalars/scalars`)
      .expect(400, 'Invalid TensorBoard proxy path');

    expect(authorizeFn).not.toHaveBeenCalled();
  });

  it('rewrites absolute TensorBoard subrequests using the referer-scoped token', async () => {
    const app = express();
    const authorizeFn = vi.fn(
      async (): Promise<ErrorDetails | undefined> => ({ message: 'denied', additionalInfo: '' }),
    );
    const proxyPath = createTensorboardProxyPath(
      'test-ns',
      'viewer-abcdefg',
      TENSORBOARD_PROXY_SIGNING_SECRET,
    );
    registerTensorboardProxy(
      app,
      '/pipeline',
      {
        clusterDomain: '.svc.cluster.local',
        proxySigningSecret: TENSORBOARD_PROXY_SIGNING_SECRET,
        tfImageName: 'tensorflow/tensorflow',
      },
      authorizeFn,
    );

    await requests(app)
      .get('/data/plugin/scalars/scalars')
      .set('Referer', `http://127.0.0.1/pipeline/${proxyPath}`)
      .expect(403, 'Access denied to namespace');

    expect(authorizeFn).toHaveBeenCalledWith(
      expect.objectContaining({
        namespace: 'test-ns',
      }),
      expect.objectContaining({
        url: expect.stringContaining('/pipeline/apps/tensorboard/proxy/'),
      }),
    );
  });
});
