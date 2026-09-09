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

import { describe, it, expect } from 'vitest';
import { BucketProviders, resolveS3ProviderInfo } from './provider-policy.js';
import { S3ProviderInfo } from '../handlers/artifacts.js';

function hostileQuery(overrides: Partial<S3ProviderInfo['Params']> = {}): S3ProviderInfo {
  return {
    Provider: 's3',
    Params: {
      fromEnv: 'true',
      endpoint: 'attacker.example',
      region: 'custom',
      disableSSL: 'false',
      ...overrides,
    },
  };
}

describe('resolveS3ProviderInfo', () => {
  // Conformance case shared with the Go side's
  // TestS3ProvideSessionInfoOverrideWinsOverQuery: a matching admin Override
  // must win over anything the client's query asked for.
  it('a matching override wins over a hostile query', () => {
    const providers: BucketProviders = {
      s3: {
        default: { endpoint: 's3.company.example', credentials: { fromEnv: true } },
        Overrides: [
          {
            bucketName: 'team-bucket',
            keyPrefix: 'team-a',
            endpoint: 'minio.team-a:9000',
            region: 'us-west-2',
            credentials: { fromEnv: true },
          },
        ],
      },
    };

    const decision = resolveS3ProviderInfo(
      providers,
      's3',
      'team-bucket',
      'team-a/model',
      hostileQuery(),
    );

    expect(decision.allowed).toBe(true);
    expect(decision.effectiveProviderInfo?.Params.endpoint).toBe('minio.team-a:9000');
    expect(decision.effectiveProviderInfo?.Params.endpoint).not.toBe('attacker.example');
  });

  // Conformance case shared with the Go side's
  // TestS3ProvideSessionInfoDefaultWinsOverQueryWithoutMatchingOverride.
  it('the bare Default wins over a hostile query when no override matches', () => {
    const providers: BucketProviders = {
      s3: {
        default: { endpoint: 's3.company.example', credentials: { fromEnv: true } },
      },
    };

    const decision = resolveS3ProviderInfo(
      providers,
      's3',
      'some-other-bucket',
      'path/model',
      hostileQuery(),
    );

    expect(decision.allowed).toBe(true);
    expect(decision.effectiveProviderInfo?.Params.endpoint).toBe('s3.company.example');
  });

  describe('unmanaged provider queries (no admin config at all)', () => {
    it('is allowed by default when unset', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery(),
      );
      expect(decision.allowed).toBe(true);
      expect(decision.effectiveProviderInfo?.Params.endpoint).toBe('attacker.example');
    });

    it('is rejected when explicitly disabled', () => {
      const providers: BucketProviders = { s3: { allowUnmanagedProviderQueries: false } };
      const decision = resolveS3ProviderInfo(
        providers,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery(),
      );
      expect(decision.allowed).toBe(false);
      expect(decision.rejectionReason).toContain('allowUnmanagedProviderQueries');
    });

    it('a request with no client-supplied provider info is unaffected by the gate', () => {
      const providers: BucketProviders = { s3: { allowUnmanagedProviderQueries: false } };
      const decision = resolveS3ProviderInfo(providers, 's3', 'unconfigured-bucket', 'path', null);
      expect(decision.allowed).toBe(true);
      expect(decision.effectiveProviderInfo).toBeNull();
    });
  });

  describe('unmanaged-query guardrails', () => {
    it('allows a public endpoint', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: 's3.us-west-2.amazonaws.com' }),
      );
      expect(decision.allowed).toBe(true);
    });

    it('rejects a loopback IP endpoint', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: '127.0.0.1:9999' }),
      );
      expect(decision.allowed).toBe(false);
      expect(decision.rejectionReason).toContain('loopback/link-local/private');
    });

    it('rejects a link-local IP endpoint (cloud metadata address)', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: '169.254.169.254' }),
      );
      expect(decision.allowed).toBe(false);
      expect(decision.rejectionReason).toContain('loopback/link-local/private');
    });

    it('rejects an RFC1918 private IP endpoint', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: '10.0.0.5:9000' }),
      );
      expect(decision.allowed).toBe(false);
      expect(decision.rejectionReason).toContain('loopback/link-local/private');
    });

    it('rejects disableSSL=true on the unmanaged path', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: 's3.us-west-2.amazonaws.com', disableSSL: 'true' }),
      );
      expect(decision.allowed).toBe(false);
      expect(decision.rejectionReason).toContain('disableSSL');
    });

    it('allows disableSSL=false', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: 's3.us-west-2.amazonaws.com', disableSSL: 'false' }),
      );
      expect(decision.allowed).toBe(true);
    });

    it('allows a hostname endpoint without resolving it via DNS', () => {
      const decision = resolveS3ProviderInfo(
        null,
        's3',
        'unconfigured-bucket',
        'path',
        hostileQuery({ endpoint: 'internal.svc.cluster.local' }),
      );
      expect(decision.allowed).toBe(true);
    });

    it('does not apply guardrails when admin config exists, even if its own endpoint looks unsafe', () => {
      const providers: BucketProviders = {
        s3: {
          default: {
            endpoint: 'seaweedfs.kubeflow:9000',
            credentials: { fromEnv: true },
            disableSSL: true,
          },
        },
      };
      const decision = resolveS3ProviderInfo(providers, 's3', 'team-bucket', 'path', null);
      expect(decision.allowed).toBe(true);
      expect(decision.effectiveProviderInfo?.Params.endpoint).toBe('seaweedfs.kubeflow:9000');
      expect(decision.effectiveProviderInfo?.Params.disableSSL).toBe('true');
    });
  });

  it('minio provider config is resolved independently of s3', () => {
    const providers: BucketProviders = {
      minio: {
        default: { endpoint: 'minio.internal:9000', credentials: { fromEnv: true } },
      },
    };
    const decision = resolveS3ProviderInfo(
      providers,
      'minio',
      'some-bucket',
      'path',
      hostileQuery(),
    );
    expect(decision.allowed).toBe(true);
    expect(decision.effectiveProviderInfo?.Provider).toBe('minio');
    expect(decision.effectiveProviderInfo?.Params.endpoint).toBe('minio.internal:9000');
  });
});
