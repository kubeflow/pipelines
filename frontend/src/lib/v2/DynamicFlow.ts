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
  ArtifactFlowElementData,
  ExecutionFlowElementData,
  FlowElementDataBase,
  SubDagFlowElementData,
} from 'src/components/graph/Constants';
import { PipelineSpec, PipelineTaskSpec } from 'src/generated/pipeline_spec';
import {
  buildDag,
  buildGraphLayout,
  getArtifactNodeKey,
  getKeysFromArtifactNodeKey,
  getIterationIdFromNodeKey,
  getTaskKeyFromNodeKey,
  isNode,
  getTaskNodeKey,
  NodeTypeNames,
  PipelineFlowElement,
  TaskType,
} from 'src/lib/v2/StaticFlow';
import { getArtifactNameFromEvent, LinkedArtifact, ExecutionHelpers } from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';

export const TASK_NAME_KEY = 'task_name';
export const PARENT_DAG_ID_KEY = 'parent_dag_id';
export const ITERATION_COUNT_KEY = 'iteration_count';
export const ITERATION_INDEX_KEY = 'iteration_index';

export function convertSubDagToRuntimeFlowElements(
  spec: PipelineSpec,
  layers: string[],
  executions: Execution[],
): PipelineFlowElement[] {
  let componentSpec = spec.root;
  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }

  const executionLayers = getExecutionLayers(layers, executions);

  let isParallelForRootDag = false;
  const componentsMap = spec.components;
  for (let index = 1; index < layers.length; index++) {
    if (canvasIsParallelForDag(executionLayers, layers)) {
      isParallelForRootDag = true;
    } else {
      isParallelForRootDag = false;
    }

    if (layers[index].indexOf('.') <= 0) {
      // Regular layer. This layer is not an iteration of ParallelFor SubDAG.

      const tasksMap = componentSpec.dag?.tasks || {};
      const pipelineTaskSpec: PipelineTaskSpec = tasksMap[layers[index]];
      const componetRef = pipelineTaskSpec.componentRef;
      const componentName = componetRef?.name;
      if (!componentName) {
        throw new Error(
          'Unable to find the component reference for task name: ' +
            pipelineTaskSpec.taskInfo?.name || 'Task name unknown',
        );
      }
      componentSpec = componentsMap[componentName];
      if (!componentSpec) {
        throw new Error('Component not found in pipeline spec. Component name: ' + componentName);
      }
    }
  }

  if (isParallelForRootDag) {
    return buildParallelForDag(executionLayers[executionLayers.length - 1]);
    // draw subdag nodes equal to the number of iteration_count
  }
  return buildDag(spec, componentSpec);
}
function canvasIsParallelForDag(executionLayers: Execution[], layers: string[]) {
  return (
    executionLayers.length === layers.length &&
    executionLayers[executionLayers.length - 1].getCustomPropertiesMap().has(ITERATION_COUNT_KEY)
  );
}

function getExecutionLayers(layers: string[], executions: Execution[]) {
  let exectuionLayers: Execution[] = [];
  if (layers.length <= 0) {
    return exectuionLayers;
  }
  const taskNameToExecution = getTaskNameToExecution(executions);

  // Get the root execution which doesn't have a task_name.
  const rootExecutions = taskNameToExecution.get('');
  if (!rootExecutions) {
    return exectuionLayers;
  }
  exectuionLayers.push(rootExecutions[0]);

  for (let index = 1; index < layers.length; index++) {
    const parentExecution = exectuionLayers[index - 1];
    const taskName = layers[index];

    let executions = taskNameToExecution.get(taskName) || [];
    // If this is an iteration of parrallelFor, remove the iteration index from layer name.
    if (taskName.indexOf('.') > 0) {
      const parallelForName = taskName.split('.')[0];
      executions = taskNameToExecution.get(parallelForName) || [];
    }

    executions = executions.filter((exec) => {
      const customProperties = exec.getCustomPropertiesMap();
      if (!customProperties.has(PARENT_DAG_ID_KEY)) {
        return false;
      }
      const parentDagId = customProperties.get(PARENT_DAG_ID_KEY)?.getIntValue();
      if (parentExecution.getId() !== parentDagId) {
        return false;
      }
      if (taskName.indexOf('.') > 0) {
        const iterationIndex = Number(taskName.split('.')[1]);
        const executionIterationIndex = customProperties.get(ITERATION_INDEX_KEY)?.getIntValue();
        return iterationIndex === executionIterationIndex;
      }
      return true;
    });
    if (executions.length <= 0) {
      break;
    }

    exectuionLayers.push(executions[0]);
  }
  return exectuionLayers;
}

