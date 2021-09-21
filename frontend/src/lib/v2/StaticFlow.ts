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
import * as jspb from 'google-protobuf';
import {
  ArrowHeadType,
  Edge,
  Elements,
  FlowElement,
  isNode,
  Node,
  Position,
} from 'react-flow-renderer';
import ExecutionNode from 'src/components/graph/ExecutionNode';
import { ComponentSpec, PipelineSpec } from 'src/generated/pipeline_spec';
import { PipelineTaskSpec } from 'src/generated/pipeline_spec/pipeline_spec_pb';

const nodeWidth = 140;
const nodeHeight = 100;

export enum NodeTypeNames {
  EXECUTION = 'EXECUTION',
}

export const NODE_TYPES = {
  [NodeTypeNames.EXECUTION]: ExecutionNode,
};

export enum TaskType {
  EXECUTOR,
  DAG,
}

interface ComponentSpecPair {
  componentRefName: string;
  componentSpec: ComponentSpec;
}

export type PipelineFlowElement = FlowElement<any>;

/**
 * Convert static IR to Reactflow compatible graph description.
 * @param spec KFP v2 Pipeline definition
 * @returns Graph visualization as Reactflow elements (nodes and edges)
 */
export function convertFlowElements(spec: PipelineSpec): Elements {
  // Find all tasks      --> nodes
  // Find all depdencies --> edges
  const root = spec.getRoot();
  if (!root) {
    throw new Error('root not found in pipeline spec.');
  }

  return buildDag(spec, root);
}

/**
 * Build single layer graph of a pipeline definition in Reactflow.
 * @param pipelineSpec Full pipeline definition
 * @param componentSpec Designated layer of a DAG/sub-DAG as part of pipelineSpec
 * @returns Graph visualization as Reactflow elements (nodes and edges)
 */
function buildDag(pipelineSpec: PipelineSpec, componentSpec: ComponentSpec): Elements {
  const dag = componentSpec.getDag();
  if (!dag) {
    throw new Error('dag not found in component spec.');
  }

  const componentsMap = pipelineSpec.getComponentsMap();
  let flowGraph: FlowElement[] = [];

  const tasksMap = dag.getTasksMap();
  console.log('tasksMap count: ' + tasksMap.getLength());

  addTaskNodes(tasksMap, componentsMap, flowGraph);
  addArtifactNodes(tasksMap, componentsMap, flowGraph);

  addTaskToArtifactEdges(tasksMap, componentsMap, flowGraph);
  addArtifactToTaskEdges(tasksMap, componentsMap, flowGraph);
  addTaskToTaskEdges(tasksMap, flowGraph);

  return buildGraphLayout(flowGraph);
}

function addTaskNodes(
  tasksMap: jspb.Map<string, PipelineTaskSpec>,
  componentsMap: jspb.Map<string, ComponentSpec>,
  flowGraph: PipelineFlowElement[],
) {
  // Add tasks as nodes to the Reactflow graph.
  tasksMap.forEach((taskSpec, taskKey) => {
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      return;
    }
    const { componentRefName, componentSpec } = componentPair;

    // Component can be either an executor or subDAG,
    // If this is executor, add the node directly.
    // If subDAG, add a node which can represent expandable graph.
    const name = taskSpec?.getTaskInfo()?.getName();
    if (componentSpec.getExecutorLabel().length > 0) {
      // executor label exists means this is a single execution node.
      const node: Node = {
        id: getTaskNodeKey(taskKey), // Assume that key of `tasks` in `dag` is unique.
        data: { label: name, taskType: TaskType.EXECUTOR },
        position: { x: 100, y: 200 },
        type: NodeTypeNames.EXECUTION,
      };
      flowGraph.push(node);
    } else if (componentSpec.hasDag()) {
      // dag exists means this is a sub-DAG instance.
      const node: Node = {
        id: getTaskNodeKey(taskKey),
        data: { label: 'DAG: ' + name, taskType: TaskType.DAG },
        position: { x: 100, y: 200 },
        // TODO(zijianjoy): This node styling is temporarily.
        style: {},
      };
      flowGraph.push(node);
    } else {
      console.warn('Component ' + componentRefName + ' has neither `executorLabel` nor `dag`');
    }
  });
}

function addArtifactNodes(
  tasksMap: jspb.Map<string, PipelineTaskSpec>,
  componentsMap: jspb.Map<string, ComponentSpec>,
  flowGraph: PipelineFlowElement[],
) {
  tasksMap.forEach((taskSpec, taskKey) => {
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      return;
    }
    const { componentSpec } = componentPair;

    // Find all artifacts --> nodes with custom style
    // Input: components -> key/value -> inputDefinitions -> artifacts -> name/key
    // Output: components -> key/value -> outputDefinitions -> artifacts -> name/key
    // Calculate Output in this function.
    const outputDefinitions = componentSpec.getOutputDefinitions();
    if (!outputDefinitions) return;
    const artifacts = outputDefinitions.getArtifactsMap();
    artifacts.forEach((artifactSpec, artifactKey) => {
      const node: Node = {
        id: getArtifactNodeKey(taskKey, artifactKey),
        data: { label: artifactSpec.getArtifactType()?.getSchemaTitle() + ': ' + artifactKey },
        position: { x: 300, y: 200 },
        // TODO(zijianjoy): This node styling is temporarily.
        style: {
          backgroundColor: '#fff59d',
          borderColor: 'transparent',
        },
      };
      flowGraph.push(node);
    });
  });
}

