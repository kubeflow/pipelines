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

import { load } from 'js-yaml';
import { getConfigMap, K8sError } from '../k8s-helper.js';
import {
  ArtifactProvider,
  artifactProviderForSource,
  LauncherArtifactSource,
} from './artifact-sources.js';

const LAUNCHER_CONFIG_MAP = 'kfp-launcher';

interface SecretRef {
  secretName?: string;
  accessKeyKey?: string;
  secretKeyKey?: string;
  tokenKey?: string;
}

interface Credentials {
  fromEnv?: boolean;
  secretRef?: SecretRef;
}

interface ProviderEntry {
  endpoint?: string;
  region?: string;
  disableSSL?: boolean;
  forcePathStyle?: boolean;
  maxRetries?: number;
  bucketName?: string;
  keyPrefix?: string;
  credentials?: Credentials;
}

interface ProviderConfig {
  default?: ProviderEntry;
  overrides?: ProviderEntry[];
  Overrides?: ProviderEntry[];
}

interface LauncherProviders {
  minio?: ProviderConfig;
  s3?: ProviderConfig;
  gs?: ProviderConfig;
}

interface StoreSessionInfo {
  Provider: ArtifactProvider;
  Params: Record<string, string>;
}

export interface ArtifactCoordinates {
  source: LauncherArtifactSource;
  bucket: string;
  key: string;
}

export class LauncherConfigParseError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigParseError';
  }
}

export class LauncherConfigReadError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigReadError';
  }
}

export class LauncherConfigValidationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigValidationError';
  }
}

/**
 * Reconstructs the launcher store session information for an artifact URI.
 *
 * Launcher intentionally no longer persists store_session_info on artifacts.
 * The UI server therefore resolves the same default/override provider entry
 * from the namespace's kfp-launcher ConfigMap before downloading an artifact.
 */
export async function getLauncherProviderInfo(
  coordinates: ArtifactCoordinates,
  namespace: string,
): Promise<string | undefined> {
  const [configMap, configMapError] = await getConfigMap(LAUNCHER_CONFIG_MAP, namespace);
  if (configMapError) {
    if (isNotFoundError(configMapError)) {
      return undefined;
    }
    throw new LauncherConfigReadError(
      `${configMapError.message}. Verify that the UI service account can read the ` +
        `${LAUNCHER_CONFIG_MAP} ConfigMap and retry the artifact request.`,
    );
  }
  const providersYaml = configMap?.data?.providers;
  if (!providersYaml) {
    return undefined;
  }

  let parsed: unknown;
  try {
    parsed = load(providersYaml);
  } catch (error) {
    throw new LauncherConfigParseError(
      `kfp-launcher providers contains invalid YAML. Correct the providers entry and retry: ${error}`,
    );
  }
  if (!isRecord(parsed)) {
    throw new LauncherConfigParseError(
      'kfp-launcher providers must be a YAML object. Correct the providers entry and retry.',
    );
  }
  const providers = parsed as LauncherProviders;
  const provider = artifactProviderForSource(coordinates.source);
  const config = providers[provider];
  if (!config) {
    return undefined;
  }

  return JSON.stringify(buildSessionInfo(provider, coordinates.bucket, coordinates.key, config));
}

function buildSessionInfo(
  provider: ArtifactProvider,
  bucket: string,
  key: string,
  config: ProviderConfig,
): StoreSessionInfo {
  const configuredOverrides = config.Overrides ?? config.overrides;
  const overrides = Array.isArray(configuredOverrides) ? configuredOverrides : [];
  const override = overrides.find(
    (entry) => entry.bucketName === bucket && prefixMatches(key, entry.keyPrefix || ''),
  );

  if (!config.default && configuredOverrides === undefined) {
    return { Provider: provider, Params: { fromEnv: 'true' } };
  }
  if (!config.default?.credentials) {
    throw new LauncherConfigValidationError(
      `kfp-launcher ${provider} provider is missing default credentials. ` +
        'Add default credentials or remove the provider configuration and retry.',
    );
  }

  const params: Record<string, string> = {};
  if (provider !== 'gs') {
    params.endpoint = config.default.endpoint || '';
    params.region = config.default.region || '';
    params.disableSSL = String(config.default.disableSSL ?? false);
    params.forcePathStyle = String(config.default.forcePathStyle ?? true);
    params.maxRetries = String(config.default.maxRetries ?? 5);
  }
  applyS3Settings(params, config.default);
  applyCredentials(params, config.default.credentials, provider);

  if (override) {
    if (!override.credentials) {
      throw new LauncherConfigValidationError(
        `kfp-launcher ${provider} override is missing credentials. ` +
          'Add override credentials and retry.',
      );
    }
    applyS3Settings(params, override);
    applyCredentials(params, override.credentials, provider);
  }

  return { Provider: provider, Params: params };
}

function applyS3Settings(params: Record<string, string>, entry: ProviderEntry): void {
  if (entry.endpoint !== undefined) params.endpoint = entry.endpoint;
  if (entry.region !== undefined) params.region = entry.region;
  if (entry.disableSSL !== undefined) params.disableSSL = String(entry.disableSSL);
  if (entry.forcePathStyle !== undefined) params.forcePathStyle = String(entry.forcePathStyle);
  if (entry.maxRetries !== undefined) params.maxRetries = String(entry.maxRetries);
}

function applyCredentials(
  params: Record<string, string>,
  credentials: Credentials,
  provider: ArtifactProvider,
): void {
  const fromEnv = credentials.fromEnv === true;
  params.fromEnv = String(fromEnv);
  delete params.secretName;
  delete params.accessKeyKey;
  delete params.secretKeyKey;
  delete params.tokenKey;

  if (fromEnv) {
    return;
  }
  const secretRef = credentials.secretRef;
  if (!secretRef?.secretName) {
    throw new LauncherConfigValidationError(
      `kfp-launcher ${provider} credentials are missing secretRef. ` +
        'Add a secretRef or set fromEnv to true and retry.',
    );
  }
  params.secretName = secretRef.secretName;
  if (provider === 'gs') {
    if (!secretRef.tokenKey) {
      throw new LauncherConfigValidationError(
        'kfp-launcher gs credentials are missing tokenKey. Add tokenKey and retry.',
      );
    }
    params.tokenKey = secretRef.tokenKey;
    return;
  }
  if (!secretRef.accessKeyKey || !secretRef.secretKeyKey) {
    throw new LauncherConfigValidationError(
      `kfp-launcher ${provider} credentials are missing access/secret key names. ` +
        'Add accessKeyKey and secretKeyKey and retry.',
    );
  }
  params.accessKeyKey = secretRef.accessKeyKey;
  params.secretKeyKey = secretRef.secretKeyKey;
}

function prefixMatches(key: string, overridePrefix: string): boolean {
  const normalizedKey = key.replace(/^\/+|\/+$/g, '');
  const normalizedPrefix = overridePrefix.replace(/^\/+|\/+$/g, '');
  return (
    normalizedPrefix === '' ||
    normalizedKey === normalizedPrefix ||
    normalizedKey.startsWith(`${normalizedPrefix}/`)
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isNotFoundError(error: K8sError): boolean {
  const details = error.additionalInfo;
  const statusCode =
    details?.statusCode ?? details?.code ?? details?.response?.statusCode ?? details?.body?.code;
  return (
    Number(statusCode) === 404 ||
    details?.reason === 'NotFound' ||
    details?.body?.reason === 'NotFound' ||
    error.message.trim().toLowerCase() === 'not found'
  );
}
