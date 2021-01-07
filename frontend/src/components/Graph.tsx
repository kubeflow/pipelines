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
import { fontsize, color, fonts, zIndex } from '../Css';
import { Constants } from '../lib/Constants';
import Tooltip from '@material-ui/core/Tooltip';
import { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

interface Segment {
  angle: number;
  length: number;
  x1: number;
  x2: number;
  y1: number;
  y2: number;
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
    borderColor: color.grey + ' transparent transparent transparent',
    borderStyle: 'solid',
    borderWidth: '7px 6px 0 6px',
    content: `''`,
    position: 'absolute',
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
    overflow: 'hidden',
    padding: 10,
    textOverflow: 'ellipsis',
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
    zIndex: zIndex.GRAPH_NODE,
  },
  nodeSelected: {
    border: `solid 2px ${color.theme}`,
  },
  placeholderNode: {
    margin: 10,
    position: 'absolute',
    // TODO: can this be calculated?
    transform: 'translate(71px, 14px)',
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
  t: TFunction;
}

interface GraphState {
  hoveredNode?: string;
}

interface GraphErrorBoundaryProps {
  onError?: (message: string, additionalInfo: string) => void;
  t: TFunction;
}
class GraphErrorBoundary extends React.Component<GraphErrorBoundaryProps> {
  state = {
    hasError: false,
  };

  componentDidCatch(error: Error): void {
    const { t } = this.props;
    const message = t('errorRenderGraph');
    const additionalInfo = `${message} ${t('bugKubeflowError')}: '${error.message}'.`;
    if (this.props.onError) {
      this.props.onError(message, additionalInfo);
    }
    this.setState({
      hasError: true,
    });
  }

  render() {
    return this.state.hasError ? <div className={css.root} /> : this.props.children;
  }
}

export class Graph extends React.Component<GraphProps, GraphState> {
  private LEFT_OFFSET = 100;
  private TOP_OFFSET = 44;
  private EDGE_THICKNESS = 2;
  private EDGE_X_BUFFER = Math.round(Constants.NODE_WIDTH / 6);

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
    graph.edges().forEach(edgeInfo => {
      const edge = graph.edge(edgeInfo);
      const segments: Segment[] = [];
      const { t } = this.props;

      if (edge.points.length > 1) {
        for (let i = 1; i < edge.points.length; i++) {
          let xStart = edge.points[i - 1].x;
          let yStart = edge.points[i - 1].y;
          let xEnd = edge.points[i].x;
          let yEnd = edge.points[i].y;

          const downwardPointingSegment = yStart <= yEnd;

          // Adjustments made to the start of the first segment for each edge to ensure that it
          // begins at the bottom of the source node and that there are at least EDGE_X_BUFFER
          // pixels between it and the right and left side of the node.
          // Note that these adjustments may cause edges to overlap with nodes since we are
          // deviating from the explicit layout provided by dagre.
          if (i === 1) {
            const sourceNode = graph.node(edgeInfo.v);

            if (!sourceNode) {
              throw new Error(`${t('common:graphDefInvalid', { edgeInfo: edgeInfo.v })}`);
            }

            // Set the edge's first segment to start at the bottom or top of the source node.
            yStart = downwardPointingSegment
              ? sourceNode.y + sourceNode.height / 2 - 3
              : sourceNode.y - sourceNode.height / 2;

            xStart = this._ensureXIsWithinNode(sourceNode, xStart);
          }

          const finalSegment = i === edge.points.length - 1;

          // Adjustments made to the end of the final segment for each edge to ensure that it ends
          // at the top of the destination node and that there are at least EDGE_X_BUFFER pixels
          // between it and the right and left side of the node. The adjustments are only needed
          // when there are multiple inbound edges as dagre seems to always layout a single inbound
          // edge so that it terminates at the center-top of the destination node. For this reason,
          // placeholder nodes do not need adjustments since they always have only a single inbound
          // edge.
          // Note that these adjustments may cause edges to overlap with nodes since we are
          // deviating from the explicit layout provided by dagre.
          if (finalSegment) {
            const destinationNode = graph.node(edgeInfo.w);

            // Placeholder nodes never need adjustment because they always have only a single
            // incoming edge.
            if (!destinationNode.isPlaceholder) {
              // Set the edge's final segment to terminate at the top or bottom of the destination
              // node.
              yEnd = downwardPointingSegment
                ? destinationNode.y - this.TOP_OFFSET + 5
                : destinationNode.y + destinationNode.height / 2 + 3;

              xEnd = this._ensureXIsWithinNode(destinationNode, xEnd);
            }
          }

          // For the final segment of the edge, if the segment is diagonal, split it into a diagonal
          // and a vertical piece so that all edges terminate with a vertical segment.
          if (finalSegment && xStart !== xEnd) {
            const yHalf = (yStart + yEnd) / 2;
            this._addDiagonalSegment(segments, xStart, yStart, xEnd, yHalf);

            // Vertical segment
            if (downwardPointingSegment) {
              segments.push({
                angle: 270,
                length: yEnd - yHalf,
                x1: xEnd - 5,
                x2: xEnd,
                y1: yHalf + 4,
                y2: yEnd,
              });
            } else {
              segments.push({
                angle: 90,
                length: yHalf - yEnd,
                x1: xEnd - 5,
                x2: xEnd,
                y1: yHalf - 4,
                y2: yEnd,
              });
            }
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
        to: edgeInfo.w,
      });
    });

    const { hoveredNode } = this.state;
    const highlightNode = this.props.selectedNodeId || hoveredNode;

    return (
      <div className={css.root}>
        {graph
          .nodes()
          .map(id => Object.assign(graph.node(id), { id }))
          .map((node, i) => (
            <div
              className={classes(
                node.isPlaceholder ? css.placeholderNode : css.node,
                'graphNode',
                node.id === this.props.selectedNodeId ? css.nodeSelected : '',
              )}
              key={i}
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
              onClick={() =>
                !node.isPlaceholder && this.props.onClick && this.props.onClick(node.id)
              }
              style={{
                backgroundColor: node.bgColor,
                left: node.x,
                maxHeight: node.height,
                minHeight: node.height,
                top: node.y,
                transition: 'left 0.5s, top 0.5s',
                width: node.width,
              }}
            >
              {!node.isPlaceholder && (
                <Tooltip title={node.label} enterDelay={300}>
                  <div className={css.label}>{node.label}</div>
                </Tooltip>
              )}
              <div className={css.icon} style={{ background: node.statusColoring }}>
                {node.icon}
              </div>
            </div>
          ))}

        {displayEdges.map((edge, i) => {
          const edgeColor = this._getEdgeColor(edge, highlightNode);
          const lastSegment = edge.segments[edge.segments.length - 1];
          return (
            <div key={i}>
              {edge.segments.map((segment, l) => (
                <div
                  className={css.line}
                  key={l}
                  style={{
                    backgroundColor: edgeColor,
                    height: this.EDGE_THICKNESS,
                    left: segment.x1 + this.LEFT_OFFSET,
                    top: segment.y1 + this.TOP_OFFSET,
                    transform: `rotate(${segment.angle}deg)`,
                    transition: 'left 0.5s, top 0.5s',
                    width: segment.length,
                  }}
                />
              ))}
              {/* Arrowhead */}
              {!edge.isPlaceholder && lastSegment.x2 !== undefined && lastSegment.y2 !== undefined && (
                <div
                  className={css.arrowHead}
                  style={{
                    borderTopColor: edgeColor,
                    left: lastSegment.x2 + this.LEFT_OFFSET - 6,
                    top: lastSegment.y2 + this.TOP_OFFSET - 3,
                    transform: `rotate(${lastSegment.angle + 90}deg)`,
                  }}
                />
              )}
            </div>
          );
        })}
      </div>
    );
  }

  private _addDiagonalSegment(
    segments: Segment[],
    xStart: number,
    yStart: number,
    xEnd: number,
    yEnd: number,
  ): void {
    const xMid = (xStart + xEnd) / 2;
    // The + 0.5 at the end of 'length' helps fill out the elbows of the edges.
    const length = Math.sqrt(Math.pow(xStart - xEnd, 2) + Math.pow(yStart - yEnd, 2)) + 0.5;
    const x1 = xMid - length / 2;
    const y1 = (yStart + yEnd) / 2;
    const angle = (Math.atan2(yStart - yEnd, xStart - xEnd) * 180) / Math.PI;
    segments.push({
      angle,
      length,
      x1,
      x2: xEnd,
      y1,
      y2: yEnd,
    });
  }

  /**
   * Adjusts the x positioning of the start or end of an edge so that it is at least EDGE_X_BUFFER
   * pixels in from the left and right.
   * @param node the node where the edge is originating from or terminating at
   * @param originalX the initial x position provided by dagre
   */
  private _ensureXIsWithinNode(node: dagre.Node, originalX: number): number {
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

const EnhancedGraph = (props: GraphProps & GraphErrorBoundaryProps) => {
  const { t } = useTranslation('common');
  return (
    <GraphErrorBoundary onError={props.onError} t={t}>
      <Graph {...props} />
    </GraphErrorBoundary>
  );
};
EnhancedGraph.displayName = 'EnhancedGraph';

export default EnhancedGraph;
