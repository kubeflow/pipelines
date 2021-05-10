// Copyright 2021 The Kubeflow Authors
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
import * as fs from 'fs';
import { Server } from 'http';
import * as path from 'path';
import requests from 'supertest';
import { UIServer } from '../app';
import { loadConfigs } from '../configs';
import { TEST_ONLY as K8S_TEST_EXPORT } from '../k8s-helper';
import { buildQuery, commonSetup, mkTempDir } from './test-helper';

beforeEach(() => {
  jest.spyOn(global.console, 'info').mockImplementation();
  jest.spyOn(global.console, 'log').mockImplementation();
  jest.spyOn(global.console, 'debug').mockImplementation();
});

describe('/apps/tensorboard', () => {
  let app: UIServer;
  afterEach(() => {
    if (app) {
      app.close();
    }
  });
  const tagName = '1.0.0';
  const commitHash = 'abcdefg';
  const { argv } = commonSetup({ tagName, commitHash });

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
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
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
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
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
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
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
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
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

    it('gets tensorboard url, version and image', done => {
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
            image: 'tensorflow:2.0.0',
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

    it('requires tfversion or image', done => {
      app = new UIServer(loadConfigs(argv, {}));
      requests(app.start())
        .post('/apps/tensorboard?logdir=some-log-dir&namespace=test-ns')
        .expect(400, 'missing required argument: tfversion (tensorflow version) or image', done);
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

    it('creates tensorboard viewer with specified image', done => {
      let getRequestCount = 0;
      const image = 'gcr.io/deeplearning-platform-release/tf2-cpu.2-4';
      k8sGetCustomObjectSpy.mockImplementation(() => {
        ++getRequestCount;
        switch (getRequestCount) {
          case 1:
            return Promise.reject('Not found');
          case 2:
            return Promise.resolve(
              newGetTensorboardResponse({
                tensorflowImage: image,
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
          `/apps/tensorboard${buildQuery({ logdir: 'log-dir-1', namespace: 'test-ns', image })}`,
        )
        .expect(200, err => {
          expect(
            k8sCreateCustomObjectSpy.mock.calls[0][4].spec.tensorboardSpec.tensorflowImage,
          ).toEqual(image);
          done(err);
        });
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

      const tempPath = path.join(mkTempDir(), 'config.json');
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

      const tempPath = path.join(mkTempDir(), 'config.json');
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

      const tempPath = path.join(mkTempDir(), 'config.json');
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

      const tempPath = path.join(mkTempDir(), 'config.json');
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
