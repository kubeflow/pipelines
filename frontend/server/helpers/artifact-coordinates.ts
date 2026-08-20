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
import {
  buildArtifactUri,
  isArtifactSource,
  isLauncherArtifactSource,
} from './artifact-sources.js';
import { applyArtifactPathPolicy, ARTIFACT_PATH_POLICIES } from './artifact-path.js';

export interface ArtifactCoordinates<TSource extends string = string> {
  source: TSource;
  bucket: string;
  key: string;
  // Preview callers declare whether their key is decoded storage text or a canonical URI path.
  keyEncoding?: 'storage' | 'uri';
  // Exact path spelling used by persisted artifact identity when `key` is decoded storage.
  uriKey?: string;
  artifactUriQuery?: string;
}

/**
 * Rejects alternate URI spellings that decode to an object key with a different identity.
 *
 * For authorization, we require exactly one URI spelling for each decoded storage object.
 */
export function isCanonicalArtifactUriKey(key: string): boolean {
  try {
    const decodedKey = decodeURIComponent(key);
    // Query and fragment delimiters are not supported inside native KFP object keys.
    // Uppercase escapes provide one authorization identity for each decoded storage object.
    return !/%26/i.test(key) && !/[?#]/.test(decodedKey) && key === encodeURI(decodedKey);
  } catch {
    return false;
  }
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

function decodeExactArtifactUriKey(uriKey: string, source: string): string | undefined {
  try {
    const decodedKey = decodeURIComponent(uriKey);
    const storageKey =
      isLauncherArtifactSource(source) && decodedKey.endsWith('/')
        ? decodedKey.slice(0, -1)
        : decodedKey;
    const pathPolicy =
      source === 'http' || source === 'https'
        ? ARTIFACT_PATH_POLICIES.http
        : source === 'volume'
          ? undefined
          : ARTIFACT_PATH_POLICIES.ownership;
    if (
      /%2f/i.test(uriKey) ||
      /%26/i.test(uriKey) ||
      (pathPolicy !== undefined && applyArtifactPathPolicy(storageKey, pathPolicy) === undefined) ||
      /[?#]/.test(storageKey)
    ) {
      return undefined;
    }
    return storageKey;
  } catch {
    return undefined;
  }
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
    const source = asString(request.query.source);
    const bucket = asString(request.query.bucket);
    const requestKey = asString(request.query.key);
    const requestUriKey = asString(request.query.uriKey);
    const requestedKeyEncoding = asString(request.query.keyEncoding) || 'storage';
    const artifactUriQuery = asString(request.query.artifactUriQuery);
    if (requestUriKey) {
      const decodedUriKey = decodeExactArtifactUriKey(requestUriKey, source);
      if (decodedUriKey === undefined || decodedUriKey !== requestKey) {
        return null;
      }
      return {
        source,
        bucket,
        key: requestKey,
        keyEncoding: 'storage',
        uriKey: requestUriKey,
        artifactUriQuery,
      };
    }
    if (isArtifactSource(source) && !isLauncherArtifactSource(source)) {
      return {
        source,
        bucket,
        key: requestKey,
        keyEncoding: 'storage',
        artifactUriQuery,
      };
    }
    if (requestedKeyEncoding !== 'storage' && requestedKeyEncoding !== 'uri') {
      return null;
    }
    if (requestedKeyEncoding === 'uri') {
      if (!isCanonicalArtifactUriKey(requestKey)) {
        return null;
      }
      return { source, bucket, key: requestKey, keyEncoding: 'uri', artifactUriQuery };
    }

    // Legacy preview callers pass decoded object-store keys. Keep that storage spelling intact,
    // while encoding it once for the distinct URI identity used by ownership validation.
    if (/[?#]/.test(requestKey)) {
      return null;
    }
    const uriKey = encodeURI(requestKey);
    return {
      source,
      bucket,
      key: requestKey,
      keyEncoding: 'storage',
      ...(uriKey === requestKey ? {} : { uriKey }),
      artifactUriQuery,
    };
  }

  const downloadPathMatch = artifactPath.match(/^\/artifacts\/([^/]+)\/([^/]+)\/(.+)$/);
  if (!downloadPathMatch) {
    return undefined;
  }
  try {
    const uriKey = downloadPathMatch[3];
    if (!isCanonicalArtifactUriKey(uriKey)) {
      return null;
    }
    const source = decodeURIComponent(downloadPathMatch[1]);
    const decodedKey = decodeURIComponent(uriKey);
    const key =
      isLauncherArtifactSource(source) && decodedKey.endsWith('/')
        ? decodedKey.slice(0, -1)
        : decodedKey;
    const requestedIdentityKey =
      typeof request.query.uriKey === 'string' ? request.query.uriKey : undefined;
    if (requestedIdentityKey !== undefined) {
      const decodedIdentityKey = decodeExactArtifactUriKey(requestedIdentityKey, source);
      if (decodedIdentityKey === undefined || decodedIdentityKey !== key) {
        return null;
      }
    }
    return {
      source,
      bucket: decodeURIComponent(downloadPathMatch[2]),
      key,
      keyEncoding: 'storage',
      ...((requestedIdentityKey ?? uriKey) === key
        ? {}
        : { uriKey: requestedIdentityKey ?? uriKey }),
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
