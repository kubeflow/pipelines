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

import { describe, expect, it } from 'vitest';
import {
  isCanonicalArtifactUriKey,
  normalizeArtifactStorageCoordinates,
  stripArtifactUriQuery,
} from './artifact-coordinates.js';

describe('isCanonicalArtifactUriKey', () => {
  it.each(['root%20dir/artifact', 'caf%C3%A9/model', '100%25complete', 'literal/path'])(
    'accepts canonical key %s',
    (key) => expect(isCanonicalArtifactUriKey(key, 's3')).toBe(true),
  );

  it.each([
    '%73ecret',
    'path%2Fsecret',
    'caf%c3%a9/model',
    'raw space',
    'raw-café',
    'query%3Fkey',
    'query%26key',
    'bad%ZZkey',
  ])('rejects alias or malformed key %s', (key) =>
    expect(isCanonicalArtifactUriKey(key, 's3')).toBe(false),
  );

  it('accepts encoded ampersands for HTTP while launcher sources reject them', () => {
    expect(isCanonicalArtifactUriKey('reports/A%26B.csv', 'https')).toBe(true);
    expect(isCanonicalArtifactUriKey('reports/A%26B.csv', 's3')).toBe(false);
  });
});

describe('normalizeArtifactStorageCoordinates', () => {
  it('decodes URI-path keys exactly once for storage', () => {
    expect(
      normalizeArtifactStorageCoordinates({
        source: 's3',
        bucket: 'bucket',
        key: 'root%20dir/100%25complete',
        keyEncoding: 'uri',
      }),
    ).toEqual({
      source: 's3',
      bucket: 'bucket',
      key: 'root dir/100%complete',
      keyEncoding: 'storage',
    });
  });

  it('does not decode an already-normalized storage key again', () => {
    const coordinates = {
      source: 's3',
      bucket: 'bucket',
      key: 'root dir/100%complete',
      keyEncoding: 'storage' as const,
    };

    expect(normalizeArtifactStorageCoordinates(coordinates)).toBe(coordinates);
  });

  it('rejects malformed URI-path encoding', () => {
    expect(() =>
      normalizeArtifactStorageCoordinates({
        source: 's3',
        bucket: 'bucket',
        key: 'root%2/artifact',
        keyEncoding: 'uri',
      }),
    ).toThrow(URIError);
  });
});

describe('stripArtifactUriQuery', () => {
  it('treats the first raw question mark as the provider-query boundary', () => {
    expect(
      stripArtifactUriQuery(
        's3://bucket/root/model?endpoint=https%3A%2F%2Fstore.example%3A9443&token=a%3Fb',
      ),
    ).toBe('s3://bucket/root/model');
  });
});
