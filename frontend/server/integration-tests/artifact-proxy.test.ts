import { vi, describe, it, expect, afterEach, beforeEach } from 'vitest';
import { UIServer } from '../app.js';
import { commonSetup, buildQuery } from './test-helper.js';
import requests from 'supertest';
import { loadConfigs } from '../configs.js';
import * as minioHelper from '../minio-helper.js';
import { PassThrough } from 'stream';
import express from 'express';
import { Server } from 'http';
import * as artifactsHandler from '../handlers/artifacts.js';

beforeEach(() => {
  vi.spyOn(global.console, 'info').mockImplementation(() => {});
  vi.spyOn(global.console, 'log').mockImplementation(() => {});
  vi.spyOn(global.console, 'debug').mockImplementation(() => {});
});

const commonParams = {
  source: 'minio',
  bucket: 'ml-pipeline',
  key: 'hello.txt',
};

describe('/artifacts/get namespaced proxy', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  function setupMinioArtifactDeps({ content }: { content: string }) {
    const getObjectStreamSpy = vi.spyOn(minioHelper, 'getObjectStream');
    const objStream = new PassThrough();
    objStream.end(content);
    getObjectStreamSpy.mockImplementationOnce(() => Promise.resolve(objStream));
  }

  let artifactServerInUserNamespace: Server;
  function setUpNamespacedArtifactService({
    namespace = 'any-ns',
    port = 3002,
  }: {
    namespace?: string;
    port?: number;
  }) {
    const receivedUrls: string[] = [];
    const artifactService = express();
    const response = `artifact service in ${namespace}`;
    artifactService.use((req, res) => {
      receivedUrls.push(req.url);
      res.status(200).send(response);
    });
    artifactServerInUserNamespace = artifactService.listen(port);
    const getArtifactServiceGetterSpy = vi
      .spyOn(artifactsHandler, 'getArtifactServiceGetter')
      .mockImplementation(() => () => `http://localhost:${port}`);
    return { receivedUrls, getArtifactServiceGetterSpy, response };
  }
  afterEach(async () => {
    if (artifactServerInUserNamespace) {
      await new Promise<void>(resolve => artifactServerInUserNamespace.close(() => resolve()));
      artifactServerInUserNamespace = undefined as any;
    }
  });

  it('is disabled by default', async () => {
    setupMinioArtifactDeps({ content: 'text-data' });
    const configs = loadConfigs(argv, {});
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'text-data');
  });

  it('proxies a request to namespaced artifact service', async () => {
    const { receivedUrls, getArtifactServiceGetterSpy } = setUpNamespacedArtifactService({
      namespace: 'ns2',
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'artifact service in ns2');
    expect(getArtifactServiceGetterSpy).toHaveBeenCalledWith({
      serviceName: 'artifact-svc',
      servicePort: 80,
      enabled: true,
    });
    expect(receivedUrls).toEqual(
      // url is the same, except namespace query is omitted
      ['/artifacts/get?source=minio&bucket=ml-pipeline&key=hello.txt'],
    );
  });

  it('proxies a download request to namespaced artifact service', async () => {
    const { receivedUrls, getArtifactServiceGetterSpy } = setUpNamespacedArtifactService({
      namespace: 'ns2',
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/artifacts/minio/ml-pipeline/hello.txt${buildQuery({
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'artifact service in ns2');
    expect(getArtifactServiceGetterSpy).toHaveBeenCalledWith({
      serviceName: 'artifact-svc',
      servicePort: 80,
      enabled: true,
    });
    expect(receivedUrls).toEqual(
      // url is the same, except namespace query is omitted
      ['/artifacts/minio/ml-pipeline/hello.txt'],
    );
  });

  it('does not proxy requests without namespace argument', async () => {
    setupMinioArtifactDeps({ content: 'text-data2' });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: undefined,
        })}`,
      )
      .expect(200, 'text-data2');
  });

  it('proxies a request with basePath too', async () => {
    const { receivedUrls, response } = setUpNamespacedArtifactService({});
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/pipeline/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns-any',
        })}`,
      )
      .expect(200, response);
    expect(receivedUrls).toEqual(
      // url is the same with base path, except namespace query is omitted
      ['/pipeline/artifacts/get?source=minio&bucket=ml-pipeline&key=hello.txt'],
    );
  });
});
