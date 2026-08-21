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
import { getTaskDisplayName } from 'src/lib/v2/RunTaskUtils';
import {
  formatRuntimeIterationLayer,
  isRuntimeIterationLayer,
  parseRuntimeIterationLayer,
} from 'src/lib/v2/RuntimeLayerUtils';

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
  runCompletedSuccessfully: boolean;
  runIsTerminal: boolean;
}

export function convertSubDagToRuntimeFlowElements(
  spec: PipelineSpec,
  layers: string[],
  tasks: V2beta1PipelineTask[],
  runIsTerminal = false,
  runCompletedSuccessfully = false,
): PipelineFlowElement[] {
  let componentSpec = spec.root;
  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }

  const taskIndex = buildTaskIndex(tasks);
  const runtimeContext = getRuntimeLayerContext(layers, taskIndex);
  const componentsMap = spec.components;

  for (let index = 1; index < layers.length; index++) {
    if (isRuntimeIterationLayer(layers[index])) {
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
    const expectedTaskCount = Object.keys(componentSpec.dag?.tasks || {}).length;
    return (
      buildParallelForDag(
        runtimeContext.task,
        taskIndex,
        expectedTaskCount,
        runIsTerminal,
        runCompletedSuccessfully,
      ) || annotateExpectedTaskCount(buildDag(spec, componentSpec), expectedTaskCount)
    );
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
      const data = updatedElement.data as SubDagFlowElementData;
      data.state = getIterationState(
        taskIndex.childrenByParentId.get(runtimeContext.task!.task_id || '') || [],
        iterationIndex,
        data.expectedTaskCount,
        runtimeContext.task!.state,
        flowContext.runIsTerminal,
        flowContext.runCompletedSuccessfully,
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
    const parentCompletedSuccessfully = taskCompletedSuccessfully(runtimeContext.task?.state);
    if (updatedElement.type === NodeTypeNames.EXECUTION && runtimeInfo.task) {
      const data = updatedElement.data as ExecutionFlowElementData;
      data.state = getRuntimeTaskState(
        runtimeInfo.task.state,
        flowContext.runIsTerminal,
        flowContext.runCompletedSuccessfully || parentCompletedSuccessfully,
      );
      data.taskId = runtimeInfo.task.task_id;
      data.label = getTaskDisplayName(runtimeInfo.task, data.label);
    } else if (updatedElement.type === NodeTypeNames.SUB_DAG && runtimeInfo.task) {
      const data = updatedElement.data as SubDagFlowElementData;
      data.state = getRuntimeTaskState(
        runtimeInfo.task.state,
        flowContext.runIsTerminal,
        flowContext.runCompletedSuccessfully || parentCompletedSuccessfully,
      );
      data.taskId = runtimeInfo.task.task_id;
      data.label = getTaskDisplayName(runtimeInfo.task, data.label);
    } else if (updatedElement.type === NodeTypeNames.ARTIFACT && runtimeInfo.artifactGroup) {
      const data = updatedElement.data as ArtifactFlowElementData;
      data.hasArtifact = !!runtimeInfo.artifactGroup.artifacts?.some(
        (artifact) => artifact.artifact_id,
      );
    }
    return updatedElement;
  });
}

function taskCompletedSuccessfully(state: PipelineTaskTaskState | undefined): boolean {
  return (
    state === PipelineTaskTaskState.SUCCEEDED ||
    state === PipelineTaskTaskState.SKIPPED ||
    state === PipelineTaskTaskState.CACHED
  );
}

export function reconcileRuntimeFlowElements(
  layers: string[],
  elements: PipelineFlowElement[],
  tasks: V2beta1PipelineTask[],
  existingFlowContext?: RuntimeFlowContext,
): PipelineFlowElement[] {
  const flowContext = existingFlowContext || buildRuntimeFlowContext(layers, tasks);
  const runtimeContext = flowContext.runtimeLayerContext;
  let runtimeStructure = elements;
  if (
    runtimeContext.task?.type === PipelineTaskTaskType.LOOP &&
    runtimeContext.iterationIndex === undefined &&
    !hasParallelForStructure(elements, runtimeContext.task)
  ) {
    // A LOOP row can exist before the driver has persisted iteration_count (including driver
    // failure and non-triggered paths). Keep the declarative body visible until runtime iteration
    // structure is authoritative instead of replacing it with an empty graph.
    runtimeStructure =
      buildParallelForDag(
        runtimeContext.task,
        flowContext.taskIndex,
        getExpectedTaskCount(elements),
        flowContext.runIsTerminal,
        flowContext.runCompletedSuccessfully,
      ) || runtimeStructure;
  }

  return updateFlowElementsState(layers, runtimeStructure, tasks, flowContext);
}

