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
import { Storage as GCSStorage } from '@google-cloud/storage';
import * as fs from 'fs';
import * as minio from 'minio';
import * as path from 'path';
import { PassThrough, Readable } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import * as serverInfo from '../helpers/server-info.js';
import { commonSetup, mkTempDir } from './test-helper.js';
import { getK8sSecret } from '../k8s-helper.js';

const MinioClient = minio.Client;
vi.mock('minio');
vi.mock('@google-cloud/storage');
vi.mock('../k8s-helper');

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
    mockedMinioClient.mockImplementation(function() {
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
      const mockedGcsStorage: Mock = GCSStorage as any;
      const mockedGetK8sSecret: Mock = getK8sSecret as any;
      mockedGetK8sSecret.mockResolvedValue('{"private_key":"testkey","client_email":"testemail"}');
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();
      mockedGcsStorage.mockImplementationOnce(function() {
        return {
          bucket: () => ({
            getFiles: () =>
              Promise.resolve([[{ name: 'hello/world.txt', createReadStream: () => stream }]]),
          }),
        };
      });
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
        credentials: {
          client_email: 'testemail',
          private_key: 'testkey',
        },
        scopes: 'https://www.googleapis.com/auth/devstorage.read_write',
      };
      expect(mockedGcsStorage).toBeCalledWith(expectedArg);
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
      const mockedGcsStorage: Mock = GCSStorage as any;
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();
      mockedGcsStorage.mockImplementationOnce(function() {
        return {
          bucket: () => ({
            getFiles: () =>
              Promise.resolve([[{ name: 'hello/world.txt', createReadStream: () => stream }]]),
          }),
        };
      });
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent + '\n');
    });

    it('responds with a partial gcs artifact if peek=5 is set', async () => {
      const artifactContent = 'hello world';
      const mockedGcsStorage: Mock = GCSStorage as any;
      const stream = new PassThrough();
      stream.end(artifactContent);
      mockedGcsStorage.mockImplementationOnce(function() {
        return {
          bucket: () => ({
            getFiles: () =>
              Promise.resolve([[{ name: 'hello/world.txt', createReadStream: () => stream }]]),
          }),
        };
      });
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.app);
      await request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5));
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
