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
import { getConfigMap } from '../k8s-helper.js';
import { TEST_ONLY as launcherConfigTestOnly } from '../helpers/launcher-config.js';
import {
  buildArtifactCoordinateUri,
  resolveArtifactCoordinates,
} from '../helpers/artifact-coordinates.js';

vi.mock('../k8s-helper.js', () => ({
  getArgoWorkflow: vi.fn(),
  getConfigMap: vi.fn().mockResolvedValue([undefined, { message: 'not found' }]),
  getK8sSecret: vi.fn(),
  getPod: vi.fn(),
  getPodLogs: vi.fn(),
  getServerNamespace: vi.fn(),
}));

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

  beforeEach(() => {
    launcherConfigTestOnly.clearLauncherConfigurationCache();
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
  });

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

  it('translates query downloads to the legacy tenant route with canonical multi-segment keys', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);

    const response = await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          ...commonParams,
          key: 'reports/daily/report.txt',
          namespace: 'ns2',
          download: 'true',
        })}`,
      )
      .expect(200);
    expect(response.body.toString()).toBe('artifact service in ns2');

    expect(receivedUrls).toEqual(['/artifacts/minio/ml-pipeline/reports/daily/report.txt']);
  });

  it.each([
    { key: 'archives/run.tar.gz', keyEncoding: 'storage', storageKey: 'archives/run.tar.gz' },
    {
      key: 'archives%20dir/run:1.tar.gz',
      keyEncoding: 'uri',
      storageKey: 'archives dir/run:1.tar.gz',
    },
    {
      key: 'archives%20dir/run:1.tar.gz',
      keyEncoding: 'storage',
      storageKey: 'archives%20dir/run:1.tar.gz',
    },
  ])(
    'returns raw archives from a previous-version tenant for $keyEncoding key $key',
    async ({ key, keyEncoding, storageKey }) => {
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
            key,
            keyEncoding,
            namespace: 'ns2',
            providerInfo: '{"Provider":"minio"}',
            download: 'true',
          })}`,
        )
        .expect(200);

      expect(response.body).toEqual(rawArchive);
      expect(receivedUrls).toEqual([`/artifacts/minio/ml-pipeline/${encodeURI(storageKey)}`]);
      expect(receivedKeys).toEqual([storageKey]);
    },
  );

  it.each([
    {
      key: 'models%20dir/run:1.tar.gz',
      keyEncoding: 'uri',
      uriKey: undefined,
      storageKey: 'models dir/run:1.tar.gz',
    },
    {
      key: 'models%20dir/run:1.tar.gz',
      keyEncoding: 'storage',
      uriKey: 'models%2520dir/run:1.tar.gz',
      storageKey: 'models%20dir/run:1.tar.gz',
    },
  ])(
    'returns raw archives through an upgraded tenant for $keyEncoding keys',
    async ({ key, keyEncoding, uriKey, storageKey }) => {
      const rawArchive = Buffer.from('unchanged archive bytes');
      const readObject = vi
        .spyOn(minioHelper, 'getObjectStream')
        .mockImplementation(async (options) => {
          options.onTransformationDetermined?.(false);
          const stream = new PassThrough();
          stream.end(rawArchive);
          return stream;
        });
      const tenantConfigs = loadConfigs(argv, {});
      const tenant = express.Router();
      const identities: string[] = [];
      tenant.get(
        '/artifacts/:source/:bucket/*',
        (req, _res, next) => {
          const coordinates = resolveArtifactCoordinates(req);
          expect(coordinates).toBeTruthy();
          identities.push(buildArtifactCoordinateUri(coordinates!));
          next();
        },
        artifactsHandler.getArtifactsHandler({
          artifactsConfigs: tenantConfigs.artifacts,
          options: tenantConfigs,
          tryExtract: false,
          useParameter: true,
        }),
      );
      await setUpNamespacedArtifactService({ namespace: 'ns2', requestHandler: tenant });
      app = new UIServer(loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' }));
      const response = await requests(app.app)
        .get(
          `/artifacts/get${buildQuery({
            ...commonParams,
            key,
            keyEncoding,
            uriKey,
            artifactUriQuery: 'anonymous=true',
            namespace: 'ns2',
            download: 'true',
          })}`,
        )
        .expect(200);
      expect(response.body).toEqual(rawArchive);
      expect(response.headers['content-disposition']).toContain('attachment');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(readObject).toHaveBeenCalledWith(
        expect.objectContaining({ key: storageKey, tryExtract: false }),
      );
      expect(identities).toEqual([`minio://ml-pipeline/${uriKey ?? key}?anonymous=true`]);
    },
  );

  it('retains volume identity when the legacy download path removes harmless dot segments', async () => {
    const tenant = express.Router();
    tenant.get('/artifacts/:source/:bucket/*', (req, res) => {
      const coordinates = resolveArtifactCoordinates(req);
      expect(coordinates?.key).toBe('outputs/report.txt');
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        'volume://artifact/outputs/./report.txt',
      );
      res.end('volume bytes');
    });
    await setUpNamespacedArtifactService({ namespace: 'ns2', requestHandler: tenant });
    app = new UIServer(loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' }));
    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          source: 'volume',
          bucket: 'artifact',
          key: 'outputs/./report.txt',
          namespace: 'ns2',
          download: 'true',
        })}`,
      )
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('volume bytes'));
  });

  it('proxies authenticated volume access to the namespace-isolated artifact service', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
      text: () => Promise.resolve(''),
    } as Response);
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
      KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
      KUBEFLOW_USERID_PREFIX: '',
      ML_PIPELINE_SERVICE_HOST: 'localhost',
      ML_PIPELINE_SERVICE_PORT: '8888',
    });
    configs.auth.enabled = true;
    app = new UIServer(configs);

    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          source: 'volume',
          bucket: 'artifact',
          key: 'outputs/result.txt',
          namespace: 'ns2',
          providerInfo: '{"Provider":"s3","Params":{"endpoint":"https://attacker.example"}}',
        })}`,
      )
      .set('kubeflow-userid', 'user@example.com')
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('artifact service in ns2'));

    expect(receivedUrls).toEqual([
      '/artifacts/get?source=volume&bucket=artifact&key=outputs%2Fresult.txt',
    ]);
  });

  it('discards caller-supplied providerInfo when proxying a download request', async () => {
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
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('artifact service in ns2'));
    expect(receivedUrls).toEqual(['/artifacts/s3/mlpipeline/model']);
  });

  it('replaces caller providerInfo with trusted launcher-root query settings', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    vi.mocked(getConfigMap).mockResolvedValueOnce([
      {
        data: {
          defaultPipelineRoot:
            's3://mlpipeline/models?endpoint=https%3A%2F%2Ftrusted.example&region=trusted',
        },
      },
      undefined,
    ]);
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    await requests(app.app)
      .get(
        `/artifacts/s3/mlpipeline/models/model${buildQuery({
          namespace: 'ns2',
          providerInfo: '{"Provider":"s3","Params":{"endpoint":"https://attacker.example"}}',
        })}`,
      )
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('artifact service in ns2'));

    const received = new URL(receivedUrls[0], 'http://artifact.test');
    expect(JSON.parse(received.searchParams.get('providerInfo') || '')).toEqual({
      Provider: 's3',
      Params: {
        endpoint: 'https://trusted.example',
        fromEnv: 'true',
        nativeQuery: 'true',
        region: 'trusted',
      },
    });
  });

  it('applies trusted launcher-root settings to URI-escaped preview keys', async () => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    vi.mocked(getConfigMap).mockResolvedValueOnce([
      {
        data: {
          defaultPipelineRoot:
            's3://mlpipeline/models%20dir?endpoint=https%3A%2F%2Ftrusted.example&region=trusted',
        },
      },
      undefined,
    ]);
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);

    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          source: 's3',
          bucket: 'mlpipeline',
          key: 'models%20dir/model',
          keyEncoding: 'uri',
          namespace: 'ns2',
        })}`,
      )
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('artifact service in ns2'));

    const received = new URL(receivedUrls[0], 'http://artifact.test');
    expect(received.searchParams.get('key')).toBe('models%20dir/model');
    expect(JSON.parse(received.searchParams.get('providerInfo') || '')).toEqual({
      Provider: 's3',
      Params: {
        endpoint: 'https://trusted.example',
        fromEnv: 'true',
        nativeQuery: 'true',
        region: 'trusted',
      },
    });
  });

  it.each([
    ['providers YAML is malformed', [{ data: { providers: 's3: [unterminated' } }, undefined]],
    [
      'the ConfigMap read fails',
      [
        undefined,
        {
          additionalInfo: { code: 403, reason: 'Forbidden' },
          message: 'Could not read kfp-launcher',
        },
      ],
    ],
    [
      'the provider configuration is invalid',
      [
        {
          data: {
            providers: `