function hasParallelForStructure(
  elements: PipelineFlowElement[],
  loopTask: V2beta1PipelineTask,
): boolean {
  const iterationCount = getParallelForIterationCount(loopTask);
  if (iterationCount === undefined) {
    return false;
  }
  if (elements.length !== iterationCount) {
    return false;
  }
  const expectedNodeIds = new Set(
    Array.from({ length: iterationCount }, (_, index) =>
      getTaskNodeKey(`${loopTask.name}.${index}`),
    ),
  );
  return elements.every(
    (element) =>
      isNode(element) && element.type === NodeTypeNames.SUB_DAG && expectedNodeIds.has(element.id),
  );
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
  runIsTerminal = false,
  runCompletedSuccessfully = false,
): RuntimeFlowContext {
  const taskIndex = buildTaskIndex(tasks);
  return {
    taskIndex,
    runtimeLayerContext: getRuntimeLayerContext(layers, taskIndex),
    runCompletedSuccessfully,
    runIsTerminal,
  };
}

export function getTaskRuntimeLayers(
  task: V2beta1PipelineTask,
  tasks: V2beta1PipelineTask[],
): string[] {
  const tasksById = new Map<string, V2beta1PipelineTask>();
  for (const candidate of tasks) {
    if (candidate.task_id) {
      tasksById.set(candidate.task_id, candidate);
    }
  }
  const ancestry: V2beta1PipelineTask[] = [];
  const visited = new Set<string>();
  let current: V2beta1PipelineTask | undefined = task;
  while (current) {
    ancestry.unshift(current);
    if (current.type === PipelineTaskTaskType.ROOT || !current.parent_task_id) {
      break;
    }
    if (visited.has(current.parent_task_id)) {
      break;
    }
    visited.add(current.parent_task_id);
    current = tasksById.get(current.parent_task_id);
  }

  if (ancestry[0]?.type !== PipelineTaskTaskType.ROOT && task.scope_path) {
    const scopeLayers = task.scope_path.split('.').filter(Boolean).slice(0, -1);
    return scopeLayers[0] === 'root' ? scopeLayers : ['root', ...scopeLayers];
  }

  const runtimeTasks =
    ancestry[0]?.type === PipelineTaskTaskType.ROOT ? ancestry.slice(1) : ancestry;
  const layers = ['root'];
  for (let index = 0; index < runtimeTasks.length - 1; index++) {
    const contextTask = runtimeTasks[index];
    const childTask = runtimeTasks[index + 1];
    if (!contextTask.name) {
      continue;
    }
    layers.push(contextTask.name);
    if (
      contextTask.type === PipelineTaskTaskType.LOOP &&
      childTask.type_attributes?.iteration_index !== undefined
    ) {
      layers.push(
        formatRuntimeIterationLayer(
          contextTask.name,
          Number(childTask.type_attributes.iteration_index),
        ),
      );
    }
  }
  return layers;
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
    const iterationLayer = parseRuntimeIterationLayer(layer, contextTask.name);
    if (iterationLayer && contextTask.type === PipelineTaskTaskType.LOOP) {
      context = { task: contextTask, iterationIndex: iterationLayer.iterationIndex };
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

function getParallelForIterationCount(loopTask: V2beta1PipelineTask): number | undefined {
  const value = loopTask.type_attributes?.iteration_count;
  if (value === undefined || value === '') {
    return undefined;
  }
  const count = Number(value);
  return Number.isInteger(count) && count >= 0 ? count : undefined;
}

function buildParallelForDag(
  loopTask: V2beta1PipelineTask,
  taskIndex: TaskIndex,
  expectedTaskCount?: number,
  runIsTerminal = false,
  runCompletedSuccessfully = false,
): PipelineFlowElement[] | undefined {
  const flowGraph: PipelineFlowElement[] = [];
  const iterationCount = getParallelForIterationCount(loopTask);
  if (iterationCount === undefined) {
    return undefined;
  }
  const children = taskIndex.childrenByParentId.get(loopTask.task_id || '') || [];
  for (let index = 0; index < iterationCount; index++) {
    const iterationNodeName = formatRuntimeIterationLayer(loopTask.name || '', index);
    const node: Node<FlowElementDataBase> = {
      id: getTaskNodeKey(iterationNodeName),
      data: {
        label: iterationNodeName,
        expectedTaskCount,
        state: getIterationState(
          children,
          index,
          expectedTaskCount,
          loopTask.state,
          runIsTerminal,
          runCompletedSuccessfully,
        ),
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
  expectedTaskCount?: number,
  loopState?: PipelineTaskTaskState,
  runIsTerminal = false,
  runCompletedSuccessfully = false,
): PipelineTaskTaskState | undefined {
  const iterationTasks = childTasks.filter(
    (task) => Number(task.type_attributes?.iteration_index) === iterationIndex,
  );
  const states = iterationTasks
    .map((task) => task.state)
    .filter((state): state is PipelineTaskTaskState => !!state);
  if (!states.length) {
    return undefined;
  }
  const loopCompletedSuccessfully =
    loopState === PipelineTaskTaskState.SUCCEEDED ||
    loopState === PipelineTaskTaskState.SKIPPED ||
    loopState === PipelineTaskTaskState.CACHED;
  if (
    (runCompletedSuccessfully || loopCompletedSuccessfully) &&
    states.includes(PipelineTaskTaskState.FAILED)
  ) {
    return PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED;
  }
  if (states.includes(PipelineTaskTaskState.FAILED)) {
    return PipelineTaskTaskState.FAILED;
  }
  const loopIsTerminal = runIsTerminal || loopCompletedSuccessfully;
  if (loopIsTerminal && states.includes(PipelineTaskTaskState.RUNNING)) {
    return PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED;
  }
  if (states.includes(PipelineTaskTaskState.RUNNING)) {
    return PipelineTaskTaskState.RUNNING;
  }
  const iterationIsIncomplete =
    expectedTaskCount !== undefined && iterationTasks.length < expectedTaskCount;
  if (iterationIsIncomplete) {
    if (
      !runIsTerminal &&
      (loopState === PipelineTaskTaskState.RUNNING || loopState === PipelineTaskTaskState.FAILED)
    ) {
      return PipelineTaskTaskState.RUNNING;
    }
    if (
      runIsTerminal &&
      (loopState === PipelineTaskTaskState.RUNNING || loopState === PipelineTaskTaskState.FAILED)
    ) {
      return PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED;
    }
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

function getRuntimeTaskState(
  taskState: PipelineTaskTaskState | undefined,
  runIsTerminal: boolean,
  enclosingScopeCompletedSuccessfully: boolean,
): PipelineTaskTaskState | undefined {
  return ((runIsTerminal || enclosingScopeCompletedSuccessfully) &&
    taskState === PipelineTaskTaskState.RUNNING) ||
    (enclosingScopeCompletedSuccessfully && taskState === PipelineTaskTaskState.FAILED)
    ? PipelineTaskTaskState.RUNTIME_STATE_UNSPECIFIED
    : taskState;
}

function annotateExpectedTaskCount(
  elements: PipelineFlowElement[],
  expectedTaskCount: number,
): PipelineFlowElement[] {
  return elements.map((element) =>
    isNode(element) ? { ...element, data: { ...element.data, expectedTaskCount } } : element,
  );
}

function getExpectedTaskCount(elements: PipelineFlowElement[]): number | undefined {
  for (const element of elements) {
    if (isNode(element) && typeof element.data.expectedTaskCount === 'number') {
      return element.data.expectedTaskCount;
    }
  }
  return undefined;
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
