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
import { getConfigMap } from '../k8s-helper.js';
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
  const configMapResult = await getConfigMap(LAUNCHER_CONFIG_MAP, namespace);
  const [configMap] = configMapResult;
  const providersYaml = configMap?.data?.providers;
  if (!providersYaml) {
    return undefined;
  }

  const parsed = load(providersYaml);
  if (!isRecord(parsed)) {
    throw new Error('kfp-launcher providers must be a YAML object');
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

  if (!config.default && !override) {
    return { Provider: provider, Params: { fromEnv: 'true' } };
  }
  if (!config.default?.credentials) {
    throw new Error(`kfp-launcher ${provider} provider is missing default credentials`);
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
      throw new Error(`kfp-launcher ${provider} override is missing credentials`);
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
    throw new Error(`kfp-launcher ${provider} credentials are missing secretRef`);
  }
  params.secretName = secretRef.secretName;
  if (provider === 'gs') {
    if (!secretRef.tokenKey) {
      throw new Error('kfp-launcher gs credentials are missing tokenKey');
    }
    params.tokenKey = secretRef.tokenKey;
    return;
  }
  if (!secretRef.accessKeyKey || !secretRef.secretKeyKey) {
    throw new Error(`kfp-launcher ${provider} credentials are missing access/secret key names`);
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