function addTaskToArtifactEdges(
  tasksMap: jspb.Map<string, PipelineTaskSpec>,
  componentsMap: jspb.Map<string, ComponentSpec>,
  flowGraph: PipelineFlowElement[],
) {
  // Find output and input artifacts --> edges
  // Task to Artifact: components -> key/value -> outputDefinitions -> artifacts -> key
  tasksMap.forEach((taskSpec, taskKey) => {
    const componentPair = getComponent(taskKey, taskSpec, componentsMap);
    if (componentPair === undefined) {
      return;
    }
    const { componentSpec } = componentPair;
    const outputDefinitions = componentSpec.getOutputDefinitions();
    if (!outputDefinitions) return;
    const artifacts = outputDefinitions.getArtifactsMap();
    artifacts.forEach((artifactSpec, artifactKey) => {
      const edge: Edge = {
        id: getTaskToArtifactEdgeKey(taskKey, artifactKey),
        source: getTaskNodeKey(taskKey),
        target: getArtifactNodeKey(taskKey, artifactKey),
        arrowHeadType: ArrowHeadType.ArrowClosed,
      };
      flowGraph.push(edge);
    });
  });
}

function addArtifactToTaskEdges(
  tasksMap: jspb.Map<string, PipelineTaskSpec>,
  componentsMap: jspb.Map<string, ComponentSpec>,
  flowGraph: PipelineFlowElement[],
) {
  // Artifact to Task: root -> dag -> tasks -> key/value -> inputs -> artifacts -> key/value
  //                   -> taskOutputArtifact -> outputArtifactKey+producerTask
  tasksMap.forEach((taskSpec, inputTaskKey) => {
    const inputs = taskSpec.getInputs();
    if (!inputs) {
      return;
    }
    const artifacts = inputs.getArtifactsMap();
    artifacts.forEach((artifactSpec, artifactKey) => {
      const taskOutputArtifact = artifactSpec.getTaskOutputArtifact();
      if (!taskOutputArtifact) {
        return;
      }
      const outputArtifactKey = taskOutputArtifact.getOutputArtifactKey();
      const producerTask = taskOutputArtifact.getProducerTask();
      const edge: Edge = {
        id: getArtifactToTaskEdgeKey(outputArtifactKey, inputTaskKey),
        source: getArtifactNodeKey(producerTask, outputArtifactKey),
        target: getTaskNodeKey(inputTaskKey),
        arrowHeadType: ArrowHeadType.ArrowClosed,
      };
      flowGraph.push(edge);
    });
  });
}

function addTaskToTaskEdges(
  tasksMap: jspb.Map<string, PipelineTaskSpec>,
  flowGraph: PipelineFlowElement[],
) {
  const edgeKeys = new Map<String, Edge>();
  // Input Parameters: inputs => parameters => taskOutputParameter => producerTask
  tasksMap.forEach((taskSpec, inputTaskKey) => {
    const inputs = taskSpec.getInputs();
    if (!inputs) {
      return;
    }
    const parameters = inputs.getParametersMap();
    parameters.forEach((paramSpec, paramName) => {
      const taskOutputParameter = paramSpec.getTaskOutputParameter();
      if (taskOutputParameter) {
        const producerTask = taskOutputParameter.getProducerTask();
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
    });
  });

  // DependentTasks: task => dependentTasks list
  tasksMap.forEach((taskSpec, inputTaskKey) => {
    const dependentTasks = taskSpec.getDependentTasksList();
    if (!dependentTasks) {
      return;
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
  });
}

function buildGraphLayout(flowGraph: PipelineFlowElement[]) {
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
  componentsMap: jspb.Map<string, ComponentSpec>,
): ComponentSpecPair | undefined {
  const componentRef = taskSpec.getComponentRef();
  if (componentRef === undefined) {
    console.warn('ComponentRef not found for task: ' + taskKey);
    return undefined;
  }
  const componentRefName = componentRef.getName();
  if (!componentsMap.has(componentRefName)) {
    console.warn(
      `Cannot find componentRef name ${componentRefName} from pipeline's components Map`,
    );
    return undefined;
  }
  const componentSpec = componentsMap.get(componentRefName);
  if (componentSpec === undefined) {
    console.warn('Component undefined for componentRef name: ' + componentRefName);
    return undefined;
  }
  return { componentRefName, componentSpec };
}

function getTaskNodeKey(taskKey: string) {
  return 'task.' + taskKey;
}

function getArtifactNodeKey(taskKey: string, artifactKey: string): string {
  // id is in pattern artifact.producerTaskKey.outputArtifactKey
  // Because task name and artifact name cannot contain dot in python.
  return 'artifact.' + taskKey + '.' + artifactKey;
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
