// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import dagre from 'dagre';
import {
  ArrowHeadType,
  Edge,
  Elements,
  FlowElement,
  isNode,
  Node,
  Position,
} from 'react-flow-renderer';
import ArtifactNode from 'src/components/graph/ArtifactNode';
import { FlowElementDataBase } from 'src/components/graph/Constants';
import ExecutionNode from 'src/components/graph/ExecutionNode';
import SubDagNode from 'src/components/graph/SubDagNode';
import { ComponentSpec, PipelineSpec, PipelineTaskSpec } from 'src/generated/pipeline_spec';

const nodeWidth = 224;
const nodeHeight = 48;

export enum NodeTypeNames {
  EXECUTION = 'EXECUTION',
  ARTIFACT = 'ARTIFACT',
  SUB_DAG = 'SUB_DAG',
}

export const NODE_TYPES = {
  [NodeTypeNames.EXECUTION]: ExecutionNode,
  [NodeTypeNames.ARTIFACT]: ArtifactNode,
  [NodeTypeNames.SUB_DAG]: SubDagNode,
};

export enum TaskType {
  EXECUTOR,
  DAG,
}

interface ComponentSpecPair {
  componentRefName: string;
  componentSpec: ComponentSpec;
}

export type PipelineFlowElement = FlowElement<FlowElementDataBase>;

/**
 * Convert static IR to Reactflow compatible graph description.
 * @param spec KFP v2 Pipeline definition
 * @returns Graph visualization as Reactflow elements (nodes and edges)
 */
export function convertFlowElements(spec: PipelineSpec): Elements {
  // Find all tasks      --> nodes
  // Find all depdencies --> edges
  const root = spec.root;
  if (!root) {
    throw new Error('root not found in pipeline spec.');
  }

  return buildDag(spec, root);
}

export function convertSubDagToFlowElements(spec: PipelineSpec, layers: string[]): Elements {
  let componentSpec = spec.root;
  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }

  const componentsMap = spec.components;
  for (let index = 1; index < layers.length; index++) {
    const tasksMap:
      | {
          [key: string]: PipelineTaskSpec;
        }
      | undefined = componentSpec.dag?.tasks;
    if (!tasksMap) {
      throw new Error("Unable to get task maps from Pipeline Spec's dag.");
    }
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

  return buildDag(spec, componentSpec);
}

/**
 * Build single layer graph of a pipeline definition in Reactflow.
 * @param pipelineSpec Full pipeline definition
 * @param componentSpec Designated layer of a DAG/sub-DAG as part of pipelineSpec
 * @returns Graph visualization as Reactflow elements (nodes and edges)
 */
function buildDag(pipelineSpec: PipelineSpec, componentSpec: ComponentSpec): Elements {
  const dag = componentSpec.dag;
  if (!dag) {
    throw new Error('dag not found in component spec.');
  }

  const componentsMap = pipelineSpec.components || {};
  let flowGraph: FlowElement[] = [];

  const tasksMap = dag.tasks || {};
  console.log('tasksMap count: ' + tasksMap.length);

  addTaskNodes(tasksMap, componentsMap, flowGraph);
  addArtifactNodes(tasksMap, componentsMap, flowGraph);

  addTaskToArtifactEdges(tasksMap, componentsMap, flowGraph);
  addArtifactToTaskEdges(tasksMap, flowGraph);
  addTaskToTaskEdges(tasksMap, flowGraph);

  return buildGraphLayout(flowGraph);
}

function addTaskNodes(
  tasksMap: {
    [key: string]: PipelineTaskSpec;
  },
  componentsMap: { [key: string]: ComponentSpec },
  flowGraph: PipelineFlowElement[],
) {
  // Add tasks as nodes to the Reactflow graph.
  for (let taskKey in tasksMap) {
    const taskSpec = tasksMap[taskKey];
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      console.warn("Component for specific task doesn't exist.");
      continue;
    }
    const { componentRefName, componentSpec } = componentPair;

    // Component can be either an executor or subDAG,
    // If this is executor, add the node directly.
    // If subDAG, add a node which can represent expandable graph.
    const name = taskSpec.taskInfo?.name;
    if (!name) {
      console.warn("Task name doesn't exist.");
      continue;
    }
    if (componentSpec.executorLabel && componentSpec.executorLabel.length > 0) {
      // executor label exists means this is a single execution node.
      const node: Node<FlowElementDataBase> = {
        id: getTaskNodeKey(taskKey), // Assume that key of `tasks` in `dag` is unique.
        data: { label: name, taskType: TaskType.EXECUTOR },
        position: { x: 100, y: 200 },
        type: NodeTypeNames.EXECUTION,
      };
      flowGraph.push(node);
    } else if (componentSpec.dag) {
      // dag exists means this is a sub-DAG instance.
      const node: Node<FlowElementDataBase> = {
        id: getTaskNodeKey(taskKey),
        data: { label: 'DAG: ' + name, taskType: TaskType.DAG },
        position: { x: 100, y: 200 },
        type: NodeTypeNames.SUB_DAG,
      };
      flowGraph.push(node);
    } else {
      console.warn('Component ' + componentRefName + ' has neither `executorLabel` nor `dag`');
    }
  }
}

