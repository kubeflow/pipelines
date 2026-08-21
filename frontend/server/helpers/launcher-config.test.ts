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
import { getLauncherProviderInfo, TEST_ONLY } from './launcher-config.js';

vi.mock('../k8s-helper.js', () => ({ getConfigMap: vi.fn() }));

const mockedGetConfigMap = vi.mocked(getConfigMap);

describe('getLauncherProviderInfo', () => {
  beforeEach(() => {
    mockedGetConfigMap.mockReset();
    TEST_ONLY.clearLauncherConfigurationCache();
  });

  it('coalesces and caches launcher configuration reads by namespace', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);

    await Promise.all([
      getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/first' },
        'team-a',
      ),
      getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/second' },
        'team-a',
      ),
    ]);
    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/third' },
      'team-a',
    );

    expect(mockedGetConfigMap).toHaveBeenCalledTimes(1);
  });

  it('keeps launcher configuration caches isolated by namespace', async () => {
    mockedGetConfigMap.mockImplementation(async (_name, namespace) => [
      { data: { defaultPipelineRoot: `s3://${namespace}-bucket/root` } },
      undefined,
    ]);

    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-a-bucket', key: 'root/artifact' },
      'team-a',
    );
    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-b-bucket', key: 'root/artifact' },
      'team-b',
    );

    expect(mockedGetConfigMap).toHaveBeenCalledTimes(2);
    expect(mockedGetConfigMap).toHaveBeenNthCalledWith(1, 'kfp-launcher', 'team-a');
    expect(mockedGetConfigMap).toHaveBeenNthCalledWith(2, 'kfp-launcher', 'team-b');
  });

  it('refreshes cached launcher configuration after the TTL', async () => {
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_000);
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);
    const coordinates = {
      source: 's3' as const,
      bucket: 'team-bucket',
      key: 'pipelines/team-a/artifact',
    };

    await getLauncherProviderInfo(coordinates, 'team-a');
    nowSpy.mockReturnValue(31_001);
    await getLauncherProviderInfo(coordinates, 'team-a');

    expect(mockedGetConfigMap).toHaveBeenCalledTimes(2);
    nowSpy.mockRestore();
  });

  it('prunes expired entries when another namespace is requested', async () => {
    const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_000);
    mockedGetConfigMap.mockImplementation(async (_name, namespace) => [
      { data: { defaultPipelineRoot: `s3://${namespace}-bucket/root` } },
      undefined,
    ]);

    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-a-bucket', key: 'root/artifact' },
      'team-a',
    );
    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-b-bucket', key: 'root/artifact' },
      'team-b',
    );
    nowSpy.mockReturnValue(31_001);
    await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-c-bucket', key: 'root/artifact' },
      'team-c',
    );

    expect(TEST_ONLY.getLauncherConfigurationCacheKeys()).toEqual(['team-c']);
    nowSpy.mockRestore();
  });

  it('bounds namespace churn with least-recently-used eviction', async () => {
    const maxEntries = TEST_ONLY.launcherConfigurationCacheMaxEntries;
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://bucket/root' } },
      undefined,
    ]);

    await Promise.all(
      Array.from({ length: maxEntries + 1 }, (_, index) =>
        getLauncherProviderInfo(
          { source: 's3', bucket: 'bucket', key: 'root/artifact' },
          `team-${index}`,
        ),
      ),
    );

    const cacheKeys = TEST_ONLY.getLauncherConfigurationCacheKeys();
    expect(cacheKeys).toHaveLength(maxEntries);
    expect(cacheKeys).not.toContain('team-0');
    expect(cacheKeys).toContain(`team-${maxEntries}`);

    await getLauncherProviderInfo(
      { source: 's3', bucket: 'bucket', key: 'root/artifact' },
      'team-0',
    );
    expect(mockedGetConfigMap).toHaveBeenCalledTimes(maxEntries + 2);
    expect(TEST_ONLY.getLauncherConfigurationCacheKeys()).not.toContain('team-1');
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

  it('matches overrides against the artifact parent prefix like the launcher', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    endpoint: default-s3.example.com
    credentials:
      fromEnv: true
  Overrides:
    - bucketName: team-bucket
      keyPrefix: pipelines/team-a
      endpoint: override-s3.example.com
      credentials:
        fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const boundaryResult = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a' },
      'team-a',
    );
    expect(JSON.parse(boundaryResult || '').Params.endpoint).toBe('default-s3.example.com');

    const nestedResult = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/artifact' },
      'team-a',
    );
    expect(JSON.parse(nestedResult || '').Params.endpoint).toBe('override-s3.example.com');
  });

  it('inherits default S3 settings when a matching override uses empty strings', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    endpoint: default-s3.example.com
    region: default-region
    credentials:
      fromEnv: true
  Overrides:
    - bucketName: team-bucket
      keyPrefix: pipelines/team-a
      endpoint: ''
      region: ''
      credentials:
        fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toMatchObject({
      endpoint: 'default-s3.example.com',
      region: 'default-region',
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

  it.each([
    ['URI-escaped spaces', 'root%20dir', 'root%20dir/run/artifact', 'uri'],
    ['decoded spaces', 'root%20dir', 'root dir/run/artifact', 'storage'],
    ['URI-escaped Unicode', '%E6%A8%A1%E5%9E%8B', '%E6%A8%A1%E5%9E%8B/run/artifact', 'uri'],
    ['decoded Unicode', '%E6%A8%A1%E5%9E%8B', '模型/run/artifact', 'storage'],
  ])(
    'matches %s in defaultPipelineRoot against artifact coordinates',
    async (_description, encodedRoot, artifactKey, keyEncoding) => {
      mockedGetConfigMap.mockResolvedValue([
        {
          data: {
            defaultPipelineRoot:
              `s3://team-bucket/${encodedRoot}` +
              '?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
          },
        },
        undefined,
      ]);

      const result = await getLauncherProviderInfo(
        {
          source: 's3',
          bucket: 'team-bucket',
          key: artifactKey,
          keyEncoding: keyEncoding as 'storage' | 'uri',
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
    },
  );

  it('does not treat a literal escaped sibling as the decoded pipeline root', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket/root%20dir',
          providers: `
s3:
  default:
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        {
          source: 's3',
          bucket: 'team-bucket',
          key: 'root%20dir/run/artifact',
          keyEncoding: 'storage',
        },
        'team-a',
      ),
    ).rejects.toThrow('is outside defaultPipelineRoot and has no explicit provider query');
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
        key: 'pipelines/team-a/model',
        artifactUriQuery: 'endpoint=https%3A%2F%2Funtrusted.example',
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
        key: 'model',
        artifactUriQuery: 'endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph',
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

  it('preserves the first duplicate S3 query value used by Go Cloud', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'external-bucket',
        key: 'model',
        artifactUriQuery:
          'anonymous=1&anonymous=false&endpoint=https%3A%2F%2Fs3.us-west-2.amazonaws.com',
      },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: {
        anonymous: '1',
        endpoint: 'https://s3.us-west-2.amazonaws.com',
        fromEnv: 'true',
      },
    });
  });

  it.each(['disableSSL=true', 'anonymuos=true'])(
    'rejects non-native raw S3 query option %s',
    async (artifactUriQuery) => {
      mockedGetConfigMap.mockResolvedValue([
        { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
        undefined,
      ]);

      await expect(
        getLauncherProviderInfo(
          {
            source: 's3',
            bucket: 'external-bucket',
            key: 'model',
            artifactUriQuery,
          },
          'team-a',
        ),
      ).rejects.toThrow('is not supported by Go Cloud');
    },
  );

  it('treats a fragment marker as object-key data when checking the pipeline root', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket/pipelines/team-a/model',
          providers: `
s3:
  default:
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        {
          source: 's3',
          bucket: 'team-bucket',
          key: 'pipelines/team-a/model#checkpoint',
        },
        'team-a',
      ),
    ).rejects.toThrow('is outside defaultPipelineRoot and has no explicit provider query');
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

  it('keeps configured GCS credentials authoritative over artifact URI queries', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 'gs://bucket/root?anonymous=true',
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
      {
        source: 'gcs',
        bucket: 'bucket',
        key: 'root/artifact',
      },
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

  it('keeps GCS URI query settings when the provider block has no policy', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 'gs://bucket/root?access_id=-',
          providers: 'gs: {}',
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 'gcs', bucket: 'bucket', key: 'root/artifact' },
      'kubeflow',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 'gs',
      Params: { access_id: '-', fromEnv: 'true' },
    });
  });

  it('keeps GCS URI query settings when the provider default is null', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 'gs://bucket/root?access_id=-',
          providers: `
gs:
  default:
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 'gcs', bucket: 'bucket', key: 'root/artifact' },
      'kubeflow',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 'gs',
      Params: { access_id: '-', fromEnv: 'true' },
    });
  });

  it('uses normal server defaults under the default pipeline root when config is absent', async () => {
    mockedGetConfigMap.mockResolvedValue([
      undefined,
      { additionalInfo: { code: 404, reason: 'NotFound' }, message: 'not found' },
    ]);

    await expect(
      getLauncherProviderInfo(
        { source: 'minio', bucket: 'mlpipeline', key: 'v2/artifacts/run/artifact' },
        'kubeflow',
      ),
    ).resolves.toBeUndefined();
  });

  it('preserves legacy environment-credential reads when launcher config is absent', async () => {
    mockedGetConfigMap.mockResolvedValue([
      undefined,
      { additionalInfo: { code: 404, reason: 'NotFound' }, message: 'not found' },
    ]);

    await expect(
      getLauncherProviderInfo(
        { source: 'minio', bucket: 'mlpipeline', key: 'outside/run/artifact' },
        'kubeflow',
      ),
    ).resolves.toBeUndefined();
  });

  it('preserves legacy environment-credential reads when the ConfigMap has no provider policy', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 'minio://mlpipeline/v2/artifacts' } },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        {
          source: 'minio',
          bucket: 'mlpipeline',
          key: 'artifacts/my-workflow/pod/mlpipeline-ui-metadata.tgz',
        },
        'kubeflow',
      ),
    ).resolves.toBeUndefined();
  });

  it('surfaces ConfigMap read failures instead of silently using environment credentials', async () => {
    mockedGetConfigMap
      .mockResolvedValueOnce([
        undefined,
        { additionalInfo: { code: 403, reason: 'Forbidden' }, message: 'read denied' },
      ])
      .mockResolvedValueOnce([
        undefined,
        { additionalInfo: { code: 404, reason: 'NotFound' }, message: 'not found' },
      ]);

    await expect(
      getLauncherProviderInfo({ source: 'minio', bucket: 'mlpipeline', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow(
      'read denied. Verify that the UI service account can read the kfp-launcher ConfigMap',
    );
    await expect(
      getLauncherProviderInfo(
        { source: 'minio', bucket: 'mlpipeline', key: 'v2/artifacts/artifact' },
        'team-a',
      ),
    ).resolves.toBeUndefined();
    expect(mockedGetConfigMap).toHaveBeenCalledTimes(2);
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
