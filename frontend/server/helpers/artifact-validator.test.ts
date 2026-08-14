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
  normalizeArtifactOwnershipMode,
  requiresArtifactOwnershipValidation,
  resolveArtifactValidationTimeoutMs,
  validateArtifactKeyPrefix,
  validateArtifactNamespace,
  validateArtifactNotFound,
} from './artifact-validator.js';

describe('artifact-validator', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
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

  it('accepts a matching namespace prefix for untracked objects', () => {
    expect(
      validateArtifactKeyPrefix(
        'minio://bucket/private-artifacts/team-a/run/executor.log',
        'team-a',
      ),
    ).toEqual({ valid: true, reason: 'prefix-match' });
  });

  it('does not interpret an artifact provider query as object-key path segments', () => {
    expect(
      validateArtifactKeyPrefix(
        's3://bucket/private-artifacts/team-a/model?endpoint=https://ceph.example:9443',
        'team-a',
      ),
    ).toEqual({ valid: true, reason: 'prefix-match' });
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

  it('accepts a custom-root ArtifactService match only for namespace-isolated reads', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace(
        'http://api-server',
        's3://bucket/shared/output',
        'team-a',
        {
          'kubeflow-userid': 'user@example.com',
        },
        true,
      ),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });

    const [requestUrl, requestInit] = fetchSpy.mock.calls[0] as [string, RequestInit];
    const url = new URL(requestUrl);
    expect(url.pathname).toBe('/apis/v2beta1/artifacts');
    expect(url.searchParams.get('namespace')).toBe('team-a');
    expect(JSON.parse(decodeURIComponent(url.searchParams.get('filter') || '{}'))).toEqual({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: 's3://bucket/shared/output' }],
    });
    expect(new Headers(requestInit.headers).get('kubeflow-userid')).toBe('user@example.com');
    expect(requestInit.signal).toBeInstanceOf(AbortSignal);
  });

  it('preserves a provider query in the exact ArtifactService ownership lookup', async () => {
    const artifactUri =
      's3://bucket/shared/output%25+plus?endpoint=https://ceph.example:9443&token=a%2Bb';
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ artifacts: [{ artifact_id: 'artifact-1' }] }), {
        headers: { 'Content-Type': 'application/json' },
        status: 200,
      }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    await expect(
      validateArtifactNamespace('http://api-server', artifactUri, 'team-a', undefined, true),
    ).resolves.toEqual({ valid: true, reason: 'artifact-api-match' });

    const requestUrl = new URL(fetchSpy.mock.calls[0][0] as string);
    expect(JSON.parse(decodeURIComponent(requestUrl.searchParams.get('filter') || '{}'))).toEqual({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: artifactUri }],
    });
  });

  it('rejects a custom-root row when the central server would read it directly', async () => {
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
      reason: 'custom-root-requires-namespace-isolation',
      valid: false,
    });
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