function buildParallelForDag(rootDagExecution: Execution): PipelineFlowElement[] {
  let flowGraph: PipelineFlowElement[] = [];
  addIterationNodes(rootDagExecution, flowGraph);
  return buildGraphLayout(flowGraph);
}

function addIterationNodes(rootDagExecution: Execution, flowGraph: PipelineFlowElement[]) {
  const taskName = rootDagExecution.getCustomPropertiesMap().get(TASK_NAME_KEY);
  const iterationCount = rootDagExecution.getCustomPropertiesMap().get(ITERATION_COUNT_KEY);
  if (taskName === undefined || !taskName.getStringValue()) {
    console.warn("Task name for the parallelFor Execution doesn't exist.");
    return;
  }
  if (iterationCount === undefined || !iterationCount.getIntValue()) {
    console.warn("Iteration Count for the parallelFor Execution doesn't exist.");
    return;
  }

  const taskNameStr = taskName.getStringValue();
  const iterationCountInt = iterationCount.getIntValue();
  for (let index = 0; index < iterationCountInt; index++) {
    const iterationNodeName = `${taskNameStr}.${index}`;
    // One iteration is a sub-DAG instance.
    const node: Node<FlowElementDataBase> = {
      id: getTaskNodeKey(iterationNodeName),
      data: { label: iterationNodeName, taskType: TaskType.DAG },
      position: { x: 100, y: 200 },
      type: NodeTypeNames.SUB_DAG,
    };
    flowGraph.push(node);
  }
}

// 1. Get the Pipeline Run context using run ID (FOR subDAG, we need to wait for design)
// 2. Fetch all executions by context. Create Map for task_name => Execution
// 3. Fetch all Events by Context. Create Map for OUTPUT events: execution_id => Events
// 5. Fetch all Artifacts by Context.
// 6. Create Map for artifacts: artifact_id => Artifact
//    a. For each task in the flowElements, find its execution state.
//    b. For each artifact node, get its task name.
//    c. get Execution from Map, then get execution_id.
//    d. get Events from Map, then get artifact name from path.
//    e. for the Event which matches artifact name, get artifact_id.
//    f. get Artifact and update the state.

// Construct ArtifactNodeKey -> Artifact Map
//    for each OUTPUT event, get execution id and artifact id
//         get execution task_name from Execution map
//         get artifact name from Event path
//         get Artifact from Artifact map
//         set ArtifactNodeKey -> Artifact.
// Elements change to Map node key => node, edge key => edge
// For each node: (DAG execution doesn't have design yet)
//     If TASK:
//         Find exeuction from using task_name
//         Update with execution state
//     If ARTIFACT:
//         Get task_name and artifact_name
//         Get artifact from Master Map
//         Update with artifact state
//     IF SUBDAG: (Not designed)
//         similar to TASK, but needs to determine subDAG type.

// Questions:
//    How to handle DAG state?
//    How to handle subDAG input artifacts and parameters?
//    How to handle if-condition? and show the state
//    How to handle parallel-for? and list of workers.

