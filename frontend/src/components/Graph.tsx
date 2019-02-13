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

interface Segment {
  angle: number;
  distance: number;
  finalY: number;
  finalX: number;
  height?: number;
  left: number;
  top: number;
}

interface Edge {
  color?: string;
  from: string;
  to: string;
  segments: Segment[];
  isPlaceholder?: boolean;
}

const css = stylesheet({
  arrowHead: {
    borderBottomColor: 'transparent',
    borderLeftColor: 'transparent',
    borderRightColor: 'transparent',
    borderStyle: 'solid',
    borderTopColor: color.grey,
    borderWidth: '7px 6px 0 6px',
    content: `''`,
    position: 'absolute',
    zIndex: 2,
  },
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
  line: {
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

interface GraphState {
  hoveredNode?: string;
}

export default class Graph extends React.Component<GraphProps, GraphState> {
  private LEFT_OFFSET = 100;
  private TOP_OFFSET = 42;
  private EDGE_THICKNESS = 2;
  private EDGE_X_BUFFER = 30;

  constructor(props: any) {
    super(props);

    this.state = {};
  }

  public render(): JSX.Element | null {
    const { graph } = this.props;
    const displayEdges: Edge[] = [];

    if (!graph.nodes().length) {
      return null;
    }
    dagre.layout(graph);

    // Creates the lines that constitute the edges connecting the graph
    graph.edges().forEach((edgeInfo) => {
      const edge = graph.edge(edgeInfo);
      const segments: Segment[] = [];

      if (edge.points.length > 1) {
        for (let i = 1; i < edge.points.length; i++) {

          let x1 = edge.points[i - 1].x;
          let y1 = edge.points[i - 1].y;

          // Make all edges start at the bottom center of the node
          if (i === 1) {
            const sourceNode = graph.node(edgeInfo.v);
            // If first segment's x value is too far to the right, move it EDGE_X_BUFFER pixels in
            // from the right end of the destination node
            const rightmostAcceptableXPos =
              sourceNode.x + sourceNode.width - this.LEFT_OFFSET - this.EDGE_X_BUFFER;
            if (rightmostAcceptableXPos <= x1) {
              x1 = rightmostAcceptableXPos;
            }

            // If first segment's x value is too far to the left, move it EDGE_X_BUFFER pixels in
            // from the left end of the destination node
            const leftmostAcceptableXPos =
              sourceNode.x - this.LEFT_OFFSET + this.EDGE_X_BUFFER;
            if (leftmostAcceptableXPos >= x1) {
              x1 = leftmostAcceptableXPos;
            }
            y1 = sourceNode.y + (sourceNode.height / 2) - 3;
          }

          let x2 = edge.points[i].x;
          let y2 = edge.points[i].y;

          // Adjustments made to final segment for each edge
          if (i === edge.points.length - 1) {
            const destinationNode = graph.node(edgeInfo.w);

            // Placeholder nodes never need adjustment because they always have only a single incoming edge
            if (!destinationNode.isPlaceholder) {
              // Set the final segment's y value to point to the top of the destination node
              y2 = destinationNode.y - this.TOP_OFFSET + 7;

              // If final segment's x value is too far to the right, move it EDGE_X_BUFFER pixels in
              // from the right end of the destination node
              const rightmostAcceptableXPos =
                destinationNode.x + destinationNode.width - this.LEFT_OFFSET - this.EDGE_X_BUFFER;
              if (rightmostAcceptableXPos <= x2) {
                x2 = rightmostAcceptableXPos;
              }

              // If final segment's x value is too far to the left, move it EDGE_X_BUFFER pixels in
              // from the left end of the destination node
              const leftmostAcceptableXPos =
                destinationNode.x - this.LEFT_OFFSET + this.EDGE_X_BUFFER;
              if (leftmostAcceptableXPos >= x2) {
                x2 = leftmostAcceptableXPos;
              }
            }
          }

          if (i === edge.points.length - 1 && x1 !== x2) {
            // Diagonal
            const yHalf = (y1 + y2) / 2;
            // The + 0.5 at the end of 'distance' helps fill out the elbows of the edges.
            const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - yHalf, 2)) + 0.5;
            const xMid = (x1 + x2) / 2;
            const top = (y1 + yHalf) / 2;
            const angle = Math.atan2(y1 - yHalf, x1 - x2) * 180 / Math.PI;
            const left = xMid - (distance / 2);
            segments.push({ angle, finalY: yHalf, finalX: x2, left, top, distance });

            // Vertical
            segments.push({ angle: 270, finalY: y2, finalX: x2, left: x2 - 5, top: y2 - 5, distance: y2 - yHalf });
          } else {
            // The + 0.5 at the end of 'distance' helps fill out the elbows of the edges.
            const distance = Math.sqrt(Math.pow(x1 - x2, 2) + Math.pow(y1 - y2, 2)) + 0.5;
            const xMid = (x1 + x2) / 2;
            const top = (y1 + y2) / 2;
            const angle = Math.atan2(y1 - y2, x1 - x2) * 180 / Math.PI;
            const left = xMid - (distance / 2);
            segments.push({ angle, finalY: y2, finalX: x2, left, top, distance });
          }
        }
      }
      displayEdges.push({
        color: edge.color,
        from: edgeInfo.v,
        isPlaceholder: edge.isPlaceholder,
        segments,
        to: edgeInfo.w
      });
    });

    const { hoveredNode } = this.state;
    const highlightNode = this.props.selectedNodeId || hoveredNode;

    return (
      <div className={css.root}>
        {graph.nodes().map(id => Object.assign(graph.node(id), { id })).map((node, i) => (
          <div className={classes(node.isPlaceholder ? css.placeholderNode : css.node, 'graphNode',
            node.id === this.props.selectedNodeId ? css.nodeSelected : '')} key={i}
            onMouseEnter={() => {
              if (!this.props.selectedNodeId) {
                this.setState({ hoveredNode: node.id });
              }
            }}
            onMouseLeave={() => {
              if (this.state.hoveredNode === node.id) {
                this.setState({ hoveredNode: undefined });
              }
            }}
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
            <div className={css.icon} style={{ background: node.statusColoring }}>{node.icon}</div>
          </div>
        ))}

        {displayEdges.map((edge, i) => {
          const edgeColor = this.getEdgeColor(edge, highlightNode);
          return (
            <div key={i}>
              {edge.segments.map((segment, l) => (
                <div className={css.line}
                  key={l} style={{
                    backgroundColor: edgeColor,
                    // height: segment.height,
                    height: this.EDGE_THICKNESS,
                    left: segment.left,
                    top: segment.top,
                    transform: `translate(100px, 44px) rotate(${segment.angle}deg)`,
                    transition: 'left 0.5s, top 0.5s',
                    width: segment.distance,
                  }} />
              ))}
              {!edge.isPlaceholder && (
                <div className={css.arrowHead} style={{
                  borderTopColor: edgeColor,
                  left: edge.segments[edge.segments.length - 1].finalX,
                  top: edge.segments[edge.segments.length - 1].finalY,
                  transform: `translate(94px, 41px) rotate(${edge.segments[edge.segments.length - 1].angle + 90}deg)`,
                }} />
              )}
            </div>
          );}
        )}
      </div>
    );
  }

  private getEdgeColor(edge: Edge, highlightNode?: string): string {
    if (highlightNode) {
      if (edge.from === highlightNode) {
        return color.theme;
      }
      if (edge.to === highlightNode) {
        return color.themeDarker;
      }
    }
    if (edge.isPlaceholder) {
      return color.weak;
    }
    return color.grey;
  }
}
