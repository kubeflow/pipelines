/*
 * Copyright 2019 Google LLC
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
import Tooltip from '@material-ui/core/Tooltip';
import WarningIcon from '@material-ui/icons/WarningRounded';
import { Row, Column } from './CustomTable';
import { color, fonts, fontsize } from '../Css';
import { stylesheet } from 'typestyle';

export const css = stylesheet({
  cell: {
    $nest: {
      '&:not(:nth-child(2))': {
        color: color.inactive,
      },
    },
    alignSelf: 'center',
    borderBottom: 'initial',
    color: color.foreground,
    fontFamily: fonts.secondary,
    fontSize: fontsize.base,
    letterSpacing: 0.25,
    marginRight: 20,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  icon: {
    color: color.alert,
    height: 18,
    paddingRight: 4,
    verticalAlign: 'sub',
    width: 18,
  },
  row: {
    $nest: {
      '&:hover': {
        backgroundColor: '#f3f3f3',
      },
    },
    borderBottom: '1px solid #ddd',
    display: 'flex',
    flexShrink: 0,
    height: 40,
    outline: 'none',
  },
});

interface CustomTableRowProps {
  row: Row;
  columns: Column[];
}

function calculateColumnWidths(columns: Column[]): number[] {
  const totalFlex = columns.reduce((total, c) => (total += c.flex || 1), 0);
  return columns.map(c => ((c.flex || 1) / totalFlex) * 100);
}

// tslint:disable-next-line:variable-name
export const CustomTableRow: React.FC<CustomTableRowProps> = (props: CustomTableRowProps) => {
  const { row, columns } = props;
  const widths = calculateColumnWidths(columns);
  return (
    <React.Fragment>
      {row.otherFields.map((cell, i) => (
        <div key={i} style={{ width: widths[i] + '%' }} className={css.cell}>
          {i === 0 && row.error && (
            <Tooltip title={row.error}>
              <WarningIcon className={css.icon} />
            </Tooltip>
          )}
          {columns[i].customRenderer
            ? columns[i].customRenderer!({ value: cell, id: row.id })
            : cell}
        </div>
      ))}
    </React.Fragment>
  );
};
