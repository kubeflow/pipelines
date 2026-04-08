// Copyright 2025 The Kubeflow Authors
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

import {
  vi,
  describe,
  it,
  expect,
  afterAll,
  afterEach,
  beforeEach,
  MockInstance,
} from 'vitest';
import * as minio from 'minio';
import { PassThrough } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { commonSetup } from './test-helper.js';

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('../k8s-helper.js');

const mockedFetch = vi.fn();
vi.stubGlobal('fetch', mockedFetch);

describe('/artifacts authorization', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  const artifactContent = 'hello world';

  beforeEach(() => {
    const mockedMinioClient = MinioClient as any;
    mockedMinioClient.mockImplementation(function () {
      return {
        getObject: async (bucket: string, key: string) => {
          const objStream = new PassThrough();
          objStream.end(artifactContent);
          if (bucket === 'ml-pipeline' && key === 'hello/world.txt') {
            return objStream;
          } else {
            throw new Error(`Unable to retrieve ${bucket}/${key} artifact.`);
          }
        },
      };
    });
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  describe('when auth is disabled', () => {
    it('allows artifact access without namespace parameter', async () => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
    });

    it('allows artifact access with any namespace parameter', async () => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=any-namespace',
        )
        .expect(200, artifactContent);
    });
  });

  describe('when auth is enabled', () => {
    it('requires namespace parameter when auth is enabled', async () => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      const response = await request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .set('kubeflow-userid', 'user@example.com')
        .expect(400);
      expect(response.text).toContain('Namespace parameter is required');
    });

    it('rejects unauthenticated users before checking namespace', async () => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      const response = await request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(401);
      expect(response.text).toContain('Authentication required');
    });

    it('rejects requests with invalid namespace format', async () => {
      mockedFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=../../../etc',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(400);
      expect(response.text).toContain('Invalid namespace');
    });

    it('rejects unauthorized cross-namespace access', async () => {
      // Mock fetch to resolve with 403 (HTTP errors resolve, not reject).
      // The Swagger client checks response.status and throws the response
      // object when status is not 2xx. parseError then extracts the message.
      mockedFetch.mockResolvedValue({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        url: '/apis/v1beta1/auth',
        json: () =>
          Promise.resolve({
            error: 'User is not authorized to GET VIEWERS in namespace other-namespace',
            details: {},
          }),
        text: () => Promise.resolve('User is not authorized'),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=other-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('not authorized');
    });

    it('allows authorized namespace access', async () => {
      // Mock fetch to resolve successfully (simulates auth service approval)
      mockedFetch.mockImplementation(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({}),
          text: () => Promise.resolve(''),
        }),
      );

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, artifactContent);
    });

    it('rejects XSS key longer than 1024 characters even when auth passes', async () => {
      mockedFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(''),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=s3&namespace=my-namespace&bucket=ml-pipeline&key=' +
            '<script>alert(1)</script>' +
            'a'.repeat(1025),
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(500, 'Object key too long');
    });

    it('enforces auth before proxy when ARTIFACTS_SERVICE_PROXY_ENABLED=true', async () => {
      // When proxy is enabled, auth middleware on the catch-all /artifacts/*
      // route must execute before the proxy handler forwards the request.
      // Unauthenticated users must be rejected with 401, never proxied.
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
        ARTIFACTS_SERVICE_PROXY_NAME: 'ml-pipeline-ui-artifact',
        ARTIFACTS_SERVICE_PROXY_PORT: '80',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .expect(401);
      expect(response.text).toContain('Authentication required');
    });

    it('fails for secret-backed provider when RBAC denies secret access', async () => {
      // When providerInfo.Params.fromEnv === 'false', the handler calls
      // getK8sSecret() to retrieve credentials from a Kubernetes secret.
      // Since ml-pipeline-ui ClusterRole no longer has secrets:get/list,
      // getK8sSecret() is rejected by RBAC and the request fails with 500.
      mockedFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(''),
      });

      // Mock getK8sSecret to simulate RBAC denial
      const k8sHelper = await import('../k8s-helper.js');
      (k8sHelper.getK8sSecret as any) = vi.fn().mockRejectedValue(
        new Error('secrets "mlpipeline-minio-artifact" is forbidden'),
      );

      const configurations = loadConfigs(argv, {
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const providerInfo = {
        Params: {
          accessKeyKey: 'accesskey',
          secretKeyKey: 'secretkey',
          secretName: 'mlpipeline-minio-artifact',
          endpoint: 'minio-service.kubeflow',
          disableSSL: 'true',
          fromEnv: 'false',
        },
        Provider: 'minio',
      };

      const request = requests(app.app);
      const response = await request
        .get(
          `/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace&providerInfo=${encodeURIComponent(JSON.stringify(providerInfo))}`,
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(500);
      expect(response.text).toContain('Failed to initialize Minio Client');
    });
  });

  describe('security logging', () => {
    let consoleSpy: MockInstance;

    beforeEach(() => {
      consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    });

    afterEach(() => {
      consoleSpy.mockRestore();
    });

    it('logs unauthorized access attempts with user info', async () => {
      // Mock fetch to resolve with 403 — same pattern as auth rejection test
      mockedFetch.mockResolvedValue({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        url: '/apis/v1beta1/auth',
        json: () =>
          Promise.resolve({
            error: 'User is not authorized to GET VIEWERS in namespace unauthorized-ns',
            details: {},
          }),
        text: () => Promise.resolve('User is not authorized'),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=unauthorized-ns',
        )
        .set('kubeflow-userid', 'attacker@example.com')
        .expect(403);

      expect(consoleSpy).toHaveBeenCalled();
      const logCall = consoleSpy.mock.calls.find(
        (call) => call[0] && call[0].includes('[SECURITY]'),
      );
      expect(logCall).toBeDefined();
      if (logCall) {
        expect(logCall[0]).toContain('attacker@example.com');
        expect(logCall[0]).toContain('unauthorized-ns');
      }
    });

    it('logs unauthenticated access attempts', async () => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(401);

      expect(consoleSpy).toHaveBeenCalled();
      const logCall = consoleSpy.mock.calls.find(
        (call) => call[0] && call[0].includes('[SECURITY]') && call[0].includes('Unauthenticated'),
      );
      expect(logCall).toBeDefined();
    });
  });
});
