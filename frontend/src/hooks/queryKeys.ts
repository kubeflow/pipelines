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
 * Centralized query key factory for React Query.
 * Prevents typos, enables discoverable cache invalidation, and documents query key structure.
 *
 * staleTime strategy:
 * - STALE_TIME_STATIC: Reference data (artifact types, pipeline specs) — rarely changes during session
 * - STALE_TIME_RUNTIME: Run/MLMD data — may update (e.g. RUNNING → SUCCEEDED); allows refetch on window focus
 *
 * Query key conventions:
 * - Single-entity keys use consistent shapes so cache is shared across call sites (e.g. v2RunDetail
 *   used by RunDetailsRouter and NewRunSwitcher share cache when fetching the same run).
 * - Keys accept optional IDs (string | undefined | null) for call sites where the ID comes from
 *   URL params; empty string is used in the key when disabled to avoid cache collisions.
 */
export const STALE_TIME_STATIC = Infinity;
export const STALE_TIME_RUNTIME = 60_000; // 60 seconds — run/MLMD can refetch when user returns to tab

export const queryKeys = {
  artifactTypes: () => ['artifact_types'] as const,

  pipelineVersionTemplate: (
    pipelineId: string | undefined,
    pipelineVersionId: string | undefined,
  ) => ['PipelineVersionTemplate', { pipelineId, pipelineVersionId }] as const,

  /** Single v2 run by ID. Shared cache for RunDetailsRouter and NewRunSwitcher. */
  v2RunDetail: (runId: string | undefined | null) =>
    ['v2_run_detail', { id: runId ?? '' }] as const,

  v2RunDetails: (runIds: string[]) => ['v2_run_details', { ids: runIds }] as const,

  v2RecurringRunDetail: (recurringRunId: string) =>
    ['v2_recurring_run_detail', { id: recurringRunId }] as const,

  recurringRun: (recurringRunId: string | undefined | null) =>
    ['recurringRun', recurringRunId ?? ''] as const,

  experiment: (experimentId: string | undefined | null) =>
    ['experiment', experimentId ?? null] as const,

  executionArtifact: (executionId: string, state: number) =>
    ['execution_artifact', { id: executionId, state }] as const,

  executionOutputArtifact: (executionId: string, state: number) =>
    ['execution_output_artifact', { id: executionId, state }] as const,

  executionLogs: (executionId: string | undefined, namespace: string | undefined) =>
    ['execution_logs', { executionId, namespace }] as const,

  runArtifacts: (runIds: string[]) => ['run_artifacts', { runIds }] as const,

  mlmdPackage: (runId: string) => ['mlmd_package', { id: runId }] as const,

  runDetailsV2Experiment: (runId: string, experimentId: string | null) =>
    ['RunDetailsV2_experiment', { runId, experimentId }] as const,

  pipeline: (pipelineId: string | undefined | null) => ['pipeline', pipelineId ?? ''] as const,

  pipelineVersions: (pipelineId: string | undefined | null) =>
    ['pipeline_versions', pipelineId ?? ''] as const,

  pipelineVersion: (pipelineId: string | undefined, pipelineVersionId: string | undefined) =>
    ['pipelineVersion', pipelineId ?? '', pipelineVersionId ?? ''] as const,

  v1PipelineVersionTemplate: (
    pipelineId: string | undefined,
    pipelineVersionId: string | undefined,
  ) => ['v1PipelineVersionTemplate', pipelineId ?? '', pipelineVersionId ?? ''] as const,

  runDetails: (runIds: string[]) => ['run_details', { ids: runIds }] as const,

  artifactPreview: (params: {
    value: string | undefined;
    namespace: string | undefined;
    maxbytes: number;
    maxlines: number;
  }) => ['artifact_preview', params] as const,

  contextByExecution: (executionId: string, state: number) =>
    ['context_by_execution', { id: executionId, state }] as const,

  viewconfig: (params: {
    artifactId: string | undefined;
    state: number;
    namespace: string | undefined;
  }) => ['viewconfig', params] as const,

  htmlViewerConfig: (params: {
    artifactIds: string[];
    state: number;
    namespace: string | undefined;
  }) => ['htmlViewerConfig', params] as const,

  markdownViewerConfig: (params: {
    artifactIds: string[];
    state: number;
    namespace: string | undefined;
  }) => ['markdownViewerConfig', params] as const,

  viewerConfig: (params: { artifactId: string | undefined; namespace: string | undefined }) =>
    ['viewerConfig', params] as const,
};
