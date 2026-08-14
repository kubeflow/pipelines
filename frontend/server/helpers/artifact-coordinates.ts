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

export function resolveArtifactCoordinates(
  request: Pick<Request, 'path' | 'query'>,
): { source: string; bucket: string; key: string } | null | undefined {
  const artifactPathStart = request.path.indexOf('/artifacts/');
  const artifactPath =
    artifactPathStart >= 0 ? request.path.slice(artifactPathStart) : request.path;
  const isExactGetEndpoint = artifactPath === '/artifacts/get';
  if (isExactGetEndpoint) {
    const asString = (value: unknown): string => (typeof value === 'string' ? value : '');
    return {
      source: asString(request.query.source),
      bucket: asString(request.query.bucket),
      key: appendArtifactUriQuery(
        asString(request.query.key),
        asString(request.query.artifactUriQuery),
      ),
    };
  }

  const downloadPathMatch = artifactPath.match(/^\/artifacts\/([^/]+)\/([^/]+)\/(.+)$/);
  if (!downloadPathMatch) {
    return undefined;
  }
  try {
    return {
      source: decodeURIComponent(downloadPathMatch[1]),
      bucket: decodeURIComponent(downloadPathMatch[2]),
      key: appendArtifactUriQuery(
        decodeURIComponent(downloadPathMatch[3]),
        typeof request.query.artifactUriQuery === 'string' ? request.query.artifactUriQuery : '',
      ),
    };
  } catch {
    return null;
  }
}

export function appendArtifactUriQuery(key: string, artifactUriQuery: string): string {
  return key && artifactUriQuery ? `${key}?${artifactUriQuery}` : key;
}
