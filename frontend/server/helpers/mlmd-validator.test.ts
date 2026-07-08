// Copyright 2025 The Kubeflow Authors
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
import {
  decideFromContexts,
  decideFromPrefixFallback,
  decodeGrpcWebResponse,
  encodeGrpcWebRequest,
  namespaceFromArtifactUri,
} from './mlmd-validator.js';

const PIPELINE_RUN = 'system.PipelineRun';

function buildFrame(type: number, payload: Uint8Array): Uint8Array {
  const frame = new Uint8Array(5 + payload.length);
  frame[0] = type;
  new DataView(frame.buffer).setUint32(1, payload.length, false);
  frame.set(payload, 5);
  return frame;
}

function concat(...arrs: Uint8Array[]): ArrayBuffer {
  const len = arrs.reduce((s, a) => s + a.length, 0);
  const out = new Uint8Array(len);
  let pos = 0;
  for (const a of arrs) {
    out.set(a, pos);
    pos += a.length;
  }
  return out.buffer;
}

describe('encodeGrpcWebRequest', () => {
  it('prefixes payload with a 5-byte data-frame header (big-endian length)', () => {
    const payload = new Uint8Array([0x01, 0x02, 0x03, 0x04]);
    const frame = encodeGrpcWebRequest(payload);

    expect(frame.length).toBe(9);
    expect(frame[0]).toBe(0x00);
    expect(new DataView(frame.buffer).getUint32(1, false)).toBe(4);
    expect(Array.from(frame.slice(5))).toEqual(Array.from(payload));
  });

  it('handles empty payload', () => {
    const frame = encodeGrpcWebRequest(new Uint8Array(0));
    expect(frame.length).toBe(5);
    expect(frame[0]).toBe(0x00);
    expect(new DataView(frame.buffer).getUint32(1, false)).toBe(0);
  });
});

describe('decodeGrpcWebResponse', () => {
  const okTrailer = new TextEncoder().encode('grpc-status:0\r\n');
  const errTrailer = new TextEncoder().encode('grpc-status:13\r\ngrpc-message:internal\r\n');

  it('returns the data-frame body when followed by a status-0 trailer', () => {
    const data = new Uint8Array([0xde, 0xad, 0xbe, 0xef]);
    const buf = concat(buildFrame(0x00, data), buildFrame(0x80, okTrailer));
    expect(Array.from(decodeGrpcWebResponse(buf))).toEqual(Array.from(data));
  });

  it('returns the data-frame body when no trailer is present', () => {
    const data = new Uint8Array([0x11, 0x22]);
    const buf = concat(buildFrame(0x00, data));
    expect(Array.from(decodeGrpcWebResponse(buf))).toEqual(Array.from(data));
  });

  it('throws on a non-zero grpc-status trailer', () => {
    const data = new Uint8Array([0x01]);
    const buf = concat(buildFrame(0x00, data), buildFrame(0x80, errTrailer));
    expect(() => decodeGrpcWebResponse(buf)).toThrow(/gRPC error status 13/);
  });

  it('throws when a frame claims more bytes than the buffer holds (truncation)', () => {
    const buf = new Uint8Array(9);
    buf[0] = 0x00;
    new DataView(buf.buffer).setUint32(1, 100, false);
    expect(() => decodeGrpcWebResponse(buf.buffer)).toThrow(/claims length 100/);
  });

  it('throws on trailing bytes that do not form a complete frame header', () => {
    const data = new Uint8Array([0xaa]);
    const validFrame = buildFrame(0x00, data);
    const out = new Uint8Array(validFrame.length + 3);
    out.set(validFrame, 0);
    out[validFrame.length] = 0xff;
    out[validFrame.length + 1] = 0xff;
    out[validFrame.length + 2] = 0xff;
    expect(() => decodeGrpcWebResponse(out.buffer)).toThrow(/trailing bytes/);
  });

  it('accumulates multiple data frames into a single payload', () => {
    const chunk1 = new Uint8Array([0x01, 0x02]);
    const chunk2 = new Uint8Array([0x03, 0x04, 0x05]);
    const buf = concat(buildFrame(0x00, chunk1), buildFrame(0x00, chunk2));
    expect(Array.from(decodeGrpcWebResponse(buf))).toEqual([0x01, 0x02, 0x03, 0x04, 0x05]);
  });

  it('throws on an empty response (no data frame)', () => {
    expect(() => decodeGrpcWebResponse(new ArrayBuffer(0))).toThrow(/no data frame/);
  });

  it('throws on an unknown frame type', () => {
    const buf = new Uint8Array(5);
    buf[0] = 0x42;
    expect(() => decodeGrpcWebResponse(buf.buffer)).toThrow(/Unexpected gRPC-web frame type/);
  });
});

