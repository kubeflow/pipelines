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

export const ARTIFACT_SOURCES = ['minio', 's3', 'gcs', 'http', 'https', 'volume'] as const;
export type ArtifactSource = (typeof ARTIFACT_SOURCES)[number];

export const LAUNCHER_ARTIFACT_SOURCES = [
  'minio',
  's3',
  'gcs',
] as const satisfies readonly ArtifactSource[];
export type LauncherArtifactSource = (typeof LAUNCHER_ARTIFACT_SOURCES)[number];
export type ArtifactProvider = Exclude<LauncherArtifactSource, 'gcs'> | 'gs';

const ARTIFACT_SOURCE_SET: ReadonlySet<string> = new Set(ARTIFACT_SOURCES);
const LAUNCHER_ARTIFACT_SOURCE_SET: ReadonlySet<string> = new Set(LAUNCHER_ARTIFACT_SOURCES);
const OWNERSHIP_VALIDATED_ARTIFACT_SOURCE_SET: ReadonlySet<string> = new Set([
  ...LAUNCHER_ARTIFACT_SOURCES,
  'http',
  'https',
]);

export function isArtifactSource(source: string): source is ArtifactSource {
  return ARTIFACT_SOURCE_SET.has(source);
}

export function isLauncherArtifactSource(source: string): source is LauncherArtifactSource {
  return LAUNCHER_ARTIFACT_SOURCE_SET.has(source);
}

export function requiresArtifactOwnershipValidation(source: string): boolean {
  return OWNERSHIP_VALIDATED_ARTIFACT_SOURCE_SET.has(source);
}

export function artifactProviderForSource(source: LauncherArtifactSource): ArtifactProvider {
  return source === 'gcs' ? 'gs' : source;
}

export function buildArtifactUri(source: string, bucket: string, key: string): string {
  const scheme = source === 'gcs' ? 'gs' : source;
  return `${scheme}://${bucket}/${key}`;
}
