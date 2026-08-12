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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getConfigMap } from '../k8s-helper.js';
import { getLauncherProviderInfo } from './launcher-config.js';

vi.mock('../k8s-helper.js', () => ({ getConfigMap: vi.fn() }));

const mockedGetConfigMap = vi.mocked(getConfigMap);

describe('getLauncherProviderInfo', () => {
  beforeEach(() => {
    mockedGetConfigMap.mockReset();
  });

  it('selects the first matching S3 override for the artifact path', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          providers: `
s3:
  default:
    endpoint: s3.amazonaws.com
    region: us-east-1
    credentials:
      fromEnv: true
  Overrides:
    - bucketName: team-bucket
      keyPrefix: pipelines/team-a
      endpoint: https://custom-s3.example.com:9443
      region: custom-region
      disableSSL: false
      credentials:
        fromEnv: false
        secretRef:
          secretName: team-a-store
          accessKeyKey: access-key
          secretKeyKey: secret-key
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'team-bucket',
        key: 'pipelines/team-a/run/artifact',
      },
      'team-a',
    );

    expect(mockedGetConfigMap).toHaveBeenCalledWith('kfp-launcher', 'team-a');
    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: {
        accessKeyKey: 'access-key',
        disableSSL: 'false',
        endpoint: 'https://custom-s3.example.com:9443',
        forcePathStyle: 'true',
        fromEnv: 'false',
        maxRetries: '5',
        region: 'custom-region',
        secretKeyKey: 'secret-key',
        secretName: 'team-a-store',
      },
    });
  });

  it('maps gcs artifacts to the launcher gs provider format', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          providers: `
gs:
  default:
    credentials:
      fromEnv: false
      secretRef:
        secretName: gcs-credentials
        tokenKey: key.json
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 'gcs', bucket: 'bucket', key: 'artifact' },
      'kubeflow',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 'gs',
      Params: {
        fromEnv: 'false',
        secretName: 'gcs-credentials',
        tokenKey: 'key.json',
      },
    });
  });

  it('uses normal server defaults when the optional launcher config is absent', async () => {
    mockedGetConfigMap.mockResolvedValue([undefined, { message: 'not found' }]);

    await expect(
      getLauncherProviderInfo(
        { source: 'minio', bucket: 'mlpipeline', key: 'artifact' },
        'kubeflow',
      ),
    ).resolves.toBeUndefined();
  });

  it('rejects secret-backed provider entries with incomplete secret references', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          providers: `
s3:
  default:
    endpoint: custom-s3
    credentials:
      fromEnv: false
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo({ source: 's3', bucket: 'bucket', key: 'artifact' }, 'kubeflow'),
    ).rejects.toThrow('credentials are missing secretRef');
  });
});
