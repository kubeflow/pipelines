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

import { createRequire } from 'module';

const require = createRequire(import.meta.url);

// Lazy proto load — a missing bundle denies retrieval instead of crashing at import.
let servicePb: any = null;
let storePb: any = null;
let protoLoadAttempted = false;

function loadProtos(): boolean {
  if (servicePb !== null && storePb !== null) {
    return true;
  }
  const candidateBases = [
    '../../src/third_party/mlmd/generated/ml_metadata/proto/',
    '../../../src/third_party/mlmd/generated/ml_metadata/proto/',
  ];
  for (const base of candidateBases) {
    try {
      servicePb = require(`${base}metadata_store_service_pb.js`);
      storePb = require(`${base}metadata_store_pb.js`);
      return true;
    } catch {
      servicePb = null;
      storePb = null;
    }
  }
  if (!protoLoadAttempted) {
    console.warn(
      `[SECURITY] MLMD proto bundle could not be loaded — namespace-ownership ` +
        `validation cannot establish ownership; denying artifact access.`,
    );
    protoLoadAttempted = true;
  }
  return false;
}

const PIPELINE_RUN_CONTEXT_TYPE = 'system.PipelineRun';
const NAMESPACE_PROPERTY_KEY = 'namespace';

const GRPC_WEB_PROTO = 'application/grpc-web+proto';

const DEFAULT_TIMEOUT_MS = (() => {
  const raw = process.env.MLMD_VALIDATION_TIMEOUT_MS;
  const parsed = raw ? parseInt(raw, 10) : NaN;
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 5000;
})();

// When the metadata store has no record of an artifact, the artifact is still owned by
// exactly one namespace, because both the Argo v1 `keyFormat` and the v2
// `defaultPipelineRoot` store every object under a `private-artifacts/<namespace>/`
// key prefix, and the per-namespace object-storage policy isolates each namespace to
// its own prefix. Deriving the owning namespace from that prefix restores retrieval of
// pod logs and other objects that are legitimately not tracked as metadata-store
// artifacts, while still blocking cross-namespace access. Because nothing is ever
// written to the bucket root, an object that carries no such prefix has no derivable
// owning namespace and is denied. The strict upstream behavior that denies every
// artifact absent from the metadata store is preserved under the `mlmd-only` mode.
//
// The mode is normalized to lower case so that an operator-supplied value such as
// `MLMD-ONLY` selects the strict mode instead of silently falling through to the
// prefix-based behavior.
const NAMESPACE_OWNERSHIP_MODE = (
  process.env.ARTIFACT_NAMESPACE_OWNERSHIP_MODE || 'mlmd-then-prefix'
)
  .trim()
  .toLowerCase();

const NAMESPACE_KEY_PREFIX = (process.env.ARTIFACT_NAMESPACE_KEY_PREFIX || 'private-artifacts')
  .trim()
  .replace(/^\/+|\/+$/g, '');

const ARTIFACT_OWNERSHIP_ENFORCEMENT = (process.env.ARTIFACT_OWNERSHIP_ENFORCEMENT || 'enforce')
  .trim()
  .toLowerCase();
if (ARTIFACT_OWNERSHIP_ENFORCEMENT === 'audit') {
  console.warn(
    '[SECURITY] Artifact ownership audit mode permits legacy custom-root reads with matching ' +
      'MLMD evidence. Shared storage credentials can expose other tenants. Verify downstream ' +
      'isolation and migrate to namespace-prefixed roots before 3.0; audit mode is temporary.',
  );
}