function addArtifactNodes(
  tasksMap: {
    [key: string]: PipelineTaskSpec;
  },
  componentsMap: { [key: string]: ComponentSpec },
  flowGraph: PipelineFlowElement[],
) {
  for (let taskKey in tasksMap) {
    const taskSpec = tasksMap[taskKey];
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      console.warn("Component for specific task doesn't exist.");
      continue;
    }
    const { componentSpec } = componentPair;

    // Find all artifacts --> nodes with custom style
    // Input: components -> key/value -> inputDefinitions -> artifacts -> name/key
    // Output: components -> key/value -> outputDefinitions -> artifacts -> name/key
    // Calculate Output in this function.
    const outputDefinitions = componentSpec.outputDefinitions;
    if (!outputDefinitions) return;
    const artifacts = outputDefinitions.artifacts;
    for (let artifactKey in artifacts) {
      const node: Node<FlowElementDataBase> = {
        id: getArtifactNodeKey(taskKey, artifactKey),
        data: { label: artifactKey },
        position: { x: 300, y: 200 },
        type: NodeTypeNames.ARTIFACT,
      };
      flowGraph.push(node);
    }
  }
}

function addTaskToArtifactEdges(
  tasksMap: {
    [key: string]: PipelineTaskSpec;
  },
  componentsMap: { [key: string]: ComponentSpec },
  flowGraph: PipelineFlowElement[],
) {
  // Find output and input artifacts --> edges
  // Task to Artifact: components -> key/value -> outputDefinitions -> artifacts -> key

  for (let taskKey in tasksMap) {
    const taskSpec = tasksMap[taskKey];
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      console.warn("Component for specific task doesn't exist.");
      continue;
    }
    const { componentSpec } = componentPair;
    const outputDefinitions = componentSpec.outputDefinitions;
    if (!outputDefinitions) return;
    const artifacts = outputDefinitions.artifacts;
    for (let artifactKey in artifacts) {
      const edge: Edge = {
        id: getTaskToArtifactEdgeKey(taskKey, artifactKey),
        source: getTaskNodeKey(taskKey),
        target: getArtifactNodeKey(taskKey, artifactKey),
        arrowHeadType: ArrowHeadType.ArrowClosed,
      };
      flowGraph.push(edge);
    }
  }
}

function addArtifactToTaskEdges(
  tasksMap: {
    [key: string]: PipelineTaskSpec;
  },
  flowGraph: PipelineFlowElement[],
) {
  // Artifact to Task: root -> dag -> tasks -> key/value -> inputs -> artifacts -> key/value
  //                   -> taskOutputArtifact -> outputArtifactKey+producerTask
  for (let inputTaskKey in tasksMap) {
    const taskSpec = tasksMap[inputTaskKey];
    const inputs = taskSpec.inputs;
    if (!inputs) {
      continue;
    }
    const artifacts = inputs.artifacts;
    for (let artifactKey in artifacts) {
      const artifactSpec = artifacts[artifactKey];
      const taskOutputArtifact = artifactSpec.taskOutputArtifact;
      if (!taskOutputArtifact) {
        continue;
      }

      const outputArtifactKey = taskOutputArtifact.outputArtifactKey;
      const producerTask = taskOutputArtifact.producerTask;
      const edge: Edge = {
        id: getArtifactToTaskEdgeKey(outputArtifactKey, inputTaskKey),
        source: getArtifactNodeKey(producerTask, outputArtifactKey),
        target: getTaskNodeKey(inputTaskKey),
        arrowHeadType: ArrowHeadType.ArrowClosed,
      };
      flowGraph.push(edge);
    }
  }
}

function addTaskToTaskEdges(
  tasksMap: {
    [key: string]: PipelineTaskSpec;
  },
  flowGraph: PipelineFlowElement[],
) {
  const edgeKeys = new Map<String, Edge>();
  // Input Parameters: inputs => parameters => taskOutputParameter => producerTask

  for (let inputTaskKey in tasksMap) {
    const taskSpec = tasksMap[inputTaskKey];
    const inputs = taskSpec.inputs;
    if (!inputs) {
      continue;
    }
    const parameters = inputs.parameters;
    for (let paramName in parameters) {
      const paramSpec = parameters[paramName];

      const taskOutputParameter = paramSpec.taskOutputParameter;
      if (taskOutputParameter) {
        const producerTask = taskOutputParameter.producerTask;
        const edgeId = getTaskToTaskEdgeKey(producerTask, inputTaskKey);
        if (edgeKeys.has(edgeId)) {
          return;
        }

        const edge: Edge = {
          // id is combination of producerTask+inputTask
          id: edgeId,
          source: getTaskNodeKey(producerTask),
          target: getTaskNodeKey(inputTaskKey),
          // TODO(zijianjoy): This node styling is temporarily.
          arrowHeadType: ArrowHeadType.ArrowClosed,
        };
        flowGraph.push(edge);
        edgeKeys.set(edgeId, edge);
      }
    }
  }

  // DependentTasks: task => dependentTasks list

  for (let inputTaskKey in tasksMap) {
    const taskSpec = tasksMap[inputTaskKey];
    const dependentTasks = taskSpec.dependentTasks;
    if (!dependentTasks) {
      continue;
    }
    dependentTasks.forEach(upStreamTaskName => {
      const edgeId = getTaskToTaskEdgeKey(upStreamTaskName, inputTaskKey);
      if (edgeKeys.has(edgeId)) {
        return;
      }

      const edge: Edge = {
        // id is combination of producerTask+inputTask
        id: edgeId,
        source: getTaskNodeKey(upStreamTaskName),
        target: getTaskNodeKey(inputTaskKey),
        // TODO(zijianjoy): This node styling is temporarily.
        arrowHeadType: ArrowHeadType.ArrowClosed,
      };
      flowGraph.push(edge);
      edgeKeys.set(edgeId, edge);
    });
  }
}

