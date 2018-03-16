import 'iron-icons/iron-icons.html';

import * as dagre from 'dagre';

import { customElement, property } from '../../decorators';
import { jobStatusToIcon } from '../../lib/utils';
import { Workflow as ArgoTemplate } from '../../model/argo_template';

import './job-graph.html';

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

@customElement('job-graph')
export class JobGraph extends Polymer.Element {

  @property({ type: Object })
  jobGraph: ArgoTemplate | null = null;

  @property({ type: Array })
  protected _workflowNodes: DisplayNode[] = [];

  @property({ type: Array })
  protected _workflowEdges: Edge[] = [];

  refresh(graph: ArgoTemplate) {
    this.jobGraph = graph;

    const g = new dagre.graphlib.Graph();
    g.setGraph({});
    g.setDefaultEdgeLabel(() => ({}));
    const nodes = this.jobGraph.status.nodes;
    Object.keys(nodes).forEach((k) =>
      g.setNode(k, { label: k, width: 150, height: 50 }));
    Object.keys(nodes).forEach((k) => {
      if (nodes[k].children) {
        nodes[k].children.forEach((c) => g.setEdge(k, c));
      }
    });

    dagre.layout(g);

    this._workflowNodes = g.nodes().map((id) => {
      return Object.assign(g.node(id), {
        finishedAt: nodes[id].finishedAt,
        icon: jobStatusToIcon(nodes[id].phase),
        name: nodes[id].name,
        startedAt: nodes[id].startedAt,
      });
    });

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
        }
      }
      this.push('_workflowEdges', { from: edgeInfo.v, to: edgeInfo.w, lines });
    });
  }
}
