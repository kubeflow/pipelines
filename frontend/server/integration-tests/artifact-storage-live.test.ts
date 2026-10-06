// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import { afterAll, beforeAll, describe, expect, it } from 'vitest';
import { execFileSync } from 'node:child_process';
import { createServer, Server } from 'node:http';
import { AddressInfo } from 'node:net';
import { gzipSync } from 'node:zlib';
import { Client } from 'minio';
import requests from 'supertest';
import type { UIServer } from '../app.js';
import { commonSetup } from './test-helper.js';

// Opt-in: real Kubernetes, HTTP and S3 clients. No API/storage mocks.
const enabled = process.env.KFP_LIVE_ARTIFACT_STORAGE === 'true';
describe.skipIf(!enabled)('live standalone artifact storage migration', () => {
  const { argv } = commonSetup();
  const namespace = 'kfp-artifact-acceptance';
  const bucket = 'migration-fixture';
  const content = '<html><script>throw new Error("inert")</script>existing artifact</html>';
  const archive = gzipSync(Buffer.from('existing compressed artifact bytes'));
  const logs = 'existing archived pod logs\n';
  const port = Number(process.env.KFP_LIVE_S3_PORT || 8333);
  const origin = `http://127.0.0.1:${port}`;
  const alias = `http://localhost:${port}`;
  const apps: UIServer[] = [];
  let ServerClass: typeof import('../app.js').UIServer;
  let loadConfigs: typeof import('../configs.js').loadConfigs;
  let http: Server;
  let httpOrigin: string;
  let calls: string[] = [];
  const env = {
    FRONTEND_SERVER_NAMESPACE: namespace,
    MINIO_NAMESPACE: '',
    MINIO_HOST: '127.0.0.1',
    MINIO_PORT: String(port),
    MINIO_ACCESS_KEY: 'fixture',
    MINIO_SECRET_KEY: 'fixture-secret',
    AWS_ACCESS_KEY_ID: 'fixture',
    AWS_SECRET_ACCESS_KEY: 'fixture-secret',
    AWS_S3_ENDPOINT: origin,
    AWS_SSL: 'false',
  };
  function ui(extra: Record<string, string> = {}) {
    const app = new ServerClass(loadConfigs(argv, { ...env, ...extra }));
    apps.push(app);
    return requests(app.app);
  }
  function provider(endpoint: string) {
    return JSON.stringify({
      Provider: 's3',
      Params: {
        fromEnv: 'false',
        endpoint,
        disableSSL: 'true',
        region: 'us-east-1',
        secretName: 'artifact-store',
        accessKeyKey: 'accesskey',
        secretKeyKey: 'secretkey',
      },
    });
  }
  function workflow(name: string, endpoint: string) {
    execFileSync('kubectl', ['apply', '-f', '-'], {
      input: JSON.stringify({
        apiVersion: 'argoproj.io/v1alpha1',
        kind: 'Workflow',
        metadata: { name, namespace },
        spec: {},
        status: {
          artifactRepositoryRef: {
            artifactRepository: {
              archiveLogs: true,
              s3: {
                endpoint,
                insecure: true,
                bucket,
                accessKeySecret: { name: 'artifact-store', key: 'accesskey' },
                secretKeySecret: { name: 'artifact-store', key: 'secretkey' },
              },
            },
          },
          nodes: {
            [name]: { outputs: { artifacts: [{ name: 'main-logs', s3: { key: 'pod.log' } }] } },
          },
        },
      }),
      stdio: ['pipe', 'pipe', 'pipe'],
    });
  }
  function attachment(response: requests.Response) {
    expect(response.headers['content-disposition']).toMatch(/^attachment/);
    expect(response.headers['x-content-type-options']).toBe('nosniff');
  }
  beforeAll(async () => {
    expect(process.env.FRONTEND_SERVER_NAMESPACE).toBe(namespace);
    ServerClass = (await import('../app.js')).UIServer;
    loadConfigs = (await import('../configs.js')).loadConfigs;
    const client = new Client({
      endPoint: '127.0.0.1',
      port,
      useSSL: false,
      accessKey: 'fixture',
      secretKey: 'fixture-secret',
    });
    if (!(await client.bucketExists(bucket))) await client.makeBucket(bucket);
    await client.putObject(bucket, 'existing.html', content);
    await client.putObject(bucket, 'existing.gz', archive);
    await client.putObject(bucket, 'pod.log', logs);
    http = createServer((req, res) => {
      calls.push(req.url || '');
      if (req.url === '/reports/redirect') {
        res.writeHead(302, { Location: '/private/report.html' });
        res.end();
      } else if (req.url === '/reports/cross-origin') {
        res.writeHead(302, {
          Location: httpOrigin.replace('127.0.0.1', 'localhost') + '/reports/existing.html',
        });
        res.end();
      } else {
        res.setHeader('Content-Type', 'text/html');
        res.end(content);
      }
    });
    await new Promise<void>((resolve) => http.listen(0, '127.0.0.1', resolve));
    httpOrigin = `http://127.0.0.1:${(http.address() as AddressInfo).port}`;
    workflow('stock-archive', origin);
    workflow('alias-archive', alias);
    workflow('untrusted-archive', httpOrigin);
  }, 60000);
  afterAll(async () => {
    await Promise.all(apps.map((app) => app.close()));
    if (http)
      await new Promise<void>((resolve, reject) =>
        http.close((error) => (error ? reject(error) : resolve())),
      );
  });
  it('reads existing stock SeaweedFS HTML as an inert attachment', async () => {
    const response = await ui()
      .get('/artifacts/get')
      .query({ source: 'minio', bucket, key: 'existing.html', download: 'true' })
      .expect(200);
    attachment(response);
    expect(Buffer.from(response.body).toString()).toBe(content);
  });
  it('preserves original compressed download bytes', async () => {
    const response = await ui()
      .get('/artifacts/get')
      .query({ source: 'minio', bucket, key: 'existing.gz', download: 'true' })
      .expect(200);
    attachment(response);
    expect(response.body).toEqual(archive);
  });
  it('reads custom S3 configuration using the actual configured credentials', async () => {
    const response = await ui()
      .get('/artifacts/get')
      .query({ source: 's3', bucket, key: 'existing.html', download: 'true' })
      .expect(200);
    expect(Buffer.from(response.body).toString()).toBe(content);
  });
  it('requires explicit trust of an alias, then reads the unchanged object with Kubernetes Secret credentials', async () => {
    const query = {
      source: 's3',
      bucket,
      key: 'existing.html',
      namespace,
      providerInfo: provider(alias),
      download: 'true',
    };
    const rejected = await ui().get('/artifacts/get').query(query).expect(400);
    expect(rejected.text).toContain('ALLOWED_ARTIFACT_ENDPOINTS');
    const response = await ui({ ALLOWED_ARTIFACT_ENDPOINTS: alias })
      .get('/artifacts/get')
      .query(query)
      .expect(200);
    attachment(response);
    expect(Buffer.from(response.body).toString()).toBe(content);
  });
  it('rejects an untrusted S3 origin before making any outbound request', async () => {
    calls = [];
    const response = await ui()
      .get('/artifacts/get')
      .query({
        source: 's3',
        bucket,
        key: 'existing.html',
        namespace,
        providerInfo: provider(httpOrigin),
      })
      .expect(400);
    expect(response.text).toContain('ALLOWED_ARTIFACT_ENDPOINTS');
    expect(calls).toEqual([]);
  });
  it('requires HTTP_BASE_URL, then fetches the unchanged original HTTP URI', async () => {
    const query = {
      source: 'http',
      bucket: new URL(httpOrigin).host,
      key: 'reports/existing.html',
    };
    calls = [];
    const rejected = await ui().get('/artifacts/get').query(query).expect(400);
    expect(rejected.text).toContain('HTTP_BASE_URL');
    expect(calls).toEqual([]);
    const response = await ui({ HTTP_BASE_URL: httpOrigin + '/reports/' })
      .get('/artifacts/get')
      .query(query)
      .expect(200);
    attachment(response);
    expect(response.text || Buffer.from(response.body).toString()).toBe(content);
    expect(calls).toEqual(['/reports/existing.html']);
  });
  it.each(['private/existing.html', 'reports/../private/existing.html'])(
    'rejects HTTP path outside the configured base: %s',
    async (key) => {
      calls = [];
      await ui({ HTTP_BASE_URL: httpOrigin + '/reports/' })
        .get('/artifacts/get')
        .query({ source: 'http', bucket: new URL(httpOrigin).host, key })
        .expect(400);
      expect(calls).toEqual([]);
    },
  );
  it.each(['redirect', 'cross-origin'])(
    'does not follow a redirect outside its approved boundary: %s',
    async (target) => {
      calls = [];
      await ui({ HTTP_BASE_URL: httpOrigin + '/reports/' })
        .get('/artifacts/get')
        .query({ source: 'http', bucket: new URL(httpOrigin).host, key: 'reports/' + target })
        .expect(400);
      expect(calls).toEqual(['/reports/' + target]);
    },
  );
  it.each([
    ['stock-archive', ''],
    ['alias-archive', alias],
  ])('reads retained workflow-status log archives: %s', async (name, allowlist) => {
    const response = await ui({ ALLOWED_ARTIFACT_ENDPOINTS: allowlist })
      .get('/k8s/pod/logs')
      .query({ podname: name, podnamespace: namespace })
      .expect(200);
    attachment(response);
    expect(response.text).toBe(logs);
  });
  it('rejects an untrusted archived-log origin without contacting it', async () => {
    calls = [];
    const response = await ui()
      .get('/k8s/pod/logs')
      .query({ podname: 'untrusted-archive', podnamespace: namespace })
      .expect(500);
    expect(response.text).toContain('ALLOWED_ARTIFACT_ENDPOINTS');
    expect(calls).toEqual([]);
  });
});
