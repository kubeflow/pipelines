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
import { buildArtifactCoordinateUri } from './artifact-coordinates.js';
import {
  MAX_ARTIFACT_PREVIEW_BYTES,
  parsePreviewParentUri,
  readArtifactPreviewBytes,
} from './artifact-preview.js';

describe('preview parent identity', () => {
  it.each([
    's3://bucket/private-artifacts/team-a/run/dataset',
    'minio://bucket/private-artifacts/team-a/run/dataset/',
    'gs://bucket/private-artifacts/team-a/with%20space?anonymous=true',
    's3://bucket/private-artifacts/team-a/%E2%82%AC/100%25?region=us-west-1',
    's3://bucket/%70rivate-artifacts/team-a/output',
  ])('preserves the exact registered identity: %s', (uri) => {
    expect(buildArtifactCoordinateUri(parsePreviewParentUri(uri))).toBe(uri);
  });

  it.each([
    's3://bucket/',
    's3://bucket/parent//',
    's3://bucket/parent/../another',
    's3://bucket/parent/%2e%2e/another',
    's3://bucket/parent/%2Fanother',
    's3://bucket/parent/%26another',
    's3://bucket/parent/%3Fanother',
    's3://bucket/parent/%',
    's3://bucket/parent/%2a',
    's3://bucket/parent?',
    's3://bucket/parent#fragment',
    'https://example.com/parent',
    'volume://pvc/parent',
    'oci://registry/model',
  ])('rejects unsupported or ambiguous parent paths: %s', (uri) => {
    expect(() => parsePreviewParentUri(uri)).toThrow();
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
