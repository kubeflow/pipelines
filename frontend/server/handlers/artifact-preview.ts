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

import type { Handler, Request, Response } from 'express';
import { Readable } from 'stream';
import type { UIConfigs } from '../configs.js';
import type { AuthorizeFn } from '../helpers/auth.js';
import {
  ArtifactPreviewError,
  ARTIFACT_PREVIEW_RESPONSE_HEADER,
  parsePreviewParentUri,
  readArtifactPreviewBytes,
} from '../helpers/artifact-preview.js';
import {
  ArtifactArtifactType,
  ArtifactServiceApi,
  Configuration,
} from '../src/generated/apisv2/artifact/index.js';
import { ResponseError } from '../src/generated/apisv2/artifact/runtime.js';
import { isAllowedResourceName } from '../utils.js';
import {
  getArtifactsAuthMiddleware,
  getArtifactsHandler,
  getArtifactServiceGetter,
  sendArtifactError,
  TOO_MANY_ARTIFACT_REQUESTS_MESSAGE,
  waitForArtifactOperation,
} from './artifacts.js';
import { isAllowedDomain } from './domain-checker.js';

const PREVIEW_REQUEST_TIMEOUT_MS = 30_000;

/** Returns the receiver's delay-seconds retry guidance; express-rate-limit never sends a date. */
function retryAfterSeconds(headers: Headers): string | undefined {
  const value = headers.get('retry-after')?.trim();
  return value && /^\d{1,9}$/.test(value) ? value : undefined;
}

function previewNamespace(request: Request, authEnabled: boolean): string | undefined {
  const namespace = request.query.namespace;
  if (
    Object.keys(request.query).some((name) => name !== 'namespace') ||
    (namespace !== undefined &&
      (typeof namespace !== 'string' || !isAllowedResourceName(namespace)))
  ) {
    throw new ArtifactPreviewError(
      400,
      'Preview requests accept only one valid namespace parameter.',
    );
  }
  if (authEnabled && !namespace) {
    throw new ArtifactPreviewError(400, 'Namespace is required for artifact previews.');
  }
  return typeof namespace === 'string' ? namespace : undefined;
}

async function proxyPreview(
  options: UIConfigs,
  artifactId: string,
  namespace: string,
  headers: Record<string, string>,
  signal: AbortSignal,
): Promise<Buffer> {
  const target = getArtifactServiceGetter(options.artifacts.proxy)(namespace);
  if (!isAllowedDomain(target, options.artifacts.allowedDomain)) {
    throw new ArtifactPreviewError(
      403,
      'The namespace artifact service is not an allowed destination.',
    );
  }
  const url = new URL(`/artifacts/${encodeURIComponent(artifactId)}/preview`, target);
  url.searchParams.set('namespace', namespace);
  // Forward identity, not caller-supplied storage coordinates or provider authority. The namespace
  // service resolves the same stored parent and applies its own credential configuration.
  const response = await fetch(url, { headers, signal, redirect: 'error' });
  if (response.status === 429) {
    await response.body?.cancel();
    // Report the receiver's quota decision as retryable, not as a broken artifact service.
    const retryAfter = retryAfterSeconds(response.headers);
    throw new ArtifactPreviewError(
      429,
      retryAfter ? TOO_MANY_ARTIFACT_REQUESTS_MESSAGE : 'Too many artifact requests. Retry later.',
      retryAfter,
    );
  }
  if (
    !response.ok ||
    response.headers.get(ARTIFACT_PREVIEW_RESPONSE_HEADER) !== '1' ||
    !response.body
  ) {
    await response.body?.cancel();
    const status = [401, 403, 404, 413, 422, 504].includes(response.status) ? response.status : 502;
    throw new ArtifactPreviewError(
      status,
      'The namespace artifact service could not return this preview. Check access and preview.json, and ensure the existing service supports artifact previews.',
    );
  }
  return readArtifactPreviewBytes(Readable.fromWeb(response.body), signal);
}

function sendPreviewFailure(error: unknown, response: Response): void {
  if (error instanceof ArtifactPreviewError) {
    if (error.retryAfterSeconds && !response.headersSent) {
      response.setHeader('Retry-After', error.retryAfterSeconds);
    }
    sendArtifactError(response, error.status, error.message);
  } else if (error instanceof ResponseError && [401, 403, 404].includes(error.response.status)) {
    sendArtifactError(
      response,
      error.response.status,
      'The parent artifact is unavailable or access was denied.',
    );
  } else if (error instanceof Error && error.name === 'TimeoutError') {
    sendArtifactError(
      response,
      504,
      'Artifact preview timed out. Check storage connectivity and retry.',
    );
  } else {
    console.error('[artifact-preview] Unable to resolve or read the preview.', error);
    sendArtifactError(
      response,
      502,
      'Unable to read the artifact preview. Check the artifact service and storage configuration, then retry.',
    );
  }
}

