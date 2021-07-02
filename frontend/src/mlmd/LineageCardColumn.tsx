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

import grey from '@material-ui/core/colors/grey';
import React from 'react';
import { classes, stylesheet } from 'typestyle';
import { LineageCard } from './LineageCard';
import { LineageCardType, LineageRow } from './LineageTypes';
import { CARD_OFFSET, EdgeCanvas } from './EdgeCanvas';
import { Artifact } from 'src/third_party/mlmd';
import { ControlledEdgeCanvas } from './ControlledEdgeCanvas';
import { CARD_ROW_HEIGHT } from './LineageCss';

export interface CardDetails {
  title: string;
  elements: LineageRow[];
}

export interface LineageCardColumnProps {
  type: LineageCardType;
  title: string;
  cards: CardDetails[];
  connectedCards?: CardDetails[];
  columnWidth: number;
  columnPadding: number;
  reverseBindings?: boolean;
  skipEdgeCanvas?: boolean;
  outputExecutionToOutputArtifactMap?: Map<number, number[]>;
  setLineageViewTarget?(artifact: Artifact): void;
}

const NEXT_ITEM_SAME_CARD_OFFSET = CARD_ROW_HEIGHT;
const NEXT_ITEM_NEXT_CARD_OFFSET = CARD_ROW_HEIGHT + CARD_OFFSET;

export class LineageCardColumn extends React.Component<LineageCardColumnProps> {
  public render(): JSX.Element | null {
    const { columnPadding, type, title } = this.props;

    const css = stylesheet({
      mainColumn: {
        display: 'inline-block',
        justifyContent: 'center',
        minHeight: '100%',
        padding: `0 ${columnPadding}px`,
        width: this.props.columnWidth,
        boxSizing: 'border-box',
        $nest: {
          h2: {
            color: grey[600],
            fontFamily: 'PublicSans-Regular',
            fontSize: '12px',
            letterSpacing: '0.5px',
            lineHeight: '40px',
            textAlign: 'left',
            textTransform: 'capitalize',
          },
        },
      },
      columnBody: {
        position: 'relative',
        width: '100%',
      },
      columnHeader: {
        height: '40px',
        margin: '10px 0px',
        textAlign: 'left',
        width: '100%',
      },
    });

    return (
      <div className={classes(css.mainColumn, type)}>
        <div className={classes(css.columnHeader)}>
          <h2>{title}</h2>
        </div>
        <div className={classes(css.columnBody)}>{this.drawColumnContent()}</div>
      </div>
    );
  }
  private jsxFromCardDetails(det: CardDetails, i: number): JSX.Element {
    const isNotFirstEl = i > 0;
    return (
      <LineageCard
        key={i}
        title={det.title}
        type={this.props.type}
        addSpacer={isNotFirstEl}
        rows={det.elements}
        isTarget={/Target/i.test(this.props.title)}
        setLineageViewTarget={this.props.setLineageViewTarget}
      />
    );
  }

  private drawColumnContent(): JSX.Element {
    const { cards, columnPadding, columnWidth, skipEdgeCanvas } = this.props;
    const edgeWidth = columnPadding * 2;
    const cardWidth = columnWidth - edgeWidth;

    let edgeCanvases: JSX.Element[] = [];

    if (this.props.outputExecutionToOutputArtifactMap && this.props.connectedCards) {
      edgeCanvases = this.buildOutputExecutionToOutputArtifactEdgeCanvases(
        this.props.outputExecutionToOutputArtifactMap,
        cards,
        this.props.connectedCards,
        edgeWidth,
        cardWidth,
      );
    } else {
      edgeCanvases.push(
        <EdgeCanvas
          cardWidth={cardWidth}
          edgeWidth={edgeWidth}
          cards={cards}
          reverseEdges={!!this.props.reverseBindings}
        />,
      );
    }

    return (
      <React.Fragment>
        {skipEdgeCanvas ? null : edgeCanvases}
        {cards.map(this.jsxFromCardDetails.bind(this))}
      </React.Fragment>
    );
  }

  private buildOutputExecutionToOutputArtifactEdgeCanvases(
    outputExecutionToOutputArtifactMap: Map<number, number[]>,
    artifactCards: CardDetails[],
    executionCards: CardDetails[],
    edgeWidth: number,
    cardWidth: number,
  ): JSX.Element[] {
    const edgeCanvases: JSX.Element[] = [];

    const artifactIdToCardMap = new Map<number, number>();
    artifactCards.forEach((card, index) => {
      card.elements.forEach(row => {
        artifactIdToCardMap.set(row.typedResource.resource.getId(), index);
      });
    });

    const executionIdToCardMap = new Map<number, number>();
    executionCards.forEach((card, index) => {
      card.elements.forEach(row => {
        executionIdToCardMap.set(row.typedResource.resource.getId(), index);
      });
    });

    // Offset of the top of the card relative to the top of the column
    let artifactOffset = 0;
    let executionOffset = 0;

    let artifactCardIndex: number | undefined;

    let executionIndex = 0;
    let executionCardIndex: number | undefined;
    let previousExecutionCardIndex: number | undefined;

    outputExecutionToOutputArtifactMap.forEach((artifactIds, executionId) => {
      if (executionIndex > 0) {
        executionCardIndex = executionIdToCardMap.get(executionId);
        if (previousExecutionCardIndex === executionCardIndex) {
          // Next execution is on the same card
          executionOffset += NEXT_ITEM_SAME_CARD_OFFSET;
        } else {
          // Next execution is on the next card
          executionOffset += NEXT_ITEM_NEXT_CARD_OFFSET;
        }
      }

      edgeCanvases.push(
        <ControlledEdgeCanvas
          cardWidth={cardWidth}
          edgeWidth={edgeWidth}
          reverseEdges={!!this.props.reverseBindings}
          artifactIds={artifactIds}
          artifactToCardMap={artifactIdToCardMap}
          offset={artifactOffset - executionOffset}
          outputExecutionToOutputArtifactMap={outputExecutionToOutputArtifactMap}
          top={executionOffset}
        />,
      );

      // Advance starting artifact offset.
      artifactIds.forEach(artifactId => {
        if (artifactCardIndex === null) {
          artifactCardIndex = artifactIdToCardMap.get(artifactId) as number;
          return;
        }

        const newArtifactIndex = artifactIdToCardMap.get(artifactId);
        if (artifactCardIndex === newArtifactIndex) {
          // Next artifact row is on the same card
          artifactOffset += NEXT_ITEM_SAME_CARD_OFFSET;
        } else {
          // Next artifact row is on the next card
          artifactOffset += NEXT_ITEM_NEXT_CARD_OFFSET;
        }
        artifactCardIndex = newArtifactIndex;
      });

      previousExecutionCardIndex = executionIdToCardMap.get(executionId);

      executionIndex++;
    });

    return edgeCanvases;
  }
}
