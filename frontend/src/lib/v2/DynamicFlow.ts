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
import { FlowElement } from 'react-flow-renderer';
import {
  ArtifactFlowElementData,
  ExecutionFlowElementData,
  FlowElementDataBase,
} from 'src/components/graph/Constants';
import {
  getArtifactNodeKey,
  getTaskKeyFromNodeKey,
  NodeTypeNames,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';
import { getArtifactNameFromEvent, LinkedArtifact } from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';

export const TASK_NAME_KEY = 'task_name';

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
// IMPORTANT: All the updates are in-place for PipelineFlowElement. Therefore it is no return value.

// Questions:
//    How to handle DAG state?
//    How to handle subDAG input artifacts and parameters?
//    How to handle if-condition? and show the state
//    How to handle parallel-for? and list of workers.

export function updateFlowElementsState(
  elems: PipelineFlowElement[],
  executions: Execution[],
  events: Event[],
  artifacts: Artifact[],
) {
  // IMPORTANT: PipelineFlowElement update is in-place.
  const taskNameToExecution = getTaskNameToExecution(executions);
  const executionIdToExectuion = getExectuionIdToExecution(executions);
  const artifactIdToArtifact = getArtifactIdToArtifact(artifacts);
  const artifactNodeKeyToArtifact = getArtifactNodeKeyToArtifact(
    events,
    executionIdToExectuion,
    artifactIdToArtifact,
  );

  for (let elem of elems) {
    let updatedElem = elem;
    if (NodeTypeNames.EXECUTION === elem.type) {
      const taskName = getTaskKeyFromNodeKey(elem.id);
      const execution = taskNameToExecution.get(taskName);
      (updatedElem.data as ExecutionFlowElementData).state = execution?.getLastKnownState();
    } else if (NodeTypeNames.ARTIFACT === elem.type) {
      const linkedArtifact = artifactNodeKeyToArtifact.get(elem.id);
      (updatedElem.data as ArtifactFlowElementData).state = linkedArtifact?.artifact?.getState();
    } else if (NodeTypeNames.SUB_DAG === elem.type) {
      // TODO: Update sub-dag state based on future design.
    } else {
      // Edges don't have types yet.
      // For any element that don't match the above types, copy over directly.
    }
  }
}

export function getNodeMlmdInfo(
  elem: FlowElement<FlowElementDataBase> | null,
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
  const artifactNodeKeyToArtifact = getArtifactNodeKeyToArtifact(
    events,
    executionIdToExectuion,
    artifactIdToArtifact,
  );

  if (NodeTypeNames.EXECUTION === elem.type) {
    const taskName = getTaskKeyFromNodeKey(elem.id);
    const execution = taskNameToExecution.get(taskName);
    return { execution };
  } else if (NodeTypeNames.ARTIFACT === elem.type) {
    const linkedArtifact = artifactNodeKeyToArtifact.get(elem.id);
    const executionId = linkedArtifact?.event.getExecutionId();
    const execution = executionId ? executionIdToExectuion.get(executionId) : undefined;
    return { execution, linkedArtifact };
  } else if (NodeTypeNames.SUB_DAG === elem.type) {
    // TODO: Update sub-dag state based on future design.
  } else {
    // Edges don't have types yet.
    // For any element that don't match the above types, copy over directly.
  }
  return {};
}

function getTaskNameToExecution(executions: Execution[]): Map<string, Execution> {
  const map = new Map<string, Execution>();
  for (let exec of executions) {
    const taskName = getTaskName(exec);
    if (!taskName) {
      continue;
    }
    map.set(taskName.getStringValue(), exec);
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

function getArtifactNodeKeyToArtifact(
  events: Event[],
  executionIdToExectuion: Map<number, Execution>,
  artifactIdToArtifact: Map<number, Artifact>,
): Map<string, LinkedArtifact> {
  const map = new Map<string, LinkedArtifact>();
  const outputEvents = events.filter(event => event.getType() === Event.Type.OUTPUT);
  for (let event of outputEvents) {
    const executionId = event.getExecutionId();
    const execution = executionIdToExectuion.get(executionId);
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
    map.set(key, linkedArtifact);
  }
  return map;
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
