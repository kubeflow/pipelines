// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { PipelineSpec } from 'src/generated/pipeline_spec';
import { Execution, Value } from 'src/third_party/mlmd';
import {
  convertSubDagToRuntimeFlowElements,
  getNodeMlmdInfo,
  updateFlowElementsState,
} from './DynamicFlow';
import { getMlmdTimelineTasks } from './MlmdTaskTiming';

const taskSpec = (component: string, name = component) => ({
  componentRef: { name: component },
  taskInfo: { name },
});
const spec = PipelineSpec.fromJSON({
  root: {
    dag: {
      tasks: {
        left: taskSpec('dag', 'left'),
        right: taskSpec('dag', 'right'),
        loop: taskSpec('dag', 'loop'),
        importer: taskSpec('importer'),
      },
    },
  },
  components: {
    dag: {
      dag: {
        tasks: { train: taskSpec('leaf', 'train'), nested: taskSpec('nested-dag', 'nested') },
      },
    },
    'nested-dag': { dag: { tasks: { train: taskSpec('leaf', 'train') } } },
    leaf: { executorLabel: 'train' },
    importer: { executorLabel: 'importer' },
  },
});
function execution(
  id: number,
  name: string,
  parent?: number,
  type = 'system.DAGExecution',
  iteration?: number,
) {
  const result = new Execution()
    .setId(id)
    .setType(type)
    .setLastKnownState(Execution.State.COMPLETE)
    .setCreateTimeSinceEpoch(1000)
    .setLastUpdateTimeSinceEpoch(3000);
  const properties = result.getCustomPropertiesMap();
  properties.set('task_name', new Value().setStringValue(name));
  if (parent !== undefined) properties.set('parent_dag_id', new Value().setIntValue(parent));
  if (iteration !== undefined)
    properties.set('iteration_index', new Value().setIntValue(iteration));
  return result;
}
const root = () => execution(1, '');

it('excludes root, empty DAGs, loops, and iterations but keeps containers and importers', () => {
  const executions = [
    root(),
    execution(2, 'left', 1),
    execution(3, 'right', 1),
    execution(4, 'train', 2, 'system.ContainerExecution'),
    execution(5, 'importer', 1, 'system.ImporterExecution'),
  ];
  expect(getMlmdTimelineTasks(spec, executions).map((task) => task.id)).toEqual(['4', '5']);
});

it('includes importers that download artifacts to a workspace', () => {
  const importer = execution(2, 'importer', 1, 'system.ImporterWorkspaceExecution');
  expect(getMlmdTimelineTasks(spec, [root(), importer])[0]).toMatchObject({
    id: '2',
    graphTarget: { layers: ['root'], executionId: 2 },
  });
});

it('falls back to the pipeline spec when older MLMD responses omit execution type names', () => {
  const executions = [
    root(),
    execution(2, 'left', 1, ''),
    execution(3, 'train', 2, ''),
    execution(4, 'importer', 1, ''),
  ];
  expect(getMlmdTimelineTasks(spec, executions).map((task) => task.id)).toEqual(['3', '4']);
});

it('resolves sibling DAGs with identical component names to their exact graph executions', () => {
  const executions = [
    root(),
    execution(2, 'left', 1),
    execution(3, 'right', 1),
    execution(4, 'train', 2, 'system.ContainerExecution'),
    execution(5, 'train', 3, 'system.ContainerExecution'),
  ];
  const tasks = getMlmdTimelineTasks(spec, executions);
  expect(tasks.map((task) => task.graphTarget?.layers)).toEqual([
    ['root', 'left'],
    ['root', 'right'],
  ]);
  for (const task of tasks) {
    const target = task.graphTarget!;
    const elements = updateFlowElementsState(
      target.layers,
      convertSubDagToRuntimeFlowElements(spec, target.layers, executions),
      executions,
      [],
      [],
    );
    const element = elements.find((node) => node.id === target.nodeId)!;
    expect(getNodeMlmdInfo(element, executions, [], []).execution?.getId()).toBe(
      target.executionId,
    );
  }
});

