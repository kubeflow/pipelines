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

/**
 * Independently applies the same admin S3/MinIO provider policy the Go
 * launcher applies in backend/src/v2/config/s3.go, so a bug in one
 * implementation cannot bypass the other. See kubeflow/pipelines#14046.
 *
 * The rule, mirrored exactly from the Go side:
 *   1. If the launcher's `kfp-launcher` ConfigMap has ANY `default`/
 *      `Overrides` configured for this provider, that admin configuration is
 *      always authoritative -- a client-supplied providerInfo query can never
 *      override it, regardless of what it asks for.
 *   2. Only when no admin configuration exists for this provider at all does
 *      a client-supplied providerInfo get honored, gated by
 *      `allowUnmanagedProviderQueries` (default true), and subject to the
 *      SSRF/TLS-downgrade guardrails below.
 */

import { S3ProviderInfo } from '../handlers/artifacts.js';

export interface S3Credentials {
  fromEnv?: boolean;
  secretRef?: {
    secretName: string;
    accessKeyKey: string;
    secretKeyKey: string;
  };
}

export interface S3ProviderDefault {
  endpoint: string;
  credentials: S3Credentials;
  region?: string;
  disableSSL?: boolean;
  forcePathStyle?: boolean;
  maxRetries?: number;
}

export interface S3Override {
  bucketName: string;
  keyPrefix: string;
  credentials: S3Credentials;
  endpoint?: string;
  region?: string;
  disableSSL?: boolean;
  forcePathStyle?: boolean;
  maxRetries?: number;
}

export interface S3ProviderConfig {
  default?: S3ProviderDefault;
  // Capitalized to match the Go struct's `json:"Overrides"` tag exactly --
  // the same kfp-launcher ConfigMap YAML is parsed by both sides.
  Overrides?: S3Override[];
  allowUnmanagedProviderQueries?: boolean;
}

export interface BucketProviders {
  minio?: S3ProviderConfig;
  s3?: S3ProviderConfig;
}

export interface S3ProviderDecision {
  /** Whether this request may proceed at all. */
  allowed: boolean;
  /** Set when allowed is false: why the request was rejected. */
  rejectionReason?: string;
  /**
   * The provider info to actually use. Null means "no admin config and no
   * (or an ignored) client query -- fall back to environment credentials",
   * matching the Go SessionInfo{Provider, Params: {fromEnv: "true"}} case.
   */
  effectiveProviderInfo: S3ProviderInfo | null;
}

function getOverrideByPrefix(
  overrides: S3Override[] | undefined,
  bucketName: string,
  keyPrefix: string,
): S3Override | undefined {
  return overrides?.find((o) => o.bucketName === bucketName && keyPrefix.startsWith(o.keyPrefix));
}

function providerInfoFromAdminConfig(
  provider: 's3' | 'minio',
  config: S3ProviderConfig,
  bucketName: string,
  keyPrefix: string,
): S3ProviderInfo {
  const base = config.default!;
  const override = getOverrideByPrefix(config.Overrides, bucketName, keyPrefix);

  const endpoint = override?.endpoint || base.endpoint;
  const region = override?.region ?? base.region;
  const disableSSL = override?.disableSSL ?? base.disableSSL ?? false;
  const credentials = override?.credentials ?? base.credentials;

  const params: S3ProviderInfo['Params'] = {
    fromEnv: String(credentials.fromEnv ?? false),
    endpoint,
    region,
    disableSSL: String(disableSSL),
  };
  if (!credentials.fromEnv && credentials.secretRef) {
    params.secretName = credentials.secretRef.secretName;
    params.accessKeyKey = credentials.secretRef.accessKeyKey;
    params.secretKeyKey = credentials.secretRef.secretKeyKey;
  }
  return { Provider: provider, Params: params };
}

/**
 * Resolves the S3/MinIO provider decision for one artifact request. See the
 * module-level rule summary above; this is the frontend's independent
 * enforcement of the same rule the Go launcher applies.
 */
