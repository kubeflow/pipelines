// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// https://www.apache.org/licenses/LICENSE-2.0

import express from 'express';
import type { Server } from 'node:http';
import { Readable } from 'node:stream';
import type { Client as MinioClient } from 'minio';
import requests from 'supertest';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs } from '../configs.js';
import {
  ARTIFACT_STORAGE_PROXY_PREFIX,
  getArtifactReceiverNamespaceMiddleware,
  getArtifactStorageReceiverMiddleware,
  getArtifactsAuthMiddleware,
  getArtifactsHandler,
  getArtifactsProxyHandler,
} from '../handlers/artifacts.js';
import {
  createExternalArtifactClient,
  getExternalArtifactStorageBinding,
  type ExternalArtifactStorageConfig,
} from '../helpers/external-artifact-storage.js';
import { getK8sSecret } from '../k8s-helper.js';
import { createMinioClient } from '../minio-helper.js';

vi.mock('../k8s-helper.js', () => ({ getK8sSecret: vi.fn(), getConfigMap: vi.fn() }));
vi.mock('../helpers/external-artifact-storage.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../helpers/external-artifact-storage.js')>()),
  createExternalArtifactClient: vi.fn(),
}));
vi.mock('../minio-helper.js', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../minio-helper.js')>()),
  createMinioClient: vi.fn(),
}));

