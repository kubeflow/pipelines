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

import { vi, describe, it, expect, afterAll, afterEach, beforeEach, MockInstance } from 'vitest';
import * as minio from 'minio';
import { PassThrough } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { commonSetup } from './test-helper.js';

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('../k8s-helper.js');

vi.mock('../gcs-helper.js', () => ({
  getGCSClient: () => Promise.resolve({}),
  listGCSObjectNames: ({ bucket, prefix }: { bucket: string; prefix: string }) =>
    bucket === 'ml-pipeline' && prefix === 'hello/world.txt'
      ? Promise.resolve(['hello/world.txt'])
      : Promise.resolve([]),
  downloadGCSObjectStream: () => {
    const s = new PassThrough();
    s.end('hello world');
    return Promise.resolve(s);
  },
}));

const mockedValidateArtifactNamespace = vi.fn();
vi.mock('../helpers/mlmd-validator.js', () => ({
  validateArtifactNamespace: (...args: unknown[]) => mockedValidateArtifactNamespace(...args),
  buildArtifactUri: (source: string, bucket: string, key: string) => {
    const scheme = source === 'gcs' ? 'gs' : source;
    return `${scheme}://${bucket}/${key}`;
  },
}));

const mockedFetch = vi.fn();
vi.stubGlobal('fetch', mockedFetch);

describe('/artifacts authorization', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  const artifactContent = 'hello world';

  beforeEach(() => {
    // Isolate each test from prior call records.
    mockedValidateArtifactNamespace.mockClear();
    mockedFetch.mockClear();

    // Default: MLMD validation passes (artifact belongs to claimed namespace)
    mockedValidateArtifactNamespace.mockResolvedValue({ valid: true });

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

    it('ignores secret-backed provider info for a customer namespace and never reads the Secret', async () => {
      // When providerInfo.Params.fromEnv === 'false', the pipeline names a
      // Kubernetes Secret to source object-store credentials from. The
      // ml-pipeline-ui service account may only read Secrets from its own
      // namespace, so for a customer namespace the provider info must be
      // ignored entirely (never calling getK8sSecret). Credential resolution
      // falls back to the server's own environment credentials (SeaweedFS in
      // the kubeflow namespace) or, when enabled, the per-namespace artifact
      // proxy. See: https://github.com/kubeflow/pipelines/pull/12860
      mockedFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(''),
      });

      const k8sHelper = await import('../k8s-helper.js');
      const getK8sSecretMock = k8sHelper.getK8sSecret as unknown as MockInstance;
      getK8sSecretMock.mockClear();

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SSL: 'false',
        MINIO_SECRET_KEY: 'minio123',
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
          `/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace&providerInfo=${encodeURIComponent(
            JSON.stringify(providerInfo),
          )}`,
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200);
      expect(response.text).toContain(artifactContent);
      expect(getK8sSecretMock).not.toHaveBeenCalled();
    });
  });

  describe('IDOR prevention (cross-namespace artifact access via namespace swap)', () => {
    const authEnabledConfigs = () => {
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
      return configurations;
    };

    // Mock auth to pass (user has access to the claimed namespace)
    const mockAuthPass = () => {
      mockedFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(''),
      });
    };

    it('rejects artifact access when MLMD shows namespace mismatch', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        actualNamespace: 'victim-namespace',
        reason: 'namespace-mismatch',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('does not belong to the requested namespace');
    });

    it('allows artifact access when MLMD confirms namespace matches', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({ valid: true });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, artifactContent);
    });

    it('rejects when any artifact with same URI belongs to different namespace', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        actualNamespace: 'other-namespace',
        reason: 'namespace-mismatch',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('does not belong to the requested namespace');
    });

    it('rejects when artifact is not found in MLMD (no ownership evidence)', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        reason: 'artifact-not-found',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('does not belong to the requested namespace');
    });

    it('falls through (fail-open) when MLMD is unreachable', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: true,
        reason: 'mlmd-unavailable',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, artifactContent);
    });

    it('rejects http source artifact when namespace ownership mismatches', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        actualNamespace: 'victim-namespace',
        reason: 'namespace-mismatch',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/get?source=http&bucket=internal.example.com&key=victim%2Fsecret.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('does not belong to the requested namespace');

      expect(mockedValidateArtifactNamespace).toHaveBeenCalledWith(
        expect.any(String),
        'http://internal.example.com/victim/secret.txt',
        'my-namespace',
      );
    });

    it('passes correct URI to MLMD validation for s3 source', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({ valid: true });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200);

      expect(mockedValidateArtifactNamespace).toHaveBeenCalledWith(
        expect.any(String),
        's3://ml-pipeline/hello/world.txt',
        'my-namespace',
      );
    });

    it('passes correct URI to MLMD validation for gcs source', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({ valid: true });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      await request
        .get(
          '/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, `${artifactContent}\n`);

      expect(mockedValidateArtifactNamespace).toHaveBeenCalledWith(
        expect.any(String),
        'gs://ml-pipeline/hello/world.txt',
        'my-namespace',
      );
    });

    it('validates path-based download route against MLMD, not query params', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({ valid: true });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      await request
        .get('/artifacts/minio/ml-pipeline/hello/world.txt?namespace=my-namespace')
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, artifactContent);

      expect(mockedValidateArtifactNamespace).toHaveBeenCalledWith(
        expect.any(String),
        'minio://ml-pipeline/hello/world.txt',
        'my-namespace',
      );
    });

    it('rejects path-based route even if query params point to a valid artifact', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        actualNamespace: 'victim-namespace',
        reason: 'namespace-mismatch',
      });

      app = new UIServer(authEnabledConfigs());

      const request = requests(app.app);
      const response = await request
        .get(
          '/artifacts/minio/ml-pipeline/hello/world.txt?source=minio&bucket=safe-bucket&key=safe-key&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403);
      expect(response.text).toContain('does not belong to the requested namespace');

      expect(mockedValidateArtifactNamespace).toHaveBeenCalledWith(
        expect.any(String),
        'minio://ml-pipeline/hello/world.txt',
        'my-namespace',
      );
    });

    it('logs IDOR attempt with attacker and victim namespace details', async () => {
      mockAuthPass();
      mockedValidateArtifactNamespace.mockResolvedValue({
        valid: false,
        actualNamespace: 'secret-namespace',
        reason: 'namespace-mismatch',
      });

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      try {
        app = new UIServer(authEnabledConfigs());

        const request = requests(app.app);
        await request
          .get(
            '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=attacker-namespace',
          )
          .set('kubeflow-userid', 'attacker@example.com')
          .expect(403);

        const idorLog = consoleSpy.mock.calls.find(
          (call) => call[0] && call[0].includes('IDOR blocked'),
        );
        expect(idorLog).toBeDefined();
        if (idorLog) {
          expect(idorLog[0]).toContain('attacker@example.com');
          expect(idorLog[0]).toContain('attacker-namespace');
          expect(idorLog[0]).toContain('secret-namespace');
        }
      } finally {
        consoleSpy.mockRestore();
      }
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
