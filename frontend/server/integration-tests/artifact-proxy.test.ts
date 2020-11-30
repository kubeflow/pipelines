import { UIServer } from '../app';
import { commonSetup, buildQuery } from './test-helper';
import requests from 'supertest';
import { loadConfigs } from '../configs';
import * as minioHelper from '../minio-helper';
import { PassThrough } from 'stream';
import express from 'express';
import { Server } from 'http';
import * as artifactsHandler from '../handlers/artifacts';

beforeEach(() => {
  jest.spyOn(global.console, 'info').mockImplementation();
  jest.spyOn(global.console, 'log').mockImplementation();
  jest.spyOn(global.console, 'debug').mockImplementation();
});

const commonParams = {
  source: 'minio',
  bucket: 'ml-pipeline',
  key: 'hello.txt',
};

describe('/artifacts/get namespaced proxy', () => {
  let app: UIServer;
  const { argv } = commonSetup();

  afterEach(() => {
    if (app) {
      app.close();
    }
  });

  function setupMinioArtifactDeps({ content }: { content: string }) {
    const getObjectStreamSpy = jest.spyOn(minioHelper, 'getObjectStream');
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
    artifactService.all('/*', (req, res) => {
      receivedUrls.push(req.url);
      res.status(200).send(response);
    });
    artifactServerInUserNamespace = artifactService.listen(port);
    const getArtifactServiceGetterSpy = jest
      .spyOn(artifactsHandler, 'getArtifactServiceGetter')
      .mockImplementation(() => () => `http://localhost:${port}`);
    return { receivedUrls, getArtifactServiceGetterSpy, response };
  }
  afterEach(() => {
    if (artifactServerInUserNamespace) {
      artifactServerInUserNamespace.close();
    }
  });

  it('is disabled by default', done => {
    setupMinioArtifactDeps({ content: 'text-data' });
    const configs = loadConfigs(argv, {});
    app = new UIServer(configs);
    requests(app.start())
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'text-data', done);
  });

  it('proxies a request to namespaced artifact service', done => {
    const { receivedUrls, getArtifactServiceGetterSpy } = setUpNamespacedArtifactService({
      namespace: 'ns2',
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    requests(app.start())
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'artifact service in ns2', err => {
        expect(getArtifactServiceGetterSpy).toHaveBeenCalledWith({
          serviceName: 'artifact-svc',
          servicePort: 80,
          enabled: true,
        });
        expect(receivedUrls).toEqual(
          // url is the same, except namespace query is omitted
          ['/artifacts/get?source=minio&bucket=ml-pipeline&key=hello.txt'],
        );
        done(err);
      });
  });

  it('proxies a download request to namespaced artifact service', done => {
    const { receivedUrls, getArtifactServiceGetterSpy } = setUpNamespacedArtifactService({
      namespace: 'ns2',
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    requests(app.start())
      .get(
        `/artifacts/minio/ml-pipeline/hello.txt${buildQuery({
          namespace: 'ns2',
        })}`,
      )
      .expect(200, 'artifact service in ns2', err => {
        expect(getArtifactServiceGetterSpy).toHaveBeenCalledWith({
          serviceName: 'artifact-svc',
          servicePort: 80,
          enabled: true,
        });
        expect(receivedUrls).toEqual(
          // url is the same, except namespace query is omitted
          ['/artifacts/minio/ml-pipeline/hello.txt'],
        );
        done(err);
      });
  });

  it('does not proxy requests without namespace argument', done => {
    setupMinioArtifactDeps({ content: 'text-data2' });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);
    requests(app.start())
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: undefined,
        })}`,
      )
      .expect(200, 'text-data2', done);
  });

  it('proxies a request with basePath too', done => {
    const { receivedUrls, response } = setUpNamespacedArtifactService({});
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    requests(app.start())
      .get(
        `/pipeline/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns-any',
        })}`,
      )
      .expect(200, response, err => {
        expect(receivedUrls).toEqual(
          // url is the same with base path, except namespace query is omitted
          ['/pipeline/artifacts/get?source=minio&bucket=ml-pipeline&key=hello.txt'],
        );
        done(err);
      });
  });
});
