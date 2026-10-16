// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

import { createHash } from 'node:crypto';
import type { MinioConfigs } from '../configs.js';
import { getK8sSecret } from '../k8s-helper.js';

export interface ArtifactStorageAuthorization {
  version: 1;
  namespace: string;
  namespaceUID: string;
  state: 'ready' | 'pending' | 'unsupported';
  credentialBinding?: string;
  bucket: 'mlpipeline';
}

/** Bind approval to the actual receiver configuration, never a tenant-supplied annotation. */
export function getArtifactCredentialBinding(
  namespace: string,
  namespaceUID: string,
  minio: MinioConfigs,
): string {
  return createHash('sha256')
    .update(
      [
        namespace,
        namespaceUID,
        `${minio.useSSL ? 'https' : 'http'}://${minio.endPoint}:${minio.port}`,
        minio.accessKey,
        minio.secretKey,
      ].join('\0'),
      'utf8',
    )
    .digest('hex');
}

/** Only the installation's controller may write these records. Never read tenant Secrets. */
export async function getArtifactStorageAuthorization(
  namespace: string,
  installationNamespace: string,
): Promise<ArtifactStorageAuthorization | undefined> {
  let raw: string | undefined;
  try {
    raw = await getK8sSecret(
      `kfp-artifact-storage-${namespace}`,
      'authorization.json',
      installationNamespace,
    );
  } catch (error) {
    const status = error as {
      code?: number;
      statusCode?: number;
      response?: { status?: number };
      body?: { code?: number };
    };
    if (
      status?.code === 404 ||
      status?.statusCode === 404 ||
      status?.response?.status === 404 ||
      status?.body?.code === 404
    )
      return undefined;
    // Kubernetes error bodies can contain Secret data. Deliberately discard them,
    // including the cause, before creating the public error below.
  }
  if (raw === undefined)
    throw new Error('Artifact storage authorization is temporarily unavailable');
  let record: unknown;
  try {
    record = JSON.parse(raw);
  } catch {
    throw new Error('Invalid artifact storage authorization record');
  }
  if (!record || typeof record !== 'object' || Array.isArray(record))
    throw new Error('Invalid artifact storage authorization record');
  const value = record as Record<string, unknown>;
  if (
    value.version !== 1 ||
    value.namespace !== namespace ||
    typeof value.namespaceUID !== 'string' ||
    !/^[a-zA-Z0-9-]{1,128}$/.test(value.namespaceUID) ||
    !['ready', 'pending', 'unsupported'].includes(String(value.state)) ||
    value.bucket !== 'mlpipeline' ||
    (value.state === 'ready' &&
      (typeof value.credentialBinding !== 'string' ||
        !/^[a-f0-9]{64}$/.test(value.credentialBinding)))
  )
    throw new Error('Invalid artifact storage authorization record');
  return value as unknown as ArtifactStorageAuthorization;
}
