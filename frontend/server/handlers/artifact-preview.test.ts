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

import express from 'express';
import requests from 'supertest';
import { Client } from 'minio';
import { JWT } from 'google-auth-library';
import { Readable } from 'stream';
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { loadConfigs } from '../configs.js';
import { UIServer } from '../app.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import { commonSetup } from '../integration-tests/test-helper.js';
import * as launcher from '../helpers/launcher-config.js';
import * as minioHelper from '../minio-helper.js';
import * as gcs from '../gcs-helper.js';
import * as artifacts from './artifacts.js';
import { getArtifactPreviewHandler } from './artifact-preview.js';
import { ArtifactArtifactType, V2Artifact } from '../src/generated/apisv2/artifact/index.js';
import {
  ARTIFACT_PREVIEW_RESPONSE_HEADER,
  MAX_ARTIFACT_PREVIEW_BYTES,
} from '../helpers/artifact-preview.js';

vi.hoisted(() => vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', 'artifact-only'));
afterAll(() => vi.unstubAllEnvs());

const content = '{"version":1,"kind":"model","hyperparameters":{"max_depth":3}}';
const { argv } = commonSetup();
const authorize = vi.fn<AuthorizeFn>();
const storage = new Client({ endPoint: 'localhost', port: 9000, useSSL: false });
let storedArtifact: V2Artifact;
let metadataStatus: number;
let proxyResponse: Response | undefined;
let ownershipLookupAvailable: boolean;

function setup(auth = true, proxy = false) {
  const options = loadConfigs(argv, {
    AUTH_ENABLED: String(auth),
    KUBEFLOW_USERID_HEADER: 'kubeflow-userid',
    FRONTEND_SERVER_NAMESPACE: 'kubeflow',
    ARTIFACTS_SERVICE_PROXY_ENABLED: String(proxy),
  });
  options.auth.enabled = auth;
  const app = express();
  app.get(
    '/artifacts/:artifactId/preview',
    artifacts.getArtifactsRateLimiter({
      authEnabled: options.auth.enabled,
      kubeflowUserIdHeader: options.auth.kubeflowUserIdHeader,
    }),
    getArtifactPreviewHandler(options, 'http://api-server', authorize),
  );
  return app;
}

function previewRequest(auth = true, proxy = false) {
  return requests(setup(auth, proxy))
    .get('/artifacts/artifact-1/preview?namespace=team-a')
    .set('kubeflow-userid', 'reader@example.com');
}

beforeEach(() => {
  storedArtifact = {
    artifact_id: 'artifact-1',
    type: ArtifactArtifactType.Model,
    uri: 's3://bucket/private-artifacts/team-a/model',
    namespace: 'team-a',
    metadata: { _kfp_preview: { version: 1, path: 'preview.json' } },
  };
  metadataStatus = 200;
  ownershipLookupAvailable = true;
  proxyResponse = undefined;
  authorize.mockReset().mockResolvedValue(undefined);
  vi.spyOn(launcher, 'getLauncherProviderInfo').mockResolvedValue('');
  vi.spyOn(minioHelper, 'createArtifactStoreClient').mockResolvedValue(storage);
  vi.spyOn(minioHelper, 'createMinioClient').mockResolvedValue(storage);
  vi.spyOn(storage, 'getObject').mockImplementation(async () => Readable.from([content]));
  vi.spyOn(storage, 'listObjectsV2').mockImplementation(() => {
    throw new Error('Must not list a preview directory');
  });
  vi.spyOn(gcs, 'getGCSClient').mockResolvedValue(new JWT());
  vi.spyOn(gcs, 'downloadGCSObjectStream').mockImplementation(async () => Readable.from([content]));
  vi.spyOn(gcs, 'listGCSObjectNames').mockImplementation(async () => {
    throw new Error('Must not enumerate GCS previews');
  });
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(
        typeof input === 'string' ? input : input instanceof URL ? input.href : input.url,
      );
      if (url.hostname !== 'api-server') {
        if (proxyResponse) return proxyResponse;
        throw new Error('Unexpected external request');
      }
      if (url.pathname === '/apis/v2/artifacts/artifact-1') {
        return new Response(JSON.stringify(storedArtifact), { status: metadataStatus });
      }
      if (url.pathname === '/apis/v2/artifacts') {
        if (!ownershipLookupAvailable) return new Response('Unavailable', { status: 503 });
        const filter = JSON.parse(decodeURIComponent(url.searchParams.get('filter') || ''));
        const exactParent = filter.predicates[0].stringValue === storedArtifact.uri;
        return new Response(JSON.stringify({ artifacts: exactParent ? [storedArtifact] : [] }));
      }
      throw new Error(`Unexpected API path ${url.pathname}`);
    }),
  );
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe('artifact preview endpoint', () => {
  it.each(['s3', 'minio', 'gs'])(
    'reads only preview.json with strict parent ownership on %s',
    async (scheme) => {
      storedArtifact.uri = `${scheme}://bucket/private-artifacts/team-a/model/`;
      const response = await previewRequest().expect(200);
      expect(response.body.hyperparameters).toEqual({ max_depth: 3 });
      expect(response.headers['cache-control']).toBe('no-store');
      expect(response.headers['x-content-type-options']).toBe('nosniff');
      expect(response.headers[ARTIFACT_PREVIEW_RESPONSE_HEADER.toLowerCase()]).toBe('1');
      expect(authorize).toHaveBeenCalledOnce();
      expect(fetch).toHaveBeenCalledTimes(2);
      if (scheme === 'gs') {
        expect(gcs.downloadGCSObjectStream).toHaveBeenCalledWith(
          expect.objectContaining({
            bucket: 'bucket',
            objectName: 'private-artifacts/team-a/model/preview.json',
          }),
        );
      } else {
        expect(storage.getObject).toHaveBeenCalledExactlyOnceWith(
          'bucket',
          'private-artifacts/team-a/model/preview.json',
        );
      }
      expect(gcs.listGCSObjectNames).not.toHaveBeenCalled();
      expect(storage.listObjectsV2).not.toHaveBeenCalled();
    },
  );

  it('resolves provider options from the original parent, not the child key', async () => {
    storedArtifact.uri = 's3://bucket/private-artifacts/team-a/a%20b?region=us-west-1';
    await previewRequest().expect(200);
    expect(launcher.getLauncherProviderInfo).toHaveBeenCalledWith(
      expect.objectContaining({
        key: 'private-artifacts/team-a/a b',
        uriKey: 'private-artifacts/team-a/a%20b',
        artifactUriQuery: 'region=us-west-1',
      }),
      'team-a',
    );
    expect(storage.getObject).toHaveBeenCalledWith(
      'bucket',
      'private-artifacts/team-a/a b/preview.json',
    );
  });

  it('supports standalone metadata reads without an explicit namespace', async () => {
    await requests(setup(false)).get('/artifacts/artifact-1/preview').expect(200);
    expect(fetch).toHaveBeenCalledOnce();
    expect(authorize).not.toHaveBeenCalled();
  });

  it('requires identity and namespace in authenticated mode', async () => {
    await requests(setup()).get('/artifacts/artifact-1/preview?namespace=team-a').expect(401);
    await requests(setup())
      .get('/artifacts/artifact-1/preview')
      .set('kubeflow-userid', 'reader')
      .expect(400);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it.each([
    'path=other.json',
    'uri=s3%3A%2F%2Fother%2Ffile',
    'providerInfo=untrusted',
    'peek=9999999',
    'namespace=team-b',
  ])('rejects caller-selected authority or repeated query values: %s', async (query) => {
    await requests(setup())
      .get(`/artifacts/artifact-1/preview?namespace=team-a&${query}`)
      .set('kubeflow-userid', 'reader')
      .expect(400);
    expect(fetch).not.toHaveBeenCalled();
  });

  it('denies viewers independently of artifact read access', async () => {
    authorize.mockResolvedValue({ message: 'Viewers denied' });
    await previewRequest().expect(403);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('denies a different stored namespace and imported cross-namespace roots', async () => {
    storedArtifact.namespace = 'team-b';
    await previewRequest().expect(403);
    storedArtifact.namespace = 'team-a';
    storedArtifact.uri = 's3://bucket/private-artifacts/team-b/model';
    await previewRequest().expect(403);
    storedArtifact.uri = 's3://bucket/custom-root/model';
    await previewRequest().expect(403);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('fails closed when strict ownership lookup is unavailable', async () => {
    ownershipLookupAvailable = false;
    await previewRequest().expect(403);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it.each([
    undefined,
    { version: 2, path: 'preview.json' },
    { version: 1, path: '../private.json' },
  ])('does not probe files without a supported declaration: %j', async (descriptor) => {
    storedArtifact.metadata = descriptor === undefined ? {} : { _kfp_preview: descriptor };
    await previewRequest().expect(descriptor === undefined ? 404 : 422);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('reports missing parent artifacts and sidecar files distinctly', async () => {
    metadataStatus = 404;
    await previewRequest().expect(404);
    metadataStatus = 200;
    vi.mocked(storage.getObject).mockRejectedValue(
      Object.assign(new Error('missing'), { code: 'NoSuchKey' }),
    );
    const response = await previewRequest().expect(404);
    expect(response.text).toContain('preview.json is missing');
    expect(storage.listObjectsV2).not.toHaveBeenCalled();
  });

  it('reports missing GCS previews without treating them as empty data', async () => {
    storedArtifact.uri = 'gs://bucket/private-artifacts/team-a/model';
    vi.mocked(gcs.downloadGCSObjectStream).mockRejectedValue(
      Object.assign(new Error('missing'), { response: { status: 404 } }),
    );
    await previewRequest().expect(404);
  });

  it('rejects an oversized object before sending success headers', async () => {
    const source = Readable.from([content.padEnd(MAX_ARTIFACT_PREVIEW_BYTES + 1)]);
    vi.mocked(storage.getObject).mockResolvedValue(source);
    await previewRequest().expect(413);
    expect(source.destroyed).toBe(true);
  });

  it.each([
    'metadata',
    'viewers',
    'ownership',
    'provider',
    's3-client',
    'minio-client',
    'gcs-client',
  ])('times out a stalled %s lookup before reading artifact bytes', async (stage) => {
    const deadline = new AbortController();
    vi.spyOn(AbortSignal, 'timeout').mockReturnValue(deadline.signal);
    let started: () => void = () => undefined;
    const ready = new Promise<void>((resolve) => {
      started = resolve;
    });
    function pending<T>(): Promise<T> {
      started();
      return new Promise<T>(() => undefined);
    }
    if (stage === 'metadata') {
      vi.mocked(fetch).mockImplementationOnce(() => pending());
    } else if (stage === 'viewers') {
      authorize.mockImplementation(() => pending());
    } else if (stage === 'ownership') {
      vi.mocked(fetch)
        .mockResolvedValueOnce(new Response(JSON.stringify(storedArtifact)))
        .mockImplementationOnce(() => pending());
    } else if (stage === 'provider') {
      vi.mocked(launcher.getLauncherProviderInfo).mockImplementation(() => pending());
    } else if (stage === 's3-client') {
      vi.mocked(minioHelper.createMinioClient).mockImplementation(() => pending());
    } else if (stage === 'minio-client') {
      storedArtifact.uri = 'minio://bucket/private-artifacts/team-a/model';
      vi.mocked(minioHelper.createArtifactStoreClient).mockImplementation(() => pending());
    } else {
      storedArtifact.uri = 'gs://bucket/private-artifacts/team-a/model';
      vi.mocked(gcs.getGCSClient).mockImplementation(() => pending());
    }
    const result = previewRequest().then((response) => response);
    await ready;
    deadline.abort(new DOMException('Preview deadline exceeded', 'TimeoutError'));
    expect((await result).status).toBe(504);
    expect(storage.getObject).not.toHaveBeenCalled();
    expect(gcs.downloadGCSObjectStream).not.toHaveBeenCalled();
  });

  it('keeps proxy isolation and forwards only the artifact ID and identity', async () => {
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(
      () => 'http://preview-service.team-a:3000',
    );
    proxyResponse = new Response(content, { headers: { [ARTIFACT_PREVIEW_RESPONSE_HEADER]: '1' } });
    await previewRequest(true, true).expect(200);
    expect(fetch).toHaveBeenLastCalledWith(
      new URL('http://preview-service.team-a:3000/artifacts/artifact-1/preview?namespace=team-a'),
      expect.objectContaining({
        headers: { 'kubeflow-userid': 'reader@example.com' },
        redirect: 'error',
        signal: expect.any(AbortSignal),
      }),
    );
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('rejects legacy proxy responses and enforces the limit at the shared boundary', async () => {
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(
      () => 'http://preview-service.team-a:3000',
    );
    proxyResponse = new Response('<html>old service</html>');
    await previewRequest(true, true).expect(502);
    proxyResponse = new Response('x'.repeat(MAX_ARTIFACT_PREVIEW_BYTES + 1), {
      headers: { [ARTIFACT_PREVIEW_RESPONSE_HEADER]: '1' },
    });
    await previewRequest(true, true).expect(413);
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('preserves a namespace receiver rate limit and only its delay-seconds retry guidance', async () => {
    vi.spyOn(artifacts, 'getArtifactServiceGetter').mockReturnValue(
      () => 'http://preview-service.team-a:3000',
    );
    proxyResponse = new Response('receiver quota exceeded', {
      status: 429,
      headers: { 'Retry-After': '17', RateLimit: '"300-in-1min"; r=0; t=17' },
    });
    const limited = await previewRequest(true, true).expect(429);
    expect(limited.headers['retry-after']).toBe('17');
    expect(limited.headers['ratelimit']).toContain('r=299');
    expect(limited.text).toBe(artifacts.TOO_MANY_ARTIFACT_REQUESTS_MESSAGE);

    proxyResponse = new Response('receiver quota exceeded', {
      status: 429,
      headers: { 'Retry-After': 'Wed, 21 Oct 2015 07:28:00 GMT' },
    });
    const undated = await previewRequest(true, true).expect(429);
    expect(undated.headers['retry-after']).toBeUndefined();
    expect(undated.text).toBe('Too many artifact requests. Retry later.');
    expect(storage.getObject).not.toHaveBeenCalled();
  });

  it('registers both normal and base-path routes before the generic artifact handler', async () => {
    const options = loadConfigs(argv, { AUTH_ENABLED: 'false' });
    options.auth.enabled = false;
    options.pipeline.host = 'api-server';
    options.pipeline.port = 80;
    options.server.basePath = '/pipeline';
    const app = new UIServer(options);
    await requests(app.app).get('/artifacts/artifact-1/preview?namespace=team-a').expect(200);
    await requests(app.app)
      .get('/pipeline/artifacts/artifact-1/preview?namespace=team-a')
      .expect(200);
    expect(storage.getObject).toHaveBeenCalledTimes(2);
  });
});
