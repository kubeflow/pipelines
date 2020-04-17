// Copyright 2019-2020 Google LLC
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
import * as os from 'os';
import * as fs from 'fs';
import * as path from 'path';
import { PassThrough } from 'stream';
import express from 'express';

import fetch from 'node-fetch';
import requests from 'supertest';
import { Client as MinioClient } from 'minio';
import { Storage as GCSStorage } from '@google-cloud/storage';

import { UIServer } from './app';
import { loadConfigs } from './configs';
import * as minioHelper from './minio-helper';
import { TEST_ONLY as K8S_TEST_EXPORT } from './k8s-helper';
import { Server } from 'http';

jest.mock('minio');
jest.mock('node-fetch');
jest.mock('@google-cloud/storage');
jest.mock('./minio-helper');

const mockedFetch: jest.Mock = fetch as any;

beforeEach(() => {
  const consoleInfoSpy = jest.spyOn(global.console, 'info');
  consoleInfoSpy.mockImplementation(() => null);
  const consoleLogSpy = jest.spyOn(global.console, 'log');
  consoleLogSpy.mockImplementation(() => null);
});

afterEach(() => {
  jest.restoreAllMocks();
  jest.resetAllMocks();
});

describe('UIServer apis', () => {
  let app: UIServer;
  const indexHtmlPath = path.resolve(os.tmpdir(), 'index.html');
  const argv = ['node', 'dist/server.js', os.tmpdir(), '3000'];
  const buildDate = new Date().toISOString();
  const commitHash = 'abcdefg';
  const indexHtmlContent = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT=null
  </script>
  <script id="kubeflow-client-placeholder"></script>
</head>
</html>`;

  beforeAll(() => {
    fs.writeFileSync(path.resolve(__dirname, 'BUILD_DATE'), buildDate);
    fs.writeFileSync(path.resolve(__dirname, 'COMMIT_HASH'), commitHash);
    fs.writeFileSync(indexHtmlPath, indexHtmlContent);
  });

  afterAll(() => {
    fs.unlinkSync(path.resolve(__dirname, 'BUILD_DATE'));
    fs.unlinkSync(path.resolve(__dirname, 'COMMIT_HASH'));
    fs.unlinkSync(indexHtmlPath);
  });

  beforeEach(() => {
    jest.resetAllMocks();
  });

  afterEach(() => {
    if (app) {
      app.close();
    }
  });

  describe('/', () => {
    it('responds with unmodified index.html if it is not a kubeflow deployment', done => {
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, indexHtmlContent, done);
    });

    it('responds with a modified index.html if it is a kubeflow deployment', done => {
      const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT="KUBEFLOW"
  </script>
  <script id="kubeflow-client-placeholder" src="/dashboard_lib.bundle.js"></script>
</head>
</html>`;
      const configs = loadConfigs(argv, { DEPLOYMENT: 'kubeflow' });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml, done);
    });

    it('responds with flag DEPLOYMENT=MARKETPLACE if it is a marketplace deployment', done => {
      const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT="MARKETPLACE"
  </script>
  <script id="kubeflow-client-placeholder"></script>
</head>
</html>`;
      const configs = loadConfigs(argv, { DEPLOYMENT: 'marketplace' });
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml, done);
    });
  });

  describe('/apis/v1beta1/healthz', () => {
    it('responds with apiServerReady to be false if ml-pipeline api server is not ready.', done => {
      (fetch as any).mockImplementationOnce((_url: string, _opt: any) => ({
        json: () => Promise.reject('Unknown error'),
      }));

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      requests(app.start())
        .get('/apis/v1beta1/healthz')
        .expect(
          200,
          {
            apiServerReady: false,
            buildDate,
            frontendCommitHash: commitHash,
          },
          done,
        );
    });

    it('responds with both ui server and ml-pipeline api state if ml-pipeline api server is also ready.', done => {
      (fetch as any).mockImplementationOnce((_url: string, _opt: any) => ({
        json: () =>
          Promise.resolve({
            commit_sha: 'commit_sha',
          }),
      }));

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      requests(app.start())
        .get('/apis/v1beta1/healthz')
        .expect(
          200,
          {
            apiServerCommitHash: 'commit_sha',
            apiServerReady: true,
            buildDate,
            frontendCommitHash: commitHash,
          },
          done,
        );
    });
  });

  describe('/artifacts/get', () => {
    it('responds with a minio artifact if source=minio', done => {
      const artifactContent = 'hello world';
      const mockedMinioClient: jest.Mock = MinioClient as any;
      const mockedGetObjectStream: jest.Mock = minioHelper.getObjectStream as any;
      const objStream = new PassThrough();
      objStream.end(artifactContent);

      mockedGetObjectStream.mockImplementationOnce(opt =>
        opt.bucket === 'ml-pipeline' && opt.key === 'hello/world.txt'
          ? Promise.resolve(objStream)
          : Promise.reject('Unable to retrieve minio artifact.'),
      );
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
      const artifactContent = 'hello world';
      const mockedMinioClient: jest.Mock = minioHelper.createMinioClient as any;
      const mockedGetObjectStream: jest.Mock = minioHelper.getObjectStream as any;
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();

      mockedGetObjectStream.mockImplementationOnce(opt =>
        opt.bucket === 'ml-pipeline' && opt.key === 'hello/world.txt'
          ? Promise.resolve(stream)
          : Promise.reject('Unable to retrieve s3 artifact.'),
      );
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
      const artifactContent = 'hello world';
      const mockedMinioClient: jest.Mock = minioHelper.createMinioClient as any;
      const mockedGetObjectStream: jest.Mock = minioHelper.getObjectStream as any;
      const stream = new PassThrough();
      stream.write(artifactContent);
      stream.end();

      mockedGetObjectStream.mockImplementationOnce(opt =>
        opt.bucket === 'ml-pipeline' && opt.key === 'hello/world.txt'
          ? Promise.resolve(stream)
          : Promise.reject('Unable to retrieve s3 artifact.'),
      );
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
  });

  describe('/system', () => {
    describe('/cluster-name', () => {
      it('responds with cluster name data from gke metadata', done => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/instance/attributes/cluster-name'
            ? Promise.resolve({ ok: true, text: () => Promise.resolve('test-cluster') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, {}));

        const request = requests(app.start());
        request
          .get('/system/cluster-name')
          .expect('Content-Type', 'text/html; charset=utf-8')
          .expect(200, 'test-cluster', done);
      });
      it('responds with 500 status code if corresponding endpoint is not ok', done => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/instance/attributes/cluster-name'
            ? Promise.resolve({ ok: false, text: () => Promise.resolve('404 not found') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, {}));

        const request = requests(app.start());
        request.get('/system/cluster-name').expect(500, 'Failed fetching GKE cluster name', done);
      });
      it('responds with endpoint disabled if DISABLE_GKE_METADATA env is true', done => {
        const configs = loadConfigs(argv, { DISABLE_GKE_METADATA: 'true' });
        app = new UIServer(configs);

        const request = requests(app.start());
        request
          .get('/system/cluster-name')
          .expect(500, 'GKE metadata endpoints are disabled.', done);
      });
    });

    describe('/project-id', () => {
      it('responds with project id data from gke metadata', done => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/project/project-id'
            ? Promise.resolve({ ok: true, text: () => Promise.resolve('test-project') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, {}));

        const request = requests(app.start());
        request.get('/system/project-id').expect(200, 'test-project', done);
      });
      it('responds with 500 status code if metadata request is not ok', done => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/project/project-id'
            ? Promise.resolve({ ok: false, text: () => Promise.resolve('404 not found') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, {}));

        const request = requests(app.start());
        request.get('/system/project-id').expect(500, 'Failed fetching GKE project id', done);
      });
      it('responds with endpoint disabled if DISABLE_GKE_METADATA env is true', done => {
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'true' }));

        const request = requests(app.start());
        request.get('/system/project-id').expect(500, 'GKE metadata endpoints are disabled.', done);
      });
    });
  });

  describe('/k8s/pod', () => {
    let request: requests.SuperTest<requests.Test>;
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.start());
    });

    it('asks for podname if not provided', done => {
      request.get('/k8s/pod').expect(422, 'podname argument is required', done);
    });

    it('asks for podnamespace if not provided', done => {
      request
        .get('/k8s/pod?podname=test-pod')
        .expect(422, 'podnamespace argument is required', done);
    });

    it('responds with pod info in JSON', done => {
      const readPodSpy = jest.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
      readPodSpy.mockImplementation(() =>
        Promise.resolve({
          body: { kind: 'Pod' }, // only body is used
        } as any),
      );
      request
        .get('/k8s/pod?podname=test-pod&podnamespace=test-ns')
        .expect(200, '{"kind":"Pod"}', err => {
          expect(readPodSpy).toHaveBeenCalledWith('test-pod', 'test-ns');
          done(err);
        });
    });

    it('responds with error when failed to retrieve pod info', done => {
      const readPodSpy = jest.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
      readPodSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'pod not found',
            code: 404,
          },
        } as any),
      );
      const spyError = jest.spyOn(console, 'error').mockImplementation(() => null);
      request
        .get('/k8s/pod?podname=test-pod&podnamespace=test-ns')
        .expect(500, 'Could not get pod test-pod in namespace test-ns: pod not found', () => {
          expect(spyError).toHaveBeenCalledTimes(1);
          done();
        });
    });
  });

  describe('/k8s/pod/events', () => {
    let request: requests.SuperTest<requests.Test>;
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.start());
    });

    it('asks for podname if not provided', done => {
      request.get('/k8s/pod/events').expect(422, 'podname argument is required', done);
    });

    it('asks for podnamespace if not provided', done => {
      request
        .get('/k8s/pod/events?podname=test-pod')
        .expect(422, 'podnamespace argument is required', done);
    });

    it('responds with pod info in JSON', done => {
      const listEventSpy = jest.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
      listEventSpy.mockImplementation(() =>
        Promise.resolve({
          body: { kind: 'EventList' }, // only body is used
        } as any),
      );
      request
        .get('/k8s/pod/events?podname=test-pod&podnamespace=test-ns')
        .expect(200, '{"kind":"EventList"}', err => {
          expect(listEventSpy).toHaveBeenCalledWith(
            'test-ns',
            undefined,
            undefined,
            undefined,
            'involvedObject.namespace=test-ns,involvedObject.name=test-pod,involvedObject.kind=Pod',
          );
          done(err);
        });
    });

    it('responds with error when failed to retrieve pod info', done => {
      const listEventSpy = jest.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
      listEventSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'no events',
            code: 404,
          },
        } as any),
      );
      const spyError = jest.spyOn(console, 'error').mockImplementation(() => null);
      request
        .get('/k8s/pod/events?podname=test-pod&podnamespace=test-ns')
        .expect(
          500,
          'Error when listing pod events for pod "test-pod" in "test-ns" namespace: no events',
          err => {
            expect(spyError).toHaveBeenCalledTimes(1);
            done(err);
          },
        );
    });
  });

  describe('/apps/tensorboard', () => {
    let request: requests.SuperTest<requests.Test>;
    let k8sGetCustomObjectSpy: jest.SpyInstance;
    let k8sDeleteCustomObjectSpy: jest.SpyInstance;
    let k8sCreateCustomObjectSpy: jest.SpyInstance;
    function newGetTensorboardResponse({
      name = 'viewer-example',
      logDir = 'log-dir-example',
      tensorflowImage = 'tensorflow:2.0.0',
      type = 'tensorboard',
    }: {
      name?: string;
      logDir?: string;
      tensorflowImage?: string;
      type?: string;
    } = {}) {
      return {
        response: undefined as any, // unused
        body: {
          metadata: {
            name,
          },
          spec: {
            tensorboardSpec: { logDir, tensorflowImage },
            type,
          },
        },
      };
    }
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.start());
      k8sGetCustomObjectSpy = jest.spyOn(
        K8S_TEST_EXPORT.k8sV1CustomObjectClient,
        'getNamespacedCustomObject',
      );
      k8sDeleteCustomObjectSpy = jest.spyOn(
        K8S_TEST_EXPORT.k8sV1CustomObjectClient,
        'deleteNamespacedCustomObject',
      );
      k8sCreateCustomObjectSpy = jest.spyOn(
        K8S_TEST_EXPORT.k8sV1CustomObjectClient,
        'createNamespacedCustomObject',
      );
    });

    describe('get', () => {
      it('requires logdir for get tensorboard', done => {
        request.get('/apps/tensorboard').expect(404, 'logdir argument is required', done);
      });

      it('requires namespace for get tensorboard', done => {
        request
          .get('/apps/tensorboard?logdir=some-log-dir')
          .expect(404, 'namespace argument is required', done);
      });

      it('does not crash with a weird query', done => {
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(newGetTensorboardResponse()),
        );
        // The special case is that, decodeURIComponent('%2') throws an
        // exception, so this can verify handler doesn't do extra
        // decodeURIComponent on queries.
        const weirdLogDir = encodeURIComponent('%2');
        request.get(`/apps/tensorboard?logdir=${weirdLogDir}&namespace=test-ns`).expect(200, done);
      });

      it('gets tensorboard url and version', done => {
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(
            newGetTensorboardResponse({
              name: 'viewer-abcdefg',
              logDir: 'log-dir-1',
              tensorflowImage: 'tensorflow:2.0.0',
            }),
          ),
        );

        request
          .get(`/apps/tensorboard?logdir=${encodeURIComponent('log-dir-1')}&namespace=test-ns`)
          .expect(
            200,
            JSON.stringify({
              podAddress:
                'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
              tfVersion: '2.0.0',
            }),
            err => {
              expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
                              Array [
                                "kubeflow.org",
                                "v1beta1",
                                "test-ns",
                                "viewers",
                                "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                              ]
                          `);
              done(err);
            },
          );
      });
    });

    describe('post (create)', () => {
      it('requires logdir', done => {
        request.post('/apps/tensorboard').expect(404, 'logdir argument is required', done);
      });

      it('requires namespace', done => {
        request
          .post('/apps/tensorboard?logdir=some-log-dir')
          .expect(404, 'namespace argument is required', done);
      });

      it('requires tfversion', done => {
        request
          .post('/apps/tensorboard?logdir=some-log-dir&namespace=test-ns')
          .expect(404, 'tfversion (tensorflow version) argument is required', done);
      });

      it('creates tensorboard viewer custom object and waits for it', done => {
        let getRequestCount = 0;
        k8sGetCustomObjectSpy.mockImplementation(() => {
          ++getRequestCount;
          switch (getRequestCount) {
            case 1:
              return Promise.reject('Not found');
            case 2:
              return Promise.resolve(
                newGetTensorboardResponse({
                  name: 'viewer-abcdefg',
                  logDir: 'log-dir-1',
                  tensorflowImage: 'tensorflow:2.0.0',
                }),
              );
            default:
              throw new Error('only expected to be called twice in this test');
          }
        });
        k8sCreateCustomObjectSpy.mockImplementation(() => Promise.resolve());

        request
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            200,
            'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
            err => {
              expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
                Array [
                  "kubeflow.org",
                  "v1beta1",
                  "test-ns",
                  "viewers",
                  "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                ]
              `);
              expect(k8sCreateCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
                Array [
                  "kubeflow.org",
                  "v1beta1",
                  "test-ns",
                  "viewers",
                  Object {
                    "apiVersion": "kubeflow.org/v1beta1",
                    "kind": "Viewer",
                    "metadata": Object {
                      "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                      "namespace": "test-ns",
                    },
                    "spec": Object {
                      "podTemplateSpec": Object {
                        "spec": Object {
                          "containers": Array [
                            Object {},
                          ],
                        },
                      },
                      "tensorboardSpec": Object {
                        "logDir": "log-dir-1",
                        "tensorflowImage": "tensorflow/tensorflow:2.0.0",
                      },
                      "type": "tensorboard",
                    },
                  },
                ]
              `);
              expect(k8sGetCustomObjectSpy.mock.calls[1]).toMatchInlineSnapshot(`
                Array [
                  "kubeflow.org",
                  "v1beta1",
                  "test-ns",
                  "viewers",
                  "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                ]
              `);
              done(err);
            },
          );
      });

      it('returns error when there is an existing tensorboard with different version', done => {
        const errorSpy = jest.spyOn(console, 'error');
        errorSpy.mockImplementation();
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(
            newGetTensorboardResponse({
              name: 'viewer-abcdefg',
              logDir: 'log-dir-1',
              tensorflowImage: 'tensorflow:2.1.0',
            }),
          ),
        );
        k8sCreateCustomObjectSpy.mockImplementation(() => Promise.resolve());

        request
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            500,
            `Failed to start Tensorboard app: Error: There's already an existing tensorboard instance with a different version 2.1.0`,
            err => {
              expect(errorSpy).toHaveBeenCalledTimes(1);
              done(err);
            },
          );
      });

      it('returns existing pod address if there is an existing tensorboard with the same version', done => {
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(
            newGetTensorboardResponse({
              name: 'viewer-abcdefg',
              logDir: 'log-dir-1',
              tensorflowImage: 'tensorflow:2.0.0',
            }),
          ),
        );
        k8sCreateCustomObjectSpy.mockImplementation(() => Promise.resolve());

        request
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            200,
            'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
            done,
          );
      });
    });

    describe('delete', () => {
      it('requires logdir', done => {
        request.delete('/apps/tensorboard').expect(404, 'logdir argument is required', done);
      });

      it('requires namespace', done => {
        request
          .delete('/apps/tensorboard?logdir=some-log-dir')
          .expect(404, 'namespace argument is required', done);
      });

      it('deletes tensorboard viewer custom object', done => {
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(
            newGetTensorboardResponse({
              name: 'viewer-abcdefg',
              logDir: 'log-dir-1',
              tensorflowImage: 'tensorflow:2.0.0',
            }),
          ),
        );
        k8sDeleteCustomObjectSpy.mockImplementation(() => Promise.resolve());

        request
          .delete(`/apps/tensorboard?logdir=${encodeURIComponent('log-dir-1')}&namespace=test-ns`)
          .expect(200, 'Tensorboard deleted.', err => {
            expect(k8sDeleteCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
              Array [
                "kubeflow.org",
                "v1beta1",
                "test-ns",
                "viewers",
                "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                V1DeleteOptions {},
              ]
            `);
            done(err);
          });
      });
    });
  });

  // TODO: Add integration tests for k8s helper related endpoints
  // describe('/k8s/pod/logs', () => {});

  describe('/apis/v1beta1/', () => {
    let request: requests.SuperTest<requests.Test>;
    let kfpApiServer: Server;

    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.start());
      const appKfpApi = express();
      appKfpApi.all('/*', (_, res) => {
        res.status(200).send('KFP API is working');
      });
      kfpApiServer = appKfpApi.listen(3001);
    });

    afterEach(() => {
      if (kfpApiServer) {
        kfpApiServer.close();
      }
    });

    it('rejects reportMetrics because it is not public kfp api', done => {
      const runId = 'a-random-run-id';
      request
        .post(`/apis/v1beta1/runs/${runId}:reportMetrics`)
        .expect(
          403,
          '/apis/v1beta1/runs/a-random-run-id:reportMetrics endpoint is not meant for external usage.',
          done,
        );
    });

    it('rejects reportWorkflow because it is not public kfp api', done => {
      const workflowId = 'a-random-workflow-id';
      request
        .post(`/apis/v1beta1/workflows/${workflowId}`)
        .expect(
          403,
          '/apis/v1beta1/workflows/a-random-workflow-id endpoint is not meant for external usage.',
          done,
        );
    });

    it('rejects reportScheduledWorkflow because it is not public kfp api', done => {
      const swf = 'a-random-swf-id';
      request
        .post(`/apis/v1beta1/scheduledworkflows/${swf}`)
        .expect(
          403,
          '/apis/v1beta1/scheduledworkflows/a-random-swf-id endpoint is not meant for external usage.',
          done,
        );
    });

    it('does not reject similar apis', done => {
      request // use reportMetrics as runId to see if it can confuse route parsing
        .post(`/apis/v1beta1/runs/xxx-reportMetrics:archive`)
        .expect(200, 'KFP API is working', done);
    });

    it('proxies other run apis', done => {
      request
        .post(`/apis/v1beta1/runs/a-random-run-id:archive`)
        .expect(200, 'KFP API is working', done);
    });
  });
});
