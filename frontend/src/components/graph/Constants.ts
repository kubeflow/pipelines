// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

/** View-only icon states. Execution status remains Argo's workflow/node status. */
export enum PipelineTaskTaskState {
  RUNTIME_STATE_UNSPECIFIED = 'RUNTIME_STATE_UNSPECIFIED',
  RUNNING = 'RUNNING',
  CACHED = 'CACHED',
  FAILED = 'FAILED',
  SKIPPED = 'SKIPPED',
  SUCCEEDED = 'SUCCEEDED',
}
export type SubDagKind = 'Sub-DAG' | 'Loop' | 'Iteration' | 'Condition';
export type FlowElementDataBase = { label: string; taskId?: string; [key: string]: any };
export type SubDagTaskData = FlowElementDataBase & {
  groupKind?: SubDagKind;
  expectedTaskCount?: number;
  state?: PipelineTaskTaskState;
  lifecycleMessage?: string;
};
export type SubDagFlowElementData = SubDagTaskData & { expand: (nodeKey: string) => void };
export type ExecutionFlowElementData = FlowElementDataBase & {
  state?: PipelineTaskTaskState;
  lifecycleMessage?: string;
};
export type ArtifactFlowElementData = FlowElementDataBase & {
  hasArtifact?: boolean;
  producerSubtask?: string;
  outputArtifactKey?: string;
};
