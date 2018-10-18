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
import Checkbox from '@material-ui/core/Checkbox';
import ChevronLeft from '@material-ui/icons/ChevronLeft';
import ChevronRight from '@material-ui/icons/ChevronRight';
import IconButton from '@material-ui/core/IconButton';
import MenuItem from '@material-ui/core/MenuItem';
import TableSortLabel from '@material-ui/core/TableSortLabel';
import TextField from '@material-ui/core/TextField';
import Tooltip from '@material-ui/core/Tooltip';
import WarningIcon from '@material-ui/icons/WarningRounded';
import { BaseListRequest } from '../lib/Apis';
import { CheckboxProps } from '@material-ui/core/Checkbox';
import { TextFieldProps } from '@material-ui/core/TextField';
import { classes, stylesheet } from 'typestyle';
import { fontsize, dimension, commonCss, color, padding } from '../Css';

export interface Column {
  flex?: number;
  label: string;
  sortKey?: string;
  customRenderer?: (value: any, id: string) => React.StatelessComponent;
}
export interface Row {
  id: string;
  otherFields: any[];
  error?: string;
}

const rowHeight = 40;

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
    fontSize: fontsize.base,
    marginRight: 10,
    overflow: 'hidden',
    padding: '0 10px',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  columnName: {
    fontSize: fontsize.medium,
    fontWeight: 'bold',
  },
  emptyMessage: {
    padding: 20,
    textAlign: 'center',
  },
  footer: {
    padding: '5px 10px',
    textAlign: 'right',
  },
  header: {
    borderBottom: 'solid 1px ' + color.divider,
    display: 'flex',
    flex: '0 0 50px',
    lineHeight: '50px', // must declare px
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
    height: rowHeight,
    outline: 'none',
  },
  rowsPerPage: {
    height: dimension.xsmall,
    minWidth: dimension.base,
  },
  selected: {
    backgroundColor: color.activeBg,
  },
});

interface CustomTableProps {
  columns: Column[];
  disablePaging?: boolean;
  disableSelection?: boolean;
  emptyMessage?: string;
  orderAscending: boolean;
  pageSize: number;
  reload: (request: BaseListRequest) => Promise<string>;
  rows: Row[];
  selectedIds?: string[];
  sortBy: string;
  updateSelection?: (selectedIds: string[]) => void;
}

interface CustomTableState {
  currentPage: number;
  maxPageNumber: number;
  tokenList: string[];
}

export default class CustomTable extends React.Component<CustomTableProps, CustomTableState> {
  constructor(props: any) {
    super(props);

    this.state = {
      currentPage: 0,
      maxPageNumber: Number.MAX_SAFE_INTEGER,
      tokenList: [''],
    };
  }

  public handleSelectAllClick(event: React.MouseEvent) {
    if (this.props.disableSelection === true) {
      return;
    }
    const selectedIds = (event.target as CheckboxProps).checked ? this.props.rows.map((v) => v.id) : [];
    if (this.props.updateSelection) {
      this.props.updateSelection(selectedIds);
    }
  }

  public handleClick(e: React.MouseEvent, id: string) {
    if (this.props.disableSelection === true) {
      return;
    }
    const selectedIds = this.props.selectedIds || [];
    const selectedIndex = selectedIds.indexOf(id);
    const newSelected = selectedIndex === -1 ?
      selectedIds.concat(id) :
      selectedIds.slice(0, selectedIndex).concat(selectedIds.slice(selectedIndex + 1));

    if (this.props.updateSelection) {
      this.props.updateSelection(newSelected);
    }

    e.stopPropagation();
  }

  public isSelected(id: string) {
    return this.props.selectedIds && this.props.selectedIds.indexOf(id) !== -1;
  }

  public componentDidMount() {
    this._pageChanged(0);
  }

