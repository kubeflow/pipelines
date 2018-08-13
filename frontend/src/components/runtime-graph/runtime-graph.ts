import 'iron-icons/av-icons.html';
import 'iron-icons/iron-icons.html';
import 'paper-icon-button/paper-icon-button.html';
import 'paper-spinner/paper-spinner.html';

import * as dagre from 'dagre';
import * as Apis from '../../lib/apis';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { nodePhaseToIcon } from '../../lib/utils';
import { NodeStatus, Workflow as ArgoTemplate } from '../../model/argo_template';

import './runtime-graph.html';

import { PageError } from '../page-error/page-error';

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
  public model: {
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
  protected _workflowNodes: DisplayNode[] = [];

  @property({ type: Array })
  protected _workflowEdges: Edge[] = [];

  @property({ type: Array })
  protected _workflowEdgeStartPoints: number[][] = [];

  @property({ type: Boolean })
  protected _loadingLogs = false;

  @property({ type: Object })
  protected _selectedNode: NodeStatus | null = null;

  private _runtimeGraph: ArgoTemplate | null = null;

  public refresh(graph: ArgoTemplate): void {
    this._exitNodeDetails();
    this._selectedTab = 0;
    // Ensure that we're working with empty arrays.
    this._workflowEdges = [];
    this._workflowNodes = [];
    this._workflowEdgeStartPoints = [];

    this._runtimeGraph = graph;

    if (!this._runtimeGraph) {
      throw new Error('Runtime graph object is null.');
    } else if (!this._runtimeGraph.status) {
      throw new Error('Runtime graph object has no status component.');
    } else if (!this._runtimeGraph.status.nodes) {
      throw new Error('Runtime graph has no nodes.');
    }

    const g = new dagre.graphlib.Graph();
    g.setGraph({});
    g.setDefaultEdgeLabel(() => ({}));

    const workflowNodes = this._runtimeGraph.status.nodes;
    const workflowName = this._runtimeGraph.metadata.name || '';

    // Ensure that the exit handler nodes are appended to the graph.
    // Uses the root node, so this needs to happen before we remove the root
    // node below.
    const onExitHandlerNodeId =
        Object.keys(workflowNodes).find((id) =>
            workflowNodes[id].name === `${workflowName}.onExit`);
    if (onExitHandlerNodeId) {
      this._getOutboundNodes(this._runtimeGraph, workflowName).forEach((nodeId) =>
        g.setEdge(nodeId, onExitHandlerNodeId));
    }

    // If there are multiple steps, then remove the root node that Argo creates to manage the
    // workflow to simplify the graph, otherwise leave it as otherwise there will be nothing to
    // display.
    if (Object.values(workflowNodes).length > 1) {
      delete workflowNodes[workflowName];
    }

    // Create dagre graph nodes from workflow nodes.
    Object.values(workflowNodes)
      .forEach((node) => {
        g.setNode(node.id, {
          height: NODE_HEIGHT,
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
      if (this._isVirtual(workflowNodes[nodeId])) {
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
    this._workflowNodes = g.nodes().map((id) => {
      return Object.assign(g.node(id), {
        finishedAt: workflowNodes[id].finishedAt,
        icon: nodePhaseToIcon(workflowNodes[id].phase),
        name: workflowNodes[id].displayName || id,
        startedAt: workflowNodes[id].startedAt,
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
            this.push('_workflowEdgeStartPoints', [x1, y1]);
          }
        }
      }
      this.push('_workflowEdges', { from: edgeInfo.v, to: edgeInfo.w, lines });
    });
  }

  protected _exitNodeDetails(): void {
    this.$.nodeDetails.classList.remove('visible');
    this._unselectAllNodes();
  }

  protected _formatDateString(date: string): string {
    return Utils.formatDateString(date);
  }

  protected async _nodeClicked(e: GraphNodeClickEvent<NodeStatus>): Promise<void> {
    this.$.nodeDetails.classList.add('visible');
    this._selectedNode = e.model.node;

    const logsContainer = this.$.logsContainer as HTMLPreElement;
    logsContainer.innerText = '';

    const errorEl = this.$.logsError as PageError;
    errorEl.style.display = 'none';
    logsContainer.style.display = 'none';

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

    try {
      this._loadingLogs = true;
      const logs = await Apis.getPodLogs(this._selectedNode.id);
      logsContainer.style.display = 'block';
      logsContainer.innerText = logs;
    } catch (err) {
      errorEl.style.display = 'block';
      errorEl.showButton = false;
      errorEl.error = `Could not retrieve logs for pod: ${this._selectedNode.id}. Error:\n${err}`;
      logsContainer.style.display = 'none';
    } finally {
      this._loadingLogs = false;
    }
  }

  private _unselectAllNodes(): void {
    const root = this.shadowRoot as ShadowRoot;
    root.querySelectorAll('.job-node').forEach((node) => node.classList.remove('selected'));
  }

  // Outbound nodes are roughly those nodes which are the final step of the
  // workflow's execution. More information can be found in the NodeStatus
  // interface definition.
  private _getOutboundNodes(graph: ArgoTemplate, nodeId: string): string[] {
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
