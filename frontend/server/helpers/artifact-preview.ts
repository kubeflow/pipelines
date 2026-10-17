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

import type { Readable } from 'stream';
import { ArtifactCoordinates, resolveArtifactCoordinates } from './artifact-coordinates.js';
import { isLauncherArtifactSource, LauncherArtifactSource } from './artifact-sources.js';

export const MAX_ARTIFACT_PREVIEW_BYTES = 64 * 1024;
export const ARTIFACT_PREVIEW_RESPONSE_HEADER = 'X-KFP-Artifact-Preview';

export class ArtifactPreviewError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    /** Delay in seconds that a rate-limited caller should wait before retrying. */
    public readonly retryAfterSeconds?: string,
  ) {
    super(message);
    this.name = 'ArtifactPreviewError';
  }
}

export interface ArtifactContentPreviewRequest {
  parent: ArtifactCoordinates<LauncherArtifactSource>;
  namespace: string;
  signal: AbortSignal;
}

/** Resolves persisted identity without URL normalization changing the object or its owner. */
export function parsePreviewParentUri(uri: string): ArtifactCoordinates<LauncherArtifactSource> {
  const match = /^(gs|s3|minio):\/\/([^/?#]+)\/([^?#]+)(?:\?([^#]+))?$/.exec(uri);
  if (!match) {
    throw new ArtifactPreviewError(
      422,
      'Previews require a directory artifact on S3, MinIO, or GCS.',
    );
  }
  const source = match[1] === 'gs' ? 'gcs' : match[1];
  let key: string;
  try {
    key = decodeURIComponent(match[3]).replace(/\/$/, '');
  } catch {
    throw new ArtifactPreviewError(
      422,
      'The parent artifact URI has invalid encoding. Correct its URI.',
    );
  }
  const coordinates = resolveArtifactCoordinates({
    path: '/artifacts/get',
    query: {
      source,
      bucket: match[2],
      key,
      uriKey: match[3],
      artifactUriQuery: match[4] || '',
    },
  });
  if (
    !coordinates ||
    !isLauncherArtifactSource(coordinates.source) ||
    !key ||
    key.includes('*') ||
    key.length + '/preview.json'.length > 1024
  ) {
    throw new ArtifactPreviewError(
      422,
      'The parent artifact URI is not a supported exact directory path.',
    );
  }
  return { ...coordinates, source: coordinates.source };
}

/** Collects one small file, cancelling the source instead of draining an oversized object. */
export async function readArtifactPreviewBytes(
  source: Readable,
  signal: AbortSignal,
): Promise<Buffer> {
  const abort = () =>
    source.destroy(
      signal.reason instanceof Error ? signal.reason : new Error('Preview read cancelled.'),
    );
  if (signal.aborted) {
    source.destroy();
    signal.throwIfAborted();
  }
  signal.addEventListener('abort', abort, { once: true });
  const chunks: Buffer[] = [];
  let size = 0;
  try {
    for await (const chunk of source) {
      const bytes: Buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
      const accepted = bytes.subarray(0, MAX_ARTIFACT_PREVIEW_BYTES + 1 - size);
      chunks.push(Buffer.from(accepted));
      size += accepted.length;
      if (size > MAX_ARTIFACT_PREVIEW_BYTES) {
        throw new ArtifactPreviewError(
          413,
          'Preview exceeds 64 KiB. Regenerate a smaller preview.json.',
        );
      }
    }
    signal.throwIfAborted();
    const result = Buffer.concat(chunks, size);
    try {
      new TextDecoder('utf-8', { fatal: true }).decode(result);
    } catch {
      throw new ArtifactPreviewError(
        422,
        'Preview is not valid UTF-8. Regenerate preview.json as UTF-8 JSON.',
      );
    }
    return result;
  } finally {
    signal.removeEventListener('abort', abort);
    source.destroy();
  }
}
