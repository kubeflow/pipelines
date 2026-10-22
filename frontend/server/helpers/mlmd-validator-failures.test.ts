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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

type ContextFixture = { contextType: string; namespace?: string };
type ArtifactFixture = { id: number; contexts: ContextFixture[] };
const fixture = vi.hoisted(() => ({
  missing: false,
  artifacts: [] as ArtifactFixture[],
  failedContextIds: [] as number[],
  requestedUris: [] as string[],
}));

// Stub the protobuf transport, not the validator. Requests carry an artifact ID so
// concurrent context lookups can independently succeed, fail or report a mismatch.
vi.mock('module', () => ({
  createRequire: () => () => {
    if (fixture.missing) throw new Error('sensitive bundle path');
    class Request {
      id = 0;
      setUrisList(uris: string[]) {
        fixture.requestedUris = uris;
      }
      setArtifactId(id: number) {
        this.id = id;
      }
      serializeBinary() {
        return new Uint8Array([this.id]);
      }
    }
    return {
      GetArtifactsByURIRequest: Request,
      GetContextsByArtifactRequest: Request,
      GetArtifactsByURIResponse: {
        deserializeBinary: () => ({
          getArtifactsList: () => fixture.artifacts.map(({ id }) => ({ getId: () => id })),
        }),
      },
      GetContextsByArtifactResponse: {
        deserializeBinary: (bytes: Uint8Array) => ({
          getContextsList: () =>
            fixture.artifacts
              .find(({ id }) => id === bytes[0])!
              .contexts.map(({ contextType, namespace }) => ({
                getType: () => contextType,
                getCustomPropertiesMap: () =>
                  new Map(
                    namespace === undefined
                      ? []
                      : [['namespace', { getValueCase: () => 1, getStringValue: () => namespace }]],
                  ),
              })),
        }),
      },
      Value: { ValueCase: { STRING_VALUE: 1 } },
    };
  },
}));

const matchingContext = { contextType: 'system.PipelineRun', namespace: 'team-a' };
const foreignContext = { contextType: 'system.PipelineRun', namespace: 'team-b' };
const customUri = 's3://bucket/custom/model';

beforeEach(() => {
  vi.resetModules();
  vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', 'mlmd-then-prefix');
  vi.stubEnv('ARTIFACT_NAMESPACE_KEY_PREFIX', 'private-artifacts');
  fixture.missing = false;
  fixture.artifacts = [{ id: 1, contexts: [matchingContext] }];
  fixture.failedContextIds = [];
  fixture.requestedUris = [];
  vi.spyOn(console, 'warn').mockImplementation(() => {});
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, options: RequestInit) => {
      const id = new Uint8Array(options.body as Uint8Array)[5];
      if (url.endsWith('/GetContextsByArtifact') && fixture.failedContextIds.includes(id)) {
        throw new Error('sensitive context token');
      }
      return new Response(new Uint8Array([0, 0, 0, 0, 1, id]));
    }),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});

async function validate(uri = customUri) {
  const { validateArtifactNamespace } = await import('./mlmd-validator.js');
  return validateArtifactNamespace('http://mlmd', uri, 'team-a');
}

