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

type SegmentPolicy = 'allow' | 'reject' | 'remove';

interface ArtifactPathPolicy {
  backslashes: 'allow' | 'reject';
  dotSegments: SegmentPolicy;
  emptySegments: SegmentPolicy;
  parentSegments: SegmentPolicy;
}

export const ARTIFACT_PATH_POLICIES = {
  http: {
    backslashes: 'reject',
    dotSegments: 'reject',
    emptySegments: 'allow',
    parentSegments: 'reject',
  },
  ownership: {
    backslashes: 'allow',
    dotSegments: 'reject',
    emptySegments: 'reject',
    parentSegments: 'reject',
  },
  tarEntry: {
    backslashes: 'allow',
    dotSegments: 'remove',
    emptySegments: 'remove',
    parentSegments: 'remove',
  },
  volume: {
    backslashes: 'allow',
    dotSegments: 'allow',
    emptySegments: 'allow',
    parentSegments: 'reject',
  },
} as const satisfies Record<string, ArtifactPathPolicy>;

/** Applies the source-specific compatibility policy without changing safe segments. */
export function applyArtifactPathPolicy(
  path: string,
  policy: ArtifactPathPolicy,
): string | undefined {
  if (policy.backslashes === 'reject' && path.includes('\\')) {
    return undefined;
  }

  const result: string[] = [];
  for (const segment of path.split('/')) {
    const segmentPolicy =
      segment === ''
        ? policy.emptySegments
        : segment === '.'
          ? policy.dotSegments
          : segment === '..'
            ? policy.parentSegments
            : 'allow';
    if (segmentPolicy === 'reject') {
      return undefined;
    }
    if (segmentPolicy === 'allow') {
      result.push(segment);
    }
  }
  return result.join('/');
}
