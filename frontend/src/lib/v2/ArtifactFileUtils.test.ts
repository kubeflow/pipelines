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
      path: { bucket: 'reports', key: 'output.html', source: StorageService.S3 },
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
      path: { bucket: 'reports', key: 'output.html', source: StorageService.GCS },
      namespace: 'artifact-namespace',
    });
  });

  it('removes provider query parameters from the object key and forwards them separately', async () => {
    const location = parseArtifactFileLocation(
      's3://reports/output.html?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
    );

    expect(location.path).toEqual({
      bucket: 'reports',
      key: 'output.html',
      source: StorageService.S3,
    });
    expect(JSON.parse(location.providerInfo || '')).toEqual({
      Provider: 's3',
      Params: {
        endpoint: 'https://ceph.example:9443',
        fromEnv: 'true',
        region: 'ceph',
      },
    });
  });
});
