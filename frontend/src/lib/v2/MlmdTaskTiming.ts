// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { V2beta1RuntimeState } from 'src/apisv2beta1/run';
import { PipelineSpec } from 'src/generated/pipeline_spec';
import { ExecutionHelpers } from 'src/mlmd/MlmdUtils';
import { Execution, Value } from 'src/third_party/mlmd';
import { ITERATION_INDEX_KEY, PARENT_DAG_ID_KEY, TASK_NAME_KEY } from './DynamicFlow';
import { getTaskNodeKey } from './StaticFlow';

export interface TimelineTask {
  id: string;
  name: string;
  state?: V2beta1RuntimeState | 'CACHED';
  createdAt?: Date;
  updatedAt?: Date;
  iteration?: number;
  graphTarget?: { layers: string[]; nodeId: string; executionId: number };
}

const STATES: Partial<Record<Execution.State, TimelineTask['state']>> = {
  [Execution.State.NEW]: 'PENDING',
  [Execution.State.RUNNING]: 'RUNNING',
  [Execution.State.COMPLETE]: 'SUCCEEDED',
  [Execution.State.FAILED]: 'FAILED',
  [Execution.State.CACHED]: 'CACHED',
  // KFP's MLMD writer uses CANCELED for tasks whose condition was not triggered.
  [Execution.State.CANCELED]: 'SKIPPED',
};

function integerProperty(execution: Execution, key: string): number | undefined {
  const value = execution.getCustomPropertiesMap().get(key);
  if (value?.getValueCase() !== Value.ValueCase.INT_VALUE) return undefined;
  const number = value.getIntValue();
  return Number.isSafeInteger(number) && number >= 0 ? number : undefined;
}

function timestamp(value: number): Date | undefined {
  return Number.isFinite(value) && value > 0 && Number.isFinite(new Date(value).getTime())
    ? new Date(value)
    : undefined;
}

/** Follow execution IDs, not names: sibling DAGs and loop iterations reuse task names. */
function executionLayers(
  execution: Execution,
  byId: Map<number, Execution>,
): { layers: string[]; iteration?: number } | undefined {
  const reversed: string[] = [];
  let nearestIteration: number | undefined;
  const seen = new Set([execution.getId()]);
  let child = execution;
  while (true) {
    const parentId = integerProperty(child, PARENT_DAG_ID_KEY);
    if (!parentId || seen.has(parentId)) return undefined;
    seen.add(parentId);
    const parent = byId.get(parentId);
    if (!parent) return undefined;
    const name = parent.getCustomPropertiesMap().get(TASK_NAME_KEY)?.getStringValue();
    if (name === '' && !integerProperty(parent, PARENT_DAG_ID_KEY)) {
      return { layers: ['root', ...reversed.reverse()], iteration: nearestIteration };
    }
    if (!name) return undefined;
    const iteration = integerProperty(parent, ITERATION_INDEX_KEY);
    nearestIteration ??= iteration;
    reversed.push(iteration === undefined ? name : `${name}.${iteration}`);
    child = parent;
  }
}

function componentIsLeaf(spec: PipelineSpec, layers: string[], taskName: string): boolean {
  let component = spec.root;
  for (const name of [...layers.slice(1), taskName]) {
    // Runtime iteration layers reuse the loop component; they are not pipeline tasks.
    if (name.includes('.')) continue;
    const reference = component?.dag?.tasks[name]?.componentRef?.name;
    if (!reference) return false;
    component = spec.components[reference];
  }
  return !!component?.executorLabel && !component.dag;
}

/** Normalize existing run-context MLMD data without adding a native task API dependency. */
export function getMlmdTimelineTasks(spec: PipelineSpec, executions: Execution[]): TimelineTask[] {
  const byId = new Map(executions.map((execution) => [execution.getId(), execution]));
  const tasks: TimelineTask[] = [];
  for (const execution of byId.values()) {
    if (!Number.isSafeInteger(execution.getId()) || execution.getId() <= 0) continue;
    const name = execution.getCustomPropertiesMap().get(TASK_NAME_KEY)?.getStringValue();
    const type = execution.getType();
    if (!name || type === 'system.DAGExecution') continue;
    const ancestry = executionLayers(execution, byId);
    const layers = ancestry?.layers;
    const knownLeaf =
      type === 'system.ContainerExecution' ||
      type === 'system.ImporterExecution' ||
      type === 'system.ImporterWorkspaceExecution';
    // Older MLMD responses may omit the type name. Only include a proven leaf, never a
    // presumed leaf based on absence of children (an unstarted/empty DAG has none).
    if (!knownLeaf && (type || !layers || !componentIsLeaf(spec, layers, name))) continue;
    const graphTarget =
      layers && componentIsLeaf(spec, layers, name)
        ? { layers, nodeId: getTaskNodeKey(name), executionId: execution.getId() }
        : undefined;
    tasks.push({
      id: String(execution.getId()),
      name: ExecutionHelpers.getName(execution),
      state: STATES[execution.getLastKnownState()],
      createdAt: timestamp(execution.getCreateTimeSinceEpoch()),
      updatedAt: timestamp(execution.getLastUpdateTimeSinceEpoch()),
      iteration: ancestry?.iteration,
      graphTarget,
    });
  }
  return tasks;
}
