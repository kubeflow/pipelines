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

import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';
import { createServer, type Server } from 'node:http';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { commonSetup } from './test-helper.js';

vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn(), getPod: vi.fn() }));

describe('ML Metadata proxy', () => {
  const { argv } = commonSetup();
  let upstream: Server;
  let upstreamPort: number;
  let upstreamRequests = 0;

  beforeAll(async () => {
    upstream = createServer((_req, res) => {
      upstreamRequests++;
      res.writeHead(200, { 'Content-Type': 'text/plain' }).end('metadata response');
    });
    await new Promise<void>((resolve) => upstream.listen(0, '127.0.0.1', resolve));
    const address = upstream.address();
    if (!address || typeof address === 'string') throw new Error('Missing upstream port');
    upstreamPort = address.port;
  });

  afterAll(async () => {
    await new Promise<void>((resolve, reject) =>
      upstream.close((error) => (error ? reject(error) : resolve())),
    );
  });

  function app(authEnabled: boolean): UIServer {
    const options = loadConfigs(argv, {
      METADATA_ENVOY_SERVICE_SERVICE_HOST: '127.0.0.1',
      METADATA_ENVOY_SERVICE_SERVICE_PORT: String(upstreamPort),
      METADATA_ENVOY_SERVICE_SERVICE_SCHEME: 'http',
    });
    options.auth.enabled = authEnabled;
    return new UIServer(options);
  }

  it('blocks MLMD read and write methods in multi-user mode before reaching the upstream', async () => {
    const ui = app(true);
    const before = upstreamRequests;
    for (const method of ['GetArtifacts', 'PutContexts']) {
      await requests(ui.app)
        .post(`/ml_metadata.MetadataStoreService/${method}`)
        .send(Buffer.from('synthetic request'))
        .expect(403);
    }
    expect(upstreamRequests).toBe(before);
  });

  it('retains the standalone MLMD proxy', async () => {
    const ui = app(false);
    const before = upstreamRequests;
    await requests(ui.app)
      .post('/ml_metadata.MetadataStoreService/GetArtifacts')
      .send(Buffer.from('synthetic request'))
      .expect(200, 'metadata response');
    expect(upstreamRequests).toBe(before + 1);
  });
});