export function buildGraphLayout(flowGraph: PipelineFlowElement[]) {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: 'TB' });

  flowGraph.forEach(el => {
    if (isNode(el)) {
      dagreGraph.setNode(el.id, { width: nodeWidth, height: nodeHeight });
    } else {
      dagreGraph.setEdge(el.source, el.target);
    }
  });

  dagre.layout(dagreGraph);

  return flowGraph.map(el => {
    if (isNode(el)) {
      const nodeWithPosition = dagreGraph.node(el.id);
      el.sourcePosition = Position.Bottom;
      el.targetPosition = Position.Top;

      // unfortunately we need this little hack to pass a slightly different position
      // to notify react flow about the change. Moreover we are shifting the dagre node position
      // (anchor=center center) to the top left so it matches the react flow node anchor point (top left).
      el.position = {
        x: nodeWithPosition.x - nodeWidth / 2 + Math.random() / 1000,
        y: nodeWithPosition.y - nodeHeight / 2,
      };
    }
    return el;
  });
}

function getComponent(
  taskKey: string,
  taskSpec: PipelineTaskSpec,
  componentsMap: { [key: string]: ComponentSpec },
): ComponentSpecPair | undefined {
  const componentRef = taskSpec.componentRef;
  if (componentRef === undefined) {
    console.warn('ComponentRef not found for task: ' + taskKey);
    return undefined;
  }
  const componentRefName = componentRef.name;
  if (!(componentRefName in componentsMap)) {
    console.warn(
      `Cannot find componentRef name ${componentRefName} from pipeline's components Map`,
    );
    return undefined;
  }
  const componentSpec = componentsMap[componentRefName];
  if (componentSpec === undefined) {
    console.warn('Component undefined for componentRef name: ' + componentRefName);
    return undefined;
  }
  return { componentRefName, componentSpec };
}

const TASK_NODE_KEY_PREFIX = 'task.';
function getTaskNodeKey(taskKey: string) {
  return TASK_NODE_KEY_PREFIX + taskKey;
}

export function getTaskKeyFromNodeKey(nodeKey: string) {
  if (!isTaskNode(nodeKey)) {
    throw new Error('Task nodeKey: ' + nodeKey + " doesn't start with " + TASK_NODE_KEY_PREFIX);
  }
  return nodeKey.substr(TASK_NODE_KEY_PREFIX.length);
}

export function isTaskNode(nodeKey: string) {
  return nodeKey.startsWith(TASK_NODE_KEY_PREFIX);
}

const ARTIFACT_NODE_KEY_PREFIX = 'artifact.';
export function getArtifactNodeKey(taskKey: string, artifactKey: string): string {
  // id is in pattern artifact.producerTaskKey.outputArtifactKey
  // Because task name and artifact name cannot contain dot in python.
  return ARTIFACT_NODE_KEY_PREFIX + taskKey + '.' + artifactKey;
}

export function isArtifactNode(nodeKey: string) {
  return nodeKey.startsWith(ARTIFACT_NODE_KEY_PREFIX);
}

export function getKeysFromArtifactNodeKey(nodeKey: string) {
  const sections = nodeKey.split('.');
  if (!isArtifactNode(nodeKey)) {
    throw new Error(
      'Artifact nodeKey: ' + nodeKey + " doesn't start with " + ARTIFACT_NODE_KEY_PREFIX,
    );
  }
  if (sections.length !== 3) {
    throw new Error(
      'Artifact nodeKey: ' + nodeKey + " doesn't have format artifact.taskName.artifactName ",
    );
  }
  return [sections[1], sections[2]];
}

function getTaskToArtifactEdgeKey(taskKey: string, artifactKey: string): string {
  // id is in pattern outedge.producerTaskKey.outputArtifactKey
  return 'outedge.' + taskKey + '.' + artifactKey;
}

function getArtifactToTaskEdgeKey(outputArtifactKey: string, inputTaskKey: string): string {
  // id is in pattern of inedge.artifactKey.inputTaskKey
  return 'inedge.' + outputArtifactKey + '.' + inputTaskKey;
}

function getTaskToTaskEdgeKey(producerTask: string, inputTaskKey: string) {
  // id is in pattern of paramedge.producerTaskKey.inputTaskKey
  return 'paramedge.' + producerTask + '.' + inputTaskKey;
}
