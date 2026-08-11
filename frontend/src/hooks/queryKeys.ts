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

  v2RunComparison: (runIds: string[]) => ['v2_run_comparison', { ids: runIds }] as const,

  v2RecurringRunDetail: (recurringRunId: string | null | undefined) =>
    ['v2_recurring_run_detail', { id: recurringRunId }] as const,

  recurringRun: (recurringRunId: string | null | undefined) =>
    ['recurringRun', recurringRunId] as const,

  runDetails: (runIds: string[]) => ['run_details', { ids: runIds }] as const,

  // --- Runtime metadata ---

  runTasks: (runId: string) => ['run_tasks', { id: runId }] as const,

  taskLogs: (taskId?: string, taskState?: string, namespace?: string) =>
    ['task_logs', { taskId, taskState, namespace }] as const,

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

  runtimeArtifactVisualizations: (artifactIds: string[], namespace?: string) =>
    ['runtime_artifact_visualizations', { artifactIds, namespace }] as const,

  // --- Misc ---

  artifactPreview: (
    value: string | undefined,
    namespace: string | undefined,
    maxbytes: number,
    maxlines: number,
  ) => ['artifact_preview', { value, namespace, maxbytes, maxlines }] as const,
};
