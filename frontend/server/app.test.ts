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
import express from 'express';

import fetch from 'node-fetch';
import requests from 'supertest';

import { UIServer } from './app';
import { loadConfigs } from './configs';
import { TEST_ONLY as K8S_TEST_EXPORT } from './k8s-helper';
import { Server } from 'http';
import { commonSetup } from './integration-tests/test-helper';

jest.mock('node-fetch');

// TODO: move sections of tests here to individual files in `frontend/server/integration-tests/`
// for better organization and shorter/more focused tests.

const mockedFetch: jest.Mock = fetch as any;

describe('UIServer apis', () => {
  let app: UIServer;
  const tagName = '1.0.0';
  const commitHash = 'abcdefg';
  const { argv, buildDate, indexHtmlContent } = commonSetup({ tagName, commitHash });

  afterEach(() => {
    if (app) {
      app.close();
    }
  });

  describe('/', () => {
    it('responds with unmodified index.html if it is not a kubeflow deployment', done => {
      const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT=null
  window.KFP_FLAGS.HIDE_SIDENAV=false
  </script>
  <script id="kubeflow-client-placeholder"></script>
</head>
</html>`;
      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);

      const request = requests(app.start());
      request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml, done);
    });

    it('responds with a modified index.html if it is a kubeflow deployment and sets HIDE_SIDENAV', done => {
      const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT="KUBEFLOW"
  window.KFP_FLAGS.HIDE_SIDENAV=true
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
  window.KFP_FLAGS.HIDE_SIDENAV=false
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

  it('responds with flag HIDE_SIDENAV=false even when DEPLOYMENT=KUBEFLOW', done => {
    const expectedIndexHtml = `
<html>
<head>
  <script>
  window.KFP_FLAGS.DEPLOYMENT="KUBEFLOW"
  window.KFP_FLAGS.HIDE_SIDENAV=false
  </script>
  <script id="kubeflow-client-placeholder" src="/dashboard_lib.bundle.js"></script>
</head>
</html>`;
    const configs = loadConfigs(argv, { DEPLOYMENT: 'KUBEFLOW', HIDE_SIDENAV: 'false' });
    app = new UIServer(configs);

    const request = requests(app.start());
    request
      .get('/')
      .expect('Content-Type', 'text/html; charset=utf-8')
      .expect(200, expectedIndexHtml, done);
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
            frontendTagName: tagName,
          },
          done,
        );
    });

    it('responds with both ui server and ml-pipeline api state if ml-pipeline api server is also ready.', done => {
      (fetch as any).mockImplementationOnce((_url: string, _opt: any) => ({
        json: () =>
          Promise.resolve({
            commit_sha: 'commit_sha',
            tag_name: '1.0.0',
            multi_user: false,
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
            apiServerTagName: '1.0.0',
            apiServerMultiUser: false,
            multi_user: false,
            apiServerReady: true,
            buildDate,
            frontendCommitHash: commitHash,
            frontendTagName: tagName,
          },
          done,
        );
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

  // TODO: Add integration tests for k8s helper related endpoints
  // describe('/k8s/pod/logs', () => {});

  describe('/apis/v1beta1/', () => {
    let request: requests.SuperTest<requests.Test>;
    let kfpApiServer: Server;

    beforeEach(() => {
      const kfpApiPort = 3001;
      kfpApiServer = express()
        .all('/*', (_, res) => {
          res.status(200).send('KFP API is working');
        })
        .listen(kfpApiPort);
      app = new UIServer(
        loadConfigs(argv, {
          ML_PIPELINE_SERVICE_PORT: `${kfpApiPort}`,
          ML_PIPELINE_SERVICE_HOST: 'localhost',
        }),
      );
      request = requests(app.start());
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