s3:
  Overrides:
    - bucketName: another-bucket
      credentials:
        fromEnv: true
`,
          },
        },
        undefined,
      ],
    ],
  ] as const)('forwards without providerInfo when %s', async (_description, configMapResult) => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns2' });
    vi.mocked(getConfigMap).mockResolvedValueOnce(
      configMapResult as unknown as Awaited<ReturnType<typeof getConfigMap>>,
    );
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_NAME: 'artifact-svc',
      ARTIFACTS_SERVICE_PROXY_PORT: '80',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);

    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          source: 's3',
          bucket: 'ml-pipeline',
          key: 'hello.txt',
          namespace: 'ns2',
        })}`,
      )
      .expect(200)
      .expect((response) => expect(response.body.toString()).toBe('artifact service in ns2'));

    expect(receivedUrls).toEqual(['/artifacts/get?source=s3&bucket=ml-pipeline&key=hello.txt']);
  });

  it('does not proxy requests without namespace argument', async () => {
    setupMinioArtifactDeps({ content: 'text-data2' });
    const configs = loadConfigs(argv, { ARTIFACTS_SERVICE_PROXY_ENABLED: 'true' });
    app = new UIServer(configs);
    await requests(app.app)
      .get(
        `/artifacts/get${buildQuery({
          source: 'minio',
          bucket: 'mlpipeline',
          key: 'v2/artifacts/hello.txt',
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

  it.each([
    'source',
    'bucket',
    'key',
    'keyEncoding',
    'uriKey',
    'download',
    'artifactUriQuery',
    'providerInfo',
    'namespace',
    'peek',
  ])('rejects ambiguous %s query parameters before proxying', async (parameterName) => {
    const { receivedUrls } = await setUpNamespacedArtifactService({ namespace: 'ns-a' });
    const configs = loadConfigs(argv, {
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
    });
    app = new UIServer(configs);
    const query = new URLSearchParams({
      source: 'minio',
      bucket: 'ml-pipeline',
      key: 'hello.txt',
      keyEncoding: 'storage',
      uriKey: 'hello.txt',
      download: 'true',
      artifactUriQuery: 'region=first',
      providerInfo: '{}',
      namespace: 'ns-a',
      peek: '10',
    });
    query.append(parameterName, 'duplicate');

    await requests(app.app)
      .get(`/artifacts/get?${query.toString()}`)
      .expect(400, `${parameterName} must be a single string value`);
    expect(receivedUrls).toEqual([]);
  });

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