describe('decideFromContexts', () => {
  it('passes when the only PipelineRun context namespace matches the claim', () => {
    expect(
      decideFromContexts(
        [{ artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] }],
        'ns-a',
      ),
    ).toEqual({ valid: true });
  });

  it('rejects when a PipelineRun context namespace differs from the claim', () => {
    expect(
      decideFromContexts(
        [{ artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] }],
        'ns-b',
      ),
    ).toEqual({
      valid: false,
      actualNamespace: 'ns-a',
      reason: 'namespace-mismatch',
    });
  });

  it('a mismatch on a later artifact wins over an unavailable lookup on an earlier one', () => {
    expect(
      decideFromContexts(
        [
          { artifactId: 1, contexts: null },
          { artifactId: 2, contexts: [{ contextType: PIPELINE_RUN, namespace: 'victim-ns' }] },
        ],
        'attacker-ns',
      ),
    ).toEqual({
      valid: false,
      actualNamespace: 'victim-ns',
      reason: 'namespace-mismatch',
    });
  });

  it('returns mlmd-unavailable when all results are unavailable', () => {
    expect(
      decideFromContexts(
        [
          { artifactId: 1, contexts: null },
          { artifactId: 2, contexts: null },
        ],
        'ns-a',
      ),
    ).toEqual({ valid: true, reason: 'mlmd-unavailable' });
  });

  it('returns mlmd-unavailable when some lookups failed but others have no mismatch', () => {
    expect(
      decideFromContexts(
        [
          { artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
          { artifactId: 2, contexts: null },
        ],
        'ns-a',
      ),
    ).toEqual({ valid: true, reason: 'mlmd-unavailable' });
  });

  it('returns no-evidence when contexts contain no PipelineRun entry at all', () => {
    expect(
      decideFromContexts(
        [
          {
            artifactId: 1,
            contexts: [{ contextType: 'system.Pipeline', namespace: 'ns-a' }],
          },
        ],
        'ns-b',
      ),
    ).toEqual({ valid: true, reason: 'no-evidence' });
  });

  it('returns no-evidence when PipelineRun context has no namespace property', () => {
    expect(
      decideFromContexts(
        [{ artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: undefined }] }],
        'ns-a',
      ),
    ).toEqual({ valid: true, reason: 'no-evidence' });
  });

  it('rejects when one artifact has multiple PipelineRun contexts and any mismatches', () => {
    expect(
      decideFromContexts(
        [
          {
            artifactId: 1,
            contexts: [
              { contextType: PIPELINE_RUN, namespace: 'ns-a' },
              { contextType: PIPELINE_RUN, namespace: 'ns-c' },
            ],
          },
        ],
        'ns-a',
      ),
    ).toEqual({
      valid: false,
      actualNamespace: 'ns-c',
      reason: 'namespace-mismatch',
    });
  });

  it('passes cleanly when many artifacts all match the claim', () => {
    expect(
      decideFromContexts(
        [
          { artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
          { artifactId: 2, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
          { artifactId: 3, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
        ],
        'ns-a',
      ),
    ).toEqual({ valid: true });
  });

  it('rejects when the only mismatching context is on the last artifact in the list', () => {
    expect(
      decideFromContexts(
        [
          { artifactId: 1, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
          { artifactId: 2, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-a' }] },
          { artifactId: 3, contexts: [{ contextType: PIPELINE_RUN, namespace: 'ns-b' }] },
        ],
        'ns-a',
      ),
    ).toEqual({
      valid: false,
      actualNamespace: 'ns-b',
      reason: 'namespace-mismatch',
    });
  });
});

describe('namespaceFromArtifactUri', () => {
  it('extracts the namespace from a v1 Argo archived-log key', () => {
    expect(
      namespaceFromArtifactUri(
        'minio://mlpipeline/private-artifacts/team-a/flip-coin-abc/2026/07/08/flip-coin-abc-1/main.log',
      ),
    ).toBe('team-a');
  });

  it('extracts the namespace from a v2 pipeline-root key', () => {
    expect(
      namespaceFromArtifactUri(
        'minio://mlpipeline/private-artifacts/team-b/v2/artifacts/run/op/out',
      ),
    ).toBe('team-b');
  });

  it('returns undefined when the key prefix is absent', () => {
    expect(namespaceFromArtifactUri('minio://mlpipeline/public/some/object')).toBeUndefined();
  });

  it('honors a custom key prefix', () => {
    expect(
      namespaceFromArtifactUri('s3://bucket/scoped-artifacts/team-c/object', 'scoped-artifacts'),
    ).toBe('team-c');
  });

  it('does not match a partial prefix segment', () => {
    expect(
      namespaceFromArtifactUri('minio://mlpipeline/not-private-artifacts/team-d/object'),
    ).toBeUndefined();
  });

  it('ignores a prefix that is not the leading object-key segment', () => {
    expect(
      namespaceFromArtifactUri('minio://mlpipeline/team-b/private-artifacts/team-a/object'),
    ).toBeUndefined();
  });
});

describe('decideFromPrefixFallback', () => {
  const uri = 'minio://mlpipeline/private-artifacts/team-a/flip-coin-abc/1/main.log';

  it('denies under mlmd-only mode without consulting the object-key prefix', () => {
    expect(decideFromPrefixFallback(uri, 'team-a', 'mlmd-only')).toEqual({
      valid: false,
      reason: 'artifact-not-found',
    });
  });

  it('denies when the object key carries no owning-namespace prefix', () => {
    expect(
      decideFromPrefixFallback('minio://mlpipeline/public/object', 'team-a', 'mlmd-then-prefix'),
    ).toEqual({ valid: false, reason: 'prefix-absent' });
  });

  it('denies and reports the actual namespace when the prefix namespace mismatches', () => {
    expect(decideFromPrefixFallback(uri, 'attacker-ns', 'mlmd-then-prefix')).toEqual({
      valid: false,
      actualNamespace: 'team-a',
      reason: 'prefix-namespace-mismatch',
    });
  });

  it('allows when the prefix namespace matches the claim', () => {
    expect(decideFromPrefixFallback(uri, 'team-a', 'mlmd-then-prefix')).toEqual({
      valid: true,
      reason: 'prefix-match',
    });
  });

  it('denies a key that hides a dot-segment traversal behind a matching prefix', () => {
    expect(
      decideFromPrefixFallback(
        'minio://mlpipeline/private-artifacts/team-a/../victim-ns/obj',
        'team-a',
        'mlmd-then-prefix',
      ),
    ).toEqual({ valid: false, reason: 'key-not-normalized' });
  });

  it('denies a key containing a single-dot or empty path segment', () => {
    expect(
      decideFromPrefixFallback(
        'minio://mlpipeline/private-artifacts/team-a/./obj',
        'team-a',
        'mlmd-then-prefix',
      ),
    ).toEqual({ valid: false, reason: 'key-not-normalized' });
    expect(
      decideFromPrefixFallback(
        'minio://mlpipeline/private-artifacts/team-a//obj',
        'team-a',
        'mlmd-then-prefix',
      ),
    ).toEqual({ valid: false, reason: 'key-not-normalized' });
  });

  it('fails closed to the strict denial for an unrecognized ownership mode', () => {
    expect(decideFromPrefixFallback(uri, 'team-a', 'mlmd_then_prefix')).toEqual({
      valid: false,
      reason: 'artifact-not-found',
    });
  });
});