export function updateFlowElementsState(
  layers: string[],
  elems: PipelineFlowElement[],
  executions: Execution[],
  events: Event[],
  artifacts: Artifact[],
): PipelineFlowElement[] {
  const executionLayers = getExecutionLayers(layers, executions);
  if (executionLayers.length < layers.length) {
    // This Sub DAG is not executed yet. There is no runtime information to update.
    return elems;
  }

  const taskNameToExecution = getTaskNameToExecution(executions);
  const executionIdToExectuion = getExectuionIdToExecution(executions);
  const artifactIdToArtifact = getArtifactIdToArtifact(artifacts);
  const artifactNodeKeyToArtifacts = getArtifactNodeKeyToArtifacts(
    events,
    executionIdToExectuion,
    artifactIdToArtifact,
  );
  const currentDagExecutionId = executionLayers[executionLayers.length - 1]?.getId();

  let flowGraph: PipelineFlowElement[] = [];

  if (canvasIsParallelForDag(executionLayers, layers)) {
    const parallelForDagExecution = executionLayers[executionLayers.length - 1];
    const executions = taskNameToExecution.get(
      parallelForDagExecution.getCustomPropertiesMap().get(TASK_NAME_KEY)?.getStringValue() ||
        parallelForDagExecution.getName(),
    );

    for (let elem of elems) {
      const updatedElem = cloneFlowElement(elem);
      const iterationId = Number(getIterationIdFromNodeKey(updatedElem.id));
      const matchedExecs = executions?.filter((exec) => {
        const customProperties = exec.getCustomPropertiesMap();
        const iteration_index = customProperties.get(ITERATION_INDEX_KEY)?.getIntValue();
        const parent_dag_id = customProperties.get(PARENT_DAG_ID_KEY)?.getIntValue();
        return parent_dag_id === parallelForDagExecution.getId() && iteration_index === iterationId;
      });
      if (matchedExecs && matchedExecs.length > 0) {
        (updatedElem.data as SubDagFlowElementData).state = matchedExecs[0].getLastKnownState();
      }
      flowGraph.push(updatedElem);
    }
    return flowGraph;
  }
  for (let elem of elems) {
    const updatedElem = cloneFlowElement(elem);
    if (NodeTypeNames.EXECUTION === elem.type) {
      const executions = getExecutionsUnderDAG(
        taskNameToExecution,
        getTaskLabelByPipelineFlowElement(elem),
        executionLayers,
      );
      if (executions) {
        (updatedElem.data as ExecutionFlowElementData).state = executions[0]?.getLastKnownState();
        (updatedElem.data as ExecutionFlowElementData).mlmdId = executions[0]?.getId();
        // Use ExecutionHelpers.getName() which reads display_name from MLMD custom properties
        (updatedElem.data as ExecutionFlowElementData).label = ExecutionHelpers.getName(
          executions[0],
        );
      }
    } else if (NodeTypeNames.ARTIFACT === elem.type) {
      let linkedArtifact = selectLinkedArtifactInDag(
        artifactNodeKeyToArtifacts.get(elem.id),
        executionIdToExectuion,
        currentDagExecutionId,
      );

      // Detect whether Artifact is an output of SubDAG, if so, search its source artifact.
      let artifactData = elem.data as ArtifactFlowElementData;
      if (artifactData && artifactData.outputArtifactKey && artifactData.producerSubtask) {
        // A SubDAG output artifact is produced by an inner subtask one layer deeper, so
        // scope the lookup to that child sub-DAG's execution (the artifact node's own task)
        // rather than the current DAG; otherwise sibling instances of the same sub-DAG
        // component collide on the inner (producerSubtask, artifact) key.
        const [subDagTaskName] = getKeysFromArtifactNodeKey(elem.id);
        const subDagExecutionId = getExecutionsUnderDAG(
          taskNameToExecution,
          subDagTaskName,
          executionLayers,
        )?.[0]?.getId();
        const subArtifactKey = getArtifactNodeKey(
          artifactData.producerSubtask,
          artifactData.outputArtifactKey,
        );
        linkedArtifact = selectLinkedArtifactInDag(
          artifactNodeKeyToArtifacts.get(subArtifactKey),
          executionIdToExectuion,
          subDagExecutionId,
        );
      }

      (updatedElem.data as ArtifactFlowElementData).state = linkedArtifact?.artifact?.getState();
      (updatedElem.data as ArtifactFlowElementData).mlmdId = linkedArtifact?.artifact?.getId();
      (updatedElem.data as ArtifactFlowElementData).producerExecutionId =
        linkedArtifact?.event.getExecutionId();
    } else if (NodeTypeNames.SUB_DAG === elem.type) {
      // TODO: Update sub-dag state based on future design.
      const executions = getExecutionsUnderDAG(
        taskNameToExecution,
        getTaskLabelByPipelineFlowElement(elem),
        executionLayers,
      );
      if (executions) {
        (updatedElem.data as SubDagFlowElementData).state = executions[0]?.getLastKnownState();
        (updatedElem.data as SubDagFlowElementData).mlmdId = executions[0]?.getId();
      }
    }
    flowGraph.push(updatedElem);
  }
  return flowGraph;
}

