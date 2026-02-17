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
import { vi, describe, it, expect, afterAll, afterEach, beforeEach, Mock } from 'vitest';
import express from 'express';

import requests from 'supertest';

import { UIServer } from './app.js';
import { loadConfigs } from './configs.js';
import { TEST_ONLY as K8S_TEST_EXPORT } from './k8s-helper.js';
import { Server } from 'http';
import { commonSetup } from './integration-tests/test-helper.js';

const mockedFetch = vi.fn();
vi.stubGlobal('fetch', mockedFetch);

// TODO: move sections of tests here to individual files in `frontend/server/integration-tests/`
// for better organization and shorter/more focused tests.

afterAll(() => {
  vi.unstubAllGlobals();
});

describe('UIServer apis', () => {
  let app: UIServer;
  const tagName = '1.0.0';
  const commitHash = 'abcdefg';
  const { argv, buildDate, indexHtmlContent } = commonSetup({ tagName, commitHash });

  async function waitForListening(server: Server): Promise<void> {
    if (server.listening) {
      return;
    }
    await new Promise<void>((resolve, reject) => {
      const onListening = (): void => {
        cleanup();
        resolve();
      };
      const onError = (err: Error): void => {
        cleanup();
        reject(err);
      };
      const cleanup = (): void => {
        server.off('listening', onListening);
        server.off('error', onError);
      };

      server.on('listening', onListening);
      server.on('error', onError);
    });
  }

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  it('allows restarting on the same port after close resolves', async () => {
    app = new UIServer(loadConfigs(argv, {}));

    const firstServer = app.start(0);
    await waitForListening(firstServer);
    const firstAddress = firstServer.address();
    if (!firstAddress || typeof firstAddress === 'string') {
      throw new Error('Expected first server to bind to a TCP port');
    }

    await app.close();

    const secondServer = app.start(firstAddress.port);
    await waitForListening(secondServer);
    const secondAddress = secondServer.address();
    if (!secondAddress || typeof secondAddress === 'string') {
      throw new Error('Expected second server to bind to a TCP port');
    }
    expect(secondAddress.port).toBe(firstAddress.port);
  });

  describe('/', () => {
    it('responds with unmodified index.html if it is not a kubeflow deployment', async () => {
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

      const request = requests(app.app);
      await request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml);
    });

    it('responds with a modified index.html if it is a kubeflow deployment and sets HIDE_SIDENAV', async () => {
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

      const request = requests(app.app);
      await request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml);
    });

    it('responds with flag DEPLOYMENT=MARKETPLACE if it is a marketplace deployment', async () => {
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

      const request = requests(app.app);
      await request
        .get('/')
        .expect('Content-Type', 'text/html; charset=utf-8')
        .expect(200, expectedIndexHtml);
    });
  });

  it('responds with flag HIDE_SIDENAV=false even when DEPLOYMENT=KUBEFLOW', async () => {
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

    const request = requests(app.app);
    await request
      .get('/')
      .expect('Content-Type', 'text/html; charset=utf-8')
      .expect(200, expectedIndexHtml);
  });

  describe('/apis/v1beta1/healthz', () => {
    it('responds with apiServerReady to be false if ml-pipeline api server is not ready.', async () => {
      (fetch as any).mockImplementationOnce((_url: string, _opt: any) => ({
        json: () => Promise.reject('Unknown error'),
      }));

      const configs = loadConfigs(argv, {});
      app = new UIServer(configs);
      const res = await requests(app.app)
        .get('/apis/v1beta1/healthz')
        .expect(200);
      expect(res.body).toMatchObject({
        apiServerReady: false,
        frontendCommitHash: commitHash,
        frontendTagName: tagName,
      });
      expect(res.body).toHaveProperty('buildDate');
    });

    it('responds with both ui server and ml-pipeline api state if ml-pipeline api server is also ready.', async () => {
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
      const res = await requests(app.app)
        .get('/apis/v1beta1/healthz')
        .expect(200);
      expect(res.body).toMatchObject({
        apiServerCommitHash: 'commit_sha',
        apiServerTagName: '1.0.0',
        apiServerMultiUser: false,
        multi_user: false,
        apiServerReady: true,
        frontendCommitHash: commitHash,
        frontendTagName: tagName,
      });
      expect(res.body).toHaveProperty('buildDate');
    });
  });

  describe('/system', () => {
    describe('/cluster-name', () => {
      it('responds with cluster name data from gke metadata', async () => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/instance/attributes/cluster-name'
            ? Promise.resolve({ ok: true, text: () => Promise.resolve('test-cluster') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'false' }));

        const request = requests(app.app);
        await request
          .get('/system/cluster-name')
          .expect('Content-Type', 'text/html; charset=utf-8')
          .expect(200, 'test-cluster');
      });
      it('responds with 500 status code if corresponding endpoint is not ok', async () => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/instance/attributes/cluster-name'
            ? Promise.resolve({ ok: false, text: () => Promise.resolve('404 not found') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'false' }));

        const request = requests(app.app);
        await request.get('/system/cluster-name').expect(500, 'Failed fetching GKE cluster name');
      });
      it('responds with endpoint disabled if DISABLE_GKE_METADATA env is true', async () => {
        const configs = loadConfigs(argv, { DISABLE_GKE_METADATA: 'true' });
        app = new UIServer(configs);

        const request = requests(app.app);
        await request
          .get('/system/cluster-name')
          .expect(500, 'GKE metadata endpoints are disabled.');
      });
    });

    describe('/project-id', () => {
      it('responds with project id data from gke metadata', async () => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/project/project-id'
            ? Promise.resolve({ ok: true, text: () => Promise.resolve('test-project') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'false' }));

        const request = requests(app.app);
        await request.get('/system/project-id').expect(200, 'test-project');
      });
      it('responds with 500 status code if metadata request is not ok', async () => {
        mockedFetch.mockImplementationOnce((url: string, _opts: any) =>
          url === 'http://metadata/computeMetadata/v1/project/project-id'
            ? Promise.resolve({ ok: false, text: () => Promise.resolve('404 not found') })
            : Promise.reject('Unexpected request'),
        );
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'false' }));

        const request = requests(app.app);
        await request.get('/system/project-id').expect(500, 'Failed fetching GKE project id');
      });
      it('responds with endpoint disabled if DISABLE_GKE_METADATA env is true', async () => {
        app = new UIServer(loadConfigs(argv, { DISABLE_GKE_METADATA: 'true' }));

        const request = requests(app.app);
        await request.get('/system/project-id').expect(500, 'GKE metadata endpoints are disabled.');
      });
    });
  });

  describe('/k8s/pod', () => {
    let request: requests.SuperTest<requests.Test>;
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.app);
    });

    it('asks for podname if not provided', async () => {
      await request.get('/k8s/pod').expect(422, 'podname argument is required');
    });

    it('asks for podnamespace if not provided', async () => {
      await request
        .get('/k8s/pod?podname=test-pod')
        .expect(422, 'podnamespace argument is required');
    });

    it('responds with pod info in JSON', async () => {
      const readPodSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
      readPodSpy.mockImplementation(() => Promise.resolve({ kind: 'Pod' } as any));
      await request
        .get('/k8s/pod?podname=test-pod&podnamespace=test-ns')
        .expect(200, '{"kind":"Pod"}');
      expect(readPodSpy).toHaveBeenCalledWith({ name: 'test-pod', namespace: 'test-ns' });
    });

    it('responds with error when failed to retrieve pod info', async () => {
      const readPodSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
      readPodSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'pod not found',
            code: 404,
          },
        } as any),
      );
      const spyError = vi.spyOn(console, 'error').mockImplementation(() => null);
      await request
        .get('/k8s/pod?podname=test-pod&podnamespace=test-ns')
        .expect(500, 'Could not get pod test-pod in namespace test-ns');
      expect(spyError).toHaveBeenCalledTimes(1);
    });

    it('responds with error when invalid resource name', async () => {
      const readPodSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'readNamespacedPod');
      readPodSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'pod not found',
            code: 404,
          },
        } as any),
      );
      const spyError = vi.spyOn(console, 'error').mockImplementation(() => null);
      await request
        .get(
          '/k8s/pod?podname=test-pod-name&podnamespace=test-namespace%7d%7dt93g1%3Cscript%3Ealert(1)%3C%2fscript%3Ej66h',
        )
        .expect(500, 'Invalid resource name');
      expect(spyError).toHaveBeenCalledTimes(1);
    });
  });

  describe('/k8s/pod with authorization enabled', () => {
    let authServer: Server;
    const authPort = 3002;

    beforeEach(async () => {
      authServer = await new Promise<Server>(resolve => {
        const server = express()
          .post('/apis/v1beta1/auth', (_, res) => {
            res.status(401).send('Unauthorized');
          })
          .listen(authPort, () => resolve(server));
      });

      app = new UIServer(
        loadConfigs(argv, {
          ENABLE_AUTHZ: 'true',
          ML_PIPELINE_SERVICE_PORT: `${authPort}`,
          ML_PIPELINE_SERVICE_HOST: 'localhost',
        }),
      );
    });

    afterEach(async () => {
      if (authServer) {
        await new Promise<void>(resolve => authServer.close(() => resolve()));
      }
    });

    it('responds with 403 when authorization is rejected', async () => {
      const authRequest = requests(app.app);
      await authRequest
        .get('/k8s/pod?podname=test-pod&podnamespace=test-ns')
        .expect(403, 'Access denied to namespace');
    });
  });

  describe('/k8s/pod/events', () => {
    let request: requests.SuperTest<requests.Test>;
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.app);
    });

    it('asks for podname if not provided', async () => {
      await request.get('/k8s/pod/events').expect(422, 'podname argument is required');
    });

    it('asks for podnamespace if not provided', async () => {
      await request
        .get('/k8s/pod/events?podname=test-pod')
        .expect(422, 'podnamespace argument is required');
    });

    it('responds with pod info in JSON', async () => {
      const listEventSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
      listEventSpy.mockImplementation(() => Promise.resolve({ kind: 'EventList' } as any));
      await request
        .get('/k8s/pod/events?podname=test-pod&podnamespace=test-ns')
        .expect(200, '{"kind":"EventList"}');
      expect(listEventSpy).toHaveBeenCalledWith({
        namespace: 'test-ns',
        fieldSelector:
          'involvedObject.namespace=test-ns,involvedObject.name=test-pod,involvedObject.kind=Pod',
      });
    });

    it('responds with error when failed to retrieve pod info', async () => {
      const listEventSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
      listEventSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'no events',
            code: 404,
          },
        } as any),
      );
      const spyError = vi.spyOn(console, 'error').mockImplementation(() => null);
      await request
        .get('/k8s/pod/events?podname=test-pod&podnamespace=test-ns')
        .expect(500, 'Error when listing pod events for pod test-pod in namespace test-ns');
      expect(spyError).toHaveBeenCalledTimes(1);
    });

    it('responds with error when invalid resource name', async () => {
      const listEventSpy = vi.spyOn(K8S_TEST_EXPORT.k8sV1Client, 'listNamespacedEvent');
      listEventSpy.mockImplementation(() =>
        Promise.reject({
          body: {
            message: 'no events',
            code: 404,
          },
        } as any),
      );
      const spyError = vi.spyOn(console, 'error').mockImplementation(() => null);
      await request
        .get(
          '/k8s/pod/events?podname=test-pod-name&podnamespace=test-namespace%7d%7dt93g1%3Cscript%3Ealert(1)%3C%2fscript%3Ej66h',
        )
        .expect(500, 'Invalid resource name');
      expect(spyError).toHaveBeenCalledTimes(1);
    });
  });

  describe('/k8s/pod/events with authorization enabled', () => {
    let authServer: Server;
    const authPort = 3003;

    beforeEach(async () => {
      authServer = await new Promise<Server>(resolve => {
        const server = express()
          .post('/apis/v1beta1/auth', (_, res) => {
            res.status(401).send('Unauthorized');
          })
          .listen(authPort, () => resolve(server));
      });

      app = new UIServer(
        loadConfigs(argv, {
          ENABLE_AUTHZ: 'true',
          ML_PIPELINE_SERVICE_PORT: `${authPort}`,
          ML_PIPELINE_SERVICE_HOST: 'localhost',
        }),
      );
    });

    afterEach(async () => {
      if (authServer) {
        await new Promise<void>(resolve => authServer.close(() => resolve()));
      }
    });

    it('responds with 403 when authorization is rejected', async () => {
      const authRequest = requests(app.app);
      await authRequest
        .get('/k8s/pod/events?podname=test-pod&podnamespace=test-ns')
        .expect(403, 'Access denied to namespace');
    });
  });

  describe('/k8s/pod/logs', () => {
    let request: requests.SuperTest<requests.Test>;
    beforeEach(() => {
      app = new UIServer(loadConfigs(argv, {}));
      request = requests(app.app);
    });

    it('asks for podname if not provided', async () => {
      await request.get('/k8s/pod/logs').expect(400, 'podname argument is required');
    });
  });

  describe('/k8s/pod/logs with authorization enabled', () => {
    let authServer: Server;
    const authPort = 3004;

    beforeEach(async () => {
      authServer = await new Promise<Server>(resolve => {
        const server = express()
          .post('/apis/v1beta1/auth', (_, res) => {
            res.status(401).send('Unauthorized');
          })
          .listen(authPort, () => resolve(server));
      });

      app = new UIServer(
        loadConfigs(argv, {
          ENABLE_AUTHZ: 'true',
          ML_PIPELINE_SERVICE_PORT: `${authPort}`,
          ML_PIPELINE_SERVICE_HOST: 'localhost',
        }),
      );
    });

    afterEach(async () => {
      if (authServer) {
        await new Promise<void>(resolve => authServer.close(() => resolve()));
      }
    });

    it('responds with 403 when authorization is rejected', async () => {
      const authRequest = requests(app.app);
      await authRequest
        .get('/k8s/pod/logs?podname=test-pod&podnamespace=test-ns')
        .expect(403, 'Access denied to namespace');
    });

    it('asks for podnamespace if not provided when authorization is enabled', async () => {
      const authRequest = requests(app.app);
      await authRequest
        .get('/k8s/pod/logs?podname=test-pod')
        .expect(422, 'podnamespace argument is required');
    });
  });

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
      request = requests(app.app);
    });

    afterEach(async () => {
      if (kfpApiServer) {
        await new Promise<void>(resolve => kfpApiServer.close(() => resolve()));
      }
    });

    it('rejects reportMetrics because it is not public kfp api', async () => {
      const runId = 'a-random-run-id';
      await request
        .post(`/apis/v1beta1/runs/${runId}:reportMetrics`)
        .expect(
          403,
          '/apis/v1beta1/runs/a-random-run-id:reportMetrics endpoint is not meant for external usage.',
        );
    });

    it('rejects reportWorkflow because it is not public kfp api', async () => {
      const workflowId = 'a-random-workflow-id';
      await request
        .post(`/apis/v1beta1/workflows/${workflowId}`)
        .expect(
          403,
          '/apis/v1beta1/workflows/a-random-workflow-id endpoint is not meant for external usage.',
        );
    });

    it('rejects reportScheduledWorkflow because it is not public kfp api', async () => {
      const swf = 'a-random-swf-id';
      await request
        .post(`/apis/v1beta1/scheduledworkflows/${swf}`)
        .expect(
          403,
          '/apis/v1beta1/scheduledworkflows/a-random-swf-id endpoint is not meant for external usage.',
        );
    });

    it('does not reject similar apis', async () => {
      await request // use reportMetrics as runId to see if it can confuse route parsing
        .post(`/apis/v1beta1/runs/xxx-reportMetrics:archive`)
        .expect(200, 'KFP API is working');
    });

    it('proxies other run apis', async () => {
      await request
        .post(`/apis/v1beta1/runs/a-random-run-id:archive`)
        .expect(200, 'KFP API is working');
    });
  });
});
