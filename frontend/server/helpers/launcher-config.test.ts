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

describe('launcher-compatible float formatting', () => {
  it.each([
    [0x483675c8, '186839.12'],
    [0xca2ae84d, '-2.8001472e+06'],
    [0x49b46fa2, '1.4781322e+06'],
    [0x48f20474, '495651.62'],
    [0xc52d4e80, '-2772.9062'],
    [0x46c1f4a0, '24826.312'],
  ])('matches Go shortest-mode output for float32 bits %s', (bits, expected) => {
    const bytes = new ArrayBuffer(4);
    const view = new DataView(bytes);
    view.setUint32(0, bits);

    expect(TEST_ONLY.formatFloat32(view.getFloat32(0))).toBe(expected);
  });
});

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

  it('matches launcher case-insensitive provider and field decoding', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
S3:
  Default:
    Endpoint: mixed-case.example.com
    Region: mixed-region
    DisableSSL: on
    Credentials:
      FromEnv: yes
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: {
        disableSSL: 'true',
        endpoint: 'mixed-case.example.com',
        forcePathStyle: 'true',
        fromEnv: 'true',
        maxRetries: '5',
        region: 'mixed-region',
      },
    });
  });

  it.each(['yes', 'on'])(
    'accepts launcher-compatible unquoted fromEnv: %s boolean syntax',
    async (fromEnv) => {
      mockedGetConfigMap.mockResolvedValue([
        {
          data: {
            defaultPipelineRoot: 's3://team-bucket',
            providers: `
s3:
  default:
    credentials:
      fromEnv: ${fromEnv}
`,
          },
        },
        undefined,
      ]);

      const result = await getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'artifact' },
        'team-a',
      );

      expect(JSON.parse(result || '').Params.fromEnv).toBe('true');
    },
  );

  it.each(['TrUe', 'yEs', 'oN', 'FaLsE', 'nO', 'oFf'])(
    'rejects mixed-case YAML boolean token %s like the launcher',
    async (value) => {
      mockedGetConfigMap.mockResolvedValue([
        {
          data: {
            defaultPipelineRoot: 's3://team-bucket',
            providers: `
s3:
  default:
    credentials:
      fromEnv: ${value}
`,
          },
        },
        undefined,
      ]);

      await expect(
        getLauncherProviderInfo({ source: 's3', bucket: 'team-bucket', key: 'artifact' }, 'team-a'),
      ).rejects.toThrow('providers.s3.default.credentials.fromEnv must be a boolean');
    },
  );

  it.each([
    ['y', 'true'],
    ['Y', 'true'],
    ['n', 'false'],
    ['N', 'false'],
  ])(
    'accepts launcher-compatible unquoted disableSSL: %s boolean syntax',
    async (value, expected) => {
      mockedGetConfigMap.mockResolvedValue([
        {
          data: {
            defaultPipelineRoot: 's3://team-bucket',
            providers: `
s3:
  default:
    disableSSL: ${value}
    credentials:
      fromEnv: true
`,
          },
        },
        undefined,
      ]);

      const result = await getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'artifact' },
        'team-a',
      );

      expect(JSON.parse(result || '').Params.disableSSL).toBe(expected);
    },
  );

  it('treats null optional provider scalars as omitted launcher values', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    endpoint: null
    region: null
    disableSSL: null
    forcePathStyle: null
    maxRetries: null
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toMatchObject({
      disableSSL: 'false',
      endpoint: '',
      forcePathStyle: 'true',
      fromEnv: 'true',
      maxRetries: '5',
      region: '',
    });
  });

  it.each([
    ['077', '63'],
    ['0o77', '63'],
    ['0x3f', '63'],
    ['0b111111', '63'],
    ['6_3', '63'],
    ['0_', '0'],
  ])('matches yaml.v2 integer %s and string scalar decoding', async (maxRetries, expected) => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    endpoint: 2026-01-01
    maxRetries: ${maxRetries}
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toMatchObject({
      endpoint: '2026-01-01',
      maxRetries: expected,
    });
  });

  it('applies YAML merge keys before validating recognized provider fields', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
provider-defaults: &provider-defaults
  endpoint: merged.example.com
  region: us-east-2
credentials: &credentials
  fromEnv: true
s3:
  default:
    <<: *provider-defaults
    endpoint: explicit.example.com
    credentials:
      <<: *credentials
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toMatchObject({
      endpoint: 'explicit.example.com',
      fromEnv: 'true',
      region: 'us-east-2',
    });
  });

  it.each([
    ['numeric', '456', '456'],
    ['boolean', 'true', 'true'],
    ['incomplete binary prefix', '0b_', '0b_'],
    ['underscored float', '1_2.0', '12'],
    ['positive infinity', '.inf', '+Inf'],
    ['integer beyond JavaScript safe range', '9007199254740993', '9007199254740993'],
    ['float32-rounded decimal', '1.23456789', '1.2345679'],
    ['float32 scientific decimal', '123456789.0', '1.2345679e+08'],
    ['six-digit-exponent decimal', '14456108.0', '1.4456108e+07'],
    ['halfway-rounded decimal', '186839.125', '186839.12'],
    ['small float32 scientific decimal', '0.00001', '1e-05'],
    ['negative zero', '-0.0', '-0'],
  ])('coerces a %s scalar into a launcher string field', async (_name, value, expected) => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    credentials:
      fromEnv: true
  overrides:
    - bucketName: team-bucket
      keyPrefix: ${value}
      endpoint: override.example.com
      credentials:
        fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: `${expected}/artifact` },
      'team-a',
    );

    expect(JSON.parse(result || '').Params.endpoint).toBe('override.example.com');
  });

  it.each([
    ['mixed-case infinity', '.iNf'],
    ['mixed-case NaN', '.nAn'],
  ])('preserves a launcher-unrecognized %s as string data', async (_name, value) => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    endpoint: ${value}
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params.endpoint).toBe(value);
  });

  it('bounds integer resolution for very large ignored YAML scalars', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
ignoredInvalidBinary: 0b${'_'.repeat(50_000)}2
ignoredLargeDecimal: ${'9'.repeat(100_000)}
s3:
  default:
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params.fromEnv).toBe('true');
  }, 1_000);

  it('does not coerce an incomplete binary prefix into numeric policy', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
s3:
  default:
    maxRetries: 0b_
    credentials:
      fromEnv: true
`,
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo({ source: 's3', bucket: 'team-bucket', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow('providers.s3.default.maxRetries must be a number');
  });

  it.each([
    [
      'provider entry',
      `
s3:
  default:
    __proto__:
      credentials:
        fromEnv: true
`,
      'missing default credentials',
    ],
    [
      'credentials',
      `
s3:
  default:
    credentials:
      __proto__:
        fromEnv: true
`,
      'credentials are missing secretRef',
    ],
    [
      'secret reference',
      `
s3:
  default:
    credentials:
      secretRef:
        __proto__:
          secretName: artifact-secret
`,
      'credentials are missing secretRef',
    ],
    [
      'override',
      `
s3:
  default:
    credentials:
      fromEnv: true
  overrides:
    - bucketName: team-bucket
      keyPrefix: ''
      __proto__:
        credentials:
          fromEnv: true
`,
      'override is missing credentials',
    ],
  ])(
    'does not inherit prototype-backed policy at the %s mapping',
    async (_name, providers, error) => {
      mockedGetConfigMap.mockResolvedValue([
        { data: { defaultPipelineRoot: 's3://team-bucket', providers } },
        undefined,
      ]);

      await expect(
        getLauncherProviderInfo({ source: 's3', bucket: 'team-bucket', key: 'artifact' }, 'team-a'),
      ).rejects.toThrow(error);
    },
  );

  it('ignores prototype keys at the providers and provider-config mappings', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket',
          providers: `
__proto__:
  s3:
    default:
      credentials:
        fromEnv: false
s3:
  __proto__:
    default:
      credentials:
        fromEnv: false
`,
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 's3', bucket: 'team-bucket', key: 'artifact' },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toEqual({ fromEnv: 'true' });
  });

  it.each([
    ['provider', 's3: {}\nS3: {}', 'providers'],
    ['default', 's3:\n  default: {}\n  Default: {}', 'providers.s3'],
    ['overrides', 's3:\n  overrides: []\n  Overrides: []', 'providers.s3'],
  ])('rejects case-colliding %s aliases before selecting a provider', async (_name, yaml, path) => {
    mockedGetConfigMap.mockResolvedValue([{ data: { providers: yaml } }, undefined]);

    await expect(
      getLauncherProviderInfo({ source: 's3', bucket: 'bucket', key: 'artifact' }, 'team-a'),
    ).rejects.toThrow(`kfp-launcher ${path} contains case-colliding keys`);
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
            's3://team-bucket/pipelines/team-a?endpoint=https%3A%2F%2Fceph.example%3A9443&region=ceph&disable_https=false',
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
        disable_https: 'false',
        endpoint: 'https://ceph.example:9443',
        fromEnv: 'true',
        nativeQuery: 'true',
        region: 'ceph',
      },
    });
  });

  it('normalizes a query inherited from a MinIO pipeline root to native S3 settings', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 'minio://mlpipeline/v2/artifacts?anonymous=true',
        },
      },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      { source: 'minio', bucket: 'mlpipeline', key: 'v2/artifacts/run/model' },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: { anonymous: 'true', fromEnv: 'true', nativeQuery: 'true' },
    });
  });

  it('rejects an invalid anonymous value inherited from the pipeline root', async () => {
    mockedGetConfigMap.mockResolvedValue([
      {
        data: {
          defaultPipelineRoot: 's3://team-bucket/pipelines/team-a?anonymous=bogus',
        },
      },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        { source: 's3', bucket: 'team-bucket', key: 'pipelines/team-a/run/model' },
        'team-a',
      ),
    ).rejects.toThrow('Invalid boolean value');
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
          nativeQuery: 'true',
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
        nativeQuery: 'true',
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
        nativeQuery: 'true',
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

  it('normalizes a raw MinIO query to the runtime S3 provider', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 'minio://mlpipeline/v2/artifacts' } },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 'minio',
        bucket: 'external-bucket',
        key: 'model',
        artifactUriQuery: 'anonymous=true',
      },
      'team-a',
    );

    expect(JSON.parse(result || '')).toEqual({
      Provider: 's3',
      Params: { anonymous: 'true', fromEnv: 'true', nativeQuery: 'true' },
    });
  });

  it.each(['store.example:9000', 'ftp://store.example'])(
    'rejects native S3 endpoint %s that Go Cloud cannot use',
    async (endpoint) => {
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
            artifactUriQuery: `endpoint=${encodeURIComponent(endpoint)}`,
          },
          'team-a',
        ),
      ).rejects.toThrow('absolute HTTP(S) URL');
    },
  );

  it('accepts a native S3 endpoint with an uppercase HTTPS scheme', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'external-bucket',
        key: 'model',
        artifactUriQuery: `endpoint=${encodeURIComponent('HTTPS://store.example/base')}`,
      },
      'team-a',
    );

    expect(JSON.parse(result || '').Params.endpoint).toBe('HTTPS://store.example/base');
  });

  it.each(['disableSSL=true', 'anonymuos=true'])(
    'rejects non-native raw MinIO query option %s',
    async (artifactUriQuery) => {
      mockedGetConfigMap.mockResolvedValue([
        { data: { defaultPipelineRoot: 'minio://mlpipeline/v2/artifacts' } },
        undefined,
      ]);

      await expect(
        getLauncherProviderInfo(
          {
            source: 'minio',
            bucket: 'external-bucket',
            key: 'model',
            artifactUriQuery,
          },
          'team-a',
        ),
      ).rejects.toThrow('is not supported by Go Cloud');
    },
  );

  it.each([
    ['ssetype=', 'ssetype'],
    ['ssetype=invalid', 'ssetype'],
    ['kmskeyid=', 'kmskeyid'],
  ])('rejects invalid native S3 query value %s', async (artifactUriQuery, option) => {
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
    ).rejects.toThrow(option);
  });

  it('accepts valid native S3 write options without applying them to reads', async () => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 's3://team-bucket/pipelines/team-a' } },
      undefined,
    ]);

    const result = await getLauncherProviderInfo(
      {
        source: 's3',
        bucket: 'external-bucket',
        key: 'model',
        artifactUriQuery: 'ssetype=aws%3Akms&kmskeyid=key-1',
      },
      'team-a',
    );

    expect(JSON.parse(result || '').Params).toMatchObject({
      fromEnv: 'true',
      kmskeyid: 'key-1',
      ssetype: 'aws:kms',
    });
  });

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

  it.each([
    ['unknown native option', 'not_a_gcs_option=true', 'is not supported by Go Cloud'],
    [
      'unsupported private key path',
      'private_key_path=%2Fvar%2Frun%2Fkey.pem',
      'is not supported by frontend artifact reads',
    ],
  ])('rejects GCS query %s', async (_name, artifactUriQuery, expectedMessage) => {
    mockedGetConfigMap.mockResolvedValue([
      { data: { defaultPipelineRoot: 'gs://bucket/root' } },
      undefined,
    ]);

    await expect(
      getLauncherProviderInfo(
        {
          source: 'gcs',
          bucket: 'external-bucket',
          key: 'artifact',
          artifactUriQuery,
        },
        'kubeflow',
      ),
    ).rejects.toThrow(expectedMessage);
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

  it('uses server defaults for a real Kubernetes client 404 shape', async () => {
    mockedGetConfigMap.mockResolvedValue([
      undefined,
      {
        additionalInfo: '{"kind":"Status","reason":"NotFound","code":404}',
        message: 'Could not get configMap kfp-launcher in namespace kubeflow',
        statusCode: 404,
      },
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

  it.each([
    ['provider', 's3: malformed', 'providers.s3 must be a YAML object'],
    ['timestamp provider', 's3: 2026-01-01', 'providers.s3 must be a YAML object'],
    ['binary provider', 's3: !!binary bWFsZm9ybWVk', 'providers.s3 must be a YAML object'],
    ['default', 's3:\n  default: malformed', 'providers.s3.default must be a YAML object'],
    ['overrides', 's3:\n  Overrides: malformed', 'providers.s3.overrides must be a YAML list'],
    [
      'credentials',
      's3:\n  default:\n    credentials: malformed',
      'providers.s3.default.credentials must be a YAML object',
    ],
    [
      'secret reference',
      's3:\n  default:\n    credentials:\n      fromEnv: false\n      secretRef: malformed',
      'providers.s3.default.credentials.secretRef must be a YAML object',
    ],
  ])(
    'rejects a malformed nested %s instead of using environment credentials',
    async (_, yaml, error) => {
      mockedGetConfigMap.mockResolvedValue([{ data: { providers: yaml } }, undefined]);

      await expect(
        getLauncherProviderInfo({ source: 's3', bucket: 'bucket', key: 'artifact' }, 'team-a'),
      ).rejects.toThrow(error);
    },
  );

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