describe('approved external bucket routing', () => {
  const config: ExternalArtifactStorageConfig = {
    namespaceUID: 'uid-a',
    configurationRevision: 'a'.repeat(64),
    storage: {
      source: 's3',
      bucket: 'team.artifacts',
      endpoint: 'https://storage.example',
      region: 'us-east-1',
    },
    identity: {
      kind: 'shared-credentials-file',
      principal: 'broker:team-a',
      credentialFile: '/var/run/broker/credentials',
      profile: 'artifacts',
    },
  };
  let receiver: Server;
  let endpoint: string;
  const getObject = vi.fn();
  const listObjectsV2Query = vi.fn();
  const sharedFetch = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    getObject.mockImplementation(async (_bucket: string, key: string) => {
      if (key.endsWith('directory') || key.endsWith('directory/'))
        throw Object.assign(new Error('NoSuchKey'), { code: 'NoSuchKey' });
      return Readable.from(['existing artifact']);
    });
    listObjectsV2Query.mockResolvedValue({
      objects: [{ name: 'existing/directory/model.txt', size: 17 }],
      isTruncated: false,
      nextContinuationToken: '',
    });
    vi.mocked(createExternalArtifactClient).mockResolvedValue({
      getObject,
      listObjectsV2Query,
    } as unknown as MinioClient);
    vi.mocked(getK8sSecret).mockResolvedValue(
      JSON.stringify({
        ...config,
        version: 2,
        namespace: 'team-a',
        state: 'ready',
        credentialBinding: getExternalArtifactStorageBinding('team-a', config),
      }),
    );
    const options = loadConfigs(['node', 'server.js', '/tmp'], {
      FRONTEND_SERVER_NAMESPACE: 'team-a',
      KFP_ARTIFACT_NAMESPACE_UID: config.namespaceUID,
      KFP_ARTIFACT_EXTERNAL_STORAGE_CONFIG: JSON.stringify(config),
    });
    const app = express();
    app.get(`${ARTIFACT_STORAGE_PROXY_PREFIX}/*`, getArtifactStorageReceiverMiddleware(options));
    app.get('/artifacts/*', getArtifactReceiverNamespaceMiddleware('team-a', false));
    for (const [route, useParameter] of [
      ['/artifacts/get', false],
      ['/artifacts/:source/:bucket/*', true],
    ] as const)
      app.get(
        route,
        getArtifactsHandler({
          options,
          artifactsConfigs: options.artifacts,
          useParameter,
          tryExtract: false,
        }),
      );
    receiver = await new Promise<Server>((resolve) => {
      const server = app.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = receiver.address();
    if (!address || typeof address === 'string') throw new Error('Expected TCP listener');
    endpoint = `http://127.0.0.1:${address.port}`;
  });

  afterEach(async () => {
    await new Promise<void>((resolve) => receiver.close(() => resolve()));
  });

  function sharedUI() {
    const app = express();
    app.use(
      getArtifactsAuthMiddleware(
        async () => undefined,
        true,
        'user',
        'http://api.invalid',
        true,
        'auto',
      ),
      getArtifactsProxyHandler({
        enabled: true,
        allowedDomain: '^127\\.0\\.0\\.1$',
        namespacedServiceGetter: () => endpoint,
      }),
      (req, res) => {
        sharedFetch(req.url);
        res.status(500).send('Shared credentials must not be used');
      },
    );
    return app;
  }

  function route(kind: string, bucket = config.storage.bucket, key = 'existing/model.txt') {
    if (kind === 'raw') return `/artifacts/s3/${bucket}/${key}?namespace=team-a`;
    return `/artifacts/get?${new URLSearchParams({
      source: 's3',
      bucket,
      key,
      namespace: 'team-a',
      ...(kind === 'preview' ? { peek: '256' } : { download: 'true' }),
    })}`;
  }

  describe.each(['shared UI', 'receiver'])('%s', (hop) => {
    it.each(['raw', 'preview', 'download'])(
      'serves %s access to the approved dotted bucket',
      async (kind) => {
        const response = await requests(hop === 'shared UI' ? sharedUI() : endpoint)
          .get(route(kind))
          .set('user', 'alice')
          .expect(200);
        expect(response.text || response.body.toString()).toBe('existing artifact');
        expect(getObject).toHaveBeenCalledWith(config.storage.bucket, 'existing/model.txt');
        expect(createExternalArtifactClient).toHaveBeenCalledWith(config);
        expect(createMinioClient).not.toHaveBeenCalled();
        expect(sharedFetch).not.toHaveBeenCalled();
      },
    );
    it.each(['preview', 'download'])(
      'serves directory %s through the approved dotted bucket',
      async (kind) => {
        await requests(hop === 'shared UI' ? sharedUI() : endpoint)
          .get(route(kind, config.storage.bucket, 'existing/directory'))
          .set('user', 'alice')
          .expect(200);
        expect(listObjectsV2Query).toHaveBeenCalledWith(
          config.storage.bucket,
          'existing/directory/',
          '',
          '',
          expect.any(Number),
          '',
        );
        expect(createMinioClient).not.toHaveBeenCalled();
        expect(sharedFetch).not.toHaveBeenCalled();
      },
    );
    it('rejects a different dotted bucket before using credentials', async () => {
      await requests(hop === 'shared UI' ? sharedUI() : endpoint)
        .get(route('preview', 'other.artifacts'))
        .set('user', 'alice')
        .expect(403);
      expect(createExternalArtifactClient).not.toHaveBeenCalled();
      expect(createMinioClient).not.toHaveBeenCalled();
      expect(sharedFetch).not.toHaveBeenCalled();
    });
  });

  it('points pending external approval to controller, rollout, and central configuration', async () => {
    vi.mocked(getK8sSecret).mockResolvedValue(
      JSON.stringify({
        ...config,
        version: 2,
        namespace: 'team-a',
        state: 'pending',
      }),
    );
    const result = await requests(sharedUI())
      .get(route('preview'))
      .set('user', 'alice')
      .expect(503);
    const body = result.text || result.body.toString();
    expect(result.headers['retry-after']).toBe('5');
    expect(body).toContain('profile controller logs');
    expect(body).toContain('namespace artifact service rollout');
    expect(body).toContain('kfp-artifact-storage-config');
    expect(createExternalArtifactClient).not.toHaveBeenCalled();
    expect(sharedFetch).not.toHaveBeenCalled();
  });

  it('gives actionable receiver binding guidance without fetching credentials', async () => {
    const result = await requests(endpoint)
      .get(ARTIFACT_STORAGE_PROXY_PREFIX + route('preview'))
      .set('x-kfp-artifact-namespace', 'team-a')
      .set('x-kfp-artifact-credential-binding', 'f'.repeat(64))
      .expect(503);
    const body = result.text || result.body.toString();
    expect(result.headers['retry-after']).toBe('5');
    expect(body).toContain('profile controller logs');
    expect(body).toContain('namespace artifact service rollout');
    expect(body).toContain('kfp-artifact-storage-config');
    expect(body).not.toContain('Retry shortly');
    expect(createExternalArtifactClient).not.toHaveBeenCalled();
    expect(sharedFetch).not.toHaveBeenCalled();
  });

  it('keeps approved external grants authoritative for an artifacts/ key', async () => {
    await requests(sharedUI())
      .get(route('preview', config.storage.bucket, 'artifacts/model.txt'))
      .set('user', 'alice')
      .expect(200);
    expect(getObject).toHaveBeenCalledWith(config.storage.bucket, 'artifacts/model.txt');
    expect(createMinioClient).not.toHaveBeenCalled();
    expect(sharedFetch).not.toHaveBeenCalled();
  });
});
