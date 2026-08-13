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

  it('rejects default credentials for an artifact outside defaultPipelineRoot', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket/pipelines/team-a',
          providers: `
s3:
  default:
    credentials:
      fromEnv: false
      secretRef:
        secretName: broad-store
        accessKeyKey: access-key
        secretKeyKey: secret-key
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-b/run/artifact' },
        'team-a',
      ),
    ).rejects.toThrow('is outside defaultPipelineRoot and has no explicit provider query');
  });

  it('inherits explicit endpoint settings from defaultPipelineRoot', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot:
            's3://team-bucket/pipelines/team-a?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph&disableSSL=false',
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/run/artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: {
        disableSSL: 'false',
        endpoint: 'https://ceph.example:9443',
        fromEnv: 'true',
        region: 'ceph',
      },
    });
  });

  it('prefers the pipeline-root query for artifacts under that root', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot:
            's3://team-bucket/pipelines/team-a?endpoint=https%3A%2F%2Ftrusted.example',
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'team-bucket',
        key: 'pipelines/team-a/model?endpoint=https%3A%2F%2Funtrusted.example',
      },
      'team-a',
    );

    expect(JSON.parse(result || '').Params.endpoint).toBe('https://trusted.example');
  });

  it('uses explicit artifact query settings outside defaultPipelineRoot', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'external-bucket',
        key: 'model?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
      },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: {
        endpoint: 'https://ceph.example:9443',
        fromEnv: 'true',
        region: 'ceph',
      },
    });
  });

  it('maps gcs artifacts to the launcher gs provider format', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 'gs://bucket',
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
    mockedGetConfigMap.mockResolvedValue([
      undefined,
      { additionalInfo: { code: 404, reason: 'NotFound' }, message: 'not found' },
    ]);

    await expect(
      getLauncherProviderInfo(
        { source: 'minio', bucket: 'mlpipeline', key: 'artifact' },
        'kubeflow',
      ),
    ).resolves.toBeUndefined();
  });

  it('surfaces ConfigMap read failures instead of silently using environment credentials', async () => {
    mockedGetConfigMap.mockResolvedValue([
      undefined,
      { additionalInfo: { code: 403, reason: 'Forbidden' }, message: 'read denied' },
    ]);

    await expect(
      getLauncherProviderInfo({ source: 'minio', bucket: 'mlpipeline', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow(
      'read denied. Verify that the UI service account can read the kfp-launcher ConfigMap',
    );
  });

  it('rejects invalid providers YAML with a corrective action', async () => {
    mockedGetConfigMap.mockResolvedValue([{ data: { providers: 's3: [unterminated' } }, undefined]);

    await expect(
      getLauncherProviderInfo({ source: 's3', bucket: 'bucket', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow('contains invalid YAML. Correct the providers entry and retry');
  });

  it('matches launcher behavior for an overrides-only provider without a default', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://bucket',
          providers: `
s3:
  Overrides:
    - bucketName: another-bucket
      credentials:
        fromEnv: true
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo({ source: 's3', bucket: 'bucket', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow('provider is missing default credentials');
  });

  it('rejects secret-backed provider entries with incomplete secret references', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://bucket',
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
