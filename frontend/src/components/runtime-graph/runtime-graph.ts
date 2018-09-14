// Copyright 2018 Google LLC
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

import 'iron-icons/av-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-icon-button/paper-icon-button.html';
import 'paper-spinner/paper-spinner.html';

import * as dagre from 'dagre';
import * as split from 'split.js';
import * as xss from 'xss';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { NODE_PHASE, NodeStatus, Workflow } from '../../../third_party/argo-ui/argo_template';
import { nodePhaseToIcon } from '../../lib/utils';
import { OutputMetadata, PlotMetadata } from '../../model/output_metadata';
import { StoragePath, StorageService } from '../../model/storage';
import { PageError } from '../page-error/page-error';

import '../data-plotter/data-plot';
import './runtime-graph.html';

interface Line {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  distance: number;
  xMid: number;
  yMid: number;
  angle: number;
  left: number;
}

interface Edge {
  from: string;
  to: string;
  lines: Line[];
}

interface DisplayNode extends dagre.Node {
  name: string;
  icon: string;
  startedAt: string;
  finishedAt: string;
}

class GraphNodeClickEvent<Model> extends MouseEvent {
  public model!: {
    node: Model,
  };
}

const NODE_WIDTH = 180;
const NODE_HEIGHT = 70;

@customElement('runtime-graph')
export class RuntimeGraph extends Polymer.Element {

  @property({ type: Number })
  protected _selectedTab = 0;

  @property({ type: Array })
  protected _displayNodes: DisplayNode[] = [];

  @property({ type: Array })
  protected _displayEdges: Edge[] = [];

  @property({ type: Array })
  protected _displayEdgeStartPoints: number[][] = [];

  @property({ type: Boolean })
  protected _loadingLogs = false;

  @property({ type: Boolean })
  protected _loadingOutputs = false;

  @property({ type: Object })
  protected _selectedNode: NodeStatus | null = null;

  @property({ type: Object })
  protected _selectedNodeOutputs: PlotMetadata[] = [];

  private _workflowNodes: { [nodeId: string]: NodeStatus } = {};

  private _nodeDetailsPaneSplitter: split.Instance | null = null;

  public ready(): void {
    super.ready();
    this._nodeDetailsPaneSplitter = split(
      [
        this.$.dag as HTMLElement,
        this.$.nodeDetails as HTMLElement
      ],
      {
        elementStyle: (dimension, size, gutterSize) => {
          return { 'flex-basis': `calc(${size}% - ${gutterSize}px)` };
        },
        gutterStyle: (dimension, gutterSize) => {
          return { 'flex-basis':  `${gutterSize}px` };
        },
        minSize: 0,
        sizes: [100, 0],
      }
    );
  }

