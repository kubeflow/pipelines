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

export const MAX_ARTIFACT_PREVIEW_BYTES = 64 * 1024;
export const ARTIFACT_PREVIEW_FILE_NAME = '_kfp_preview.json';
const PREVIEW_SOURCES = new Set(['gcs', 'minio', 's3']);
const MAX_OBJECT_KEY_LENGTH = 1024;

export class ArtifactPreviewError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ArtifactPreviewError';
  }
}

/**
 * Returns the fixed preview object beneath an authorized parent artifact directory.
 *
 * The parent key, not the derived child, is the authorization identity: _kfp_preview.json has no
 * metadata record of its own, so it is only readable through its registered parent.
 */
export function getPreviewObjectKey(source: string, parentKey: string): string {
  if (!PREVIEW_SOURCES.has(source)) {
    throw new ArtifactPreviewError(
      422,
      'Previews require a directory artifact on S3, MinIO, or GCS.',
    );
  }
  const key = parentKey.replace(/\/$/, '');
  const segments = key.split('/');
  if (
    !key ||
    key.includes('*') ||
    segments.some((segment) => !segment || segment === '.' || segment === '..') ||
    key.length + ARTIFACT_PREVIEW_FILE_NAME.length + 1 > MAX_OBJECT_KEY_LENGTH
  ) {
    throw new ArtifactPreviewError(
      422,
      'The parent artifact URI is not a supported exact directory path.',
    );
  }
  return `${key}/${ARTIFACT_PREVIEW_FILE_NAME}`;
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
          'Preview exceeds 64 KiB. Regenerate a smaller _kfp_preview.json.',
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
        'Preview is not valid UTF-8. Regenerate _kfp_preview.json as UTF-8 JSON.',
      );
    }
    return result;
  } finally {
    signal.removeEventListener('abort', abort);
    source.destroy();
  }
}