function cloneFlowElement(elem: PipelineFlowElement): PipelineFlowElement {
  if (isNode(elem)) {
    const {
      data,
      dragging: _dragging,
      hidden: _hidden,
      position,
      resizing: _resizing,
      selected: _selected,
      ...rest
    } = elem;

    return {
      ...rest,
      data: data ? { ...data } : data,
      position: { ...position },
    };
  }

  return {
    id: elem.id,
    markerEnd: elem.markerEnd,
    source: elem.source,
    target: elem.target,
  };
}

function getTaskLabelByPipelineFlowElement(elem: PipelineFlowElement) {
  // Always use the original task name from the node ID for MLMD data lookups
  return getTaskKeyFromNodeKey(elem.id);
}

function getExecutionsUnderDAG(
  taskNameToExecution: Map<string, Execution[]>,
  taskName: string,
  executionLayers: Execution[],
) {
  return taskNameToExecution.get(taskName)?.filter((exec) => {
    return (
      exec.getCustomPropertiesMap().get(PARENT_DAG_ID_KEY)?.getIntValue() ===
      executionLayers[executionLayers.length - 1].getId()
    );
  });
}

export function getNodeMlmdInfo(
  elem: PipelineFlowElement | null,
  executions: Execution[],
  events: Event[],
  artifacts: Artifact[],
): NodeMlmdInfo {
  if (!elem) {
    return {};
  }
  const taskNameToExecution = getTaskNameToExecution(executions);
  const executionIdToExectuion = getExectuionIdToExecution(executions);
  const artifactIdToArtifact = getArtifactIdToArtifact(artifacts);

  if (NodeTypeNames.EXECUTION === elem.type) {
    const taskLabel = getTaskLabelByPipelineFlowElement(elem);
    const executions = taskNameToExecution
      .get(taskLabel)
      ?.filter((exec) => exec.getId() === elem.data?.mlmdId);
    return executions ? { execution: executions[0] } : {};
  } else if (NodeTypeNames.ARTIFACT === elem.type) {
    // updateFlowElementsState resolved this node's artifact for the DAG being viewed and
    // stamped the producing artifact id + execution id onto the node, the same way
    // execution nodes carry their resolved mlmdId. Resolving by those ids (rather than by
    // task_name + artifact_name) avoids a namesake artifact from a sibling sub-DAG, and the
    // execution id disambiguates an artifact carrying more than one OUTPUT event within a
    // run (e.g. a same-run cache hit republishes the cached artifact under a new execution).
    const artifactData = elem.data as ArtifactFlowElementData | undefined;
    const artifactId = artifactData?.mlmdId;
    const producerExecutionId = artifactData?.producerExecutionId;
    if (typeof artifactId !== 'number' || typeof producerExecutionId !== 'number') {
      return {};
    }
    const artifact = artifactIdToArtifact.get(artifactId);
    const outputEvent = events.find(
      (event) =>
        event.getType() === Event.Type.OUTPUT &&
        event.getArtifactId() === artifactId &&
        event.getExecutionId() === producerExecutionId,
    );
    if (!artifact || !outputEvent) {
      return {};
    }
    const linkedArtifact: LinkedArtifact = { event: outputEvent, artifact };
    const execution = executionIdToExectuion.get(producerExecutionId);
    return { execution, linkedArtifact };
  } else if (NodeTypeNames.SUB_DAG === elem.type) {
    // TODO: Update sub-dag state based on future design.
    const taskLabel = getTaskLabelByPipelineFlowElement(elem);
    const executions = taskNameToExecution
      .get(taskLabel)
      ?.filter((exec) => exec.getId() === elem.data?.mlmdId);
    return executions ? { execution: executions[0] } : {};
  }
  return {};
}

function getTaskNameToExecution(executions: Execution[]): Map<string, Execution[]> {
  const map = new Map<string, Execution[]>();
  for (let exec of executions) {
    const taskName = getTaskName(exec);
    if (!taskName) {
      continue;
    }
    const taskNameStr = taskName.getStringValue();
    const execs = map.get(taskNameStr);
    if (execs) {
      execs.push(exec);
    } else {
      map.set(taskNameStr, [exec]);
    }
  }
  return map;
}

