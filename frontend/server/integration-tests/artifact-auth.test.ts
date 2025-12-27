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

import * as minio from 'minio';
import { PassThrough } from 'stream';
import requests from 'supertest';
import { UIServer } from '../app';
import { loadConfigs } from '../configs';
import { commonSetup } from './test-helper';

const MinioClient = minio.Client;
jest.mock('minio');
jest.mock('../k8s-helper');

jest.mock('portable-fetch', () => {
  return jest.fn();
});

describe('/artifacts authorization', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  const artifactContent = 'hello world';

  beforeEach(() => {
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

  afterEach(() => {
    if (app) {
      app.close();
    }
    jest.clearAllMocks();
  });

  describe('when auth is disabled', () => {
    it('allows artifact access without namespace parameter', done => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(200, artifactContent, done);
    });

    it('allows artifact access with any namespace parameter', done => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
      });
      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=any-namespace',
        )
        .expect(200, artifactContent, done);
    });
  });

  describe('when auth is enabled', () => {
    let mockAuthorize: jest.Mock;

    beforeEach(() => {
      const portableFetch = require('portable-fetch');
      mockAuthorize = portableFetch as jest.Mock;
    });

    it('requires namespace parameter when auth is enabled', done => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .set('kubeflow-userid', 'user@example.com')
        .expect(400)
        .expect(response => {
          expect(response.text).toContain('Namespace parameter is required');
        })
        .end(done);
    });

    it('rejects unauthenticated users before checking namespace', done => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(401)
        .expect(response => {
          expect(response.text).toContain('Authentication required');
        })
        .end(done);
    });

    it('rejects requests with invalid namespace format', done => {
      mockAuthorize.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=../../../etc',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(400)
        .expect(response => {
          expect(response.text).toContain('Invalid namespace');
        })
        .end(done);
    });

    it('rejects unauthorized cross-namespace access', done => {
      mockAuthorize.mockRejectedValue({
        status: 403,
        statusText: 'Forbidden',
        text: () => Promise.resolve('User is not authorized'),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=other-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(403)
        .expect(response => {
          expect(response.text).toContain('not authorized');
        })
        .end(done);
    });

    it('allows authorized namespace access', done => {
      mockAuthorize.mockImplementation(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({}),
          text: () => Promise.resolve(''),
        }),
      );

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=my-namespace',
        )
        .set('kubeflow-userid', 'user@example.com')
        .expect(200, artifactContent, done);
    });
  });

  describe('security logging', () => {
    let consoleSpy: jest.SpyInstance;

    beforeEach(() => {
      consoleSpy = jest.spyOn(console, 'warn').mockImplementation();
    });

    afterEach(() => {
      consoleSpy.mockRestore();
    });

    it('logs unauthorized access attempts with user info', done => {
      const portableFetch = require('portable-fetch');
      const mockAuthorize = portableFetch as jest.Mock;
      mockAuthorize.mockRejectedValue({
        status: 403,
        statusText: 'Forbidden',
        text: () => Promise.resolve('User is not authorized'),
      });

      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get(
          '/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt&namespace=unauthorized-ns',
        )
        .set('kubeflow-userid', 'attacker@example.com')
        .expect(403)
        .end(err => {
          expect(consoleSpy).toHaveBeenCalled();
          const logCall = consoleSpy.mock.calls.find(
            call => call[0] && call[0].includes('[SECURITY]'),
          );
          expect(logCall).toBeDefined();
          if (logCall) {
            expect(logCall[0]).toContain('attacker@example.com');
            expect(logCall[0]).toContain('unauthorized-ns');
          }
          done(err);
        });
    });

    it('logs unauthenticated access attempts', done => {
      const configurations = loadConfigs(argv, {
        MINIO_ACCESS_KEY: 'minio',
        MINIO_HOST: 'minio-service',
        MINIO_NAMESPACE: 'kubeflow',
        MINIO_PORT: '9000',
        MINIO_SECRET_KEY: 'minio123',
        MINIO_SSL: 'false',
        ML_PIPELINE_SERVICE_HOST: 'localhost',
        ML_PIPELINE_SERVICE_PORT: '8888',
        KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
        KUBEFLOW_USERID_PREFIX: '',
      });

      configurations.auth.enabled = true;

      app = new UIServer(configurations);

      const request = requests(app.start());
      request
        .get('/artifacts/get?source=minio&bucket=ml-pipeline&key=hello%2Fworld.txt')
        .expect(401)
        .end(err => {
          expect(consoleSpy).toHaveBeenCalled();
          const logCall = consoleSpy.mock.calls.find(
            call =>
              call[0] && call[0].includes('[SECURITY]') && call[0].includes('Unauthenticated'),
          );
          expect(logCall).toBeDefined();
          done(err);
        });
    });
  });
});
