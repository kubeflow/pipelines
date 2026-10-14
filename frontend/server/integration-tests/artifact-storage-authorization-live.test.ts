// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import express, { type Express } from 'express';
import { Client } from 'minio';
import { readFileSync } from 'node:fs';
import type { Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import requests from 'supertest';
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs } from '../configs.js';
import {
  getArtifactsAuthMiddleware,
  getArtifactsHandler,
  getArtifactsProxyHandler,
} from '../handlers/artifacts.js';
import { getAuthorizeFn } from '../helpers/auth.js';
import { getConfigMap } from '../k8s-helper.js';
import { listObjectsUnderPrefix } from '../minio-helper.js';

vi.mock('../k8s-helper.js', () => ({ getConfigMap: vi.fn(), getK8sSecret: vi.fn() }));

interface StorageFixture {
  endpoint: string;
  identities: Record<string, { accesskey: string; secretkey: string }>;
}

// Run test/seaweedfs/artifact_authorization_test.sh to provision an isolated store.
// Storage, IAM enforcement, middleware, proxy transport, and S3 clients are real.
// Kubernetes ConfigMap discovery and the namespace authorization API are fixtures;
// this does not exercise cluster networking, Istio, or a real API server SAR.
const fixturePath = process.env.KFP_LIVE_ARTIFACT_AUTH_FIXTURE;
describe.skipIf(!fixturePath)('storage-authorized artifacts with real tenant credentials', () => {
  const servers: Server[] = [];
  const targets = new Map<string, string>();
  const routed: string[] = [];
  let central: Express;
  let fixture: StorageFixture;
  let apiOrigin: string;
  let metadataLookups = 0;

  async function listen(app: Express): Promise<string> {
    const server = await new Promise<Server>((resolve) => {
      const started = app.listen(0, '127.0.0.1', () => resolve(started));
    });
    servers.push(server);
    return `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  }

  function sharedUI(mode: 'storage' | 'key-prefix'): Express {
    const options = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
      ENABLE_AUTHZ: 'true',
      ARTIFACTS_SERVICE_PROXY_ENABLED: 'true',
      KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE: mode,
      KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
    });
    const app = express();
    app.use(
      getArtifactsAuthMiddleware(
        getAuthorizeFn(options.auth, { apiServerAddress: apiOrigin }),
        true,
        'kubeflow-userid',
        apiOrigin,
        true,
        options.artifacts.authorizationMode,
      ),
      getArtifactsProxyHandler({
        enabled: true,
        allowedDomain: '127.0.0.1',
        namespacedServiceGetter: (namespace) => {
          routed.push(namespace);
          const target = targets.get(namespace);
          if (!target) throw new Error(`No fixture service for ${namespace}`);
          return target;
        },
      }),
      (_req, res) => res.status(503).send('No shared credential fallback'),
    );
    return app;
  }

  beforeAll(async () => {
    fixture = JSON.parse(readFileSync(fixturePath!, 'utf8')) as StorageFixture;
    const endpoint = new URL(fixture.endpoint);
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    for (const [namespace, credentials] of Object.entries(fixture.identities)) {
      const options = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
        FRONTEND_SERVER_NAMESPACE: namespace,
        MINIO_NAMESPACE: '',
        MINIO_HOST: endpoint.hostname,
        MINIO_PORT: endpoint.port,
        MINIO_SSL: 'false',
        MINIO_ACCESS_KEY: credentials.accesskey,
        MINIO_SECRET_KEY: credentials.secretkey,
        AWS_ACCESS_KEY_ID: credentials.accesskey,
        AWS_SECRET_ACCESS_KEY: credentials.secretkey,
        AWS_S3_ENDPOINT: fixture.endpoint,
        AWS_SSL: 'false',
        AWS_REGION: 'us-east-1',
      });
      const service = express();
      service.get(
        '/artifacts/get',
        getArtifactsHandler({
          artifactsConfigs: options.artifacts,
          useParameter: false,
          tryExtract: true,
          options,
        }),
      );
      service.get(
        '/artifacts/:source/:bucket/*',
        getArtifactsHandler({
          artifactsConfigs: options.artifacts,
          useParameter: true,
          tryExtract: false,
          options,
        }),
      );
      targets.set(namespace, await listen(service));
    }

    const api = express();
    api.use((req, res) => {
      if (req.path.endsWith('/auth')) {
        const namespace = new URL(req.originalUrl, 'http://fixture').searchParams.get('namespace');
        const user = req.header('kubeflow-userid');
        const allowed =
          (namespace === 'team-a' && user === 'alice') ||
          (namespace === 'team-b' && user === 'bob');
        res.status(allowed ? 200 : 403).json(allowed ? {} : { message: 'Access denied' });
        return;
      }
      // Model an attacker importing the victim URI into their own artifact record.
      // Record ownership must not turn into a storage grant; storage mode bypasses
      // the record lookup entirely, so no trusted record service is required.
      metadataLookups += 1;
      res.json({ artifacts: [{ artifact_id: 'attacker-owned-import', namespace: 'team-a' }] });
    });
    apiOrigin = await listen(api);
    central = sharedUI('storage');
  });

  beforeEach(() => {
    routed.length = 0;
    metadataLookups = 0;
  });

  afterAll(async () => {
    await Promise.all(
      servers.map(
        (server) =>
          new Promise<void>((resolve, reject) => {
            server.close((error) => (error ? reject(error) : resolve()));
            server.closeAllConnections();
          }),
      ),
    );
  });

  function read(key: string, options: Record<string, string> = {}) {
    return requests(central)
      .get('/artifacts/get')
      .set('kubeflow-userid', 'alice')
      .query({
        source: 'minio',
        bucket: 'mlpipeline',
        namespace: 'team-a',
        key,
        ...options,
      });
  }

  function content(response: requests.Response): string {
    return Buffer.isBuffer(response.body) ? response.body.toString() : response.text;
  }

  it.each([
    ['private-artifacts/team-a/model.txt', 'team-a current artifact'],
    ['private/team-a/model.txt', 'team-a existing private artifact'],
    ['model/customer-17-training/model.txt', 'team-a historical artifact'],
    ['shared/model.txt', 'administrator-shared artifact'],
  ])('serves storage-authorized root %s without a metadata dependency', async (key, expected) => {
    const response = await read(key).expect(200);
    expect(content(response)).toBe(expected);
    expect(routed).toEqual(['team-a']);
    expect(metadataLookups).toBe(0);
  });

  it('uses the second tenant identity for Bob without granting Alice access', async () => {
    const own = await requests(central)
      .get('/artifacts/get')
      .set('kubeflow-userid', 'bob')
      .query({
        source: 'minio',
        bucket: 'mlpipeline',
        namespace: 'team-b',
        key: 'model/customer-23-training/model.txt',
      })
      .expect(200);
    expect(content(own)).toBe('team-b historical secret');
    const other = await requests(central)
      .get('/artifacts/get')
      .set('kubeflow-userid', 'bob')
      .query({
        source: 'minio',
        bucket: 'mlpipeline',
        namespace: 'team-b',
        key: 'model/customer-17-training/model.txt',
      });
    expect(other.status).toBeGreaterThanOrEqual(400);
    expect(content(other)).toContain('Access Denied');
    expect(routed).toEqual(['team-b', 'team-b']);
  });

  it.each(['minio', 's3'])(
    'denies another tenant object at storage even with an imported URI (%s)',
    async (source) => {
      const response = await read('model/customer-23-training/model.txt', { source });
      expect(response.status).toBeGreaterThanOrEqual(400);
      expect(content(response)).toContain('Access Denied');
      expect(content(response)).not.toContain('team-b historical secret');
      expect(routed).toEqual(['team-a']);
      expect(metadataLookups).toBe(0);
    },
  );

  it('denies another tenant private object using the standard download route', async () => {
    const response = await requests(central)
      .get('/artifacts/minio/mlpipeline/private-artifacts/team-b/model.txt?namespace=team-a')
      .set('kubeflow-userid', 'alice');
    expect(response.status).toBeGreaterThanOrEqual(400);
    expect(content(response)).toContain('Access Denied');
    expect(content(response)).not.toContain('team-b secret artifact');
  });

  it('downloads existing historical objects through the standard route', async () => {
    const response = await requests(central)
      .get('/artifacts/minio/mlpipeline/model/customer-17-training/model.txt?namespace=team-a')
      .set('kubeflow-userid', 'alice')
      .expect(200);
    expect(content(response)).toBe('team-a historical artifact');
    expect(response.headers['content-disposition']).toMatch(/^attachment/);
  });

  it('rejects a claimed victim namespace before contacting its artifact service', async () => {
    await read('model/customer-23-training/model.txt', { namespace: 'team-b' }).expect(403);
    expect(routed).toEqual([]);
  });

  it('ignores browser-supplied credential configuration for the victim URI', async () => {
    const response = await read('model/customer-23-training/model.txt', {
      providerInfo: JSON.stringify({
        Provider: 's3',
        Params: {
          fromEnv: 'false',
          endpoint: fixture.endpoint,
          secretName: 'administrator',
          accessKeyKey: 'accesskey',
          secretKeyKey: 'secretkey',
        },
      }),
    });
    expect(response.status).toBeGreaterThanOrEqual(400);
    expect(content(response)).toContain('Access Denied');
  });

  async function list(prefix: string) {
    const endpoint = new URL(fixture.endpoint);
    const credentials = fixture.identities['team-a'];
    const client = new Client({
      endPoint: endpoint.hostname,
      port: Number(endpoint.port),
      useSSL: false,
      region: 'us-east-1',
      accessKey: credentials.accesskey,
      secretKey: credentials.secretkey,
    });
    const objects = [];
    for await (const object of listObjectsUnderPrefix(client, 'mlpipeline', prefix)) {
      objects.push(object);
    }
    return objects;
  }

  it('lists an authorized historical directory using tenant storage permissions', async () => {
    // 4.34 serves virtual directory markers as empty objects, while 4.48 emits
    // NoSuchKey for a directory without its slash. Exercise real listing directly
    // so this authorization check does not depend on directory-marker behavior.
    await expect(list('model/customer-17-training/directory/')).resolves.toEqual([
      {
        name: 'model/customer-17-training/directory/part.txt',
        size: Buffer.byteLength('team-a historical directory'),
      },
    ]);
  });

  it.each(['private-artifacts/team-b/', 'model/customer-23-training/', ''])(
    'denies another tenant or bucket-wide listing at storage (%s)',
    async (prefix) => {
      await expect(list(prefix)).rejects.toMatchObject({ code: 'AccessDenied' });
    },
  );
});
