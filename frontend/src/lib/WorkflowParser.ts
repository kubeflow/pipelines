/*
 * Copyright 2018-2019 Google LLC
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

import MoreIcon from '@material-ui/icons/MoreHoriz';
import * as dagre from 'dagre';
import {
  NodeStatus,
  Parameter,
  S3Artifact,
  Workflow,
} from '../../third_party/argo-ui/argo_template';
import IconWithTooltip from '../atoms/IconWithTooltip';
import { color } from '../Css';
import { statusToIcon } from '../pages/Status';
import { Constants } from './Constants';
import { KeyValue } from './StaticGraphParser';
import { hasFinished, NodePhase, statusToBgColor, parseNodePhase } from './StatusUtils';
import { parseTaskDisplayNameByNodeId } from './ParserUtils';
import { isS3Endpoint } from './AwsHelper';
import { TFunction } from 'i18next';

export enum StorageService {
  GCS = 'gcs',
  HTTP = 'http',
  HTTPS = 'https',
  MINIO = 'minio',
  S3 = 's3',
  VOLUME = 'volume',
}

export interface StoragePath {
  source: StorageService;
  bucket: string;
  key: string;
}

export default class WorkflowParser {
  public static createRuntimeGraph(t: TFunction, workflow: Workflow): dagre.graphlib.Graph {
    const g = new dagre.graphlib.Graph();
    g.setGraph({});
    g.setDefaultEdgeLabel(() => ({}));

    const PLACEHOLDER_NODE_DIMENSION = 28;

    if (
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.metadata ||
      !workflow.metadata.name
    ) {
      return g;
    }

    const workflowNodes = workflow.status.nodes;
    const workflowName = workflow.metadata.name || '';

    // Ensure that the exit handler nodes are appended to the graph.
    // Uses the root node, so this needs to happen before we remove the root
    // node below.
    const onExitHandlerNodeId = Object.keys(workflowNodes).find(
      id => workflowNodes[id].name === `${workflowName}.onExit`,
    );
    if (onExitHandlerNodeId) {
      this.getOutboundNodes(workflow, workflowName).forEach(nodeId =>
        g.setEdge(nodeId, onExitHandlerNodeId),
      );
    }

    // If there are multiple steps, then remove the root node that Argo creates to manage the
    // workflow to simplify the graph, otherwise leave it as otherwise there will be nothing to
    // display.
    if ((Object as any).values(workflowNodes).length > 1) {
      delete workflowNodes[workflowName];
    }

    const runningNodeSuffix = '-running-placeholder';

    // Create dagre graph nodes from workflow nodes.
    (Object as any).values(workflowNodes).forEach((node: NodeStatus) => {
      const nodeLabel = parseTaskDisplayNameByNodeId(node.id, workflow);
      g.setNode(node.id, {
        height: Constants.NODE_HEIGHT,
        icon: statusToIcon(t, parseNodePhase(node), node.startedAt, node.finishedAt, node.message),
        label: nodeLabel,
        statusColoring: statusToBgColor(node.phase as NodePhase, node.message),
        width: Constants.NODE_WIDTH,
        ...node,
      });

      if (!hasFinished(node.phase as NodePhase) && !this.isVirtual(node)) {
        g.setNode(node.id + runningNodeSuffix, {
          height: PLACEHOLDER_NODE_DIMENSION,
          icon: IconWithTooltip({
            Icon: MoreIcon,
            height: 24,
            iconColor: color.weak,
            tooltip: 'More nodes may appear here',
            width: 24,
          }),
          isPlaceholder: true,
          width: PLACEHOLDER_NODE_DIMENSION,
        });
        g.setEdge(node.id, node.id + runningNodeSuffix, { color: color.weak, isPlaceholder: true });
      }
    });

    // Connect dagre graph nodes with edges.
    Object.keys(workflowNodes).forEach(nodeId => {
      if (workflowNodes[nodeId].children) {
        workflowNodes[nodeId].children.forEach(childNodeId => {
          if (workflowNodes[childNodeId]) {
            g.setEdge(nodeId, childNodeId);
          }
        });
      }
    });

    // Add BoundaryID edges. Only add these edges to nodes that don't already have inbound edges.
    Object.keys(workflowNodes).forEach(nodeId => {
      // Many nodes have the Argo root node as a boundaryID, and we can discard these.
      if (
        workflowNodes[nodeId].boundaryID &&
        workflowNodes[workflowNodes[nodeId].boundaryID] &&
        (!g.inEdges(nodeId) || !g.inEdges(nodeId)!.length) &&
        workflowNodes[nodeId].boundaryID !== workflowName
      ) {
        // BoundaryIDs point from children to parents.
        g.setEdge(workflowNodes[nodeId].boundaryID, nodeId);
      }
    });

    // Remove all virtual nodes
    g.nodes().forEach(nodeId => {
      if (workflowNodes[nodeId] && this.isVirtual(workflowNodes[nodeId])) {
        const parents = (g.inEdges(nodeId) || []).map(edge => edge.v);
        parents.forEach(p => g.removeEdge(p, nodeId));
        (g.outEdges(nodeId) || []).forEach(outboundEdge => {
          g.removeEdge(outboundEdge.v, outboundEdge.w);
          // Checking if we have a parent here to handle case where root node is virtual.
          parents.forEach(p => g.setEdge(p, outboundEdge.w));
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
  public static getNodeInputOutputParams(
    workflow?: Workflow,
    nodeId?: string,
  ): Record<'inputParams' | 'outputParams', Array<KeyValue<string>>> {
    type ParamList = Array<KeyValue<string>>;
    let inputParams: ParamList = [];
    let outputParams: ParamList = [];
    if (
      !nodeId ||
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId]
    ) {
      return { inputParams, outputParams };
    }

    const { inputs, outputs } = workflow.status.nodes[nodeId];
    if (!!inputs && !!inputs.parameters) {
      inputParams = inputs.parameters.map(p => [p.name, p.value || '']);
    }
    if (!!outputs && !!outputs.parameters) {
      outputParams = outputs.parameters.map(p => [p.name, p.value || '']);
    }
    return { inputParams, outputParams };
  }

  // Makes sure the workflow object contains the node and returns its
  // inputs/outputs artifacts if any, while looking out for any missing link in the chain to
  // the node's inputs/outputs.
  public static getNodeInputOutputArtifacts(
    workflow?: Workflow,
    nodeId?: string,
  ): Record<'inputArtifacts' | 'outputArtifacts', Array<KeyValue<S3Artifact>>> {
    type ParamList = Array<KeyValue<S3Artifact>>;
    let inputArtifacts: ParamList = [];
    let outputArtifacts: ParamList = [];
    if (
      !nodeId ||
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId]
    ) {
      return { inputArtifacts, outputArtifacts };
    }

    const { inputs, outputs } = workflow.status.nodes[nodeId];
    if (!!inputs && !!inputs.artifacts) {
      inputArtifacts = inputs.artifacts.map(({ name, s3 }) => [name, s3]);
    }
    if (!!outputs && !!outputs.artifacts) {
      outputArtifacts = outputs.artifacts.map(({ name, s3 }) => [name, s3]);
    }
    return { inputArtifacts, outputArtifacts };
  }

  // Makes sure the workflow object contains the node and returns its
  // volume mounts if any.
  public static getNodeVolumeMounts(workflow: Workflow, nodeId: string): Array<KeyValue<string>> {
    if (
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId] ||
      !workflow.spec ||
      !workflow.spec.templates
    ) {
      return [];
    }

    const node = workflow.status.nodes[nodeId];
    const tmpl = workflow.spec.templates.find(t => !!t && !!t.name && t.name === node.templateName);
    let volumeMounts: Array<KeyValue<string>> = [];
    if (tmpl && tmpl.container && tmpl.container.volumeMounts) {
      volumeMounts = tmpl.container.volumeMounts.map(v => [v.mountPath, v.name]);
    }
    return volumeMounts;
  }

  // Makes sure the workflow object contains the node and returns its
  // action and manifest.
  public static getNodeManifest(workflow: Workflow, nodeId: string): Array<KeyValue<string>> {
    if (
      !workflow ||
      !workflow.status ||
      !workflow.status.nodes ||
      !workflow.status.nodes[nodeId] ||
      !workflow.spec ||
      !workflow.spec.templates
    ) {
      return [];
    }

    const node = workflow.status.nodes[nodeId];
    const tmpl = workflow.spec.templates.find(t => !!t && !!t.name && t.name === node.templateName);
    let manifest: Array<KeyValue<string>> = [];
    if (tmpl && tmpl.resource && tmpl.resource.action && tmpl.resource.manifest) {
      manifest = [[tmpl.resource.action, tmpl.resource.manifest]];
    }
    return manifest;
  }

  // Returns a list of output paths for the given workflow Node, by looking for
  // and the Argo artifacts syntax in the outputs section.
  public static loadNodeOutputPaths(selectedWorkflowNode: NodeStatus): StoragePath[] {
    const outputPaths: StoragePath[] = [];
    if (selectedWorkflowNode && selectedWorkflowNode.outputs) {
      (selectedWorkflowNode.outputs.artifacts || [])
        .filter(a => a.name === 'mlpipeline-ui-metadata' && !!a.s3)
        .forEach(a =>
          outputPaths.push({
            bucket: a.s3!.bucket,
            key: a.s3!.key,
            source: isS3Endpoint(a.s3!.endpoint) ? StorageService.S3 : StorageService.MINIO,
          }),
        );
    }
    return outputPaths;
  }

  // Returns a list of output paths for the entire workflow, by searching all nodes in
  // the workflow, and parsing outputs for each.
  public static loadAllOutputPaths(workflow: Workflow): StoragePath[] {
    return this.loadAllOutputPathsWithStepNames(workflow).map(entry => entry.path);
  }

  // Returns a list of object mapping a step name to output path for the entire workflow,
  // by searching all nodes in the workflow, and parsing outputs for each.
  public static loadAllOutputPathsWithStepNames(
    workflow: Workflow,
  ): Array<{ stepName: string; path: StoragePath }> {
    const outputPaths: Array<{ stepName: string; path: StoragePath }> = [];
    if (workflow && workflow.status && workflow.status.nodes) {
      Object.keys(workflow.status.nodes).forEach(n =>
        this.loadNodeOutputPaths(workflow.status.nodes[n]).map(path =>
          outputPaths.push({ stepName: workflow.status.nodes[n].displayName, path }),
        ),
      );
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
    } else if (strPath.startsWith('minio://')) {
      const pathParts = strPath.substr('minio://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.MINIO,
      };
    } else if (strPath.startsWith('s3://')) {
      const pathParts = strPath.substr('s3://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.S3,
      };
    } else if (strPath.startsWith('http://')) {
      const pathParts = strPath.substr('http://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.HTTP,
      };
    } else if (strPath.startsWith('https://')) {
      const pathParts = strPath.substr('https://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.HTTPS,
      };
    } else if (strPath.startsWith('volume://')) {
      const pathParts = strPath.substr('volume://'.length).split('/');
      return {
        bucket: pathParts[0],
        key: pathParts.slice(1).join('/'),
        source: StorageService.VOLUME,
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
    return (
      (node.type === 'StepGroup' ||
        node.type === 'DAG' ||
        node.type === 'TaskGroup' ||
        node.type === 'Retry') &&
      !!node.boundaryID
    );
  }

  // Returns a workflow-level error string if found, empty string if none
  public static getWorkflowError(workflow: Workflow): string {
    if (
      workflow &&
      workflow.status &&
      workflow.status.message &&
      (workflow.status.phase === NodePhase.ERROR || workflow.status.phase === NodePhase.FAILED)
    ) {
      return workflow.status.message;
    } else {
      return '';
    }
  }
}