it('preserves individual loop iterations, including zero, with graph-compatible layer names', () => {
  const loop = execution(2, 'loop', 1);
  loop.getCustomPropertiesMap().set('iteration_count', new Value().setIntValue(2));
  const executions = [
    root(),
    loop,
    execution(3, 'loop', 2, 'system.DAGExecution', 0),
    execution(4, 'loop', 2, 'system.DAGExecution', 1),
    execution(5, 'train', 3, 'system.ContainerExecution'),
    execution(6, 'train', 4, 'system.ContainerExecution'),
  ];
  const tasks = getMlmdTimelineTasks(spec, executions);
  expect(tasks.map((task) => task.iteration)).toEqual([0, 1]);
  expect(tasks.map((task) => task.graphTarget?.layers)).toEqual([
    ['root', 'loop', 'loop.0'],
    ['root', 'loop', 'loop.1'],
  ]);
  for (const task of tasks) {
    const target = task.graphTarget!;
    const elements = updateFlowElementsState(
      target.layers,
      convertSubDagToRuntimeFlowElements(spec, target.layers, executions),
      executions,
      [],
      [],
    );
    expect(elements.find((node) => node.id === target.nodeId)?.data?.mlmdId).toBe(
      target.executionId,
    );
  }
});

it('retains the nearest iteration through a nested DAG', () => {
  const loop = execution(2, 'loop', 1);
  loop.getCustomPropertiesMap().set('iteration_count', new Value().setIntValue(1));
  const executions = [
    root(),
    loop,
    execution(3, 'loop', 2, 'system.DAGExecution', 0),
    execution(4, 'nested', 3),
    execution(5, 'train', 4, 'system.ContainerExecution'),
  ];
  const [task] = getMlmdTimelineTasks(spec, executions);
  expect(task.iteration).toBe(0);
  expect(task.graphTarget?.layers).toEqual(['root', 'loop', 'loop.0', 'nested']);
  const target = task.graphTarget!;
  const elements = updateFlowElementsState(
    target.layers,
    convertSubDagToRuntimeFlowElements(spec, target.layers, executions),
    executions,
    [],
    [],
  );
  expect(elements.find((node) => node.id === target.nodeId)?.data?.mlmdId).toBe(5);
});

it.each(['missing', 'cycle', 'unknown-task'] as const)(
  'keeps known components visible without guessing their graph location: %s',
  (reason) => {
    const parent = execution(2, 'left', reason === 'cycle' ? 2 : 1);
    const leaf = execution(
      3,
      reason === 'unknown-task' ? 'missing-task' : 'train',
      reason === 'missing' ? 99 : 2,
      'system.ContainerExecution',
    );
    expect(getMlmdTimelineTasks(spec, [root(), parent, leaf])[0]).toMatchObject({
      id: '3',
      graphTarget: undefined,
    });
  },
);

it('retains stable execution identities and does not conflate retries or same-name tasks', () => {
  const first = execution(3, 'train', 2, 'system.ContainerExecution');
  const second = execution(4, 'train', 2, 'system.ContainerExecution');
  first.setLastKnownState(Execution.State.FAILED);
  const tasks = getMlmdTimelineTasks(spec, [
    root(),
    execution(2, 'left', 1),
    first,
    second,
    second,
  ]);
  expect(tasks.map((task) => task.id)).toEqual(['3', '4']);
  expect(tasks[0].state).toBe('FAILED');
});

it.each([
  [Execution.State.NEW, 'PENDING'],
  [Execution.State.RUNNING, 'RUNNING'],
  [Execution.State.COMPLETE, 'SUCCEEDED'],
  [Execution.State.FAILED, 'FAILED'],
  [Execution.State.CACHED, 'CACHED'],
  [Execution.State.CANCELED, 'SKIPPED'],
  [Execution.State.UNKNOWN, undefined],
])('maps MLMD state %s to %s without synthesizing history', (state, expected) => {
  const leaf = execution(2, 'importer', 1, 'system.ImporterExecution').setLastKnownState(
    state as Execution.State,
  );
  const [task] = getMlmdTimelineTasks(spec, [root(), leaf]);
  expect(task.state).toBe(expected);
  expect(task).not.toHaveProperty('state_history');
});

it('uses display names and metadata timestamps without inventing missing dates', () => {
  const leaf = execution(2, 'importer', 1, 'system.ImporterExecution');
  leaf.getCustomPropertiesMap().set('display_name', new Value().setStringValue('Custom label'));
  expect(getMlmdTimelineTasks(spec, [root(), leaf])[0]).toMatchObject({
    name: 'Custom label',
    createdAt: new Date(1000),
    updatedAt: new Date(3000),
  });
  leaf.setCreateTimeSinceEpoch(0).setLastUpdateTimeSinceEpoch(-1);
  expect(getMlmdTimelineTasks(spec, [root(), leaf])[0]).toMatchObject({
    createdAt: undefined,
    updatedAt: undefined,
  });
});
