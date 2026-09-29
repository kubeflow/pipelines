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

import {
  artifactProviderForSource,
  buildArtifactUri,
  isArtifactSource,
  isLauncherArtifactSource,
  requiresArtifactOwnershipValidation,
} from './artifact-sources.js';

describe('artifact sources', () => {
  it.each(['minio', 's3', 'gcs'])('keeps %s launcher and ownership handling aligned', (source) => {
    expect(isArtifactSource(source)).toBe(true);
    expect(isLauncherArtifactSource(source)).toBe(true);
    expect(requiresArtifactOwnershipValidation(source)).toBe(true);
  });

  it('maps the GCS request source to the launcher provider and URI scheme', () => {
    expect(artifactProviderForSource('gcs')).toBe('gs');
    expect(buildArtifactUri('gcs', 'bucket', 'key')).toBe('gs://bucket/key');
  });

  it('keeps volume local and rejects unknown sources', () => {
    expect(isArtifactSource('volume')).toBe(true);
    expect(isLauncherArtifactSource('volume')).toBe(false);
    expect(requiresArtifactOwnershipValidation('volume')).toBe(false);
    expect(isArtifactSource('unknown')).toBe(false);
  });
});
