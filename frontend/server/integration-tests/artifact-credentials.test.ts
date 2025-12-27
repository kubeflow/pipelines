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

import { getNamespaceCredentials, createMinioClient } from '../minio-helper';
import { getK8sSecret } from '../k8s-helper';

jest.mock('../k8s-helper');

const mockedGetK8sSecret = getK8sSecret as jest.MockedFunction<typeof getK8sSecret>;

describe('Credential Isolation (Issue #12373)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getNamespaceCredentials', () => {
    it('returns credentials when namespace secret exists', async () => {
      mockedGetK8sSecret
        .mockResolvedValueOnce('namespace-access-key')
        .mockResolvedValueOnce('namespace-secret-key');

      const credentials = await getNamespaceCredentials('user-namespace');

      expect(credentials).toEqual({
        accessKey: 'namespace-access-key',
        secretKey: 'namespace-secret-key',
      });
      expect(mockedGetK8sSecret).toHaveBeenCalledTimes(2);
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        1,
        'mlpipeline-minio-artifact',
        'accesskey',
        'user-namespace',
      );
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        2,
        'mlpipeline-minio-artifact',
        'secretkey',
        'user-namespace',
      );
    });

    it('returns undefined when secret does not exist', async () => {
      mockedGetK8sSecret.mockRejectedValue(new Error('Secret not found'));

      const credentials = await getNamespaceCredentials('user-namespace');

      expect(credentials).toBeUndefined();
    });

    it('returns undefined when credentials are incomplete', async () => {
      mockedGetK8sSecret.mockResolvedValueOnce('namespace-access-key').mockResolvedValueOnce('');

      const credentials = await getNamespaceCredentials('user-namespace');

      expect(credentials).toBeUndefined();
    });

    it('uses custom secret name when provided', async () => {
      mockedGetK8sSecret
        .mockResolvedValueOnce('custom-access-key')
        .mockResolvedValueOnce('custom-secret-key');

      const credentials = await getNamespaceCredentials('user-namespace', 'custom-secret-name');

      expect(credentials).toEqual({
        accessKey: 'custom-access-key',
        secretKey: 'custom-secret-key',
      });
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        1,
        'custom-secret-name',
        'accesskey',
        'user-namespace',
      );
    });
  });

  describe('createMinioClient with namespace credentials', () => {
    const defaultConfig = {
      endPoint: 'minio-service.kubeflow',
      port: 9000,
      useSSL: false,
      accessKey: 'default-access-key',
      secretKey: 'default-secret-key',
    };

    it('uses namespace credentials when useNamespaceCredentials is true and namespace provided', async () => {
      mockedGetK8sSecret
        .mockResolvedValueOnce('namespace-access-key')
        .mockResolvedValueOnce('namespace-secret-key');

      const client = await createMinioClient(
        defaultConfig,
        'minio',
        undefined,
        'user-namespace',
        true,
      );

      expect(client).toBeDefined();
      expect(mockedGetK8sSecret).toHaveBeenCalledTimes(2);
    });

    it('uses default credentials when useNamespaceCredentials is false', async () => {
      const client = await createMinioClient(
        defaultConfig,
        'minio',
        undefined,
        'user-namespace',
        false,
      );

      expect(client).toBeDefined();
      expect(mockedGetK8sSecret).not.toHaveBeenCalled();
    });

    it('uses default credentials when namespace is not provided', async () => {
      const client = await createMinioClient(defaultConfig, 'minio', undefined, undefined, true);

      expect(client).toBeDefined();
      expect(mockedGetK8sSecret).not.toHaveBeenCalled();
    });

    it('falls back to default credentials when namespace secret not found', async () => {
      mockedGetK8sSecret.mockRejectedValue(new Error('Secret not found'));

      const client = await createMinioClient(
        defaultConfig,
        'minio',
        undefined,
        'user-namespace',
        true,
      );

      expect(client).toBeDefined();
      expect(mockedGetK8sSecret).toHaveBeenCalled();
    });

    it('skips namespace credential lookup when providerInfo is provided', async () => {
      const providerInfo = JSON.stringify({
        Provider: 'minio',
        Params: {
          fromEnv: 'true',
        },
      });

      const client = await createMinioClient(
        defaultConfig,
        'minio',
        providerInfo,
        'user-namespace',
        true,
      );

      expect(client).toBeDefined();
      expect(mockedGetK8sSecret).not.toHaveBeenCalled();
    });
  });

  describe('multi-user isolation scenarios', () => {
    it('different namespaces get different credentials', async () => {
      mockedGetK8sSecret
        .mockResolvedValueOnce('user-a-access-key')
        .mockResolvedValueOnce('user-a-secret-key');

      const credentialsA = await getNamespaceCredentials('namespace-a');
      expect(credentialsA).toEqual({
        accessKey: 'user-a-access-key',
        secretKey: 'user-a-secret-key',
      });
      mockedGetK8sSecret
        .mockResolvedValueOnce('user-b-access-key')
        .mockResolvedValueOnce('user-b-secret-key');

      const credentialsB = await getNamespaceCredentials('namespace-b');
      expect(credentialsB).toEqual({
        accessKey: 'user-b-access-key',
        secretKey: 'user-b-secret-key',
      });
      expect(credentialsA?.accessKey).not.toEqual(credentialsB?.accessKey);
      expect(credentialsA?.secretKey).not.toEqual(credentialsB?.secretKey);
    });

    it('credentials are fetched per-request, not cached globally', async () => {
      mockedGetK8sSecret
        .mockResolvedValueOnce('first-access-key')
        .mockResolvedValueOnce('first-secret-key');

      await getNamespaceCredentials('test-namespace');

      mockedGetK8sSecret
        .mockResolvedValueOnce('second-access-key')
        .mockResolvedValueOnce('second-secret-key');

      await getNamespaceCredentials('test-namespace');

      expect(mockedGetK8sSecret).toHaveBeenCalledTimes(4);
    });
  });
});
