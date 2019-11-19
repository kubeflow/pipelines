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

import * as React from 'react';
import { logger } from '../lib/Utils';
import { stylesheet, classes } from 'typestyle';
import { color } from '../Css';

const borderStyle = `1px solid ${color.divider}`;

const css = stylesheet({
  cell: {
    border: borderStyle,
    borderCollapse: 'collapse',
    padding: 5,
  },
  labelCell: {
    backgroundColor: color.lightGrey,
    fontWeight: 'bold',
    maxWidth: 200,
    minWidth: 50,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  root: {
    border: borderStyle,
    borderCollapse: 'collapse',
  },
  row: {
    $nest: {
      '&:hover': {
        backgroundColor: '#f7f7f7',
      },
    },
  },
});

export interface CompareTableProps {
  rows: string[][];
  xLabels: string[];
  yLabels: string[];
}

class CompareTable extends React.PureComponent<CompareTableProps> {
  public render(): JSX.Element | null {
    const { rows, xLabels, yLabels } = this.props;
    if (rows.length !== yLabels.length) {
      logger.error(
        `Number of rows (${rows.length}) should match the number of Y labels (${yLabels.length}).`,
      );
    }
    if (!rows || rows.length === 0) {
      return null;
    }

    return (
      <table className={css.root}>
        <tbody>
          <tr className={css.row}>
            <td className={css.labelCell} />
            {/* X labels row */}
            {xLabels.map((label, i) => (
              <td key={i} className={classes(css.cell, css.labelCell)} title={label}>
                {label}
              </td>
            ))}
          </tr>
          {rows.map((row, i) => (
            <tr key={i} className={css.row}>
              {/* Y label */}
              <td className={classes(css.cell, css.labelCell)} title={yLabels[i]}>
                {yLabels[i]}
              </td>

              {/* Row cells */}
              {row.map((cell, j) => (
                <td key={j} className={css.cell} title={cell}>
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    );
  }
}

export default CompareTable;
