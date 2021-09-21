/*
 * Copyright 2021 The Kubeflow Authors
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

import * as React from 'react';
import { classes } from 'typestyle';
import { edgeCanvasCss } from './EdgeCanvas';
import { EdgeLine } from './EdgeLine';
import {
  CARD_ROW_HEIGHT,
  CARD_CONTAINER_BORDERS,
  CARD_SPACER_HEIGHT,
  CARD_TITLE_HEIGHT,
} from './LineageCss';

export const CARD_OFFSET = CARD_SPACER_HEIGHT + CARD_TITLE_HEIGHT + CARD_CONTAINER_BORDERS;

interface ControlledEdgeCanvasProps {
  // The edge is drawn over the card column and offset by the card width
  cardWidth: number;
  edgeWidth: number;

  // If true edges are drawn from right to left.
  reverseEdges: boolean;

  // A list of output artifactIds
  artifactIds: number[];

  // Offset for the artifact edge lines in pixels.
  offset: number;

  // Top of the destination execution card relative to the top of the lineage card column.
  top: number;

  // Map from artifact id to card index.
  artifactToCardMap: Map<number, number>;

  outputExecutionToOutputArtifactMap: Map<number, number[]>;
}

/**
 * A version of <EdgeCanvas/> that draws the lines connecting the edges of a list of vertically
 * stacked output artifact cards <LineageCardColumn /> to the many output execution <LineageCard />
 * elements in an adjacent <LineageCardColumn />.
 */
export class ControlledEdgeCanvas extends React.Component<ControlledEdgeCanvasProps> {
  public render(): JSX.Element | null {
    const { reverseEdges, edgeWidth } = this.props;

    let viewHeight = 1;

    const lastNode = reverseEdges ? 'y1' : 'y4';
    const lastNodePositions = { y1: 0, y4: 0 };
    if (this.props.offset) {
      lastNodePositions[lastNode] += this.props.offset;
      viewHeight += this.props.offset;
    }

    const edgeLines: JSX.Element[] = [];
    this.props.artifactIds.forEach((artifactId, index) => {
      if (index !== 0) {
        let offset = CARD_OFFSET;

        if (
          this.props.artifactToCardMap.get(artifactId) !==
          this.props.artifactToCardMap.get(artifactId - 1)
        ) {
          offset += CARD_ROW_HEIGHT;
        }

        viewHeight += offset;
        lastNodePositions[lastNode] += offset;
      }

      edgeLines.push(
        <EdgeLine
          height={viewHeight}
          width={edgeWidth}
          y1={lastNodePositions.y1}
          y4={lastNodePositions.y4}
        />,
      );
    });

    const css = edgeCanvasCss(
      /* left= */ this.props.cardWidth,
      /* top= */ this.props.top + CARD_TITLE_HEIGHT + CARD_ROW_HEIGHT / 2,
      /* width= */ edgeWidth,
    );
    const edgeCanvasClasses = classes(css.edgeCanvas, reverseEdges && 'edgeCanvasReverse');
    return <div className={edgeCanvasClasses}>{edgeLines}</div>;
  }
}