export function namespaceFromArtifactUri(
  artifactUri: string,
  keyPrefix: string = NAMESPACE_KEY_PREFIX,
): string | undefined {
  if (!keyPrefix) {
    return undefined;
  }
  const escapedPrefix = keyPrefix.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  // Anchor the prefix to the very first object-key segment, immediately after the
  // "<scheme>://<bucket>/" preamble that `buildArtifactUri` produces. The object key is
  // fully caller-controlled, so a match-anywhere search would let a caller spoof
  // ownership by embedding "<prefix>/<his-namespace>/" deeper in the path; requiring the
  // prefix to be the leading key segment prevents that.
  const match = artifactUri.match(
    new RegExp(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^/]+/${escapedPrefix}/([^/]+)/`),
  );
  return match ? match[1] : undefined;
}

export function encodeGrpcWebRequest(serializedMessage: Uint8Array): Uint8Array {
  const frame = new Uint8Array(5 + serializedMessage.length);
  frame[0] = 0x00; // data frame
  const view = new DataView(frame.buffer);
  view.setUint32(1, serializedMessage.length, false); // big-endian length
  frame.set(serializedMessage, 5);
  return frame;
}

export function decodeGrpcWebResponse(buffer: ArrayBuffer): Uint8Array {
  const view = new DataView(buffer);
  const dataChunks: Uint8Array[] = [];
  let offset = 0;

  while (offset + 5 <= buffer.byteLength) {
    const frameType = view.getUint8(offset);
    const frameLength = view.getUint32(offset + 1, false);

    if (offset + 5 + frameLength > buffer.byteLength) {
      throw new Error(
        `gRPC-web frame at offset ${offset} claims length ${frameLength} ` +
          `but only ${buffer.byteLength - offset - 5} bytes remain`,
      );
    }

    if (frameType === 0x00) {
      dataChunks.push(new Uint8Array(buffer, offset + 5, frameLength));
    } else if (frameType === 0x80) {
      const trailerBytes = new Uint8Array(buffer, offset + 5, frameLength);
      const trailerText = new TextDecoder().decode(trailerBytes);
      const statusMatch = trailerText.match(/grpc-status:\s*(\d+)/);
      const messageMatch = trailerText.match(/grpc-message:\s*([^\r\n]+)/);
      const status = statusMatch ? parseInt(statusMatch[1], 10) : -1;
      if (status !== 0) {
        let message = 'unknown';
        if (messageMatch) {
          const raw = messageMatch[1].trim();
          try {
            message = decodeURIComponent(raw);
          } catch {
            message = raw;
          }
        }
        throw new Error(`gRPC error status ${status}: ${message}`);
      }
    } else {
      throw new Error(`Unexpected gRPC-web frame type: 0x${frameType.toString(16)}`);
    }

    offset += 5 + frameLength;
  }

  if (offset !== buffer.byteLength) {
    throw new Error(
      `gRPC-web response has ${
        buffer.byteLength - offset
      } trailing bytes that don't form a frame header`,
    );
  }

  if (dataChunks.length === 0) {
    throw new Error('gRPC-web response contained no data frame');
  }
  if (dataChunks.length === 1) {
    return dataChunks[0];
  }
  const merged = new Uint8Array(dataChunks.reduce((sum, c) => sum + c.length, 0));
  let pos = 0;
  for (const chunk of dataChunks) {
    merged.set(chunk, pos);
    pos += chunk.length;
  }
  return merged;
}

async function grpcWebCall(
  envoyAddress: string,
  method: string,
  requestBytes: Uint8Array,
  timeoutMs: number = DEFAULT_TIMEOUT_MS,
): Promise<Uint8Array> {
  const base = envoyAddress.replace(/\/+$/, '');
  const url = `${base}/ml_metadata.MetadataStoreService/${method}`;
  const body = encodeGrpcWebRequest(requestBytes);

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': GRPC_WEB_PROTO,
        Accept: GRPC_WEB_PROTO,
        'x-grpc-web': '1',
      },
      body,
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`MLMD gRPC-web call ${method} failed: HTTP ${response.status}`);
    }

    const responseBuffer = await response.arrayBuffer();
    return decodeGrpcWebResponse(responseBuffer);
  } finally {
    clearTimeout(timer);
  }
}

interface ArtifactInfo {
  id: number;
}

export interface ContextNamespace {
  contextType: string;
  namespace: string | undefined;
}

function getArtifactsByUri(envoyAddress: string, uris: string[]): Promise<ArtifactInfo[]> {
  const request = new servicePb.GetArtifactsByURIRequest();
  request.setUrisList(uris);

  return grpcWebCall(envoyAddress, 'GetArtifactsByURI', request.serializeBinary()).then(
    (responseBytes) => {
      const response = servicePb.GetArtifactsByURIResponse.deserializeBinary(responseBytes);
      const artifacts: ArtifactInfo[] = [];
      for (const artifact of response.getArtifactsList()) {
        artifacts.push({ id: artifact.getId() });
      }
      return artifacts;
    },
  );
}

