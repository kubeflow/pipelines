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
  ArtifactCoordinates,
  normalizeArtifactStorageCoordinates,
} from './artifact-coordinates.js';
import {
  ArtifactProvider,
  artifactProviderForSource,
  buildArtifactUri,
  LauncherArtifactSource,
} from './artifact-sources.js';

const LAUNCHER_CONFIG_MAP = 'kfp-launcher';
const DEFAULT_PIPELINE_ROOT = 'minio://mlpipeline/v2/artifacts';
const LAUNCHER_CONFIG_CACHE_TTL_MS = 30_000;
const LAUNCHER_CONFIG_CACHE_MAX_ENTRIES = 1_000;

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

interface LauncherConfiguration {
  configMapPresent: boolean;
  defaultPipelineRoot: string;
  providers: LauncherProviders;
}

interface LauncherConfigurationCacheEntry {
  expiresAt: number;
  value: Promise<LauncherConfiguration>;
}

const launcherConfigurationCache = new Map<string, LauncherConfigurationCacheEntry>();

interface StoreSessionInfo {
  Provider: ArtifactProvider;
  Params: Record<string, string>;
}

export class LauncherConfigError extends Error {}

export class LauncherConfigParseError extends LauncherConfigError {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigParseError';
  }
}

export class LauncherConfigReadError extends LauncherConfigError {
  constructor(message: string) {
    super(message);
    this.name = 'LauncherConfigReadError';
  }
}

export class LauncherConfigValidationError extends LauncherConfigError {
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
  coordinates: ArtifactCoordinates<LauncherArtifactSource>,
  namespace: string,
): Promise<string | undefined> {
  const storageCoordinates = normalizeArtifactStorageCoordinates(coordinates);
  const { configMapPresent, defaultPipelineRoot, providers } =
    await getLauncherConfiguration(namespace);

  const provider = artifactProviderForSource(storageCoordinates.source);
  const config = providers[provider];
  const override = findOverride(config, storageCoordinates.bucket, storageCoordinates.key);
  const artifactUri = buildArtifactUri(
    storageCoordinates.source,
    storageCoordinates.bucket,
    storageCoordinates.key,
  );
  const underPipelineRoot = isWithinPipelineRoot(storageCoordinates, defaultPipelineRoot);
  if (!underPipelineRoot && !storageCoordinates.artifactUriQuery && !override) {
    // A ConfigMap with only defaultPipelineRoot contributes no provider credentials for legacy or
    // custom-root artifacts. Preserve the environment-credential path unless this scheme actually
    // has provider policy that would otherwise be bypassed.
    if (!configMapPresent || !config) {
      return undefined;
    }
    throw new LauncherConfigValidationError(
      `Artifact URI ${artifactUri} is outside defaultPipelineRoot and has no explicit provider ` +
        'query or matching override. Move the artifact under the configured pipeline root, add ' +
        'an override, or add explicit provider query parameters and retry.',
    );
  }

  // Launcher replaces an under-root artifact's query with defaultPipelineRoot's query. An
  // artifact URI's own query is used only outside that root.
  const effectiveQuery = underPipelineRoot
    ? getUriQuery(defaultPipelineRoot)
    : storageCoordinates.artifactUriQuery;
  // GCS provider credential policy is authoritative whenever it exists. Unlike the S3 runtime,
  // the GCS runtime does not let an artifact URI query replace configured namespace credentials.
  const gcsProviderIsAuthoritative =
    provider === 'gs' &&
    !!config &&
    (config.default != null || config.Overrides != null || config.overrides != null);
  if (effectiveQuery && !gcsProviderIsAuthoritative) {
    return JSON.stringify(buildQuerySessionInfo(provider, effectiveQuery, !underPipelineRoot));
  }
  if (!config) {
    return undefined;
  }

  return JSON.stringify(
    buildSessionInfo(provider, storageCoordinates.bucket, storageCoordinates.key, config),
  );
}

async function getLauncherConfiguration(namespace: string): Promise<LauncherConfiguration> {
  const now = Date.now();
  pruneExpiredLauncherConfigurationCacheEntries(now);
  const cached = launcherConfigurationCache.get(namespace);
  if (cached) {
    // Refresh insertion order so the size cap evicts the least recently used namespace.
    launcherConfigurationCache.delete(namespace);
    launcherConfigurationCache.set(namespace, cached);
    return cached.value;
  }

  const value = loadLauncherConfiguration(namespace);
  launcherConfigurationCache.set(namespace, {
    expiresAt: now + LAUNCHER_CONFIG_CACHE_TTL_MS,
    value,
  });
  evictLauncherConfigurationCacheOverflow();
  void value.catch(() => {
    if (launcherConfigurationCache.get(namespace)?.value === value) {
      launcherConfigurationCache.delete(namespace);
    }
  });
  return value;
}

function pruneExpiredLauncherConfigurationCacheEntries(now: number): void {
  launcherConfigurationCache.forEach((entry, namespace) => {
    if (entry.expiresAt <= now) {
      launcherConfigurationCache.delete(namespace);
    }
  });
}

function evictLauncherConfigurationCacheOverflow(): void {
  while (launcherConfigurationCache.size > LAUNCHER_CONFIG_CACHE_MAX_ENTRIES) {
    const leastRecentlyUsedNamespace = launcherConfigurationCache.keys().next().value;
    if (leastRecentlyUsedNamespace === undefined) {
      return;
    }
    launcherConfigurationCache.delete(leastRecentlyUsedNamespace);
  }
}

