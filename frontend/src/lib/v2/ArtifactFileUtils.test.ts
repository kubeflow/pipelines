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

import { Apis } from 'src/lib/Apis';
import { StorageService } from 'src/lib/WorkflowParser';
import { parseArtifactFileLocation, readArtifactFile } from './ArtifactFileUtils';

describe('readArtifactFile', () => {
  it('uses server-side provider resolution and the explicit namespace', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('contents');

    await expect(
      readArtifactFile(
        {
          uri: 's3://reports/output.html',
          namespace: 'artifact-namespace',
          metadata: { store_session_info: 'stale-session' } as any,
        },
        'request-namespace',
      ),
    ).resolves.toBe('contents');

    expect(readFileSpy).toHaveBeenCalledWith({
      path: {
        bucket: 'reports',
        key: 'output.html',
        keyEncoding: 'storage',
        source: StorageService.S3,
      },
      artifactUriQuery: undefined,
      namespace: 'request-namespace',
    });
  });

  it('falls back to the artifact namespace', async () => {
    const readFileSpy = vi.spyOn(Apis, 'readFile').mockResolvedValue('contents');

    await readArtifactFile({
      uri: 'gs://reports/output.html',
      namespace: 'artifact-namespace',
    });

    expect(readFileSpy).toHaveBeenCalledWith({
      path: {
        bucket: 'reports',
        key: 'output.html',
        keyEncoding: 'storage',
        source: StorageService.GCS,
      },
      artifactUriQuery: undefined,
      namespace: 'artifact-namespace',
    });
  });

  it('keeps the artifact URI query separate from the object key for server validation', async () => {
    const location = parseArtifactFileLocation(
      's3://reports/output.html?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
    );

    expect(location.path).toEqual({
      bucket: 'reports',
      key: 'output.html',
      keyEncoding: 'storage',
      source: StorageService.S3,
    });
    expect(location.artifactUriQuery).toBe(
      'endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
    );
  });

  it.each([
    [
      'raw spaces and Unicode',
      's3://reports/root dir/café.txt',
      'root dir/café.txt',
      'root dir/café.txt',
    ],
    ['canonical escapes', 's3://reports/root%20dir/caf%C3%A9.txt', 'root dir/café.txt', undefined],
    [
      'a noncanonical valid escape',
      's3://reports/root%73ecret/file',
      'rootsecret/file',
      'root%73ecret/file',
    ],
    ['lowercase escape spelling', 's3://reports/caf%c3%a9/file', 'café/file', 'caf%c3%a9/file'],
    ['literal percent escape text', 's3://reports/root%2520dir/file', 'root%20dir/file', undefined],
  ])(
    'preserves %s identity while deriving the launcher storage key',
    (_description, uri, expectedKey, expectedUriKey) => {
      const location = parseArtifactFileLocation(uri);

      expect(location.path).toEqual({
        bucket: 'reports',
        key: expectedKey,
        keyEncoding: 'storage',
        source: StorageService.S3,
        ...(expectedUriKey ? { uriKey: expectedUriKey } : {}),
      });
    },
  );

  it('rejects malformed percent encoding that the launcher cannot parse', () => {
    expect(() => parseArtifactFileLocation('s3://reports/100%complete/file')).toThrow(
      'Artifact URI key has invalid encoding',
    );
  });

  it.each([
    ['raw HTTP query syntax', 'https://files.example/reports/A?B.csv', 'Percent-encode ? as %3F'],
    [
      'raw HTTP fragment syntax',
      'https://files.example/reports/A#B.csv',
      'Percent-encode # as %23',
    ],
    ['raw volume query syntax', 'volume://reports/A?B.csv', 'Percent-encode ? as %3F'],
    ['raw volume fragment syntax', 'volume://reports/A#B.csv', 'Percent-encode # as %23'],
  ])('rejects %s instead of changing path identity', (_description, uri, message) => {
    expect(() => parseArtifactFileLocation(uri)).toThrow(message);
  });

  it.each(['https://files.example/reports/A?', 's3://reports/output?'])(
    'rejects an empty trailing query marker in %s',
    (uri) => {
      expect(() => parseArtifactFileLocation(uri)).toThrow(
        'Artifact URIs cannot end with an empty query marker. Remove the trailing ? and retry.',
      );
    },
  );

  it('rejects an encoded path delimiter that would change ownership segmentation', () => {
    expect(() =>
      parseArtifactFileLocation('s3://reports/private-artifacts%2Fteam-a/model'),
    ).toThrow(
      'cannot contain empty or relative path segments, encoded separators, query delimiters, or fragment delimiters',
    );
  });

  it('rejects escaped traversal segments before building an artifact request', () => {
    expect(() =>
      parseArtifactFileLocation('s3://reports/private-artifacts/team-a/%2E%2E/secret'),
    ).toThrow(
      'cannot contain empty or relative path segments, encoded separators, query delimiters, or fragment delimiters',
    );
  });

  it('rejects encoded ampersands that Go SplitObjectURI treats as query delimiters', () => {
    expect(() => parseArtifactFileLocation('s3://reports/run/output%26token')).toThrow(
      'Artifact URI key has invalid encoding',
    );
  });

  it('preserves encoded ampersands for non-launcher HTTP sources', () => {
    expect(parseArtifactFileLocation('https://files.example/reports/A%26B.csv').path).toEqual({
      bucket: 'files.example',
      key: 'reports/A&B.csv',
      keyEncoding: 'storage',
      source: StorageService.HTTPS,
      uriKey: 'reports/A%26B.csv',
    });
  });

  it.each([
    ['HTTP query delimiter', 'https://files.example/reports/A%3FB.csv', 'reports/A?B.csv'],
    ['HTTP fragment delimiter', 'https://files.example/reports/A%23B.csv', 'reports/A#B.csv'],
    ['volume query delimiter', 'volume://reports/A%3FB.csv', 'A?B.csv'],
    ['volume fragment delimiter', 'volume://reports/A%23B.csv', 'A#B.csv'],
  ])('preserves encoded %s as exact non-launcher path data', (_description, uri, key) => {
    const location = parseArtifactFileLocation(uri);

    expect(location.path.key).toBe(key);
    expect(location.path.uriKey).toBe(uri.slice(uri.indexOf('/', uri.indexOf('://') + 3) + 1));
    expect(Apis.buildReadFileUrl({ path: location.path, isDownload: true })).toContain(
      location.path.uriKey,
    );
  });

  it('rejects non-launcher encoded slashes locally, matching the server identity boundary', () => {
    expect(() => parseArtifactFileLocation('https://example.com/a%2Fb/c')).toThrow(
      'cannot contain empty or relative path segments, encoded separators, query delimiters, or fragment delimiters',
    );
  });

  it.each([
    'https://example.com/a/../b',
    'https://example.com/a/./b',
    'https://example.com/a/%2e%2e/b',
    'volume://my-vol/a/../b',
    'volume://my-vol/a/./b',
    'volume://my-vol/a/%2e%2e/b',
  ])('rejects non-launcher relative segments before download URL normalization: %s', (uri) => {
    expect(() => parseArtifactFileLocation(uri)).toThrow(
      'cannot contain empty or relative path segments',
    );
  });

  it('preserves repeated non-launcher separators that survive URL path transport', () => {
    expect(parseArtifactFileLocation('https://example.com/a//b').path.key).toBe('a//b');
  });

  it('accepts the trailing slash that Go SplitObjectURI trims for launcher artifacts', () => {
    expect(parseArtifactFileLocation('minio://mlpipeline/v2/artifacts/run-1/dir/').path).toEqual({
      bucket: 'mlpipeline',
      key: 'v2/artifacts/run-1/dir',
      keyEncoding: 'storage',
      source: StorageService.MINIO,
      uriKey: 'v2/artifacts/run-1/dir/',
    });
  });

  it('returns URI parsing failures as rejected promises', async () => {
    const result = readArtifactFile({ uri: 's3://reports/100%complete/file' });

    await expect(result).rejects.toThrow('Artifact URI key has invalid encoding');
  });

  it('builds valid preview and download URLs from a raw native artifact URI', () => {
    const location = parseArtifactFileLocation('s3://reports/root dir/café.txt');

    expect(Apis.buildReadFileUrl({ path: location.path })).toBe(
      'artifacts/get?source=s3&bucket=reports&key=root%20dir%2Fcaf%C3%A9.txt&keyEncoding=storage&uriKey=root%20dir%2Fcaf%C3%A9.txt',
    );
    expect(Apis.buildReadFileUrl({ path: location.path, isDownload: true })).toBe(
      'artifacts/s3/reports/root%20dir/caf%C3%A9.txt?uriKey=root%20dir%2Fcaf%C3%A9.txt',
    );
  });

  it('decodes escaped non-launcher paths for storage without changing their URI identity', () => {
    const location = parseArtifactFileLocation('https://files.example/root%20dir/caf%c3%a9.txt');

    expect(location.path).toEqual({
      bucket: 'files.example',
      key: 'root dir/café.txt',
      keyEncoding: 'storage',
      source: StorageService.HTTPS,
      uriKey: 'root%20dir/caf%c3%a9.txt',
    });
    expect(Apis.buildReadFileUrl({ path: location.path })).toBe(
      'artifacts/get?source=https&bucket=files.example&key=root%20dir%2Fcaf%C3%A9.txt&keyEncoding=storage&uriKey=root%2520dir%2Fcaf%25c3%25a9.txt',
    );
  });

  it('preserves raw-space and Unicode HTTP identity separately from canonical download transport', () => {
    const location = parseArtifactFileLocation('https://files.example/root dir/café.txt');

    expect(location.path).toMatchObject({
      key: 'root dir/café.txt',
      uriKey: 'root dir/café.txt',
    });
    expect(Apis.buildReadFileUrl({ path: location.path, isDownload: true })).toBe(
      'artifacts/https/files.example/root%20dir/caf%C3%A9.txt?uriKey=root%20dir%2Fcaf%C3%A9.txt',
    );
  });
});
