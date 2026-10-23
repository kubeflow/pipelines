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

import {
  buildArtifactUri,
  namespaceFromArtifactUri,
  namespaceFromArtifactStorageKey,
  normalizeArtifactOwnershipMode,
  requiresArtifactOwnershipValidation,
  resolveArtifactValidationTimeoutMs,
  validateArtifactKeyPrefix,
  validateArtifactKeySafety,
  validateArtifactNamespace,
  validateArtifactNotFound,
} from './artifact-validator.js';

describe('artifact-validator', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it.each([
    's3://bucket/custom/model/',
    's3://bucket/private-artifacts/team-b/model',
    'https://files.example/reports/a//b',
  ])('checks key safety independently of a namespace prefix: %s', (artifactUri) => {
    expect(validateArtifactKeySafety(artifactUri)).toEqual({ valid: true });
  });

  it.each([
    ['private-artifacts/team-b/model', 'team-b'],
    ['private-artifacts/team-b', 'team-b'],
    ['private-artifacts/team-b/', 'team-b'],
    ['private-artifacts/team-b/model%ZZ', 'team-b'],
    ['private-artifacts/%74eam-b/model', '%74eam-b'],
    ['%70rivate-artifacts/team-b/model', undefined],
    ['private-artifacts-other/team-b/model', undefined],
    ['other/private-artifacts/team-b/model', undefined],
  ])('matches storage namespaces without URI decoding: %s', (key, namespace) => {
    expect(namespaceFromArtifactStorageKey(key)).toBe(namespace);
  });

  it('matches custom multi-segment storage prefixes literally', () => {
    expect(
      namespaceFromArtifactStorageKey('tenant.data/models/team-b/model', 'tenant.data/models'),
    ).toBe('team-b');
    expect(
      namespaceFromArtifactStorageKey('tenantXdata/models/team-b/model', 'tenant.data/models'),
    ).toBeUndefined();
    expect(namespaceFromArtifactStorageKey('tenant.data/models/team-b', 'tenant.data/models')).toBe(
      'team-b',
    );
    expect(namespaceFromArtifactStorageKey('private-artifacts/team-b/model', '')).toBeUndefined();
  });

  it('does not broaden the no-record URI shortcut to namespace roots', () => {
    expect(validateArtifactKeyPrefix('s3://bucket/private-artifacts/team-a', 'team-a')).toEqual({
      valid: false,
      reason: 'artifact-not-found',
    });
  });

  it('extracts only a leading namespace key prefix', () => {
    expect(namespaceFromArtifactUri('s3://bucket/private-artifacts/team-a/run/output')).toBe(
      'team-a',
    );
    expect(
      namespaceFromArtifactUri('s3://bucket/other/private-artifacts/team-a/output'),
    ).toBeUndefined();
  });

  it('supports a custom key prefix', () => {
    expect(namespaceFromArtifactUri('gs://bucket/custom/team-a/output', 'custom')).toBe('team-a');
  });

  it('decodes the namespace segment once and rejects malformed encoding', () => {
    expect(namespaceFromArtifactUri('s3://bucket/private-artifacts/%74eam-a/run/output')).toBe(
      'team-a',
    );
    expect(
      namespaceFromArtifactUri('s3://bucket/private-artifacts/%ZZteam-a/run/output'),
    ).toBeUndefined();
  });

  it('accepts a matching namespace prefix for untracked objects', () => {
    expect(
      validateArtifactKeyPrefix(
        'minio://bucket/private-artifacts/team-a/run/executor.log',
        'team-a',
      ),
    ).toEqual({ valid: true, reason: 'prefix-match' });
  });

  it('accepts an encoded matching namespace and rejects an encoded mismatch', () => {
    expect(
      validateArtifactKeyPrefix(
        'minio://bucket/private-artifacts/%74eam-a/run/executor.log',
        'team-a',
      ),
    ).toEqual({ valid: true, reason: 'encoded-prefix-match' });
    expect(
      validateArtifactKeyPrefix(
        'minio://bucket/private-artifacts/%74eam-b/run/executor.log',
        'team-a',
      ),
    ).toEqual({
      actualNamespace: 'team-b',
      reason: 'prefix-namespace-mismatch',
      valid: false,
    });
  });

  it('allows an encoded standard-prefix alias only with exact ArtifactService evidence', async () => {
    const fetchSpy = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
          headers: { 'Content-Type': 'application/json' },
          status: 200,
        }),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ artifacts: [] }), {
          headers: { 'Content-Type': 'application/json' },
          status: 200,
        }),
      );
    vi.stubGlobal('fetch', fetchSpy);
    const artifactUri = 'minio://bucket/private-artifacts/%74eam-a/run/executor.log';

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });
    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: false, reason: 'artifact-not-found' });
    expect(fetchSpy).toHaveBeenCalledTimes(2);
  });

  it('accepts one trailing slash but rejects doubled trailing separators', () => {
    expect(
      validateArtifactKeyPrefix('minio://bucket/private-artifacts/team-a/run/output/', 'team-a'),
    ).toEqual({ valid: true, reason: 'prefix-match' });
    expect(
      validateArtifactKeyPrefix('minio://bucket/private-artifacts/team-a/run/output//', 'team-a'),
    ).toEqual({ valid: false, reason: 'key-not-normalized' });
  });

  it('short-circuits ownership lookup for a matching prefix with one trailing slash', async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace(
        'http://api-server',
        'minio://bucket/private-artifacts/team-a/run/output/',
        'team-a',
      ),
    ).resolves.toEqual({ valid: true, reason: 'prefix-match' });
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('does not interpret an artifact provider query as object-key path segments', () => {
    expect(
      validateArtifactKeyPrefix(
        's3://bucket/private-artifacts/team-a/model?endpoint=https://ceph.example:9443',
        'team-a',
      ),
    ).toEqual({ valid: true, reason: 'prefix-match' });
  });

  it('allows repeated HTTP separators to proceed to exact ownership lookup', () => {
    expect(validateArtifactKeyPrefix('https://example.com/reports/a//b', 'team-a')).toEqual({
      valid: false,
      reason: 'artifact-not-found',
    });
  });

  it('accepts a custom-root HTTP artifact with an exact native Artifact API match', async () => {
    const artifactUri = 'https://example.com/reports/a//b';
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });
  });

  it('rejects mismatched, absent, and non-normalized prefixes', () => {
    expect(
      validateArtifactKeyPrefix('minio://bucket/private-artifacts/team-b/output', 'team-a'),
    ).toEqual({
      actualNamespace: 'team-b',
      reason: 'prefix-namespace-mismatch',
      valid: false,
    });
    expect(validateArtifactKeyPrefix('minio://bucket/public/output', 'team-a')).toEqual({
      reason: 'artifact-not-found',
      valid: false,
    });
    expect(
      validateArtifactKeyPrefix(
        'minio://bucket/private-artifacts/team-a/../team-b/output',
        'team-a',
      ),
    ).toEqual({ reason: 'key-not-normalized', valid: false });
  });

  it.each(['mlmd-then-prefix', 'artifact-then-prefix'])(
    'normalizes %s to namespace-prefix fallback',
    (configuredMode) => {
      const ownershipMode = normalizeArtifactOwnershipMode(configuredMode);
      expect(ownershipMode).toBe('artifact-then-prefix');
      expect(
        validateArtifactNotFound(
          'minio://bucket/private-artifacts/team-a/run/executor.log',
          'team-a',
          ownershipMode,
        ),
      ).toEqual({ valid: true, reason: 'prefix-match' });
    },
  );

  it.each([
    ['mlmd-only', 'artifact-only'],
    ['MLMD-ONLY', 'artifact-only'],
    ['artifact-only', 'artifact-only'],
    ['invalid-mode', 'invalid'],
  ] as const)('normalizes %s to %s and denies prefix fallback', (configuredMode, expectedMode) => {
    const ownershipMode = normalizeArtifactOwnershipMode(configuredMode);
    expect(ownershipMode).toBe(expectedMode);
    expect(
      validateArtifactNotFound(
        'minio://bucket/private-artifacts/team-a/run/executor.log',
        'team-a',
        ownershipMode,
      ),
    ).toEqual({ valid: false, reason: 'artifact-not-found' });
  });

  it('accepts a custom-root ArtifactService match with forwarded caller identity', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', 's3://bucket/shared/output', 'team-a', {
        'kubeflow-userid': 'user@example.com',
      }),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });

    const [requestUrl, requestInit] = fetchSpy.mock.calls[0] as [string, RequestInit];
    const url = new URL(requestUrl);
    expect(url.pathname).toBe('/apis/v2/artifacts');
    expect(url.searchParams.get('namespace')).toBe('team-a');
    expect(JSON.parse(decodeURIComponent(url.searchParams.get('filter') || '{}'))).toEqual({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: 's3://bucket/shared/output' }],
    });
    expect(new Headers(requestInit.headers).get('kubeflow-userid')).toBe('user@example.com');
    expect(requestInit.signal).toBeInstanceOf(AbortSignal);
  });

  it('preserves a provider query in the exact ArtifactService ownership lookup', async () => {
    const artifactUri =
      's3://bucket/private-artifacts/%74eam-a/output%25+plus?endpoint=https://ceph.example:9443&token=a%2Bb';
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });

    const requestUrl = new URL(fetchSpy.mock.calls[0][0] as string);
    expect(JSON.parse(decodeURIComponent(requestUrl.searchParams.get('filter') || '{}'))).toEqual({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: artifactUri }],
    });
  });

  it('accepts an importer-created custom URI row without proving ownership of the bytes', async () => {
    // This native API response can be created by importing an arbitrary URI. The compatibility
    // decision accepts it; a secure backend byte-authorization boundary is still required.
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
          headers: { 'Content-Type': 'application/json' },
          status: 200,
        }),
      ),
    );

    await expect(
      validateArtifactNamespace('http://api-server', 's3://bucket/shared/output', 'team-a'),
    ).resolves.toEqual({
      reason: 'artifact-api-match',
      valid: true,
    });
  });

  it.each([
    's3://models/team-training/run-1/model',
    'gs://models/team-training/run-1/model',
    'minio://mlpipeline/artifacts/old-workflow/model',
    'https://files.example/reports/result.json',
    'http://files.example/reports/result.json',
    's3://models/team-training/run-1/model/',
    'gs://models/team-training/run-1/model/',
    'minio://mlpipeline/artifacts/old-workflow/model/',
    'https://files.example/reports/',
    's3://models/team-training/output%25+plus?endpoint=https://ceph.example:9443&token=a%2Bb',
    'gs://models/team-training/model/?endpoint=https://storage.example/path',
    'https://files.example/reports/result%20name.json?version=a%2Bb',
  ])('accepts an exact native row without rewriting its custom URI: %s', async (artifactUri) => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });
    const requestUrl = new URL(fetchSpy.mock.calls[0][0] as string);
    expect(requestUrl.searchParams.get('namespace')).toBe('team-a');
    expect(JSON.parse(decodeURIComponent(requestUrl.searchParams.get('filter') || '{}'))).toEqual({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: artifactUri }],
    });
  });

  it.each(['artifact-then-prefix', 'artifact-only'])(
    'requires a native row for a custom root in %s mode',
    async (mode) => {
      vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', mode);
      vi.resetModules();
      const { validateArtifactNamespace: validateWithConfiguredMode } =
        await import('./artifact-validator.js');
      const fetchSpy = vi
        .fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
            headers: { 'Content-Type': 'application/json' },
            status: 200,
          }),
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ artifacts: [] }), {
            headers: { 'Content-Type': 'application/json' },
            status: 200,
          }),
        );
      vi.stubGlobal('fetch', fetchSpy);
      const artifactUri = 's3://models/team-training/run-1/model';

      await expect(
        validateWithConfiguredMode('http://api-server', artifactUri, 'team-a'),
      ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });
      await expect(
        validateWithConfiguredMode('http://api-server', artifactUri, 'team-a'),
      ).resolves.toEqual({ valid: false, reason: 'artifact-not-found' });
      expect(fetchSpy).toHaveBeenCalledTimes(2);
    },
  );

  it.each([401, 403, 500])('fails closed on a native API %s for a custom root', async (status) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status })));

    await expect(
      validateArtifactNamespace('http://api-server', 's3://models/team-training/model', 'team-a'),
    ).resolves.toEqual({ valid: false, reason: 'artifact-api-unavailable' });
  });

  it('fails closed on a native API transport error for a custom root', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unavailable')));

    await expect(
      validateArtifactNamespace('http://api-server', 's3://models/team-training/model', 'team-a'),
    ).resolves.toEqual({ valid: false, reason: 'artifact-api-unavailable' });
  });

  it.each([
    's3://models/team-training/../team-b/model',
    's3://models/team-training/./model',
    's3://models/team-training//model',
    's3://models/team-training/model//',
    'gs://models/team-training/../team-b/model',
    'https://files.example/reports/../private/model',
    'https://files.example/reports/./model',
    'https://files.example/reports/\\model',
  ])('rejects an unsafe custom key before a native row can allow it: %s', async (artifactUri) => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a'),
    ).resolves.toEqual({ valid: false, reason: 'key-not-normalized' });
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('rejects a caller-owned record that points into another namespace prefix', async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace(
        'http://api-server',
        'minio://mlpipeline/private-artifacts/team-b/run/model',
        'team-a',
      ),
    ).resolves.toEqual({
      actualNamespace: 'team-b',
      reason: 'prefix-namespace-mismatch',
      valid: false,
    });
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('accepts a canonical namespace prefix without calling ArtifactService', async () => {
    const fetchSpy = vi.fn().mockRejectedValue(new Error('unavailable'));
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace(
        'http://api-server',
        's3://bucket/private-artifacts/team-a/run/executor.log',
        'team-a',
      ),
    ).resolves.toEqual({ valid: true, reason: 'prefix-match' });
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('preserves mlmd-only as strict ArtifactService-only validation', async () => {
    vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', 'mlmd-only');
    vi.resetModules();
    const { validateArtifactNamespace: validateWithStrictMode } =
      await import('./artifact-validator.js');
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateWithStrictMode(
        'http://api-server',
        's3://bucket/private-artifacts/team-a/run/executor.log',
        'team-a',
      ),
    ).resolves.toEqual({ valid: false, reason: 'artifact-not-found' });
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it('fails closed when ArtifactService is unavailable in strict mode', async () => {
    vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', 'mlmd-only');
    vi.resetModules();
    const { validateArtifactNamespace: validateWithConfiguredMode } =
      await import('./artifact-validator.js');
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('unavailable')));

    await expect(
      validateWithConfiguredMode(
        'http://api-server',
        's3://bucket/private-artifacts/team-a/output',
        'team-a',
      ),
    ).resolves.toEqual({ valid: false, reason: 'artifact-api-unavailable' });
  });

  it('preserves the legacy MLMD timeout and prefers the replacement setting', () => {
    expect(resolveArtifactValidationTimeoutMs({ MLMD_VALIDATION_TIMEOUT_MS: '30000' })).toBe(30000);
    expect(
      resolveArtifactValidationTimeoutMs({
        ARTIFACT_VALIDATION_TIMEOUT_MS: '12000',
        MLMD_VALIDATION_TIMEOUT_MS: '30000',
      }),
    ).toBe(12000);
    expect(resolveArtifactValidationTimeoutMs({ ARTIFACT_VALIDATION_TIMEOUT_MS: 'invalid' })).toBe(
      5000,
    );
  });

  it('normalizes the GCS scheme when constructing an artifact URI', () => {
    expect(buildArtifactUri('gcs', 'bucket', 'path/output')).toBe('gs://bucket/path/output');
  });

  it('validates ownership only for object-store and remote artifact sources', () => {
    expect(requiresArtifactOwnershipValidation('minio')).toBe(true);
    expect(requiresArtifactOwnershipValidation('s3')).toBe(true);
    expect(requiresArtifactOwnershipValidation('gcs')).toBe(true);
    expect(requiresArtifactOwnershipValidation('http')).toBe(true);
    expect(requiresArtifactOwnershipValidation('https')).toBe(true);
    expect(requiresArtifactOwnershipValidation('volume')).toBe(false);
  });
});
