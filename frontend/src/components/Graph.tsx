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
import * as React from 'react';
import { classes, stylesheet } from 'typestyle';
import { fontsize, color } from '../Css';

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

const css = stylesheet({
  icon: {
    margin: 5,
  },
  label: {
    flexGrow: 1,
    fontSize: 15,
    lineHeight: '2em',
    margin: 'auto',
    overflow: 'hidden',
    paddingLeft: 15,
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  lastEdgeLine: {
    $nest: {
      // Arrowhead
      '&::after': {
        borderColor: `${color.theme} transparent transparent transparent`,
        borderStyle: 'solid',
        borderWidth: '7px 6px 0 6px',
        clear: 'both',
        content: `''`,
        left: -5,
        position: 'absolute',
        top: -5,
        transform: 'rotate(90deg)',
      },
    },
  },
  line: {
    borderTop: `2px solid ${color.theme}`,
    position: 'absolute',
  },
  node: {
    $nest: {
      '&:hover': {
        borderColor: color.theme,
      },
    },
    backgroundColor: color.background,
    border: `solid 1px ${color.theme}`,
    borderRadius: 5,
    boxShadow: '1px 1px 5px #aaa',
    boxSizing: 'content-box',
    color: '#124aa4',
    cursor: 'pointer',
    display: 'flex',
    fontSize: fontsize.medium,
    margin: 10,
    position: 'absolute',
  },
  nodeSelected: {
    backgroundColor: '#e4ebff !important',
    borderColor: color.theme,
  },
  root: {
    backgroundColor: color.graphBg,
    borderLeft: 'solid 1px ' + color.divider,
    flexGrow: 1,
    overflow: 'auto',
    position: 'relative',
  },
  startCircle: {
    backgroundColor: color.background,
    border: `1px solid ${color.theme}`,
    borderRadius: 7,
    content: '',
    display: 'inline-block',
    height: 8,
    position: 'absolute',
    width: 8,
  },
});

interface GraphProps {
  graph: dagre.graphlib.Graph;
  onClick?: (id: string) => void;
  selectedNodeId?: string;
}

export default class Graph extends React.Component<GraphProps> {
  public render(): JSX.Element {
    const { graph } = this.props;
    const displayEdges: Edge[] = [];
    const displayEdgeStartPoints: number[][] = [];

    dagre.layout(graph);

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
          // The + 0.5 at the end of 'distance' helps fill out the elbows of the edges.
          const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2)) + 0.5;
          const xMid = (x1 + x2) / 2;
          const yMid = (y1 + y2) / 2;
          const angle = Math.atan2(y1 - y2, x1 - x2) * 180 / Math.PI;
          const left = xMid - (distance / 2);
          lines.push({ x1, y1, x2, y2, distance, xMid, yMid, angle, left });

          // Store the first point of the edge to draw the edge start circle
          if (i === 1) {
            displayEdgeStartPoints.push([x1, y1]);
          }
        }
      }
      displayEdges.push({ from: edgeInfo.v, to: edgeInfo.w, lines });
    });

    return (
      <div className={css.root}>
        {graph.nodes().map(id => Object.assign(graph.node(id), { id })).map((node, i) => (
          <div className={classes(css.node, 'graphNode',
            node.id === this.props.selectedNodeId ? css.nodeSelected : '')} key={i}
            onClick={() => this.props.onClick && this.props.onClick(node.id)} style={{
              backgroundColor: node.bgColor, left: node.x,
              maxHeight: node.height, minHeight: node.height, top: node.y, width: node.width,
            }}>
            <div className={css.label}>{node.label}</div>
            <div className={css.icon}>{node.icon}</div>
          </div>
        ))}

        {displayEdges.map((edge, i) => (
          <div key={i}>
            {edge.lines.map((line, l) => (
              <div className={classes(css.line, l === edge.lines.length - 1 ? css.lastEdgeLine : '')}
                key={l} style={{
                  left: line.left,
                  top: line.yMid,
                  transform: `translate(100px, 44px) rotate(${line.angle}deg)`,
                  width: line.distance,
                }} />
            ))}
          </div>
        ))}

        {displayEdgeStartPoints.map((point, i) => (
          <div className={css.startCircle} key={i} style={{
            left: `calc(${point[0]}px - 5px)`,
            top: `calc(${point[1]}px - 3px)`,
            transform: 'translate(100px, 44px)',
          }} />
        ))}
      </div>
    );
  }
}