async function loadLauncherConfiguration(namespace: string): Promise<LauncherConfiguration> {
  const [configMap, configMapError] = await getConfigMap(LAUNCHER_CONFIG_MAP, namespace);
  if (configMapError && !isNotFoundError(configMapError)) {
    throw new LauncherConfigReadError(
      `${configMapError.message}. Verify that the UI service account can read the ` +
        `${LAUNCHER_CONFIG_MAP} ConfigMap and retry the artifact request.`,
    );
  }
  const defaultPipelineRoot = configMap?.data?.defaultPipelineRoot || DEFAULT_PIPELINE_ROOT;

  let providers: LauncherProviders = {};
  const providersYaml = configMap?.data?.providers;
  if (providersYaml) {
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
    providers = parsed as LauncherProviders;
  }
  return { configMapPresent: !!configMap, defaultPipelineRoot, providers };
}

function findOverride(
  config: ProviderConfig | undefined,
  bucket: string,
  key: string,
): ProviderEntry | undefined {
  const configuredOverrides = config?.Overrides ?? config?.overrides;
  const overrides = Array.isArray(configuredOverrides) ? configuredOverrides : [];
  // The launcher opens the artifact basename inside a bucket session selected from its parent URI.
  // Match overrides against that same parent prefix rather than the full object key.
  const normalizedKey = key.replace(/\/+$/, '');
  const separatorIndex = normalizedKey.lastIndexOf('/');
  const parentPrefix = separatorIndex === -1 ? '' : normalizedKey.slice(0, separatorIndex);
  return overrides.find(
    (entry) => entry.bucketName === bucket && prefixMatches(parentPrefix, entry.keyPrefix || ''),
  );
}

function buildSessionInfo(
  provider: ArtifactProvider,
  bucket: string,
  key: string,
  config: ProviderConfig,
): StoreSessionInfo {
  const configuredOverrides = config.Overrides ?? config.overrides;
  const override = findOverride(config, bucket, key);

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

function buildQuerySessionInfo(
  provider: ArtifactProvider,
  query: string,
  enforceNativeS3Query: boolean,
): StoreSessionInfo {
  const params: Record<string, string> = {};
  new URLSearchParams(query).forEach((value, key) => {
    if (provider === 's3' && enforceNativeS3Query && !NATIVE_S3_QUERY_OPTIONS.has(key)) {
      throw new LauncherConfigValidationError(
        `S3 artifact URI query option "${key}" is not supported by Go Cloud. ` +
          'Use a native S3 query option and retry.',
      );
    }
    // Go's url.Values.Get reads the first duplicate. Preserve that behavior so credential and
    // endpoint selection cannot diverge between the launcher and the artifact reader.
    if (!(key in params)) {
      params[key] = value;
    }
  });
  // URI queries configure the provider but never authorize namespace Secret reads.
  params.fromEnv = 'true';
  return { Provider: provider, Params: params };
}

const NATIVE_S3_QUERY_OPTIONS = new Set([
  'accelerate',
  'anonymous',
  'awssdk',
  'disable_https',
  'dualstack',
  'endpoint',
  'fips',
  'hostname_immutable',
  'kmskeyid',
  'profile',
  'rate_limiter_capacity',
  'region',
  'request_checksum_calculation',
  'response_checksum_validation',
  'role',
  's3ForcePathStyle',
  'ssetype',
  'use_path_style',
]);

function getUriQuery(uri: string): string {
  try {
    return new URL(uri).search.slice(1);
  } catch (error) {
    throw new LauncherConfigValidationError(
      `kfp-launcher defaultPipelineRoot is invalid. Correct it and retry: ${error}`,
    );
  }
}

function isWithinPipelineRoot(
  coordinates: ArtifactCoordinates<LauncherArtifactSource>,
  pipelineRoot: string,
): boolean {
  let root: URL;
  try {
    root = new URL(pipelineRoot);
  } catch (error) {
    throw new LauncherConfigValidationError(
      `Unable to compare the artifact URI with defaultPipelineRoot. Correct the launcher ` +
        `configuration and retry: ${error}`,
    );
  }
  const artifactScheme = coordinates.source === 'gcs' ? 'gs' : coordinates.source;
  const artifactPath = `/${coordinates.key}`.replace(/\/+$/, '');
  let rootPath: string;
  try {
    rootPath = decodeURIComponent(root.pathname).replace(/\/+$/, '');
  } catch (error) {
    throw new LauncherConfigValidationError(
      `kfp-launcher defaultPipelineRoot contains invalid path encoding. Correct it and retry: ${error}`,
    );
  }
  return (
    `${artifactScheme}:` === root.protocol &&
    coordinates.bucket === root.host &&
    (artifactPath === rootPath || artifactPath.startsWith(`${rootPath}/`))
  );
}

function applyS3Settings(params: Record<string, string>, entry: ProviderEntry): void {
  // Launcher treats empty string overrides as inheritance from the selected default entry.
  if (entry.endpoint) params.endpoint = entry.endpoint;
  if (entry.region) params.region = entry.region;
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

export const TEST_ONLY = {
  clearLauncherConfigurationCache: () => launcherConfigurationCache.clear(),
  getLauncherConfigurationCacheKeys: () => [...launcherConfigurationCache.keys()],
  launcherConfigurationCacheMaxEntries: LAUNCHER_CONFIG_CACHE_MAX_ENTRIES,
};