export function resolveS3ProviderInfo(
  providers: BucketProviders | null | undefined,
  provider: 's3' | 'minio',
  bucketName: string,
  keyPrefix: string,
  queryProviderInfo: S3ProviderInfo | null,
): S3ProviderDecision {
  const config = provider === 's3' ? providers?.s3 : providers?.minio;

  if (config && (config.default || config.Overrides)) {
    return {
      allowed: true,
      effectiveProviderInfo: providerInfoFromAdminConfig(provider, config, bucketName, keyPrefix),
    };
  }

  // No admin configuration exists for this provider at all: the
  // client-supplied query is all there is to go on.
  if (!queryProviderInfo) {
    return { allowed: true, effectiveProviderInfo: null };
  }

  const allowUnmanaged = config?.allowUnmanagedProviderQueries ?? true;
  if (!allowUnmanaged) {
    return {
      allowed: false,
      rejectionReason: `artifact provider query for bucket "${bucketName}" rejected: no ${provider} provider configuration exists and allowUnmanagedProviderQueries is disabled`,
      effectiveProviderInfo: null,
    };
  }

  const guardrailError = validateUnmanagedProviderQuery(queryProviderInfo);
  if (guardrailError) {
    return {
      allowed: false,
      rejectionReason: `artifact provider query for bucket "${bucketName}" rejected: ${guardrailError}`,
      effectiveProviderInfo: null,
    };
  }

  return { allowed: true, effectiveProviderInfo: queryProviderInfo };
}

/**
 * SSRF/TLS-downgrade guardrails for the unmanaged-query path only -- there is
 * no admin override here to grant permission for disableSSL, and no
 * allowlist entry to trust an endpoint against. Mirrors
 * validateUnmanagedProviderQuery in backend/src/v2/config/s3.go, including
 * its scope: only a literal loopback/link-local/private IP address is
 * rejected. A hostname is deliberately not resolved via DNS -- see the Go
 * side's comment for why (no defense against DNS rebinding, adds a real
 * network call, false confidence).
 */
function validateUnmanagedProviderQuery(providerInfo: S3ProviderInfo): string | null {
  if (providerInfo.Params.disableSSL?.toLowerCase() === 'true') {
    return 'disableSSL is only permitted via an admin-configured provider override';
  }

  const endpoint = providerInfo.Params.endpoint;
  if (!endpoint) {
    return null;
  }

  let host: string;
  try {
    host = new URL(endpoint.match(/^https?:\/\//) ? endpoint : `http://${endpoint}`).hostname;
  } catch {
    // Unparseable endpoint: let the actual connection attempt fail on its
    // own rather than blocking here.
    return null;
  }

  if (isUnsafeHost(host)) {
    return `endpoint "${host}" resolves to a loopback/link-local/private address, which is not permitted`;
  }
  return null;
}

function isUnsafeHost(host: string): boolean {
  const bare = host.replace(/^\[|\]$/g, '').toLowerCase();

  if (isIPv4(bare)) {
    const octets = bare.split('.').map(Number);
    const [a, b] = octets;
    return (
      a === 127 || // loopback
      a === 0 || // unspecified/"this network"
      (a === 169 && b === 254) || // link-local
      a === 10 || // RFC1918
      (a === 172 && b >= 16 && b <= 31) || // RFC1918
      (a === 192 && b === 168) // RFC1918
    );
  }

  if (isIPv6(bare)) {
    return (
      bare === '::' ||
      bare === '::1' ||
      bare.startsWith('fe80:') || // link-local
      bare.startsWith('fc') || // unique local fc00::/7
      bare.startsWith('fd') // unique local fc00::/7
    );
  }

  // Not a literal IP: see the caveat above on why hostnames aren't resolved.
  return false;
}

function isIPv4(host: string): boolean {
  const parts = host.split('.');
  return parts.length === 4 && parts.every((p) => /^\d{1,3}$/.test(p) && Number(p) <= 255);
}

function isIPv6(host: string): boolean {
  return host.includes(':');
}
