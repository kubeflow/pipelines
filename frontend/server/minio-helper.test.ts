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
} from './minio-helper.js';
import { fromNodeProviderChain } from '@aws-sdk/credential-providers';

vi.mock('minio');
vi.mock('@aws-sdk/credential-providers');
vi.mock('./k8s-helper.js', () => ({ getK8sSecret: vi.fn() }));

describe('minio-helper', () => {
  const MockedMinioClient: Mock = MinioClient as any;
  const MockedAuthorizeFn: Mock = vi.fn((x) => undefined);

  beforeEach(() => {
    vi.resetAllMocks();
  });

  describe('createMinioClient', () => {
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

    it('fallbacks to the provided configs if EC2 metadata is not available.', async () => {
      const client = await createMinioClient(
        {
          endPoint: 'minio.kubeflow:80',
        },
        's3',
      );

      expect(client).toBeInstanceOf(MinioClient);
      expect(MockedMinioClient).toHaveBeenCalledWith({
        endPoint: 'minio.kubeflow:80',
      });
    });

    it('applies endpoint and region settings when credentials come from the environment', async () => {
      await createMinioClient(
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

    it('lets disable_https override an explicit HTTPS endpoint scheme', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            disable_https: 'true',
            endpoint: 'https://ceph.example:9000',
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

    it('lets disable_https override an explicit HTTPS AWS endpoint scheme', async () => {
      await createMinioClient(
        { accessKey: 'accesskey', endPoint: 'default-store', secretKey: 'secretkey' },
        's3',
        JSON.stringify({
          Provider: 's3',
          Params: {
            disable_https: 'true',
            endpoint: 'https://s3.us-west-2.amazonaws.com',
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

    it('preserves endpoint base paths in the path MinIO signs and requests', async () => {
      const getRequestOptions = vi.fn(() => ({ path: '/bucket/object?versionId=1' }));
      MockedMinioClient.mockImplementationOnce(function () {
        return { getRequestOptions };
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
