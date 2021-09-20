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
import { classes, stylesheet } from 'typestyle';
import {
  CARD_ROW_HEIGHT,
  CARD_CONTAINER_BORDERS,
  CARD_SPACER_HEIGHT,
  CARD_TITLE_HEIGHT,
  px,
} from './LineageCss';
import { CardDetails } from './LineageCardColumn';
import { EdgeLine } from './EdgeLine';

export const CARD_OFFSET = CARD_SPACER_HEIGHT + CARD_TITLE_HEIGHT + CARD_CONTAINER_BORDERS;

export const edgeCanvasCss = (left: number, top: number, width: number) => {
  return stylesheet({
    edgeCanvas: {
      border: 0,
      display: 'block',
      left,
      transition: 'transform .25s',
      overflow: 'visible',
      position: 'absolute',
      top: px(top),
      width,
      zIndex: -1,

      $nest: {
        '&.edgeCanvasReverse': {
          left: 0,
          transform: 'translateX(-100%)',
        },
        svg: {
          display: 'block',
          overflow: 'visible',
          position: 'absolute',
        },
      },
    },
  });
};

export interface EdgeCanvasProps {
  cards: CardDetails[];

  // The edge is drawn over the card column and offset by the card width
  cardWidth: number;
  edgeWidth: number;

  // If true edges are drawn from right to left.
  reverseEdges: boolean;
}

/**
 * Canvas that draws the lines connecting the edges of a list of vertically stacked cards in one
 * <LineageCardColumn /> to the topmost <LineageCard /> in an adjacent <LineageCardColumn />.
 *
 * The adjacent column is assumed to be to right of the connecting cards unless `reverseEdges`
 * is set to true.
 */
export class EdgeCanvas extends React.Component<EdgeCanvasProps> {
  public render(): JSX.Element | null {
    const { cards, reverseEdges, edgeWidth } = this.props;

    let viewHeight = 1;

    const lastNode = reverseEdges ? 'y1' : 'y4';
    const lastNodePositions = { y1: 0, y4: 0 };

    const edgeLines: JSX.Element[] = [];
    cards.forEach((card, i) => {
      card.elements.forEach((element, j) => {
        if (!element.next) {
          return;
        }

        edgeLines.push(
          <EdgeLine
            height={viewHeight}
            width={edgeWidth}
            y1={lastNodePositions.y1}
            y4={lastNodePositions.y4}
          />,
        );

        viewHeight += CARD_ROW_HEIGHT;
        lastNodePositions[lastNode] += CARD_ROW_HEIGHT;
      });
      viewHeight += CARD_OFFSET;
      lastNodePositions[lastNode] += CARD_OFFSET;
    });

    const css = edgeCanvasCss(
      /* left= */ this.props.cardWidth,
      /* top= */ CARD_TITLE_HEIGHT + CARD_ROW_HEIGHT / 2,
      /* width= */ edgeWidth,
    );
    const edgeCanvasClasses = classes(css.edgeCanvas, reverseEdges && 'edgeCanvasReverse');
    return <div className={edgeCanvasClasses}>{edgeLines}</div>;
  }
}
