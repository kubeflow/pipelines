/*
 * Copyright 2018 The Kubeflow Authors
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
import { Link } from 'react-router';
import { logger } from '../lib/Utils';
import './CompareTable.css';

export interface xParentLabel {
  label: string;
  colSpan: number;
}

export interface CompareTableProps {
  rows: string[][];
  xLabels: string[];
  yLabels: string[];
  xParentLabels?: xParentLabel[];
  label?: string;
  xLinks?: string[];
  missingCells?: boolean[][];
}

class CompareTable extends React.PureComponent<CompareTableProps> {
  public render(): React.JSX.Element | null {
    const {
      rows,
      xLabels,
      yLabels,
      xParentLabels,
      xLinks,
      missingCells,
      label = 'Run comparison',
    } = this.props;
    if (rows.length !== yLabels.length) {
      logger.error(
        `Number of rows (${rows.length}) should match the number of Y labels (${yLabels.length}).`,
      );
    }
    const parentLength = xParentLabels?.reduce((length, parent) => length + parent.colSpan, 0);
    if (xParentLabels && parentLength !== xLabels.length) {
      logger.error(
        `Number of columns with data (${xLabels.length}) should match the aggregated length of parent columns (${parentLength}).`,
      );
    }
    if (!rows.length) return null;
    return (
      <div className='kfp-compare-table-scroll' role='region' aria-label={label} tabIndex={0}>
        <table className='kfp-compare-table' aria-label={label}>
          <thead>
            {xParentLabels && parentLength === xLabels.length && (
              <tr>
                <th scope='col'>Run</th>
                {xParentLabels.map((parent, index) => (
                  <th key={index} scope='colgroup' colSpan={parent.colSpan}>
                    {parent.label}
                  </th>
                ))}
              </tr>
            )}
            <tr>
              <th scope='col'>Name</th>
              {xLabels.map((name, index) => (
                <th key={index} scope='col'>
                  {xLinks?.[index] ? <Link to={xLinks[index]}>{name}</Link> : name}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => {
              const absent = (column: number) =>
                missingCells?.[rowIndex]?.[column] ??
                (row[column] === undefined || row[column] === '');
              const signatures = xLabels.map((_, column) =>
                JSON.stringify([absent(column), row[column]]),
              );
              const differs = new Set(signatures).size > 1;
              return (
                <tr key={rowIndex} data-different={differs || undefined}>
                  <th
                    scope='row'
                    aria-label={`${yLabels[rowIndex]}${differs ? ' (values differ)' : ''}`}
                  >
                    {yLabels[rowIndex]}
                  </th>
                  {xLabels.map((_, column) => (
                    <td key={column}>
                      {absent(column) ? (
                        <span aria-label='Not provided'>—</span>
                      ) : row[column] === '' ? (
                        <span aria-label='Empty string'>“”</span>
                      ) : (
                        row[column]
                      )}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default CompareTable;
