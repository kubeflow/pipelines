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
import { S3Client } from '@aws-sdk/client-s3';
import {
  MockS3Client,
  expectedS3Options,
  resolveS3TestCredentials,
} from './integration-tests/s3-client-test-helper.js';
import type { ArtifactStore } from './artifact-store.js';
import {
  createArtifactStoreClient,
  ArtifactStoreConfigurationError,
  getArtifactStoreOriginAfterRewrite,
  createS3ArtifactStore,
  isTarball,
  maybeTarball,
  getObjectStream,
  getArtifactStoreOrigin,
  isNoSuchKeyError,
  listObjectsUnderPrefix,
  summarizeDirectoryUnderPrefix,
  ArtifactStoreOptions,
  Credentials,
  TEST_ONLY,
} from './artifact-store.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';
import { getK8sSecret } from './k8s-helper.js';

vi.mock('@aws-sdk/client-s3', async () => {
  const actual = await vi.importActual<typeof import('@aws-sdk/client-s3')>('@aws-sdk/client-s3');
  const { createS3Mock } = await import('./integration-tests/s3-client-test-helper.js');
  return { ...actual, S3Client: createS3Mock(actual) };
});
const createTestArtifactStore = (
  config: ArtifactStoreOptions,
  providerType: string,
  ...args: [
    string?,
    string?,
    (() => Promise<Credentials> | Credentials)?,
    AbortSignal?,
    import('./artifact-store.js').ArtifactStoreEndpointPolicy?,
  ]
) => createS3ArtifactStore(config, providerType, 'bucket', ...args);
vi.mock('@aws-sdk/credential-providers');
vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

