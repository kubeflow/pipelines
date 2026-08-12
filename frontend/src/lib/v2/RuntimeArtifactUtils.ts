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
  ArtifactArtifactType,
  InputOutputsIOArtifact,
  InputOutputsIOParameter,
  PipelineTaskTaskState,
  V2beta1Artifact,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import { STORE_SESSION_INFO_KEY } from 'src/lib/ReservedArtifactProperties';

const ARTIFACT_SCHEMA_TITLES: Partial<Record<ArtifactArtifactType, string>> = {
  [ArtifactArtifactType.Artifact]: 'system.Artifact',
  [ArtifactArtifactType.Model]: 'system.Model',
  [ArtifactArtifactType.Dataset]: 'system.Dataset',
  [ArtifactArtifactType.HTML]: 'system.HTML',
  [ArtifactArtifactType.Markdown]: 'system.Markdown',
  [ArtifactArtifactType.Metric]: 'system.Metrics',
  [ArtifactArtifactType.ClassificationMetric]: 'system.ClassificationMetrics',
  [ArtifactArtifactType.SlicedClassificationMetric]: 'system.SlicedClassificationMetrics',
};

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
  return artifact.type ? ARTIFACT_SCHEMA_TITLES[artifact.type] || artifact.type : '-';
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

export function getScalarMetricValue(artifact: V2beta1Artifact): string {
  return String(artifact.number_value ?? artifact.metadata?.[artifact.name || ''] ?? '-');
}

export function isScalarMetricArtifact(artifact: V2beta1Artifact): boolean {
  return artifact.type === ArtifactArtifactType.Metric;
}

export function isClassificationMetricArtifact(artifact: V2beta1Artifact): boolean {
  return (
    artifact.type === ArtifactArtifactType.ClassificationMetric ||
    artifact.type === ArtifactArtifactType.SlicedClassificationMetric
  );
}

export function isHtmlArtifact(artifact: V2beta1Artifact): boolean {
  return artifact.type === ArtifactArtifactType.HTML;
}

export function isMarkdownArtifact(artifact: V2beta1Artifact): boolean {
  return artifact.type === ArtifactArtifactType.Markdown;
}

export function isVisualizableArtifact(artifact: V2beta1Artifact): boolean {
  return (
    isScalarMetricArtifact(artifact) ||
    isClassificationMetricArtifact(artifact) ||
    isHtmlArtifact(artifact) ||
    isMarkdownArtifact(artifact)
  );
}

export function isTaskFinished(state: PipelineTaskTaskState | undefined): boolean {
  return (
    state === PipelineTaskTaskState.SUCCEEDED ||
    state === PipelineTaskTaskState.FAILED ||
    state === PipelineTaskTaskState.SKIPPED ||
    state === PipelineTaskTaskState.CACHED
  );
}
