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

// Lazy proto load — missing bundle (e.g. dev) fails open instead of crashing at import.
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
  let lastError: unknown = null;
  for (const base of candidateBases) {
    try {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      servicePb = require(`${base}metadata_store_service_pb.js`);
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      storePb = require(`${base}metadata_store_pb.js`);
      return true;
    } catch (error) {
      lastError = error;
      servicePb = null;
      storePb = null;
    }
  }
  if (!protoLoadAttempted) {
    console.warn(
      `[SECURITY] MLMD proto bundle could not be loaded — namespace-ownership ` +
        `validation will be disabled (IDOR check fails open). Error: ${lastError}`,
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
const NAMESPACE_OWNERSHIP_MODE = (
  process.env.ARTIFACT_NAMESPACE_OWNERSHIP_MODE || 'mlmd-then-prefix'
).trim();

const NAMESPACE_KEY_PREFIX = (process.env.ARTIFACT_NAMESPACE_KEY_PREFIX || 'private-artifacts')
  .trim()
  .replace(/^\/+|\/+$/g, '');

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
  for (const { contexts } of contextResults) {
    if (contexts === null) {
      hadUnavailable = true;
      continue;
    }
    for (const ctx of contexts) {
      if (ctx.contextType !== PIPELINE_RUN_CONTEXT_TYPE) continue;
      if (!ctx.namespace) continue;
      hasNamespaceEvidence = true;
      if (ctx.namespace !== claimedNamespace) {
        return {
          valid: false,
          actualNamespace: ctx.namespace,
          reason: 'namespace-mismatch',
        };
      }
    }
  }
  if (hadUnavailable) {
    return { valid: true, reason: 'mlmd-unavailable' };
  }
  if (!hasNamespaceEvidence) {
    return { valid: true, reason: 'no-evidence' };
  }
  return { valid: true };
}

export async function validateArtifactNamespace(
  envoyAddress: string,
  artifactUri: string,
  claimedNamespace: string,
): Promise<ValidationResult> {
  if (!loadProtos()) {
    return { valid: true, reason: 'protos-unavailable' };
  }

  let artifacts: ArtifactInfo[];
  try {
    artifacts = await getArtifactsByUri(envoyAddress, [artifactUri]);
  } catch (error) {
    console.warn(
      `[SECURITY] MLMD artifact lookup failed for URI "${artifactUri}", ` +
        `allowing access (fail-open). Error: ${error}`,
    );
    return { valid: true, reason: 'mlmd-unavailable' };
  }

  if (artifacts.length === 0) {
    if (NAMESPACE_OWNERSHIP_MODE === 'mlmd-only') {
      console.warn(
        `[SECURITY] Artifact not found in MLMD for URI "${artifactUri}", ` +
          `denying access (no namespace ownership evidence).`,
      );
      return { valid: false, reason: 'artifact-not-found' };
    }
    const prefixNamespace = namespaceFromArtifactUri(artifactUri);
    if (prefixNamespace === undefined) {
      console.warn(
        `[SECURITY] Insecure direct object reference blocked: artifact "${artifactUri}" is ` +
          `absent from the metadata store and carries no "${NAMESPACE_KEY_PREFIX}/<namespace>/" ` +
          `object-key prefix, so its owning namespace cannot be derived; denying access.`,
      );
      return { valid: false, reason: 'prefix-absent' };
    }
    if (prefixNamespace !== claimedNamespace) {
      console.warn(
        `[SECURITY] Insecure direct object reference blocked: object-key prefix namespace ` +
          `"${prefixNamespace}" does not match the requested namespace "${claimedNamespace}" ` +
          `for URI "${artifactUri}".`,
      );
      return {
        valid: false,
        actualNamespace: prefixNamespace,
        reason: 'prefix-namespace-mismatch',
      };
    }
    return { valid: true, reason: 'prefix-match' };
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
    console.warn(
      `[SECURITY] MLMD batch context lookup failed, ` +
        `allowing access (fail-open). Error: ${error}`,
    );
    return { valid: true, reason: 'mlmd-unavailable' };
  }

  const decision = decideFromContexts(contextResults, claimedNamespace);
  if (decision.reason === 'mlmd-unavailable') {
    console.warn(
      `[SECURITY] At least one MLMD context lookup was unavailable for URI "${artifactUri}", ` +
        `allowing access (fail-open after scanning available contexts).`,
    );
  } else if (decision.reason === 'no-evidence') {
    console.warn(
      `[SECURITY] No PipelineRun namespace evidence found in MLMD for URI "${artifactUri}", ` +
        `allowing access (no ownership data to validate against).`,
    );
  }
  return decision;
}

export function buildArtifactUri(source: string, bucket: string, key: string): string {
  const scheme = source === 'gcs' ? 'gs' : source;
  return `${scheme}://${bucket}/${key}`;
}
