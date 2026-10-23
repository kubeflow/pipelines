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

import { getEventListeners } from 'events';
import { PassThrough, Readable } from 'stream';
import { describe, expect, it } from 'vitest';
import {
  getPreviewObjectKey,
  MAX_ARTIFACT_PREVIEW_BYTES,
  parsePreviewStorage,
  readArtifactPreviewBytes,
} from './artifact-preview.js';

describe('preview object key', () => {
  it.each([
    ['s3', 'private-artifacts/team-a/run/dataset'],
    ['minio', 'v2/artifacts/pipeline/run/task/dataset/'],
    ['gcs', 'root/with space/model'],
  ])('reads only _kfp_preview.json beneath the %s parent %s', (source, key) => {
    expect(getPreviewObjectKey(source, key)).toBe(`${key.replace(/\/$/, '')}/_kfp_preview.json`);
  });

  it.each([
    ['s3', 'private-artifacts/team-a/run/dataset'],
    ['minio', 'v2/artifacts/pipeline/run/task/dataset/'],
    ['gcs', 'root/with space/model'],
  ])('reads only the sibling preview beside the %s parent %s', (source, key) => {
    expect(getPreviewObjectKey(source, key, 'sibling')).toBe(
      `${key.replace(/\/$/, '')}.kfp-preview.json`,
    );
  });

  it.each([
    ['s3', ''],
    ['s3', '/'],
    ['s3', 'parent//'],
    ['s3', 'parent//child'],
    ['s3', '/parent'],
    ['s3', 'parent/../another'],
    ['s3', 'parent/./child'],
    ['s3', 'parent/*'],
    ['s3', 'a'.repeat(1024 - '_kfp_preview.json'.length)],
    ['http', 'parent'],
    ['https', 'parent'],
    ['volume', 'parent'],
  ])('rejects unsupported or ambiguous %s parent %s', (source, key) => {
    expect(() => getPreviewObjectKey(source, key)).toThrow();
  });

  it('accepts the longest parent whose preview key fits the object-key limit', () => {
    const key = 'a'.repeat(1024 - '/_kfp_preview.json'.length);
    expect(getPreviewObjectKey('s3', key)).toHaveLength(1024);
    const siblingKey = 'a'.repeat(1024 - '.kfp-preview.json'.length);
    expect(getPreviewObjectKey('s3', siblingKey, 'sibling')).toHaveLength(1024);
    expect(() => getPreviewObjectKey('s3', `${siblingKey}a`, 'sibling')).toThrow();
  });
});

describe('preview storage', () => {
  it('defaults to the in-directory v1 location', () => {
    expect(parsePreviewStorage(undefined)).toBe('directory');
    expect(parsePreviewStorage('directory')).toBe('directory');
    expect(parsePreviewStorage('sibling')).toBe('sibling');
  });

  it.each(['inline', '', '../other', ['sibling']])('rejects storage %j', (value) => {
    expect(() => parsePreviewStorage(value)).toThrow(/must be "directory" or "sibling"/);
  });
});

describe('bounded preview bytes', () => {
  it('accepts exactly the UTF-8 byte limit', async () => {
    const bytes = Buffer.from('\u{1f600}'.repeat(MAX_ARTIFACT_PREVIEW_BYTES / 4));
    await expect(
      readArtifactPreviewBytes(Readable.from([bytes]), new AbortController().signal),
    ).resolves.toEqual(bytes);
  });

  it('rejects one excess byte and stops an unbounded source', async () => {
    let chunksRead = 0;
    const source = new Readable({
      read() {
        chunksRead++;
        this.push(Buffer.alloc(MAX_ARTIFACT_PREVIEW_BYTES + 1, 'a'));
      },
    });
    await expect(
      readArtifactPreviewBytes(source, new AbortController().signal),
    ).rejects.toMatchObject({ status: 413 });
    expect(source.destroyed).toBe(true);
    expect(chunksRead).toBeLessThanOrEqual(2);
  });

  it('enforces the limit across chunks, not only against a length header', async () => {
    await expect(
      readArtifactPreviewBytes(
        Readable.from([Buffer.alloc(MAX_ARTIFACT_PREVIEW_BYTES, 'a'), Buffer.from('b')]),
        new AbortController().signal,
      ),
    ).rejects.toMatchObject({ status: 413 });
  });

  it('destroys a stalled stream on cancellation and removes its listener', async () => {
    const source = new PassThrough();
    const controller = new AbortController();
    const reading = readArtifactPreviewBytes(source, controller.signal);
    controller.abort(new Error('cancelled'));
    await expect(reading).rejects.toThrow('cancelled');
    expect(source.destroyed).toBe(true);
    expect(getEventListeners(controller.signal, 'abort')).toHaveLength(0);
  });

  it('rejects an already cancelled read', async () => {
    const source = new PassThrough();
    const controller = new AbortController();
    controller.abort();
    await expect(readArtifactPreviewBytes(source, controller.signal)).rejects.toThrow();
    expect(source.destroyed).toBe(true);
  });

  it('rejects invalid UTF-8 instead of substituting replacement characters', async () => {
    await expect(
      readArtifactPreviewBytes(Readable.from([Buffer.from([0xff])]), new AbortController().signal),
    ).rejects.toMatchObject({ status: 422 });
  });

  it('propagates source errors instead of returning partial data', async () => {
    const source = new PassThrough();
    const reading = readArtifactPreviewBytes(source, new AbortController().signal);
    source.write('{"partial":');
    source.destroy(new Error('storage failed'));
    await expect(reading).rejects.toThrow('storage failed');
  });
});
