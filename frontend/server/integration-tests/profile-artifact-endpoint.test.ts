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
import { once } from 'node:events';
import type { Server } from 'node:http';
import { Readable } from 'node:stream';
import { fileURLToPath } from 'node:url';
import { promisify } from 'node:util';
import requests from 'supertest';
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs, type ProcessEnv } from '../configs.js';
import { getArtifactPreviewHandler } from '../handlers/artifact-preview.js';
import * as artifacts from '../handlers/artifacts.js';
import { getArtifactsHandler } from '../handlers/artifacts.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import { getConfigMap, getK8sSecret } from '../k8s-helper.js';
import { TEST_ONLY as launcherConfigTestOnly } from '../helpers/launcher-config.js';

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
  const servers: Server[] = [];
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
    vi.mocked(MinioClient.prototype.getObject).mockImplementation(getObject);
  });

  afterEach(async () => {
    await Promise.all(
      servers.splice(0).map(
        (server) =>
          new Promise<void>((resolve, reject) => {
            server.close((error) => (error ? reject(error) : resolve()));
          }),
      ),
    );
    vi.restoreAllMocks();
  });

  async function listen(app: express.Application): Promise<string> {
    const server = app.listen(0, '127.0.0.1');
    servers.push(server);
    await once(server, 'listening');
    const address = server.address();
    if (!address || typeof address === 'string') {
      throw new Error('Expected the test server to listen on a TCP port.');
    }
    return `http://127.0.0.1:${address.port}`;
  }

  it.each(identityHeaders)(
    'preserves %s through the real namespace receiver and its authenticated metadata lookup',
    async (configuredHeader) => {
      const identity = 'reader@example.com';
      const sharedOptions = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
        ENABLE_AUTHZ: 'true',
        KUBEFLOW_USERID_HEADER: configuredHeader,
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
      });
      const identityHeader = sharedOptions.auth.kubeflowUserIdHeader;
      const artifact = {
        artifact_id: 'model-1',
        namespace: 'tenant',
        type: 'Model',
        uri: 'minio://mlpipeline/private-artifacts/tenant/model',
        metadata: { _kfp_preview: { version: 1, path: 'preview.json' } },
      };
      const preview = {
        version: 1,
        kind: 'model',
        framework: { name: 'pytorch', version: '2.14.1' },
        structure: { tensor_element_count: 6 },
      };
      getObject.mockImplementation(async () => Readable.from([JSON.stringify(preview)]));
      const received: Array<{ path: string; identity: string | undefined }> = [];
      const api = express();
      api.use(artifacts.getArtifactsRateLimiter(true, identityHeader));
      api.use((req, res, next) => {
        received.push({ path: req.path, identity: req.get(identityHeader) });
        if (req.get(identityHeader) !== identity) {
          res.status(401).send('The metadata API requires the configured identity header.');
          return;
        }
        next();
      });
      api.get('/apis/v2beta1/artifacts/model-1', (_req, res) => res.json(artifact));
      const apiAddress = await listen(api);
      const authorize = vi.fn<AuthorizeFn>().mockResolvedValue(undefined);
      const receiverOptions = loadConfigs(
        ['node', 'server.js', '/tmp', '3000'],
        environments[configuredHeader ?? ''],
      );
      expect(receiverOptions.auth.enabled).toBe(false);
      expect(receiverOptions.auth.kubeflowUserIdHeader).toBe(identityHeader);
      const receiver = express();
      receiver.get(
        '/artifacts/:artifactId/preview',
        artifacts.getArtifactsRateLimiter(false, receiverOptions.auth.kubeflowUserIdHeader),
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
        artifacts.getArtifactsRateLimiter(true, identityHeader),
        getArtifactPreviewHandler(sharedOptions, apiAddress, authorize),
      );

      const response = await requests(shared)
        .get('/artifacts/model-1/preview?namespace=tenant')
        .set(identityHeader, identity)
        .expect(200);
      expect(response.body).toEqual(preview);
      expect(authorize).toHaveBeenCalledOnce();
      expect(target).toHaveBeenCalledExactlyOnceWith('tenant');
      expect(received).toEqual([
        { path: '/apis/v2beta1/artifacts/model-1', identity },
        { path: '/apis/v2beta1/artifacts/model-1', identity },
      ]);
      expect(getObject).toHaveBeenCalledExactlyOnceWith(
        'mlpipeline',
        'private-artifacts/tenant/model/preview.json',
      );

      await requests(shared).get('/artifacts/model-1/preview?namespace=tenant').expect(401);
      expect(received).toHaveLength(2);
      await requests(receiver).get('/artifacts/model-1/preview?namespace=tenant').expect(401);
      expect(received.at(-1)?.identity).toBeUndefined();
      expect(getObject).toHaveBeenCalledOnce();
    },
  );
});
