// Copyright 2019-2020 The Kubeflow Authors
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
import { vi, describe, it, expect, beforeEach, Mock } from 'vitest';
import * as zlib from 'zlib';
import { PassThrough } from 'stream';
import { Client as MinioClient } from 'minio';
import {
  createMinioClient,
  isTarball,
  maybeTarball,
  getObjectStream,
  isNoSuchKeyError,
  listObjectsUnderPrefix,
  summarizeDirectoryUnderPrefix,
  MinioClientOptionsWithOptionalSecrets,
  Credentials,
  TEST_ONLY,
} from './minio-helper.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
import { getK8sSecret } from './k8s-helper.js';

vi.mock('minio');
vi.mock('@aws-sdk/credential-providers');
vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

describe('minio-helper', () => {
  const MockedMinioClient: Mock = MinioClient as any;
  const MockedAuthorizeFn: Mock = vi.fn((x) => undefined);
  const minioClientDouble = (overrides: Record<string, unknown>) => ({
    getObject: vi.fn(),
    listObjectsV2Query: vi.fn(),
    retryOptions: {},
    ...overrides,
  });

  beforeEach(() => {
    vi.resetAllMocks();
  });

  describe('createMinioClient', () => {
    it.each([
      ['', 'secret'],
      ['access', ''],
    ])(
      'fails closed when an explicit provider Secret contains credentials %j / %j',
      async (accessKey, secretKey) => {
        vi.mocked(getK8sSecret).mockResolvedValueOnce(accessKey).mockResolvedValueOnce(secretKey);

        await expect(
          createMinioClient(
            { endPoint: 'central.example' },
            's3',
            JSON.stringify({
              Provider: 's3',
              Params: {
                accessKeyKey: 'access-key',
                fromEnv: 'false',
                secretKeyKey: 'secret-key',
                secretName: 'artifact-store',
              },
            }),
            'team-a',
          ),
        ).rejects.toThrow('Provider Secret contains an empty access key or secret key');

        expect(fromNodeProviderChain).not.toHaveBeenCalled();
        expect(MockedMinioClient).not.toHaveBeenCalled();
      },
    );

    it('creates a minio client with the provided configs.', async () => {
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'minio.kubeflow:80',
          secretKey: 'secretkey',
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'minio.kubeflow:80',
        secretKey: 'secretkey',
      });
    });

    it('Builds a client where credentials are resolved using a custom provider.', async () => {
      const provider = async (): Promise<Credentials> => {
        return {
          accessKeyId: 'providedKey',
          secretAccessKey: 'providedSecret',
          sessionToken: 'providedToken',
        };
      };

      const client = await createMinioClient(
        {
          endPoint: 'minio.kubeflow:80',
        },
        's3',
        '',
        '',
        provider,
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'providedKey',
        endPoint: 'minio.kubeflow:80',
        secretKey: 'providedSecret',
        sessionToken: 'providedToken',
      });
    });

    it('fails closed if authenticated S3 credentials are unavailable.', async () => {
      await expect(createMinioClient({ endPoint: 'minio.kubeflow:80' }, 's3')).rejects.toThrow(
        'Unable to resolve AWS credentials',
      );

      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it('applies endpoint and region settings when credentials come from the environment', async () => {
      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: {
            disableSSL: 'false',
            endpoint: 'https://ceph.example:9443',
            fromEnv: 'true',
            region: 'ceph',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'ceph.example',
        port: 9443,
        region: 'ceph',
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('uses standard AWS HTTPS for an explicitly empty structured S3 endpoint', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'central.example',
          secretKey: 'secretkey',
          useSSL: false,
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: { disableSSL: 'false', endpoint: '', fromEnv: 'true' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.amazonaws.com',
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('applies structured provider retry settings', async () => {
      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '5' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        secretKey: 'secretkey',
      });
      expect((client as any).retryOptions.maximumRetryCount).toBe(0);
    });

    it('uses an operation retry budget when structured maxRetries is zero', async () => {
      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '0' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        secretKey: 'secretkey',
      });
      expect((client as any).retryOptions.maximumRetryCount).toBe(0);
    });

    it('uses the Go default retry budget when maxRetries is omitted', async () => {
      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({ Provider: 'minio', Params: { fromEnv: 'true' } }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        secretKey: 'secretkey',
      });
      expect((client as any).retryOptions.maximumRetryCount).toBe(0);
    });

    it('accepts retry budgets above the frontend safety limit', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '20' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledTimes(1);
    });

    it('rejects malformed native endpoint authorities instead of repairing them', async () => {
      await expect(
        createMinioClient(
          { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'http:///evil.example/base',
              fromEnv: 'true',
              nativeQuery: 'true',
            },
          }),
        ),
      ).rejects.toThrow('must contain a valid authority');

      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it('rejects backslashes in native endpoint authorities', async () => {
      await expect(
        createMinioClient(
          { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'http://evil.example\\@trusted.example/base',
              fromEnv: 'true',
              nativeQuery: 'true',
            },
          }),
        ),
      ).rejects.toThrow('must contain a valid authority');

      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it('uses the secure default when provider info does not specify disableSSL', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'default-store',
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: {
            endpoint: 'https://ceph.example:9443',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'ceph.example',
        port: 9443,
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('preserves an explicit HTTP endpoint scheme when disableSSL is false', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'default-store',
          secretKey: 'secretkey',
          useSSL: true,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: {
            disableSSL: 'false',
            endpoint: 'http://ceph.example:9000',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'ceph.example',
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('retains the server TLS setting when provider info does not override the endpoint', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'minio-service.kubeflow',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { disableSSL: 'false', fromEnv: 'true' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'minio-service.kubeflow',
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('applies endpoint-less disableSSL=true while inheriting the server endpoint and region', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'minio-service.kubeflow',
          region: 'environment-region',
          secretKey: 'secretkey',
          useSSL: true,
        },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { disableSSL: 'true', fromEnv: 'true', region: '' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'minio-service.kubeflow',
        region: 'environment-region',
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('supports anonymous virtual-hosted S3 access without resolving credentials', async () => {
      await createMinioClient(
        {
          accessKey: 'environment-access',
          endPoint: 'default-store',
          secretKey: 'environment-secret',
          sessionToken: 'environment-session-token',
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            anonymous: '1',
            endpoint: 'https://s3.us-west-2.amazonaws.com',
            forcePathStyle: 'false',
            fromEnv: 'true',
          },
        }),
      );

      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedMinioClient).toHaveBeenCalledWith({
        endPoint: 's3.us-west-2.amazonaws.com',
        pathStyle: false,
        port: undefined,
        useSSL: true,
      });
    });

    it('rejects unsupported S3 read options before resolving ambient credentials', async () => {
      await expect(
        createMinioClient(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'https://s3.us-west-2.amazonaws.com',
              fromEnv: 'true',
              role: 'arn:aws:iam::123456789012:role/ArtifactReader',
            },
          }),
        ),
      ).rejects.toThrow('Unsupported S3 artifact read option: role');

      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it('accepts valid neutral S3 options that do not change read behavior', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            accelerate: '0',
            dualstack: 'false',
            endpoint: 'https://s3.us-west-2.amazonaws.com',
            fips: 'False',
            fromEnv: 'true',
            hostname_immutable: 'FALSE',
            profile: '',
            rate_limiter_capacity: '0',
            request_checksum_calculation: 'when_supported',
            response_checksum_validation: 'WHEN_SUPPORTED',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.us-west-2.amazonaws.com',
        port: undefined,
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it.each([
      ['profile', 'team-profile'],
      ['request_checksum_calculation', 'when_required'],
      ['response_checksum_validation', 'when_required'],
    ])('rejects unsupported non-neutral S3 option %s', async (option, value) => {
      await expect(
        createMinioClient(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'https://s3.us-west-2.amazonaws.com',
              fromEnv: 'true',
              [option]: value,
            },
          }),
        ),
      ).rejects.toThrow(`Unsupported S3 artifact read option: ${option}`);
    });

    it.each(['', 'invalid'])(
      'rejects invalid request checksum calculation value %j',
      async (value) => {
        await expect(
          createMinioClient(
            { endPoint: 's3.amazonaws.com' },
            's3',
            JSON.stringify({
              Provider: 's3',
              Params: {
                fromEnv: 'true',
                request_checksum_calculation: value,
              },
            }),
          ),
        ).rejects.toThrow('Invalid value for provider option request_checksum_calculation');
      },
    );

    it.each([
      ['ssetype', ''],
      ['ssetype', 'invalid'],
      ['kmskeyid', ''],
    ])('rejects invalid native S3 write option %s=%j', async (option, value) => {
      await expect(
        createMinioClient(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: { fromEnv: 'true', [option]: value },
          }),
        ),
      ).rejects.toThrow(option);
    });

    it('rejects unsupported S3 behavior when its native boolean is enabled', async () => {
      await expect(
        createMinioClient(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              accelerate: '1',
              endpoint: 'https://s3.us-west-2.amazonaws.com',
              fromEnv: 'true',
            },
          }),
        ),
      ).rejects.toThrow('Unsupported S3 artifact read option: accelerate');

      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it('parses an explicit HTTP AWS endpoint into Minio client options', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'http://s3.us-west-2.amazonaws.com',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.us-west-2.amazonaws.com',
        port: undefined,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('applies disableSSL to a scheme-less AWS endpoint', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            disableSSL: 'true',
            endpoint: 's3.us-west-2.amazonaws.com',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.us-west-2.amazonaws.com',
        port: undefined,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('applies runtime S3 query aliases', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            disable_https: 'true',
            endpoint: 'ceph.example:9000',
            fromEnv: 'true',
            use_path_style: 'false',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'ceph.example',
        pathStyle: false,
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('applies endpoint-less disable_https from a native S3 query', async () => {
      await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 's3.amazonaws.com',
          secretKey: 'secretkey',
          useSSL: true,
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: { disable_https: 'true', fromEnv: 'true', nativeQuery: 'true' },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.amazonaws.com',
        pathStyle: false,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('defaults native custom S3 endpoints to virtual-host addressing', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://store.example/base',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        pathStyle: false,
        port: undefined,
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('parses uppercase native HTTPS endpoint schemes', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'HTTPS://store.example:9443/base',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        pathStyle: false,
        port: 9443,
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('rejects scheme-less endpoints from native S3 queries', async () => {
      await expect(
        createMinioClient(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'store.example:9000',
              fromEnv: 'true',
              nativeQuery: 'true',
            },
          }),
        ),
      ).rejects.toThrow('absolute HTTP(S) URL');

      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedMinioClient).not.toHaveBeenCalled();
    });

    it.each([
      ['disableSSL', 'true', 'https://store.example:9000', false],
      ['disableSSL', 'false', 'http://store.example:9000', false],
      ['disable_https', 'true', 'https://store.example:9000', false],
      ['disable_https', 'false', 'http://store.example:9000', false],
    ])(
      'applies asymmetric TLS precedence for %s=%s and an explicit endpoint',
      async (option, value, endpoint, expectedUseSSL) => {
        await createMinioClient(
          { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              [option]: value,
              endpoint,
              fromEnv: 'true',
            },
          }),
        );

        expect(MockedMinioClient).toHaveBeenCalledWith({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: expectedUseSSL,
        });
      },
    );

    it('uses the AWS default credential chain with a custom S3 endpoint', async () => {
      (fromNodeProviderChain as Mock).mockReturnValueOnce(async () => ({
        accessKeyId: 'irsa-access-key',
        secretAccessKey: 'irsa-secret-key',
        sessionToken: 'irsa-session-token',
      }));

      await createMinioClient(
        { endPoint: 's3.amazonaws.com' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://ceph.example:9443/base',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'irsa-access-key',
        endPoint: 'ceph.example',
        pathStyle: false,
        port: 9443,
        secretKey: 'irsa-secret-key',
        sessionToken: 'irsa-session-token',
        useSSL: true,
      });
    });

    it('passes bare IPv6 addresses from native endpoints to MinIO', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'http://[2001:db8::1]:9000/base',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: '2001:db8::1',
        pathStyle: false,
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('preserves endpoint base paths in the path MinIO signs and requests', async () => {
      const getRequestOptions = vi.fn(() => ({ path: '/bucket/object?versionId=1' }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://store.example/base/path/',
            fromEnv: 'true',
          },
        }),
      );

      const requestOptions = (client as any).getRequestOptions({ method: 'GET' });
      expect(requestOptions.path).toBe('/base/path/bucket/object?versionId=1');
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'store.example',
        port: undefined,
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it.each([
      ['https://store.example/root dir/café', '/root%20dir/caf%C3%A9/bucket/object'],
      ['https://store.example/base/%2e%2e/inner', '/base/../inner/bucket/object'],
      ['https://store.example/base/%3F/discarded', '/base/bucket/object'],
      ['https://store.example/base/%23/discarded', '/base/bucket/object'],
    ])('preserves the Go request spelling for endpoint path %s', async (endpoint, expectedPath) => {
      const getRequestOptions = vi.fn(() => ({ path: '/bucket/object' }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({ Provider: 's3', Params: { endpoint, fromEnv: 'true' } }),
      );

      expect((client as any).getRequestOptions({ method: 'GET' }).path).toBe(expectedPath);
    });

    it('preserves an explicitly selected global AWS endpoint authority', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.us-west-2.amazonaws.com' },
        host: 'bucket.s3.us-west-2.amazonaws.com',
        path: '/object',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.amazonaws.com/base',
            fromEnv: 'true',
            nativeQuery: 'true',
            region: 'us-west-2',
          },
        }),
      );

      expect((client as any).getRequestOptions({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 'bucket.s3.amazonaws.com' },
          host: 'bucket.s3.amazonaws.com',
          path: '/base/object',
        }),
      );
    });

    it('preserves MinIO path-style addressing for dotted HTTPS buckets', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 's3.amazonaws.com' },
        host: 's3.amazonaws.com',
        path: '/bucket.with.dot/object',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.amazonaws.com',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(
        (client as any).getRequestOptions({ bucketName: 'bucket.with.dot', method: 'GET' }),
      ).toEqual(
        expect.objectContaining({
          headers: { host: 's3.amazonaws.com' },
          host: 's3.amazonaws.com',
          path: '/bucket.with.dot/object',
        }),
      );
    });

    it.each([
      ['https://s3.amazonaws.com:80', 80],
      ['http://s3.amazonaws.com:443', 443],
    ])('preserves cross-scheme port in the signed Host for %s', async (endpoint, port) => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.amazonaws.com' },
        host: 'bucket.s3.amazonaws.com',
        path: '/object',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: { endpoint, fromEnv: 'true', nativeQuery: 'true' },
        }),
      );

      expect((client as any).getRequestOptions({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({ headers: { host: `bucket.s3.amazonaws.com:${port}` } }),
      );
    });

    it('preserves an explicit China partition S3 authority', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.cn-northwest-1.amazonaws.com.cn' },
        host: 'bucket.s3.cn-northwest-1.amazonaws.com.cn',
        path: '/object',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.cn-north-1.amazonaws.com.cn',
            fromEnv: 'true',
            nativeQuery: 'true',
            region: 'cn-northwest-1',
          },
        }),
      );

      expect((client as any).getRequestOptions({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 'bucket.s3.cn-north-1.amazonaws.com.cn' },
          host: 'bucket.s3.cn-north-1.amazonaws.com.cn',
        }),
      );
    });

    it('preserves a structured China partition S3 authority', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.us-east-1.amazonaws.com' },
        host: 'bucket.s3.us-east-1.amazonaws.com',
        path: '/object',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.cn-north-1.amazonaws.com.cn/base',
            fromEnv: 'true',
            region: 'cn-northwest-1',
          },
        }),
      );

      expect((client as any).getRequestOptions({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 'bucket.s3.cn-north-1.amazonaws.com.cn' },
          host: 'bucket.s3.cn-north-1.amazonaws.com.cn',
          path: '/base/object',
        }),
      );
    });

    it.each([
      ['native', 'true'],
      ['structured', undefined],
    ])('does not prepend a path-style bucket named s3 for %s endpoints', async (_, nativeQuery) => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 's3.s3.us-west-2.amazonaws.com' },
        host: 's3.s3.us-west-2.amazonaws.com',
        path: '/s3/hello.txt',
      }));
      MockedMinioClient.mockImplementationOnce(function () {
        return minioClientDouble({ getRequestOptions });
      });

      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.us-west-2.amazonaws.com/base',
            forcePathStyle: 'true',
            fromEnv: 'true',
            ...(nativeQuery ? { nativeQuery } : {}),
          },
        }),
      );

      expect((client as any).getRequestOptions({ bucketName: 's3', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 's3.us-west-2.amazonaws.com' },
          host: 's3.us-west-2.amazonaws.com',
          path: '/base/s3/hello.txt',
        }),
      );
    });

    it('normalizes structured global AWS endpoints through the standard resolver', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.amazonaws.com:9443/base/path',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 's3.amazonaws.com',
        secretKey: 'secretkey',
        useSSL: true,
      });
    });

    it('preserves escaped dot segments and encodes path backslashes without repairing them', () => {
      expect(TEST_ONLY.parseProviderEndpoint('https://store/base/%2e%2e/inner', true)).toEqual(
        expect.objectContaining({ basePath: '/base/../inner', host: 'store' }),
      );
      expect(TEST_ONLY.parseProviderEndpoint('https://store/base\\name/object', true)).toEqual(
        expect.objectContaining({ basePath: '/base%5Cname/object', host: 'store' }),
      );
      expect(TEST_ONLY.parseProviderEndpoint('https://store/root dir/café', true)).toEqual(
        expect.objectContaining({ basePath: '/root%20dir/caf%C3%A9', host: 'store' }),
      );
      expect(
        TEST_ONLY.parseProviderEndpoint('https://store/%41/%7e/%2f/%3A/raw[bracket]', true),
      ).toEqual(expect.objectContaining({ basePath: '/A/~///:/raw[bracket]', host: 'store' }));
      expect(TEST_ONLY.parseProviderEndpoint('https://store/%252e/%252f', true)).toEqual(
        expect.objectContaining({ basePath: '/%2e/%2f', host: 'store' }),
      );
      expect(TEST_ONLY.parseProviderEndpoint('https://store/%FF/%C0%AF/%ED%A0%80', true)).toEqual(
        expect.objectContaining({ basePath: '/%FF/%C0%AF/%ED%A0%80', host: 'store' }),
      );
    });

    it('does not normalize a hostname that merely starts with the standard AWS name', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.amazonaws.com.tenant.example/base',
            fromEnv: 'true',
          },
        }),
      );

      expect(MockedMinioClient).toHaveBeenCalledWith(
        expect.objectContaining({ endPoint: 's3.amazonaws.com.tenant.example' }),
      );
    });

    it('rejects control characters in provider endpoints instead of repairing them', () => {
      expect(() =>
        TEST_ONLY.parseProviderEndpoint('https://store.example\t.evil/base', true),
      ).toThrow('invalid control character');
      expect(() => TEST_ONLY.parseProviderEndpoint('https://store.example\n/base', true)).toThrow(
        'invalid control character',
      );
    });

    it('rejects malformed endpoint path escapes like Go net/url', () => {
      expect(() =>
        TEST_ONLY.parseProviderEndpoint('https://store.example/base/%zz/object', true),
      ).toThrow('invalid URL escape');
    });

    it('does not mutate shared defaults when applying per-request provider settings', async () => {
      const sharedConfig = {
        accessKey: 'default-access-key',
        endPoint: 'default-store',
        port: 9000,
        secretKey: 'default-secret-key',
        useSSL: false,
      };

      await createMinioClient(
        sharedConfig,
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: {
            disableSSL: 'false',
            endpoint: 'https://tenant-store.example:9443',
            fromEnv: 'true',
          },
        }),
      );
      await createMinioClient(sharedConfig, 'minio');

      expect(sharedConfig).toEqual({
        accessKey: 'default-access-key',
        endPoint: 'default-store',
        port: 9000,
        secretKey: 'default-secret-key',
        useSSL: false,
      });
      expect(MockedMinioClient).toHaveBeenNthCalledWith(2, sharedConfig);
    });

    it('uses EC2 metadata credentials if access key are not provided.', async () => {
      (fromNodeProviderChain as Mock).mockImplementation(
        () => () =>
          Promise.resolve({
            accessKeyId: 'AccessKeyId',
            secretAccessKey: 'SecretAccessKey',
            sessionToken: 'SessionToken',
          }),
      );
      const client = await createMinioClient({ endPoint: 's3.amazonaws.com' }, 's3');
      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'AccessKeyId',
        endPoint: 's3.amazonaws.com',
        secretKey: 'SecretAccessKey',
        sessionToken: 'SessionToken',
      });
      expect(MockedMinioClient).toBeCalledTimes(1);
    });

    it('rewrites configured in-cluster endpoints to local proxy endpoints.', async () => {
      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'seaweedfs.kubeflow.svc',
          endpointRewrite:
            'seaweedfs.kubeflow:9000=localhost:9000,seaweedfs.kubeflow.svc:9000=localhost:9000',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'localhost',
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });

    it('skips invalid endpoint rewrite rules.', async () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);

      const client = await createMinioClient(
        {
          accessKey: 'accesskey',
          endPoint: 'seaweedfs.kubeflow.svc',
          endpointRewrite:
            'seaweedfs.kubeflow.svc:9000=::::,seaweedfs.kubeflow.svc:9000=localhost:9000',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(consoleWarnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Ignoring invalid MinIO endpoint rewrite endpoint "::::"'),
      );
      expect(MockedMinioClient).toHaveBeenCalledWith({
        accessKey: 'accesskey',
        endPoint: 'localhost',
        port: 9000,
        secretKey: 'secretkey',
        useSSL: false,
      });
    });
  });

  describe('S3 operation retries', () => {
    it.each([
      [0, 3],
      [5, 5],
      [20, 10],
    ])('applies maxRetries=%i to configured client operations', async (maxRetries, attempts) => {
      vi.useFakeTimers();
      try {
        const getObject = vi
          .fn()
          .mockRejectedValue(Object.assign(new Error('connection reset'), { code: 'ECONNRESET' }));
        const listObjectsV2Query = vi.fn();
        MockedMinioClient.mockImplementationOnce(function () {
          return { getObject, listObjectsV2Query };
        });
        const client = await createMinioClient(
          { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
          'minio',
          JSON.stringify({
            Provider: 'minio',
            Params: { fromEnv: 'true', maxRetries: String(maxRetries) },
          }),
        );

        const request = expect(client.getObject('bucket', 'key')).rejects.toThrow(
          'connection reset',
        );
        await vi.runAllTimersAsync();
        await request;
        expect(getObject).toHaveBeenCalledTimes(attempts);
      } finally {
        vi.useRealTimers();
      }
    });

    it.each([
      'EADDRINUSE',
      'EADDRNOTAVAIL',
      'ECONNABORTED',
      'EHOSTDOWN',
      'EHOSTUNREACH',
      'ENETDOWN',
      'ENETUNREACH',
    ])('retries the Go connection error %s', async (code) => {
      const operation = vi.fn().mockRejectedValue(Object.assign(new Error(code), { code }));

      await expect(TEST_ONLY.retryS3Operation(operation, 3, async () => undefined)).rejects.toThrow(
        code,
      );
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('uses randomized exponential backoff with the Go twenty-second cap', () => {
      expect(TEST_ONLY.getS3RetryDelayMs(1, () => 0.25)).toBe(500);
      expect(TEST_ONLY.getS3RetryDelayMs(4, () => 0.25)).toBe(4_000);
      expect(TEST_ONLY.getS3RetryDelayMs(5, () => 0.25)).toBe(20_000);
      expect(TEST_ONLY.getS3RetryDelayMs(9, () => 0.75)).toBe(20_000);
    });

    it('fails loudly when a required retryable method is unavailable', async () => {
      MockedMinioClient.mockImplementationOnce(function () {
        return { getObject: vi.fn() };
      });

      await expect(
        createMinioClient(
          { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
          'minio',
          JSON.stringify({
            Provider: 'minio',
            Params: { fromEnv: 'true', maxRetries: '5' },
          }),
        ),
      ).rejects.toThrow('does not expose listObjectsV2Query');
    });

    it.each([
      [0, 3],
      [5, 5],
    ])('matches Go total-attempt semantics for maxRetries=%i', async (maxRetries, attempts) => {
      const operation = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('connection reset'), { code: 'ECONNRESET' }));

      await expect(
        TEST_ONLY.retryS3Operation(
          operation,
          maxRetries > 0 ? maxRetries : 3,
          async () => undefined,
        ),
      ).rejects.toThrow('connection reset');

      expect(operation).toHaveBeenCalledTimes(attempts);
    });

    it('does not retry non-retryable object-store failures', async () => {
      const operation = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('denied'), { code: 'AccessDenied' }));

      await expect(TEST_ONLY.retryS3Operation(operation, 5, async () => undefined)).rejects.toThrow(
        'denied',
      );
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('does not let a parsed S3 message impersonate the MinIO retry wrapper', () => {
      expect(
        TEST_ONLY.isRetryableS3Error({
          code: 'AccessDenied',
          message: 'Request failed after 0 retries: Error: Retryable HTTP status: 500',
        }),
      ).toBe(false);
      expect(TEST_ONLY.isRetryableS3Error({ code: 'UnknownError', statusCode: 500 })).toBe(true);
    });

    it('shares one failed-attempt budget across fallback operations', async () => {
      const context = TEST_ONLY.createS3RetryContext(3);
      const missingObject = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('missing'), { code: 'NoSuchKey' }));
      const failingSummary = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('reset'), { code: 'ECONNRESET' }));

      await expect(
        TEST_ONLY.retryS3Operation(missingObject, context, async () => undefined),
      ).rejects.toThrow('missing');
      await expect(
        TEST_ONLY.retryS3Operation(failingSummary, context, async () => undefined),
      ).rejects.toThrow('reset');

      expect(missingObject).toHaveBeenCalledTimes(1);
      expect(failingSummary).toHaveBeenCalledTimes(2);
    });

    it('shares the configured budget across the client object and fallback methods', async () => {
      vi.useFakeTimers();
      try {
        const getObject = vi
          .fn()
          .mockRejectedValue(Object.assign(new Error('missing'), { code: 'NoSuchKey' }));
        const listObjectsV2Query = vi
          .fn()
          .mockRejectedValue(Object.assign(new Error('reset'), { code: 'ECONNRESET' }));
        MockedMinioClient.mockImplementationOnce(function () {
          return { getObject, listObjectsV2Query };
        });
        const client = await createMinioClient(
          { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
          'minio',
          JSON.stringify({
            Provider: 'minio',
            Params: { fromEnv: 'true', maxRetries: '3' },
          }),
        );

        await expect(client.getObject('bucket', 'key')).rejects.toThrow('missing');
        const fallback = expect(
          (client as any).listObjectsV2Query('bucket', 'key/', '', 1),
        ).rejects.toThrow('reset');
        await vi.runAllTimersAsync();
        await fallback;

        expect(getObject).toHaveBeenCalledTimes(1);
        expect(listObjectsV2Query).toHaveBeenCalledTimes(2);
      } finally {
        vi.useRealTimers();
      }
    });

    it('stops retrying when the artifact HTTP request is aborted', async () => {
      const abortController = new AbortController();
      const context = TEST_ONLY.createS3RetryContext(3, abortController.signal);
      const operation = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('reset'), { code: 'ECONNRESET' }));

      await expect(
        TEST_ONLY.retryS3Operation(operation, context, async () => abortController.abort()),
      ).rejects.toMatchObject({ name: 'AbortError' });
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('passes request cancellation through a configured MinIO client', async () => {
      const abortController = new AbortController();
      const getObject = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('reset'), { code: 'ECONNRESET' }));
      MockedMinioClient.mockImplementationOnce(function () {
        return { getObject, listObjectsV2Query: vi.fn() };
      });
      const client = await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '3' },
        }),
        undefined,
        undefined,
        abortController.signal,
      );

      const request = expect(client.getObject('bucket', 'key')).rejects.toMatchObject({
        name: 'AbortError',
      });
      await vi.waitFor(() => expect(getObject).toHaveBeenCalledTimes(1));
      abortController.abort();
      await request;
    });
  });

  describe('isTarball', () => {
    it('checks magic number in buffer is a tarball.', () => {
      const tarGzBase64 =
        'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
      const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
      const tarBuffer = zlib.gunzipSync(tarGzBuffer);

      expect(isTarball(tarBuffer)).toBe(true);
    });

    it('checks magic number in buffer is not a tarball.', () => {
      expect(
        isTarball(
          Buffer.from(
            'some-random-string-more-random-string-even-more-random-string-even-even-more-random',
          ),
        ),
      ).toBe(false);
    });
  });

  describe('maybeTarball', () => {
    // hello world
    const tarGzBase64 =
      'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
    const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
    const tarBuffer = zlib.gunzipSync(tarGzBuffer);

    it('return the content for the 1st file inside a tarball', async () => {
      const stream = new PassThrough();
      const maybeTar = stream.pipe(maybeTarball());
      stream.end(tarBuffer);
      await new Promise<void>((resolve) => {
        stream.on('end', () => {
          expect(maybeTar.read().toString()).toBe('hello world\n');
          resolve();
        });
      });
    });

    it('return the content normal if is not a tarball', async () => {
      const stream = new PassThrough();
      const maybeTar = stream.pipe(maybeTarball());
      stream.end('hello world');
      await new Promise<void>((resolve) => {
        stream.on('end', () => {
          expect(maybeTar.read().toString()).toBe('hello world');
          resolve();
        });
      });
    });
  });

  describe('getObjectStream', () => {
    // hello world
    const tarGzBase64 =
      'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
    const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
    const tarBuffer = zlib.gunzipSync(tarGzBuffer);
    let minioClient: MinioClient;
    let mockedMinioGetObject: Mock;

    beforeEach(() => {
      vi.clearAllMocks();
      minioClient = new MinioClient({
        endPoint: 's3.amazonaws.com',
        accessKey: '',
        secretKey: '',
      });
      mockedMinioGetObject = minioClient.getObject as any;
    });

    it('unpacks a gzipped tarball', async () => {
      const objStream = new PassThrough();
      objStream.end(tarGzBuffer);
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(stream.read().toString().trim()).toBe('hello world');
      });
    });

    it('unpacks a uncompressed tarball', async () => {
      const objStream = new PassThrough();
      objStream.end(tarBuffer);
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(stream.read().toString().trim()).toBe('hello world');
      });
    });

    it('returns the content as a stream', async () => {
      const objStream = new PassThrough();
      objStream.end('hello world');
      mockedMinioGetObject.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ bucket: 'bucket', key: 'key', client: minioClient });
      expect(mockedMinioGetObject).toBeCalledWith('bucket', 'key');
      stream.on('finish', () => {
        expect(stream.read().toString().trim()).toBe('hello world');
      });
    });
  });

  // Different s3-compatible providers surface "object not found" through
  // different fields. minio uses lowercase `code`, the AWS SDK and some
  // proxies use uppercase `Code`, and a few wrap the SDK error in a generic
  // Error whose only signal is the message text. The helper has to recognize
  // all three so the directory-fallback download path can trigger
  // consistently.
  describe('isNoSuchKeyError', () => {
    it('matches lowercase "code: NoSuchKey" (minio convention)', () => {
      expect(isNoSuchKeyError({ code: 'NoSuchKey' })).toBe(true);
    });

    it('matches lowercase "code: NotFound"', () => {
      expect(isNoSuchKeyError({ code: 'NotFound' })).toBe(true);
    });

    it('matches uppercase "Code" (AWS SDK convention)', () => {
      expect(isNoSuchKeyError({ Code: 'NoSuchKey' })).toBe(true);
      expect(isNoSuchKeyError({ Code: 'NotFound' })).toBe(true);
    });

    it('falls back to message substring when no code field is present', () => {
      expect(isNoSuchKeyError(new Error('NoSuchKey: object does not exist'))).toBe(true);
    });

    it('does not match unrelated errors', () => {
      expect(isNoSuchKeyError({ code: 'AccessDenied' })).toBe(false);
      expect(isNoSuchKeyError({ Code: 'InternalError' })).toBe(false);
      expect(isNoSuchKeyError(new Error('something else went wrong'))).toBe(false);
    });

    it('handles non-object inputs safely', () => {
      expect(isNoSuchKeyError(null)).toBe(false);
      expect(isNoSuchKeyError(undefined)).toBe(false);
      expect(isNoSuchKeyError('NoSuchKey')).toBe(false);
      expect(isNoSuchKeyError(42)).toBe(false);
      expect(isNoSuchKeyError({})).toBe(false);
    });
  });

  // listObjectsUnderPrefix drives the directory-artifact download path. Its
  // pagination logic (continuation tokens) and result normalization (default
  // size, missing-name skip) are easy to break without coverage. Mocks
  // mirror the real minio@8.x shape — `listObjectsV2Query` is async and
  // resolves to a {objects, isTruncated, nextContinuationToken} record.
  describe('listObjectsUnderPrefix', () => {
    type Page = {
      objects: Array<{ name?: string; size?: number }>;
      isTruncated: boolean;
      nextContinuationToken: string;
    };

    async function collect<T>(iter: AsyncGenerator<T>): Promise<T[]> {
      const items: T[] = [];
      for await (const item of iter) {
        items.push(item);
      }
      return items;
    }

    it('yields a single page of objects with name and size', async () => {
      const client = {
        listObjectsV2Query: vi.fn(
          async (): Promise<Page> => ({
            objects: [
              { name: 'a.txt', size: 10 },
              { name: 'b.txt', size: 20 },
            ],
            isTruncated: false,
            nextContinuationToken: '',
          }),
        ),
      } as unknown as MinioClient;

      const results = await collect(listObjectsUnderPrefix(client, 'bucket', 'p/'));
      expect(results).toEqual([
        { name: 'a.txt', size: 10 },
        { name: 'b.txt', size: 20 },
      ]);
      expect((client as any).listObjectsV2Query).toHaveBeenCalledTimes(1);
    });

    it('paginates across multiple pages, threading continuation tokens', async () => {
      const seenTokens: string[] = [];
      const pagesByToken: Record<string, Page> = {
        '': {
          objects: [
            { name: 'page1-a', size: 1 },
            { name: 'page1-b', size: 2 },
          ],
          isTruncated: true,
          nextContinuationToken: 'tok-1',
        },
        'tok-1': {
          objects: [{ name: 'page2-a', size: 3 }],
          isTruncated: true,
          nextContinuationToken: 'tok-2',
        },
        'tok-2': {
          objects: [{ name: 'page3-a', size: 4 }],
          isTruncated: false,
          nextContinuationToken: '',
        },
      };
      const client = {
        listObjectsV2Query: vi.fn(
          async (_bucket: string, _prefix: string, continuationToken: string): Promise<Page> => {
            seenTokens.push(continuationToken);
            return pagesByToken[continuationToken];
          },
        ),
      } as unknown as MinioClient;

      const results = await collect(listObjectsUnderPrefix(client, 'bucket', 'p/'));
      expect(results.map((r) => r.name)).toEqual(['page1-a', 'page1-b', 'page2-a', 'page3-a']);
      // Pages are visited in order, with each call's token coming from the
      // previous response.
      expect(seenTokens).toEqual(['', 'tok-1', 'tok-2']);
    });

    it('defaults missing size to 0 and skips entries without a name', async () => {
      const client = {
        listObjectsV2Query: vi.fn(
          async (): Promise<Page> => ({
            objects: [{ name: 'has-size', size: 42 }, { name: 'no-size' }, { size: 99 }],
            isTruncated: false,
            nextContinuationToken: '',
          }),
        ),
      } as unknown as MinioClient;

      const results = await collect(listObjectsUnderPrefix(client, 'bucket', 'p/'));
      expect(results).toEqual([
        { name: 'has-size', size: 42 },
        { name: 'no-size', size: 0 },
      ]);
    });

    it('throws a clear error if the client does not expose listObjectsV2Query', async () => {
      const client = {} as unknown as MinioClient;
      const iter = listObjectsUnderPrefix(client, 'bucket', 'p/');
      await expect(iter.next()).rejects.toThrow(/listObjectsV2Query/);
    });
  });

  // summarizeDirectoryUnderPrefix backs the bounded directory preview path.
  // It must not paginate — large directories should still cost one round
  // trip, with `truncated: true` signalling there's more.
  describe('summarizeDirectoryUnderPrefix', () => {
    it('returns count and truncated=false for a small, complete listing', async () => {
      const listObjectsV2Query = vi.fn(async () => ({
        objects: [
          { name: 'a', size: 1 },
          { name: 'b', size: 2 },
          { name: 'c', size: 3 },
        ],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const client = { listObjectsV2Query } as unknown as MinioClient;

      const summary = await summarizeDirectoryUnderPrefix(client, 'bucket', 'p/');
      expect(summary).toEqual({ count: 3, truncated: false });
      expect(listObjectsV2Query).toHaveBeenCalledTimes(1);
    });

    it('returns truncated=true when minio reports more pages exist', async () => {
      const listObjectsV2Query = vi.fn(async () => ({
        objects: Array.from({ length: 50 }, (_, i) => ({ name: `f-${i}`, size: 1 })),
        isTruncated: true,
        nextContinuationToken: 'next',
      }));
      const client = { listObjectsV2Query } as unknown as MinioClient;

      const summary = await summarizeDirectoryUnderPrefix(client, 'bucket', 'p/');
      expect(summary).toEqual({ count: 50, truncated: true });
      // Bounded — does not loop on the continuation token.
      expect(listObjectsV2Query).toHaveBeenCalledTimes(1);
    });

    it('returns null for an empty prefix so callers can answer 404', async () => {
      const listObjectsV2Query = vi.fn(async () => ({
        objects: [],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const client = { listObjectsV2Query } as unknown as MinioClient;

      const summary = await summarizeDirectoryUnderPrefix(client, 'bucket', 'p/');
      expect(summary).toBeNull();
    });

    it('passes the configured maxKeys cap to listObjectsV2Query', async () => {
      const listObjectsV2Query = vi.fn(async () => ({
        objects: [{ name: 'a', size: 1 }],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const client = { listObjectsV2Query } as unknown as MinioClient;

      await summarizeDirectoryUnderPrefix(client, 'bucket', 'p/', 25);
      expect(listObjectsV2Query).toHaveBeenCalledWith('bucket', 'p/', '', '', 25, '');
    });
  });
});
