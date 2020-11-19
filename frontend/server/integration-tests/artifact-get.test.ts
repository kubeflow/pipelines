// Copyright 2020 Google LLC
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

import { Storage as GCSStorage } from '@google-cloud/storage';
import * as fs from 'fs';
import * as minio from 'minio';
import fetch from 'node-fetch';
import * as os from 'os';
import * as path from 'path';
import { PassThrough } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app';
import { loadConfigs } from '../configs';
import * as serverInfo from '../helpers/server-info';
import * as minioHelper from '../minio-helper';
import { commonSetup } from './test-helper';

const MinioClient = minio.Client;
jest.mock('minio');
jest.mock('node-fetch');
jest.mock('@google-cloud/storage');

const mockedFetch: jest.Mock = fetch as any;

describe('/artifacts', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  let artifactContent: any = 'hello world';
  beforeEach(() => {
    artifactContent = 'hello world'; // reset
    const mockedMinioClient: jest.Mock = MinioClient as any;
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

  afterEach(() => {
    if (app) {
      app.close();
    }
  });

  describe('/get', () => {
    it('responds with a minio artifact if source=minio', done => {
      const mockedMinioClient: jest.Mock = minio.Client as any;

      const configs = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent, err => {
          expect(mockedMinioClient).toBeCalledWith({
            accessKey: 'minio',
            endPoint: 'minio-service.kubeflow',
            port: 9000,
            secretKey: 'minio123',
            useSSL: false,
          });
          done(err);
        });
    });

    it('responds with a s3 artifact if source=s3', done => {
      const mockedMinioClient: jest.Mock = jest.spyOn(minioHelper, 'createMinioClient') as any;
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent, err => {
          expect(mockedMinioClient).toBeCalledWith({
            accessKey: 'aws123',
            endPoint: 's3.amazonaws.com',
            secretKey: 'awsSecret123',
          });
          done(err);
        });
    });

    it('responds with partial s3 artifact if peek=5 flag is set', done => {
      const mockedMinioClient = jest.spyOn(minioHelper, 'createMinioClient');
      const configs = loadConfigs(argv, {
        AWS_ACCESS_KEY_ID: 'aws123',
        AWS_SECRET_ACCESS_KEY: 'awsSecret123',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=s3&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5), err => {
          expect(mockedMinioClient).toBeCalledWith({
            accessKey: 'aws123',
            endPoint: 's3.amazonaws.com',
            secretKey: 'awsSecret123',
          });
          done(err);
        });
    });

    it('responds with a http artifact if source=http', done => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'http://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: new PassThrough().end(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=http&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent, err => {
          expect(mockedFetch).toBeCalledWith('http://foo.bar/ml-pipeline/hello/world.txt', {
            headers: {},
          });
          done(err);
        });
    });

    it('responds with partial http artifact if peek=5 flag is set', done => {
      const artifactContent = 'hello world';
      const mockedFetch: jest.Mock = fetch as any;
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'http://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: new PassThrough().end(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=http&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5), err => {
          expect(mockedFetch).toBeCalledWith('http://foo.bar/ml-pipeline/hello/world.txt', {
            headers: {},
          });
          done(err);
        });
    });

    it('responds with a https artifact if source=https', done => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt' &&
        opts.headers.Authorization === 'someToken'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: new PassThrough().end(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_AUTHORIZATION_DEFAULT_VALUE: 'someToken',
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=https&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent, err => {
          expect(mockedFetch).toBeCalledWith('https://foo.bar/ml-pipeline/hello/world.txt', {
            headers: {
              Authorization: 'someToken',
            },
          });
          done(err);
        });
    });

    it('responds with a https artifact using the inherited header if source=https and http authorization key is provided.', done => {
      const artifactContent = 'hello world';
      mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({
              buffer: () => Promise.resolve(artifactContent),
              body: new PassThrough().end(artifactContent),
            })
          : Promise.reject('Unable to retrieve http artifact.'),
      );
      const configs = loadConfigs(argv, {
        HTTP_AUTHORIZATION_KEY: 'Authorization',
        HTTP_BASE_URL: 'foo.bar/',
      });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=https&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .set('Authorization', 'inheritedToken')
        .expect(200, artifactContent, err => {
          expect(mockedFetch).toBeCalledWith('https://foo.bar/ml-pipeline/hello/world.txt', {
            headers: {
              Authorization: 'inheritedToken',
            },
          });
          done(err);
        });
    });

    it('responds with a gcs artifact if source=gcs', done => {
      const artifactContent = 'hello world';
      const mockedGcsStorage: jest.Mock = GCSStorage as any;
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();
      mockedGcsStorage.mockImplementationOnce(() => ({
        bucket: () => ({
          getFiles: () =>
            Promise.resolve([[{ name: 'hello/world.txt', createReadStream: () => stream }]]),
        }),
      }));
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent + '\n', done);
    });

    it('responds with a partial gcs artifact if peek=5 is set', done => {
      const artifactContent = 'hello world';
      const mockedGcsStorage: jest.Mock = GCSStorage as any;
      const stream = new PassThrough();
      stream.end(artifactContent);
      mockedGcsStorage.mockImplementationOnce(() => ({
        bucket: () => ({
          getFiles: () =>
            Promise.resolve([[{ name: 'hello/world.txt', createReadStream: () => stream }]]),
        }),
      }));
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=gcs&bucket=ml-pipeline&key=hello%2Fworld.txt&peek=5')
        .expect(200, artifactContent.slice(0, 5), done);
    });

    it('responds with a volume artifact if source=volume', done => {
      const artifactContent = 'hello world';
      const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'content');
      fs.writeFileSync(tempPath, artifactContent);

      jest.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
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

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=volume&bucket=artifact&key=subartifact/content')
        .expect(200, artifactContent, done);
    });

    it('responds with a partial volume artifact if peek=5 is set', done => {
      const artifactContent = 'hello world';
      const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'content');
      fs.writeFileSync(tempPath, artifactContent);

      jest.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
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

      const request = requests(app.start());
      request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=content&peek=5`)
        .expect(200, artifactContent.slice(0, 5), done);
    });

    it('responds error with a not exist volume', done => {
      jest.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
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

      const request = requests(app.start());
      request
        .get(`/artifacts/get?source=volume&bucket=notexist&key=content`)
        .expect(
          404,
          'Failed to open volume://notexist/content, Cannot find file "volume://notexist/content" in pod "ml-pipeline-ui": volume "notexist" not configured',
          done,
        );
    });

    it('responds error with a not exist volume mount path if source=volume', done => {
      jest.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
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

      const request = requests(app.start());
      request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=notexist/config`)
        .expect(
          404,
          'Failed to open volume://artifact/notexist/config, Cannot find file "volume://artifact/notexist/config" in pod "ml-pipeline-ui": volume "artifact" not mounted or volume "artifact" with subPath (which is prefix of notexist/config) not mounted',
          done,
        );
    });

    it('responds error with a not exist volume mount artifact if source=volume', done => {
      jest.spyOn(serverInfo, 'getHostPod').mockImplementation(() =>
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

      const request = requests(app.start());
      request
        .get(`/artifacts/get?source=volume&bucket=artifact&key=subartifact/notxist.csv`)
        .expect(
          500,
          "Failed to open volume://artifact/subartifact/notxist.csv: Error: ENOENT: no such file or directory, stat '/foo/bar/notxist.csv'",
          done,
        );
    });
  });

  describe('/:source/:bucket/:key', () => {
    it('downloads a minio artifact', done => {
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/minio/ml-pipeline/hello/world.txt') // url
        .expect(200, artifactContent, done);
    });

    it('downloads a tar gzipped artifact as is', done => {
      // base64 encoding for tar gzipped 'hello-world'
      const tarGzBase64 =
        'H4sIAFa7DV4AA+3PSwrCMBRG4Y5dxV1BuSGPridgwcItkTZSl++johNBJ0WE803OIHfwZ87j0fq2nmuzGVVNIcitXYqPpntXLojzSb33MToVdTG5rhHdbtLLaa55uk5ZBrMhj23ty9u7T+/rT+TZP3HozYosZbL97tdbAAAAAAAAAAAAAAAAAADfuwAyiYcHACgAAA==';
      const tarGzBuffer = Buffer.from(tarGzBase64, 'base64');
      artifactContent = tarGzBuffer;
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/artifacts/minio/ml-pipeline/hello/world.txt') // url
        .expect(200, tarGzBuffer.toString(), done);
    });
  });
});
