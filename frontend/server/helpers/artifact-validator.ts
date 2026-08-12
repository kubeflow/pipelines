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

import { ArtifactServiceApi, Configuration } from '../src/generated/apisv2beta1/artifact/index.js';

const NAMESPACE_KEY_PREFIX = (process.env.ARTIFACT_NAMESPACE_KEY_PREFIX || 'private-artifacts')
  .trim()
  .replace(/^\/+|\/+$/g, '');
const NAMESPACE_OWNERSHIP_MODE = (
  process.env.ARTIFACT_NAMESPACE_OWNERSHIP_MODE || 'mlmd-then-prefix'
)
  .trim()
  .toLowerCase();
const PREFIX_FALLBACK_OWNERSHIP_MODES: ReadonlySet<string> = new Set([
  'mlmd-then-prefix',
  'artifact-then-prefix',
]);
const VALIDATION_TIMEOUT_MS = (() => {
  const configured = Number(process.env.ARTIFACT_VALIDATION_TIMEOUT_MS);
  return Number.isFinite(configured) && configured > 0 ? configured : 5000;
})();

// Volume artifacts use local filesystem paths rather than object-store bucket/key URIs,
// so their ownership cannot be established through the artifact URI lookup below.
export const OWNERSHIP_VALIDATED_ARTIFACT_SOURCES: ReadonlySet<string> = new Set([
  'minio',
  's3',
  'gcs',
  'http',
  'https',
]);

export function requiresArtifactOwnershipValidation(source: string): boolean {
  return OWNERSHIP_VALIDATED_ARTIFACT_SOURCES.has(source);
}

export interface ValidationResult {
  valid: boolean;
  actualNamespace?: string;
  reason?: string;
}

export function namespaceFromArtifactUri(
  artifactUri: string,
  keyPrefix: string = NAMESPACE_KEY_PREFIX,
): string | undefined {
  if (!keyPrefix) {
    return undefined;
  }
  const escapedPrefix = keyPrefix.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = artifactUri.match(
    new RegExp(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^/]+/${escapedPrefix}/([^/]+)/`),
  );
  return match?.[1];
}

export function validateArtifactKeyPrefix(
  artifactUri: string,
  claimedNamespace: string,
): ValidationResult {
  const objectKey = artifactUri.replace(/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/[^/]+\//, '');
  const hasUnsafeSegment = objectKey
    .split('/')
    .some((segment) => segment === '' || segment === '.' || segment === '..');
  if (hasUnsafeSegment) {
    return { valid: false, reason: 'key-not-normalized' };
  }

  const actualNamespace = namespaceFromArtifactUri(artifactUri);
  if (actualNamespace === undefined) {
    return { valid: false, reason: 'artifact-not-found' };
  }
  if (actualNamespace !== claimedNamespace) {
    return { valid: false, actualNamespace, reason: 'prefix-namespace-mismatch' };
  }
  return { valid: true, reason: 'prefix-match' };
}

export function validateArtifactNotFound(
  artifactUri: string,
  claimedNamespace: string,
  ownershipMode: string = NAMESPACE_OWNERSHIP_MODE,
): ValidationResult {
  const normalizedMode = ownershipMode.trim().toLowerCase();
  if (!PREFIX_FALLBACK_OWNERSHIP_MODES.has(normalizedMode)) {
    const strictMode = normalizedMode === 'mlmd-only' || normalizedMode === 'artifact-only';
    console.warn(
      `[SECURITY] Artifact ownership lookup found no record for URI "${artifactUri}"; ` +
        (strictMode
          ? `denying access because ARTIFACT_NAMESPACE_OWNERSHIP_MODE is "${ownershipMode}".`
          : `denying access because ARTIFACT_NAMESPACE_OWNERSHIP_MODE "${ownershipMode}" is ` +
            `not recognized. Use "artifact-only" for strict validation or ` +
            `"artifact-then-prefix" to enable namespace-prefix fallback.`),
    );
    return { valid: false, reason: 'artifact-not-found' };
  }
  return validateArtifactKeyPrefix(artifactUri, claimedNamespace);
}

export async function validateArtifactNamespace(
  apiServerAddress: string,
  artifactUri: string,
  claimedNamespace: string,
  authenticationHeaders?: Record<string, string>,
): Promise<ValidationResult> {
  const artifactService = new ArtifactServiceApi(new Configuration({ basePath: apiServerAddress }));
  const filter = JSON.stringify({
    predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: artifactUri }],
  });
  const abortController = new AbortController();
  const timeout = setTimeout(() => abortController.abort(), VALIDATION_TIMEOUT_MS);

  try {
    const response = await artifactService.artifacts(
      claimedNamespace,
      undefined,
      1,
      undefined,
      filter,
      { headers: authenticationHeaders, signal: abortController.signal },
    );
    if (response.artifacts?.length) {
      return { valid: true, reason: 'artifact-api-match' };
    }
    return validateArtifactNotFound(artifactUri, claimedNamespace);
  } catch (error) {
    console.error(
      `[SECURITY] Artifact ownership lookup failed for URI "${artifactUri}"; denying access.`,
      error,
    );
    return { valid: false, reason: 'artifact-api-unavailable' };
  } finally {
    clearTimeout(timeout);
  }
}

export function buildArtifactUri(source: string, bucket: string, key: string): string {
  const scheme = source === 'gcs' ? 'gs' : source;
  return `${scheme}://${bucket}/${key}`;
}
