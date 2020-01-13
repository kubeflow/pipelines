// Copyright 2019 Google LLC
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

import fetch from 'node-fetch';
import * as requests from 'supertest';
import { Client as MinioClient } from 'minio';
import { Storage as GCSStorage } from '@google-cloud/storage';

import { UIServer } from './app';
import { loadConfigs } from './configs';
import * as minioHelper from './minio-helper';
import { getTensorboardInstance } from './k8s-helper';

jest.mock('minio');
jest.mock('node-fetch');
jest.mock('@google-cloud/storage');
jest.mock('./minio-helper');
jest.mock('./k8s-helper');

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
  const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT="KUBEFLOW"
  </script>
  <script id="kubeflow-client-placeholder" src="/dashboard_lib.bundle.js"></script>
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
      const configs = loadConfigs(argv, { DEPLOYMENT: 'kubeflow' });
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
      const mockedGetTarObjectAsString: jest.Mock = minioHelper.getTarObjectAsString as any;
      mockedGetTarObjectAsString.mockImplementationOnce(opt =>
        opt.bucket === 'ml-pipeline' && opt.key === 'hello/world.txt'
          ? Promise.resolve(artifactContent)
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

    it('responds with a http artifact if source=http', done => {
      const artifactContent = 'hello world';
      const mockedFetch: jest.Mock = fetch as any;
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'http://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({ buffer: () => Promise.resolve(artifactContent) })
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

    it('responds with a https artifact if source=https', done => {
      const artifactContent = 'hello world';
      const mockedFetch: jest.Mock = fetch as any;
      mockedFetch.mockImplementationOnce((url: string, opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt' &&
        opts.headers.Authorization === 'someToken'
          ? Promise.resolve({ buffer: () => Promise.resolve(artifactContent) })
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
      const mockedFetch: jest.Mock = fetch as any;
      mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
        url === 'https://foo.bar/ml-pipeline/hello/world.txt'
          ? Promise.resolve({ buffer: () => Promise.resolve(artifactContent) })
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
  });

  // TODO: refractor k8s helper module so that api that interact with k8s can be
  // mocked and tested. There is currently no way to mock k8s APIs as
  // `k8s-helper.isInCluster` is a constant that is generated when the module is
  // first loaded.

  // describe('/apps/tensorboard', () => {

  //   it('get a tensorboard url', done => {
  //     const tensorboardUrl = 'http://tensorboard.view/abc';
  //     const mockedGetTensorboardInstance: jest.Mock = getTensorboardInstance as any;
  //     mockedGetTensorboardInstance.mockResolvedValueOnce(tensorboardUrl);
  //     const configs = loadConfigs(argv, {});
  //     app = new UIServer(configs);

  //     const request = requests(app.start());
  //     request.get('/apps/tensorboard?logdir=hello%2Fworld').expect(200, tensorboardUrl, done);
  //   });

  // });

  // describe('/k8s/pod/logs', () => {});
});
