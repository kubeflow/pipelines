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
import { fontsize, color, fonts } from '../Css';

interface Segment {
  angle: number;
  // finalX and finalY are used for placing the arrowheads
  finalX: number;
  finalY: number;
  left: number;
  length: number;
  top: number;
}

interface Edge {
  color?: string;
  isPlaceholder?: boolean;
  from: string;
  segments: Segment[];
  to: string;
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
    padding: '5px 7px 0px 7px',
  },
  label: {
    color: color.strong,
    flexGrow: 1,
    fontFamily: fonts.secondary,
    fontSize: 13,
    fontWeight: 500,
    lineHeight: '16px',
    margin: 10,
    overflow: 'hidden',
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
    transform: 'translate(71px, 14px)'
  },
  root: {
    backgroundColor: color.graphBg,
    borderLeft: 'solid 1px ' + color.divider,
    flexGrow: 1,
    overflow: 'auto',
    position: 'relative',
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
  private TOP_OFFSET = 44;
  private EDGE_THICKNESS = 2;
  private EDGE_X_BUFFER = 30;

  constructor(props: any) {
    super(props);

    this.state = {};
  }

  public render(): JSX.Element | null {
    const { graph } = this.props;

    if (!graph.nodes().length) {
      return null;
    }

    dagre.layout(graph);
    const displayEdges: Edge[] = [];
    
    // Creates the lines that constitute the edges connecting the graph.
    graph.edges().forEach((edgeInfo) => {
      const edge = graph.edge(edgeInfo);
      const segments: Segment[] = [];

      if (edge.points.length > 1) {
        for (let i = 1; i < edge.points.length; i++) {

          let xStart = edge.points[i - 1].x;
          let yStart = edge.points[i - 1].y;

          // Adjustments made to first segment for each edge.
          if (i === 1) {
            const sourceNode = graph.node(edgeInfo.v);
            
            // Set the edge's first segment to start at the bottom of the source node.
            yStart = sourceNode.y + (sourceNode.height / 2) - 3;
            
            xStart = this._adjustXVal(sourceNode, xStart);
          }

          let xEnd = edge.points[i].x;
          let yEnd = edge.points[i].y;

          // Adjustments made to final segment for each edge.
          if (i === edge.points.length - 1) {
            const destinationNode = graph.node(edgeInfo.w);

            // Placeholder nodes never need adjustment because they always have only a single
            // incoming edge.
            if (!destinationNode.isPlaceholder) {
              // Set the edge's final segment to terminate at the top of the destination node.
              yEnd = destinationNode.y - this.TOP_OFFSET + 5;

              xEnd = this._adjustXVal(destinationNode, xEnd);
            }
          }
          
          // For the final segment of the edge, if the segment is diagonal, split it into a diagonal
          // and a vertical piece so that all edges terminate with a vertical segment.
          if (i === edge.points.length - 1 && xStart !== xEnd) {
            const yHalf = (yStart + yEnd) / 2;
            this._addDiagonalSegment(segments, xStart, yStart, xEnd, yHalf);

            // Vertical segment
            segments.push({
              angle: 270,
              finalX: xEnd,
              finalY: yEnd,
              left: xEnd - 5,
              length: yEnd - yHalf,
              top: yEnd - 5,
            });
          } else {
            this._addDiagonalSegment(segments, xStart, yStart, xEnd, yEnd);
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
            {!node.isPlaceholder && (<div className={css.label}>{node.label}</div>)}
            <div className={css.icon} style={{ background: node.statusColoring }}>{node.icon}</div>
          </div>
        ))}

        {displayEdges.map((edge, i) => {
          const edgeColor = this._getEdgeColor(edge, highlightNode);
          const lastSegment = edge.segments[edge.segments.length - 1];
          return (
            <div key={i}>
              {edge.segments.map((segment, l) => (
                <div className={css.line}
                  key={l} style={{
                    backgroundColor: edgeColor,
                    height: this.EDGE_THICKNESS,
                    left: segment.left,
                    top: segment.top,
                    transform: `translate(${this.LEFT_OFFSET}px, ${this.TOP_OFFSET}px) rotate(${segment.angle}deg)`,
                    transition: 'left 0.5s, top 0.5s',
                    width: segment.length,
                  }} />
              ))}
              {/* Arrowhead */}
              {!edge.isPlaceholder && (
                <div className={css.arrowHead} style={{
                  borderTopColor: edgeColor,
                  left: lastSegment.finalX,
                  top: lastSegment.finalY,
                  transform: `translate(${this.LEFT_OFFSET - 6}px, ${this.TOP_OFFSET - 3}px) rotate(${lastSegment.angle + 90}deg)`,
                }} />
              )}
            </div>
          );
        })}
      </div>
    );
  }

  private _addDiagonalSegment(segments: Segment[], xStart: number, yStart: number, xEnd: number, yEnd: number): void {
    const xMid = (xStart + xEnd) / 2;
    // The + 0.5 at the end of 'length' helps fill out the elbows of the edges.
    const length = Math.sqrt(Math.pow(xStart - xEnd, 2) + Math.pow(yStart - yEnd, 2)) + 0.5;
    const left = xMid - (length / 2);
    const top = (yStart + yEnd) / 2;
    const angle = Math.atan2(yStart - yEnd, xStart - xEnd) * 180 / Math.PI;
    segments.push({
      angle,
      finalX: xEnd,
      finalY: yEnd,
      left,
      length,
      top,
    });
  }

  private _adjustXVal(node: dagre.Node, originalX: number): number {
    // If the original X value was too far to the right, move it EDGE_X_BUFFER pixels
    // in from the left end of the node.
    const rightmostAcceptableLoc = node.x + node.width - this.LEFT_OFFSET - this.EDGE_X_BUFFER;
    if (rightmostAcceptableLoc <= originalX) {
      return rightmostAcceptableLoc;
    }

    // If the original X value was too far to the left, move it EDGE_X_BUFFER pixels
    // in from the left end of the node.
    const leftmostAcceptableLoc = node.x - this.LEFT_OFFSET + this.EDGE_X_BUFFER;
    if (leftmostAcceptableLoc >= originalX) {
      return leftmostAcceptableLoc;
    }

    return originalX;
  }

  private _getEdgeColor(edge: Edge, highlightNode?: string): string {
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
