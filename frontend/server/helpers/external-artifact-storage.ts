// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// https://www.apache.org/licenses/LICENSE-2.0

import { createHash } from 'node:crypto';
import { constants } from 'node:fs';
import { open } from 'node:fs/promises';
import { posix } from 'node:path';
import { Client as MinioClient } from 'minio';

export interface ExternalArtifactStorageConfig {
  namespaceUID: string;
  configurationRevision: string;
  storage: { source: 's3'; bucket: string; endpoint: string; region: string };
  identity: {
    kind: 'shared-credentials-file';
    principal: string;
    credentialFile: string;
    profile: string;
  };
}

const identifier = /^[a-zA-Z0-9][a-zA-Z0-9_.:@/+=,-]{0,511}$/;
const namespaceUID = /^[a-zA-Z0-9-]{1,128}$/;
const invalidConfiguration = () =>
  new Error('Invalid approved external artifact storage configuration');

/** Keep serving validation aligned with the approved object-store bucket schema. */
export function isAllowedExternalArtifactBucketName(value: unknown): value is string {
  return typeof value === 'string' && /^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$/.test(value);
}

/** Parse only administrator-supplied configuration, never browser providerInfo. */
export function parseExternalArtifactStorageConfig(value: unknown): ExternalArtifactStorageConfig {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw invalidConfiguration();
  const config = value as Partial<ExternalArtifactStorageConfig>;
  const { storage, identity } = config;
  if (
    typeof config.namespaceUID !== 'string' ||
    !namespaceUID.test(config.namespaceUID) ||
    typeof config.configurationRevision !== 'string' ||
    !/^[a-f0-9]{64}$/.test(config.configurationRevision) ||
    !storage ||
    storage.source !== 's3' ||
    !isAllowedExternalArtifactBucketName(storage.bucket) ||
    typeof storage.region !== 'string' ||
    !/^[a-zA-Z0-9-]{1,64}$/.test(storage.region) ||
    !identity ||
    identity.kind !== 'shared-credentials-file' ||
    typeof identity.principal !== 'string' ||
    !identifier.test(identity.principal) ||
    typeof identity.profile !== 'string' ||
    !/^[a-zA-Z0-9_+=,.@-]{1,128}$/.test(identity.profile) ||
    typeof identity.credentialFile !== 'string' ||
    !/^\/[a-zA-Z0-9_./-]{1,1023}$/.test(identity.credentialFile) ||
    posix.normalize(identity.credentialFile) !== identity.credentialFile ||
    identity.credentialFile.endsWith('/') ||
    typeof storage.endpoint !== 'string' ||
    storage.endpoint.length > 2048
  )
    throw invalidConfiguration();
  try {
    const endpoint = new URL(storage.endpoint);
    if (
      !['http:', 'https:'].includes(endpoint.protocol) ||
      endpoint.origin !== storage.endpoint ||
      endpoint.username ||
      endpoint.password ||
      endpoint.pathname !== '/' ||
      endpoint.search ||
      endpoint.hash
    )
      throw invalidConfiguration();
  } catch {
    throw invalidConfiguration();
  }
  return {
    namespaceUID: config.namespaceUID,
    configurationRevision: config.configurationRevision,
    storage: {
      source: 's3',
      bucket: storage.bucket,
      endpoint: storage.endpoint,
      region: storage.region,
    },
    identity: {
      kind: 'shared-credentials-file',
      principal: identity.principal,
      credentialFile: identity.credentialFile,
      profile: identity.profile,
    },
  };
}

/** Bind approved identity wiring and destination; credential renewal leaves this unchanged. */
export function getExternalArtifactStorageBinding(
  namespace: string,
  config: ExternalArtifactStorageConfig,
): string {
  return createHash('sha256')
    .update(
      [
        'external-file-v1',
        namespace,
        config.namespaceUID,
        config.storage.source,
        config.storage.bucket,
        config.storage.endpoint,
        config.storage.region,
        config.identity.principal,
        config.identity.credentialFile,
        config.identity.profile,
        config.configurationRevision,
      ].join('\0'),
      'utf8',
    )
    .digest('hex');
}

/** Read one bounded broker-written profile without role/process/default-chain fallbacks. */
export async function readExternalArtifactCredentials(config: ExternalArtifactStorageConfig) {
  const unavailable = () =>
    new Error(
      'Approved artifact credentials are unavailable; check the namespace credential broker',
    );
  let contents: string;
  try {
    const file = await open(
      config.identity.credentialFile,
      constants.O_RDONLY | constants.O_NONBLOCK,
    );
    try {
      const stat = await file.stat();
      if (!stat.isFile() || stat.size > 65536) throw unavailable();
      const bytes = Buffer.alloc(65537);
      const { bytesRead } = await file.read(bytes, 0, bytes.length, 0);
      if (bytesRead > 65536) throw unavailable();
      contents = bytes.subarray(0, bytesRead).toString('utf8');
    } finally {
      await file.close();
    }
  } catch {
    throw unavailable();
  }
  const values = new Map<string, string>();
  let selected = false;
  let found = false;
  for (const raw of contents.split(/\r?\n/)) {
    const line = raw.trim();
    if (!line || line.startsWith('#') || line.startsWith(';')) continue;
    const section = /^\[([^\]]+)\]$/.exec(line);
    if (section) {
      selected = section[1] === config.identity.profile;
      if (selected && found) throw unavailable();
      found ||= selected;
      continue;
    }
    if (!selected) continue;
    const entry = /^([a-z_]+)\s*=\s*(\S+)\s*$/.exec(line);
    if (!entry) throw unavailable();
    const key = entry[1] === 'aws_security_token' ? 'aws_session_token' : entry[1];
    if (
      ![
        'aws_access_key_id',
        'aws_secret_access_key',
        'aws_session_token',
        'expiration',
        'region',
      ].includes(key) ||
      values.has(key)
    )
      throw unavailable();
    // A profile's region is metadata only; the approved destination supplies it.
    values.set(key, entry[2]);
  }
  const accessKey = values.get('aws_access_key_id');
  const secretKey = values.get('aws_secret_access_key');
  const sessionToken = values.get('aws_session_token');
  const expiration = values.get('expiration');
  if (
    !accessKey ||
    !secretKey ||
    !sessionToken ||
    (expiration &&
      (!Number.isFinite(Date.parse(expiration)) || Date.parse(expiration) <= Date.now()))
  )
    throw unavailable();
  return { accessKey, secretKey, sessionToken };
}

/** Credentials and endpoint come solely from the approved namespace configuration. */
export async function createExternalArtifactClient(
  config: ExternalArtifactStorageConfig,
): Promise<MinioClient> {
  const credentials = await readExternalArtifactCredentials(config);
  const endpoint = new URL(config.storage.endpoint);
  return new MinioClient({
    // URL keeps IPv6 brackets; Minio expects the bare address as its host.
    endPoint: endpoint.hostname.replace(/^\[(.*)\]$/, '$1'),
    port: endpoint.port ? Number(endpoint.port) : endpoint.protocol === 'https:' ? 443 : 80,
    useSSL: endpoint.protocol === 'https:',
    region: config.storage.region,
    ...credentials,
  });
}
