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
import Viewer, { ViewerConfig, PlotType } from './Viewer';
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '../ui/button';
import './RichViewers.css';

enum SortOrder {
  ASC = 'asc',
  DESC = 'desc',
}

export interface PagedTableConfig extends ViewerConfig {
  data: string[][];
  labels: string[];
  type: PlotType;
}

interface PagedTableProps {
  configs: PagedTableConfig[];
  maxDimension?: number;
}

interface PagedTableState {
  order: SortOrder;
  orderBy: number;
  page: number;
  rowsPerPage: number;
}

class PagedTable extends Viewer<PagedTableProps, PagedTableState> {
  private _shrinkThreshold = 600;
  private _rowHeight = 30;

  constructor(props: PagedTableProps) {
    super(props);
    this.state = {
      order: SortOrder.ASC,
      orderBy: 0,
      page: 0,
      rowsPerPage: 10,
    };
  }

  public getDisplayName(): string {
    return 'Table';
  }

  public render(): React.JSX.Element | null {
    const config = this.props.configs[0];
    if (!config) {
      return null;
    }

    const { data, labels } = config;
    const { order, orderBy, rowsPerPage } = this.state;
    const lastPage = Math.max(0, Math.ceil(data.length / rowsPerPage) - 1);
    const page = Math.min(this.state.page, lastPage);
    const emptyRows = rowsPerPage - Math.min(rowsPerPage, data.length - page * rowsPerPage);

    return (
      <div className={`kfp-viewer-table ${this._isSmall() ? 'kfp-viewer-table-compact' : ''}`}>
        <div
          className='kfp-viewer-table-scroll'
          role='region'
          aria-label='Table output'
          tabIndex={0}
        >
          <table aria-label='Table output'>
            {labels.length > 0 && (
              <thead>
                <tr>
                  {labels.map((label, i) => (
                    <th
                      key={i}
                      scope='col'
                      aria-sort={
                        orderBy === i
                          ? order === SortOrder.ASC
                            ? 'ascending'
                            : 'descending'
                          : 'none'
                      }
                    >
                      <button
                        className='kfp-viewer-table-sort'
                        onClick={this._handleSort(i)}
                        aria-label={label || `Column ${i + 1}`}
                      >
                        {label}
                        {orderBy !== i ? (
                          <ArrowUpDown aria-hidden />
                        ) : order === SortOrder.ASC ? (
                          <ArrowUp aria-hidden />
                        ) : (
                          <ArrowDown aria-hidden />
                        )}
                      </button>
                    </th>
                  ))}
                </tr>
              </thead>
            )}
            <tbody className={labels.length === 0 ? 'kfp-viewer-table-unlabelled' : undefined}>
              {this._stableSort(data)
                .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
                .map((row, index) => (
                  <tr key={index} className='kfp-viewer-table-data'>
                    {row.map((cell, i) => (
                      <td key={i}>{cell}</td>
                    ))}
                  </tr>
                ))}
              {emptyRows > 0 && (
                <tr style={{ height: this._rowHeight * emptyRows }}>
                  <td colSpan={Math.max(labels.length, data[0]?.length || 0, 1)} />
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className='kfp-viewer-table-pagination'>
          <label>
            Rows per page
            <select
              className='kfp-viewer-select'
              value={rowsPerPage}
              onChange={this._handleChangeRowsPerPage}
            >
              {[10, 25, 50, 100].map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>
          <span aria-live='polite'>
            {data.length === 0 ? 0 : page * rowsPerPage + 1}–
            {Math.min((page + 1) * rowsPerPage, data.length)} of {data.length}
          </span>
          <Button
            variant='ghost'
            size='icon'
            aria-label='Go to previous page'
            disabled={page === 0}
            onClick={() => this.setState({ page: page - 1 })}
          >
            <ChevronLeft aria-hidden />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            aria-label='Go to next page'
            disabled={page >= lastPage}
            onClick={() => this.setState({ page: page + 1 })}
          >
            <ChevronRight aria-hidden />
          </Button>
        </div>
      </div>
    );
  }

  private _handleSort = (index: number) => () => {
    const orderBy = index;
    let order = SortOrder.ASC;

    if (this.state.orderBy === index && this.state.order === SortOrder.ASC) {
      order = SortOrder.DESC;
    }

    this.setState({ order, orderBy });
  };

  private _handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLSelectElement>) => {
    this.setState({ rowsPerPage: Number(event.target.value), page: 0 });
  };

  private _isSmall(): boolean {
    return !!this.props.maxDimension && this.props.maxDimension < this._shrinkThreshold;
  }

  private _stableSort(array: string[][]): string[][] {
    const stabilizedThis = array.map((row: string[], index: number): [string[], number] => [
      row,
      index,
    ]);

    const compareFn = this._getSorting(this.state.order, this.state.orderBy);

    stabilizedThis.sort((a: [string[], number], b: [string[], number]) => {
      const order = compareFn(a[0], b[0]);
      if (order !== 0) {
        return order;
      }
      return a[1] - b[1];
    });
    return stabilizedThis.map((el: [string[], number]) => el[0]);
  }

  private _desc(a: string[], b: string[], orderBy: number): number {
    if (b[orderBy] < a[orderBy]) {
      return -1;
    }
    if (b[orderBy] > a[orderBy]) {
      return 1;
    }
    return 0;
  }

  private _getSorting(order: SortOrder, orderBy: number): (a: string[], b: string[]) => number {
    return order === SortOrder.DESC
      ? (a: string[], b: string[]) => this._desc(a, b, orderBy)
      : (a: string[], b: string[]) => -this._desc(a, b, orderBy);
  }
}

export default PagedTable;