/** Reads a fixed preview file using the registered parent as the authorization boundary. */
export function getArtifactPreviewHandler(
  options: UIConfigs,
  apiServerAddress: string,
  authorizeFn: AuthorizeFn,
): Handler {
  const artifactService = new ArtifactServiceApi(new Configuration({ basePath: apiServerAddress }));
  return async (request, response) => {
    response.set({
      'Cache-Control': 'no-store',
      'X-Content-Type-Options': 'nosniff',
      'Content-Disposition': 'attachment; filename="preview.json"',
      [ARTIFACT_PREVIEW_RESPONSE_HEADER]: '1',
    });
    const disconnected = new AbortController();
    const cancel = () => disconnected.abort();
    request.once('aborted', cancel);
    response.once('close', cancel);
    const signal = AbortSignal.any([
      disconnected.signal,
      AbortSignal.timeout(PREVIEW_REQUEST_TIMEOUT_MS),
    ]);
    try {
      const artifactId = request.params.artifactId;
      if (typeof artifactId !== 'string' || !/^[a-zA-Z0-9_-]{1,128}$/.test(artifactId)) {
        throw new ArtifactPreviewError(400, 'A valid artifact ID is required for a preview.');
      }
      const requestedNamespace = previewNamespace(request, options.auth.enabled);
      const identityHeader = options.auth.kubeflowUserIdHeader.toLowerCase();
      const identity = request.headers[identityHeader];
      if (options.auth.enabled && (typeof identity !== 'string' || !identity)) {
        throw new ArtifactPreviewError(401, 'Authentication is required for artifact previews.');
      }
      const headers: Record<string, string> =
        typeof identity === 'string' ? { [identityHeader]: identity } : {};
      const artifact = await waitForArtifactOperation(
        artifactService.artifact_1(artifactId, { headers, signal }),
        signal,
      );
      if (
        (requestedNamespace && artifact.namespace && requestedNamespace !== artifact.namespace) ||
        (options.auth.enabled && !artifact.namespace)
      ) {
        throw new ArtifactPreviewError(403, 'Artifact does not belong to the requested namespace.');
      }
      const namespace = artifact.namespace || requestedNamespace || options.server.serverNamespace;
      if (!isAllowedResourceName(namespace)) {
        throw new ArtifactPreviewError(
          400,
          'The artifact has no valid namespace. Correct its namespace.',
        );
      }
      const parent = parsePreviewParentUri(artifact.uri || '');
      const contentPreview = { parent, namespace, signal };
      let authorized = false;
      await getArtifactsAuthMiddleware(
        authorizeFn,
        options.auth.enabled,
        options.auth.kubeflowUserIdHeader,
        apiServerAddress,
        options.artifacts.proxy.enabled,
        contentPreview,
      )(request, response, () => {
        authorized = true;
      });
      if (!authorized) {
        return;
      }
      if (
        artifact.type !== ArtifactArtifactType.Dataset &&
        artifact.type !== ArtifactArtifactType.Model
      ) {
        throw new ArtifactPreviewError(
          422,
          'Only Dataset and Model artifacts support content previews.',
        );
      }
      const descriptor = artifact.metadata?._kfp_preview;
      if (descriptor === undefined) {
        throw new ArtifactPreviewError(404, 'No preview is enabled for this artifact.');
      }
      if (
        !descriptor ||
        typeof descriptor !== 'object' ||
        Array.isArray(descriptor) ||
        !('version' in descriptor) ||
        descriptor.version !== 1 ||
        !('path' in descriptor) ||
        descriptor.path !== 'preview.json'
      ) {
        throw new ArtifactPreviewError(
          422,
          'Invalid preview descriptor. Use version 1 and path "preview.json".',
        );
      }
      if (options.artifacts.proxy.enabled) {
        const content = await proxyPreview(options, artifactId, namespace, headers, signal);
        response.type('application/json').send(content);
      } else {
        await getArtifactsHandler({
          artifactsConfigs: options.artifacts,
          useParameter: false,
          tryExtract: false,
          options,
          contentPreview,
        })(request, response, () => undefined);
      }
    } catch (error) {
      if (!request.aborted && !response.destroyed) {
        sendPreviewFailure(error, response);
      }
    } finally {
      request.removeListener('aborted', cancel);
      response.removeListener('close', cancel);
    }
  };
}
