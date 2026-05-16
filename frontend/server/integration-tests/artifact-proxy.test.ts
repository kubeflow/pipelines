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
  async function setUpNamespacedArtifactService({ namespace = 'any-ns' }: { namespace?: string }) {
    const receivedUrls: string[] = [];
    const artifactService = express();
    const response = `artifact service in ${namespace}`;
    artifactService.use((req, res) => {
      receivedUrls.push(req.url);
      res.status(200).send(response);
    });
    artifactServerInUserNamespace = await new Promise<Server>((resolve, reject) => {
      const server = artifactService.listen(0, '127.0.0.1', () => resolve(server));
      server.on('error', reject);
    });
    const address = artifactServerInUserNamespace.address();
    if (!address || typeof address === 'string') {
      throw new Error('Expected artifact proxy test server to bind to a TCP port');
    }
    const getArtifactServiceGetterSpy = vi
      .spyOn(artifactsHandler, 'getArtifactServiceGetter')
      .mockImplementation(() => () => `http://127.0.0.1:${address.port}`);
    return { receivedUrls, getArtifactServiceGetterSpy, response };
  }
  afterEach(async () => {
    if (artifactServerInUserNamespace) {
      await new Promise<void>((resolve) => artifactServerInUserNamespace.close(() => resolve()));
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
    const { receivedUrls, getArtifactServiceGetterSpy } = await setUpNamespacedArtifactService({
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
    const { receivedUrls, getArtifactServiceGetterSpy } = await setUpNamespacedArtifactService({
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

  it('returns 400 for invalid namespace without leaking namespace value', async () => {
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);
    const res = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: '../../etc',
        })}`,
      )
      .expect(400);
    expect(res.text).not.toContain('../../etc');
    expect(res.text).not.toContain('stack');
  });

  it('proxies a request with basePath too', async () => {
    const { receivedUrls, response } = await setUpNamespacedArtifactService({});
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
