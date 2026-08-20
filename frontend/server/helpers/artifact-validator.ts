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
import { stripArtifactUriQuery } from './artifact-coordinates.js';
import { applyArtifactPathPolicy, ARTIFACT_PATH_POLICIES } from './artifact-path.js';
export { buildArtifactUri, requiresArtifactOwnershipValidation } from './artifact-sources.js';

const NAMESPACE_KEY_PREFIX = (process.env.ARTIFACT_NAMESPACE_KEY_PREFIX || 'private-artifacts')
  .trim()
  .replace(/^\/+|\/+$/g, '');
export type ArtifactOwnershipMode = 'artifact-then-prefix' | 'artifact-only' | 'invalid';

export function normalizeArtifactOwnershipMode(value?: string): ArtifactOwnershipMode {
  switch ((value || 'artifact-then-prefix').trim().toLowerCase()) {
    case 'mlmd-then-prefix':
    case 'artifact-then-prefix':
      return 'artifact-then-prefix';
    case 'mlmd-only':
    case 'artifact-only':
      return 'artifact-only';
    default:
      return 'invalid';
  }
}

const NAMESPACE_OWNERSHIP_MODE = normalizeArtifactOwnershipMode(
  process.env.ARTIFACT_NAMESPACE_OWNERSHIP_MODE,
);
export function resolveArtifactValidationTimeoutMs(
  environment: NodeJS.ProcessEnv = process.env,
): number {
  const configured = Number(
    environment.ARTIFACT_VALIDATION_TIMEOUT_MS || environment.MLMD_VALIDATION_TIMEOUT_MS,
  );
  return Number.isFinite(configured) && configured > 0 ? configured : 5000;
}
const VALIDATION_TIMEOUT_MS = resolveArtifactValidationTimeoutMs();

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
  // A launcher provider query is part of artifact identity, not the object key. Endpoint values
  // can contain '/' and must not be interpreted as object-key segments by this prefix check.
  const artifactUriWithoutQuery = stripArtifactUriQuery(artifactUri);
  const objectKey = artifactUriWithoutQuery.replace(/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\/[^/]+\//, '');
  const scheme = artifactUriWithoutQuery.match(/^([a-zA-Z][a-zA-Z0-9+.-]*):\/\//)?.[1];
  const isLauncherArtifact = scheme === 's3' || scheme === 'minio' || scheme === 'gs';
  // Go SplitObjectURI deliberately trims one trailing slash before resolving the object key.
  // Apply the same compatibility normalization without accepting internal or doubled separators.
  const policyKey =
    isLauncherArtifact && objectKey.endsWith('/') ? objectKey.slice(0, -1) : objectKey;
  const pathPolicy =
    scheme === 'http' || scheme === 'https'
      ? ARTIFACT_PATH_POLICIES.http
      : ARTIFACT_PATH_POLICIES.ownership;
  if (applyArtifactPathPolicy(policyKey, pathPolicy) === undefined) {
    return { valid: false, reason: 'key-not-normalized' };
  }

  const actualNamespace = namespaceFromArtifactUri(artifactUriWithoutQuery);
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
  ownershipMode: ArtifactOwnershipMode = NAMESPACE_OWNERSHIP_MODE,
): ValidationResult {
  if (ownershipMode !== 'artifact-then-prefix') {
    console.warn(
      `[SECURITY] Artifact ownership lookup found no record for URI "${artifactUri}"; ` +
        (ownershipMode === 'artifact-only'
          ? 'denying access because ARTIFACT_NAMESPACE_OWNERSHIP_MODE uses strict validation.'
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
  allowNamespaceIsolatedCustomRoots = false,
): Promise<ValidationResult> {
  // A database row in the caller's namespace must not override ownership encoded by the
  // standard multi-user object key. Otherwise a tenant could import another namespace's
  // URI into its own run and make the namespace-scoped Artifact API lookup succeed.
  const keyPrefixValidation = validateArtifactKeyPrefix(artifactUri, claimedNamespace);
  if (!keyPrefixValidation.valid && keyPrefixValidation.reason !== 'artifact-not-found') {
    return keyPrefixValidation;
  }
  if (NAMESPACE_OWNERSHIP_MODE === 'artifact-then-prefix' && keyPrefixValidation.valid) {
    return keyPrefixValidation;
  }

  const artifactService = new ArtifactServiceApi(new Configuration({ basePath: apiServerAddress }));
  // The generated client encodes query parameters once, and the API server deliberately
  // QueryUnescapes filters after the gateway has decoded the request. Pre-encode the JSON so URI
  // characters such as `%` and `+` survive both decoding stages unchanged.
  const filter = encodeURIComponent(
    JSON.stringify({
      predicates: [{ key: 'uri', operation: 'EQUALS', stringValue: artifactUri }],
    }),
  );
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
      // A namespaced Artifact API row alone is not proof that a custom-root URI belongs to the
      // namespace: a tenant can import an arbitrary URI into its own run. Custom roots are safe
      // only when the actual read is delegated to the namespace-isolated artifact proxy.
      return keyPrefixValidation.valid || allowNamespaceIsolatedCustomRoots
        ? { valid: true, reason: 'artifact-api-match' }
        : { valid: false, reason: 'custom-root-requires-namespace-isolation' };
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
