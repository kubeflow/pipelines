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

import { blue, grey } from '@material-ui/core/colors';
import * as React from 'react';
import { classes, stylesheet } from 'typestyle';
import { CSSProperties } from 'typestyle/lib/types';
import { LineageCardRow } from './LineageCardRow';
import { LineageRow, LineageCardType } from './LineageTypes';
import {
  CARD_RADIUS,
  CARD_SPACER_HEIGHT,
  CARD_TITLE_BASE_HEIGHT,
  CARD_TITLE_BORDER_BOTTOM_HEIGHT,
  px,
} from './LineageCss';
import { Artifact } from 'src/third_party/mlmd';

const cardTitleBase: CSSProperties = {
  borderTopLeftRadius: '4px',
  borderTopRightRadius: '4px',
  height: px(CARD_TITLE_BASE_HEIGHT),
};

interface LineageCardProps {
  title: string;
  type: LineageCardType;
  rows: LineageRow[];
  addSpacer: boolean;
  isTarget?: boolean;
  setLineageViewTarget?(artifact: Artifact): void;
}

export class LineageCard extends React.Component<LineageCardProps> {
  public render(): JSX.Element {
    const { title, type, rows, addSpacer, isTarget, setLineageViewTarget } = this.props;
    const isExecution = type === 'execution';

    const css = stylesheet({
      addSpacer: {
        marginTop: px(CARD_SPACER_HEIGHT),
      },
      cardContainer: {
        background: 'white',
        border: `1px solid ${grey[300]}`,
        borderRadius: px(CARD_RADIUS),
        $nest: {
          h3: {
            color: blue[600],
            fontFamily: 'PublicSans-Medium',
            fontSize: '9px',
            letterSpacing: '0.8px',
            lineHeight: '42px',
            paddingLeft: '15px',
            textAlign: 'left',
            textTransform: 'uppercase',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          },
        },
      },
      cardTitle: {
        ...cardTitleBase,
        borderBottom: `${px(CARD_TITLE_BORDER_BOTTOM_HEIGHT)} solid ${grey[200]}`,
      },
      execution: {
        background: '#2C4A6D',
        border: '1px solid #CCE4FF',
        $nest: {
          h3: {
            color: '#5DC2B8',
          },
          '.cardRow': {
            borderBottom: '1px solid var(--grey-700)',
          },
          '.cardRow .rowTitle': {
            color: 'white',
          },
          '.cardRow .rowDesc': {
            color: 'var(--grey-500)',
          },
          ".cardRow [class^='edge']": {
            background: '#5DC2B8',
          },
        },
      },
      executionCardTitle: {
        ...cardTitleBase,
        borderBottom: `${px(CARD_TITLE_BORDER_BOTTOM_HEIGHT)} solid transparent`,
      },
      target: {
        border: `2px solid ${blue[500]}`,
      },
    });

    const listCardRows = () =>
      rows.map((r, i) => (
        <LineageCardRow
          key={i}
          typedResource={r.typedResource}
          resourceDetailsRoute={r.resourceDetailsPageRoute}
          leftAffordance={!!r.prev}
          rightAffordance={!!r.next}
          isLastRow={i === rows.length - 1}
          isTarget={isTarget}
          hideRadio={isExecution}
          setLineageViewTarget={setLineageViewTarget}
        />
      ));

    const cardContainerClasses = classes(
      css.cardContainer,
      css[type], // css.execution
      addSpacer ? css.addSpacer : '',
      isTarget ? css.target : '',
    );

    return (
      <div className={cardContainerClasses}>
        <div className={classes(isExecution ? css.executionCardTitle : css.cardTitle)}>
          <h3 title={title}>{title}</h3>
        </div>
        <div className='cardBody'>{listCardRows()}</div>
      </div>
    );
  }
}
