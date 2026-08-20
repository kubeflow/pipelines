// Copyright 2025 The Kubeflow Authors
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

import { describe, it, expect, vi } from 'vitest';
import type { Request } from 'express';
import type { ArtifactCoordinates } from '../helpers/artifact-coordinates.js';
import {
  buildArtifactCoordinateUri,
  normalizeArtifactStorageCoordinates,
  resolveArtifactCoordinates,
} from '../helpers/artifact-coordinates.js';
import { getArtifactsAuthMiddleware, getArtifactsHandler } from './artifacts.js';

vi.mock('../k8s-helper.js', () => ({
  getConfigMap: vi.fn(),
  getK8sSecret: vi.fn(),
  getPod: vi.fn(),
}));

function makeRequest(path: string, query: Record<string, unknown> = {}): Request {
  return { path, query } as unknown as Request;
}

describe('resolveArtifactCoordinates', () => {
  describe('path-based routes', () => {
    it('extracts coordinates from /artifacts/:source/:bucket/* style URLs', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello/world.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('extracts coordinates from /pipeline-prefixed routes', () => {
      const req = makeRequest('/pipeline/artifacts/s3/my-bucket/path/to/file.csv');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'path/to/file.csv',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('extracts coordinates from a non-/pipeline base path', () => {
      const req = makeRequest('/foo/bar/artifacts/minio/my-bucket/some/key');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'my-bucket',
        key: 'some/key',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('rejects an encoded slash alias for a literal storage path delimiter', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%2Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toBeNull();
    });

    it('preserves a literal %2F when the URL is double-encoded (%252F)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%252Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello%2Fworld.txt',
        keyEncoding: 'storage',
        uriKey: 'hello%252Fworld.txt',
        artifactUriQuery: '',
      });
    });

    it('builds download identity from the escaped path while preserving the decoded storage key', () => {
      const coordinates = resolveArtifactCoordinates(
        makeRequest('/artifacts/s3/ml-pipeline/root%20dir/artifact.txt'),
      );

      expect(coordinates).toEqual({
        source: 's3',
        bucket: 'ml-pipeline',
        key: 'root dir/artifact.txt',
        keyEncoding: 'storage',
        uriKey: 'root%20dir/artifact.txt',
        artifactUriQuery: '',
      });
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        's3://ml-pipeline/root%20dir/artifact.txt',
      );
    });

    it('trims one trailing slash from a markerless launcher download storage key', () => {
      const coordinates = resolveArtifactCoordinates(
        makeRequest('/artifacts/minio/ml-pipeline/private-artifacts/team-a/output/'),
      );

      expect(coordinates).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'private-artifacts/team-a/output',
        keyEncoding: 'storage',
        uriKey: 'private-artifacts/team-a/output/',
        artifactUriQuery: '',
      });
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        'minio://ml-pipeline/private-artifacts/team-a/output/',
      );
    });

    it('uses an exact persisted URI identity for a canonical storage download path', () => {
      const coordinates = resolveArtifactCoordinates(
        makeRequest('/artifacts/s3/ml-pipeline/rootsecret/caf%C3%A9.txt', {
          uriKey: 'root%73ecret/caf%c3%a9.txt',
        }),
      );

      expect(coordinates).toEqual({
        source: 's3',
        bucket: 'ml-pipeline',
        key: 'rootsecret/café.txt',
        keyEncoding: 'storage',
        uriKey: 'root%73ecret/caf%c3%a9.txt',
        artifactUriQuery: '',
      });
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        's3://ml-pipeline/root%73ecret/caf%c3%a9.txt',
      );
    });

    it('rejects escaped traversal identity for a literal traversal download path', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest(
            '/artifacts/minio/mlpipeline/private-artifacts/attacker-ns/../../victim-ns/secret.txt',
            {
              uriKey: 'private-artifacts/attacker-ns/%2E%2E/%2E%2E/victim-ns/secret.txt',
            },
          ),
        ),
      ).toBeNull();
    });

    it('returns null on malformed percent-encoding (fail-closed)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/bad%ZZkey');
      expect(resolveArtifactCoordinates(req)).toBeNull();
    });

    it('rejects encoded aliases for URI-path literal characters', () => {
      expect(
        resolveArtifactCoordinates(makeRequest('/artifacts/s3/shared/root/%73ecret')),
      ).toBeNull();
      expect(
        resolveArtifactCoordinates(makeRequest('/artifacts/s3/shared/root/%2fsecret')),
      ).toBeNull();
    });

    it('handles keys that contain multiple slashes', () => {
      const req = makeRequest('/artifacts/gcs/my-bucket/a/b/c/d.json');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'gcs',
        bucket: 'my-bucket',
        key: 'a/b/c/d.json',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });
  });

  describe('query-based /artifacts/get route', () => {
    it('accepts request.query keys that are already decoded (spaces)', () => {
      const req = {
        path: '/artifacts/get',
        query: {
          source: 's3',
          bucket: 'reports',
          key: 'root dir/artifact.txt',
          artifactUriQuery: '',
        },
        url: '/artifacts/get?source=s3&bucket=reports&key=root%20dir%2Fartifact.txt',
      } as unknown as Request;
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'root dir/artifact.txt',
        keyEncoding: 'storage',
        uriKey: 'root%20dir/artifact.txt',
        artifactUriQuery: '',
      });
    });

    it('accepts decoded Unicode keys from legacy preview clients', () => {
      const req = {
        path: '/artifacts/get',
        query: {
          source: 's3',
          bucket: 'reports',
          key: 'café/model.txt',
          artifactUriQuery: '',
        },
        url: '/artifacts/get?source=s3&bucket=reports&key=caf%C3%A9%2Fmodel.txt',
      } as unknown as Request;
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'café/model.txt',
        keyEncoding: 'storage',
        uriKey: 'caf%C3%A9/model.txt',
        artifactUriQuery: '',
      });
    });

    it('accepts decoded percent characters from legacy preview clients', () => {
      const req = {
        path: '/artifacts/get',
        query: {
          source: 's3',
          bucket: 'reports',
          key: '100%complete/model.txt',
          artifactUriQuery: '',
        },
        url: '/artifacts/get?source=s3&bucket=reports&key=100%2525complete%2Fmodel.txt',
      } as unknown as Request;
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: '100%complete/model.txt',
        keyEncoding: 'storage',
        uriKey: '100%25complete/model.txt',
        artifactUriQuery: '',
      });
    });

    it('keeps a valid-looking percent escape literal in a legacy storage key', () => {
      const req = makeRequest('/artifacts/get', {
        source: 's3',
        bucket: 'reports',
        key: 'literal%20token/model.txt',
        keyEncoding: 'storage',
      });

      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'literal%20token/model.txt',
        keyEncoding: 'storage',
        uriKey: 'literal%2520token/model.txt',
        artifactUriQuery: '',
      });
    });

    it('decodes a canonical native URI key only when explicitly declared', () => {
      const req = makeRequest('/artifacts/get', {
        source: 's3',
        bucket: 'reports',
        key: 'root%20dir/model.txt',
        keyEncoding: 'uri',
      });

      const coordinates = resolveArtifactCoordinates(req);
      expect(coordinates).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'root%20dir/model.txt',
        keyEncoding: 'uri',
        artifactUriQuery: '',
      });
      expect(normalizeArtifactStorageCoordinates(coordinates as ArtifactCoordinates)).toMatchObject(
        { key: 'root dir/model.txt', keyEncoding: 'storage' },
      );
    });

    it('preserves exact native identity while using its decoded storage key', () => {
      const req = makeRequest('/artifacts/get', {
        source: 's3',
        bucket: 'reports',
        key: 'rootsecret/café.txt',
        keyEncoding: 'storage',
        uriKey: 'root%73ecret/caf%c3%a9.txt',
      });

      const coordinates = resolveArtifactCoordinates(req);
      expect(coordinates).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'rootsecret/café.txt',
        keyEncoding: 'storage',
        uriKey: 'root%73ecret/caf%c3%a9.txt',
        artifactUriQuery: '',
      });
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        's3://reports/root%73ecret/caf%c3%a9.txt',
      );
    });

    it('preserves one trailing slash in identity while trimming it from storage', () => {
      const coordinates = resolveArtifactCoordinates(
        makeRequest('/artifacts/get', {
          source: 'minio',
          bucket: 'mlpipeline',
          key: 'private-artifacts/team-a/run/output',
          keyEncoding: 'storage',
          uriKey: 'private-artifacts/team-a/run/output/',
        }),
      );

      expect(coordinates).toEqual({
        source: 'minio',
        bucket: 'mlpipeline',
        key: 'private-artifacts/team-a/run/output',
        keyEncoding: 'storage',
        uriKey: 'private-artifacts/team-a/run/output/',
        artifactUriQuery: '',
      });
      expect(buildArtifactCoordinateUri(coordinates!)).toBe(
        'minio://mlpipeline/private-artifacts/team-a/run/output/',
      );
    });

    it('rejects an exact URI identity that resolves to a different storage key', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 's3',
            bucket: 'reports',
            key: 'root%73ecret/café.txt',
            keyEncoding: 'storage',
            uriKey: 'root%73ecret/caf%c3%a9.txt',
          }),
        ),
      ).toBeNull();
    });

    it('rejects an exact URI identity containing an encoded path delimiter', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 's3',
            bucket: 'reports',
            key: 'private-artifacts/team-a/model',
            keyEncoding: 'storage',
            uriKey: 'private-artifacts%2Fteam-a/model',
          }),
        ),
      ).toBeNull();
    });

    it('rejects an exact URI identity containing an encoded ampersand', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 's3',
            bucket: 'reports',
            key: 'run/output&token',
            keyEncoding: 'storage',
            uriKey: 'run/output%26token',
          }),
        ),
      ).toBeNull();
    });

    it('preserves encoded ampersands in HTTP preview and download identities', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'https',
            bucket: 'files.example',
            key: 'reports/A&B.csv',
            keyEncoding: 'storage',
            uriKey: 'reports/A%26B.csv',
          }),
        ),
      ).toEqual({
        source: 'https',
        bucket: 'files.example',
        key: 'reports/A&B.csv',
        keyEncoding: 'storage',
        uriKey: 'reports/A%26B.csv',
        artifactUriQuery: '',
      });
      expect(
        resolveArtifactCoordinates(makeRequest('/artifacts/https/files.example/reports/A%26B.csv')),
      ).toEqual({
        source: 'https',
        bucket: 'files.example',
        key: 'reports/A&B.csv',
        keyEncoding: 'storage',
        uriKey: 'reports/A%26B.csv',
        artifactUriQuery: '',
      });
    });

    it('preserves encoded HTTP query and fragment delimiters as path identity', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'https',
            bucket: 'files.example',
            key: 'reports/A?B#C.csv',
            keyEncoding: 'storage',
            uriKey: 'reports/A%3FB%23C.csv',
          }),
        ),
      ).toMatchObject({ key: 'reports/A?B#C.csv', uriKey: 'reports/A%3FB%23C.csv' });
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/https/files.example/reports/A%3FB%23C.csv'),
        ),
      ).toMatchObject({ key: 'reports/A?B#C.csv', uriKey: 'reports/A%3FB%23C.csv' });
    });

    it('rejects raw query and fragment delimiters in exact non-launcher identity', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'https',
            bucket: 'files.example',
            key: 'reports/A?B.csv',
            keyEncoding: 'storage',
            uriKey: 'reports/A?B.csv',
          }),
        ),
      ).toBeNull();
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'volume',
            bucket: 'reports',
            key: 'A#B.csv',
            keyEncoding: 'storage',
            uriKey: 'A#B.csv',
          }),
        ),
      ).toBeNull();
    });

    it('rejects markerless non-launcher keys containing decoded query or fragment delimiters', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'https',
            bucket: 'files.example',
            key: 'reports/A?B.csv',
          }),
        ),
      ).toBeNull();
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'volume',
            bucket: 'reports',
            key: 'A#B.csv',
          }),
        ),
      ).toBeNull();
    });

    it('rejects escaped traversal segments in exact preview identity', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'minio',
            bucket: 'mlpipeline',
            key: 'private-artifacts/attacker-ns/../../victim-ns/secret.txt',
            keyEncoding: 'storage',
            uriKey: 'private-artifacts/attacker-ns/%2E%2E/%2E%2E/victim-ns/secret.txt',
          }),
        ),
      ).toBeNull();
    });

    it('rejects escaped traversal for volume while allowing handler-normalized forms', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'volume',
            bucket: 'artifact-volume',
            key: '../../etc/passwd',
            keyEncoding: 'storage',
            uriKey: '%2E%2E/%2E%2E/etc/passwd',
          }),
        ),
      ).toBeNull();

      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 'volume',
            bucket: 'artifact-volume',
            key: 'reports//./output.csv',
            keyEncoding: 'storage',
            uriKey: 'reports//./output.csv',
          }),
        ),
      ).toMatchObject({ key: 'reports//./output.csv', source: 'volume' });
    });

    it('rejects a noncanonical percent escape instead of treating it as an alias', () => {
      const req = {
        path: '/artifacts/get',
        query: {
          source: 's3',
          bucket: 'reports',
          key: '%73ecret/model.txt',
          keyEncoding: 'uri',
          artifactUriQuery: '',
        },
      } as unknown as Request;
      expect(resolveArtifactCoordinates(req)).toBeNull();
    });

    it('uses query coordinates when path is /artifacts/get', () => {
      const req = makeRequest('/artifacts/get', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
        artifactUriQuery: '',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('reconstructs a query-bearing artifact identity without changing its object key', () => {
      const req = makeRequest('/artifacts/get', {
        source: 's3',
        bucket: 'reports',
        key: 'output.html',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example&region=ceph',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'output.html',
        keyEncoding: 'storage',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example&region=ceph',
      });
    });

    it('rejects an encoded alias supplied through the preview query', () => {
      expect(
        resolveArtifactCoordinates(
          makeRequest('/artifacts/get', {
            source: 's3',
            bucket: 'shared',
            key: 'root/%73ecret',
            keyEncoding: 'uri',
          }),
        ),
      ).toBeNull();
    });

    it('uses query coordinates when path is /pipeline/artifacts/get', () => {
      const req = makeRequest('/pipeline/artifacts/get', {
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
        artifactUriQuery: '',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('does not trust query coordinates on an unrecognized path', () => {
      const req = makeRequest('/foo/bar', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toBeUndefined();
    });
  });

  describe('missing or non-string query values', () => {
    it('returns empty strings when query params are absent', () => {
      const req = makeRequest('/artifacts/get');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: '',
        key: '',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('rejects array-valued query params (treats them as missing)', () => {
      const req = makeRequest('/artifacts/get', {
        source: ['minio', 'sneaky'],
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: 'ml-pipeline',
        key: 'k',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('rejects object-valued query params (treats them as missing)', () => {
      const req = makeRequest('/artifacts/get', {
        source: { nested: 'minio' },
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: 'ml-pipeline',
        key: 'k',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });
  });

  describe('coordinate-source spoofing defense', () => {
    it('uses path coordinates when both path and query are present', () => {
      // Attacker plants benign values in query, real values in path.
      const req = makeRequest('/artifacts/minio/victim-bucket/secret.txt', {
        source: 'minio',
        bucket: 'safe-bucket',
        key: 'safe-key',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'victim-bucket',
        key: 'secret.txt',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('adds the stored artifact query to path coordinates', () => {
      const req = makeRequest('/artifacts/s3/reports/output.html', {
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'reports',
        key: 'output.html',
        keyEncoding: 'storage',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example',
      });
    });

    it('treats /artifacts/get/x/y as a path-based route with source=get (not the get endpoint)', () => {
      // Only an exact /artifacts/get path uses the query string; any other
      // path that matches /:source/:bucket/* uses path values, even if the
      // first segment happens to literally be "get". The downstream handler
      // then rejects source="get" as an unknown storage source (500).
      const req = makeRequest('/artifacts/get/some-bucket/some-key', {
        source: 'minio',
        bucket: 'safe-bucket',
        key: 'safe-key',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'get',
        bucket: 'some-bucket',
        key: 'some-key',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });
  });

  it('builds the canonical artifact URI without interpreting key fragments as query data', () => {
    expect(
      buildArtifactCoordinateUri({
        source: 's3',
        bucket: 'reports',
        key: 'models/checkpoint#latest',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example',
      }),
    ).toBe('s3://reports/models/checkpoint#latest?endpoint=https%3A%2F%2Fceph.example');
  });
});

describe('getArtifactsHandler authorization handoff', () => {
  it('rejects coordinates that differ from the URI authorized by middleware', async () => {
    const send = vi.fn();
    const status = vi.fn().mockReturnValue({ send });
    const handler = getArtifactsHandler({
      artifactsConfigs: {},
      options: {
        auth: { enabled: true },
        server: { serverNamespace: 'kubeflow' },
      },
      tryExtract: true,
      useParameter: false,
    } as unknown as Parameters<typeof getArtifactsHandler>[0]);
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    await handler(
      {
        path: '/artifacts/get',
        params: {},
        query: { bucket: 'reports', key: 'changed.csv', source: 's3' },
      } as never,
      {
        locals: { authorizedArtifactUri: 's3://reports/authorized.csv' },
        status,
      } as never,
      vi.fn(),
    );

    expect(status).toHaveBeenCalledWith(403);
    expect(send).toHaveBeenCalledWith('Artifact request coordinates changed after authorization');
    consoleSpy.mockRestore();
  });

  it.each([
    ['legacy-decoded-space', 'root dir/artifact.txt', 's3://reports/root%20dir/artifact.txt'],
    ['legacy-decoded-unicode', 'café/model.txt', 's3://reports/caf%C3%A9/model.txt'],
    ['legacy-decoded-percent', '100%complete/model.txt', 's3://reports/100%25complete/model.txt'],
  ])(
    'authorizes legacy %s /artifacts/get keys with decoded request keys',
    async (_, key, expectedArtifactUri) => {
      const send = vi.fn();
      const status = vi.fn().mockReturnValue({ send });
      const middleware = getArtifactsAuthMiddleware(
        () => Promise.resolve(undefined),
        true,
        'x-kubeflow-user',
        undefined,
        false,
      );
      const next = vi.fn();
      const response = { locals: {} } as { locals: { authorizedArtifactUri?: string } };

      await middleware(
        {
          path: '/artifacts/get',
          query: {
            source: 's3',
            bucket: 'reports',
            key,
            namespace: 'kubeflow',
          },
          url: `/artifacts/get?source=s3&bucket=reports&key=${encodeURIComponent(key)}&namespace=kubeflow`,
          headers: {
            'x-kubeflow-user': 'user-id',
          },
        } as any,
        response as any,
        next as any,
      );

      expect(status).not.toHaveBeenCalled();
      expect(next).toHaveBeenCalledTimes(1);
      expect(response.locals.authorizedArtifactUri).toBe(expectedArtifactUri);
    },
  );
});
