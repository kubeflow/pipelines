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

import { describe, it, expect } from 'vitest';
import type { Request } from 'express';
import { resolveArtifactCoordinates } from '../helpers/artifact-coordinates.js';

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
      });
    });

    it('extracts coordinates from /pipeline-prefixed routes', () => {
      const req = makeRequest('/pipeline/artifacts/s3/my-bucket/path/to/file.csv');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'path/to/file.csv',
      });
    });

    it('extracts coordinates from a non-/pipeline base path', () => {
      const req = makeRequest('/foo/bar/artifacts/minio/my-bucket/some/key');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'my-bucket',
        key: 'some/key',
      });
    });

    it('decodes percent-encoded path segments once (matching Express req.params semantics)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%2Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
    });

    it('preserves a literal %2F when the URL is double-encoded (%252F)', () => {
      const req = makeRequest('/artifacts/minio/ml-pipeline/hello%252Fworld.txt');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello%2Fworld.txt',
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
      });
    });
  });

  describe('query-based fallback (/artifacts/get and unrecognized paths)', () => {
    it('falls back to query when path is /artifacts/get', () => {
      const req = makeRequest('/artifacts/get', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'hello/world.txt',
      });
    });

    it('falls back to query when path is /pipeline/artifacts/get', () => {
      const req = makeRequest('/pipeline/artifacts/get', {
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 's3',
        bucket: 'my-bucket',
        key: 'data.csv',
      });
    });

    it('falls back to query when path does not match the artifact patterns', () => {
      const req = makeRequest('/foo/bar', {
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'k',
      });
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: 'minio',
        bucket: 'ml-pipeline',
        key: 'k',
      });
    });
  });

  describe('missing or non-string query values', () => {
    it('returns empty strings when query params are absent', () => {
      const req = makeRequest('/artifacts/get');
      expect(resolveArtifactCoordinates(req)).toEqual({
        source: '',
        bucket: '',
        key: '',
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
      });
    });
  });
});
