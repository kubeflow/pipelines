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

import { readFileSync } from 'fs';
import { loadAll } from 'js-yaml';
import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';
import { describe, expect, it } from 'vitest';

interface PolicyRule {
  apiGroups?: string[];
  resources?: string[];
  verbs?: string[];
}

interface ClusterRole {
  kind?: string;
  metadata?: { name?: string };
  rules?: PolicyRule[];
}

const manifestPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../manifests/kustomize/base/installs/multi-user/view-edit-cluster-roles.yaml',
);
const roles = loadAll(readFileSync(manifestPath, 'utf8')) as ClusterRole[];

function artifactVerbs(roleName: string): string[] {
  const role = roles.find(
    (candidate) => candidate.kind === 'ClusterRole' && candidate.metadata?.name === roleName,
  );
  const rule = role?.rules?.find(
    (candidate) =>
      candidate.apiGroups?.includes('pipelines.kubeflow.org') &&
      candidate.resources?.includes('artifacts'),
  );
  return rule?.verbs || [];
}

describe('multi-user artifact RBAC', () => {
  it('allows namespace viewers to read but not mutate native artifacts', () => {
    expect(artifactVerbs('aggregate-to-kubeflow-pipelines-view').sort()).toEqual(['get', 'list']);
  });

  it('retains artifact creation in the edit aggregate', () => {
    expect(artifactVerbs('aggregate-to-kubeflow-pipelines-edit').sort()).toEqual([
      'create',
      'get',
      'list',
    ]);
  });
});