function getExectuionIdToExecution(executions: Execution[]): Map<number, Execution> {
  const map = new Map<number, Execution>();
  for (let exec of executions) {
    map.set(exec.getId(), exec);
  }
  return map;
}

function getArtifactIdToArtifact(artifacts: Artifact[]): Map<number, Artifact> {
  const map = new Map<number, Artifact>();
  for (let artifact of artifacts) {
    map.set(artifact.getId(), artifact);
  }
  return map;
}

// A single (task_name, artifact_name) pair does not uniquely identify an artifact: the
// same component can run in sibling sub-DAGs (e.g. the same reporting task in two
// pipelines that differ only by an input parameter), producing several executions with
// the same task_name and the same output artifact name. Collecting every candidate here
// lets callers disambiguate by the producing execution's DAG instead of silently keeping
// whichever OUTPUT event happened to be processed last.
function getArtifactNodeKeyToArtifacts(
  events: Event[],
  executionIdToExecution: Map<number, Execution>,
  artifactIdToArtifact: Map<number, Artifact>,
): Map<string, LinkedArtifact[]> {
  const artifactsByNodeKey = new Map<string, LinkedArtifact[]>();
  const outputEvents = events.filter((event) => event.getType() === Event.Type.OUTPUT);
  for (let event of outputEvents) {
    const executionId = event.getExecutionId();
    const execution = executionIdToExecution.get(executionId);
    if (!execution) {
      console.warn("Execution doesn't exist for ID " + executionId);
      continue;
    }
    const taskName = getTaskName(execution);
    if (!taskName) {
      continue;
    }
    const artifactId = event.getArtifactId();
    const artifact = artifactIdToArtifact.get(artifactId);
    if (!artifact) {
      console.warn("Artifact doesn't exist for ID " + artifactId);
      continue;
    }
    const artifactName = getArtifactNameFromEvent(event);
    if (!artifactName) {
      console.warn("Artifact name doesn't exist in Event. Artifact ID " + artifactId);
      continue;
    }
    const linkedArtifact: LinkedArtifact = { event, artifact };
    const key = getArtifactNodeKey(taskName.getStringValue(), artifactName);
    const linkedArtifacts = artifactsByNodeKey.get(key);
    if (linkedArtifacts) {
      linkedArtifacts.push(linkedArtifact);
    } else {
      artifactsByNodeKey.set(key, [linkedArtifact]);
    }
  }
  return artifactsByNodeKey;
}

// Several executions can share a (task_name, artifact_name) key across sibling sub-DAGs,
// so pick the artifact whose producing execution is a direct child of the given DAG
// execution. Falling back to the first candidate keeps a best-effort result when no
// candidate matches (e.g. a ParallelFor output produced under a per-iteration execution).
function selectLinkedArtifactInDag(
  candidateLinkedArtifacts: LinkedArtifact[] | undefined,
  executionIdToExecution: Map<number, Execution>,
  dagExecutionId: number | undefined,
): LinkedArtifact | undefined {
  if (!candidateLinkedArtifacts || candidateLinkedArtifacts.length === 0) {
    return undefined;
  }
  if (candidateLinkedArtifacts.length === 1 || dagExecutionId === undefined) {
    return candidateLinkedArtifacts[0];
  }
  const artifactInDag = candidateLinkedArtifacts.find((linkedArtifact) => {
    const producingExecution = executionIdToExecution.get(linkedArtifact.event.getExecutionId());
    return (
      producingExecution?.getCustomPropertiesMap().get(PARENT_DAG_ID_KEY)?.getIntValue() ===
      dagExecutionId
    );
  });
  return artifactInDag ?? candidateLinkedArtifacts[0];
}

function getTaskName(exec: Execution): Value | undefined {
  const customProperties = exec.getCustomPropertiesMap();
  if (!customProperties.has(TASK_NAME_KEY)) {
    console.warn("task_name key doesn't exist for custom properties of Execution " + exec.getId());
    return undefined;
  }
  const taskName = customProperties.get(TASK_NAME_KEY);
  if (!taskName) {
    console.warn(
      "task_name value doesn't exist for custom properties of Execution " + exec.getId(),
    );
    return undefined;
  }
  return taskName;
}
