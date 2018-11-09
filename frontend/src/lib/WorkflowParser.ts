/*
 * Copyright 2018 Google LLC
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

import * as dagre from 'dagre';
import { Workflow, NodeStatus, Parameter } from '../../third_party/argo-ui/argo_template';
import { statusToIcon, NodePhase } from '../pages/Status';

export enum StorageService {
  GCS = 'gcs',
  MINIO = 'minio',
}

export interface StoragePath {
  source: StorageService;
  bucket: string;
  key: string;
}

export default class WorkflowParser {
  public static createRuntimeGraph(workflow: Workflow): dagre.graphlib.Graph {
    const g = new dagre.graphlib.Graph();
    g.setGraph({});
    g.setDefaultEdgeLabel(() => ({}));

    const NODE_WIDTH = 180;
    const NODE_HEIGHT = 70;

    if (!workflow || !workflow.status || !workflow.status.nodes ||
      !workflow.metadata || !workflow.metadata.name) {
      return g;
    }

    const workflowNodes = workflow.status.nodes;
    const workflowName = workflow.metadata.name || '';

    // Ensure that the exit handler nodes are appended to the graph.
    // Uses the root node, so this needs to happen before we remove the root
    // node below.
    const onExitHandlerNodeId =
      Object.keys(workflowNodes).find((id) =>
        workflowNodes[id].name === `${workflowName}.onExit`);
    if (onExitHandlerNodeId) {
      this.getOutboundNodes(workflow, workflowName).forEach((nodeId) =>
        g.setEdge(nodeId, onExitHandlerNodeId));
    }

    // If there are multiple steps, then remove the root node that Argo creates to manage the
    // workflow to simplify the graph, otherwise leave it as otherwise there will be nothing to
    // display.
    if ((Object as any).values(workflowNodes).length > 1) {
      delete workflowNodes[workflowName];
    }

    // Create dagre graph nodes from workflow nodes.
    (Object as any).values(workflowNodes)
      .forEach((node: NodeStatus) => {
        g.setNode(node.id, {
          height: NODE_HEIGHT,
          icon: statusToIcon(workflowNodes[node.id].phase as NodePhase),
          label: node.displayName || node.id,
          width: NODE_WIDTH,
          ...node,
        });
      });

    // Connect dagre graph nodes with edges.
    Object.keys(workflowNodes)
      .forEach((nodeId) => {
        if (workflowNodes[nodeId].children) {
          workflowNodes[nodeId].children.forEach((childNodeId) =>
            g.setEdge(nodeId, childNodeId));
        }
      });

    // Add BoundaryID edges. Only add these edges to nodes that don't already have inbound edges.
    Object.keys(workflowNodes)
      .forEach((nodeId) => {
        // Many nodes have the Argo root node as a boundaryID, and we can discard these.
        if (workflowNodes[nodeId].boundaryID &&
          (!g.inEdges(nodeId) || !g.inEdges(nodeId)!.length) &&
          workflowNodes[nodeId].boundaryID !== workflowName) {
          // BoundaryIDs point from children to parents.
          g.setEdge(workflowNodes[nodeId].boundaryID, nodeId);
        }
      });

    // Remove all virtual nodes
    g.nodes().forEach((nodeId) => {
      if (workflowNodes[nodeId] && this.isVirtual(workflowNodes[nodeId])) {
        const parents = (g.inEdges(nodeId) || []).map((edge) => edge.v);
        parents.forEach((p) => g.removeEdge(p, nodeId));
        (g.outEdges(nodeId) || []).forEach((outboundEdge) => {
          g.removeEdge(outboundEdge.v, outboundEdge.w);
          // Checking if we have a parent here to handle case where root node is virtual.
          parents.forEach((p) => g.setEdge(p, outboundEdge.w));
        });
        g.removeNode(nodeId);
      }
    });

    return g;
  }

  public static getParameters(workflow?: Workflow): Parameter[] {
    if (workflow && workflow.spec && workflow.spec.arguments) {
      return workflow.spec.arguments.parameters || [];
    }
    return [];
  }

  // Makes sure the workflow object contains the node and returns its
  // inputs/outputs if any, while looking out for any missing link in the chain to
  // the node's inputs/outputs.
  public static getNodeInputOutputParams(workflow: Workflow, nodeId: string): [string[][], string[][]] {
    type paramList = string[][];
    if (!workflow || !workflow.status || !workflow.status.nodes || !workflow.status.nodes[nodeId]) {
      return [[], []];
    }

    const node = workflow.status.nodes[nodeId];
    const inputsOutputs: [paramList, paramList] = [[], []];
    if (node.inputs && node.inputs.parameters) {
      inputsOutputs[0] = node.inputs.parameters.map(p => [p.name, p.value || '']);
    }
    if (node.outputs && node.outputs.parameters) {
      inputsOutputs[1] = node.outputs.parameters.map(p => [p.name, p.value || '']);
    }
    return inputsOutputs;
  }

  // Returns a list of output paths for the given workflow Node, by looking for
  // and the Argo artifacts syntax in the outputs section.
  public static loadNodeOutputPaths(selectedWorkflowNode: NodeStatus): StoragePath[] {
    const outputPaths: StoragePath[] = [];
    if (selectedWorkflowNode && selectedWorkflowNode.outputs) {
      (selectedWorkflowNode.outputs.artifacts || [])
        .filter((a) => a.name === 'mlpipeline-ui-metadata' && !!a.s3)
        .forEach((a) =>
          outputPaths.push({
            bucket: a.s3!.bucket,
            key: a.s3!.key,
            source: StorageService.MINIO,
          })
        );
    }

    return outputPaths;
  }

  // Returns a list of output paths for the entire workflow, by searching all nodes in
  // the workflow, and parsing outputs for each.
  public static loadAllOutputPaths(workflow: Workflow): StoragePath[] {
    let outputPaths: StoragePath[] = [];
    if (workflow && workflow.status && workflow.status.nodes) {
      Object.keys(workflow.status.nodes).forEach(n =>
        outputPaths = outputPaths.concat(this.loadNodeOutputPaths(workflow.status.nodes[n])));
    }

    return outputPaths;
  }

  // Given a storage path, returns a structured object that contains the storage
  // service (currently only GCS), and bucket and key in that service.
  public static parseStoragePath(strPath: string): StoragePath {
    if (strPath.startsWith('gs://')) {
      const pathParts = strPath.substr('gs://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.GCS,
      };
    } else {
      throw new Error('Unsupported storage path: ' + strPath);
    }
  }

  // Outbound nodes are roughly those nodes which are the final step of the
  // workflow's execution. More information can be found in the NodeStatus
  // interface definition.
  public static getOutboundNodes(graph: Workflow, nodeId: string): string[] {
    let outbound: string[] = [];

    if (!graph || !graph.status || !graph.status.nodes) {
      return outbound;
    }

    const node = graph.status.nodes[nodeId];
    if (!node) {
      return outbound;
    }

    if (node.type === 'Pod') {
      return [node.id];
    }
    for (const outboundNodeID of node.outboundNodes || []) {
      const outNode = graph.status.nodes[outboundNodeID];
      if (outNode && outNode.type === 'Pod') {
        outbound.push(outboundNodeID);
      } else {
        outbound = outbound.concat(this.getOutboundNodes(graph, outboundNodeID));
      }
    }
    return outbound;
  }

  // Returns whether or not the given node is one of the intermediate nodes used
  // by Argo to orchestrate the workflow. Such nodes are not generally
  // meaningful from a user's perspective.
  public static isVirtual(node: NodeStatus): boolean {
    return (node.type === 'StepGroup' || node.type === 'DAG') && !!node.boundaryID;
  }

  // Returns a workflow-level error string if found, empty string if none
  public static getWorkflowError(workflow: Workflow): string {
    if (workflow && workflow.status && workflow.status.message && (
      workflow.status.phase === NodePhase.ERROR || workflow.status.phase === NodePhase.FAILED)) {
      return workflow.status.message;
    } else {
      return '';
    }
  }
}