  public render() {
    const { orderAscending, sortBy, pageSize } = this.props;
    const numSelected = (this.props.selectedIds || []).length;
    const totalFlex = this.props.columns.reduce((total, c) => total += (c.flex || 1), 0);
    const widths = this.props.columns.map(c => (c.flex || 1) / totalFlex * 100);

    return (
      <div className={commonCss.pageOverflowHidden}>

        {/* Header */}
        <div className={css.header}>
          {this.props.disableSelection !== true && (
            <div className={classes(css.columnName, css.cell)}>
              <Checkbox indeterminate={!!numSelected && numSelected < this.props.rows.length}
                color='primary' checked={numSelected === this.props.rows.length}
                onChange={this.handleSelectAllClick.bind(this)} />
            </div>
          )}
          {this.props.columns.map((col, i) => {
            const isColumnSortable = !!this.props.columns[i].sortKey;
            const isCurrentSortColumn = sortBy === this.props.columns[i].sortKey;
            return (
              <div key={i} style={{ width: widths[i] + '%' }}
                className={css.columnName}>
                <Tooltip title={isColumnSortable ? 'Sort' : 'Cannot sort by this column'}
                  enterDelay={300}>
                  <TableSortLabel active={isCurrentSortColumn} className={commonCss.ellipsis}
                    direction={isColumnSortable ? orderAscending ? 'asc' : 'desc' : undefined}
                    onClick={() => this._requestSort(this.props.columns[i].sortKey)}>
                    {col.label}
                  </TableSortLabel>
                </Tooltip>
              </div>
            );
          })}
        </div>

        {/* Body */}
        <div className={commonCss.scrollContainer}>
          {/* Empty experience */}
          {this.props.rows.length === 0 && !!this.props.emptyMessage && (
            <div className={css.emptyMessage}>{this.props.emptyMessage}</div>
          )}
          {this.props.rows.map((row: Row) => {
            if (row.otherFields.length !== this.props.columns.length) {
              throw new Error('Rows must have the same number of cells defined in columns');
            }
            return (
              <div role='checkbox' key={row.id} tabIndex={-1}
                className={classes('tableRow', css.row, this.isSelected(row.id) && css.selected)}
                onClick={e => this.handleClick(e, row.id)}>
                {this.props.disableSelection !== true && (
                  <div className={css.cell}>
                    <Checkbox color='primary' checked={this.isSelected(row.id)} />
                  </div>
                )}
                {row.otherFields.map((cell, c) => (
                  <div key={c} style={{ width: widths[c] + '%' }} className={css.cell}>
                    {c === 0 && row.error && (
                      <Tooltip title={row.error}><WarningIcon className={css.icon} /></Tooltip>
                    )}
                    {this.props.columns[c].customRenderer ?
                      this.props.columns[c].customRenderer!(cell, row.id) : cell}
                  </div>
                ))}
              </div>
            );
          })}
        </div>

        {/* Footer */}
        {!this.props.disablePaging && <div className={css.footer}>
          <span className={padding()}>Rows per page:</span>
          <TextField select={true} variant='outlined' className={css.rowsPerPage}
            onChange={this._requestRowsPerPage.bind(this)} value={pageSize}>
            {[10, 20, 50, 100].map((size, i) => (
              <MenuItem key={i} value={size}>{size}</MenuItem>
            ))}
          </TextField>

          <IconButton onClick={() => this._pageChanged(-1)} disabled={!this.state.currentPage}>
            <ChevronLeft />
          </IconButton>
          <IconButton onClick={() => this._pageChanged(1)}
            disabled={this.state.currentPage >= this.state.maxPageNumber}>
            <ChevronRight />
          </IconButton>
        </div>}
      </div>
    );
  }

  private _requestSort(sortBy?: string) {
    if (sortBy) {
      const orderAscending = this.props.sortBy === sortBy ? !this.props.orderAscending : true;
      this.props.reload({ orderAscending, sortBy });
    }
  }

  private async _pageChanged(offset: number) {
    let newCurrentPage = this.state.currentPage + offset;
    let maxPageNumber = this.state.maxPageNumber;
    newCurrentPage = Math.max(0, newCurrentPage);
    newCurrentPage = Math.min(this.state.maxPageNumber, newCurrentPage);

    const newPageToken = await this.props.reload({
      pageToken: this.state.tokenList[newCurrentPage],
    });

    if (newPageToken) {
      // If we're using the greatest yet known page, then the pageToken will be new.
      if (newCurrentPage + 1 === this.state.tokenList.length) {
        this.state.tokenList.push(newPageToken);
      }
    } else {
      maxPageNumber = newCurrentPage;
    }

    // TODO: saw this warning:
    // Warning: Can't call setState (or forceUpdate) on an unmounted component.
    // This is a no-op, but it indicates a memory leak in your application.
    // To fix, cancel all subscriptions and asynchronous tasks in the componentWillUnmount method.
    this.setState({ currentPage: newCurrentPage, maxPageNumber });
  }

  private async _requestRowsPerPage(event: React.ChangeEvent) {
    const pageSize = (event.target as TextFieldProps).value as number;

    const newToken = await this.props.reload({
      pageSize,
      pageToken: '',
    });

    const newTokenList = [''];
    if (newToken) {
      newTokenList.push(newToken);
    }

    // Reset state, since this invalidates the token list and page counter calculations
    this.setState({
      currentPage: 0,
      maxPageNumber: Number.MAX_SAFE_INTEGER,
      tokenList: newTokenList,
    });
  }
}
