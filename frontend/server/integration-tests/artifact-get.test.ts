// Copyright 2020 The Kubeflow Authors
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

import { vi, describe, it, expect, afterAll, afterEach, beforeEach, Mock } from 'vitest';
import * as fs from 'fs';
import * as minio from 'minio';
import * as path from 'path';
import * as tar from 'tar-stream';
import * as zlib from 'zlib';
import { PassThrough, Readable } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import * as serverInfo from '../helpers/server-info.js';
import { commonSetup, mkTempDir } from './test-helper.js';
import { getK8sSecret } from '../k8s-helper.js';
import { downloadGCSObjectStream, getGCSClient, listGCSObjectNames } from '../gcs-helper.js';

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('../k8s-helper');
vi.mock('../gcs-helper.js');

const mockedFetch = vi.fn();
vi.stubGlobal('fetch', mockedFetch);

/** Create a web ReadableStream from a string, matching what global fetch returns. */
function toWebStream(content: string): ReadableStream<Uint8Array> {
  const pt = new PassThrough();
  pt.end(content);
  return Readable.toWeb(pt) as ReadableStream<Uint8Array>;
}

describe('/artifacts', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  let artifactContent: any = 'hello world';
  beforeEach(() => {
    artifactContent = 'hello world'; // reset
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

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  describe('/get', () => {
    it('responds with a minio artifact if source=minio', async () => {
      const mockedMinioClient: Mock = minio.Client as any;

      const configs = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'seaweedfs',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'minio',
        endPoint: 'seaweedfs.kubeflow',
        port: 9000,
        secretKey: 'minio123',
        useSSL: false,
      });
    });

    it('responds with artifact if source is AWS S3, and creds are sourced from Env', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      process.env.AWS_ACCESS_KEY_ID = 'aws123';
      process.env.AWS_SECRET_ACCESS_KEY = 'awsSecret123';
      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'aws123',
        endPoint: 's3.amazonaws.com',
        region: 'us-east-1',
        secretKey: 'awsSecret123',
        useSSL: true,
      });
    });

    it('responds with artifact if source is AWS S3, and creds are sourced from Load Configs', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
      });
      app = new UIServer(configs);
      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'aws123',
        endPoint: 's3.amazonaws.com',
        region: 'us-east-1',
        secretKey: 'awsSecret123',
        useSSL: true,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
    });

    it('responds with artifact if source is AWS S3, and creds are sourced from Provider Configs', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('someSecret');
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          accessKeyKey: 'someSecret',
          // this not set and default is used (tls=true)
          // since aws connections are always tls secured
          disableSSL: 'false',
          endpoint: 's3.amazonaws.com',
          fromEnv: 'false',
          // this not set and default is used
          // since aws connections always have the same port
          port: '0001',
          region: 'us-east-2',
          secretKeyKey: 'someSecret',
          secretName: 'aws-s3-creds',
        },
        Provider: 's3',
      };
      const namespace = 'test';
      await request
        .get(
          `/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=${namespace}&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'someSecret',
        endPoint: 's3.amazonaws.com',
        port: undefined,
        region: 'us-east-2',
        secretKey: 'someSecret',
        useSSL: undefined,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
      expect(mockedGetK8sSecret).toBeCalledWith('aws-s3-creds', 'someSecret', `${namespace}`);
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
    });

    it('responds with artifact if source is AWS S3, and creds are sourced from Provider Configs, and uses default kubeflow namespace when no namespace is provided', async () => {
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('somevalue');
      const mockedMinioClient: Mock = minio.Client as any;
      const namespace = 'kubeflow';
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          accessKeyKey: 'AWS_ACCESS_KEY_ID',
          disableSSL: 'false',
          endpoint: 's3.amazonaws.com',
          fromEnv: 'false',
          region: 'us-east-2',
          secretKeyKey: 'AWS_SECRET_ACCESS_KEY',
          secretName: 'aws-s3-creds',
        },
        Provider: 's3',
      };
      await request
        .get(
          `/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'somevalue',
        endPoint: 's3.amazonaws.com',
        port: undefined,
        region: 'us-east-2',
        secretKey: 'somevalue',
        useSSL: undefined,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        1,
        'aws-s3-creds',
        'AWS_ACCESS_KEY_ID',
        `${namespace}`,
      );
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        2,
        'aws-s3-creds',
        'AWS_SECRET_ACCESS_KEY',
        `${namespace}`,
      );
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
    });

    it('responds with artifact if source is AWS S3, and creds are sourced from Provider Configs, and uses default namespace when no namespace is provided, as specified in ENV', async () => {
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('somevalue');
      const mockedMinioClient: Mock = minio.Client as any;
      const namespace = 'notkubeflow';
      const configs = loadConfigs(argv, { FRONTEND_SERVER_NAMESPACE: namespace });
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          accessKeyKey: 'AWS_ACCESS_KEY_ID',
          disableSSL: 'false',
          endpoint: 's3.amazonaws.com',
          fromEnv: 'false',
          region: 'us-east-2',
          secretKeyKey: 'AWS_SECRET_ACCESS_KEY',
          secretName: 'aws-s3-creds',
        },
        Provider: 's3',
      };
      await request
        .get(
          `/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'somevalue',
        endPoint: 's3.amazonaws.com',
        port: undefined,
        region: 'us-east-2',
        secretKey: 'somevalue',
        useSSL: undefined,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        1,
        'aws-s3-creds',
        'AWS_ACCESS_KEY_ID',
        `${namespace}`,
      );
      expect(mockedGetK8sSecret).toHaveBeenNthCalledWith(
        2,
        'aws-s3-creds',
        'AWS_SECRET_ACCESS_KEY',
        `${namespace}`,
      );
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
    });

    it('responds with artifact if source is s3-compatible, and creds are sourced from Provider Configs', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('someSecret');
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          accessKeyKey: 'someSecret',
          disableSSL: 'false',
          endpoint: 'https://mys3.com',
          fromEnv: 'false',
          region: 'auto',
          secretKeyKey: 'someSecret',
          secretName: 'my-secret',
        },
        Provider: 's3',
      };
      const namespace = 'test';
      await request
        .get(
          `/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=${namespace}&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'someSecret',
        endPoint: 'mys3.com',
        port: undefined,
        region: 'auto',
        secretKey: 'someSecret',
        useSSL: true,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
      expect(mockedGetK8sSecret).toBeCalledWith('my-secret', 'someSecret', `${namespace}`);
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
    });

    it('responds with artifact if source is s3-compatible, and creds are sourced from Provider Configs, with endpoint port', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('someSecret');
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          accessKeyKey: 'someSecret',
          disableSSL: 'false',
          endpoint: 'https://mys3.ns.svc.cluster.local:1234',
          fromEnv: 'false',
          region: 'auto',
          secretKeyKey: 'someSecret',
          secretName: 'my-secret',
        },
        Provider: 's3',
      };
      const namespace = 'test';
      await request
        .get(
          `/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=${namespace}&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'someSecret',
        endPoint: 'mys3.ns.svc.cluster.local',
        port: 1234,
        region: 'auto',
        secretKey: 'someSecret',
        useSSL: true,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
      expect(mockedGetK8sSecret).toBeCalledWith('my-secret', 'someSecret', `${namespace}`);
      expect(mockedGetK8sSecret).toBeCalledTimes(2);
    });

    it('responds with artifact if source is gcs, and creds are sourced from Provider Configs', async () => {
      const artifactContent = 'hello world';
      const mockedGetGCSClient: Mock = getGCSClient as any;
      const mockedListGCSObjectNames: Mock = listGCSObjectNames as any;
      const mockedDownloadGCSObjectStream: Mock = downloadGCSObjectStream as any;
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      const client = { request: vi.fn() };
      mockedGetK8sSecret.mockResolvedValue('{"private_key":"testkey","client_email":"testemail"}');
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();
      mockedGetGCSClient.mockResolvedValueOnce(client);
      mockedListGCSObjectNames.mockResolvedValueOnce(['hello/world.txt']);
      mockedDownloadGCSObjectStream.mockResolvedValueOnce(stream);
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const request = requests(app.app);
      const providerInfo = {
        Params: {
          fromEnv: 'false',
          secretName: 'someSecret',
          tokenKey: 'somekey',
        },
        Provider: 'gs',
      };
      const namespace = 'test';
      await request
        .get(
          `/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=${namespace}&providerInfo=${JSON.stringify(
            providerInfo,
          )}`,
        )
        .expect(200, artifactContent + '\n');
      const expectedArg = {
        bucket: 'ml-pipeline',
        client,
        credentials: {
          client_email: 'testemail',
          private_key: 'testkey',
        },
        prefix: 'hello/world.txt',
      };
      expect(mockedListGCSObjectNames).toBeCalledWith(expectedArg);
      expect(mockedDownloadGCSObjectStream).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: {
          client_email: 'testemail',
          private_key: 'testkey',
        },
        objectName: 'hello/world.txt',
      });
      expect(mockedGetGCSClient).toBeCalledWith({
        client_email: 'testemail',
        private_key: 'testkey',
      });
      expect(mockedGetK8sSecret).toBeCalledWith('someSecret', 'somekey', `${namespace}`);
      expect(mockedGetK8sSecret).toBeCalledTimes(1);
    });

    it('responds with partial s3 artifact if peek=5 flag is set', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5));
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'aws123',
        endPoint: 's3.amazonaws.com',
        region: 'us-east-1',
        secretKey: 'awsSecret123',
        useSSL: true,
      });
      expect(mockedMinioClient).toBeCalledTimes(1);
    });

    it('responds with a s3 artifact from bucket in non-default region if source=s3', async () => {
      const mockedMinioClient: Mock = minio.Client as any;
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
        AWS_REGION: 'eu-central-1',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedMinioClient).toBeCalledWith({
        accessKey: 'aws123',
        endPoint: 's3.amazonaws.com',
        region: 'eu-central-1',
        secretKey: 'awsSecret123',
        useSSL: true,
      });
    });

    it('responds with a http artifact if source=http', async () => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'http://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: toWebStream(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=http&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedFetch).toBeCalledWith('http://foo.bar/ml-pipeline/hello/world.txt', {
        headers: {},
      });
    });

    it('responds with partial http artifact if peek=5 flag is set', async () => {
      const artifactContent = 'hello world';
      const mockedFetch: Mock = fetch as any;
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'http://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: toWebStream(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=http&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5));
      expect(mockedFetch).toBeCalledWith('http://foo.bar/ml-pipeline/hello/world.txt', {
        headers: {},
      });
    });

    it('responds with a https artifact if source=https', async () => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt' &&
        opts.headers.Authorization === 'someToken'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: toWebStream(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_AUTHORIZATION_DEFAULT_VALUE: 'someToken',
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=https&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent);
      expect(mockedFetch).toBeCalledWith('https://foo.bar/ml-pipeline/hello/world.txt', {
        headers: {
          Authorization: 'someToken',
        },
      });
    });

    it('responds with a https artifact using the inherited header if source=https and http authorization key is provided.', async () => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: toWebStream(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=https&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .set('Authorization', 'inheritedToken')
        .expect(200, artifactContent);
      expect(mockedFetch).toBeCalledWith('https://foo.bar/ml-pipeline/hello/world.txt', {
        headers: {
          Authorization: 'inheritedToken',
        },
      });
    });

    it('responds with a gcs artifact if source=gcs', async () => {
      const artifactContent = 'hello world';
      const mockedGetGCSClient: Mock = getGCSClient as any;
      const mockedListGCSObjectNames: Mock = listGCSObjectNames as any;
      const mockedDownloadGCSObjectStream: Mock = downloadGCSObjectStream as any;
      const client = { request: vi.fn() };
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();
      mockedGetGCSClient.mockResolvedValueOnce(client);
      mockedListGCSObjectNames.mockResolvedValueOnce(['hello/world.txt']);
      mockedDownloadGCSObjectStream.mockResolvedValueOnce(stream);
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent + '\n');
      expect(mockedGetGCSClient).toBeCalledWith(undefined);
      expect(mockedListGCSObjectNames).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        prefix: 'hello/world.txt',
      });
      expect(mockedDownloadGCSObjectStream).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        objectName: 'hello/world.txt',
      });
    });

    it('responds with a partial gcs artifact if peek=5 is set', async () => {
      const artifactContent = 'hello world';
      const mockedGetGCSClient: Mock = getGCSClient as any;
      const mockedListGCSObjectNames: Mock = listGCSObjectNames as any;
      const mockedDownloadGCSObjectStream: Mock = downloadGCSObjectStream as any;
      const client = { request: vi.fn() };
      const stream = new PassThrough();
      stream.end(artifactContent);
      mockedGetGCSClient.mockResolvedValueOnce(client);
      mockedListGCSObjectNames.mockResolvedValueOnce(['hello/world.txt']);
      mockedDownloadGCSObjectStream.mockResolvedValueOnce(stream);
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5));
      expect(mockedGetGCSClient).toBeCalledWith(undefined);
      expect(mockedListGCSObjectNames).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        prefix: 'hello/world.txt',
      });
      expect(mockedDownloadGCSObjectStream).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        objectName: 'hello/world.txt',
      });
    });

    it('responds with concatenated gcs artifacts for wildcard keys and reuses one auth client', async () => {
      const mockedGetGCSClient: Mock = getGCSClient as any;
      const mockedListGCSObjectNames: Mock = listGCSObjectNames as any;
      const mockedDownloadGCSObjectStream: Mock = downloadGCSObjectStream as any;
      const client = { request: vi.fn() };
      const streamA = new PassThrough();
      const streamB = new PassThrough();
      streamA.end('hello');
      streamB.end('world');
      mockedGetGCSClient.mockResolvedValueOnce(client);
      mockedListGCSObjectNames.mockResolvedValueOnce([
        'hello/world-1.txt',
        'hello/world-2.txt',
        'hello/not-a-match.log',
      ]);
      mockedDownloadGCSObjectStream.mockResolvedValueOnce(streamA);
      mockedDownloadGCSObjectStream.mockResolvedValueOnce(streamB);
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld-*.txt')
        .expect(200, 'hello\nworld\n');
      expect(mockedGetGCSClient).toBeCalledTimes(1);
      expect(mockedListGCSObjectNames).toBeCalledWith({
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        prefix: 'hello/world-',
      });
      expect(mockedDownloadGCSObjectStream).toHaveBeenNthCalledWith(1, {
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        objectName: 'hello/world-1.txt',
      });
      expect(mockedDownloadGCSObjectStream).toHaveBeenNthCalledWith(2, {
        bucket: 'ml-pipeline',
        client,
        credentials: undefined,
        objectName: 'hello/world-2.txt',
      });
    });

    it('responds with a volume artifact if source=volume', async () => {
      const artifactContent = 'hello world';
      const tempPath = path.join(mkTempDir(), 'content');
      fs.writeFileSync(tempPath, artifactContent);

      vi.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
        Promise.resolve([
          {
            spec: {
              containers: [
                {
                  volumeMounts: [
                    {
                      name: 'artifact',
                      mountPath: path.dirname(tempPath),
                      subPath: 'subartifact',
                    },
                  ],
                  name: 'ml-pipeline-ui',
                },
              ],
              volumes: [
                {
                  name: 'artifact',
                  persistentVolumeClaim: {
                    claimName: 'artifact_pvc',
                  },
                },
              ],
            },
          } as any,
          undefined,
        ]),
      );

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=volume&bucket=artifact&key=subartifact/content')
        .expect(200, artifactContent);
    });

    it('responds with a partial volume artifact if peek=5 is set', async () => {
      const artifactContent = 'hello world';
      const tempPath = path.join(mkTempDir(), 'content');
      fs.writeFileSync(tempPath, artifactContent);

      vi.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
        Promise.resolve([
          {
            spec: {
              containers: [
                {
                  volumeMounts: [
                    {
                      name: 'artifact',
                      mountPath: path.dirname(tempPath),
                    },
                  ],
                  name: 'ml-pipeline-ui',
                },
              ],
              volumes: [
                {
                  name: 'artifact',
                  persistentVolumeClaim: {
                    claimName: 'artifact_pvc',
                  },
                },
              ],
            },
          } as any,
          undefined,
        ]),
      );

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=content&peek=5`)
        .expect(200, artifactContent.slice(0, 5));
    });

    it('responds error with a not exist volume', async () => {
      vi.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
        Promise.resolve([
          {
            metadata: {
              name: 'ml-pipeline-ui',
            },
            spec: {
              containers: [
                {
                  volumeMounts: [
                    {
                      name: 'artifact',
                      mountPath: '/foo/bar/path',
                    },
                  ],
                  name: 'ml-pipeline-ui',
                },
              ],
              volumes: [
                {
                  name: 'artifact',
                  persistentVolumeClaim: {
                    claimName: 'artifact_pvc',
                  },
                },
              ],
            },
          } as any,
          undefined,
        ]),
      );

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get(`/artifacts/get?source=volume&bucket=notexist&key=content`)
        .expect(404, 'Failed to open volume.');
    });

    it('responds error with a not exist volume mount path if source=volume', async () => {
      vi.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
        Promise.resolve([
          {
            metadata: {
              name: 'ml-pipeline-ui',
            },
            spec: {
              containers: [
                {
                  volumeMounts: [
                    {
                      name: 'artifact',
                      mountPath: '/foo/bar/path',
                      subPath: 'subartifact',
                    },
                  ],
                  name: 'ml-pipeline-ui',
                },
              ],
              volumes: [
                {
                  name: 'artifact',
                  persistentVolumeClaim: {
                    claimName: 'artifact_pvc',
                  },
                },
              ],
            },
          } as any,
          undefined,
        ]),
      );

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=notexist/config`)
        .expect(404, 'Failed to open volume.');
    });

    it('responds error with a not exist volume mount artifact if source=volume', async () => {
      vi.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
        Promise.resolve([
          {
            spec: {
              containers: [
                {
                  volumeMounts: [
                    {
                      name: 'artifact',
                      mountPath: '/foo/bar',
                      subPath: 'subartifact',
                    },
                  ],
                  name: 'ml-pipeline-ui',
                },
              ],
              volumes: [
                {
                  name: 'artifact',
                  persistentVolumeClaim: {
                    claimName: 'artifact_pvc',
                  },
                },
              ],
            },
          } as any,
          undefined,
        ]),
      );

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=subartifact/notxist.csv`)
        .expect(500, 'Failed to open volume.');
    });

    it('rejects keys longer than 1024 characters', async () => {
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
      });
      app = new UIServer(configs);
      const request = requests(app.start());
      await request
        .get(
          '/artifacts/get?source=s3&namespace=test&peek=256&bucket=ml-pipeline&key=' +
            'a'.repeat(1025),
        )
        .expect(500, 'Object key too long');
    });

    // KFP v2 stores some output artifacts as object-store directories
    // (prefixes) instead of single objects. When the user clicks download on
    // one, getObject fails with NoSuchKey and the handler falls back to
    // packaging the contents of the prefix as a .tar.gz. See
    // https://github.com/kubeflow/pipelines/issues/7809.
    describe('directory artifact fallback', () => {
      const minioConfigEnv = {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'seaweedfs',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      };

      function makeNoSuchKeyError(): Error {
        return Object.assign(new Error('The specified key does not exist.'), {
          code: 'NoSuchKey',
        });
      }

      function mockMinioForDirectory(fileContents: Record<string, string>) {
        const mockedMinioClient = minio.Client as any;
        // The production code uses `new MinioClient(...)`, so the mock must
        // be constructable. Use a `function` expression rather than an arrow
        // — vitest warns otherwise and the constructed instance ends up
        // missing the methods we attach below.
        mockedMinioClient.mockImplementation(function () {
          return {
            getObject: async (bucket: string, key: string) => {
              if (bucket !== 'ml-pipeline') {
                throw new Error(`unexpected bucket ${bucket}`);
              }
              if (key in fileContents) {
                const objStream = new PassThrough();
                objStream.end(fileContents[key]);
                return objStream;
              }
              throw makeNoSuchKeyError();
            },
            // minio@8.x exposes listObjectsV2Query as an async method that
            // resolves to the parsed page; mirror that here so the mock
            // matches the real client's runtime shape.
            listObjectsV2Query: async (_bucket: string, prefix: string) => ({
              objects: Object.entries(fileContents)
                .filter(([key]) => key.startsWith(prefix))
                .map(([name, content]) => ({ name, size: content.length })),
              isTruncated: false,
              nextContinuationToken: '',
            }),
          };
        });
      }

      async function readTarGzEntries(buffer: Buffer): Promise<Map<string, Buffer>> {
        const extract = tar.extract();
        const entries = new Map<string, Buffer>();
        return new Promise((resolve, reject) => {
          extract.on('entry', (header, stream, next) => {
            const chunks: Uint8Array[] = [];
            stream.on('data', (chunk: Uint8Array) => chunks.push(chunk));
            stream.on('end', () => {
              entries.set(header.name, Buffer.concat(chunks));
              next();
            });
            stream.on('error', reject);
            stream.resume();
          });
          extract.on('finish', () => resolve(entries));
          extract.on('error', reject);
          // Write the gunzipped tar bytes directly into the extract stream
          // rather than piping through a PassThrough — newer @types/node and
          // older @types/tar-stream disagree on the WritableStream shape, and
          // pipe() trips that mismatch.
          extract.end(zlib.gunzipSync(buffer));
        });
      }

      function captureBinaryResponse(req: requests.Test): requests.Test {
        return req.buffer(true).parse((response: any, callback: any) => {
          const chunks: Uint8Array[] = [];
          response.on('data', (chunk: Uint8Array) => chunks.push(chunk));
          response.on('end', () => callback(null, Buffer.concat(chunks)));
          response.on('error', callback);
        });
      }

      it('packages the prefix as a tar.gz when getObject returns NoSuchKey', async () => {
        mockMinioForDirectory({
          'directory/file1.txt': 'first file contents',
          'directory/sub/file2.txt': 'second file contents',
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        const res = await captureBinaryResponse(
          request.get('/artifacts/get?source=minio&bucket=ml-pipeline&key=directory'),
        ).expect(200);

        expect(res.headers['content-type']).toBe('application/gzip');
        expect(res.headers['content-disposition']).toContain('filename="directory.tar.gz"');

        const entries = await readTarGzEntries(res.body as Buffer);
        expect(Array.from(entries.keys()).sort()).toEqual(['file1.txt', 'sub/file2.txt']);
        expect(entries.get('file1.txt')!.toString()).toBe('first file contents');
        expect(entries.get('sub/file2.txt')!.toString()).toBe('second file contents');
      });

      it('returns a small text summary for preview requests instead of streaming the archive', async () => {
        // The run details panel calls Apis.readFile with peek=N to render a
        // small inline preview. Falling through to the tar fallback would
        // list every object under the prefix and stream the whole gzip just
        // to render a few KB. The preview path must instead answer with a
        // bounded summary: exactly one capped listObjectsV2Query call (no
        // pagination), no per-child getObject fetches, small text body.
        const listObjectsV2Query = vi.fn(
          async (
            _bucket: string,
            _prefix: string,
            _continuationToken: string,
            _delimiter: string,
            _maxKeys: number,
          ) => ({
            objects: Array.from({ length: 5 }, (_, i) => ({
              name: `some-directory/file-${i}.txt`,
              size: 10,
            })),
            isTruncated: false,
            nextContinuationToken: '',
          }),
        );
        const getObject = vi.fn(async (bucket: string, key: string) => {
          if (bucket === 'ml-pipeline' && key === 'some-directory') {
            throw makeNoSuchKeyError();
          }
          throw new Error(`unexpected getObject(${bucket}, ${key})`);
        });
        const mockedMinioClient = minio.Client as any;
        mockedMinioClient.mockImplementation(function () {
          return { getObject, listObjectsV2Query };
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        const res = await request
          .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=some-directory&peek=256')
          .expect(200);

        // Response is small text, not a gzip archive.
        expect(res.headers['content-type']).toMatch(/^text\/plain/);
        expect(res.headers['content-type']).not.toMatch(/gzip/);
        expect(res.text.length).toBeLessThan(512);
        expect(res.text).toContain('Directory artifact');
        expect(res.text).toContain('some-directory');
        expect(res.text).toContain('5');

        // Exactly one listing call; no pagination loop.
        expect(listObjectsV2Query).toHaveBeenCalledTimes(1);
        // Only the original single-object lookup was attempted; no per-file
        // fetches under the prefix.
        expect(getObject).toHaveBeenCalledTimes(1);
        expect(getObject).toHaveBeenCalledWith('ml-pipeline', 'some-directory');
      });

      it('marks the file count as truncated when minio reports more pages exist', async () => {
        // For very large directories the single capped list call only sees
        // the first page; surface that to the user as "N+" rather than a
        // misleading exact count.
        const listObjectsV2Query = vi.fn(async () => ({
          objects: Array.from({ length: 50 }, (_, i) => ({
            name: `huge-dir/file-${i}.txt`,
            size: 1,
          })),
          isTruncated: true,
          nextContinuationToken: 'next-page-token',
        }));
        const getObject = vi.fn(async () => {
          throw makeNoSuchKeyError();
        });
        const mockedMinioClient = minio.Client as any;
        mockedMinioClient.mockImplementation(function () {
          return { getObject, listObjectsV2Query };
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        const res = await request
          .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=huge-dir&peek=256')
          .expect(200);

        expect(res.text).toContain('50+');
        // Did not paginate even though more pages exist.
        expect(listObjectsV2Query).toHaveBeenCalledTimes(1);
      });

      it('returns 404 for preview requests on an empty prefix', async () => {
        const listObjectsV2Query = vi.fn(async () => ({
          objects: [],
          isTruncated: false,
          nextContinuationToken: '',
        }));
        const getObject = vi.fn(async () => {
          throw makeNoSuchKeyError();
        });
        const mockedMinioClient = minio.Client as any;
        mockedMinioClient.mockImplementation(function () {
          return { getObject, listObjectsV2Query };
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        await request
          .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=missing-dir&peek=256')
          .expect(404);
      });

      it('returns 404 when the prefix has no objects', async () => {
        mockMinioForDirectory({});

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        await request
          .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=missing-directory')
          .expect(404);
      });

      it('produces a safe Content-Disposition for keys with non-ASCII or quoting characters', async () => {
        const trickyKey = 'weird name "with quotes" 中文';
        mockMinioForDirectory({
          [`${trickyKey}/file.txt`]: 'payload',
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        const res = await captureBinaryResponse(
          request.get(
            `/artifacts/get?source=minio&bucket=ml-pipeline&key=${encodeURIComponent(trickyKey)}`,
          ),
        ).expect(200);

        const disposition = res.headers['content-disposition'] as string;
        expect(disposition).toMatch(/^attachment;/);

        // The legacy `filename=` parameter must be ASCII-only and quote-safe.
        const fallback = disposition.match(/filename="([^"]+)"/);
        expect(fallback).not.toBeNull();
        expect(fallback![1]).toMatch(/^[A-Za-z0-9._-]+$/);

        // The `filename*` parameter carries the real name via RFC 5987.
        expect(disposition).toContain("filename*=UTF-8''");
        // Spaces, quotes, and the Chinese characters must all be percent-encoded.
        expect(disposition).toContain(encodeURIComponent(`${trickyKey}.tar.gz`));
      });

      it('strips path-traversal segments from tar entry names (tar-slip protection)', async () => {
        // listObjectsV2Query under prefix "directory/" returning a key whose
        // relative path begins with ".." would, without sanitization, write
        // outside the user's extraction target.
        mockMinioForDirectory({
          'directory/../etc/passwd': 'malicious payload',
          'directory/legit.txt': 'real payload',
        });

        const configs = loadConfigs(argv, minioConfigEnv);
        app = new UIServer(configs);

        const request = requests(app.app);
        const res = await captureBinaryResponse(
          request.get('/artifacts/get?source=minio&bucket=ml-pipeline&key=directory'),
        ).expect(200);

        const entries = await readTarGzEntries(res.body as Buffer);
        const names = Array.from(entries.keys());
        expect(names).toContain('legit.txt');
        // No entry whose path traverses out of the extraction root.
        for (const name of names) {
          expect(name.startsWith('/')).toBe(false);
          expect(name.split('/')).not.toContain('..');
        }
      });
    });
  });

  describe('/:source/:bucket/:key', () => {
    it('downloads a minio artifact', async () => {
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/minio/ml-pipeline/hello/world.txt') // url
        .expect(200, artifactContent);
    });

    it('downloads a tar gzipped artifact as is', async () => {
      // base64 encoding for tar gzipped 'hello-world'
      const tarGzBase64 =
        'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
      const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
      artifactContent = tarGzBuffer;
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/minio/ml-pipeline/hello/world.txt') // url
        .expect(200, tarGzBuffer.toString());
    });
  });
});
