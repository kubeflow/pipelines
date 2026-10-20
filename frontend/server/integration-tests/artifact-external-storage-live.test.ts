// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import express, { type Express } from 'express';
import { readFileSync, writeFileSync } from 'node:fs';
import type { Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import requests from 'supertest';
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs } from '../configs.js';
import {
  ARTIFACT_STORAGE_PROXY_PREFIX,
  getArtifactReceiverNamespaceMiddleware,
  getArtifactStorageReceiverMiddleware,
  getArtifactsAuthMiddleware,
  getArtifactsHandler,
  getArtifactsProxyHandler,
} from '../handlers/artifacts.js';
import type { ArtifactStorageAuthorization } from '../helpers/artifact-authorization.js';
import { getAuthorizeFn } from '../helpers/auth.js';
import {
  createExternalArtifactClient,
  getExternalArtifactStorageBinding,
  type ExternalArtifactStorageConfig,
} from '../helpers/external-artifact-storage.js';
import { getConfigMap, getK8sSecret } from '../k8s-helper.js';
import { listObjectsUnderPrefix } from '../minio-helper.js';

vi.mock('../k8s-helper.js', () => ({ getConfigMap: vi.fn(), getK8sSecret: vi.fn() }));

interface ExternalFixture {
  namespaces: Record<string, ExternalArtifactStorageConfig>;
  controlURL: string;
  controlToken: string;
}
interface Refresh {
  generation: number;
  credentialFingerprint: string;
  expiration: string;
}

