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

export function readArtifactFile(artifact: V2beta1Artifact, namespace?: string): Promise<string> {
  if (!artifact.uri) {
    return Promise.reject(new Error('Artifact has no URI. Verify the artifact output location.'));
  }
  return Apis.readFile({
    path: WorkflowParser.parseStoragePath(artifact.uri),
    namespace: namespace || artifact.namespace,
  });
}
