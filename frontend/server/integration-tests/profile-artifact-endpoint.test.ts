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
import { Client as MinioClient } from 'minio';
import { execFile } from 'node:child_process';
import { Readable } from 'node:stream';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';
import requests from 'supertest';
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs, type ProcessEnv, type UIConfigs } from '../configs.js';
import { getArtifactPreviewHandler } from '../handlers/artifact-preview.js';
import * as artifacts from '../handlers/artifacts.js';
import { getArtifactsHandler } from '../handlers/artifacts.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import { getConfigMap, getK8sSecret } from '../k8s-helper.js';
import { TEST_ONLY as launcherConfigTestOnly } from '../helpers/launcher-config.js';
import { setupLoopbackServers } from './test-helper.js';

vi.mock('minio');
vi.mock('../k8s-helper.js', () => ({
  getConfigMap: vi.fn(),
  getK8sSecret: vi.fn(),
}));

const origin = 'https://objects.example.com:9443';
const getObject = vi.fn();

async function profileEnvironments(
  allowedOrigins: string[],
  userIdHeader?: string,
): Promise<Record<string, ProcessEnv>> {
  const { stdout } = await promisify(execFile)(
    'python3',
    [
      fileURLToPath(new URL('./testdata/profile-artifact-environment.py', import.meta.url)),
      ...allowedOrigins,
    ],
    {
      encoding: 'utf8',
      timeout: 30_000,
      env: { ...process.env, USERID_HEADER: userIdHeader },
    },
  );
  const generated = JSON.parse(stdout) as Record<string, Array<{ name: string; value?: string }>>;
  return Object.fromEntries(
    Object.entries(generated).map(([allowed, entries]) => [
      allowed,
      // Kubernetes resolves Secret references separately; the provider uses its own Secret.
      Object.fromEntries(
        entries
          .filter((entry) => entry.value !== undefined)
          .map(({ name, value }) => [name, value]),
      ),
    ]),
  );
}

describe('profile-generated custom artifact endpoint configuration', () => {
  let environments: Record<string, ProcessEnv>;

  beforeAll(async () => {
    // Cold Python startup belongs to fixture setup, not each endpoint test's budget.
    environments = await profileEnvironments(['', origin]);
    expect(Object.keys(environments)).toEqual(['', origin]);
  }, 35_000);

  beforeEach(() => {
    vi.resetAllMocks();
    launcherConfigTestOnly.clearLauncherConfigurationCache();
    vi.spyOn(console, 'log').mockImplementation(() => undefined);
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    vi.mocked(getK8sSecret).mockImplementation(async (_name, key) => `tenant-${key}`);
    getObject.mockImplementation(async () => Readable.from(['custom store artifact']));
    vi.mocked(MinioClient.prototype.getObject).mockImplementation(getObject);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each([false, true])(
    'allows a profile-local custom store only after operator allowlisting (allowed=%s)',
    async (allowed) => {
      const env = environments[allowed ? origin : ''];
      expect(env.FRONTEND_SERVER_NAMESPACE).toBe('tenant');
      expect(env.ALLOWED_ARTIFACT_ENDPOINTS).toBe(allowed ? origin : '');
      const configs = loadConfigs(['node', 'server.js', '/tmp', '3000'], env);
      const app = express();
      app.get(
        '/artifacts/get',
        getArtifactsHandler({
          artifactsConfigs: configs.artifacts,
          useParameter: false,
          tryExtract: false,
          options: configs,
        }),
      );
      const response = await requests(app)
        .get('/artifacts/get')
        .query({
          source: 's3',
          bucket: 'tenant-artifacts',
          key: 'report.txt',
          namespace: 'tenant',
          providerInfo: JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: origin,
              disableSSL: 'false',
              fromEnv: 'false',
              secretName: 'tenant-store',
              accessKeyKey: 'access-key',
              secretKeyKey: 'secret-key',
            },
          }),
        });
      if (!allowed) {
        expect(response.status).toBe(400);
        expect(response.text).toContain('ALLOWED_ARTIFACT_ENDPOINTS');
        expect(getK8sSecret).not.toHaveBeenCalled();
        expect(MinioClient).not.toHaveBeenCalled();
        expect(getObject).not.toHaveBeenCalled();
        return;
      }
      expect(response.status).toBe(200);
      expect(response.text).toBe('custom store artifact');
      expect(getK8sSecret).toHaveBeenCalledWith('tenant-store', 'access-key', 'tenant');
      expect(getK8sSecret).toHaveBeenCalledWith('tenant-store', 'secret-key', 'tenant');
      expect(MinioClient).toHaveBeenCalledWith(
        expect.objectContaining({
          endPoint: 'objects.example.com',
          port: 9443,
          useSSL: true,
          accessKey: 'tenant-access-key',
          secretKey: 'tenant-secret-key',
        }),
      );
      expect(getObject).toHaveBeenCalledWith('tenant-artifacts', 'report.txt');
    },
  );
});

