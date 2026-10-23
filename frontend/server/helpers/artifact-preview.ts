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
/** Preview stored inside the artifact directory (v1, manual producers). */
export const IN_DIRECTORY_PREVIEW_SUFFIX = '/_kfp_preview.json';
/** Preview stored beside the artifact directory (v2 sibling, SDK default). */
export const SIBLING_PREVIEW_SUFFIX = '.kfp-preview.json';
export type ArtifactPreviewStorage = 'directory' | 'sibling';
export const MISSING_PREVIEW_MESSAGE =
  'The preview file is missing. Regenerate the artifact with its preview.';
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
 * Parses the preview storage requested by the UI from the artifact's `_kfp_preview` descriptor.
 * Requests without a storage value keep the original in-directory v1 location.
 */
export function parsePreviewStorage(value: unknown): ArtifactPreviewStorage {
  if (value === undefined || value === 'directory') {
    return 'directory';
  }
  if (value === 'sibling') {
    return 'sibling';
  }
  throw new ArtifactPreviewError(400, 'Preview storage must be "directory" or "sibling".');
}

/** Returns the filename sent with a preview response for the given storage. */
export function getPreviewFileName(storage: ArtifactPreviewStorage): string {
  return storage === 'sibling' ? 'kfp-preview.json' : '_kfp_preview.json';
}

/**
 * Returns the fixed preview object derived from an authorized parent artifact directory: either
 * `<parent>/_kfp_preview.json` (v1) or the sibling `<parent>.kfp-preview.json` (v2).
 *
 * The parent key, not the derived object, is the authorization identity: the preview has no
 * metadata record of its own, so it is only readable through its registered parent.
 */
export function getPreviewObjectKey(
  source: string,
  parentKey: string,
  storage: ArtifactPreviewStorage = 'directory',
): string {
  if (!PREVIEW_SOURCES.has(source)) {
    throw new ArtifactPreviewError(
      422,
      'Previews require a directory artifact on S3, MinIO, or GCS.',
    );
  }
  const key = parentKey.replace(/\/$/, '');
  const suffix = storage === 'sibling' ? SIBLING_PREVIEW_SUFFIX : IN_DIRECTORY_PREVIEW_SUFFIX;
  const segments = key.split('/');
  if (
    !key ||
    key.includes('*') ||
    segments.some((segment) => !segment || segment === '.' || segment === '..') ||
    key.length + suffix.length > MAX_OBJECT_KEY_LENGTH
  ) {
    throw new ArtifactPreviewError(
      422,
      'The parent artifact URI is not a supported exact directory path.',
    );
  }
  return `${key}${suffix}`;
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
          'Preview exceeds 64 KiB. Regenerate a smaller preview.',
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
        'Preview is not valid UTF-8. Regenerate the preview as UTF-8 JSON.',
      );
    }
    return result;
  } finally {
    signal.removeEventListener('abort', abort);
    source.destroy();
  }
}
