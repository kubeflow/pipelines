/*
 * Copyright 2025 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Central query key factory for React Query.
 *
 * Eliminates ad-hoc string literals spread across components, prevents typos,
 * and makes cache invalidation discoverable.
 *
 * staleTime guidelines:
 *   - Infinity: appropriate for immutable or static reference data (artifact
 *     types, published pipeline versions, historical run records).
 *   - Finite / refetchInterval: use for data that may change while the user
 *     is viewing the page (e.g. active run MLMD state in RunDetailsV2).
 */
export const queryKeys = {
  // --- Shared hooks (actively imported) ---

  artifactTypes: () => ['artifact_types'] as const,

  pipelineVersionTemplate: (pipelineId?: string, pipelineVersionId?: string) =>
    ['PipelineVersionTemplate', { pipelineId, pipelineVersionId }] as const,

  // --- Run & recurring run detail ---

  v2RunDetail: (runId: string | null | undefined) => ['v2_run_detail', { id: runId }] as const,

  v2RunDetailSingle: (runId: string | null | undefined) => ['v2_run_details', runId] as const,

  v2RunDetails: (runIds: string[]) => ['v2_run_details', { ids: runIds }] as const,

  v2RecurringRunDetail: (recurringRunId: string | null | undefined) =>
    ['v2_recurring_run_detail', { id: recurringRunId }] as const,

  recurringRun: (recurringRunId: string | null | undefined) =>
    ['recurringRun', recurringRunId] as const,

  runDetails: (runIds: string[]) => ['run_details', { ids: runIds }] as const,

  // --- MLMD ---

  runArtifacts: (runIds: string[]) => ['run_artifacts', { runIds }] as const,

  mlmdPackage: (runId: string) => ['mlmd_package', { id: runId }] as const,

  executionArtifact: (executionId: number, executionState: number) =>
    ['execution_artifact', { id: executionId, state: executionState }] as const,

  executionOutputArtifact: (executionId: number, executionState: number) =>
    ['execution_output_artifact', { id: executionId, state: executionState }] as const,

  executionLogs: (executionId: number | undefined, namespace: string | undefined) =>
    ['execution_logs', { executionId, namespace }] as const,

  contextByExecution: (executionId: number, executionState: number) =>
    ['context_by_execution', { id: executionId, state: executionState }] as const,

  // --- Pipeline & version ---

  pipeline: (pipelineId: string | null | undefined) => ['pipeline', pipelineId] as const,

  // Includes both IDs for correct cache invalidation (version IDs may not be globally unique).
  pipelineVersion: (
    pipelineId: string | null | undefined,
    pipelineVersionId: string | null | undefined,
  ) => ['pipelineVersion', pipelineId, pipelineVersionId] as const,

  pipelineVersions: (pipelineId: string | null | undefined) =>
    ['pipeline_versions', pipelineId ?? ''] as const,

  // Includes both IDs for correct cache invalidation (version IDs may not be globally unique).
  v1PipelineVersionTemplate: (
    pipelineId: string | null | undefined,
    pipelineVersionId: string | null | undefined,
  ) => ['v1PipelineVersionTemplate', pipelineId, pipelineVersionId] as const,

  // --- Experiment ---

  experiment: (experimentId: string | null | undefined) => ['experiment', experimentId] as const,

  runDetailsV2Experiment: (runId: string, experimentId: string | null) =>
    ['RunDetailsV2_experiment', { runId, experimentId }] as const,

  // --- Viewer configs ---

  viewConfig: (artifactId: number | undefined, executionState: number, namespace?: string) =>
    ['viewconfig', { artifact: artifactId, state: executionState, namespace }] as const,

  htmlViewerConfig: (artifactIds: number[], executionState: number, namespace?: string) =>
    ['htmlViewerConfig', { artifacts: artifactIds, state: executionState, namespace }] as const,

  markdownViewerConfig: (artifactIds: number[], executionState: number, namespace?: string) =>
    ['markdownViewerConfig', { artifacts: artifactIds, state: executionState, namespace }] as const,

  visualizationPanelViewerConfig: (artifactId: number | undefined, namespace?: string) =>
    ['viewerConfig', { artifact: artifactId, namespace }] as const,

  // --- Misc ---

  artifactPreview: (
    value: string | undefined,
    namespace: string | undefined,
    maxbytes: number,
    maxlines: number,
  ) => ['artifact_preview', { value, namespace, maxbytes, maxlines }] as const,
};
