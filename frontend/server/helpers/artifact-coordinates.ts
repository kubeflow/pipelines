// Copyright 2025 The Kubeflow Authors
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

import type { Request } from 'express';
import { buildArtifactUri } from './artifact-sources.js';

export interface ArtifactCoordinates<TSource extends string = string> {
  source: TSource;
  bucket: string;
  key: string;
  // Query-based preview routes carry the artifact URI path; download routes are already decoded.
  keyEncoding?: 'storage' | 'uri';
  // Exact escaped path spelling used by persisted artifact identity when `key` is decoded storage.
  uriKey?: string;
  artifactUriQuery?: string;
}

/** Converts a URI-path key to the object-store key representation used by the launcher. */
export function normalizeArtifactStorageCoordinates<TSource extends string>(
  coordinates: ArtifactCoordinates<TSource>,
): ArtifactCoordinates<TSource> {
  if (coordinates.keyEncoding !== 'uri') {
    return coordinates;
  }
  return {
    ...coordinates,
    key: decodeURIComponent(coordinates.key),
    keyEncoding: 'storage',
  };
}

export function resolveArtifactCoordinates(
  request: Pick<Request, 'path' | 'query'>,
): ArtifactCoordinates | null | undefined {
  const artifactPathStart = request.path.indexOf('/artifacts/');
  const artifactPath =
    artifactPathStart >= 0 ? request.path.slice(artifactPathStart) : request.path;
  const isExactGetEndpoint = artifactPath === '/artifacts/get';
  if (isExactGetEndpoint) {
    const asString = (value: unknown): string => (typeof value === 'string' ? value : '');
    return {
      source: asString(request.query.source),
      bucket: asString(request.query.bucket),
      key: asString(request.query.key),
      keyEncoding: 'uri',
      artifactUriQuery: asString(request.query.artifactUriQuery),
    };
  }

  const downloadPathMatch = artifactPath.match(/^\/artifacts\/([^/]+)\/([^/]+)\/(.+)$/);
  if (!downloadPathMatch) {
    return undefined;
  }
  try {
    const uriKey = downloadPathMatch[3];
    const key = decodeURIComponent(uriKey);
    return {
      source: decodeURIComponent(downloadPathMatch[1]),
      bucket: decodeURIComponent(downloadPathMatch[2]),
      key,
      keyEncoding: 'storage',
      ...(uriKey === key ? {} : { uriKey }),
      artifactUriQuery:
        typeof request.query.artifactUriQuery === 'string' ? request.query.artifactUriQuery : '',
    };
  } catch {
    return null;
  }
}

export function buildArtifactCoordinateUri(coordinates: ArtifactCoordinates): string {
  const artifactUri = buildArtifactUri(
    coordinates.source,
    coordinates.bucket,
    coordinates.uriKey ?? coordinates.key,
  );
  return coordinates.key && coordinates.artifactUriQuery
    ? `${artifactUri}?${coordinates.artifactUriQuery}`
    : artifactUri;
}

/**
 * Removes launcher provider configuration from a KFP artifact URI.
 *
 * Raw `?` starts the provider query. Native object-store parsing rejects percent-encoded query
 * delimiters in object paths, so object keys containing those delimiters are outside the supported
 * KFP artifact URI contract.
 */
export function stripArtifactUriQuery(artifactUri: string): string {
  const queryStart = artifactUri.indexOf('?');
  return queryStart < 0 ? artifactUri : artifactUri.slice(0, queryStart);
}