describe('profile-generated artifact preview identity', () => {
  const identityHeaders = ['kubeflow-userid', 'x-company-user', undefined];
  const listen = setupLoopbackServers();
  const artifact = {
    artifact_id: 'model-1',
    namespace: 'tenant',
    type: 'Model',
    uri: 'minio://mlpipeline/private-artifacts/tenant/model',
    metadata: { _kfp_preview: { version: 1, path: '_kfp_preview.json' } },
  };
  const preview = {
    version: 1,
    kind: 'model',
    framework: { name: 'pytorch', version: '2.14.1' },
    structure: { tensor_element_count: 6 },
  };
  let environments: Record<string, ProcessEnv>;

  beforeAll(async () => {
    const generated = await Promise.all(
      identityHeaders.map(
        async (header) => [header ?? '', (await profileEnvironments([''], header))['']] as const,
      ),
    );
    environments = Object.fromEntries(generated);
  }, 35_000);

  beforeEach(() => {
    vi.resetAllMocks();
    launcherConfigTestOnly.clearLauncherConfigurationCache();
    vi.spyOn(console, 'log').mockImplementation(() => undefined);
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    getObject.mockImplementation(async () => Readable.from([JSON.stringify(preview)]));
    vi.mocked(MinioClient.prototype.getObject).mockImplementation(getObject);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  /** Builds the artifact limiter from configuration, as the UI server does. */
  function configuredRateLimiter(options: UIConfigs): express.Handler {
    return artifacts.getArtifactsRateLimiter({
      authEnabled: options.auth.enabled,
      kubeflowUserIdHeader: options.auth.kubeflowUserIdHeader,
      disabled: options.artifacts.rateLimitDisabled,
    });
  }

  /**
   * Starts a shared UI server that proxies previews to a profile-generated namespace receiver.
   * Both servers resolve the artifact through a metadata API that accepts only the given readers.
   * Every hop listens on loopback, so hundreds of requests never use per-request wildcard servers.
   */
  async function startPreviewChain({
    configuredHeader,
    readers,
    receiverEnvironment = environments[configuredHeader ?? ''],
  }: {
    configuredHeader: string | undefined;
    readers: string[];
    receiverEnvironment?: ProcessEnv;
  }) {
    const sharedOptions = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
      ENABLE_AUTHZ: 'true',
      KUBEFLOW_USERID_HEADER: configuredHeader,
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    const identityHeader = sharedOptions.auth.kubeflowUserIdHeader;
    const metadataRequests: Array<{ path: string; identity: string | undefined }> = [];
    const api = express();
    api.use((req, res, next) => {
      const identity = req.get(identityHeader);
      metadataRequests.push({ path: req.path, identity });
      if (!identity || !readers.includes(identity)) {
        res.status(401).send('The metadata API requires the configured identity header.');
        return;
      }
      next();
    });
    api.get('/apis/v2/artifacts/model-1', (_req, res) => res.json(artifact));
    const apiAddress = await listen(api);
    const authorize = vi.fn<AuthorizeFn>().mockResolvedValue(undefined);
    const receiverOptions = loadConfigs(['node', 'server.js', '/tmp', '3000'], receiverEnvironment);
    const receiver = express();
    receiver.get(
      '/artifacts/:artifactId/preview',
      configuredRateLimiter(receiverOptions),
      getArtifactPreviewHandler(receiverOptions, apiAddress, authorize),
    );
    const receiverAddress = await listen(receiver);
    const target = vi.fn((namespace: string) => {
      expect(namespace).toBe('tenant');
      return receiverAddress;
    });
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(target);
    const shared = express();
    shared.get(
      '/artifacts/:artifactId/preview',
      configuredRateLimiter(sharedOptions),
      getArtifactPreviewHandler(sharedOptions, apiAddress, authorize),
    );
    const sharedAddress = await listen(shared);
    return {
      authorize,
      identityHeader,
      metadataRequests,
      receiverAddress,
      receiverOptions,
      sharedAddress,
      target,
    };
  }

  it.each(identityHeaders)(
    'preserves %s through the real namespace receiver and its authenticated metadata lookup',
    async (configuredHeader) => {
      const identity = 'reader@example.com';
      const {
        authorize,
        identityHeader,
        metadataRequests,
        receiverAddress,
        receiverOptions,
        sharedAddress,
        target,
      } = await startPreviewChain({ configuredHeader, readers: [identity] });
      expect(receiverOptions.auth.enabled).toBe(false);
      expect(receiverOptions.auth.kubeflowUserIdHeader).toBe(identityHeader);

      const response = await requests(sharedAddress)
        .get('/artifacts/model-1/preview?namespace=tenant')
        .set(identityHeader, identity)
        .expect(200);
      expect(response.body).toEqual(preview);
      expect(authorize).toHaveBeenCalledOnce();
      expect(target).toHaveBeenCalledExactlyOnceWith('tenant');
      expect(metadataRequests).toEqual([
        { path: '/apis/v2/artifacts/model-1', identity },
        { path: '/apis/v2/artifacts/model-1', identity },
      ]);
      expect(getObject).toHaveBeenCalledExactlyOnceWith(
        'mlpipeline',
        'private-artifacts/tenant/model/_kfp_preview.json',
      );

      await requests(sharedAddress).get('/artifacts/model-1/preview?namespace=tenant').expect(401);
      expect(metadataRequests).toHaveLength(2);
      await requests(receiverAddress)
        .get('/artifacts/model-1/preview?namespace=tenant')
        .expect(401);
      expect(metadataRequests.at(-1)?.identity).toBeUndefined();
      expect(getObject).toHaveBeenCalledOnce();
    },
  );

  it('enforces each user quota once at the shared server', async () => {
    const [firstReader, secondReader] = ['first@example.com', 'second@example.com'];
    const { identityHeader, receiverOptions, sharedAddress, target } = await startPreviewChain({
      configuredHeader: 'kubeflow-userid',
      readers: [firstReader, secondReader],
    });
    expect(receiverOptions.artifacts.rateLimitDisabled).toBe(true);
    const previewAs = (reader: string) =>
      requests(sharedAddress)
        .get('/artifacts/model-1/preview?namespace=tenant')
        .set(identityHeader, reader);

    for (let index = 0; index < 300; index++) {
      const response = await previewAs(firstReader);
      expect(response.status, `Preview request ${index}: ${response.text}`).toBe(200);
    }
    // The receiver sees every proxied user at the shared server's address.
    await previewAs(secondReader).expect(200);
    expect(target).toHaveBeenCalledTimes(301);

    const limited = await previewAs(firstReader).expect(429);
    expect(limited.headers['retry-after']).toMatch(/^\d+$/);
    expect(limited.text).toBe(artifacts.TOO_MANY_ARTIFACT_REQUESTS_MESSAGE);
    expect(target).toHaveBeenCalledTimes(301);
  }, 30_000);

  it('preserves the rate limit and retry delay from a receiver that enforces its own quota', async () => {
    const reader = 'reader@example.com';
    const { identityHeader, receiverAddress, sharedAddress } = await startPreviewChain({
      configuredHeader: 'kubeflow-userid',
      readers: [reader],
      receiverEnvironment: {
        ...environments['kubeflow-userid'],
        ARTIFACTS_RATE_LIMIT_DISABLED: 'false',
      },
    });
    // Exhaust the receiver's loopback-address budget, which the shared server's proxied requests
    // also use. Invalid IDs are rejected before any metadata lookup.
    for (let index = 0; index < 300; index++) {
      await requests(receiverAddress).get('/artifacts/invalid.id/preview').expect(400);
    }

    const response = await requests(sharedAddress)
      .get('/artifacts/model-1/preview?namespace=tenant')
      .set(identityHeader, reader)
      .expect(429);
    expect(Number(response.headers['retry-after'])).toBeGreaterThan(0);
    expect(response.headers['ratelimit']).toContain('r=299');
    expect(response.headers['cache-control']).toBe('no-store');
    expect(response.text).toBe(artifacts.TOO_MANY_ARTIFACT_REQUESTS_MESSAGE);
    expect(getObject).not.toHaveBeenCalled();
  });
});
