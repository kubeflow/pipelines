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

import { applyArtifactPathPolicy, ARTIFACT_PATH_POLICIES } from './artifact-path.js';

describe('applyArtifactPathPolicy', () => {
  it('preserves repeated slashes for HTTP compatibility but rejects traversal and backslashes', () => {
    expect(applyArtifactPathPolicy('reports//output.csv', ARTIFACT_PATH_POLICIES.http)).toBe(
      'reports//output.csv',
    );
    expect(
      applyArtifactPathPolicy('reports/../secret', ARTIFACT_PATH_POLICIES.http),
    ).toBeUndefined();
    expect(applyArtifactPathPolicy('reports\\secret', ARTIFACT_PATH_POLICIES.http)).toBeUndefined();
  });

  it('rejects non-normalized ownership keys without changing object-store backslash semantics', () => {
    expect(
      applyArtifactPathPolicy('private-artifacts/team-a/output', ARTIFACT_PATH_POLICIES.ownership),
    ).toBe('private-artifacts/team-a/output');
    expect(
      applyArtifactPathPolicy('private-artifacts//team-a/output', ARTIFACT_PATH_POLICIES.ownership),
    ).toBeUndefined();
    expect(
      applyArtifactPathPolicy('private-artifacts/team-a\\output', ARTIFACT_PATH_POLICIES.ownership),
    ).toBe('private-artifacts/team-a\\output');
  });

  it('removes unsafe tar segments and returns an empty path when none remain', () => {
    expect(applyArtifactPathPolicy('/reports/../output.csv', ARTIFACT_PATH_POLICIES.tarEntry)).toBe(
      'reports/output.csv',
    );
    expect(applyArtifactPathPolicy('/../', ARTIFACT_PATH_POLICIES.tarEntry)).toBe('');
  });
});