function getContextsByArtifact(
  envoyAddress: string,
  artifactId: number,
): Promise<ContextNamespace[]> {
  const request = new servicePb.GetContextsByArtifactRequest();
  request.setArtifactId(artifactId);

  return grpcWebCall(envoyAddress, 'GetContextsByArtifact', request.serializeBinary()).then(
    (responseBytes) => {
      const response = servicePb.GetContextsByArtifactResponse.deserializeBinary(responseBytes);
      const contexts: ContextNamespace[] = [];
      for (const context of response.getContextsList()) {
        const contextType = context.getType();
        let namespace: string | undefined;
        const customProps = context.getCustomPropertiesMap();
        if (customProps) {
          const nsValue = customProps.get(NAMESPACE_PROPERTY_KEY);
          if (nsValue) {
            const valueCase = nsValue.getValueCase();
            if (valueCase === storePb.Value.ValueCase.STRING_VALUE) {
              namespace = nsValue.getStringValue();
            }
          }
        }
        contexts.push({ contextType, namespace });
      }
      return contexts;
    },
  );
}

export interface ValidationResult {
  valid: boolean;
  actualNamespace?: string;
  reason?: string;
}

// A concrete mismatch on any artifact beats an unavailable context on another.
export function decideFromContexts(
  contextResults: { artifactId: number; contexts: ContextNamespace[] | null }[],
  claimedNamespace: string,
): ValidationResult {
  let hasNamespaceEvidence = false;
  let hadUnavailable = false;
  let hadMissingEvidence = false;
  for (const { contexts } of contextResults) {
    if (contexts === null) {
      hadUnavailable = true;
      continue;
    }
    let artifactHasEvidence = false;
    for (const ctx of contexts) {
      if (ctx.contextType !== PIPELINE_RUN_CONTEXT_TYPE) continue;
      if (!ctx.namespace) continue;
      hasNamespaceEvidence = true;
      artifactHasEvidence = true;
      if (ctx.namespace !== claimedNamespace) {
        return {
          valid: false,
          actualNamespace: ctx.namespace,
          reason: 'namespace-mismatch',
        };
      }
    }
    if (!artifactHasEvidence) hadMissingEvidence = true;
  }
  if (hadUnavailable) {
    return { valid: false, reason: 'mlmd-unavailable' };
  }
  if (!hasNamespaceEvidence || hadMissingEvidence) {
    return { valid: false, reason: 'no-evidence' };
  }
  return { valid: true };
}

