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

import * as dagre from 'dagre';
import * as Utils from '../../lib/utils';

import { customElement, property } from 'polymer-decorators/src/decorators';
import { GraphNodeClickEvent } from '../../model/events';

import './pipeline-graph.html';

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

// Alias the Graph class to hide implementation
export import Graph = dagre.graphlib.Graph;

@customElement('pipeline-graph')
export class PipelineGraph extends Polymer.Element {

  @property({ type: Array })
  protected _displayNodes: dagre.Node[] = [];

  @property({ type: Array })
  protected _displayEdges: Edge[] = [];

  @property({ type: Array })
  protected _displayEdgeStartPoints: number[][] = [];

  public refresh(graph: Graph): void {
    this._displayEdges = [];
    this._displayEdgeStartPoints = [];

    dagre.layout(graph);

    // Creates the array of nodes used that will be rendered.
    this._displayNodes = graph.nodes().map((id) => graph.node(id));

    // Creates the lines that constitute the edges connecting the graph.
    graph.edges().forEach((edgeInfo) => {
      const edge = graph.edge(edgeInfo);
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

  public clear(): void {
    this._displayNodes = this._displayEdges = this._displayEdgeStartPoints = [];
  }

  public selectNode(id: string): void {
    // Apply 'selected' CSS class to just this node
    this.unselectAllNodes();
    const root = this.shadowRoot as ShadowRoot;
    const selectedNode = root.querySelector('#node_' + id);

    if (selectedNode) {
      selectedNode.classList.add('selected');
    } else {
      // This should never happen
      Utils.showDialog('Error', 'Cannot find clicked node with id: ' + id);
      Utils.log.verbose('Cannot find clicked node with id ' + id);
    }
  }

  public unselectAllNodes(): void {
    const root = this.shadowRoot as ShadowRoot;
    root.querySelectorAll('.job-node').forEach((node) => node.classList.remove('selected'));
  }

  protected _nodeClicked(e: { model: { node: dagre.Node } }): void {
    const node = e.model.node;
    if (!node) {
      return;
    }
    this.dispatchEvent(new GraphNodeClickEvent(node.id));
  }
}