describe('2.17 compatibility through the actual MLMD validator', () => {
  it.each([
    's3://bucket/custom/model',
    's3://another-bucket/old-team-a-training/model',
    'gs://bucket/projects/team-a/model',
    'https://storage.example/legacy/model',
    's3://bucket/custom/my%20model',
    's3://bucket/custom/%E2%82%AC-model',
    'minio://mlpipeline/private-artifacts/team-a/model',
    's3://bucket/custom/model/',
    'minio://mlpipeline/private-artifacts/team-a/model/',
  ])('accepts a tracked artifact at %s without a namespace-prefix requirement', async (uri) => {
    expect(await validate(uri)).toEqual({ valid: true });
    expect(fixture.requestedUris).toEqual([uri]);
  });

  it.each([
    ['no contexts', []],
    ['no namespace property', [{ contextType: 'system.PipelineRun' }]],
    ['no PipelineRun context', [{ contextType: 'system.Pipeline', namespace: 'team-b' }]],
  ] as const)('preserves reads with %s', async (_scenario, contexts) => {
    fixture.artifacts = [{ id: 1, contexts: [...contexts] }];
    expect(await validate()).toEqual({ valid: true, reason: 'no-evidence' });
  });

  it.each([false, true])(
    'accepts matching and missing contexts (missing first: %s)',
    async (first) => {
      fixture.artifacts = [
        { id: 1, contexts: first ? [] : [matchingContext] },
        { id: 2, contexts: first ? [matchingContext] : [] },
      ];
      expect(await validate()).toEqual({ valid: true });
    },
  );

  it.each([
    ['all unavailable', [1, 2]],
    ['first unavailable', [1]],
    ['last unavailable', [2]],
  ] as const)('preserves reads when contexts are %s', async (_scenario, ids) => {
    fixture.artifacts = [
      { id: 1, contexts: [matchingContext] },
      { id: 2, contexts: [] },
    ];
    fixture.failedContextIds = [...ids];
    expect(await validate()).toEqual({ valid: true, reason: 'mlmd-unavailable' });
    expect(JSON.stringify(vi.mocked(console.warn).mock.calls)).not.toContain('sensitive');
    expect(JSON.stringify(vi.mocked(console.warn).mock.calls)).not.toContain(customUri);
  });

  it.each(['missing-bundle', 'lookup-error', 'lookup-http-error'])(
    'preserves reads on %s',
    async (scenario) => {
      fixture.missing = scenario === 'missing-bundle';
      if (scenario === 'lookup-error') {
        vi.mocked(fetch).mockRejectedValue(new Error('sensitive lookup token'));
      } else if (scenario === 'lookup-http-error') {
        vi.mocked(fetch).mockResolvedValue(new Response('sensitive body', { status: 503 }));
      }
      expect(await validate()).toEqual({
        valid: true,
        reason: fixture.missing ? 'protos-unavailable' : 'mlmd-unavailable',
      });
      expect(JSON.stringify(vi.mocked(console.warn).mock.calls)).not.toContain('sensitive');
      expect(JSON.stringify(vi.mocked(console.warn).mock.calls)).not.toContain(customUri);
    },
  );

  it.each([
    ['matching then foreign', [matchingContext], [foreignContext], []],
    ['missing then foreign', [], [foreignContext], []],
    ['unavailable then foreign', [matchingContext], [foreignContext], [1]],
    ['foreign then unavailable', [foreignContext], [matchingContext], [2]],
    ['mixed contexts in one record', [matchingContext, foreignContext], [], []],
  ] as const)(
    'retains a known namespace denial: %s',
    async (_scenario, first, second, failedIds) => {
      fixture.artifacts = [
        { id: 1, contexts: [...first] },
        { id: 2, contexts: [...second] },
      ];
      fixture.failedContextIds = [...failedIds];
      expect(await validate()).toEqual({
        valid: false,
        actualNamespace: 'team-b',
        reason: 'namespace-mismatch',
      });
    },
  );
});

describe('existing no-record namespace-prefix policy', () => {
  beforeEach(() => {
    fixture.artifacts = [];
  });

  it.each([
    ['private-artifacts/team-a/model', true, 'prefix-match'],
    ['private-artifacts/team-b/model', false, 'prefix-namespace-mismatch'],
    ['private%2dartifacts/team-b/model', false, 'prefix-namespace-mismatch'],
    ['custom/model', false, 'prefix-absent'],
    ['private-artifacts/team-a/model/', false, 'key-not-normalized'],
  ] as const)('retains the decision for %s', async (key, valid, reason) => {
    expect(await validate(`s3://bucket/${key}`)).toMatchObject({ valid, reason });
  });

  it('honors the configured namespace prefix', async () => {
    vi.stubEnv('ARTIFACT_NAMESPACE_KEY_PREFIX', 'historical');
    expect(await validate('s3://bucket/historical/team-a/model')).toEqual({
      valid: true,
      reason: 'prefix-match',
    });
    expect(await validate('s3://bucket/private-artifacts/team-a/model')).toEqual({
      valid: false,
      reason: 'prefix-absent',
    });
  });

  it.each(['mlmd-only', 'invalid'])('retains strict denial in %s mode', async (mode) => {
    vi.stubEnv('ARTIFACT_NAMESPACE_OWNERSHIP_MODE', mode);
    expect(await validate('s3://bucket/private-artifacts/team-a/model')).toEqual({
      valid: false,
      reason: 'artifact-not-found',
    });
  });
});

describe('path hardening independent of ownership evidence', () => {
  it.each([
    'private-artifacts/team-a/../team-b/model',
    'private-artifacts/team-a/%2e%2e/team-b/model',
    'private-artifacts/team-a%2f..%2fteam-b/model',
    'private%252dartifacts/team-b/model',
    'custom//model',
    'custom/model//',
    'custom/./model/',
    'custom/%5cmodel',
    'custom/%00model',
    'custom/%ZZmodel',
  ])('rejects unsafe key %s before metadata or storage access', async (key) => {
    expect(await validate(`s3://bucket/${key}`)).toEqual({
      valid: false,
      reason: 'key-not-normalized',
    });
    expect(fetch).not.toHaveBeenCalled();
  });
});