// The wrapper provisions real OIDC, STS, expiring sessions, bucket policies, and
// broker-written profile files. Namespace SAR and Kubernetes observations are
// fixtures; this suite does not establish live controller or CNI acceptance.
const fixturePath = process.env.KFP_EXTERNAL_ARTIFACT_FIXTURE;
describe.skipIf(!fixturePath)('external storage with real rotating STS credentials', () => {
  const servers: Server[] = [];
  const targets = new Map<string, string>();
  let fixture: ExternalFixture;
  let central: Express;
  let apiOrigin: string;
  let records: Record<string, ArtifactStorageAuthorization>;
  let metadataLookups = 0;

  async function listen(app: Express): Promise<string> {
    const server = await new Promise<Server>((resolve) => {
      const started = app.listen(0, '127.0.0.1', () => resolve(started));
    });
    servers.push(server);
    return `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  }

  function sharedUI(mode: 'auto' | 'storage' | 'key-prefix'): Express {
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
          const target = targets.get(namespace);
          if (!target) throw new Error('Missing fixture receiver');
          return target;
        },
      }),
      (_req, res) => res.status(503).send('No shared credential fallback'),
    );
    return app;
  }

  async function refresh(namespace = 'team-a'): Promise<Refresh> {
    const response = await fetch(`${fixture.controlURL}/refresh/${namespace}`, {
      method: 'POST',
      headers: { 'X-Fixture-Control': fixture.controlToken },
    });
    expect(response.status).toBe(200);
    return response.json() as Promise<Refresh>;
  }

  beforeAll(async () => {
    fixture = JSON.parse(readFileSync(fixturePath!, 'utf8')) as ExternalFixture;
    vi.mocked(getConfigMap).mockResolvedValue([
      undefined,
      { message: 'not found', statusCode: 404 },
    ]);
    for (const [namespace, config] of Object.entries(fixture.namespaces)) {
      const endpoint = new URL(config.storage.endpoint);
      const options = loadConfigs(['node', 'server.js', '/tmp', '3000'], {
        FRONTEND_SERVER_NAMESPACE: namespace,
        KFP_ARTIFACT_NAMESPACE_UID: config.namespaceUID,
        KFP_ARTIFACT_EXTERNAL_STORAGE_CONFIG: JSON.stringify(config),
        // Deliberately give the ordinary client broad fixture credentials. The
        // approved route must never fall back to these on denial or expiration.
        AWS_S3_ENDPOINT: config.storage.endpoint,
        AWS_REGION: 'us-east-1',
        AWS_SSL: 'false',
        AWS_ACCESS_KEY_ID: 'fixture-admin',
        AWS_SECRET_ACCESS_KEY: 'fixture-admin-secret',
        MINIO_HOST: endpoint.hostname,
        MINIO_PORT: endpoint.port,
        MINIO_SSL: 'false',
        MINIO_ACCESS_KEY: 'fixture-admin',
        MINIO_SECRET_KEY: 'fixture-admin-secret',
      });
      const receiver = express();
      receiver.get(
        `${ARTIFACT_STORAGE_PROXY_PREFIX}/*`,
        getArtifactStorageReceiverMiddleware(options),
      );
      receiver.get('/artifacts/*', getArtifactReceiverNamespaceMiddleware(namespace, false));
      receiver.get(
        '/artifacts/get',
        getArtifactsHandler({
          artifactsConfigs: options.artifacts,
          useParameter: false,
          tryExtract: true,
          options,
        }),
      );
      targets.set(namespace, await listen(receiver));
    }
    const api = express();
    api.use((req, res) => {
      if (req.path.endsWith('/auth')) {
        const namespace = new URL(req.originalUrl, 'http://fixture').searchParams.get('namespace');
        const user = req.header('kubeflow-userid');
        const allowed =
          (user === 'alice' && namespace === 'team-a') ||
          (user === 'bob' && namespace === 'team-b');
        res.status(allowed ? 200 : 403).json({});
        return;
      }
      metadataLookups += 1;
      res.json({ artifacts: [{ artifact_id: 'imported-victim-uri', namespace: 'team-a' }] });
    });
    apiOrigin = await listen(api);
  });

  beforeEach(async () => {
    metadataLookups = 0;
    records = Object.fromEntries(
      Object.entries(fixture.namespaces).map(([namespace, config]) => [
        namespace,
        {
          ...config,
          version: 2,
          namespace,
          state: 'ready',
          credentialBinding: getExternalArtifactStorageBinding(namespace, config),
        },
      ]),
    );
    vi.mocked(getK8sSecret).mockImplementation(async (name, key, namespace) => {
      expect(namespace).toBe('kubeflow');
      expect(key).toBe('authorization.json');
      const record = records[name.replace('kfp-artifact-storage-', '')];
      if (!record) throw { code: 404 };
      return JSON.stringify(record);
    });
    await refresh();
    await refresh('team-b');
    central = sharedUI('auto');
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

  function read(key = 'model/customer-17-training/model.txt', query: Record<string, string> = {}) {
    return requests(central)
      .get('/artifacts/get')
      .set('kubeflow-userid', 'alice')
      .query({
        source: 's3',
        bucket: 'external-artifacts',
        namespace: 'team-a',
        key,
        providerInfo: JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true' } }),
        ...query,
      });
  }

  function legacyRead(
    key = 'model/customer-17-training/model.txt',
    query: Record<string, string> = {},
  ) {
    return requests(targets.get('team-a')!)
      .get('/artifacts/get')
      .set('x-kfp-artifact-namespace', 'team-a')
      .query({
        source: 's3',
        bucket: 'external-artifacts',
        key,
        providerInfo: JSON.stringify({ Provider: 's3', Params: { fromEnv: 'true' } }),
        ...query,
      });
  }

  function content(response: requests.Response): string {
    return Buffer.isBuffer(response.body) ? response.body.toString() : response.text;
  }

  async function list(prefix: string) {
    const client = await createExternalArtifactClient(fixture.namespaces['team-a']);
    const items = [];
    for await (const item of listObjectsUnderPrefix(client, 'external-artifacts', prefix))
      items.push(item);
    return items;
  }

  it.each(['auto', 'storage'] as const)('preserves existing roots through %s', async (mode) => {
    central = sharedUI(mode);
    expect(content(await read().expect(200))).toBe('team-a historical artifact');
    expect(metadataLookups).toBe(0);
  });

  it('preserves the old shared UI route without weakening storage isolation', async () => {
    expect(content(await legacyRead().expect(200))).toBe('team-a historical artifact');
    const denied = await legacyRead('model/customer-23-training/model.txt').expect(500);
    expect(content(denied)).not.toContain('team-b historical artifact');
  });

  it('pins legacy requests to the namespace store and ignores browser authority', async () => {
    const response = await legacyRead(undefined, {
      providerInfo: JSON.stringify({
        Provider: 's3',
        Params: { fromEnv: 'false', endpoint: 'http://127.0.0.1:1', secretName: 'victim-secret' },
      }),
    }).expect(200);
    expect(content(response)).toBe('team-a historical artifact');
    await legacyRead(undefined, { namespace: 'team-b' }).expect(403);
    await legacyRead('model/customer-17-training/model.txt?endpoint=http://127.0.0.1:1').expect(
      400,
    );
  });

  it('reproduces the historical root rejection without an external approval', async () => {
    delete records['team-a'];
    await read().expect(403);
  });

  it('keeps receiver and identity binding unchanged across multiple real STS renewals', async () => {
    const target = targets.get('team-a');
    const initialRecord = JSON.stringify(records['team-a']);
    const fingerprints = new Set<string>();
    for (let generation = 0; generation < 3; generation++) {
      const session = await refresh();
      fingerprints.add(session.credentialFingerprint);
      expect(content(await read().expect(200))).toBe('team-a historical artifact');
      const denied = await read('model/customer-23-training/model.txt').expect(500);
      expect(content(denied)).not.toContain('team-b historical artifact');
      await expect(list('model/customer-23-training/')).rejects.toMatchObject({
        code: 'AccessDenied',
      });
      expect(targets.get('team-a')).toBe(target);
      expect(JSON.stringify(records['team-a'])).toBe(initialRecord);
    }
    expect(fingerprints.size).toBe(3);
  });

  it('denies other tenant objects and namespace impersonation without broader credentials', async () => {
    const response = await read('model/customer-23-training/model.txt');
    expect(response.status).toBe(500);
    expect(content(response)).not.toContain('team-b historical artifact');
    await read('model/customer-23-training/model.txt', { namespace: 'team-b' }).expect(403);
    expect(metadataLookups).toBe(0);
  });

  it('enforces real directory-list permissions for historical prefixes', async () => {
    await expect(list('model/customer-17-training/directory/')).resolves.toEqual([
      {
        name: 'model/customer-17-training/directory/part.txt',
        size: Buffer.byteLength('team-a historical artifact'),
      },
    ]);
    await expect(list('model/customer-23-training/')).rejects.toMatchObject({
      code: 'AccessDenied',
    });
    await expect(list('')).rejects.toMatchObject({ code: 'AccessDenied' });
  });

  it('ignores browser endpoints and credential selectors in the approved path', async () => {
    const response = await read(undefined, {
      providerInfo: JSON.stringify({
        Provider: 's3',
        Params: {
          fromEnv: 'false',
          endpoint: 'http://127.0.0.1:1',
          secretName: 'victim-secret',
          accessKeyKey: 'access-key',
          secretKeyKey: 'secret-key',
        },
      }),
    }).expect(200);
    expect(content(response)).toBe('team-a historical artifact');
    expect(getK8sSecret).not.toHaveBeenCalledWith(
      'victim-secret',
      expect.anything(),
      expect.anything(),
    );
  });

  it('fails closed when credentials are absent and recovers without restarting', async () => {
    const file = fixture.namespaces['team-a'].identity.credentialFile;
    writeFileSync(file, '[artifact-reader]\n');
    await read().expect(503);
    await legacyRead().expect(503);
    await refresh();
    expect(content(await read().expect(200))).toBe('team-a historical artifact');
  });

  it('rejects an actually expired session then reads the broker-renewed profile', async () => {
    const session = await refresh();
    expect(content(await read().expect(200))).toBe('team-a historical artifact');
    await new Promise((resolve) =>
      setTimeout(resolve, Math.max(0, Date.parse(session.expiration) - Date.now()) + 1000),
    );
    const expired = await read();
    expect(expired.status).toBe(500);
    expect(content(expired)).not.toContain('team-a historical artifact');
    await refresh();
    expect(content(await read().expect(200))).toBe('team-a historical artifact');
  }, 25_000);

  it('denies buckets outside the approved destination', async () => {
    await read(undefined, { bucket: 'another-bucket' }).expect(403);
  });

  it('rejects a newly approved identity until its receiver is updated', async () => {
    const config = structuredClone(fixture.namespaces['team-a']);
    config.identity.principal = 'arn:aws:iam::role/replacement';
    records['team-a'] = {
      ...config,
      version: 2,
      namespace: 'team-a',
      state: 'ready',
      credentialBinding: getExternalArtifactStorageBinding('team-a', config),
    };
    await read().expect(503);
  });

  it('rejects an approved template revision that differs from the running receiver', async () => {
    const config = structuredClone(fixture.namespaces['team-a']);
    config.configurationRevision = 'f'.repeat(64);
    records['team-a'] = {
      ...config,
      version: 2,
      namespace: 'team-a',
      state: 'ready',
      credentialBinding: getExternalArtifactStorageBinding('team-a', config),
    };
    await read().expect(503);
  });

  it('stops delegating while an approval is pending or revoked', async () => {
    records['team-a'] = {
      version: 2,
      namespace: 'team-a',
      namespaceUID: fixture.namespaces['team-a'].namespaceUID,
      state: 'pending',
    };
    await read().expect(503);
  });
});
