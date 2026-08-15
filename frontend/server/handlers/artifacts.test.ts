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
import {
  buildArtifactCoordinateUri,
  resolveArtifactCoordinates,
} from '../helpers/artifact-coordinates.js';
import { getArtifactsHandler } from './artifacts.js';

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

    it('decodes percent-encoded path segments once (matching Express req.params semantics)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%2Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('preserves a literal %2F when the URL is double-encoded (%252F)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%252Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello%2Fworld.txt',
        keyEncoding: 'storage',
        artifactUriQuery: '',
      });
    });

    it('returns null on malformed percent-encoding (fail-closed)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/bad%ZZkey');
      expect(resolveArtifactCoordinates(req)).toBeNull();
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
        keyEncoding: 'uri',
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
        keyEncoding: 'uri',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example&region=ceph',
      });
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
        keyEncoding: 'uri',
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
        keyEncoding: 'uri',
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
        keyEncoding: 'uri',
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
        keyEncoding: 'uri',
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
});
