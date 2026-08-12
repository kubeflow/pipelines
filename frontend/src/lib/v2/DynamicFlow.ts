/*
 * Copyright 2021 The Kubeflow Authors
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
import { Node } from '@xyflow/react';
import {
  InputOutputsIOArtifact,
  PipelineTaskTaskState,
  PipelineTaskTaskType,
  V2beta1PipelineTask,
} from 'src/apisv2beta1/run';
import {
  ArtifactFlowElementData,
  ExecutionFlowElementData,
  FlowElementDataBase,
  SubDagFlowElementData,
} from 'src/components/graph/Constants';
import { ComponentSpec, PipelineSpec, PipelineTaskSpec } from 'src/generated/pipeline_spec';
import {
  buildDag,
  buildGraphLayout,
  getKeysFromArtifactNodeKey,
  getIterationIdFromNodeKey,
  getTaskKeyFromNodeKey,
  getTaskNodeKey,
  isNode,
  NodeTypeNames,
  PipelineFlowElement,
  TaskType,
} from 'src/lib/v2/StaticFlow';

export interface NodeRuntimeInfo {
  task?: V2beta1PipelineTask;
  artifactGroup?: InputOutputsIOArtifact;
}

interface TaskIndex {
  rootTask?: V2beta1PipelineTask;
  tasksById: Map<string, V2beta1PipelineTask>;
  childrenByParentId: Map<string, V2beta1PipelineTask[]>;
}

interface RuntimeLayerContext {
  task?: V2beta1PipelineTask;
  iterationIndex?: number;
}

interface RuntimeFlowContext {
  taskIndex: TaskIndex;
  runtimeLayerContext: RuntimeLayerContext;
}

const ITERATION_LAYER_PATTERN = /^(.*)\.(\d+)$/;

export function convertSubDagToRuntimeFlowElements(
  spec: PipelineSpec,
  layers: string[],
  tasks: V2beta1PipelineTask[],
): PipelineFlowElement[] {
  let componentSpec = spec.root;
  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }

  const taskIndex = buildTaskIndex(tasks);
  const runtimeContext = getRuntimeLayerContext(layers, taskIndex);
  const componentsMap = spec.components;

  for (let index = 1; index < layers.length; index++) {
    if (isIterationLayer(layers[index])) {
      continue;
    }

    const tasksMap: Record<string, PipelineTaskSpec> = componentSpec.dag?.tasks || {};
    const pipelineTaskSpec: PipelineTaskSpec | undefined = tasksMap[layers[index]];
    const componentName: string | undefined = pipelineTaskSpec?.componentRef?.name;
    if (!componentName) {
      throw new Error(
        'Unable to find the component reference for task name: ' +
          (pipelineTaskSpec?.taskInfo?.name || layers[index] || 'Task name unknown'),
      );
    }
    componentSpec = componentsMap[componentName] as ComponentSpec | undefined;
    if (!componentSpec) {
      throw new Error('Component not found in pipeline spec. Component name: ' + componentName);
    }
  }

  if (
    runtimeContext.task?.type === PipelineTaskTaskType.LOOP &&
    runtimeContext.iterationIndex === undefined
  ) {
    return buildParallelForDag(runtimeContext.task, taskIndex);
  }
  return buildDag(spec, componentSpec);
}

export function updateFlowElementsState(
  layers: string[],
  elements: PipelineFlowElement[],
  tasks: V2beta1PipelineTask[],
  existingFlowContext?: RuntimeFlowContext,
): PipelineFlowElement[] {
  const flowContext = existingFlowContext || buildRuntimeFlowContext(layers, tasks);
  const { taskIndex, runtimeLayerContext: runtimeContext } = flowContext;
  if (!runtimeContext.task) {
    return elements;
  }

  if (
    runtimeContext.task.type === PipelineTaskTaskType.LOOP &&
    runtimeContext.iterationIndex === undefined
  ) {
    return elements.map((element) => {
      const updatedElement = cloneFlowElement(element);
      if (!isNode(updatedElement) || updatedElement.type !== NodeTypeNames.SUB_DAG) {
        return updatedElement;
      }
      const iterationIndex = Number(getIterationIdFromNodeKey(updatedElement.id));
      (updatedElement.data as SubDagFlowElementData).state = getIterationState(
        taskIndex.childrenByParentId.get(runtimeContext.task!.task_id || '') || [],
        iterationIndex,
      );
      return updatedElement;
    });
  }

  return elements.map((element) => {
    const updatedElement = cloneFlowElement(element);
    if (!isNode(updatedElement)) {
      return updatedElement;
    }

    const runtimeInfo = getNodeRuntimeInfo(updatedElement, tasks, layers, flowContext);
    if (updatedElement.type === NodeTypeNames.EXECUTION && runtimeInfo.task) {
      const data = updatedElement.data as ExecutionFlowElementData;
      data.state = runtimeInfo.task.state;
      data.taskId = runtimeInfo.task.task_id;
      data.label = runtimeInfo.task.display_name || runtimeInfo.task.name || data.label;
    } else if (updatedElement.type === NodeTypeNames.SUB_DAG && runtimeInfo.task) {
      const data = updatedElement.data as SubDagFlowElementData;
      data.state = runtimeInfo.task.state;
      data.taskId = runtimeInfo.task.task_id;
      data.label = runtimeInfo.task.display_name || runtimeInfo.task.name || data.label;
    } else if (updatedElement.type === NodeTypeNames.ARTIFACT && runtimeInfo.artifactGroup) {
      const data = updatedElement.data as ArtifactFlowElementData;
      data.artifactIds = runtimeInfo.artifactGroup.artifacts
        ?.map((artifact) => artifact.artifact_id)
        .filter((artifactId): artifactId is string => !!artifactId);
      data.hasArtifact = !!data.artifactIds?.length;
    }
    return updatedElement;
  });
}

export function getNodeRuntimeInfo(
  element: PipelineFlowElement | null,
  tasks: V2beta1PipelineTask[],
  layers: string[],
  existingFlowContext?: RuntimeFlowContext,
): NodeRuntimeInfo {
  if (!element || !isNode(element)) {
    return {};
  }

  const flowContext = existingFlowContext || buildRuntimeFlowContext(layers, tasks);
  const { taskIndex, runtimeLayerContext: runtimeContext } = flowContext;
  if (!runtimeContext.task) {
    return {};
  }

  if (element.type === NodeTypeNames.EXECUTION || element.type === NodeTypeNames.SUB_DAG) {
    const task = findTaskForElement(element, runtimeContext, taskIndex);
    return task ? { task } : {};
  }

  if (element.type === NodeTypeNames.ARTIFACT) {
    return getArtifactRuntimeInfo(element, runtimeContext, taskIndex);
  }
  return {};
}

export function buildRuntimeFlowContext(
  layers: string[],
  tasks: V2beta1PipelineTask[],
): RuntimeFlowContext {
  const taskIndex = buildTaskIndex(tasks);
  return { taskIndex, runtimeLayerContext: getRuntimeLayerContext(layers, taskIndex) };
}

function buildTaskIndex(tasks: V2beta1PipelineTask[]): TaskIndex {
  const tasksById = new Map<string, V2beta1PipelineTask>();
  const childrenByParentId = new Map<string, V2beta1PipelineTask[]>();
  let rootTask: V2beta1PipelineTask | undefined;

  for (const task of tasks) {
    if (task.task_id) {
      tasksById.set(task.task_id, task);
    }
    if (task.type === PipelineTaskTaskType.ROOT) {
      rootTask = task;
    }
    if (task.parent_task_id) {
      const children = childrenByParentId.get(task.parent_task_id) || [];
      children.push(task);
      childrenByParentId.set(task.parent_task_id, children);
    }
  }

  rootTask ||= tasks.find((task) => !task.parent_task_id);
  return { rootTask, tasksById, childrenByParentId };
}

function getRuntimeLayerContext(layers: string[], taskIndex: TaskIndex): RuntimeLayerContext {
  let context: RuntimeLayerContext = { task: taskIndex.rootTask };
  if (!context.task) {
    return context;
  }

  for (let index = 1; index < layers.length; index++) {
    const layer = layers[index];
    const contextTask = context.task;
    if (!contextTask) {
      return {};
    }
    const iterationMatch = layer.match(ITERATION_LAYER_PATTERN);
    if (
      iterationMatch &&
      contextTask.type === PipelineTaskTaskType.LOOP &&
      iterationMatch[1] === contextTask.name
    ) {
      context = { task: contextTask, iterationIndex: Number(iterationMatch[2]) };
      continue;
    }

    const matchedTask = getTasksUnderContext(context, taskIndex).find(
      (task) => task.name === layer,
    );
    if (!matchedTask) {
      return {};
    }
    context = { task: matchedTask };
  }
  return context;
}

function findTaskForElement(
  element: Node<FlowElementDataBase>,
  runtimeContext: RuntimeLayerContext,
  taskIndex: TaskIndex,
): V2beta1PipelineTask | undefined {
  if (element.data?.taskId) {
    const task = taskIndex.tasksById.get(element.data.taskId);
    if (task) {
      return task;
    }
  }

  const taskName = getTaskKeyFromNodeKey(element.id);
  return getTasksUnderContext(runtimeContext, taskIndex).find((task) => task.name === taskName);
}

function getArtifactRuntimeInfo(
  element: Node<FlowElementDataBase>,
  runtimeContext: RuntimeLayerContext,
  taskIndex: TaskIndex,
): NodeRuntimeInfo {
  const artifactData = element.data as ArtifactFlowElementData;
  const [taskName, artifactKey] = getKeysFromArtifactNodeKey(element.id);

  if (!taskName) {
    const artifactGroup = runtimeContext.task?.inputs?.artifacts?.find(
      (group) => group.artifact_key === artifactKey,
    );
    return artifactGroup ? { task: runtimeContext.task, artifactGroup } : {};
  }

  const task = getTasksUnderContext(runtimeContext, taskIndex).find(
    (candidate) => candidate.name === taskName,
  );
  if (!task) {
    return {};
  }

  let artifactGroup = task.outputs?.artifacts?.find((group) => group.artifact_key === artifactKey);
  if (!artifactGroup && artifactData.producerSubtask && artifactData.outputArtifactKey) {
    const producerTask = getTasksUnderContext({ task }, taskIndex).find(
      (candidate) => candidate.name === artifactData.producerSubtask,
    );
    artifactGroup = producerTask?.outputs?.artifacts?.find(
      (group) => group.artifact_key === artifactData.outputArtifactKey,
    );
    return artifactGroup ? { task: producerTask, artifactGroup } : { task };
  }
  return artifactGroup ? { task, artifactGroup } : { task };
}

function getTasksUnderContext(
  runtimeContext: RuntimeLayerContext,
  taskIndex: TaskIndex,
): V2beta1PipelineTask[] {
  const children = taskIndex.childrenByParentId.get(runtimeContext.task?.task_id || '') || [];
  if (runtimeContext.iterationIndex === undefined) {
    return children;
  }
  return children.filter(
    (task) => Number(task.type_attributes?.iteration_index) === runtimeContext.iterationIndex,
  );
}

function buildParallelForDag(loopTask: V2beta1PipelineTask, taskIndex: TaskIndex) {
  const flowGraph: PipelineFlowElement[] = [];
  const iterationCount = Number(loopTask.type_attributes?.iteration_count || 0);
  const children = taskIndex.childrenByParentId.get(loopTask.task_id || '') || [];
  for (let index = 0; index < iterationCount; index++) {
    const iterationNodeName = `${loopTask.name}.${index}`;
    const node: Node<FlowElementDataBase> = {
      id: getTaskNodeKey(iterationNodeName),
      data: {
        label: iterationNodeName,
        state: getIterationState(children, index),
        taskType: TaskType.DAG,
      },
      position: { x: 100, y: 200 },
      type: NodeTypeNames.SUB_DAG,
    };
    flowGraph.push(node);
  }
  return buildGraphLayout(flowGraph);
}

function getIterationState(
  childTasks: V2beta1PipelineTask[],
  iterationIndex: number,
): PipelineTaskTaskState | undefined {
  const states = childTasks
    .filter((task) => Number(task.type_attributes?.iteration_index) === iterationIndex)
    .map((task) => task.state)
    .filter((state): state is PipelineTaskTaskState => !!state);
  if (!states.length) {
    return undefined;
  }
  if (states.includes(PipelineTaskTaskState.FAILED)) {
    return PipelineTaskTaskState.FAILED;
  }
  if (states.includes(PipelineTaskTaskState.RUNNING)) {
    return PipelineTaskTaskState.RUNNING;
  }
  if (states.every((state) => state === PipelineTaskTaskState.SKIPPED)) {
    return PipelineTaskTaskState.SKIPPED;
  }
  if (states.every((state) => state === PipelineTaskTaskState.CACHED)) {
    return PipelineTaskTaskState.CACHED;
  }
  if (
    states.every(
      (state) =>
        state === PipelineTaskTaskState.SUCCEEDED ||
        state === PipelineTaskTaskState.CACHED ||
        state === PipelineTaskTaskState.SKIPPED,
    )
  ) {
    return PipelineTaskTaskState.SUCCEEDED;
  }
  return PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED;
}

function isIterationLayer(layer: string): boolean {
  return ITERATION_LAYER_PATTERN.test(layer);
}

function cloneFlowElement(element: PipelineFlowElement): PipelineFlowElement {
  if (isNode(element)) {
    const {
      data,
      dragging: _dragging,
      hidden: _hidden,
      position,
      resizing: _resizing,
      selected: _selected,
      ...rest
    } = element;
    return {
      ...rest,
      data: data ? { ...data } : data,
      position: { ...position },
    };
  }
  return {
    id: element.id,
    markerEnd: element.markerEnd,
    source: element.source,
    target: element.target,
  };
}