// Decides ownership for an artifact that the metadata store does not track. This is the
// security-critical fallback reached when `getArtifactsByUri` returns zero records. It is
// factored out of `validateArtifactNamespace` so that every branch (strict `mlmd-only`
// denial, absent prefix denial, prefix/namespace mismatch denial, and prefix match) is
// unit-testable without a live metadata store. The ownership mode is a parameter, again
// defaulting to the module constant, so tests can exercise the strict mode directly.
export function decideFromPrefixFallback(
  artifactUri: string,
  claimedNamespace: string,
  ownershipMode: string = NAMESPACE_OWNERSHIP_MODE,
): ValidationResult {
  // Only the documented opt-in value enables the prefix fallback; any other value
  // (including typos) fails closed to the strict denial, so a misconfigured mode can
  // never silently weaken the guard.
  if (ownershipMode !== 'mlmd-then-prefix') {
    return { valid: false, reason: 'artifact-not-found' };
  }
  // The object key is caller-controlled and is later passed verbatim to the object
  // store, so a key containing empty or dot segments (for example
  // "private-artifacts/team-a/../victim-ns/obj") would carry the claimant's prefix
  // while addressing another namespace's object on stores that normalize paths.
  // Deny non-normalized keys outright rather than trusting every backend to reject them.
  const objectKey = artifactUri.replace(/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/[^/]+\//, '');
  const hasUnsafeSegment = objectKey.split('/').some((segment) => {
    try {
      const decoded = decodeURIComponent(segment);
      return (
        decoded === '' ||
        decoded === '.' ||
        decoded === '..' ||
        /[\\/?#]/.test(decoded) ||
        /%[0-9a-f]{2}/i.test(decoded) ||
        [...decoded].some(
          (character) => character.charCodeAt(0) < 32 || character.charCodeAt(0) === 127,
        )
      );
    } catch {
      return true;
    }
  });
  if (hasUnsafeSegment) {
    return { valid: false, reason: 'key-not-normalized' };
  }
  const decodedKey = objectKey
    .split('/')
    .map((segment) => decodeURIComponent(segment))
    .join('/');
  const prefixNamespace = namespaceFromArtifactUri(
    artifactUri.slice(0, artifactUri.length - objectKey.length) + decodedKey,
  );
  if (prefixNamespace === undefined) {
    return { valid: false, reason: 'prefix-absent' };
  }
  if (prefixNamespace !== claimedNamespace) {
    return {
      valid: false,
      actualNamespace: prefixNamespace,
      reason: 'prefix-namespace-mismatch',
    };
  }
  return { valid: true, reason: 'prefix-match' };
}

export function decideArtifactOwnership(
  artifactUri: string,
  claimedNamespace: string,
  contextResults: { artifactId: number; contexts: ContextNamespace[] | null }[],
  enforcementMode: string = ARTIFACT_OWNERSHIP_ENFORCEMENT,
): ValidationResult {
  if (enforcementMode !== 'enforce' && enforcementMode !== 'audit') {
    return { valid: false, reason: 'invalid-enforcement-mode' };
  }
  const decision = decideFromContexts(contextResults, claimedNamespace);
  if (!decision.valid) return decision;
  // The prefix is required even when MLMD has a caller-associated record. The audit
  // exception relaxes only an absent prefix, never a mismatch or unsafe path.
  const prefix = decideFromPrefixFallback(artifactUri, claimedNamespace, 'mlmd-then-prefix');
  if (prefix.valid) return prefix;
  if (enforcementMode === 'audit' && prefix.reason === 'prefix-absent') {
    return { valid: true, reason: 'audit-custom-root' };
  }
  return prefix;
}

export async function validateArtifactNamespace(
  envoyAddress: string,
  artifactUri: string,
  claimedNamespace: string,
): Promise<ValidationResult> {
  if (ARTIFACT_OWNERSHIP_ENFORCEMENT !== 'enforce' && ARTIFACT_OWNERSHIP_ENFORCEMENT !== 'audit') {
    return { valid: false, reason: 'invalid-enforcement-mode' };
  }
  if (!loadProtos()) {
    return { valid: false, reason: 'protos-unavailable' };
  }

  let artifacts: ArtifactInfo[];
  try {
    artifacts = await getArtifactsByUri(envoyAddress, [artifactUri]);
  } catch (error) {
    console.warn(
      `[SECURITY] MLMD artifact lookup failed for URI "${artifactUri}", ` + `denying access.`,
    );
    return { valid: false, reason: 'mlmd-unavailable' };
  }

  if (artifacts.length === 0) {
    return decideFromPrefixFallback(artifactUri, claimedNamespace);
  }

  let contextResults: { artifactId: number; contexts: ContextNamespace[] | null }[];
  try {
    contextResults = await Promise.all(
      artifacts.map(async (artifact) => {
        try {
          const contexts = await getContextsByArtifact(envoyAddress, artifact.id);
          return { artifactId: artifact.id, contexts };
        } catch (error) {
          console.warn(
            `[SECURITY] MLMD context lookup failed for artifact ${artifact.id}, ` +
              `marking as unavailable. Error: ${error}`,
          );
          return { artifactId: artifact.id, contexts: null };
        }
      }),
    );
  } catch (error) {
    console.warn(`[SECURITY] MLMD batch context lookup failed, ` + `denying access.`);
    return { valid: false, reason: 'mlmd-unavailable' };
  }

  const decision = decideArtifactOwnership(artifactUri, claimedNamespace, contextResults);
  if (decision.reason === 'mlmd-unavailable') {
    console.warn(
      `[SECURITY] At least one MLMD context lookup was unavailable for URI "${artifactUri}", ` +
        `denying access.`,
    );
  } else if (decision.reason === 'no-evidence') {
    console.warn(
      `[SECURITY] No PipelineRun namespace evidence found in MLMD for URI "${artifactUri}", ` +
        `denying access.`,
    );
  }
  return decision;
}

export function buildArtifactUri(source: string, bucket: string, key: string): string {
  const scheme = source === 'gcs' ? 'gs' : source;
  return `${scheme}://${bucket}/${key}`;
}
