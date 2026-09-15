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
import type { Server } from 'http';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import * as artifacts from '../handlers/artifacts.js';
import { getConfigMap } from '../k8s-helper.js';
import { commonSetup } from './test-helper.js';

vi.mock('../k8s-helper.js', () => ({
  getArgoWorkflow: vi.fn(),
  getConfigMap: vi.fn(),
  getK8sSecret: vi.fn(),
  getPod: vi.fn(),
  getPodLogs: vi.fn(),
  getServerNamespace: vi.fn(),
}));

describe('artifact proxy ownership with the production validator', () => {
  const { argv } = commonSetup();
  let app: UIServer;
  let downstream: Server;
  let forwarded: string[];
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    forwarded = [];
    // Model a tenant service with shared storage access. The central route must deny
    // the victim URI before sending anything to this service.
    const service = express();
    service.use((req, res) => {
      forwarded.push(req.url);
      res.send(req.url.includes('victim') ? 'victim-data' : 'own-data');
    });
    downstream = await new Promise<Server>((resolve) => {
      const server = service.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = downstream.address();
    if (!address || typeof address === 'string') throw new Error('Expected TCP listener');
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(
      () => `http://127.0.0.1:${address.port}`,
    );
    // API responses are fixtures; ownership validation and proxy transport are real.
    fetchSpy = vi.fn(async (input: string | URL | Request) => {
      const url = new URL(String(input));
      return new Response(
        JSON.stringify(
          url.pathname.endsWith('/artifacts')
            ? { artifacts: [{ artifact_id: 'caller-owned-import' }] }
            : {},
        ),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      );
    });
    vi.stubGlobal('fetch', fetchSpy);
    app = new UIServer(
      loadConfigs(argv, {
        ENABLE_AUTHZ: 'true',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
      }),
    );
  });

  afterEach(async () => {
    vi.unstubAllGlobals();
    if (downstream) await new Promise<void>((resolve) => downstream.close(() => resolve()));
    if (app) await app.close();
  });

  it.each([
    '/artifacts/get?source=minio&bucket=shared&key=custom/victim&namespace=team-a',
    '/artifacts/get?source=minio&bucket=shared&key=custom/victim&namespace=team-a&download=true',
    '/artifacts/minio/shared/custom/victim?namespace=team-a',
    '/artifacts/get?source=s3&bucket=shared&key=custom/victim&namespace=team-a',
    '/artifacts/get?source=gcs&bucket=shared&key=custom/victim&namespace=team-a',
    '/artifacts/get?source=https&bucket=store.example&key=custom/victim&namespace=team-a',
  ])('denies an imported custom-root URI before proxying: %s', async (path) => {
    await requests(app.app).get(path).set('kubeflow-userid', 'a@example.com').expect(403);
    expect(forwarded).toEqual([]);
    expect(
      fetchSpy.mock.calls.some(([url]) => String(url).includes('/apis/v2beta1/artifacts')),
    ).toBe(true);
  });

  it.each([false, true])('still serves own namespace prefix (download=%s)', async (download) => {
    const response = await requests(app.app)
      .get(
        `/artifacts/get?source=minio&bucket=shared&key=private-artifacts/team-a/own&namespace=team-a&download=${download}`,
      )
      .set('kubeflow-userid', 'a@example.com')
      .expect(200);
    expect(response.body.toString()).toBe('own-data');
    expect(forwarded).toHaveLength(1);
  });
});