  public refresh(graph: Workflow): void {
    this._exitNodeDetails();
    this._selectedTab = 0;
    // Ensure that we're working with empty arrays.
    this._displayEdges = [];
    this._displayNodes = [];
    this._displayEdgeStartPoints = [];

    if (!graph) {
      throw new Error('Runtime graph object is null.');
    } else if (!graph.status) {
      throw new Error('Runtime graph object has no status component.');
    } else if (!graph.status.nodes) {
      throw new Error('Runtime graph has no nodes.');
    }

    const g = new dagre.graphlib.Graph();
    g.setGraph({});
    g.setDefaultEdgeLabel(() => ({}));

    this._workflowNodes = graph.status.nodes;
    const workflowName = graph.metadata.name || '';

    // Ensure that the exit handler nodes are appended to the graph.
    // Uses the root node, so this needs to happen before we remove the root
    // node below.
    const onExitHandlerNodeId =
        Object.keys(this._workflowNodes).find((id) =>
        this._workflowNodes[id].name === `${workflowName}.onExit`);
    if (onExitHandlerNodeId) {
      this._getOutboundNodes(graph, workflowName).forEach((nodeId) =>
        g.setEdge(nodeId, onExitHandlerNodeId));
    }

    // If there are multiple steps, then remove the root node that Argo creates to manage the
    // workflow to simplify the graph, otherwise leave it as otherwise there will be nothing to
    // display.
    if (Object.values(this._workflowNodes).length > 1) {
      delete this._workflowNodes[workflowName];
    }

    // Create dagre graph nodes from workflow nodes.
    Object.values(this._workflowNodes)
      .forEach((node) => {
        g.setNode(node.id, {
          height: NODE_HEIGHT,
          label: node.displayName || node.id,
          width: NODE_WIDTH,
          ...node,
        });
      });

    // Connect dagre graph nodes with edges.
    Object.keys(this._workflowNodes)
      .forEach((nodeId) => {
        if (this._workflowNodes[nodeId].children) {
          this._workflowNodes[nodeId].children.forEach((childNodeId) =>
            g.setEdge(nodeId, childNodeId));
        }
      });

    // Add BoundaryID edges. Only add these edges to nodes that don't already have inbound edges.
    Object.keys(this._workflowNodes)
      .forEach((nodeId) => {
        // Many nodes have the Argo root node as a boundaryID, and we can discard these.
        if (this._workflowNodes[nodeId].boundaryID &&
            (!g.inEdges(nodeId) || !g.inEdges(nodeId)!.length) &&
            this._workflowNodes[nodeId].boundaryID !== workflowName) {
          // BoundaryIDs point from children to parents.
          g.setEdge(this._workflowNodes[nodeId].boundaryID, nodeId);
        }
      });

    // Remove all virtual nodes
    g.nodes().forEach((nodeId) => {
      if (this._isVirtual(this._workflowNodes[nodeId])) {
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

    dagre.layout(g);

    // Creates the array of nodes used that will be rendered and adds additional
    // metadata to them.
    this._displayNodes = g.nodes().map((id) => {
      return Object.assign(g.node(id), {
        finishedAt: this._workflowNodes[id].finishedAt,
        icon: nodePhaseToIcon(this._workflowNodes[id].phase),
        name: this._workflowNodes[id].displayName || id,
        startedAt: this._workflowNodes[id].startedAt,
      });
    });

    // Creates the lines that constitute the edges connecting the graph.
    g.edges().forEach((edgeInfo) => {
      const edge = g.edge(edgeInfo);
      const lines: Line[] = [];
      if (edge.points.length > 1) {
        for (let i = 1; i < edge.points.length; i++) {
          const x1 = edge.points[i - 1].x;
          const y1 = edge.points[i - 1].y;
          const x2 = edge.points[i].x;
          const y2 = edge.points[i].y;
          const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2));
          const xMid = (x1 + x2) / 2;
          const yMid = (y1 + y2) / 2;
          const angle = Math.atan2(y1 - y2, x1 - x2) * 180 / Math.PI;
          const left = xMid - (distance / 2);
          lines.push({ x1, y1, x2, y2, distance, xMid, yMid, angle, left });

          // Store the first point of the edge to draw the edge start circle
          if (i === 1) {
            this.push('_displayEdgeStartPoints', [x1, y1]);
          }
        }
      }
      this.push('_displayEdges', { from: edgeInfo.v, to: edgeInfo.w, lines });
    });
  }

  protected _exitNodeDetails(): void {
    this._nodeDetailsPaneSplitter!.setSizes([100, 0]);
    this.$.nodeDetails.classList.remove('visible');
    this.shadowRoot!.querySelector('.gutter')!.setAttribute('hidden', 'true');
    this._unselectAllNodes();
  }

  protected _formatDateString(date: Date): string {
    return Utils.formatDateString(date);
  }

  protected async _nodeClicked(e: GraphNodeClickEvent<any>): Promise<void> {
    // Unhide resize element and open side panel to last-used size if present, otherwise use 50% of
    // width.
    this.shadowRoot!.querySelector('.gutter')!.removeAttribute('hidden');
    const [ left, right ] = this._nodeDetailsPaneSplitter!.getSizes();
    if (right > 0) {
      this._nodeDetailsPaneSplitter!.setSizes([ left, right ]);
    } else {
      this._nodeDetailsPaneSplitter!.setSizes([ 50, 50 ]);
    }
    this.$.nodeDetails.classList.add('visible');

    this._selectedNode = e.model.node;

    if (!this._selectedNode) {
      return;
    }

    const podErrorEl = this.$.podError as PageError;
    podErrorEl.style.display = 'none';
    if ((this._selectedNode.phase === NODE_PHASE.FAILED ||
        this._selectedNode.phase === NODE_PHASE.ERROR) &&
        !!this._selectedNode.message) {
      podErrorEl.style.display = 'block';
      podErrorEl.error = 'There were some errors while scheduling this pod';
      podErrorEl.details = this._selectedNode.message;
    }

    const logsContainer = this.$.logsContainer as HTMLPreElement;
    logsContainer.innerText = '';

    const errorEl = this.$.logsError as PageError;
    errorEl.style.display = 'none';
    logsContainer.style.display = 'none';

    // Reset each time a new node is clicked
    this._selectedNodeOutputs = [];

    // Apply 'selected' CSS class to just this node
    this._unselectAllNodes();
    const root = this.shadowRoot as ShadowRoot;
    const selectedNode = root.querySelector('#node_' + this._selectedNode.id);

    if (selectedNode) {
      selectedNode.classList.add('selected');
    } else {
      // This should never happen
      Utils.showDialog('Error', 'Cannot find clicked node with id: ' + this._selectedNode.id);
      Utils.log.verbose('Cannot find clicked node with id ' + this._selectedNode.id);
    }

    // Load outputs
    this._loadNodeOutputs(this._workflowNodes[this._selectedNode.id]);

    // Load logs
    try {
      this._loadingLogs = true;
      const logs = await Apis.getPodLogs(this._selectedNode.id);
      logsContainer.style.display = 'block';
      // Linkify URLs starting with http:// or https://
      const urlPattern = /(\b(https?):\/\/[-A-Z0-9+&@#\/%?=~_|!:,.;]*[-A-Z0-9+&@#\/%=~_|])/gim;
      let lastMatch = 0;
      let match = urlPattern.exec(logs);
      while (match) {
        // Append all text before URL match
        const textSpan = document.createElement('span');
        textSpan.innerText = logs.substring(lastMatch, match.index);
        logsContainer.appendChild(textSpan);

        // Append URL via an anchor element
        const anchor = document.createElement('a');
        anchor.setAttribute('href', xss(match[0]));
        anchor.setAttribute('target', '_blank');
        anchor.innerText = match[0];
        logsContainer.appendChild(anchor);

        lastMatch = match.index + match[0].length;

        match = urlPattern.exec(logs);
      }
      // Append all text after final URL
      const remainingTextSpan = document.createElement('span');
      remainingTextSpan.innerText = logs.substring(lastMatch);
      logsContainer.appendChild(remainingTextSpan);
    } catch (err) {
      errorEl.style.display = 'block';
      errorEl.showButton = false;
      errorEl.error = `Could not retrieve logs for pod: ${this._selectedNode.id}. Error:\n${err}`;
      logsContainer.style.display = 'none';
    } finally {
      this._loadingLogs = false;
    }
  }

  private async _loadNodeOutputs(selectedWorkflowNode: NodeStatus): Promise<void> {
    if (selectedWorkflowNode && selectedWorkflowNode.outputs) {
      const outputPaths: StoragePath[] = [];
      (selectedWorkflowNode.outputs.artifacts || [])
        .filter((a) => a.name === 'mlpipeline-ui-metadata' && !!a.s3)
        .forEach((a) =>
          outputPaths.push({
            bucket: a.s3!.bucket,
            key: a.s3!.key,
            source: StorageService.MINIO,
          })
        );
      try {
        this._loadingOutputs = true;
        const outputMetadata: PlotMetadata[] = [];
        await Promise.all(outputPaths.map(async (path) => {
          const metadataFile = await Apis.readFile(path);
          if (metadataFile) {
            try {
              outputMetadata.push(...(JSON.parse(metadataFile) as OutputMetadata).outputs);
            } catch (e) {
              Utils.log.error('Could not parse metadata file for path: ' + JSON.stringify(path));
            }
          }
        }));
        this._selectedNodeOutputs =
            outputMetadata.sort((metadata1, metadata2) => {
              return metadata1.source < metadata2.source ? -1 : 1;
            });
      } catch (err) {
        Utils.log.error('Error loading run outputs:', err.message);
      } finally {
        this._loadingOutputs = false;
      }
    }
  }

  private _unselectAllNodes(): void {
    const root = this.shadowRoot as ShadowRoot;
    root.querySelectorAll('.job-node').forEach((node) => node.classList.remove('selected'));
  }

  // Outbound nodes are roughly those nodes which are the final step of the
  // workflow's execution. More information can be found in the NodeStatus
  // interface definition.
  private _getOutboundNodes(graph: Workflow, nodeId: string): string[] {
    const node = graph.status.nodes[nodeId];
    if (node.type === 'Pod') {
      return [node.id];
    }
    let outbound = Array<string>();
    for (const outboundNodeID of node.outboundNodes || []) {
      const outNode = graph.status.nodes[outboundNodeID];
      if (outNode.type === 'Pod') {
        outbound.push(outboundNodeID);
      } else {
        outbound = outbound.concat(this._getOutboundNodes(graph, outboundNodeID));
      }
    }
    return outbound;
  }

  // Returns whether or not the given node is one of the intermediate nodes used
  // by Argo to orchestrate the workflow. Such nodes are not generally
  // meaningful from a user's perspective.
  private _isVirtual(node: NodeStatus): boolean {
    return (node.type === 'StepGroup' || node.type === 'DAG') && !!node.boundaryID;
  }
}
