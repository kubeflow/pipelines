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
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TablePagination from '@material-ui/core/TablePagination';
import TableRow from '@material-ui/core/TableRow';
import TableSortLabel from '@material-ui/core/TableSortLabel';
import Tooltip from '@material-ui/core/Tooltip';
import { ViewerConfig, PlotType } from './Viewer';
import { color, fontsize, commonCss } from '../../Css';
import { stylesheet } from 'typestyle';

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

class PagedTable extends React.Component<PagedTableProps, PagedTableState> {
  private _shrinkThreshold = 600;
  private _config = this.props.configs[0];
  private _rowHeight = 30;

  private _css = stylesheet({
    cell: {
      borderRight: 'solid 1px ' + color.divider,
      color: color.foreground,
      fontSize: this._isSmall() ? fontsize.small : fontsize.base,
      paddingLeft: this._isSmall() ? 5 : 'invalid',
      paddingRight: 5,
      pointerEvents: this._isSmall() ? 'none' : 'initial',
    },
    columnName: {
      fontSize: this._isSmall() ? fontsize.base : fontsize.medium,
      fontWeight: 'bold',
      paddingLeft: this._isSmall() ? 5 : 'invalid',
    },
    row: {
      borderBottom: '1px solid #ddd',
      height: this._isSmall() ? 25 : this._rowHeight,
    },
  });

  constructor(props: any) {
    super(props);
    this.state = {
      order: SortOrder.ASC,
      orderBy: 0,
      page: 0,
      rowsPerPage: 10,
    };
  }

  public render(): JSX.Element | null {
    if (!this._config) {
      return null;
    }

    const { data, labels } = this._config;
    const { order, orderBy, rowsPerPage, page } = this.state;
    const emptyRows = rowsPerPage - Math.min(rowsPerPage, data.length - page * rowsPerPage);

    return (
      <div style={{ width: '100%' }} className={commonCss.page}>
        <Table style={{ display: 'block', overflow: 'auto' }}>
          <TableHead>
            <TableRow>
              {labels.map((label, i) => {
                return (
                  <TableCell
                    className={this._css.columnName}
                    key={i}
                    sortDirection={orderBy === i ? order : false}
                  >
                    <Tooltip title='Sort' enterDelay={300}>
                      <TableSortLabel
                        active={orderBy === i}
                        direction={order}
                        onClick={this._handleSort(i)}
                      >
                        {label}
                      </TableSortLabel>
                    </Tooltip>
                  </TableCell>
                );
              }, this)}
            </TableRow>
          </TableHead>

          <TableBody>
            {/* TODO: bug: sorting keeps appending items */}
            {this._stableSort(data)
              .slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
              .map(row => {
                return (
                  <TableRow hover={true} tabIndex={-1} key={row[0]} className={this._css.row}>
                    {row.map((cell, i) => (
                      <TableCell key={i} className={this._css.cell}>
                        {cell}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })}
            {emptyRows > 0 && (
              <TableRow style={{ height: this._rowHeight * emptyRows }}>
                <TableCell colSpan={6} />
              </TableRow>
            )}
          </TableBody>
        </Table>

        <TablePagination
          component='div'
          count={data.length}
          rowsPerPage={rowsPerPage}
          page={page}
          onChangePage={this._handleChangePage}
          onChangeRowsPerPage={this._handleChangeRowsPerPage}
        />
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

  private _handleChangePage = (event: any, page: number) => {
    this.setState({ page });
  };

  private _handleChangeRowsPerPage = (event: any) => {
    this.setState({ rowsPerPage: event.target.value });
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

  private _getSorting(order: SortOrder, orderBy: number): (a: any, b: any) => number {
    return order === SortOrder.DESC
      ? (a: any, b: any) => this._desc(a, b, orderBy)
      : (a: any, b: any) => -this._desc(a, b, orderBy);
  }
}

export default PagedTable;
