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

import type { Request, Response } from 'express';
import type { Readable } from 'stream';
import {
  ArtifactPreviewError,
  ARTIFACT_PREVIEW_FILE_NAME,
  getPreviewObjectKey,
  readArtifactPreviewBytes,
} from '../helpers/artifact-preview.js';
import { isNoSuchKeyError } from '../minio-helper.js';

const PREVIEW_REQUEST_TIMEOUT_MS = 30_000;
const MISSING_PREVIEW_MESSAGE =
  '_kfp_preview.json is missing. Regenerate the artifact with its preview file.';

/** Opens one object from the storage provider already selected for the parent artifact. */
export type ArtifactPreviewObjectOpener = (
  objectKey: string,
  signal: AbortSignal,
) => Promise<Readable>;

function isMissingObjectError(error: unknown): boolean {
  if (isNoSuchKeyError(error)) {
    return true;
  }
  if (!error || typeof error !== 'object') {
    return false;
  }
  const status =
    'status' in error
      ? error.status
      : 'response' in error && error.response && typeof error.response === 'object'
        ? (error.response as { status?: unknown }).status
        : undefined;
  return status === 404;
}

function sendPreviewFailure(error: unknown, response: Response): void {
  if (response.headersSent || response.destroyed || response.writableEnded) {
    response.destroy();
    return;
  }
  response.setHeader('Cache-Control', 'no-store');
  if (error instanceof ArtifactPreviewError) {
    response.status(error.status).type('text/plain').send(error.message);
  } else if (isMissingObjectError(error)) {
    response.status(404).type('text/plain').send(MISSING_PREVIEW_MESSAGE);
  } else if (error instanceof Error && error.name === 'TimeoutError') {
    response
      .status(504)
      .type('text/plain')
      .send('Artifact preview timed out. Check storage connectivity and retry.');
  } else {
    console.error('[artifact-preview] Unable to read the preview.', error);
    response
      .status(502)
      .type('text/plain')
      .send('Unable to read the artifact preview. Check the storage configuration, then retry.');
  }
}

/**
 * Sends the bounded _kfp_preview.json beneath an already authorized parent artifact.
 *
 * Callers must authorize and resolve storage for the parent coordinates. This function derives
 * only the fixed child object, so a request cannot select another object through the preview
 * route, and it never lists, archives, extracts, or partially returns a directory.
 */
export async function sendArtifactContentPreview(
  request: Request,
  response: Response,
  source: string,
  parentKey: string,
  openObject: ArtifactPreviewObjectOpener,
): Promise<void> {
  response.setHeader('Cache-Control', 'no-store');
  response.setHeader('X-Content-Type-Options', 'nosniff');
  response.setHeader('Content-Disposition', `attachment; filename="${ARTIFACT_PREVIEW_FILE_NAME}"`);
  const disconnected = new AbortController();
  const cancel = () => disconnected.abort();
  request.once('aborted', cancel);
  response.once('close', cancel);
  const signal = AbortSignal.any([
    disconnected.signal,
    AbortSignal.timeout(PREVIEW_REQUEST_TIMEOUT_MS),
  ]);
  try {
    const objectKey = getPreviewObjectKey(source, parentKey);
    const stream = await openObject(objectKey, signal);
    const content = await readArtifactPreviewBytes(stream, signal);
    response.type('application/json').send(content);
  } catch (error) {
    if (!disconnected.signal.aborted) {
      sendPreviewFailure(error, response);
    }
  } finally {
    request.removeListener('aborted', cancel);
    response.removeListener('close', cancel);
  }
}
