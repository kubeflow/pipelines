// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import type { Request } from 'express';
import { PassThrough } from 'node:stream';
import type { ArgoConfigs, MinioConfigs, AWSConfigs } from '../configs.js';
import { getAuthorizedPodLogs, getPodForLogAuthorization } from '../k8s-helper.js';
import {
  createS3ArtifactStore,
  getObjectStream,
  getArtifactStoreOrigin,
  parseArtifactStoreEndpoint,
} from '../artifact-store.js';
import {
  DEFAULT_LOG_ARCHIVE_KEY_FORMAT,
  LogArchiveContext,
  validateLogArchiveKey,
} from './log-archive-policy.js';

interface ArchiveContext extends LogArchiveContext {
  archiveKey?: string;
  archiveBucket?: string;
  archiveEndpoint?: string;
  archiveInsecure?: boolean;
  workflowCreationTimestamp?: string;
}

export class PodLogAccessError extends Error {
  constructor(
    public readonly statusCode: number,
    message: string,
  ) {
    super(message);
  }
}

function errorStatus(error: unknown): number | undefined {
  const value = error as {
    code?: number;
    statusCode?: number;
    response?: { statusCode?: number };
    body?: { code?: number };
  };
  return value?.statusCode ?? value?.response?.statusCode ?? value?.body?.code ?? value?.code;
}

/** Authorize through the API before any Kubernetes or object-store request. */
export function createAuthorizedPodLogsReader(
  apiServerAddress: string,
  apiPrefix: string,
  argo: ArgoConfigs,
  artifacts: { minio: MinioConfigs; aws: AWSConfigs },
  containerName = 'main',
) {
  return async (req: Request) => {
    const runId = req.query.runid;
    const podName = req.query.podname;
    if (
      typeof runId !== 'string' ||
      !runId ||
      runId === '.' ||
      runId === '..' ||
      typeof podName !== 'string' ||
      !podName ||
      podName === '.' ||
      podName === '..'
    ) {
      throw new PodLogAccessError(400, 'runid and podname must be non-empty path segments');
    }
    const headers = new Headers();
    for (const [name, value] of Object.entries(req.headers)) {
      if (
        value !== undefined &&
        !['host', 'connection', 'content-length', 'transfer-encoding'].includes(name.toLowerCase())
      ) {
        headers.set(name, Array.isArray(value) ? value.join(', ') : value);
      }
    }
    const response = await fetch(
      `${apiServerAddress.replace(/\/$/, '')}/${apiPrefix}/runs/${encodeURIComponent(runId)}/nodes/${encodeURIComponent(podName)}/log?archive_context=true`,
      { headers, redirect: 'manual', signal: AbortSignal.timeout(30000) },
    );
    if (!response.ok) {
      await response.body?.cancel();
      throw new PodLogAccessError(
        response.status >= 400 ? response.status : 502,
        'Run log authorization failed',
      );
    }
    let body: unknown;
    try {
      body = await response.json();
    } catch {
      throw new PodLogAccessError(502, 'Invalid run log context response');
    }
    if (!body || typeof body !== 'object' || Array.isArray(body)) {
      throw new PodLogAccessError(502, 'Invalid run log context response');
    }
    const context = body as ArchiveContext;
    if (
      typeof context.runId !== 'string' ||
      typeof context.namespace !== 'string' ||
      typeof context.workflowName !== 'string'
    ) {
      throw new PodLogAccessError(502, 'Invalid run log context response');
    }
    const identityProbe = `private-artifacts/${context.namespace}/${context.workflowName}/2000/01/01/${podName}/main.log`;
    if (
      context.runId !== runId ||
      !validateLogArchiveKey(identityProbe, podName, context, DEFAULT_LOG_ARCHIVE_KEY_FORMAT)
    ) {
      throw new PodLogAccessError(403, 'Run log context is missing a valid native identity');
    }
    try {
      const pod = await getPodForLogAuthorization(podName, context.namespace);
      if (
        pod.metadata?.namespace !== context.namespace ||
        pod.metadata?.labels?.['pipeline/runid'] !== runId
      ) {
        throw new PodLogAccessError(403, 'Pod does not belong to the authorized run');
      }
      const stream = new PassThrough();
      stream.end(await getAuthorizedPodLogs(podName, context.namespace, containerName));
      return stream;
    } catch (error) {
      if (errorStatus(error) !== 404) throw error;
    }

    if (containerName !== 'main') {
      throw new PodLogAccessError(404, 'Only main container logs have a supported archive layout');
    }
    if (!argo.archiveLogs || !argo.archiveBucketName) {
      throw new PodLogAccessError(404, 'Pod is gone and log archives are not configured');
    }
    const options = argo.archiveArtifactory === 'minio' ? artifacts.minio : artifacts.aws;
    const origins = [
      getArtifactStoreOrigin(options),
      ...(argo.archiveArtifactory === 'minio' ? argo.artifactRepositoryEndpoints || [] : []),
    ];
    if (context.archiveBucket !== undefined && context.archiveBucket !== argo.archiveBucketName) {
      throw new PodLogAccessError(403, 'Archived log bucket is not the configured log store');
    }
    if (context.archiveEndpoint !== undefined) {
      if (
        typeof context.archiveEndpoint !== 'string' ||
        (context.archiveInsecure !== undefined && typeof context.archiveInsecure !== 'boolean')
      ) {
        throw new PodLogAccessError(403, 'Invalid archived log endpoint');
      }
      const endpoint = parseArtifactStoreEndpoint(
        context.archiveEndpoint,
        context.archiveInsecure === true,
      );
      if (!endpoint || !origins.includes(endpoint.origin)) {
        throw new PodLogAccessError(403, 'Archived log endpoint is not the configured log store');
      }
    }
    const format = argo.logArchiveKeyFormat ?? DEFAULT_LOG_ARCHIVE_KEY_FORMAT;
    let key = context.archiveKey;
    if (key === undefined) {
      const dateHint = context.workflowCreationTimestamp ?? req.query.createdat;
      const date =
        typeof dateHint === 'string' ? /^(\d{4})-(\d{2})-(\d{2})(?:T|$)/.exec(dateHint) : null;
      const values: Record<string, string> = {
        '{{workflow.namespace}}': context.namespace,
        '{{workflow.name}}': context.workflowName,
        '{{pod.name}}': podName,
      };
      if (date) {
        values['{{workflow.creationTimestamp.Y}}'] = date[1];
        values['{{workflow.creationTimestamp.m}}'] = date[2];
        values['{{workflow.creationTimestamp.d}}'] = date[3];
      }
      key =
        format
          .split('/')
          .map((segment) => values[segment] ?? segment)
          .join('/') + '/main.log';
    }
    if (typeof key !== 'string' || !validateLogArchiveKey(key, podName, context, format)) {
      throw new PodLogAccessError(403, 'Archived log key is outside the authorized run layout');
    }
    const store = await createS3ArtifactStore(options, 's3', argo.archiveBucketName);
    return getObjectStream({ store, key });
  };
}
