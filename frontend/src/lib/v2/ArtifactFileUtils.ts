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

function decodeArtifactUriKey(key: string, enforceLauncherPathPolicy: boolean): string {
  try {
    // Native artifacts use Go URL parsing in SplitObjectURI: valid escapes are decoded for storage
    // and malformed raw percent text is rejected. Exact persisted spelling is carried in uriKey.
    const decodedKey = decodeURIComponent(key);
    const segments = decodedKey.split('/');
    if (decodedKey.endsWith('/')) {
      segments.pop();
    }
    if (
      /[?#]/.test(decodedKey) ||
      (enforceLauncherPathPolicy &&
        (/%2f/i.test(key) ||
          (decodedKey !== '' &&
            segments.some((segment) => segment === '' || segment === '.' || segment === '..'))))
    ) {
      throw new Error(
        'Artifact URI keys cannot contain empty, relative, query, or fragment path segments.',
      );
    }
    return decodedKey;
  } catch (error) {
    throw new Error(`Artifact URI key has invalid encoding. Correct the artifact URI: ${error}`, {
      cause: error,
    });
  }
}

export function parseArtifactFileLocation(uri: string): ArtifactFileLocation {
  const queryStart = uri.indexOf('?');
  const uriWithoutQuery = queryStart < 0 ? uri : uri.slice(0, queryStart);
  const query = queryStart < 0 ? '' : uri.slice(queryStart + 1);
  const parsedPath = WorkflowParser.parseStoragePath(uriWithoutQuery);
  const schemeEnd = uriWithoutQuery.indexOf('://');
  const keyStart = uriWithoutQuery.indexOf('/', schemeEnd + 3);
  const uriKey = keyStart < 0 ? '' : uriWithoutQuery.slice(keyStart + 1);
  const isLauncherArtifact = ['gcs', 'minio', 's3'].includes(parsedPath.source);
  const key = decodeArtifactUriKey(uriKey, isLauncherArtifact);
  const path = {
    ...parsedPath,
    key,
    keyEncoding: 'storage' as const,
    ...(uriKey === encodeURI(key) ? {} : { uriKey }),
  };
  if (!query || !isLauncherArtifact) {
    return { path };
  }

  return { path, artifactUriQuery: query };
}

export async function readArtifactFile(
  artifact: V2beta1Artifact,
  namespace?: string,
): Promise<string> {
  if (!artifact.uri) {
    throw new Error('Artifact has no URI. Verify the artifact output location.');
  }
  const location = parseArtifactFileLocation(artifact.uri);
  return await Apis.readFile({
    path: location.path,
    namespace: namespace || artifact.namespace,
    artifactUriQuery: location.artifactUriQuery,
  });
}
