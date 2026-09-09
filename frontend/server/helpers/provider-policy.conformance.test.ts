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

// Runs the same test cases the Go side runs in
// provider_policy_conformance_test.go, from a single shared fixture, so the
// Go and TypeScript implementations of the artifact-URI trust decision
// can't silently drift apart. See kubeflow/pipelines#14046.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { BucketProviders, resolveS3ProviderInfo } from './provider-policy.js';
import { S3ProviderInfo } from '../handlers/artifacts.js';

interface ConformanceQuery {
  fromEnv?: string;
  endpoint?: string;
  region?: string;
  disableSSL?: string;
}

interface ConformanceExpect {
  allowed: boolean;
  rejectionContains?: string;
  endpoint?: string;
  disableSSL?: string;
  effectiveIsNull?: boolean;
}

interface ConformanceCase {
  name: string;
  provider: 's3' | 'minio';
  adminConfig: unknown | null;
  bucket: string;
  keyPrefix: string;
  query: ConformanceQuery | null;
  expect: ConformanceExpect;
}

const fixturePath = fileURLToPath(
  new URL('../../../test/conformance/artifact-provider-policy/cases.json', import.meta.url),
);
const cases: ConformanceCase[] = JSON.parse(readFileSync(fixturePath, 'utf-8'));

function toProviderInfo(
  provider: 's3' | 'minio',
  query: ConformanceQuery | null,
): S3ProviderInfo | null {
  if (!query) {
    return null;
  }
  return {
    Provider: provider,
    Params: {
      fromEnv: query.fromEnv ?? 'true',
      endpoint: query.endpoint,
      region: query.region,
      disableSSL: query.disableSSL,
    },
  };
}

function toBucketProviders(
  provider: 's3' | 'minio',
  adminConfig: unknown | null,
): BucketProviders | null {
  if (adminConfig === null) {
    return null;
  }
  return { [provider]: adminConfig } as BucketProviders;
}

describe('resolveS3ProviderInfo conformance (shared with the Go side)', () => {
  expect(cases.length).toBeGreaterThan(0);

  for (const tc of cases) {
    it(tc.name, () => {
      const providers = toBucketProviders(tc.provider, tc.adminConfig);
      const queryProviderInfo = toProviderInfo(tc.provider, tc.query);

      const decision = resolveS3ProviderInfo(
        providers,
        tc.provider,
        tc.bucket,
        tc.keyPrefix,
        queryProviderInfo,
      );

      expect(decision.allowed).toBe(tc.expect.allowed);

      if (!tc.expect.allowed) {
        if (tc.expect.rejectionContains) {
          expect(decision.rejectionReason).toContain(tc.expect.rejectionContains);
        }
        return;
      }

      if (tc.expect.effectiveIsNull) {
        expect(decision.effectiveProviderInfo).toBeNull();
        return;
      }
      if (tc.expect.endpoint) {
        expect(decision.effectiveProviderInfo?.Params.endpoint).toBe(tc.expect.endpoint);
      }
      if (tc.expect.disableSSL) {
        expect(decision.effectiveProviderInfo?.Params.disableSSL).toBe(tc.expect.disableSSL);
      }
    });
  }
});
