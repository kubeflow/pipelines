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

import { vi, describe, it, expect, afterEach, beforeEach, SpyInstance } from 'vitest';
import express from 'express';
import * as fs from 'fs';
import { Server } from 'http';
import * as path from 'path';
import requests from 'supertest';
import { UIServer } from '../app.js';
import { loadConfigs } from '../configs.js';
import { TEST_ONLY as K8S_TEST_EXPORT } from '../k8s-helper.js';
import { buildQuery, commonSetup, mkTempDir } from './test-helper.js';

beforeEach(() => {
  vi.spyOn(global.console, 'info').mockImplementation(() => {});
  vi.spyOn(global.console, 'log').mockImplementation(() => {});
  vi.spyOn(global.console, 'debug').mockImplementation(() => {});
});

describe('/apps/tensorboard', () => {
  let app: UIServer;
  afterEach(async () => {
    if (app) {
      await app.close();
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

  let k8sGetCustomObjectSpy: SpyInstance;
  let k8sDeleteCustomObjectSpy: SpyInstance;
  let k8sCreateCustomObjectSpy: SpyInstance;
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
    k8sGetCustomObjectSpy = vi.spyOn(
      K8S_TEST_EXPORT.k8sV1CustomObjectClient,
      'getNamespacedCustomObject',
    );
    k8sDeleteCustomObjectSpy = vi.spyOn(
      K8S_TEST_EXPORT.k8sV1CustomObjectClient,
      'deleteNamespacedCustomObject',
    );
    k8sCreateCustomObjectSpy = vi.spyOn(
      K8S_TEST_EXPORT.k8sV1CustomObjectClient,
      'createNamespacedCustomObject',
    );
  });

  afterEach(async () => {
    if (kfpApiServer) {
      await new Promise<void>(resolve => kfpApiServer.close(() => resolve()));
      kfpApiServer = undefined as any;
    }
  });

  describe('get', () => {
    it('requires logdir for get tensorboard', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .get('/apps/tensorboard')
        .expect(400, 'logdir argument is required');
    });

    it('requires namespace for get tensorboard', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .get('/apps/tensorboard?logdir=some-log-dir')
        .expect(400, 'namespace argument is required');
    });

    it('does not crash with a weird query', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
      // The special case is that, decodeURIComponent('%2') throws an
      // exception, so this can verify handler doesn't do extra
      // decodeURIComponent on queries.
      const weirdLogDir = encodeURIComponent('%2');
      await requests(app.app)
        .get(`/apps/tensorboard?logdir=${weirdLogDir}&namespace=test-ns`)
        .expect(200);
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

    it('authorizes user requests from KFP auth api', async () => {
      const { receivedHeaders, host, port } = setupMockKfpApiService();
      app = new UIServer(
        loadConfigs(argv, {
          ENABLE_AUTHZ: 'true',
          ML_PIPELINE_SERVICE_PORT: `${port}`,
          ML_PIPELINE_SERVICE_HOST: host,
        }),
      );
      k8sGetCustomObjectSpy.mockImplementation(() => Promise.resolve(newGetTensorboardResponse()));
      await requests(app.app)
        .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
        .set('x-goog-authenticated-user-email', 'accounts.google.com:user@google.com')
        .expect(200);
      expect(receivedHeaders).toHaveLength(1);
      expect(receivedHeaders[0]).toMatchObject({
        accept: '*/*',
        host: 'localhost:3001',
        'x-goog-authenticated-user-email': 'accounts.google.com:user@google.com',
      });
    });

    it('uses configured KUBEFLOW_USERID_HEADER for user identity', async () => {
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
      await requests(app.app)
        .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
        .set('x-kubeflow-userid', 'user@kubeflow.org')
        .expect(200);
      expect(receivedHeaders).toHaveLength(1);
      expect(receivedHeaders[0]).toHaveProperty('x-kubeflow-userid', 'user@kubeflow.org');
    });

    it('rejects user requests when KFP auth api rejected', async () => {
      const errorSpy = vi.spyOn(console, 'error');
      errorSpy.mockImplementation(() => {});

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
      await requests(app.app)
        .get(`/apps/tensorboard?logdir=some-log-dir&namespace=test-ns`)
        .expect(
          401,
          'User is not authorized to GET VIEWERS in namespace test-ns: User xxx is not unauthorized to list viewers',
        );
      expect(errorSpy).toHaveBeenCalledTimes(1);
      expect(
        errorSpy,
      ).toHaveBeenCalledWith(
        'User is not authorized to GET VIEWERS in namespace test-ns: User xxx is not unauthorized to list viewers',
        ['unauthorized', 'callstack'],
      );
    });

    it('gets tensorboard url, version and image', async () => {
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

      await requests(app.app)
        .get(`/apps/tensorboard?logdir=${encodeURIComponent('log-dir-1')}&namespace=test-ns`)
        .expect(
          200,
          JSON.stringify({
            podAddress:
              'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
            tfVersion: '2.0.0',
            image: 'tensorflow:2.0.0',
          }),
        );
      expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
    });

    it('gets tensorboard url with custom cluster domain (defensive normalization)', async () => {
      // Test both with and without leading dot
      app = new UIServer(loadConfigs(argv, { CLUSTER_DOMAIN: 'cluster.test' }));
      k8sGetCustomObjectSpy.mockImplementation(() =>
        Promise.resolve(
          newGetTensorboardResponse({
            name: 'viewer-abcdefg',
            logDir: 'log-dir-1',
            tensorflowImage: 'tensorflow:2.0.0',
          }),
        ),
      );

      await requests(app.app)
        .get(`/apps/tensorboard?logdir=${encodeURIComponent('log-dir-1')}&namespace=test-ns`)
        .expect(
          200,
          JSON.stringify({
            podAddress:
              'http://viewer-abcdefg-service.test-ns.cluster.test:80/tensorboard/viewer-abcdefg/',
            tfVersion: '2.0.0',
            image: 'tensorflow:2.0.0',
          }),
        );
    });
  });

  describe('post (create)', () => {
    it('requires logdir', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .post('/apps/tensorboard')
        .expect(400, 'logdir argument is required');
    });

    it('requires namespace', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .post('/apps/tensorboard?logdir=some-log-dir')
        .expect(400, 'namespace argument is required');
    });

    it('requires tfversion or image', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .post('/apps/tensorboard?logdir=some-log-dir&namespace=test-ns')
        .expect(400, 'missing required argument: tfversion (tensorflow version) or image');
    });

    it('creates tensorboard viewer custom object and waits for it', async () => {
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
      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'log-dir-1',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          200,
          'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
        );
      expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sCreateCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "body": {
              "apiVersion": "kubeflow.org/v1beta1",
              "kind": "Viewer",
              "metadata": {
                "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
                "namespace": "test-ns",
              },
              "spec": {
                "podTemplateSpec": {
                  "spec": {
                    "containers": [
                      {},
                    ],
                  },
                },
                "tensorboardSpec": {
                  "logDir": "log-dir-1",
                  "tensorflowImage": "tensorflow/tensorflow:2.0.0",
                },
                "type": "tensorboard",
              },
            },
            "group": "kubeflow.org",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sGetCustomObjectSpy.mock.calls[1]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
    });

    it('creates tensorboard viewer with specified image', async () => {
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
      await requests(app.app)
        .post(
          `/apps/tensorboard${buildQuery({ logdir: 'log-dir-1', namespace: 'test-ns', image })}`,
        )
        .expect(200);
      expect(
        k8sCreateCustomObjectSpy.mock.calls[0][0].body.spec.tensorboardSpec.tensorflowImage,
      ).toEqual(image);
    });

    it('creates tensorboard viewer with exist volume', async () => {
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

      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'Series1:volume://tensorboard/log-dir-1,Series2:volume://tensorboard/log-dir-2',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          200,
          'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
        );
      expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sCreateCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "body": {
              "apiVersion": "kubeflow.org/v1beta1",
              "kind": "Viewer",
              "metadata": {
                "name": "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
                "namespace": "test-ns",
              },
              "spec": {
                "podTemplateSpec": {
                  "spec": {
                    "containers": [
                      {
                        "volumeMounts": [
                          {
                            "mountPath": "/logs",
                            "name": "tensorboard",
                          },
                          {
                            "mountPath": "/data",
                            "name": "data",
                            "subPath": "tensorboard",
                          },
                        ],
                      },
                    ],
                    "volumes": [
                      {
                        "name": "tensorboard",
                        "persistentVolumeClaim": {
                          "claimName": "logs",
                        },
                      },
                      {
                        "name": "data",
                        "persistentVolumeClaim": {
                          "claimName": "data",
                        },
                      },
                    ],
                  },
                },
                "tensorboardSpec": {
                  "logDir": "Series1:/logs/log-dir-1,Series2:/logs/log-dir-2",
                  "tensorflowImage": "tensorflow/tensorflow:2.0.0",
                },
                "type": "tensorboard",
              },
            },
            "group": "kubeflow.org",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sGetCustomObjectSpy.mock.calls[1]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-a800f945f0934d978f9cce9959b82ff44dac8493",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
    });

    it('creates tensorboard viewer with exist subPath volume', async () => {
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

      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'Series1:volume://data/tensorboard/log-dir-1,Series2:volume://data/tensorboard/log-dir-2',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          200,
          'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
        );
      expect(k8sGetCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sCreateCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "body": {
              "apiVersion": "kubeflow.org/v1beta1",
              "kind": "Viewer",
              "metadata": {
                "name": "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
                "namespace": "test-ns",
              },
              "spec": {
                "podTemplateSpec": {
                  "spec": {
                    "containers": [
                      {
                        "volumeMounts": [
                          {
                            "mountPath": "/logs",
                            "name": "tensorboard",
                          },
                          {
                            "mountPath": "/data",
                            "name": "data",
                            "subPath": "tensorboard",
                          },
                        ],
                      },
                    ],
                    "volumes": [
                      {
                        "name": "tensorboard",
                        "persistentVolumeClaim": {
                          "claimName": "logs",
                        },
                      },
                      {
                        "name": "data",
                        "persistentVolumeClaim": {
                          "claimName": "data",
                        },
                      },
                    ],
                  },
                },
                "tensorboardSpec": {
                  "logDir": "Series1:/data/log-dir-1,Series2:/data/log-dir-2",
                  "tensorflowImage": "tensorflow/tensorflow:2.0.0",
                },
                "type": "tensorboard",
              },
            },
            "group": "kubeflow.org",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
      expect(k8sGetCustomObjectSpy.mock.calls[1]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-82d7d06a6ecb1e4dcba66d06b884d6445a88e4ca",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
    });

    it('creates tensorboard viewer with not exist volume and return error', async () => {
      const errorSpy = vi.spyOn(console, 'error');
      errorSpy.mockImplementation(() => {});

      k8sGetCustomObjectSpy.mockImplementation(() => {
        return Promise.reject('Not found');
      });

      const tempPath = path.join(mkTempDir(), 'config.json');
      fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
      app = new UIServer(
        loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
      );

      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'volume://notexistvolume/logs/log-dir-1',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          500,
          `Failed to start Tensorboard app: Cannot find file "volume://notexistvolume/logs/log-dir-1" in pod "unknown": volume "notexistvolume" not configured`,
        );
      expect(errorSpy).toHaveBeenCalledTimes(1);
    });

    it('creates tensorboard viewer with not exist subPath volume mount and return error', async () => {
      const errorSpy = vi.spyOn(console, 'error');
      errorSpy.mockImplementation(() => {});

      k8sGetCustomObjectSpy.mockImplementation(() => {
        return Promise.reject('Not found');
      });

      const tempPath = path.join(mkTempDir(), 'config.json');
      fs.writeFileSync(tempPath, JSON.stringify(POD_TEMPLATE_SPEC));
      app = new UIServer(
        loadConfigs(argv, { VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH: tempPath }),
      );

      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'volume://data/notexit/mountnotexist/log-dir-1',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          500,
          `Failed to start Tensorboard app: Cannot find file "volume://data/notexit/mountnotexist/log-dir-1" in pod "unknown": volume "data" not mounted or volume "data" with subPath (which is prefix of notexit/mountnotexist/log-dir-1) not mounted`,
        );
      expect(errorSpy).toHaveBeenCalledTimes(1);
    });

    it('returns error when there is an existing tensorboard with different version', async () => {
      const errorSpy = vi.spyOn(console, 'error');
      errorSpy.mockImplementation(() => {});
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
      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'log-dir-1',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          500,
          `Failed to start Tensorboard app: There's already an existing tensorboard instance with a different version 2.1.0`,
        );
      expect(errorSpy).toHaveBeenCalledTimes(1);
    });

    it('returns existing pod address if there is an existing tensorboard with the same version', async () => {
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
      await requests(app.app)
        .post(
          `/apps/tensorboard?logdir=${encodeURIComponent(
            'log-dir-1',
          )}&namespace=test-ns&tfversion=2.0.0`,
        )
        .expect(
          200,
          'http://viewer-abcdefg-service.test-ns.svc.cluster.local:80/tensorboard/viewer-abcdefg/',
        );
    });
  });

  describe('delete', () => {
    it('requires logdir', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .delete('/apps/tensorboard')
        .expect(400, 'logdir argument is required');
    });

    it('requires namespace', async () => {
      app = new UIServer(loadConfigs(argv, {}));
      await requests(app.app)
        .delete('/apps/tensorboard?logdir=some-log-dir')
        .expect(400, 'namespace argument is required');
    });

    it('deletes tensorboard viewer custom object', async () => {
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
      await requests(app.app)
        .delete(`/apps/tensorboard?logdir=${encodeURIComponent('log-dir-1')}&namespace=test-ns`)
        .expect(200, 'Tensorboard deleted.');
      expect(k8sDeleteCustomObjectSpy.mock.calls[0]).toMatchInlineSnapshot(`
        [
          {
            "group": "kubeflow.org",
            "name": "viewer-5e1404e679e27b0f0b8ecee8fe515830eaa736c5",
            "namespace": "test-ns",
            "plural": "viewers",
            "version": "v1beta1",
          },
        ]
      `);
    });
  });
});
