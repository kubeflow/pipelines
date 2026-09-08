import { vi, describe, it, expect, afterEach, beforeEach } from 'vitest';
import { UIServer } from '../app.js';
import { commonSetup, buildQuery } from './test-helper.js';
import requests from 'supertest';
import { loadConfigs } from '../configs.js';
import * as minioHelper from '../minio-helper.js';
import { PassThrough } from 'stream';
import express, { type RequestHandler } from 'express';
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
  async function setUpNamespacedArtifactService({
    namespace = 'any-ns',
    responseHeaders = {},
    requestHandler,
  }: {
    namespace?: string;
    responseHeaders?: Record<string, string>;
    requestHandler?: RequestHandler;
  }) {
    const receivedUrls: string[] = [];
    const artifactService = express();
    const response = `artifact service in ${namespace}`;
    artifactService.use((req, res, next) => {
      receivedUrls.push(req.url);
      res.set(responseHeaders);
      next();
    });
    artifactService.use(requestHandler ?? ((_req, res) => res.status(200).send(response)));
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
    const response = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200);
    expect(response.body.toString()).toBe('artifact service in ns2');
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

  it('overrides unsafe response headers from the namespaced artifact service', async () => {
    await setUpNamespacedArtifactService({
      namespace: 'ns2',
      responseHeaders: {
        'Content-Disposition': 'inline',
        'Content-Security-Policy': "default-src 'self'",
        'Content-Type': 'application/javascript',
        'Access-Control-Allow-Credentials': 'true',
        'Access-Control-Allow-Origin': '*',
        'Clear-Site-Data': '"cookies", "storage"',
        Link: '</tenant.js>; rel=preload; as=script',
        Location: 'https://tenant.example/redirect',
        Refresh: '0; url=https://tenant.example/redirect',
        'Set-Cookie': 'session=tenant-controlled; Path=/',
        'X-Tenant-Header': 'unsafe-value',
        'X-Content-Type-Options': 'unsafe-value',
      },
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns2',
        })}`,
      )
      .expect(200);

    expect(response.headers['content-type']).toBe('application/octet-stream');
    expect(response.headers['content-disposition']).toBe('attachment');
    expect(response.headers['x-content-type-options']).toBe('nosniff');
    expect(response.headers['access-control-allow-credentials']).toBeUndefined();
    expect(response.headers['access-control-allow-origin']).toBeUndefined();
    expect(response.headers['clear-site-data']).toBeUndefined();
    expect(response.headers['content-security-policy']).toBeUndefined();
    expect(response.headers.link).toBeUndefined();
    expect(response.headers.location).toBeUndefined();
    expect(response.headers.refresh).toBeUndefined();
    expect(response.headers['set-cookie']).toBeUndefined();
    expect(response.headers['x-powered-by']).toBe('Express');
    expect(response.headers['x-tenant-header']).toBeUndefined();
  });

  it('preserves a safe proxy download filename while forcing attachment', async () => {
    await setUpNamespacedArtifactService({
      namespace: 'ns2',
      responseHeaders: {
        'Content-Disposition':
          'attachment; filename="directory.tar.gz"; filename*=UTF-8\'\'directory.tar.gz',
      },
    });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(`/artifacts/get${buildQuery({ ...commonParams, namespace: 'ns2' })}`)
      .expect(200);

    expect(response.headers['content-disposition']).toBe(
      'attachment; filename="directory.tar.gz"; filename*=UTF-8\'\'directory.tar.gz',
    );
  });

  it('preserves RFC 8187 filenames with a language tag', async () => {
    await setUpNamespacedArtifactService({
      namespace: 'ns2',
      responseHeaders: {
        'Content-Disposition': "attachment; filename*=UTF-8'en'report%20final.csv",
      },
    });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(`/artifacts/get${buildQuery({ ...commonParams, namespace: 'ns2' })}`)
      .expect(200);

    expect(response.headers['content-disposition']).toBe(
      'attachment; filename="report_final.csv"; filename*=UTF-8\'\'report%20final.csv',
    );
  });

  it('preserves RFC 8187 ISO-8859-1 filenames', async () => {
    await setUpNamespacedArtifactService({
      namespace: 'ns2',
      responseHeaders: {
        'Content-Disposition': "attachment; filename*=ISO-8859-1''caf%E9.txt",
      },
    });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(`/artifacts/get${buildQuery({ ...commonParams, namespace: 'ns2' })}`)
      .expect(200);

    expect(response.headers['content-disposition']).toBe(
      'attachment; filename="caf_.txt"; filename*=UTF-8\'\'caf%C3%A9.txt',
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
    const response = await requests(app.app)
      .get(
        `/artifacts/minio/ml-pipeline/hello.txt${buildQuery({
          namespace: 'ns2',
        })}`,
      )
      .expect(200);
    expect(response.body.toString()).toBe('artifact service in ns2');
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

  it('translates query downloads to the legacy tenant route without normalizing keys', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          key: 'reports/../secret.txt',
          namespace: 'ns2',
          download: 'true',
        })}`,
      )
      .expect(200);
    expect(response.body.toString()).toBe('artifact service in ns2');

    expect(receivedUrls).toEqual(['/artifacts/minio/ml-pipeline/reports%2F..%2Fsecret.txt']);
  });

  it('returns raw archives from a previous-version tenant artifact handler', async () => {
    const rawArchive = Buffer.from('raw archive bytes');
    const receivedKeys: string[] = [];
    const previousVersionHandler = express.Router();
    previousVersionHandler.get('/artifacts/get', (_req, res) => {
      res.status(200).send('extracted first archive member');
    });
    previousVersionHandler.get('/artifacts/:source/:bucket/*', (req, res) => {
      receivedKeys.push(req.params[0]);
      res.status(200).end(rawArchive);
    });
    const { receivedUrls } = await setUpNamespacedArtifactService({
      namespace: 'ns2',
      requestHandler: previousVersionHandler,
    });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          key: 'archives/run.tar.gz',
          namespace: 'ns2',
          providerInfo: '{"Provider":"minio"}',
          download: 'true',
        })}`,
      )
      .expect(200);

    expect(response.body).toEqual(rawArchive);
    expect(receivedUrls).toEqual([
      '/artifacts/minio/ml-pipeline/archives%2Frun.tar.gz' +
        '?providerInfo=%7B%22Provider%22%3A%22minio%22%7D',
    ]);
    expect(receivedKeys).toEqual(['archives/run.tar.gz']);
  });

  it('preserves providerInfo when proxying a download request (issue #13717)', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({
      namespace: 'ns2',
    });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    const response = await requests(app.app)
      .get(
        `/artifacts/s3/mlpipeline/model${buildQuery({
          namespace: 'ns2',
          providerInfo: '{"Provider":"s3"}',
        })}`,
      )
      .expect(200);
    expect(response.body.toString()).toBe('artifact service in ns2');
    expect(receivedUrls).toEqual(
      // same url, except the namespace query is omitted and providerInfo is kept
      ['/artifacts/s3/mlpipeline/model?providerInfo=%7B%22Provider%22%3A%22s3%22%7D'],
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

  it.each(['source', 'bucket', 'key', 'providerInfo', 'namespace', 'peek', 'download'])(
    'rejects ambiguous %s query parameters before proxying',
    async (parameterName) => {
      const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns-a' });
      const configs = loadConfigs(argv, {
        ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
      });
      app = new UIServer(configs);
      const query = new URLSearchParams({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello.txt',
        providerInfo: '{}',
        namespace: 'ns-a',
        peek: '10',
        download: 'true',
      });
      query.append(parameterName, 'duplicate');

      await requests(app.app)
        .get(`/artifacts/get?${query.toString()}`)
        .expect(400, `${parameterName} must be a single string value`);
      expect(receivedUrls).toEqual([]);
    },
  );

  it('proxies a request with basePath too', async () => {
    const { receivedUrls, response } = await setUpNamespacedArtifactService({});
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    const proxiedResponse = await requests(app.app)
      .get(
        `/pipeline/artifacts/get${buildQuery({
          ...commonParams,
          namespace: 'ns-any',
        })}`,
      )
      .expect(200);
    expect(proxiedResponse.body.toString()).toBe(response);
    expect(receivedUrls).toEqual(
      // url is the same with base path, except namespace query is omitted
      ['/pipeline/artifacts/get?source=minio&bucket=ml-pipeline&key=hello.txt'],
    );
  });
});
