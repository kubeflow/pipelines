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

import { V2beta1Artifact } from 'src/apisv2beta1/run';
import { Apis } from 'src/lib/Apis';
import WorkflowParser from 'src/lib/WorkflowParser';

export interface ArtifactFileLocation {
  path: ReturnType<typeof WorkflowParser.parseStoragePath>;
  artifactUriQuery?: string;
}

function canonicalizeArtifactUriKey(key: string): string {
  try {
    return encodeURI(decodeURIComponent(key));
  } catch {
    // A raw percent sign is object-key data, not a malformed URI escape.
    return encodeURI(key);
  }
}

export function parseArtifactFileLocation(uri: string): ArtifactFileLocation {
  const queryStart = uri.indexOf('?');
  const uriWithoutQuery = queryStart < 0 ? uri : uri.slice(0, queryStart);
  const query = queryStart < 0 ? '' : uri.slice(queryStart + 1);
  const parsedPath = WorkflowParser.parseStoragePath(uriWithoutQuery);
  const isLauncherArtifact = ['gcs', 'minio', 's3'].includes(parsedPath.source);
  const path = isLauncherArtifact
    ? {
        ...parsedPath,
        key: canonicalizeArtifactUriKey(parsedPath.key),
        keyEncoding: 'uri' as const,
      }
    : { ...parsedPath, keyEncoding: 'storage' as const };
  if (!query || !isLauncherArtifact) {
    return { path };
  }

  return { path, artifactUriQuery: query };
}

export function readArtifactFile(artifact: V2beta1Artifact, namespace?: string): Promise<string> {
  if (!artifact.uri) {
    return Promise.reject(new Error('Artifact has no URI. Verify the artifact output location.'));
  }
  const location = parseArtifactFileLocation(artifact.uri);
  return Apis.readFile({
    path: location.path,
    namespace: namespace || artifact.namespace,
    artifactUriQuery: location.artifactUriQuery,
  });
}
