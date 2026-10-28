// Copyright 2026 The Kubeflow Authors
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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import express from 'express';
import type { Server } from 'node:http';
import requests from 'supertest';
import { getArtifactsAuthMiddleware, getArtifactsProxyHandler } from '../handlers/artifacts.js';
import {
  normalizeArtifactStorageCoordinates,
  resolveArtifactCoordinates,
} from '../helpers/artifact-coordinates.js';
import { getConfigMap } from '../k8s-helper.js';

vi.mock('../k8s-helper.js', () => ({ getConfigMap: vi.fn(), getK8sSecret: vi.fn() }));

// Use production authorization middleware, native validation, coordinates and HTTP proxying.
// Namespace permission/API responses and final storage reads are fixtures with observable calls.
describe.each([false, true])('artifact storage-prefix authorization (proxy=%s)', (proxy) => {
  let receiver: Server;
  let app: express.Express;
  let makeApp: (apiAddress?: string) => express.Express;
  let reads: string[];
  let metadataUris: string[];
  let hasMetadata: boolean;

  beforeEach(async () => {
    reads = [];
    metadataUris = [];
    hasMetadata = true;
    vi.mocked(getConfigMap).mockResolvedValue([undefined, { message: 'not found' }]);
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = new URL(String(input));
        const filter = JSON.parse(decodeURIComponent(url.searchParams.get('filter')!));
        metadataUris.push(filter.predicates[0].stringValue);
        return new Response(
          JSON.stringify({ artifacts: hasMetadata ? [{ artifact_id: 'imported' }] : [] }),
          {
            headers: { 'Content-Type': 'application/json' },
          },
        );
      }),
    );
    const serve: express.RequestHandler = (req, res) => {
      const coordinates = resolveArtifactCoordinates(req);
      if (!coordinates) throw new Error('Expected resolved serving coordinates');
      reads.push(normalizeArtifactStorageCoordinates(coordinates).key);
      res.send('fixture-object-bytes');
    };
    const downstream = express();
    downstream.use(serve);
    receiver = await new Promise<Server>((resolve) => {
      const server = downstream.listen(0, '127.0.0.1', () => resolve(server));
    });
    const address = receiver.address();
    if (!address || typeof address === 'string') throw new Error('Expected TCP listener');
    makeApp = (apiAddress) => {
      const ui = express();
      ui.use(
        getArtifactsAuthMiddleware(async () => undefined, true, 'kubeflow-userid', apiAddress),
      );
      ui.use(
        getArtifactsProxyHandler({
          enabled: proxy,
          allowedDomain: '127\\.0\\.0\\.1',
          namespacedServiceGetter: () => `http://127.0.0.1:${address.port}`,
        }),
      );
      ui.use(serve);
      return ui;
    };
    app = makeApp('http://api-server');
  });

  afterEach(async () => {
    vi.unstubAllGlobals();
    await new Promise<void>((resolve) => {
      receiver.close(() => resolve());
      receiver.closeAllConnections();
    });
  });

  const aliases = [
    '%70rivate-artifacts/team-b/secret',
    'private%2Dartifacts/team-b/secret',
    '%70rivate-artifacts/%74eam-b/secret',
  ];
  const routes = ['preview', 'raw-download', 'path-download'] as const;
  it.each(aliases.flatMap((uriKey) => routes.map((route) => ({ uriKey, route }))))(
    'denies foreign storage prefix $uriKey on $route before metadata or serving',
    async ({ uriKey, route }) => {
      const key = 'private-artifacts/team-b/secret';
      const path = route === 'path-download' ? `/artifacts/s3/bucket/${key}` : '/artifacts/get';
      await requests(app)
        .get(path)
        .query({
          ...(route === 'path-download' ? {} : { source: 's3', bucket: 'bucket', key }),
          uriKey,
          namespace: 'team-a',
          ...(route === 'raw-download' ? { download: 'true' } : {}),
        })
        .set('kubeflow-userid', 'a@example.com')
        .expect(403);
      expect(metadataUris).toEqual([]);
      expect(reads).toEqual([]);
    },
  );

  it.each(routes)('denies another namespace directory root on %s', async (route) => {
    const key = 'private-artifacts/team-b';
    const path = route === 'path-download' ? `/artifacts/s3/bucket/${key}` : '/artifacts/get';
    await requests(app)
      .get(path)
      .query({
        ...(route === 'path-download' ? {} : { source: 's3', bucket: 'bucket', key }),
        uriKey: '%70rivate-artifacts/team-b',
        namespace: 'team-a',
        ...(route === 'raw-download' ? { download: 'true' } : {}),
      })
      .set('kubeflow-userid', 'a@example.com')
      .expect(403);
    expect(metadataUris).toEqual([]);
    expect(reads).toEqual([]);
  });

  it('denies a foreign alias even without metadata API configuration', async () => {
    await requests(makeApp())
      .get('/artifacts/get')
      .query({
        source: 's3',
        bucket: 'bucket',
        key: 'private-artifacts/team-b/secret',
        uriKey: '%70rivate-artifacts/team-b/secret',
        namespace: 'team-a',
      })
      .set('kubeflow-userid', 'a@example.com')
      .expect(403);
    expect(metadataUris).toEqual([]);
    expect(reads).toEqual([]);
  });

  it.each(['minio', 'gcs', 'http', 'https'])(
    'denies an encoded foreign prefix for %s before serving',
    async (source) => {
      await requests(app)
        .get('/artifacts/get')
        .query({
          source,
          bucket: 'bucket',
          key: 'private-artifacts/team-b/secret',
          uriKey: '%70rivate-artifacts/team-b/secret',
          namespace: 'team-a',
        })
        .set('kubeflow-userid', 'a@example.com')
        .expect(403);
      expect(metadataUris).toEqual([]);
      expect(reads).toEqual([]);
    },
  );

  it.each(['%70rivate-artifacts/team-a/model', 'private%2Dartifacts/%74eam-a/model'])(
    'preserves exact own-prefix metadata identity %s and requires its record',
    async (uriKey) => {
      const query = {
        source: 's3',
        bucket: 'bucket',
        key: 'private-artifacts/team-a/model',
        uriKey,
        namespace: 'team-a',
        artifactUriQuery: 'region=us-west-2',
      };
      await requests(app)
        .get('/artifacts/get')
        .query(query)
        .set('kubeflow-userid', 'a@example.com')
        .expect(200);
      hasMetadata = false;
      await requests(app)
        .get('/artifacts/get')
        .query(query)
        .set('kubeflow-userid', 'a@example.com')
        .expect(403);
      expect(metadataUris).toEqual(Array(2).fill(`s3://bucket/${uriKey}?region=us-west-2`));
      expect(reads).toEqual(['private-artifacts/team-a/model']);
    },
  );

  it.each([
    ['s3', 'custom/team-training/model', '%63ustom/team-training/model'],
    ['s3', '%70rivate-artifacts/team-b/model', '%2570rivate-artifacts/team-b/model'],
    ['http', '%70rivate-artifacts/team-b/model', undefined],
    ['http', 'custom/model%ZZ', undefined],
  ])(
    'preserves custom storage key %s/%s without decoding it again',
    async (source, key, uriKey) => {
      await requests(app)
        .get('/artifacts/get')
        .query({
          source,
          bucket: 'bucket',
          key,
          ...(uriKey ? { uriKey } : {}),
          namespace: 'team-a',
        })
        .set('kubeflow-userid', 'a@example.com')
        .expect(200);
      expect(metadataUris).toEqual([`${source}://bucket/${uriKey ?? key}`]);
      expect(reads).toEqual([key]);
    },
  );

  it('denies a known foreign HTTP prefix with a literal malformed escape in its filename', async () => {
    await requests(app)
      .get('/artifacts/get')
      .query({
        source: 'http',
        bucket: 'bucket',
        key: 'private-artifacts/team-b/model%ZZ',
        namespace: 'team-a',
      })
      .set('kubeflow-userid', 'a@example.com')
      .expect(403);
    expect(metadataUris).toEqual([]);
    expect(reads).toEqual([]);
  });
});
