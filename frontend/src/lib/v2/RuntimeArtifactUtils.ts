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
  InputOutputsIOArtifact,
  InputOutputsIOParameter,
  PipelineTaskTaskState,
  V2beta1Artifact,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';

export const STORE_SESSION_INFO_KEY = 'store_session_info';

export interface RuntimeArtifactEntry {
  artifact: V2beta1Artifact;
  artifactKey: string;
  group: InputOutputsIOArtifact;
  index: number;
}

export function flattenArtifactGroups(
  groups: InputOutputsIOArtifact[] | undefined,
): RuntimeArtifactEntry[] {
  return (groups || []).flatMap((group) =>
    (group.artifacts || []).map((artifact, index) => ({
      artifact,
      artifactKey: group.artifact_key || artifact.name || 'artifact',
      group,
      index,
    })),
  );
}

export function getArtifactDisplayName(
  artifact: V2beta1Artifact,
  artifactKey?: string,
  index?: number,
): string {
  const baseName = artifact.name || artifactKey || artifact.artifact_id || 'Artifact';
  return index && index > 0 ? `${baseName} (${index + 1})` : baseName;
}

export function getArtifactTypeName(artifact: V2beta1Artifact): string {
  return artifact.type ? `system.${artifact.type}` : '-';
}

export function getArtifactSessionInfo(artifact: V2beta1Artifact): string | undefined {
  const value = artifact.metadata?.[STORE_SESSION_INFO_KEY];
  return typeof value === 'string' ? value : undefined;
}

export function getOutputArtifactByName(
  task: V2beta1PipelineTask,
  name: string,
): V2beta1Artifact | undefined {
  return flattenArtifactGroups(task.outputs?.artifacts).find(
    ({ artifact, artifactKey }) => artifactKey === name || artifact.name === name,
  )?.artifact;
}

export function formatParameters(
  parameters: InputOutputsIOParameter[] | undefined,
): Array<[string, string]> {
  return (parameters || []).map((parameter) => [
    parameter.parameter_key || '-',
    formatParameterValue(parameter.value),
  ]);
}

export function formatParameterValue(value: object | undefined): string {
  if (value === undefined) {
    return '-';
  }
  if (typeof value === 'string') {
    return value;
  }
  return JSON.stringify(value);
}

export function isTaskFinished(state: PipelineTaskTaskState | undefined): boolean {
  return (
    state === PipelineTaskTaskState.SUCCEEDED ||
    state === PipelineTaskTaskState.FAILED ||
    state === PipelineTaskTaskState.SKIPPED ||
    state === PipelineTaskTaskState.CACHED
  );
}