async function serializedRequest(options: { bucketName?: string; method: string }) {
  const actual = await vi.importActual<typeof import('@aws-sdk/client-s3')>('@aws-sdk/client-s3');
  const config = vi.mocked(S3Client).mock.calls.at(-1)![0]!;
  const fixture = MockS3Client.mock.results.at(-1)?.value as {
    getRequestOptions?: () => { path: string };
  };
  const originalPath = fixture.getRequestOptions?.().path ?? '/object';
  const bucket = options.bucketName ?? 'bucket';
  const key = (
    config.forcePathStyle || bucket.includes('.')
      ? originalPath.replace(new RegExp('^/' + bucket + '/'), '/')
      : originalPath
  )
    .replace(/^\//, '')
    .split('?')[0];
  let captured: { headers: { host: string }; host: string; path: string } | undefined;
  const realClient = new actual.S3Client({
    ...config,
    requestHandler: {
      handle: async (request: {
        headers: Record<string, string>;
        hostname: string;
        path: string;
        protocol: string;
      }) => {
        captured = {
          headers: { host: request.headers.host },
          host: request.hostname,
          path: request.path,
        };
        Object.defineProperty(captured, 'protocol', { value: request.protocol });
        return { response: { statusCode: 200, headers: {}, body: PassThrough.from(['artifact']) } };
      },
    },
  });
  const stack = vi.mocked(S3Client).mock.results.at(-1)!.value.middlewareStack;
  for (const [middleware, options] of stack.add.mock.calls)
    realClient.middlewareStack.add(middleware, options);
  await realClient.send(new actual.GetObjectCommand({ Bucket: bucket, Key: key }));
  return captured!;
}
async function storeForFixture(fixture: { listPage: Mock }) {
  MockS3Client.mockImplementationOnce(() => ({ getObject: vi.fn(), ...fixture }));
  return createTestArtifactStore(
    { endPoint: 'store.example', accessKey: 'access', secretKey: 'secret' },
    'minio',
  );
}
describe('artifact-store', () => {
  const MockedS3Client = MockS3Client;
  const MockedAuthorizeFn: Mock = vi.fn((x) => undefined);
  const storageFixture = (overrides: Record<string, unknown>) => ({
    getObject: vi.fn(),
    listPage: vi.fn(),
    retryOptions: {},
    ...overrides,
  });
  const transportError = (message: string, code: string = 'ECONNRESET', syscall = 'read') =>
    TEST_ONLY.markS3TransportError(
      Object.assign(new Error(message), { code, errno: -104, syscall }),
    );

  beforeEach(() => {
    vi.resetAllMocks();
  });

  describe('getArtifactStoreOrigin', () => {
    it.each([
      [{ endPoint: 'store.example.com' }, 'https://store.example.com'],
      [{ endPoint: 'STORE.example.com', useSSL: false }, 'http://store.example.com'],
      [
        { endPoint: 'store.example.com', port: 9000, useSSL: false },
        'http://store.example.com:9000',
      ],
      [{ endPoint: 'store.example.com', port: 443, useSSL: true }, 'https://store.example.com'],
      [
        { endPoint: 'https://store.example.com:8443', useSSL: true },
        'https://store.example.com:8443',
      ],
      [{ endPoint: 'http://store.example.com', useSSL: true }, undefined],
      [{ endPoint: 'https://user@store.example.com', port: 9000 }, undefined],
      [{ endPoint: 'https://store.example.com/path', port: 9000 }, undefined],
      [{ endPoint: 'https://store.example.com?query=value', port: 9000 }, undefined],
      [{ endPoint: 'https://store.example.com#fragment', port: 9000 }, undefined],
      [{ endPoint: 'store.example.com:invalid' }, undefined],
    ])('resolves configured client %j to %s', (config, origin) => {
      expect(getArtifactStoreOrigin(config)).toBe(origin);
    });
  });

  describe('artifact endpoint authorization', () => {
    it.each([undefined, 9000])(
      'preserves port %s on endpoint-less structured TLS upgrades',
      async (port) => {
        const authorizeEndpoint = vi.fn();
        await createTestArtifactStore(
          {
            endPoint: 'store.example',
            useSSL: false,
            port,
            accessKey: 'access',
            secretKey: 'secret',
          },
          's3',
          JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true', disableSSL: 'false' } }),
          undefined,
          undefined,
          undefined,
          authorizeEndpoint,
        );
        expect(authorizeEndpoint).toHaveBeenCalledWith(
          expect.objectContaining({
            origin: port ? `https://store.example:${port}` : 'https://store.example',
            useSSL: true,
            hasEndpointOverride: false,
          }),
        );
        await resolveS3TestCredentials();
        expect(MockedS3Client).toHaveBeenCalledWith(
          expectedS3Options({ endPoint: 'store.example', useSSL: true, port }),
        );
      },
    );

    it('authorizes native IPv6 origins without losing their client host or port', async () => {
      const authorizeEndpoint = vi.fn();
      await createTestArtifactStore(
        { endPoint: 'store.example', accessKey: 'access', secretKey: 'secret' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            fromEnv: 'true',
            endpoint: 'https://[2001:db8::1]:9443/base',
            nativeQuery: 'true',
          },
        }),
        undefined,
        undefined,
        undefined,
        authorizeEndpoint,
      );
      expect(authorizeEndpoint).toHaveBeenCalledWith(
        expect.objectContaining({
          origin: 'https://[2001:db8::1]:9443',
          endPoint: '2001:db8::1',
          port: 9443,
        }),
      );
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({ endPoint: '2001:db8::1', port: 9443 }),
      );
    });

    it.each([0, -1, 65536, 1.5, NaN])(
      'rejects invalid configured port %s before credentials',
      async (port) => {
        const customProvider = vi.fn();
        await expect(
          createTestArtifactStore(
            { endPoint: 'store.example', port },
            's3',
            undefined,
            undefined,
            customProvider,
            undefined,
            vi.fn(),
          ),
        ).rejects.toThrow(ArtifactStoreConfigurationError);
        expect(customProvider).not.toHaveBeenCalled();
        expect(MockedS3Client).not.toHaveBeenCalled();
        expect(
          getArtifactStoreOriginAfterRewrite({ endPoint: 'store.example', port }),
        ).toBeUndefined();
      },
    );

    it('classifies malformed anonymous options as configuration errors', async () => {
      const customProvider = vi.fn();
      await expect(
        createTestArtifactStore(
          { endPoint: 'store.example' },
          's3',
          JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true', anonymous: 'invalid' } }),
          undefined,
          customProvider,
          undefined,
          vi.fn(),
        ),
      ).rejects.toThrow(ArtifactStoreConfigurationError);
      expect(customProvider).not.toHaveBeenCalled();
      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it.each(['true', 'false'])(
      'rejects an untrusted native endpoint before credentials with fromEnv=%s',
      async (fromEnv) => {
        const customProvider = vi.fn();
        const authorizeEndpoint = vi.fn(() => {
          throw new ArtifactStoreConfigurationError('untrusted origin');
        });
        await expect(
          createTestArtifactStore(
            { endPoint: 'trusted.example', useSSL: true },
            's3',
            JSON.stringify({
              Provider: 's3',
              Params: {
                endpoint: 'https://untrusted.example/base',
                nativeQuery: 'true',
                fromEnv,
                secretName: 'store',
                accessKeyKey: 'access',
                secretKeyKey: 'secret',
              },
            }),
            'kubeflow',
            customProvider,
            undefined,
            authorizeEndpoint,
          ),
        ).rejects.toThrow(ArtifactStoreConfigurationError);
        expect(authorizeEndpoint).toHaveBeenCalledWith(
          expect.objectContaining({
            origin: 'https://untrusted.example',
            providerType: 's3',
            hasEndpointOverride: true,
          }),
        );
        expect(customProvider).not.toHaveBeenCalled();
        expect(getK8sSecret).not.toHaveBeenCalled();
        expect(fromNodeProviderChain).not.toHaveBeenCalled();
        expect(MockedS3Client).not.toHaveBeenCalled();
      },
    );

    it('authorizes the same rewritten request-local config used by the client', async () => {
      const defaults = {
        endPoint: 'central.example',
        useSSL: true,
        accessKey: 'access',
        secretKey: 'secret',
        endpointRewrite:
          'native.example:9443=http://proxy.example:9000,proxy.example:9000=other.example:8000',
      };
      const authorizeEndpoint = vi.fn();
      await createArtifactStoreClient(
        { minio: { endPoint: 'minio.example' }, s3: defaults },
        'minio',
        'bucket',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://native.example:9443/base',
            nativeQuery: 'true',
            fromEnv: 'true',
          },
        }),
        undefined,
        undefined,
        authorizeEndpoint,
      );
      expect(authorizeEndpoint).toHaveBeenCalledExactlyOnceWith(
        expect.objectContaining({
          providerType: 's3',
          origin: 'http://proxy.example:9000',
          useSSL: false,
          port: 9000,
        }),
      );
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          endPoint: 'proxy.example',
          port: 9000,
          useSSL: false,
        }),
      );
      expect(defaults.endPoint).toBe('central.example');
      expect(
        getArtifactStoreOriginAfterRewrite({ ...defaults, endPoint: 'native.example', port: 9443 }),
      ).toBe('http://proxy.example:9000');
    });

    it('exposes endpoint-less native TLS aliases to policy before reading credentials', async () => {
      const authorizeEndpoint = vi.fn(() => {
        throw new ArtifactStoreConfigurationError('TLS downgrade');
      });
      await expect(
        createTestArtifactStore(
          { endPoint: 'store.example', useSSL: true },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: { fromEnv: 'true', nativeQuery: 'true', disable_https: '1' },
          }),
          undefined,
          undefined,
          undefined,
          authorizeEndpoint,
        ),
      ).rejects.toThrow('TLS downgrade');
      expect(authorizeEndpoint).toHaveBeenCalledWith(
        expect.objectContaining({
          origin: 'http://store.example',
          hasEndpointOverride: false,
        }),
      );
      expect(fromNodeProviderChain).not.toHaveBeenCalled();
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('rejects endpoint credentials before resolving Secret credentials', async () => {
      await expect(
        createTestArtifactStore(
          { endPoint: 'store.example' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              endpoint: 'https://user:password@store.example',
              fromEnv: 'false',
              secretName: 'store',
              accessKeyKey: 'access',
              secretKeyKey: 'secret',
            },
          }),
          'kubeflow',
        ),
      ).rejects.toThrow(ArtifactStoreConfigurationError);
      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MockedS3Client).not.toHaveBeenCalled();
    });
  });

  describe('createTestArtifactStore', () => {
    it('preserves a configured HTTP transport with regional AWS routing', async () => {
      MockedS3Client.mockImplementationOnce(() =>
        storageFixture({ getRequestOptions: () => ({ path: '/object' }) }),
      );
      await createTestArtifactStore(
        {
          endPoint: 's3.amazonaws.com',
          region: 'us-west-2',
          useSSL: false,
          accessKey: 'access',
          secretKey: 'secret',
        },
        's3',
      );
      const request = await serializedRequest({ bucketName: 'bucket', method: 'GET' });
      expect(request.host).toBe('bucket.s3.us-west-2.amazonaws.com');
      expect((request as { protocol?: string }).protocol).toBe('http:');
    });

    it.each([
      ['missing Params', { Provider: 's3' }],
      ['null Params', { Params: null, Provider: 's3' }],
      ['array Params', { Params: [], Provider: 's3' }],
    ])('rejects provider info with %s', async (_name, providerInfo) => {
      await expect(
        createTestArtifactStore(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify(providerInfo),
          'kubeflow',
        ),
      ).rejects.toThrow('Invalid provider info');
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it.each([undefined, null, false, 1, 'yes'])(
      'rejects invalid fromEnv value %j',
      async (fromEnv) => {
        await expect(
          createTestArtifactStore(
            { endPoint: 's3.amazonaws.com' },
            's3',
            JSON.stringify({ Params: { fromEnv }, Provider: 's3' }),
            'kubeflow',
          ),
        ).rejects.toThrow('Provider info fromEnv must be true or false');
        expect(MockedS3Client).not.toHaveBeenCalled();
      },
    );

    it.each([
      ['', 'secret'],
      ['access', ''],
    ])(
      'fails closed when an explicit provider Secret contains credentials %j / %j',
      async (accessKey, secretKey) => {
        vi.mocked(getK8sSecret).mockResolvedValueOnce(accessKey).mockResolvedValueOnce(secretKey);

        await expect(
          createTestArtifactStore(
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
        expect(MockedS3Client).not.toHaveBeenCalled();
      },
    );

    it('does not mutate shared defaults when applying request provider info', async () => {
      const defaults = {
        accessKey: 'default-access',
        endPoint: 's3.amazonaws.com',
        port: 443,
        region: 'us-east-1',
        secretKey: 'default-secret',
        useSSL: true,
      };
      (getK8sSecret as Mock).mockResolvedValue('request-secret');

      await createTestArtifactStore(
        defaults,
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            accessKeyKey: 'access',
            disableSSL: 'true',
            endpoint: 'http://store.example.com:9000',
            fromEnv: 'false',
            region: 'custom',
            secretKeyKey: 'secret',
            secretName: 'credentials',
          },
        }),
        'kubeflow',
      );

      expect(defaults).toEqual({
        accessKey: 'default-access',
        endPoint: 's3.amazonaws.com',
        port: 443,
        region: 'us-east-1',
        secretKey: 'default-secret',
        useSSL: true,
      });
    });
    it('creates a S3 store with the provided configs.', async () => {
      const client = await createTestArtifactStore(
        {
          accessKey: 'accesskey',
          endPoint: 'minio.kubeflow:80',
          secretKey: 'secretkey',
        },
        's3',
      );

      expect(client.openRead).toBeTypeOf('function');
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'minio.kubeflow:80',
          secretKey: 'secretkey',
        }),
      );
    });

    it('Builds a client where credentials are resolved using a custom provider.', async () => {
      const provider = async (): Promise<Credentials> => {
        return {
          accessKeyId: 'providedKey',
          secretAccessKey: 'providedSecret',
          sessionToken: 'providedToken',
        };
      };

      const client = await createTestArtifactStore(
        {
          endPoint: 'minio.kubeflow:80',
        },
        's3',
        '',
        '',
        provider,
      );

      expect(client.openRead).toBeTypeOf('function');
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'providedKey',
          endPoint: 'minio.kubeflow:80',
          secretKey: 'providedSecret',
          sessionToken: 'providedToken',
        }),
      );
    });

    it('fails closed if authenticated S3 credentials are unavailable.', async () => {
      await expect(
        createTestArtifactStore({ endPoint: 'minio.kubeflow:80' }, 's3'),
      ).rejects.toThrow('Unable to resolve AWS credentials');

      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('applies endpoint and region settings when credentials come from the environment', async () => {
      const client = await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'ceph.example',
          port: 9443,
          region: 'ceph',
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it('uses standard AWS HTTPS for an explicitly empty structured S3 endpoint', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 's3.amazonaws.com',
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it('applies structured provider retry settings', async () => {
      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '5' },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          secretKey: 'secretkey',
        }),
      );
      expect(vi.mocked(S3Client).mock.calls.at(-1)?.[0]?.maxAttempts).toBe(1);
    });

    it('uses an operation retry budget when structured maxRetries is zero', async () => {
      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '0' },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          secretKey: 'secretkey',
        }),
      );
      expect(vi.mocked(S3Client).mock.calls.at(-1)?.[0]?.maxAttempts).toBe(1);
    });

    it('uses the Go default retry budget when maxRetries is omitted', async () => {
      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({ Provider: 'minio', Params: { fromEnv: 'true' } }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          secretKey: 'secretkey',
        }),
      );
      expect(vi.mocked(S3Client).mock.calls.at(-1)?.[0]?.maxAttempts).toBe(1);
    });

    it('accepts retry budgets above the frontend safety limit', async () => {
      await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
        JSON.stringify({
          Provider: 'minio',
          Params: { fromEnv: 'true', maxRetries: '20' },
        }),
      );

      expect(MockedS3Client).toHaveBeenCalledTimes(1);
    });

    it('rejects malformed native endpoint authorities instead of repairing them', async () => {
      await expect(
        createTestArtifactStore(
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

      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('rejects backslashes in native endpoint authorities', async () => {
      await expect(
        createTestArtifactStore(
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

      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('uses the secure default when provider info does not specify disableSSL', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'ceph.example',
          port: 9443,
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it('preserves an explicit HTTP endpoint scheme when disableSSL is false', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'ceph.example',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('retains the server TLS setting when provider info does not override transport', async () => {
      await createTestArtifactStore(
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
          Params: { fromEnv: 'true' },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'minio-service.kubeflow',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('applies endpoint-less disableSSL=true while inheriting the server endpoint and region', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'minio-service.kubeflow',
          region: 'environment-region',
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('supports anonymous virtual-hosted S3 access without resolving credentials', async () => {
      await createTestArtifactStore(
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
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          endPoint: 's3.us-west-2.amazonaws.com',
          pathStyle: false,
          port: undefined,
          useSSL: true,
        }),
      );
    });

    it('rejects unsupported S3 read options before resolving ambient credentials', async () => {
      await expect(
        createTestArtifactStore(
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
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('accepts valid neutral S3 options that do not change read behavior', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 's3.us-west-2.amazonaws.com',
          port: undefined,
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it.each([
      ['profile', 'team-profile'],
      ['request_checksum_calculation', 'when_required'],
      ['response_checksum_validation', 'when_required'],
    ])('rejects unsupported non-neutral S3 option %s', async (option, value) => {
      await expect(
        createTestArtifactStore(
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
          createTestArtifactStore(
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
        createTestArtifactStore(
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
        createTestArtifactStore(
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
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('parses an explicit HTTP custom endpoint into S3 configuration', async () => {
      await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'http://store.example.com',
            fromEnv: 'true',
          },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example.com',
          port: undefined,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('applies disableSSL to a scheme-less custom endpoint', async () => {
      await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            disableSSL: 'true',
            endpoint: 'store.example.com',
            fromEnv: 'true',
          },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example.com',
          port: undefined,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('applies runtime S3 query aliases', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'ceph.example',
          pathStyle: false,
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('applies endpoint-less disable_https from a native S3 query', async () => {
      await createTestArtifactStore(
        {
          accessKey: 'accesskey',
          endPoint: 'store.example.com',
          secretKey: 'secretkey',
          useSSL: true,
        },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: { disable_https: 'true', fromEnv: 'true', nativeQuery: 'true' },
        }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example.com',
          pathStyle: false,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('defaults native custom S3 endpoints to virtual-host addressing', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          pathStyle: false,
          port: undefined,
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it('parses uppercase native HTTPS endpoint schemes', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          pathStyle: false,
          port: 9443,
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it('rejects scheme-less endpoints from native S3 queries', async () => {
      await expect(
        createTestArtifactStore(
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
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it.each([
      ['disableSSL', 'true', 'https://store.example:9000', false],
      ['disableSSL', 'false', 'http://store.example:9000', false],
      ['disable_https', 'true', 'https://store.example:9000', false],
      ['disable_https', 'false', 'http://store.example:9000', false],
    ])(
      'applies asymmetric TLS precedence for %s=%s and an explicit endpoint',
      async (option, value, endpoint, expectedUseSSL) => {
        await createTestArtifactStore(
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

        await resolveS3TestCredentials();
        expect(MockedS3Client).toHaveBeenCalledWith(
          expectedS3Options({
            accessKey: 'accesskey',
            endPoint: 'store.example',
            port: 9000,
            secretKey: 'secretkey',
            useSSL: expectedUseSSL,
          }),
        );
      },
    );

    it('uses the AWS default credential chain with a custom S3 endpoint', async () => {
      (fromNodeProviderChain as Mock).mockReturnValueOnce(async () => ({
        accessKeyId: 'irsa-access-key',
        secretAccessKey: 'irsa-secret-key',
        sessionToken: 'irsa-session-token',
      }));

      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'irsa-access-key',
          endPoint: 'ceph.example',
          pathStyle: false,
          port: 9443,
          secretKey: 'irsa-secret-key',
          sessionToken: 'irsa-session-token',
          useSSL: true,
        }),
      );
    });

    it('passes bare IPv6 addresses from native endpoints to the S3 client', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: '2001:db8::1',
          pathStyle: false,
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('preserves endpoint base paths in the signed path and requests', async () => {
      const getRequestOptions = vi.fn(() => ({ path: '/bucket/object?versionId=1' }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      const requestOptions = await serializedRequest({ method: 'GET' });
      expect(requestOptions.path).toBe('/base/path/bucket/object');
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'store.example',
          port: undefined,
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
    });

    it.each([
      ['https://store.example/root dir/café', '/root%20dir/caf%C3%A9/bucket/object'],
      ['https://store.example/base/%2e%2e/inner', '/base/../inner/bucket/object'],
      ['https://store.example/base/%3F/discarded', '/base/object'],
      ['https://store.example/base/%23/discarded', '/base/object'],
    ])('preserves the Go request spelling for endpoint path %s', async (endpoint, expectedPath) => {
      const getRequestOptions = vi.fn(() => ({ path: '/bucket/object' }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({ Provider: 's3', Params: { endpoint, fromEnv: 'true' } }),
      );

      expect((await serializedRequest({ method: 'GET' })).path).toBe(expectedPath);
    });

    it('preserves an explicitly selected global AWS endpoint authority', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.us-west-2.amazonaws.com' },
        host: 'bucket.s3.us-west-2.amazonaws.com',
        path: '/object',
      }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      expect(await serializedRequest({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 'bucket.s3.amazonaws.com' },
          host: 'bucket.s3.amazonaws.com',
          path: '/base/object',
        }),
      );
    });

    it('preserves a virtual-host key matching the bucket across endpoint delimiter truncation', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'mybucket.s3.us-west-2.amazonaws.com' },
        host: 'mybucket.s3.us-west-2.amazonaws.com',
        path: '/mybucket/data.csv',
      }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            endpoint: 'https://s3.us-west-2.amazonaws.com/base/%3F/discarded',
            fromEnv: 'true',
            nativeQuery: 'true',
          },
        }),
      );

      expect(await serializedRequest({ bucketName: 'mybucket', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 'mybucket.s3.us-west-2.amazonaws.com' },
          host: 'mybucket.s3.us-west-2.amazonaws.com',
          path: '/base/mybucket/data.csv',
        }),
      );
    });

    it('preserves path-style addressing for dotted HTTPS buckets', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 's3.amazonaws.com' },
        host: 's3.amazonaws.com',
        path: '/bucket.with.dot/object',
      }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      expect(await serializedRequest({ bucketName: 'bucket.with.dot', method: 'GET' })).toEqual(
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
      if (endpoint.startsWith('http:')) {
        await expect(
          createTestArtifactStore(
            { endPoint: 's3.amazonaws.com' },
            's3',
            JSON.stringify({
              Provider: 's3',
              Params: { endpoint, fromEnv: 'true', nativeQuery: 'true' },
            }),
          ),
        ).rejects.toThrow('AWS S3 provider endpoints must use HTTPS');
        expect(MockedS3Client).not.toHaveBeenCalled();
        expect(fromNodeProviderChain).not.toHaveBeenCalled();
        return;
      }
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.amazonaws.com' },
        host: 'bucket.s3.amazonaws.com',
        path: '/object',
      }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: { endpoint, fromEnv: 'true', nativeQuery: 'true' },
        }),
      );

      expect(await serializedRequest({ bucketName: 'bucket', method: 'GET' })).toEqual(
        expect.objectContaining({ headers: { host: `bucket.s3.amazonaws.com:${port}` } }),
      );
    });

    it('preserves an explicit China partition S3 authority', async () => {
      const getRequestOptions = vi.fn(() => ({
        headers: { host: 'bucket.s3.cn-northwest-1.amazonaws.com.cn' },
        host: 'bucket.s3.cn-northwest-1.amazonaws.com.cn',
        path: '/object',
      }));
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      expect(await serializedRequest({ bucketName: 'bucket', method: 'GET' })).toEqual(
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
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      expect(await serializedRequest({ bucketName: 'bucket', method: 'GET' })).toEqual(
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
      MockedS3Client.mockImplementationOnce(function () {
        return storageFixture({ getRequestOptions });
      });

      const client = await createTestArtifactStore(
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

      expect(await serializedRequest({ bucketName: 's3', method: 'GET' })).toEqual(
        expect.objectContaining({
          headers: { host: 's3.us-west-2.amazonaws.com' },
          host: 's3.us-west-2.amazonaws.com',
          path: '/base/s3/hello.txt',
        }),
      );
    });

    it('normalizes structured global AWS endpoints through the standard resolver', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 's3.amazonaws.com',
          secretKey: 'secretkey',
          useSSL: true,
        }),
      );
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
      expect(() => TEST_ONLY.parseProviderEndpoint('https://store/%00', true)).toThrow(
        'unsupported URL escape',
      );
      expect(TEST_ONLY.parseProviderEndpoint('https://store/%FF/%C0%AF/%ED%A0%80', true)).toEqual(
        expect.objectContaining({ basePath: '/%FF/%C0%AF/%ED%A0%80', host: 'store' }),
      );
    });

    it.each([
      ['%2525', '/%25'],
      ['%25FF', '/%FF'],
      ['%253F', '/%3F'],
    ])('accepts the Go-valid second-layer endpoint escape %s', (path, basePath) => {
      expect(TEST_ONLY.parseProviderEndpoint(`https://store/${path}`, true)).toEqual(
        expect.objectContaining({ basePath, host: 'store' }),
      );
    });

    it.each(['%25', '%00', '%7F', '%25zz', '%252G'])(
      'rejects the endpoint path escape %s that Go rejects',
      (path) => {
        expect(() => TEST_ONLY.parseProviderEndpoint(`https://store/${path}`, true)).toThrow(
          'unsupported URL escape',
        );
      },
    );

    it('does not normalize a hostname that merely starts with the standard AWS name', async () => {
      await createTestArtifactStore(
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

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({ endPoint: 's3.amazonaws.com.tenant.example' }),
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

      await createTestArtifactStore(
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
      await createTestArtifactStore(sharedConfig, 'minio');

      expect(sharedConfig).toEqual({
        accessKey: 'default-access-key',
        endPoint: 'default-store',
        port: 9000,
        secretKey: 'default-secret-key',
        useSSL: false,
      });
      expect(MockedS3Client).toHaveBeenNthCalledWith(2, expectedS3Options(sharedConfig));
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
      const client = await createTestArtifactStore({ endPoint: 's3.amazonaws.com' }, 's3');
      expect(client.openRead).toBeTypeOf('function');
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'AccessKeyId',
          endPoint: 's3.amazonaws.com',
          secretKey: 'SecretAccessKey',
          sessionToken: 'SessionToken',
        }),
      );
      expect(MockedS3Client).toBeCalledTimes(1);
    });

    it.each([
      ['s3-fips.us-gov-west-1.amazonaws.com', 'us-gov-west-1'],
      ['s3-fips.us-gov-east-1.amazonaws.com', 'us-gov-east-1'],
      ['s3-fips-us-gov-west-1.amazonaws.com', 'us-gov-west-1'],
      ['s3-fips-us-gov-east-1.amazonaws.com', 'us-gov-east-1'],
    ])('resolves AWS credentials for the GovCloud FIPS endpoint %s', async (endPoint, region) => {
      const credentialProvider = vi.fn().mockResolvedValue({
        accessKeyId: 'AccessKeyId',
        secretAccessKey: 'SecretAccessKey',
        sessionToken: 'SessionToken',
      });
      (fromNodeProviderChain as Mock).mockReturnValue(credentialProvider);

      await createTestArtifactStore({ endPoint, region, useSSL: true }, 's3');

      expect(fromNodeProviderChain).toHaveBeenCalledExactlyOnceWith({ ignoreCache: true });
      expect(credentialProvider).toHaveBeenCalledTimes(1);
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledExactlyOnceWith(
        expectedS3Options({
          accessKey: 'AccessKeyId',
          endPoint,
          region,
          secretKey: 'SecretAccessKey',
          sessionToken: 'SessionToken',
          useSSL: true,
        }),
      );
    });

    it('uses the AWS credential chain for S3 acceleration endpoints.', async () => {
      (fromNodeProviderChain as Mock).mockImplementation(
        () => () =>
          Promise.resolve({
            accessKeyId: 'AccessKeyId',
            secretAccessKey: 'SecretAccessKey',
            sessionToken: 'SessionToken',
          }),
      );

      await createTestArtifactStore({ endPoint: 's3-accelerate.amazonaws.com' }, 's3');

      expect(fromNodeProviderChain).toHaveBeenCalledWith({ ignoreCache: true });
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'AccessKeyId',
          endPoint: 's3-accelerate.amazonaws.com',
          secretKey: 'SecretAccessKey',
        }),
      );
    });

    it('preserves the configured AWS region when provider info omits a region.', async () => {
      (getK8sSecret as Mock).mockResolvedValue('request-secret');

      await createTestArtifactStore(
        { endPoint: 's3.amazonaws.com', region: 'eu-west-2' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            accessKeyKey: 'access',
            fromEnv: 'false',
            secretKeyKey: 'secret',
            secretName: 'credentials',
          },
        }),
        'kubeflow',
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(expectedS3Options({ region: 'eu-west-2' }));
    });

    it('rejects an insecure AWS provider endpoint before reading its Secret', async () => {
      await expect(
        createTestArtifactStore(
          { endPoint: 's3.amazonaws.com' },
          's3',
          JSON.stringify({
            Provider: 's3',
            Params: {
              accessKeyKey: 'access',
              disableSSL: 'true',
              endpoint: 'http://s3.amazonaws.com:80',
              fromEnv: 'false',
              secretKeyKey: 'secret',
              secretName: 'credentials',
            },
          }),
          'kubeflow',
        ),
      ).rejects.toThrow('AWS S3 provider endpoints must use HTTPS');

      expect(getK8sSecret).not.toHaveBeenCalled();
      expect(MockedS3Client).not.toHaveBeenCalled();
    });

    it('selects the S3 base configuration inside artifact client creation', async () => {
      (fromNodeProviderChain as Mock).mockImplementation(
        () => () =>
          Promise.resolve({ accessKeyId: 'AccessKeyId', secretAccessKey: 'SecretAccessKey' }),
      );

      await createArtifactStoreClient(
        {
          minio: { endPoint: 'central-minio.example' },
          s3: { endPoint: 's3.amazonaws.com', useSSL: true },
        },
        'minio',
        'bucket',
        JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true' } }),
      );

      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({ endPoint: 's3.amazonaws.com', useSSL: true }),
      );
    });
    it('rewrites configured in-cluster endpoints to local proxy endpoints.', async () => {
      const client = await createTestArtifactStore(
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

      expect(client.openRead).toBeTypeOf('function');
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'localhost',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
    });

    it('skips invalid endpoint rewrite rules.', async () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);

      const client = await createTestArtifactStore(
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

      expect(client.openRead).toBeTypeOf('function');
      expect(consoleWarnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Ignoring invalid artifact store endpoint rewrite endpoint "::::"'),
      );
      await resolveS3TestCredentials();
      expect(MockedS3Client).toHaveBeenCalledWith(
        expectedS3Options({
          accessKey: 'accesskey',
          endPoint: 'localhost',
          port: 9000,
          secretKey: 'secretkey',
          useSSL: false,
        }),
      );
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
        const getObject = vi.fn().mockRejectedValue(transportError('connection reset'));
        const listPage = vi.fn();
        MockedS3Client.mockImplementationOnce(function () {
          return { getObject, listPage };
        });
        const client = await createTestArtifactStore(
          { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
          'minio',
          JSON.stringify({
            Provider: 'minio',
            Params: { fromEnv: 'true', maxRetries: String(maxRetries) },
          }),
        );

        const request = expect(client.openRead('key')).rejects.toThrow('connection reset');
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
      const operation = vi.fn().mockRejectedValue(transportError(code, code, 'connect'));

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

    it('fails loudly when the SDK returns no readable body', async () => {
      MockedS3Client.mockImplementationOnce(function () {
        return { getObject: vi.fn() };
      });

      const store = await createTestArtifactStore(
        { accessKey: 'accesskey', secretKey: 'secretkey', endPoint: 'store.example' },
        'minio',
      );
      await expect(store.openRead('key')).rejects.toThrow('no readable object body');
    });

    it.each([
      [0, 3],
      [5, 5],
    ])('matches Go total-attempt semantics for maxRetries=%i', async (maxRetries, attempts) => {
      const operation = vi.fn().mockRejectedValue(transportError('connection reset'));

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
      expect(
        TEST_ONLY.isRetryableS3Error({
          code: '',
          message: 'Request failed after 0 retries: Error: Retryable HTTP status: 500',
        }),
      ).toBe(false);
      expect(
        TEST_ONLY.isRetryableS3Error({
          message: 'Request failed after 0 retries: Error: Retryable HTTP status: 500',
        }),
      ).toBe(false);
      ['EAI_FAIL', 'ENOBUFS', 'EINTR', 'ESHUTDOWN'].forEach((code) => {
        expect(TEST_ONLY.isRetryableS3Error({ code, errno: -1, syscall: 'read' })).toBe(false);
      });
      expect(
        TEST_ONLY.isRetryableS3Error({ code: 'EALREADY', errno: -1, syscall: 'connect' }),
      ).toBe(false);
      expect(TEST_ONLY.isRetryableS3Error({ code: 'AccessDenied', status: 500 }, 400)).toBe(false);
    });

    it.each(['EALREADY', 'ENETRESET', 'ESHUTDOWN', 'ENOBUFS'])(
      'retries structurally identified dial failure %s',
      async (code) => {
        const operation = vi.fn().mockRejectedValue(transportError(code, code, 'connect'));

        await expect(
          TEST_ONLY.retryS3Operation(operation, 3, async () => undefined),
        ).rejects.toThrow(code);
        expect(operation).toHaveBeenCalledTimes(3);
      },
    );

    it('does not retry a permanent DNS lookup failure', async () => {
      const operation = vi.fn().mockRejectedValue(
        Object.assign(new Error('not found'), {
          code: 'ENOTFOUND',
          errno: -3008,
          syscall: 'getaddrinfo',
        }),
      );

      await expect(TEST_ONLY.retryS3Operation(operation, 3, async () => undefined)).rejects.toThrow(
        'not found',
      );
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it.each([
      ['EAI_FAIL', 'getaddrinfo'],
      ['ENETRESET', 'read'],
      ['ENOBUFS', 'read'],
      ['EINTR', 'read'],
      ['ESHUTDOWN', 'read'],
    ])('retries transient network failure %s during %s', async (code, syscall) => {
      const operation = vi.fn().mockRejectedValue(transportError(code, code, syscall));

      await expect(TEST_ONLY.retryS3Operation(operation, 3, async () => undefined)).rejects.toThrow(
        code,
      );
      expect(operation).toHaveBeenCalledTimes(3);
    });

    it('gives each fallback operation its own attempt budget', async () => {
      const context = TEST_ONLY.createS3RetryContext(3);
      const missingObject = vi
        .fn()
        .mockRejectedValue(Object.assign(new Error('missing'), { code: 'NoSuchKey' }));
      const failingSummary = vi.fn().mockRejectedValue(transportError('reset'));

      await expect(
        TEST_ONLY.retryS3Operation(missingObject, context, async () => undefined),
      ).rejects.toThrow('missing');
      await expect(
        TEST_ONLY.retryS3Operation(failingSummary, context, async () => undefined),
      ).rejects.toThrow('reset');

      expect(missingObject).toHaveBeenCalledTimes(1);
      expect(failingSummary).toHaveBeenCalledTimes(3);
    });

    it('gives configured client object and fallback methods independent budgets', async () => {
      vi.useFakeTimers();
      try {
        const getObject = vi
          .fn()
          .mockRejectedValue(Object.assign(new Error('missing'), { code: 'NoSuchKey' }));
        const listPage = vi.fn().mockRejectedValue(transportError('reset'));
        MockedS3Client.mockImplementationOnce(function () {
          return { getObject, listPage };
        });
        const client = await createTestArtifactStore(
          { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
          'minio',
          JSON.stringify({
            Provider: 'minio',
            Params: { fromEnv: 'true', maxRetries: '3' },
          }),
        );

        await expect(client.openRead('key')).rejects.toThrow('missing');
        const fallback = expect(client.list('key/')[Symbol.asyncIterator]().next()).rejects.toThrow(
          'reset',
        );
        await vi.runAllTimersAsync();
        await fallback;

        expect(getObject).toHaveBeenCalledTimes(1);
        expect(listPage).toHaveBeenCalledTimes(3);
      } finally {
        vi.useRealTimers();
      }
    });

    it('preserves method instrumentation when adding retry behavior', async () => {
      const stream = new PassThrough();
      stream.end('artifact');
      const getObject = vi.fn().mockResolvedValue(stream);
      MockedS3Client.mockImplementationOnce(function () {
        return { getObject, listPage: vi.fn() };
      });
      const client = await createTestArtifactStore(
        { accessKey: 'accesskey', endPoint: 'store.example', secretKey: 'secretkey' },
        'minio',
      );

      await expect(client.openRead('key')).resolves.toBe(stream);
      expect(getObject).toHaveBeenCalledTimes(1);
      expect(getObject).toHaveBeenCalledWith('bucket', 'key');
    });

    it('keeps retry classification local to concurrent operations', async () => {
      const context = TEST_ONLY.createS3RetryContext(2);
      const attemptStorage = (
        context as unknown as {
          attemptStorage: { getStore: () => { transportStatus?: number } | undefined };
        }
      ).attemptStorage;
      let releaseFirst!: () => void;
      const firstGate = new Promise<void>((resolve) => (releaseFirst = resolve));
      let firstStarted!: () => void;
      const firstStartedPromise = new Promise<void>((resolve) => (firstStarted = resolve));
      const firstOperation = vi.fn(async () => {
        if (firstOperation.mock.calls.length === 1) {
          attemptStorage.getStore()!.transportStatus = 500;
          firstStarted();
          await firstGate;
          throw new Error('server error');
        }
        return 'recovered';
      });
      const secondOperation = vi.fn(async () => {
        attemptStorage.getStore()!.transportStatus = 400;
        throw Object.assign(new Error('denied'), { code: 'AccessDenied' });
      });

      const firstRequest = TEST_ONLY.retryS3Operation(
        firstOperation,
        context,
        async () => undefined,
      );
      await firstStartedPromise;
      await expect(
        TEST_ONLY.retryS3Operation(secondOperation, context, async () => undefined),
      ).rejects.toThrow('denied');
      releaseFirst();

      await expect(firstRequest).resolves.toBe('recovered');
      expect(firstOperation).toHaveBeenCalledTimes(2);
      expect(secondOperation).toHaveBeenCalledTimes(1);
    });

    it('enforces the request-wide ten-retry ceiling across operations', async () => {
      const context = TEST_ONLY.createS3RetryContext(10);
      const firstOperation = vi.fn().mockRejectedValue(transportError('first reset'));
      const secondOperation = vi.fn().mockRejectedValue(transportError('second reset'));
      const thirdOperation = vi.fn().mockRejectedValue(transportError('third reset'));

      await expect(
        TEST_ONLY.retryS3Operation(firstOperation, context, async () => undefined),
      ).rejects.toThrow('first reset');
      await expect(
        TEST_ONLY.retryS3Operation(secondOperation, context, async () => undefined),
      ).rejects.toThrow('second reset');
      await expect(
        TEST_ONLY.retryS3Operation(thirdOperation, context, async () => undefined),
      ).rejects.toThrow('third reset');

      expect(firstOperation).toHaveBeenCalledTimes(10);
      expect(secondOperation).toHaveBeenCalledTimes(2);
      expect(thirdOperation).toHaveBeenCalledTimes(1);
    });

    it('does not charge successful child reads against the retry ceiling', async () => {
      const context = TEST_ONLY.createS3RetryContext(10);
      const operation = vi.fn().mockResolvedValue('object');

      for (let index = 0; index < 20; index++) {
        await expect(TEST_ONLY.retryS3Operation(operation, context)).resolves.toBe('object');
      }
      expect(operation).toHaveBeenCalledTimes(20);
    });

    it('stops retrying when the artifact HTTP request is aborted', async () => {
      const abortController = new AbortController();
      const context = TEST_ONLY.createS3RetryContext(3, abortController.signal);
      const operation = vi.fn().mockRejectedValue(transportError('reset'));

      await expect(
        TEST_ONLY.retryS3Operation(operation, context, async () => abortController.abort()),
      ).rejects.toMatchObject({ name: 'AbortError' });
      expect(operation).toHaveBeenCalledTimes(1);
    });

    it('passes request cancellation through a configured MinIO client', async () => {
      const abortController = new AbortController();
      const getObject = vi.fn().mockRejectedValue(transportError('reset'));
      MockedS3Client.mockImplementationOnce(function () {
        return { getObject, listPage: vi.fn() };
      });
      const client = await createTestArtifactStore(
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

      const request = expect(client.openRead('key')).rejects.toMatchObject({
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
    let store: ArtifactStore;
    let mockedOpenRead: Mock;

    beforeEach(() => {
      vi.clearAllMocks();
      mockedOpenRead = vi.fn();
      store = { openRead: mockedOpenRead, list: async function* () {} };
    });

    it('unpacks a gzipped tarball', async () => {
      const objStream = new PassThrough();
      objStream.end(tarGzBuffer);
      mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ key: 'key', store: store });
      expect(mockedOpenRead).toBeCalledWith('key', { signal: undefined });
      const chunks: Buffer[] = [];
      for await (const chunk of stream) {
        chunks.push(Buffer.from(chunk));
      }
      expect(Buffer.concat(chunks).toString().trim()).toBe('hello world');
    });

    it('unpacks a uncompressed tarball', async () => {
      const objStream = new PassThrough();
      objStream.end(tarBuffer);
      mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));

      const onTransformationDetermined = vi.fn();
      const stream = await getObjectStream({
        key: 'key',
        store: store,
        onTransformationDetermined,
      });
      expect(mockedOpenRead).toBeCalledWith('key', { signal: undefined });
      const chunks: Buffer[] = [];
      for await (const chunk of stream) {
        chunks.push(Buffer.from(chunk));
      }
      expect(Buffer.concat(chunks).toString().trim()).toBe('hello world');
      expect(onTransformationDetermined).toHaveBeenCalledWith(true);
    });

    it('reports decompression even when the decompressed object is not a tarball', async () => {
      const objStream = new PassThrough();
      objStream.end(zlib.gzipSync(Buffer.from('hello world')));
      mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));
      const onTransformationDetermined = vi.fn();

      const stream = await getObjectStream({
        key: 'key',
        store: store,
        onTransformationDetermined,
      });
      const chunks: Buffer[] = [];
      stream.on('data', (chunk) => chunks.push(Buffer.from(chunk)));
      await new Promise<void>((resolve, reject) => {
        stream.once('end', resolve);
        stream.once('error', reject);
      });

      expect(Buffer.concat(chunks).toString()).toBe('hello world');
      expect(onTransformationDetermined).toHaveBeenCalledWith(true);
    });

    it.each([0, 6, 9])(
      'keeps transformation detection aligned with gunzip-maybe for deflate level %i',
      async (level) => {
        const objStream = new PassThrough();
        objStream.end(zlib.deflateSync(Buffer.from('hello world'), { level }));
        mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));
        const onTransformationDetermined = vi.fn();

        const stream = await getObjectStream({
          key: 'key',
          store: store,
          onTransformationDetermined,
        });
        const chunks: Buffer[] = [];
        for await (const chunk of stream) {
          chunks.push(Buffer.from(chunk));
        }

        expect(Buffer.concat(chunks).toString()).toBe('hello world');
        expect(onTransformationDetermined).toHaveBeenCalledWith(true);
      },
    );

    it('returns the content as a stream', async () => {
      const objStream = new PassThrough();
      objStream.end('hello world');
      mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));

      const stream = await getObjectStream({ key: 'key', store: store });
      expect(mockedOpenRead).toBeCalledWith('key', { signal: undefined });
      const chunks: Buffer[] = [];
      for await (const chunk of stream) {
        chunks.push(Buffer.from(chunk));
      }
      expect(Buffer.concat(chunks).toString().trim()).toBe('hello world');
    });

    it('forwards errors from the underlying object stream', async () => {
      const objStream = new PassThrough();
      mockedOpenRead.mockResolvedValueOnce(Promise.resolve(objStream));
      const stream = await getObjectStream({
        key: 'key',
        store: store,
        tryExtract: false,
      });
      const streamError = new Error('storage connection reset');
      const receivedError = new Promise<Error>((resolve) => stream.once('error', resolve));

      objStream.write('partial artifact');
      objStream.destroy(streamError);

      await expect(receivedError).resolves.toBe(streamError);
    });

    it('installs the supplied error handler before the source can fail', async () => {
      const streamError = new Error('immediate storage failure');
      mockedOpenRead.mockImplementationOnce(async () => {
        const objStream = new PassThrough();
        setImmediate(() => objStream.destroy(streamError));
        return objStream;
      });
      const onError = vi.fn();

      await getObjectStream({
        key: 'key',
        store: store,
        onError,
      });
      await new Promise<void>((resolve) => setImmediate(resolve));

      expect(onError).toHaveBeenCalledWith(streamError);
    });

    it('destroys the upstream object stream when the request is aborted', async () => {
      const abortController = new AbortController();
      const objStream = new PassThrough();
      const destroy = vi.spyOn(objStream, 'destroy');
      mockedOpenRead.mockResolvedValueOnce(objStream);

      const onError = vi.fn();
      const output = await getObjectStream({
        store: store,
        key: 'key',
        signal: abortController.signal,
        onError,
      });
      const closed = new Promise<void>((resolve) => output.once('close', resolve));
      abortController.abort();

      expect(destroy).toHaveBeenCalledWith();
      await closed;
      expect(output.destroyed).toBe(true);
      expect(onError).toHaveBeenCalledTimes(1);
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
  // describe SDK responses while the adapter owns pagination and normalization.
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
      const fixture = {
        listPage: vi.fn(async (): Promise<Page> => ({
          objects: [
            { name: 'a.txt', size: 10 },
            { name: 'b.txt', size: 20 },
          ],
          isTruncated: false,
          nextContinuationToken: '',
        })),
      };
      const client = await storeForFixture(fixture);

      const results = await collect(listObjectsUnderPrefix(client, 'p/'));
      expect(results).toEqual([
        { name: 'a.txt', size: 10 },
        { name: 'b.txt', size: 20 },
      ]);
      expect(fixture.listPage).toHaveBeenCalledTimes(1);
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
      const fixture = {
        listPage: vi.fn(
          async (_bucket: string, _prefix: string, continuationToken: string): Promise<Page> => {
            seenTokens.push(continuationToken);
            return pagesByToken[continuationToken];
          },
        ),
      };
      const client = await storeForFixture(fixture);

      const results = await collect(listObjectsUnderPrefix(client, 'p/'));
      expect(results.map((r) => r.name)).toEqual(['page1-a', 'page1-b', 'page2-a', 'page3-a']);
      // Pages are visited in order, with each call's token coming from the
      // previous response.
      expect(seenTokens).toEqual(['', 'tok-1', 'tok-2']);
    });

    it('defaults missing size to 0 and skips entries without a name', async () => {
      const fixture = {
        listPage: vi.fn(async (): Promise<Page> => ({
          objects: [{ name: 'has-size', size: 42 }, { name: 'no-size' }, { size: 99 }],
          isTruncated: false,
          nextContinuationToken: '',
        })),
      };
      const client = await storeForFixture(fixture);

      const results = await collect(listObjectsUnderPrefix(client, 'p/'));
      expect(results).toEqual([
        { name: 'has-size', size: 42 },
        { name: 'no-size', size: 0 },
      ]);
    });

    it('rejects a truncated listing without a continuation token', async () => {
      const client = await storeForFixture({
        listPage: vi
          .fn()
          .mockResolvedValue({ objects: [], isTruncated: true, nextContinuationToken: '' }),
      });
      const iter = listObjectsUnderPrefix(client, 'p/');
      await expect(iter.next()).rejects.toThrow(/continuation token/);
    });

    it('stops waiting for a paginated listing when its signal is aborted', async () => {
      const controller = new AbortController();
      const listPage = vi
        .fn()
        .mockResolvedValueOnce({
          objects: [{ name: 'page1-a', size: 1 }],
          isTruncated: true,
          nextContinuationToken: 'tok-1',
        })
        .mockImplementationOnce(() => new Promise(() => undefined));
      const fixture = { listPage };
      const client = await storeForFixture(fixture);
      const iter = listObjectsUnderPrefix(client, 'p/', controller.signal);

      await expect(iter.next()).resolves.toEqual({
        done: false,
        value: { name: 'page1-a', size: 1 },
      });
      const next = iter.next();
      await vi.waitFor(() => expect(listPage).toHaveBeenCalledTimes(2));
      controller.abort(new Error('response closed'));

      await expect(next).rejects.toMatchObject({ name: 'AbortError' });
    });
  });

  // summarizeDirectoryUnderPrefix backs the bounded directory preview path.
  // It must not paginate — large directories should still cost one round
  // trip, with `truncated: true` signalling there's more.
  describe('summarizeDirectoryUnderPrefix', () => {
    it('returns count and truncated=false for a small, complete listing', async () => {
      const listPage = vi.fn(async () => ({
        objects: [
          { name: 'a', size: 1 },
          { name: 'b', size: 2 },
          { name: 'c', size: 3 },
        ],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const fixture = { listPage };
      const client = await storeForFixture(fixture);

      const summary = await summarizeDirectoryUnderPrefix(client, 'p/');
      expect(summary).toEqual({ count: 3, truncated: false });
      expect(listPage).toHaveBeenCalledTimes(1);
    });

    it('returns truncated=true when the listing exceeds the preview cap', async () => {
      const listPage = vi.fn(async () => ({
        objects: Array.from({ length: 51 }, (_, i) => ({ name: `f-${i}`, size: 1 })),
        isTruncated: true,
        nextContinuationToken: 'next',
      }));
      const fixture = { listPage };
      const client = await storeForFixture(fixture);

      const summary = await summarizeDirectoryUnderPrefix(client, 'p/');
      expect(summary).toEqual({ count: 50, truncated: true });
      // Bounded — does not loop on the continuation token.
      expect(listPage).toHaveBeenCalledTimes(1);
    });

    it('returns null for an empty prefix so callers can answer 404', async () => {
      const listPage = vi.fn(async () => ({
        objects: [],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const fixture = { listPage };
      const client = await storeForFixture(fixture);

      const summary = await summarizeDirectoryUnderPrefix(client, 'p/');
      expect(summary).toBeNull();
    });

    it('passes the configured maxKeys cap to listPage', async () => {
      const listPage = vi.fn(async () => ({
        objects: [{ name: 'a', size: 1 }],
        isTruncated: false,
        nextContinuationToken: '',
      }));
      const fixture = { listPage };
      const client = await storeForFixture(fixture);

      await summarizeDirectoryUnderPrefix(client, 'p/', 25);
      expect(listPage).toHaveBeenCalledWith('bucket', 'p/', '', '', 26);
    });
  });
});
