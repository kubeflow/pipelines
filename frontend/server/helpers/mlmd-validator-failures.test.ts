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

import { afterEach, describe, expect, it, vi } from 'vitest';

const fixture = vi.hoisted(() => ({ missing: false, noArtifacts: false, noContexts: false }));
vi.mock('module', () => ({
  createRequire: () => () => {
    if (fixture.missing) throw new Error('missing bundle');
    class Request {
      setUrisList() {}
      setArtifactId() {}
      serializeBinary() {
        return new Uint8Array();
      }
    }
    return {
      GetArtifactsByURIRequest: Request,
      GetContextsByArtifactRequest: Request,
      GetArtifactsByURIResponse: {
        deserializeBinary: () => ({
          getArtifactsList: () => (fixture.noArtifacts ? [] : [{ getId: () => 1 }]),
        }),
      },
      GetContextsByArtifactResponse: {
        deserializeBinary: () => ({
          getContextsList: () =>
            fixture.noContexts
              ? []
              : [
                  {
                    getType: () => 'system.PipelineRun',
                    getCustomPropertiesMap: () =>
                      new Map([
                        ['namespace', { getValueCase: () => 1, getStringValue: () => 'team-a' }],
                      ]),
                  },
                ],
        }),
      },
      Value: { ValueCase: { STRING_VALUE: 1 } },
    };
  },
}));

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});

describe.each(['enforce', 'audit'])('MLMD request failures in %s mode', (mode) => {
  it.each([
    'missing-bundle',
    'lookup-error',
    'context-error',
    'no-contexts',
    'custom-match',
    'empty-custom',
    'empty-prefix',
    'strict-empty-prefix',
  ])('%s', async (scenario) => {
    vi.resetModules();
    vi.stubEnv('ARTIFACT_OWNERSHIP_ENFORCEMENT', mode);
    vi.stubEnv(
      'ARTIFACT_NAMESPACE_OWNERSHIP_MODE',
      scenario === 'strict-empty-prefix' ? 'mlmd-only' : 'mlmd-then-prefix',
    );
    fixture.missing = scenario === 'missing-bundle';
    fixture.noArtifacts = scenario.includes('empty');
    fixture.noContexts = scenario === 'no-contexts';
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        if (
          scenario === 'lookup-error' ||
          (scenario === 'context-error' && url.endsWith('/GetContextsByArtifact'))
        ) {
          throw new Error('unavailable');
        }
        return new Response(new Uint8Array([0, 0, 0, 0, 0]));
      }),
    );
    const { validateArtifactNamespace } = await import('./mlmd-validator.js');
    const key = scenario.includes('prefix') ? 'private-artifacts/team-a/model' : 'custom/model';
    const result = await validateArtifactNamespace('http://mlmd', `s3://bucket/${key}`, 'team-a');
    expect(result.valid).toBe(
      scenario === 'empty-prefix' || (mode === 'audit' && scenario === 'custom-match'),
    );
    if (mode === 'audit' && scenario === 'custom-match')
      expect(result.reason).toBe('audit-custom-root');
  });
});
