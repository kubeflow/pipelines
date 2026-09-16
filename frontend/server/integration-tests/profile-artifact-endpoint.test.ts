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
import { execFileSync } from 'node:child_process';
import { Readable } from 'node:stream';
import { fileURLToPath } from 'node:url';
import requests from 'supertest';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs, type ProcessEnv } from '../configs.js';
import { getArtifactsHandler } from '../handlers/artifacts.js';
import { getConfigMap, getK8sSecret } from '../k8s-helper.js';
import { TEST_ONLY as launcherConfigTestOnly } from '../helpers/launcher-config.js';

vi.mock('minio');
vi.mock('../k8s-helper.js', () => ({
  getConfigMap: vi.fn(),
  getK8sSecret: vi.fn(),
}));

const origin = 'https://objects.example.com:9443';
const getObject = vi.fn();

function profileEnvironment(allowedOrigins: string): ProcessEnv {
  const generated = JSON.parse(
    execFileSync(
      'python3',
      [
        fileURLToPath(new URL('./testdata/profile-artifact-environment.py', import.meta.url)),
        allowedOrigins,
      ],
      { encoding: 'utf8', timeout: 5000 },
    ),
  ) as Array<{ name: string; value?: string }>;
  // Kubernetes resolves Secret references separately; the provider below uses its own Secret.
  return Object.fromEntries(
    generated.filter((entry) => entry.value !== undefined).map(({ name, value }) => [name, value]),
  );
}

describe('profile-generated custom artifact endpoint configuration', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    launcherConfigTestOnly.clearLauncherConfigurationCache();
    vi.spyOn(console, 'log').mockImplementation(() => undefined);
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    vi.mocked(getK8sSecret).mockImplementation(async (_name, key) => `tenant-${key}`);
    getObject.mockImplementation(async () => Readable.from(['custom store artifact']));
    vi.mocked(MinioClient).mockImplementation(function () {
      return { getObject, listObjectsV2Query: vi.fn() } as unknown as MinioClient;
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each([false, true])(
    'allows a profile-local custom store only after operator allowlisting (allowed=%s)',
    async (allowed) => {
      const env = profileEnvironment(allowed ? origin : '');
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
