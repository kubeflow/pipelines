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
import express from 'express';

import fetch from 'node-fetch';
import requests from 'supertest';

import { UIServer } from './app';
import { loadConfigs } from './configs';
import { TEST_ONLY as K8S_TEST_EXPORT } from './k8s-helper';
import { Server } from 'http';
import { commonSetup } from './integration-tests/test-helper';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

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

  describe('/apps/tensorboard', () => {
    const POD_TEMPLATE_SPEC = {
      spec: {
        containers: [
          {
            volumeMounts: [
              {
                name: 'tensorboard',
                mountPath: '/logs',
              },
              {
                name: 'data',
                subPath: 'tensorboard',
                mountPath: '/data',
              },
            ],
          },
        ],
        volumes: [
          {
            name: 'tensorboard',
            persistentVolumeClaim: {
              claimName: 'logs',
            },
          },
          {
            name: 'data',
            persistentVolumeClaim: {
              claimName: 'data',
            },
          },
        ],
      },
    };

    let k8sGetCustomObjectSpy: jest.SpyInstance;
    let k8sDeleteCustomObjectSpy: jest.SpyInstance;
    let k8sCreateCustomObjectSpy: jest.SpyInstance;
    let kfpApiServer: Server;

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

    afterEach(() => {
      if (kfpApiServer) {
        kfpApiServer.close();
      }
    });

    describe('get', () => {
      it('requires logdir for get tensorboard', done => {
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .get('/apps/tensorboard')
          .expect(400, 'logdir argument is required', done);
      });

      it('requires namespace for get tensorboard', done => {
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .get('/apps/tensorboard?logdir=some-log-dir')
          .expect(400, 'namespace argument is required', done);
      });

      it('does not crash with a weird query', done => {
        app = new UIServer(loadConfigs(argv, {}));
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(newGetTensorboardResponse()),
        );
        // The special case is that, decodeURIComponent('%2') throws an
        // exception, so this can verify handler doesn't do extra
        // decodeURIComponent on queries.
        const weirdLogDir = encodeURIComponent('%2');
        requests(app.start())
          .get(`/apps/tensorboard?logdir=${weirdLogDir}&namespace=test-ns`)
          .expect(200, done);
      });

      function setupMockKfpApiService({ port = 3001 }: { port?: number } = {}) {
        const receivedHeaders: any[] = [];
        kfpApiServer = express()
          .get('/apis/v1beta1/auth', (req, res) => {
            receivedHeaders.push(req.headers);
            res.status(200).send('{}'); // Authorized
          })
          .listen(port);
        return { receivedHeaders, host: 'localhost', port };
      }

      it('authorizes user requests from KFP auth api', done => {
        const { receivedHeaders, host, port } = setupMockKfpApiService();
        app = new UIServer(
          loadConfigs(argv, {
            ENABLE_AUTHZ: 'true',
            ML_PIPELINE_SERVICE_PORT: `${port}`,
            ML_PIPELINE_SERVICE_HOST: host,
          }),
        );
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(newGetTensorboardResponse()),
        );
        requests(app.start())
          .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
          .set('x-goog-authenticated-user-email', 'accounts.google.com:user@google.com')
          .expect(200, err => {
            expect(receivedHeaders).toHaveLength(1);
            expect(receivedHeaders[0]).toMatchInlineSnapshot(`
              Object {
                "accept": "*/*",
                "accept-encoding": "gzip,deflate",
                "connection": "close",
                "host": "localhost:3001",
                "user-agent": "node-fetch/1.0 (+https://github.com/bitinn/node-fetch)",
                "x-goog-authenticated-user-email": "accounts.google.com:user@google.com",
              }
            `);
            done(err);
          });
      });

      it('uses configured KUBEFLOW_USERID_HEADER for user identity', done => {
        const { receivedHeaders, host, port } = setupMockKfpApiService();
        app = new UIServer(
          loadConfigs(argv, {
            ENABLE_AUTHZ: 'true',
            KUBEFLOW_USERID_HEADER: 'x-kubeflow-userid',
            ML_PIPELINE_SERVICE_PORT: `${port}`,
            ML_PIPELINE_SERVICE_HOST: host,
          }),
        );
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(newGetTensorboardResponse()),
        );
        requests(app.start())
          .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
          .set('x-kubeflow-userid', 'user@kubeflow.org')
          .expect(200, err => {
            expect(receivedHeaders).toHaveLength(1);
            expect(receivedHeaders[0]).toHaveProperty('x-kubeflow-userid', 'user@kubeflow.org');
            done(err);
          });
      });

      it('rejects user requests when KFP auth api rejected', done => {
        const errorSpy = jest.spyOn(console, 'error');
        errorSpy.mockImplementation();

        const apiServerPort = 3001;
        kfpApiServer = express()
          .get('/apis/v1beta1/auth', (_, res) => {
            res.status(400).send(
              JSON.stringify({
                error: 'User xxx is not unauthorized to list viewers',
                details: ['unauthorized', 'callstack'],
              }),
            );
          })
          .listen(apiServerPort);
        app = new UIServer(
          loadConfigs(argv, {
            ENABLE_AUTHZ: 'true',
            ML_PIPELINE_SERVICE_PORT: `${apiServerPort}`,
            ML_PIPELINE_SERVICE_HOST: 'localhost',
          }),
        );
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(newGetTensorboardResponse()),
        );
        requests(app.start())
          .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
          .expect(
            401,
            'User is not authorized to GET VIEWERS in namespace test-ns: User xxx is not unauthorized to list viewers',
            err => {
              expect(errorSpy).toHaveBeenCalledTimes(1);
              expect(
                errorSpy,
              ).toHaveBeenCalledWith(
                'User is not authorized to GET VIEWERS in namespace test-ns: User xxx is not unauthorized to list viewers',
                ['unauthorized', 'callstack'],
              );
              done(err);
            },
          );
      });

      it('gets tensorboard url and version', done => {
        app = new UIServer(loadConfigs(argv, {}));
        k8sGetCustomObjectSpy.mockImplementation(() =>
          Promise.resolve(
            newGetTensorboardResponse({
              name: 'viewer-abcdefg',
              logDir: 'log-dir-1',
              tensorflowImage: 'tensorflow:2.0.0',
            }),
          ),
        );

        requests(app.start())
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
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .post('/apps/tensorboard')
          .expect(400, 'logdir argument is required', done);
      });

      it('requires namespace', done => {
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .post('/apps/tensorboard?logdir=some-log-dir')
          .expect(400, 'namespace argument is required', done);
      });

      it('requires tfversion', done => {
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .post('/apps/tensorboard?logdir=some-log-dir&namespace=test-ns')
          .expect(400, 'tfversion (tensorflow version) argument is required', done);
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

        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
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

      it('creates tensorboard viewer with exist volume', done => {
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
                  logDir: 'Series1:/logs/log-dir-1,Series2:/logs/log-dir-2',
                  tensorflowImage: 'tensorflow:2.0.0',
                }),
              );
            default:
              throw new Error('only expected to be called twice in this test');
          }
        });
        k8sCreateCustomObjectSpy.mockImplementation(() => Promise.resolve());

        const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'config.json');
        fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
        app = new UIServer(
          loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
        );

        requests(app.start())
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'Series1:volume://tensorboard/log-dir-1,Series2:volume://tensorboard/log-dir-2',
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
                  "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
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
                      "name": "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
                      "namespace": "test-ns",
                    },
                    "spec": Object {
                      "podTemplateSpec": Object {
                        "spec": Object {
                          "containers": Array [
                            Object {
                              "volumeMounts": Array [
                                Object {
                                  "mountPath": "/logs",
                                  "name": "tensorboard",
                                },
                                Object {
                                  "mountPath": "/data",
                                  "name": "data",
                                  "subPath": "tensorboard",
                                },
                              ],
                            },
                          ],
                          "volumes": Array [
                            Object {
                              "name": "tensorboard",
                              "persistentVolumeClaim": Object {
                                "claimName": "logs",
                              },
                            },
                            Object {
                              "name": "data",
                              "persistentVolumeClaim": Object {
                                "claimName": "data",
                              },
                            },
                          ],
                        },
                      },
                      "tensorboardSpec": Object {
                        "logDir": "Series1:/logs/log-dir-1,Series2:/logs/log-dir-2",
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
                  "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
                ]
              `);
              done(err);
            },
          );
      });

      it('creates tensorboard viewer with exist subPath volume', done => {
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
                  logDir: 'Series1:/data/log-dir-1,Series2:/data/log-dir-2',
                  tensorflowImage: 'tensorflow:2.0.0',
                }),
              );
            default:
              throw new Error('only expected to be called twice in this test');
          }
        });
        k8sCreateCustomObjectSpy.mockImplementation(() => Promise.resolve());

        const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'config.json');
        fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
        app = new UIServer(
          loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
        );

        requests(app.start())
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'Series1:volume://data/tensorboard/log-dir-1,Series2:volume://data/tensorboard/log-dir-2',
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
                  "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
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
                      "name": "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
                      "namespace": "test-ns",
                    },
                    "spec": Object {
                      "podTemplateSpec": Object {
                        "spec": Object {
                          "containers": Array [
                            Object {
                              "volumeMounts": Array [
                                Object {
                                  "mountPath": "/logs",
                                  "name": "tensorboard",
                                },
                                Object {
                                  "mountPath": "/data",
                                  "name": "data",
                                  "subPath": "tensorboard",
                                },
                              ],
                            },
                          ],
                          "volumes": Array [
                            Object {
                              "name": "tensorboard",
                              "persistentVolumeClaim": Object {
                                "claimName": "logs",
                              },
                            },
                            Object {
                              "name": "data",
                              "persistentVolumeClaim": Object {
                                "claimName": "data",
                              },
                            },
                          ],
                        },
                      },
                      "tensorboardSpec": Object {
                        "logDir": "Series1:/data/log-dir-1,Series2:/data/log-dir-2",
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
                  "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
                ]
              `);
              done(err);
            },
          );
      });

      it('creates tensorboard viewer with not exist volume and return error', done => {
        const errorSpy = jest.spyOn(console, 'error');
        errorSpy.mockImplementation();

        k8sGetCustomObjectSpy.mockImplementation(() => {
          return Promise.reject('Not found');
        });

        const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'config.json');
        fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
        app = new UIServer(
          loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
        );

        requests(app.start())
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'volume://notexistvolume/logs/log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            500,
            `Failed to start Tensorboard app: Cannot find file "volume://notexistvolume/logs/log-dir-1" in pod "unknown": volume "notexistvolume" not configured`,
            err => {
              expect(errorSpy).toHaveBeenCalledTimes(1);
              done(err);
            },
          );
      });

      it('creates tensorboard viewer with not exist subPath volume mount and return error', done => {
        const errorSpy = jest.spyOn(console, 'error');
        errorSpy.mockImplementation();

        k8sGetCustomObjectSpy.mockImplementation(() => {
          return Promise.reject('Not found');
        });

        const tempPath = path.join(fs.mkdtempSync(os.tmpdir()), 'config.json');
        fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
        app = new UIServer(
          loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
        );

        requests(app.start())
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'volume://data/notexit/mountnotexist/log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            500,
            `Failed to start Tensorboard app: Cannot find file "volume://data/notexit/mountnotexist/log-dir-1" in pod "unknown": volume "data" not mounted or volume "data" with subPath (which is prefix of notexit/mountnotexist/log-dir-1) not mounted`,
            err => {
              expect(errorSpy).toHaveBeenCalledTimes(1);
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

        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .post(
            `/apps/tensorboard?logdir=${encodeURIComponent(
              'log-dir-1',
            )}&namespace=test-ns&tfversion=2.0.0`,
          )
          .expect(
            500,
            `Failed to start Tensorboard app: There's already an existing tensorboard instance with a different version 2.1.0`,
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

        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
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
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .delete('/apps/tensorboard')
          .expect(400, 'logdir argument is required', done);
      });

      it('requires namespace', done => {
        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
          .delete('/apps/tensorboard?logdir=some-log-dir')
          .expect(400, 'namespace argument is required', done);
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

        app = new UIServer(loadConfigs(argv, {}));
        requests(app.start())
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
