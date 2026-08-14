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
 *   - Infinity: appropriate for immutable or static reference data (published
 *     pipeline versions, historical run records).
 *   - Finite / refetchInterval: use for data that may change while the user
 *     is viewing the page (e.g. active run task state in RunDetailsV2).
 */
export const queryKeys = {
  // --- Shared hooks (actively imported) ---

  pipelineVersionTemplate: (pipelineId?: string, pipelineVersionId?: string) =>
    ['PipelineVersionTemplate', { pipelineId, pipelineVersionId }] as const,

  // --- Run & recurring run detail ---

  v2RunDetail: (runId: string | null | undefined) => ['v2_run_detail', { id: runId }] as const,

  v2RunDetailSingle: (runId: string | null | undefined) => ['v2_run_details', runId] as const,

  v2RunDetails: (runIds: string[]) => ['v2_run_details', { ids: runIds }] as const,

  v2RunComparison: (runId: string) => ['v2_run_comparison', { id: runId }] as const,

  v2RecurringRunDetail: (recurringRunId: string | null | undefined) =>
    ['v2_recurring_run_detail', { id: recurringRunId }] as const,

  recurringRun: (recurringRunId: string | null | undefined) =>
    ['recurringRun', recurringRunId] as const,

  runDetailForComparisonRouting: (runId: string) =>
    ['run_detail_for_comparison_routing', { id: runId }] as const,

  // --- Runtime metadata ---

  runTasks: (runId: string, retryRefreshVersion?: number) =>
    retryRefreshVersion === undefined
      ? (['run_tasks', { id: runId }] as const)
      : (['run_tasks', { id: runId, retryRefreshVersion }] as const),

  artifactTasksPage: (artifactId: string, pageToken?: string, pageSize?: number) =>
    ['artifact_tasks', { artifactId, pageSize, pageToken }] as const,

  artifactVisualizationKey: (artifactId: string) =>
    ['artifact_visualization_key', { id: artifactId }] as const,

  taskLogs: (
    taskId?: string,
    taskState?: string,
    namespace?: string,
    sourceIdentity?: string,
    sourceFinished?: boolean,
  ) => ['task_logs', { taskId, taskState, namespace, sourceIdentity, sourceFinished }] as const,

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

  runtimeArtifactVisualization: (
    artifactId: string | undefined,
    namespace?: string,
    sourceFinished?: boolean,
  ) => ['runtime_artifact_visualization', artifactId, namespace, sourceFinished] as const,

  legacyRuntimeUiMetadata: (
    artifactId: string | undefined,
    namespace?: string,
    sourceFinished?: boolean,
  ) => ['legacy_runtime_ui_metadata', artifactId, namespace, sourceFinished] as const,

  // --- Misc ---

  artifactPreview: (
    value: string | undefined,
    namespace: string | undefined,
    artifactUriQuery: string | undefined,
    providerInfo: string | undefined,
    maxbytes: number,
    maxlines: number,
  ) =>
    [
      'artifact_preview',
      { value, namespace, artifactUriQuery, providerInfo, maxbytes, maxlines },
    ] as const,
};
