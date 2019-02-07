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
  distance: number;
  top: number;
  angle: number;
  left: number;
}

interface Edge {
  color?: string;
  from: string;
  to: string;
  lines: Line[];
  isPlaceholder?: boolean;
}

const css = stylesheet({
  icon: {
    borderRadius: '0px 2px 2px 0px',
    padding: '5px 7px 0px 7px',
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
        borderColor: `${color.grey} transparent transparent transparent`,
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
    zIndex: 2,
  },
  line: {
    borderTop: `2px solid ${color.grey}`,
    position: 'absolute',
  },
  node: {
    $nest: {
      '&:hover': {
        borderColor: color.theme,
      },
    },
    backgroundColor: color.background,
    border: 'solid 1px #d6d6d6',
    borderRadius: 3,
    // boxShadow: '1px 1px 5px #aaa',
    boxSizing: 'content-box',
    color: '#124aa4',
    cursor: 'pointer',
    display: 'flex',
    fontSize: fontsize.medium,
    margin: 10,
    position: 'absolute',
    zIndex: 1,
  },
  nodeSelected: {
    // backgroundColor: '#e4ebff !important',
    border: `solid 2px ${color.theme}`,
  },
  placeholderNode: {
    margin: 10,
    position: 'absolute',
    // TODO: can this be calculated?
    transform: 'translate(73px, 16px)'
  },
  root: {
    backgroundColor: color.graphBg,
    borderLeft: 'solid 1px ' + color.divider,
    flexGrow: 1,
    overflow: 'auto',
    position: 'relative',
  },
  startCircle: {
    backgroundColor: color.grey,
    borderRadius: 7,
    content: '',
    display: 'inline-block',
    height: 8,
    position: 'absolute',
    width: 8,
    zIndex: 0,
  },
});

interface GraphProps {
  graph: dagre.graphlib.Graph;
  onClick?: (id: string) => void;
  selectedNodeId?: string;
}

export default class Graph extends React.Component<GraphProps> {
  public render(): JSX.Element | null {
    const { graph } = this.props;
    const displayEdges: Edge[] = [];
    const displayEdgeStartPoints: number[][] = [];

    if (!graph.nodes().length) {
      return null;
    }
    dagre.layout(graph);

    // Creates the lines that constitute the edges connecting the graph.
    graph.edges().forEach((edgeInfo) => {
      const edge = graph.edge(edgeInfo);
      const lines: Line[] = [];
      // if (edge.points.length > 1) {
      //   for (let i = 1; i < edge.points.length; i++) {
      //     const x1 = edge.points[i - 1].x;
      //     const y1 = edge.points[i - 1].y;
      //     const x2 = edge.points[i].x;
      //     let y2 = edge.points[i].y;

      //     // Small adjustment of final edge to not intersect as much with destination node.
      //     if (i === edge.points.length - 1) {
      //       y2 = y2 - 3;
      //     }

      //     // The + 0.5 at the end of 'distance' helps fill out the elbows of the edges.
      //     const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2)) + 0.5;
      //     const xMid = (x1 + x2) / 2;
      //     const yMid = (y1 + y2) / 2;
      //     const angle = Math.atan2(y1 - y2, x1 - x2) * 180 / Math.PI;
      //     const left = xMid - (distance / 2);

      //     lines.push({ distance, yMid, angle, left });

      //     // Store the first point of the edge to draw the edge start circle
      //     if (i === 1) {
      //       displayEdgeStartPoints.push([x1, y1]);
      //     }
      //   }
      // }
      // const edgeStart
      if (edge.points.length > 1) {
        for (let i = 1; i < edge.points.length; i++) {
          const x1 = edge.points[i - 1].x;
          const y1 = edge.points[i - 1].y;
          const x2 = edge.points[i].x;
          let y2 = edge.points[i].y;

          // Small adjustment of final edge to not intersect as much with destination node.
          if (i === edge.points.length - 1) {
            y2 = y2 - 3;
          }

          if (x1 === x2) {
            const distance = y2-y1;
            lines.push({ distance, top: (y1 + y2) / 2, angle: 270, left: x1 - (distance / 2) });
          } else if (y1 === y2) {
            const distance = x2-x1;
            const xMid = (x1 + x2) / 2;
            lines.push({ distance, top: y1, angle: 0, left: xMid - (distance / 2) });
          } else {
            const verticalSegmentLength = (y2-y1) / 2;
            const top1 = (3*y1 + y2) / 4;
            lines.push({ distance: verticalSegmentLength, top: top1, angle: 270, left: x1 - (verticalSegmentLength / 2)});
            const distanceX = Math.abs(x2-x1);
            // const xMid = (x1 + x2) / 2;
            lines.push({ distance: distanceX, top: (y1+y2)/2, angle: 0, left: Math.min(x1, x2) });
            const top2 = (y1 + 3*y2) / 4;
            lines.push({ distance: verticalSegmentLength, top: top2, angle: 270, left: x2 - (verticalSegmentLength / 2)});
          }
          // The + 0.5 at the end of 'distance' helps fill out the elbows of the edges.
          // const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2)) + 0.5;
          // const xMid = (x1 + x2) / 2;
          // const yMid = (y1 + y2) / 2;
          // const left = xMid - (distance / 2);


          // Store the first point of the edge to draw the edge start circle
          if (i === 1) {
            displayEdgeStartPoints.push([x1, y1]);
          }
        }
      }
      displayEdges.push({
        color: edge.color,
        from: edgeInfo.v,
        isPlaceholder: edge.isPlaceholder,
        lines,
        to: edgeInfo.w
      });
    });

    return (
      <div className={css.root}>
        {graph.nodes().map(id => Object.assign(graph.node(id), { id })).map((node, i) => (
          <div className={classes(node.isPlaceholder ? css.placeholderNode : css.node, 'graphNode',
            node.id === this.props.selectedNodeId ? css.nodeSelected : '')} key={i}
            onClick={() => (!node.isPlaceholder && this.props.onClick) && this.props.onClick(node.id)}
            style={{
              backgroundColor: node.bgColor, left: node.x,
              maxHeight: node.height,
              minHeight: node.height,
              top: node.y,
              transition: 'left 0.5s, top 0.5s',
              width: node.width,
            }}>
            <div className={css.label}>{node.label}</div>
            <div className={css.icon} style ={{ background: node.statusColoring }}>{node.icon}</div>
          </div>
        ))}

        {displayEdges.map((edge, i) => (
          <div key={i}>
            {edge.lines.map((line, l) => (
              <div className={classes(
                  css.line,
                  (l === edge.lines.length - 1 && !edge.isPlaceholder) ? css.lastEdgeLine : ''
                )}
                key={l} style={{
                  borderTopColor: edge.color,
                  borderTopStyle: edge.isPlaceholder ? 'dotted' : 'solid',
                  left: line.left,
                  top: line.top,
                  transform: `translate(100px, 44px) rotate(${line.angle}deg)`,
                  transition: 'left 0.5s, top 0.5s',
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
