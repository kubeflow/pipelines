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
        keyEncoding: 'uri',
        source: StorageService.S3,
      },
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
        keyEncoding: 'uri',
        source: StorageService.GCS,
      },
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
      keyEncoding: 'uri',
      source: StorageService.S3,
    });
    expect(location.artifactUriQuery).toBe(
      'endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
    );
  });

  it.each([
    ['raw spaces and Unicode', 's3://reports/root dir/café.txt', 'root%20dir/caf%C3%A9.txt'],
    ['canonical escapes', 's3://reports/root%20dir/caf%C3%A9.txt', 'root%20dir/caf%C3%A9.txt'],
    ['literal percent escape text', 's3://reports/root%2520dir/file', 'root%2520dir/file'],
    ['a raw literal percent', 's3://reports/100%complete/file', '100%25complete/file'],
  ])('canonicalizes %s in native artifact URI paths', (_description, uri, expectedKey) => {
    const location = parseArtifactFileLocation(uri);

    expect(location.path).toEqual({
      bucket: 'reports',
      key: expectedKey,
      keyEncoding: 'uri',
      source: StorageService.S3,
    });
  });

  it('builds valid preview and download URLs from a raw native artifact URI', () => {
    const location = parseArtifactFileLocation('s3://reports/root dir/café.txt');

    expect(Apis.buildReadFileUrl({ path: location.path })).toBe(
      'artifacts/get?source=s3&bucket=reports&key=root%2520dir%2Fcaf%25C3%25A9.txt&keyEncoding=uri',
    );
    expect(Apis.buildReadFileUrl({ path: location.path, isDownload: true })).toBe(
      'artifacts/s3/reports/root%20dir/caf%C3%A9.txt',
    );
  });

  it('keeps non-launcher paths as decoded storage keys', () => {
    expect(parseArtifactFileLocation('https://files.example/raw path/café.txt').path).toEqual({
      bucket: 'files.example',
      key: 'raw path/café.txt',
      keyEncoding: 'storage',
      source: StorageService.HTTPS,
    });
  });
});
